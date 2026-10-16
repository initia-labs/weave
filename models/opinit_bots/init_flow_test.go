package opinit_bots

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/common"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/internal/features"
	weaveio "github.com/initia-labs/weave/io"
	"github.com/initia-labs/weave/registry"
	"github.com/initia-labs/weave/service"
	"github.com/initia-labs/weave/ui"
	"github.com/stretchr/testify/require"
)

type daInitService struct {
	service.Service // Unexpected daemon operations panic instead of touching the OS.
	created         [][2]string
	pruned          int
}

func (s *daInitService) Create(version, home string) error {
	s.created = append(s.created, [2]string{version, home})
	return nil
}
func (s *daInitService) PruneLogs() error { s.pruned++; return nil }

func daInitServices(t *testing.T) (*daInitService, *[]service.CommandName) {
	t.Helper()
	srv := &daInitService{}
	var calls []service.CommandName
	old := newService
	newService = func(command service.CommandName, vm string) (service.Service, error) {
		require.Empty(t, vm)
		calls = append(calls, command)
		return srv, nil
	}
	t.Cleanup(func() { newService = old })
	return srv, &calls
}

// The fake replaces only external binaries. Keyring contents are real files copied
// by production code; unsupported commands fail instead of silently succeeding.
func daInitBinaries(t *testing.T, home string) string {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("WEAVE_FLOW_HOME", filepath.Join(home, "opinit"))
	trace := filepath.Join(home, "opinit-commands")
	t.Setenv("WEAVE_FLOW_TRACE", trace)
	daWrite(t, trace, "", 0600)
	daWrite(t, GetBinaryPath(home, "v1.2.3"), `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "$WEAVE_FLOW_TRACE"
keydir="$WEAVE_FLOW_HOME/weave-dummy/keyring-test"
if [ "$*" = version ]; then printf 'v1.2.3\n'; exit 0; fi
if [ "$*" = 'keys list weave-dummy' ]; then
 for f in "$keydir"/*.fixture; do [ -f "$f" ] || continue; basename "$f" .fixture; done
 exit 0
fi
if [ "$1 $2" = 'tx grant-oracle' ]; then
 [ "$3" = 'init16pawh0v7w996jrmtzugz3hmhq0wx6ndq5pp0dr' ] && [ "$4" = --home ] && [ "$5" = "$WEAVE_FLOW_HOME" ] && [ "$#" = 5 ]
 exit $?
fi
[ "$1" = keys ] && [ "$3" = weave-dummy ] || exit 40
action="$2"; key="$4"; shift 4
case "$key" in weave_bridge_executor|weave_output_submitter|weave_batch_submitter|weave_challenger|weave_oracle_bridge_executor) ;; *) exit 41;; esac
case "$action" in
 show|delete)
  [ "$*" = "--home $WEAVE_FLOW_HOME" ] || exit 42
  [ -f "$keydir/$key.fixture" ] || exit 1
  if [ "$action" = delete ]; then rm "$keydir/$key.fixture"; else printf '%s: init16pawh0v7w996jrmtzugz3hmhq0wx6ndq5pp0dr\n' "$key"; fi
  ;;
 add)
  [ "$1 $2 $3 $4" = '--coin-type 118 --key-type secp256k1' ] || exit 43
  shift 4
  recover=false
  if [ "$1" = --recover ]; then recover=true; shift; fi
  address=init16pawh0v7w996jrmtzugz3hmhq0wx6ndq5pp0dr
  if [ "$1" = --bech32 ]; then
   [ "$key" = weave_batch_submitter ] && [ "$2" = celestia ] || exit 44
   address=celestia16pawh0v7w996jrmtzugz3hmhq0wx6ndqtasvuv
   shift 2
  fi
  [ "$*" = "--home $WEAVE_FLOW_HOME" ] || exit 45
  read -r mnemonic
  if [ "$recover" = false ]; then
   [ "$mnemonic" = y ] || exit 46
   mnemonic='imitate sick vibrant bonus weather spice pave announce direct impulse strategy math'
  fi
  mkdir -p "$keydir"
  printf '%s\n' "$mnemonic" > "$keydir/$key.fixture"
  printf 'address: %s\n%s\n' "$address" "$mnemonic"
  ;;
 *) exit 47 ;;
esac
`, 0755)
	daWrite(t, filepath.Join(home, common.WeaveDataDirectory, "minievm@v1.2.3", "minitiad"), `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "$WEAVE_FLOW_TRACE"
[ "$1 $2" = 'query opchild' ] && [ "$4" = --node ] && [ "$5" = "$WEAVE_FLOW_L2_RPC" ] && [ "$6 $7" = '--output json' ] && [ "$#" = 7 ] || exit 48
case "$3" in
 next-l1-sequence) printf '{"next_l1_sequence":"1"}\n' ;;
 bridge-info) printf '{"bridge_info":{"bridge_id":"17"}}\n' ;;
 *) exit 49 ;;
esac
`, 0755)
	return trace
}

func daInitFixtures(t *testing.T, home, l2RPC string, manualHeight bool) string {
	t.Helper()
	daFlowFixtures(t)
	oldClient := analytics.Client
	analytics.Client = &analytics.NoOpClient{}
	t.Cleanup(func() { analytics.Client = oldClient })
	oldSpec := registry.OPInitBotsSpecVersion
	registry.OPInitBotsSpecVersion = map[string]int{"test-l1": 7, "artifact-l1": 7}
	t.Cleanup(func() { registry.OPInitBotsSpecVersion = oldSpec })
	t.Setenv("WEAVE_FLOW_L2_RPC", l2RPC)
	daUseTransport(t, func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodGet && r.URL.Host == "api.github.com" && (r.URL.Path == "/repos/initia-labs/opinit-bots/releases" || r.URL.Path == "/repos/initia-labs/minievm/releases") {
			return daResponse(r, daRelease), nil
		}
		if r.Method == http.MethodPost && strings.Contains(r.URL.Host, "graphql") {
			wantURL := "https://graphql.testnet.initia.xyz/v1/graphql"
			if l2RPC == "https://l2-custom.invalid:26657/custom" {
				wantURL = "https://graphql.initia.xyz/v1/graphql"
			}
			require.Equal(t, wantURL, r.URL.String())
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Contains(t, string(body), `"bridgeId":"17"`)
			require.Contains(t, string(body), "CreateBridgeHeight")
			if manualHeight {
				return daResponse(r, `{"data":{"transaction_events":[]}}`), nil
			}
			return daResponse(r, `{"data":{"transaction_events":[{"block_height":876}]}}`), nil
		}
		t.Errorf("unexpected HTTP request: %s %s", r.Method, r.URL)
		return nil, fmt.Errorf("unexpected fixture HTTP request")
	})
	return daInitBinaries(t, home)
}

func daExistingKeys(t *testing.T, home string, roles ...string) {
	t.Helper()
	for _, role := range roles {
		daWrite(t, filepath.Join(home, "opinit", "weave-dummy", "keyring-test", role+".fixture"), "existing "+role+"\n", 0600)
	}
}

func daRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}

// Execute the actual work from Init's batch exactly once. Spinner ticks are UI
// messages and are discarded; the completion goes through the real Update.
func daRunLoading(t *testing.T, model tea.Model, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	t.Helper()
	require.NotNil(t, cmd, "Init command missing for %T", model)
	batch, ok := cmd().(tea.BatchMsg)
	require.True(t, ok, "expected loading Init for %T", model)
	var end tea.Msg
	for _, work := range batch {
		msg := work()
		if _, tick := msg.(ui.TickMsg); tick {
			continue
		}
		require.NoError(t, daError(msg))
		require.IsType(t, ui.EndLoading{}, msg)
		require.Nil(t, end, "loader work ran twice")
		end = msg
	}
	require.NotNil(t, end)
	return model.Update(end)
}

func daCollectKeys(t *testing.T, model tea.Model, cmd tea.Cmd, celestia bool) (tea.Model, tea.Cmd) {
	t.Helper()
	for i := 0; i < 12; i++ {
		switch m := model.(type) {
		case *RecoverKeySelector:
			model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		case *DALayerSelector:
			if celestia {
				m.Update(tea.KeyMsg{Type: tea.KeyDown})
			}
			model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		default:
			return model, cmd
		}
	}
	t.Fatal("key input did not terminate")
	return nil, nil
}

// Catches loading keys before an existing config is validated, even when only
// non-batch keys are pending and the user has selected a DB reset.
func TestDAFlowInitDefersKeysUntilConfigValidation(t *testing.T) {
	for _, entry := range []string{"explicit", "role selector", "explicit existing keys"} {
		t.Run(entry, func(t *testing.T) {
			home := t.TempDir()
			trace := daInitFixtures(t, home, "https://l2-custom.invalid:26657/custom", false)
			daExistingKeys(t, home, OutputSubmitterKeyName, BatchSubmitterKeyName)
			if entry == "explicit existing keys" {
				daExistingKeys(t, home, BridgeExecutorKeyName, OracleBridgeExecutorKeyName)
			}
			_, _ = daCurrentExecutorSelector(t, home, "celestia", true)
			daWrite(t, filepath.Join(home, "opinit", "weave.keyfile.json"), "{}\n", 0640)
			before := daSnapshot(t, filepath.Join(home, "opinit"))
			ctx := daTestContext(t, daFlowState(), home)
			var model tea.Model
			var cmd tea.Cmd
			if entry == "role selector" {
				selector, err := NewOPInitBotInitSelector(ctx)
				require.NoError(t, err)
				model, cmd = selector.Update(tea.KeyMsg{Type: tea.KeyEnter})
			} else {
				var err error
				model, err = PrepareKey(ctx, true)
				require.NoError(t, err)
				cmd = model.Init()
			}
			model, cmd = daCollectKeys(t, model, cmd, false)
			db, ok := model.(*DeleteDBSelector)
			require.True(t, ok, "keys must wait for config validation; got %T", model)
			require.Nil(t, cmd)
			db.Update(tea.KeyMsg{Type: tea.KeyDown})
			model, _ = db.Update(tea.KeyMsg{Type: tea.KeyEnter})
			current := model.(*UseCurrentConfigSelector)
			model, cmd = current.Update(tea.KeyMsg{Type: tea.KeyEnter})
			if features.CelestiaDAEnabled {
				require.IsType(t, &FetchL1StartHeightLoading{}, model)
				require.NotNil(t, cmd)
			} else {
				require.Same(t, current, model)
				require.Nil(t, cmd)
				require.ErrorIs(t, current.validationErr, common.ErrCelestiaDADisabled)
			}
			require.Equal(t, before, daSnapshot(t, filepath.Join(home, "opinit")))
			commands := daRead(t, trace)
			require.NotContains(t, commands, "keys add")
			require.NotContains(t, commands, "keys delete")
			require.NotContains(t, commands, "tx grant")
		})
	}
}

// Catches either height exit bypassing the key loader.
func TestDAFlowHeightRunsMissingKeys(t *testing.T) {
	for _, manual := range []bool{false, true} {
		t.Run(fmt.Sprint(manual), func(t *testing.T) {
			home := t.TempDir()
			trace := daInitFixtures(t, home, "https://final-l2.invalid", manual)
			state := daFlowState()
			state.InitExecutorBot = true
			state.botConfig = map[string]string{"l1_node.chain_id": "test-l1", "l2_node.rpc_address": "https://final-l2.invalid", "da_node.bech32_prefix": "init"}
			state.BotInfos[0].IsGenerateKey = true
			model, err := NewFetchL1StartHeightLoading(daTestContext(t, state, home))
			require.NoError(t, err)
			model, cmd := daRunLoading(t, model, model.Init())
			if manual {
				input, ok := model.(*L1StartHeightInput)
				require.True(t, ok, "%T", model)
				input.Text = "876"
				model, cmd = input.Update(tea.KeyMsg{Type: tea.KeyEnter})
			}
			loader, ok := model.(*SetupOPInitBotsMissingKey)
			require.True(t, ok, "height must precede key execution; got %T", model)
			require.Equal(t, 876, weavecontext.GetCurrentState[OPInitBotsState](loader.Ctx).L1StartHeight)
			model, _ = daRunLoading(t, model, cmd)
			require.Same(t, loader, model)
			require.Contains(t, loader.View(), daInitAddress)
			require.Equal(t, 1, strings.Count(daRead(t, trace), "keys add "))
		})
	}
}

// Catches using package-global Oracle flags, stale loader context, or returning
// to config selection after keys were already generated.
func TestDAFlowMissingKeysUsesCompletionContext(t *testing.T) {
	for _, oracleNew := range []bool{false, true} {
		t.Run(fmt.Sprint(oracleNew), func(t *testing.T) {
			home := t.TempDir()
			daInitFixtures(t, home, "", false)
			state := daFlowState()
			state.InitExecutorBot = true
			state.AddMinitiaConfig = true
			ctx := daTestContext(t, state, home)
			loader := NewSetupOPInitBotsMissingKey(ctx)
			completed := state.Clone()
			completed.L1StartHeight = 876
			completed.botConfig["da_node.bech32_prefix"] = "init"
			completed.BotInfos[4].IsGenerateKey = oracleNew
			completed.SetupOpinitResponses[BridgeExecutor] = "address: " + daInitAddress + "\n" + daMnemonic + "\n"
			if oracleNew {
				completed.SetupOpinitResponses[OracleBridgeExecutor] = "address: " + daInitAddress + "\n" + daMnemonic + "\n"
			}
			daWrite(t, filepath.Join(home, "opinit", "weave.keyfile.json"), "{}", 0644)
			endCtx := weavecontext.SetCurrentState(ctx, completed)
			model, cmd := loader.Update(ui.EndLoading{Ctx: endCtx})
			require.Equal(t, completed, weavecontext.GetCurrentState[OPInitBotsState](loader.Ctx))
			if oracleNew {
				require.Same(t, loader, model, "new Oracle must show the generated key")
				require.Contains(t, loader.View(), daInitAddress)
				model, cmd = loader.Update(tea.KeyMsg{Type: tea.KeyEnter})
				require.Contains(t, daRead(t, filepath.Join(home, "opinit", "weave.keyfile.json")), "weave_oracle_bridge_executor")
			}
			starting, ok := model.(*StartingInitBot)
			require.True(t, ok, "%T", model)
			require.Equal(t, completed, weavecontext.GetCurrentState[OPInitBotsState](starting.Ctx))
			require.NotNil(t, cmd, "StartingInitBot.Init must run on both exits")
		})
	}
}

func daInitFields(t *testing.T, model tea.Model) (tea.Model, tea.Cmd) {
	t.Helper()
	fields, ok := model.(*FieldInputModel)
	require.True(t, ok, "%T", model)
	values := map[string]string{"listen_address": "localhost:3000", "l1_node.rpc_address": "https://final-l1.invalid:26657/custom", "l2_node.chain_id": "final-l2", "l2_node.rpc_address": "https://final-l2.invalid", "l2_node.gas_denom": "umin"}
	var cmd tea.Cmd
	for i := range fields.subModels {
		fields.subModels[i].Text = values[fields.subModels[i].field.Name]
		model, cmd = fields.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	return model, cmd
}

func daFinishInit(t *testing.T, model tea.Model, cmd tea.Cmd, manual, confirm bool) *OPinitBotSuccessful {
	t.Helper()
	require.IsType(t, &FetchL1StartHeightLoading{}, model)
	model, cmd = daRunLoading(t, model, cmd)
	if manual {
		input, ok := model.(*L1StartHeightInput)
		require.True(t, ok, "%T", model)
		input.Text = "876"
		model, cmd = input.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	loader, ok := model.(*SetupOPInitBotsMissingKey)
	require.True(t, ok, "%T", model)
	require.Equal(t, 876, weavecontext.GetCurrentState[OPInitBotsState](loader.Ctx).L1StartHeight)
	model, cmd = daRunLoading(t, model, cmd)
	if confirm {
		require.Same(t, loader, model)
		require.Contains(t, model.View(), "Press enter")
		require.Contains(t, model.View(), daInitAddress)
		model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	starting, ok := model.(*StartingInitBot)
	require.True(t, ok, "%T", model)
	before := weavecontext.GetCurrentState[OPInitBotsState](loader.Ctx)
	after := weavecontext.GetCurrentState[OPInitBotsState](starting.Ctx)
	require.Equal(t, 876, after.L1StartHeight)
	require.Equal(t, before.botConfig, after.botConfig)
	require.Equal(t, before.SetupOpinitResponses, after.SetupOpinitResponses)
	model, cmd = daRunLoading(t, model, cmd)
	success, ok := model.(*OPinitBotSuccessful)
	require.True(t, ok, "%T", model)
	require.Nil(t, cmd)
	require.Contains(t, success.View(), "OPInit bot setup successfully")
	return success
}

func daAssertMutationCommands(t *testing.T, trace, opHome string, adds, deletes []string, recover, celestia bool, grants int) {
	t.Helper()
	commands := strings.Split(strings.TrimSpace(daRead(t, trace)), "\n")
	var gotAdds, gotDeletes, gotGrants []string
	for _, command := range commands {
		switch {
		case strings.HasPrefix(command, "keys add "):
			gotAdds = append(gotAdds, command)
		case strings.HasPrefix(command, "keys delete "):
			gotDeletes = append(gotDeletes, command)
		case strings.HasPrefix(command, "tx grant-oracle "):
			gotGrants = append(gotGrants, command)
		}
	}
	var wantAdds, wantDeletes []string
	for _, key := range adds {
		command := "keys add weave-dummy " + key + " --coin-type 118 --key-type secp256k1"
		if recover && key != OracleBridgeExecutorKeyName {
			command += " --recover"
		}
		if celestia && key == BatchSubmitterKeyName {
			command += " --bech32 celestia"
		}
		wantAdds = append(wantAdds, command+" --home "+opHome)
	}
	for _, key := range deletes {
		wantDeletes = append(wantDeletes, "keys delete weave-dummy "+key+" --home "+opHome)
	}
	require.Equal(t, wantAdds, gotAdds)
	require.Equal(t, wantDeletes, gotDeletes)
	require.Len(t, gotGrants, grants)
	for _, command := range gotGrants {
		require.Equal(t, "tx grant-oracle "+daInitAddress+" --home "+opHome, command)
	}
}

func daAssertCopiedKeys(t *testing.T, home string, chains, keys []string) {
	t.Helper()
	for _, chain := range chains {
		for _, key := range keys {
			original := daRead(t, filepath.Join(home, "opinit", "weave-dummy", "keyring-test", key+".fixture"))
			require.Equal(t, original, daRead(t, filepath.Join(home, "opinit", chain, "keyring-test", key+".fixture")))
		}
	}
}

// Catches missed Init commands, repeated mutation, lost height/responses and
// incorrect final JSON/file modes through manual, imported and reused configs.
func TestDAFlowInitiaExecutorCompletes(t *testing.T) {
	for _, scenario := range []string{"manual", "artifact new oracle", "artifact existing oracle", "current missing keys", "current existing keys"} {
		t.Run(scenario, func(t *testing.T) {
			home := t.TempDir()
			current := strings.HasPrefix(scenario, "current")
			artifact := strings.HasPrefix(scenario, "artifact")
			l2RPC := "https://final-l2.invalid"
			if current {
				l2RPC = "https://l2-custom.invalid:26657/custom"
			}
			trace := daInitFixtures(t, home, l2RPC, scenario == "manual")
			srv, serviceCalls := daInitServices(t)
			state := daFlowState()
			ctx := daTestContext(t, state, home)
			var original string
			if current {
				_, original = daCurrentExecutorSelector(t, home, "init", false)
				daExistingKeys(t, home, OutputSubmitterKeyName, BatchSubmitterKeyName)
				if scenario == "current existing keys" {
					daExistingKeys(t, home, BridgeExecutorKeyName, OracleBridgeExecutorKeyName)
				}
			}
			var model tea.Model
			var cmd tea.Cmd
			if artifact {
				state.MinitiaConfig = daArtifact()
				ctx = weavecontext.SetCurrentState(ctx, state)
				daExistingKeys(t, home, BridgeExecutorKeyName, OutputSubmitterKeyName, BatchSubmitterKeyName, ChallengerKeyName)
				if scenario == "artifact existing oracle" {
					daExistingKeys(t, home, OracleBridgeExecutorKeyName)
				}
				daWrite(t, filepath.Join(home, "minitia", "artifacts", "artifacts.json"), `{"BRIDGE_ID":"17","EXECUTOR_L1_MONITOR_HEIGHT":"876","EXECUTOR_L2_MONITOR_HEIGHT":"1"}`, 0600)
				importer, err := NewProcessingMinitiaConfig(ctx, PrepareExecutorBotKey)
				require.NoError(t, err)
				model, cmd = importer.Update(tea.KeyMsg{Type: tea.KeyEnter})
			} else if current {
				selector, err := NewOPInitBotInitSelector(ctx)
				require.NoError(t, err)
				model, cmd = selector.Update(tea.KeyMsg{Type: tea.KeyEnter})
			} else {
				var err error
				model, err = PrepareKey(ctx, true)
				require.NoError(t, err)
				cmd = model.Init()
			}
			model, cmd = daCollectKeys(t, model, cmd, false)
			require.NotContains(t, daRead(t, trace), "keys add")
			switch m := model.(type) {
			case *UseCurrentConfigSelector:
				model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			case *L1PrefillSelector:
				model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				model, _ = daInitFields(t, model)
				require.IsType(t, &SetDALayer{}, model)
				model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
			case *PrefillMinitiaConfig:
				model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				model, cmd = daInitFields(t, model)
			default:
				t.Fatalf("unexpected config model %T", model)
			}
			success := daFinishInit(t, model, cmd, scenario == "manual", scenario != "current existing keys" && scenario != "artifact existing oracle")
			finalState := weavecontext.GetCurrentState[OPInitBotsState](success.Ctx)
			require.Equal(t, "init", finalState.botConfig["da_node.bech32_prefix"])
			require.Equal(t, [][2]string{{"", filepath.Join(home, "opinit")}}, srv.created)
			require.Equal(t, 1, srv.pruned)
			require.Equal(t, []service.CommandName{service.OPinitExecutor, service.OPinitExecutor}, *serviceCalls)
			var adds, deletes []string
			if scenario == "manual" {
				adds = []string{BridgeExecutorKeyName, OutputSubmitterKeyName, BatchSubmitterKeyName, OracleBridgeExecutorKeyName}
			}
			if scenario == "current missing keys" {
				adds = []string{BridgeExecutorKeyName, OracleBridgeExecutorKeyName}
			}
			if artifact {
				adds = []string{BridgeExecutorKeyName, OutputSubmitterKeyName, BatchSubmitterKeyName, ChallengerKeyName}
				deletes = append([]string(nil), adds...)
				if scenario == "artifact new oracle" {
					adds = append(adds, OracleBridgeExecutorKeyName)
				}
			}
			grants := 1
			if current {
				grants = 0
			}
			daAssertMutationCommands(t, trace, filepath.Join(home, "opinit"), adds, deletes, artifact, false, grants)
			keyfile := weaveio.NewKeyFile()
			require.NoError(t, keyfile.Load(filepath.Join(home, "opinit", "weave.keyfile.json")))
			require.Len(t, keyfile, len(adds))
			for _, key := range adds {
				wantMnemonic := daMnemonic
				if artifact && key == OutputSubmitterKeyName {
					wantMnemonic = daOutputMnemonic
				}
				if artifact && key == ChallengerKeyName {
					wantMnemonic = daChallengerMnemonic
				}
				require.Equal(t, wantMnemonic, keyfile[key].Mnemonic)
				require.Equal(t, daInitAddress, keyfile[key].Address)
				require.Equal(t, wantMnemonic+"\n", daRead(t, filepath.Join(home, "opinit", "weave-dummy", "keyring-test", key+".fixture")))
			}
			configPath := filepath.Join(home, "opinit", "executor.json")
			info, err := os.Stat(configPath)
			require.NoError(t, err)
			allKeys := []string{BridgeExecutorKeyName, OutputSubmitterKeyName, BatchSubmitterKeyName, OracleBridgeExecutorKeyName}
			if current {
				require.Equal(t, original, daRead(t, configPath))
				require.Equal(t, os.FileMode(0640), info.Mode().Perm())
				require.Equal(t, "https://da-custom.invalid", finalState.botConfig["da_node.rpc_address"])
				require.Equal(t, "0.031uinit", finalState.botConfig["da_node.gas_price"])
				daAssertCopiedKeys(t, home, []string{"existing-l1", "existing-l2"}, allKeys)
			} else {
				require.Equal(t, os.FileMode(0600), info.Mode().Perm())
				var config ExecutorConfig
				require.NoError(t, json.Unmarshal([]byte(daRead(t, configPath)), &config))
				l1Chain, gas := "test-l1", "0.015uinit"
				if artifact {
					l1Chain, gas = "artifact-l1", "0.025uinit"
				}
				require.Equal(t, NodeSettings{ChainID: l1Chain, Bech32Prefix: "init", RPCAddress: "https://final-l1.invalid:26657/custom", GasPrice: gas, GasAdjustment: 1.5, TxTimeout: 60}, config.DANode)
				require.Equal(t, config.DANode, config.L1Node)
				require.Equal(t, NodeSettings{ChainID: "final-l2", Bech32Prefix: "init", RPCAddress: "https://final-l2.invalid", GasPrice: "0umin", GasAdjustment: 1.5, TxTimeout: 60}, config.L2Node)
				require.Equal(t, 876, config.L1StartHeight)
				require.Equal(t, 7, config.Version)
				require.True(t, config.DisableAutoSetL1Height)
				daAssertCopiedKeys(t, home, []string{l1Chain, "final-l2"}, allKeys)
			}
			require.NoDirExists(t, filepath.Join(home, "opinit", "celestia-test"))
			require.Equal(t, 1, strings.Count(daRead(t, trace), "query opchild next-l1-sequence --node "+l2RPC+" --output json"))
			require.Equal(t, 1, strings.Count(daRead(t, trace), "query opchild bridge-info --node "+l2RPC+" --output json"))
		})
	}
}

// Catches policy rejection leaving the init workflow unable to replace/skip
// the rejected configuration, or mutating any existing state before that choice.
func TestDAFlowRejectedConfigCanCompleteInitia(t *testing.T) {
	if features.CelestiaDAEnabled {
		t.Skip("Celestia rejection applies to the default build")
	}
	for _, artifact := range []bool{false, true} {
		t.Run(fmt.Sprintf("artifact=%t", artifact), func(t *testing.T) {
			home := t.TempDir()
			trace := daInitFixtures(t, home, "https://final-l2.invalid", false)
			srv, _ := daInitServices(t)
			daExistingKeys(t, home, OutputSubmitterKeyName, BatchSubmitterKeyName)
			daWrite(t, filepath.Join(home, "opinit", "weave.keyfile.json"), "{}\n", 0640)
			state := daFlowState()
			if artifact {
				state.MinitiaConfig = daArtifact()
				state.MinitiaConfig.OpBridge.BatchSubmissionTarget = "CELESTIA"
				state.MinitiaConfig.SystemKeys.BatchSubmitter.DAAddress = daCelestiaAddress
			} else {
				_, _ = daCurrentExecutorSelector(t, home, "celestia", true)
			}
			before := daSnapshot(t, filepath.Join(home, "opinit"))
			ctx := daTestContext(t, state, home)
			var model tea.Model
			var cmd tea.Cmd
			if artifact {
				importer, err := NewProcessingMinitiaConfig(ctx, PrepareExecutorBotKey)
				require.NoError(t, err)
				model, cmd = importer.Update(tea.KeyMsg{Type: tea.KeyEnter})
				require.Same(t, importer, model)
				require.Nil(t, cmd)
				require.ErrorIs(t, importer.validationErr, common.ErrCelestiaDADisabled)
				importer.Update(tea.KeyMsg{Type: tea.KeyDown})
				model, cmd = importer.Update(tea.KeyMsg{Type: tea.KeyEnter})
			} else {
				var err error
				model, err = PrepareKey(ctx, true)
				require.NoError(t, err)
				cmd = model.Init()
			}
			model, cmd = daCollectKeys(t, model, cmd, false)
			if artifact {
				prefill, ok := model.(*PrefillMinitiaConfig)
				require.True(t, ok, "%T", model)
				model, cmd = prefill.Update(tea.KeyMsg{Type: tea.KeyEnter})
				require.Same(t, prefill, model)
				require.Nil(t, cmd)
				require.ErrorIs(t, prefill.validationErr, common.ErrCelestiaDADisabled)
			} else {
				db, ok := model.(*DeleteDBSelector)
				require.True(t, ok, "%T", model)
				db.Update(tea.KeyMsg{Type: tea.KeyDown})
				model, _ = db.Update(tea.KeyMsg{Type: tea.KeyEnter})
				current := model.(*UseCurrentConfigSelector)
				model, cmd = current.Update(tea.KeyMsg{Type: tea.KeyEnter})
				require.Same(t, current, model)
				require.Nil(t, cmd)
				require.ErrorIs(t, current.validationErr, common.ErrCelestiaDADisabled)
			}
			require.Equal(t, before, daSnapshot(t, filepath.Join(home, "opinit")))
			require.Empty(t, srv.created)
			daAssertMutationCommands(t, trace, filepath.Join(home, "opinit"), nil, nil, false, false, 0)
			model.Update(tea.KeyMsg{Type: tea.KeyDown})
			model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
			require.IsType(t, &L1PrefillSelector{}, model)
			model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
			model, _ = daInitFields(t, model)
			model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
			daFinishInit(t, model, cmd, false, true)
			var config ExecutorConfig
			configPath := filepath.Join(home, "opinit", "executor.json")
			require.NoError(t, json.Unmarshal([]byte(daRead(t, configPath)), &config))
			require.Equal(t, "init", config.DANode.Bech32Prefix)
			require.Equal(t, "https://final-l1.invalid:26657/custom", config.DANode.RPCAddress)
			require.Equal(t, "0.015uinit", config.DANode.GasPrice)
			require.Equal(t, 876, config.L1StartHeight)
			daAssertMutationCommands(t, trace, filepath.Join(home, "opinit"), []string{BridgeExecutorKeyName, OracleBridgeExecutorKeyName}, nil, false, false, 1)
			require.NoDirExists(t, filepath.Join(home, "opinit", "executor.db"))
			require.Len(t, srv.created, 1)
			daAssertCopiedKeys(t, home, []string{"test-l1", "final-l2"}, []string{BridgeExecutorKeyName, BatchSubmitterKeyName, OracleBridgeExecutorKeyName})
		})
	}
}

// Catches the same reordered init path suppressing explicitly enabled Celestia.
func TestDAFlowTaggedCelestiaExecutorCompletes(t *testing.T) {
	if !features.CelestiaDAEnabled {
		t.Skip("requires explicit celestia build")
	}
	home := t.TempDir()
	trace := daInitFixtures(t, home, "https://final-l2.invalid", false)
	daCelestiaRegistryFixture()
	srv, calls := daInitServices(t)
	model, err := PrepareKey(daTestContext(t, daFlowState(), home), true)
	require.NoError(t, err)
	model, cmd := daCollectKeys(t, model, model.Init(), true)
	require.IsType(t, &L1PrefillSelector{}, model)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model, _ = daInitFields(t, model)
	da := model.(*SetDALayer)
	require.Equal(t, "Celestia", string(da.Options[da.Cursor]))
	model, cmd = da.Update(tea.KeyMsg{Type: tea.KeyEnter})
	daFinishInit(t, model, cmd, false, true)
	var config ExecutorConfig
	configPath := filepath.Join(home, "opinit", "executor.json")
	require.NoError(t, json.Unmarshal([]byte(daRead(t, configPath)), &config))
	require.Equal(t, NodeSettings{ChainID: "celestia-test", Bech32Prefix: "celestia", RPCAddress: "https://celestia-test.invalid", GasPrice: "0.04utia", GasAdjustment: 1.5, TxTimeout: 60}, config.DANode)
	require.Equal(t, "https://final-l1.invalid:26657/custom", config.L1Node.RPCAddress)
	require.Equal(t, 876, config.L1StartHeight)
	info, err := os.Stat(configPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	keys := []string{BridgeExecutorKeyName, OutputSubmitterKeyName, BatchSubmitterKeyName, OracleBridgeExecutorKeyName}
	daAssertMutationCommands(t, trace, filepath.Join(home, "opinit"), keys, nil, false, true, 1)
	daAssertCopiedKeys(t, home, []string{"test-l1", "final-l2", "celestia-test"}, keys)
	keyfile := weaveio.NewKeyFile()
	require.NoError(t, keyfile.Load(filepath.Join(home, "opinit", "weave.keyfile.json")))
	require.Len(t, keyfile, 4)
	require.Equal(t, daCelestiaAddress, keyfile[BatchSubmitterKeyName].Address)
	require.Equal(t, daMnemonic, keyfile[BatchSubmitterKeyName].Mnemonic)
	require.Equal(t, [][2]string{{"", filepath.Join(home, "opinit")}}, srv.created)
	require.Equal(t, []service.CommandName{service.OPinitExecutor, service.OPinitExecutor}, *calls)
}

// Catches inactive Celestia artifact keys blocking Challenger or causing a
// Celestia lookup/key mutation after the user skips artifact import.
func TestDAFlowChallengerCompletesAfterSkip(t *testing.T) {
	home := t.TempDir()
	trace := daInitFixtures(t, home, "https://final-l2.invalid", false)
	srv, calls := daInitServices(t)
	state := daFlowState()
	state.MinitiaConfig = daArtifact()
	state.MinitiaConfig.OpBridge.BatchSubmissionTarget = "CELESTIA"
	state.MinitiaConfig.SystemKeys.BatchSubmitter.DAAddress = daCelestiaAddress
	ctx := daTestContext(t, state, home)
	importer, err := NewProcessingMinitiaConfig(ctx, NewOPInitBotInitSelector)
	require.NoError(t, err)
	importer.Update(tea.KeyMsg{Type: tea.KeyDown})
	model, _ := importer.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model, cmd = daCollectKeys(t, model, cmd, false)
	require.IsType(t, &PrefillMinitiaConfig{}, model)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model, cmd = daInitFields(t, model)
	daWrite(t, filepath.Join(home, "minitia", "artifacts", "artifacts.json"), `{"BRIDGE_ID":"17","EXECUTOR_L1_MONITOR_HEIGHT":"876"}`, 0600)
	daFinishInit(t, model, cmd, false, true)
	var config ChallengerConfig
	configPath := filepath.Join(home, "opinit", "challenger.json")
	require.NoError(t, json.Unmarshal([]byte(daRead(t, configPath)), &config))
	require.Equal(t, NodeConfig{ChainID: "artifact-l1", Bech32Prefix: "init", RPCAddress: "https://final-l1.invalid:26657/custom"}, config.L1Node)
	require.Equal(t, NodeConfig{ChainID: "final-l2", Bech32Prefix: "init", RPCAddress: "https://final-l2.invalid"}, config.L2Node)
	require.Equal(t, 876, config.L1StartHeight)
	require.Equal(t, 7, config.Version)
	info, err := os.Stat(configPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	daAssertMutationCommands(t, trace, filepath.Join(home, "opinit"), []string{ChallengerKeyName}, nil, false, false, 0)
	daAssertCopiedKeys(t, home, []string{"artifact-l1", "final-l2"}, []string{ChallengerKeyName})
	require.Equal(t, [][2]string{{"", filepath.Join(home, "opinit")}}, srv.created)
	// The existing success screen prunes executor logs even for Challenger.
	require.Equal(t, []service.CommandName{service.OPinitChallenger, service.OPinitExecutor}, *calls)
	require.NoFileExists(t, filepath.Join(home, "opinit", "executor.json"))
	keyfile := weaveio.NewKeyFile()
	require.NoError(t, keyfile.Load(filepath.Join(home, "opinit", "weave.keyfile.json")))
	require.Len(t, keyfile, 1)
	require.Equal(t, daMnemonic, keyfile[ChallengerKeyName].Mnemonic)
}

// Catches direct config generation/initialization losing Initia batch keys or
// replacing filesystem work when the OS daemon boundary is faked.
func TestDAFlowInitiaDirectInitializerCompletes(t *testing.T) {
	home := t.TempDir()
	trace := daInitFixtures(t, home, "", false)
	srv, calls := daInitServices(t)
	raw := []byte(`{"version":7,"l1_node":{"chain_id":"direct-l1","bech32_prefix":"init","rpc_address":"https://direct-l1.invalid","gas_price":"0.023uinit"},"l2_node":{"chain_id":"direct-l2","bech32_prefix":"init","rpc_address":"https://direct-l2.invalid","gas_price":"0umin"},"da_node":{"chain_id":"direct-l1","bech32_prefix":"init","rpc_address":"https://direct-l1.invalid","gas_price":"0.023uinit"},"l1_start_height":876}`)
	keyfile, err := GenerateMnemonicKeyfile(raw, "executor")
	require.NoError(t, err)
	require.Len(t, keyfile, 4)
	var config ExecutorConfig
	require.NoError(t, json.Unmarshal(raw, &config))
	require.NoError(t, InitializeExecutorWithConfig(config, keyfile, filepath.Join(home, "opinit"), home))
	keys := []string{BridgeExecutorKeyName, OutputSubmitterKeyName, BatchSubmitterKeyName, OracleBridgeExecutorKeyName}
	commands := daRead(t, trace)
	require.Equal(t, 4, strings.Count(commands, "keys add "))
	for _, key := range keys {
		require.True(t, strings.HasPrefix(keyfile[key].Address, "init1"))
		require.NoError(t, common.ValidateMnemonic(keyfile[key].Mnemonic))
		require.Equal(t, keyfile[key].Mnemonic+"\n", daRead(t, filepath.Join(home, "opinit", "weave-dummy", "keyring-test", key+".fixture")))
		require.Equal(t, 1, strings.Count(commands, "keys add weave-dummy "+key+" --coin-type 118 --key-type secp256k1 --recover --home "+filepath.Join(home, "opinit")+"\n"))
	}
	require.NotContains(t, commands, "keys delete ")
	require.NotContains(t, commands, "--bech32")
	require.NotContains(t, commands, "tx grant")
	daAssertCopiedKeys(t, home, []string{"direct-l1", "direct-l2"}, keys)
	var final ExecutorConfig
	path := filepath.Join(home, "opinit", "executor.json")
	require.NoError(t, json.Unmarshal([]byte(daRead(t, path)), &final))
	require.Equal(t, "init", final.DANode.Bech32Prefix)
	require.Equal(t, "direct-l1", final.DANode.ChainID)
	require.Equal(t, "https://direct-l1.invalid", final.DANode.RPCAddress)
	require.Equal(t, "0.023uinit", final.DANode.GasPrice)
	require.Equal(t, 876, final.L1StartHeight)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	require.Equal(t, [][2]string{{"", filepath.Join(home, "opinit")}}, srv.created)
	require.Equal(t, 1, srv.pruned)
	require.Equal(t, []service.CommandName{service.OPinitExecutor}, *calls)
	require.NoDirExists(t, filepath.Join(home, "opinit", "celestia-test"))
}

// Catches treating standalone setup as initialization, losing returned Init, or
// recovering twice after the checkbox/mnemonic/DA callers change signatures.
func TestDAFlowStandaloneKeySetupStillCompletes(t *testing.T) {
	home := t.TempDir()
	trace := daInitFixtures(t, home, "", false)
	srv, calls := daInitServices(t)
	daExistingKeys(t, home, BatchSubmitterKeyName)
	model, err := NewSetupBotCheckbox(daTestContext(t, daFlowState(), home))
	require.NoError(t, err)
	checkbox := model.(*SetupBotCheckbox)
	checkbox.Selected[2] = true
	model, _ = checkbox.Update(tea.KeyMsg{Type: tea.KeyEnter})
	recoverKey, ok := model.(*RecoverKeySelector)
	require.True(t, ok, "%T", model)
	recoverKey.Update(tea.KeyMsg{Type: tea.KeyDown})
	model, _ = recoverKey.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mnemonic := model.(*RecoverFromMnemonic)
	mnemonic.Text = daMnemonic
	model, _ = mnemonic.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.IsType(t, &DALayerSelector{}, model)
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.IsType(t, &SetupOPInitBots{}, model)
	model, cmd = daRunLoading(t, model, cmd)
	require.IsType(t, &TerminalState{}, model)
	require.NotNil(t, cmd)
	daAssertMutationCommands(t, trace, filepath.Join(home, "opinit"), []string{BatchSubmitterKeyName}, []string{BatchSubmitterKeyName}, true, false, 0)
	keyfile := weaveio.NewKeyFile()
	require.NoError(t, keyfile.Load(filepath.Join(home, "opinit", "weave.keyfile.json")))
	require.Len(t, keyfile, 1)
	require.Equal(t, daMnemonic, keyfile[BatchSubmitterKeyName].Mnemonic)
	require.Equal(t, daInitAddress, keyfile[BatchSubmitterKeyName].Address)
	require.Empty(t, srv.created)
	require.Empty(t, *calls)
	require.NoFileExists(t, filepath.Join(home, "opinit", "executor.json"))
}

// Catches callers dropping the new selection error or starting mutation when
// configuration discovery fails after the final key input.
func TestDAFlowKeySelectionPropagatesConfigErrors(t *testing.T) {
	for _, caller := range []string{"executor role", "challenger role", "generate", "mnemonic", "DA", "explicit"} {
		t.Run(caller, func(t *testing.T) {
			home := t.TempDir()
			trace := daInitFixtures(t, home, "", false)
			daExistingKeys(t, home, BridgeExecutorKeyName, OutputSubmitterKeyName, BatchSubmitterKeyName, ChallengerKeyName, OracleBridgeExecutorKeyName)
			daWrite(t, filepath.Join(home, "opinit", "executor.json"), "{broken", 0640)
			daWrite(t, filepath.Join(home, "opinit", "challenger.json"), "{broken", 0640)
			before := daSnapshot(t, filepath.Join(home, "opinit"))
			state := daFlowState()
			state.InitExecutorBot = caller != "challenger role"
			ctx := daTestContext(t, state, home)
			var model tea.Model
			if strings.HasSuffix(caller, "role") {
				var err error
				model, err = NewOPInitBotInitSelector(ctx)
				require.NoError(t, err)
				if caller == "challenger role" {
					model.Update(tea.KeyMsg{Type: tea.KeyDown})
				}
			} else {
				switch caller {
				case "generate":
					model = NewRecoverKeySelector(ctx, 0)
				case "mnemonic":
					input := NewRecoverFromMnemonic(ctx, 0)
					input.Text = daMnemonic
					model = input
				case "DA":
					model = NewDALayerSelector(ctx, 2)
				case "explicit":
					got, err := PrepareKey(ctx, true)
					require.Nil(t, got)
					require.ErrorContains(t, err, "failed to parse executor.json")
				}
			}
			if model != nil {
				next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
				require.Same(t, model, next)
				require.NotNil(t, cmd)
				require.Contains(t, model.View(), "failed to parse")
				require.IsType(t, tea.QuitMsg{}, cmd())
			}
			require.Equal(t, before, daSnapshot(t, filepath.Join(home, "opinit")))
			daAssertMutationCommands(t, trace, filepath.Join(home, "opinit"), nil, nil, false, false, 0)
		})
	}
}
