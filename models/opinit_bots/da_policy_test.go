package opinit_bots

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/weave/common"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/internal/features"
	weaveio "github.com/initia-labs/weave/io"
	"github.com/initia-labs/weave/registry"
	"github.com/initia-labs/weave/types"
	"github.com/initia-labs/weave/ui"
)

// Catches missing prefix checks, including an incorrect disabled-batch exemption.
func TestDAPolicyExecutorConfig(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		for _, tc := range []struct {
			prefix string
			valid  bool
		}{
			{"init", true}, {"celestia", features.CelestiaDAEnabled}, {"", false},
			{"other", false}, {"INIT", false}, {" init", false},
		} {
			t.Run(fmt.Sprintf("%q/disabled=%t", tc.prefix, disabled), func(t *testing.T) {
				err := ValidateExecutorDA(ExecutorConfig{DANode: NodeSettings{Bech32Prefix: tc.prefix}, DisableBatchSubmitter: disabled})
				if tc.valid {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
				}
				if tc.prefix == "celestia" && !tc.valid {
					require.True(t, errors.Is(err, common.ErrCelestiaDADisabled), "%v", err)
				}
			})
		}
	}
}

// Catches treating IsSetup/IsNotExist as pending and failing to scan all pending keys.
func TestDAPolicyPendingKeys(t *testing.T) {
	for _, tc := range []struct {
		name  string
		info  BotInfo
		valid bool
	}{
		{"inactive Celestia", BotInfo{BotName: BatchSubmitter, DALayer: "Celestia", IsSetup: true, IsNotExist: true}, true},
		{"inactive unknown", BotInfo{BotName: BatchSubmitter, DALayer: "other"}, true},
		{"generate Initia", BotInfo{BotName: BatchSubmitter, DALayer: "Initia", IsGenerateKey: true}, true},
		{"generate Celestia", BotInfo{BotName: BatchSubmitter, DALayer: "Celestia", IsGenerateKey: true}, features.CelestiaDAEnabled},
		{"recover Celestia", BotInfo{BotName: BatchSubmitter, DALayer: "Celestia", Mnemonic: "pending mnemonic"}, features.CelestiaDAEnabled},
		{"empty", BotInfo{BotName: BatchSubmitter, IsGenerateKey: true}, false},
		{"unknown", BotInfo{BotName: BatchSubmitter, DALayer: "other", Mnemonic: "pending mnemonic"}, false},
		{"wrong case", BotInfo{BotName: BatchSubmitter, DALayer: "initia", IsGenerateKey: true}, false},
		{"other role", BotInfo{BotName: Challenger, DALayer: "Celestia", IsGenerateKey: true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePendingBotKeys([]BotInfo{{BotName: BridgeExecutor, IsGenerateKey: true}, tc.info})
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			if strings.Contains(tc.name, "Celestia") && !tc.valid {
				require.True(t, errors.Is(err, common.ErrCelestiaDADisabled), "%v", err)
			}
		})
	}
}

// Catches bool-based DA decisions and mismatched key/config DA in either build.
func TestDAPolicyExecutorKeyDAMatchesConfig(t *testing.T) {
	for _, tc := range []struct {
		name, prefix, layer      string
		executor, pending, valid bool
	}{
		{"Initia match", "init", "Initia", true, true, true},
		{"Celestia match", "celestia", "Celestia", true, true, features.CelestiaDAEnabled},
		{"Initia key Celestia config", "celestia", "Initia", true, true, false},
		{"Celestia key Initia config", "init", "Celestia", true, true, false},
		{"inactive batch", "init", "Celestia", true, false, true},
		{"missing config", "", "Initia", true, true, false},
		{"unknown config", "other", "Initia", true, true, false},
		{"challenger", "", "Celestia", false, false, true},
		{"standalone", "", "Initia", false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := daTestState()
			state.InitExecutorBot = tc.executor
			state.InitChallengerBot = !tc.executor
			state.botConfig["da_node.bech32_prefix"] = tc.prefix
			state.daIsCelestia = tc.prefix != "celestia" // Deliberately stale; the actual prefix is authoritative.
			state.BotInfos = []BotInfo{{BotName: BatchSubmitter, DALayer: tc.layer, IsGenerateKey: tc.pending}}
			err := validateExecutorStateDA(state)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func daTestState() OPInitBotsState {
	return OPInitBotsState{botConfig: make(map[string]string), SetupOpinitResponses: make(map[BotName]string), weave: types.NewWeaveState()}
}

type daRoundTripFunc func(*http.Request) (*http.Response, error)

func (f daRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func daResponse(r *http.Request, body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}
}
func daUseTransport(t *testing.T, f daRoundTripFunc) {
	t.Helper()
	old := http.DefaultTransport
	http.DefaultTransport = f
	t.Cleanup(func() { http.DefaultTransport = old })
}

const daRelease = `[{"tag_name":"v1.2.3","prerelease":false,"assets":[{"browser_download_url":"https://fixture.invalid/Darwin_aarch64.tar.gz"},{"browser_download_url":"https://fixture.invalid/Darwin_x86_64.tar.gz"},{"browser_download_url":"https://fixture.invalid/Linux_x86_64.tar.gz"},{"browser_download_url":"https://fixture.invalid/Linux_aarch64.tar.gz"}]}]`

func daWrite(t *testing.T, path, content string, mode fs.FileMode) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
	require.NoError(t, os.WriteFile(path, []byte(content), mode))
	// Preserve the requested fixture permissions regardless of the process umask.
	require.NoError(t, os.Chmod(path, mode))
}

type daFileSnapshot struct {
	Mode  fs.FileMode
	Bytes string
}

func daSnapshot(t *testing.T, root string) map[string]daFileSnapshot {
	t.Helper()
	result := make(map[string]daFileSnapshot)
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		value := daFileSnapshot{Mode: info.Mode()}
		if !entry.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value.Bytes = string(data)
		}
		result[rel] = value
		return nil
	}))
	return result
}
func daError(msg tea.Msg) error {
	switch m := msg.(type) {
	case ui.NonRetryableErrorLoading:
		return m.Err
	case ui.ErrorLoading:
		return m.Err
	default:
		return nil
	}
}
func daTestContext(t *testing.T, state OPInitBotsState, home string) context.Context {
	t.Helper()
	ctx := weavecontext.NewAppContext(state)
	ctx = weavecontext.SetOPInitHome(ctx, filepath.Join(home, "opinit"))
	return weavecontext.SetMinitiaHome(ctx, filepath.Join(home, "minitia"))
}

// Catches guards placed after binary lookup, earlier keys, DB deletion or keyring copies.
func TestDAPolicyDirectInitializersRejectBeforeWork(t *testing.T) {
	for _, prefix := range []string{"celestia", "other", ""} {
		t.Run(fmt.Sprintf("prefix=%q", prefix), func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			requests := 0
			daUseTransport(t, func(r *http.Request) (*http.Response, error) { requests++; return daResponse(r, `[]`), nil })
			config := ExecutorConfig{DANode: NodeSettings{Bech32Prefix: prefix}, DisableBatchSubmitter: true}
			err := InitializeExecutorWithConfig(config, weaveio.NewKeyFile(), filepath.Join(home, "opinit"), home)
			raw, marshalErr := json.Marshal(config)
			require.NoError(t, marshalErr)
			keys, keyErr := GenerateMnemonicKeyfile(raw, "executor")
			if prefix == "celestia" && features.CelestiaDAEnabled {
				require.ErrorContains(t, err, "no releases found") // Allowed through policy to binary discovery.
				require.NoError(t, keyErr)
				require.Len(t, keys, 4)
				require.Equal(t, 1, requests)
			} else {
				require.Error(t, err)
				require.Error(t, keyErr)
				require.Nil(t, keys)
				if prefix == "celestia" {
					require.True(t, errors.Is(err, common.ErrCelestiaDADisabled), "%v", err)
					require.True(t, errors.Is(keyErr, common.ErrCelestiaDADisabled), "%v", keyErr)
				}
				require.Zero(t, requests, "policy must precede even binary discovery")
			}
		})
	}
	for _, missing := range []bool{false, true} {
		for _, generate := range []bool{false, true} {
			t.Run(fmt.Sprintf("keys/missing=%t/generate=%t", missing, generate), func(t *testing.T) {
				home := t.TempDir()
				t.Setenv("HOME", home)
				trace := filepath.Join(home, "key-commands")
				t.Setenv("WEAVE_TEST_OPINIT_TRACE", trace)
				daWrite(t, GetBinaryPath(home, "v1.2.3"), "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$WEAVE_TEST_OPINIT_TRACE\"\nif [ \"$2\" = show ]; then exit 1; fi\nprintf 'address: fixture\\nmnemonic fixture\\n'\n", 0700)
				requests := 0
				daUseTransport(t, func(r *http.Request) (*http.Response, error) {
					requests++
					require.Equal(t, "/repos/initia-labs/opinit-bots/releases", r.URL.Path)
					return daResponse(r, daRelease), nil
				})
				state := daTestState()
				batch := BotInfo{BotName: BatchSubmitter, KeyName: BatchSubmitterKeyName, DALayer: "Celestia", IsGenerateKey: generate}
				if !generate {
					batch.Mnemonic = "pending batch mnemonic"
				}
				state.BotInfos = []BotInfo{{BotName: BridgeExecutor, KeyName: BridgeExecutorKeyName, IsGenerateKey: true}, batch}
				ctx := daTestContext(t, state, home)
				before := daSnapshot(t, home)
				var msg tea.Msg
				if missing {
					msg = WaitSetupOPInitBotsMissingKey(ctx)()
				} else {
					msg = WaitSetupOPInitBots(ctx)()
				}
				if features.CelestiaDAEnabled {
					require.IsType(t, ui.EndLoading{}, msg)
					data, err := os.ReadFile(trace)
					require.NoError(t, err)
					require.Contains(t, string(data), "keys add weave-dummy weave_bridge_executor")
					require.Contains(t, string(data), "keys add weave-dummy weave_batch_submitter")
					require.Contains(t, string(data), "--bech32 celestia")
				} else {
					require.True(t, errors.Is(daError(msg), common.ErrCelestiaDADisabled), "got %T: %v", msg, daError(msg))
					require.Zero(t, requests)
					require.Equal(t, before, daSnapshot(t, home), "no earlier BridgeExecutor key may run")
				}
			})
		}
	}
	for _, replace := range []bool{false, true} {
		t.Run(fmt.Sprintf("final/replace=%t", replace), func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			state := daTestState()
			state.InitExecutorBot = true
			state.ReplaceBotConfig = replace
			state.botConfig["da_node.bech32_prefix"] = "celestia"
			state.botConfig["l1_node.chain_id"] = "l1"
			state.botConfig["l2_node.chain_id"] = "l2"
			state.isDeleteDB = true
			state.dbPath = filepath.Join(home, "opinit", "executor.db")
			daWrite(t, filepath.Join(state.dbPath, "sentinel"), "database", 0640)
			daWrite(t, filepath.Join(home, "opinit", "l1", "keyring", "key"), "existing key", 0600)
			before := daSnapshot(t, home)
			msg := WaitStartingInitBot(daTestContext(t, state, home))()
			if features.CelestiaDAEnabled {
				require.ErrorContains(t, daError(msg), "failed to copy dummy key for l1")
			} else {
				require.True(t, errors.Is(daError(msg), common.ErrCelestiaDADisabled), "%v", daError(msg))
				require.Equal(t, before, daSnapshot(t, home))
			}
		})
	}
}

func daInitiaRegistry(t *testing.T) {
	t.Helper()
	old := registry.LoadedChainRegistry
	registry.LoadedChainRegistry = map[registry.ChainType]*registry.ChainRegistry{
		registry.InitiaL1Testnet: {ChainId: "existing-l1", ActiveRpcs: []string{"https://l1-custom.invalid"}},
	}
	t.Cleanup(func() { registry.LoadedChainRegistry = old })
}

func daCurrentExecutorSelector(t *testing.T, home, prefix string, withDB bool) (*UseCurrentConfigSelector, string) {
	t.Helper()
	path := filepath.Join(home, "opinit", "executor.json")
	original := fmt.Sprintf(`{
 "l1_node":{"chain_id":"existing-l1","rpc_address":"https://l1-custom.invalid","gas_price":"0.023uinit"},
 "l2_node":{"chain_id":"existing-l2","rpc_address":"https://l2-custom.invalid:26657/custom"},
 "da_node":{"chain_id":"existing-da","rpc_address":"https://da-custom.invalid","bech32_prefix":%q,"gas_price":"0.031uinit"},
 "future_field":{"preserve":true},"l1_start_height":123
}`, prefix)
	daWrite(t, path, original, 0640)
	if withDB {
		daWrite(t, filepath.Join(home, "opinit", "executor.db", "sentinel"), "existing DB", 0600)
	}
	state := daTestState()
	state.daIsCelestia = true // Existing Initia config must clear a stale previous selection.
	ctx := daTestContext(t, state, home)
	model, err := OPInitBotInitSelectExecutor(ctx)
	require.NoError(t, err)
	if withDB {
		db, ok := model.(*DeleteDBSelector)
		require.True(t, ok, "%T", model)
		model, _ = db.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	selector, ok := model.(*UseCurrentConfigSelector)
	require.True(t, ok, "%T", model)
	return selector, original
}

// Catches chain-only config loading on both paths and queries sent to an empty/default RPC.
func TestDAFlowInitiaCurrentConfigPreservesRPC(t *testing.T) {
	for _, withDB := range []bool{false, true} {
		t.Run(fmt.Sprintf("db=%t", withDB), func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			daInitiaRegistry(t)
			trace := filepath.Join(home, "query-commands")
			t.Setenv("WEAVE_TEST_QUERY_TRACE", trace)
			daWrite(t, filepath.Join(home, common.WeaveDataDirectory, "minievm@v1.2.3", "minitiad"), `#!/bin/sh
printf '%s\n' "$*" >> "$WEAVE_TEST_QUERY_TRACE"
[ "$5" = 'https://l2-custom.invalid:26657/custom' ] || exit 46
case "$3" in
 next-l1-sequence) printf '{"next_l1_sequence":"1"}\n' ;;
 bridge-info) printf '{"bridge_info":{"bridge_id":"17"}}\n' ;;
 *) exit 47 ;;
esac
`, 0755)
			graphql := 0
			daUseTransport(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/repos/initia-labs/minievm/releases" {
					return daResponse(r, daRelease), nil
				}
				require.Equal(t, http.MethodPost, r.Method, "unexpected request %s", r.URL)
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.Contains(t, string(body), `"bridgeId":"17"`)
				require.Contains(t, string(body), "CreateBridgeHeight")
				graphql++
				return daResponse(r, `{"data":{"transaction_events":[{"block_height":876}]}}`), nil
			})
			selector, original := daCurrentExecutorSelector(t, home, "init", withDB)
			require.NotContains(t, selector.View(), "<nil>")
			state := weavecontext.GetCurrentState[OPInitBotsState](selector.Ctx)
			require.False(t, state.daIsCelestia)
			want := map[string]string{"l1_node.chain_id": "existing-l1", "l1_node.rpc_address": "https://l1-custom.invalid", "l1_node.gas_price": "0.023uinit", "l2_node.chain_id": "existing-l2", "l2_node.rpc_address": "https://l2-custom.invalid:26657/custom", "da_node.chain_id": "existing-da", "da_node.rpc_address": "https://da-custom.invalid", "da_node.bech32_prefix": "init", "da_node.gas_price": "0.031uinit"}
			for key, value := range want {
				assert.Equal(t, value, state.botConfig[key], key)
			}
			next, command := selector.Update(tea.KeyMsg{Type: tea.KeyEnter})
			loading, ok := next.(*FetchL1StartHeightLoading)
			require.True(t, ok, "%T", next)
			require.False(t, weavecontext.GetCurrentState[OPInitBotsState](loading.Ctx).ReplaceBotConfig)
			require.NotNil(t, command)
			batch, ok := command().(tea.BatchMsg)
			require.True(t, ok)
			require.Len(t, batch, 2)
			msg := batch[1]()
			end, ok := msg.(ui.EndLoading)
			require.True(t, ok, "got %T: %v", msg, daError(msg))
			require.Equal(t, 876, weavecontext.GetCurrentState[OPInitBotsState](end.Ctx).L1StartHeight)
			require.Equal(t, 1, graphql)
			commands, err := os.ReadFile(trace)
			require.NoError(t, err)
			require.Equal(t, "query opchild next-l1-sequence --node https://l2-custom.invalid:26657/custom --output json\nquery opchild bridge-info --node https://l2-custom.invalid:26657/custom --output json\n", string(commands))
			data, err := os.ReadFile(filepath.Join(home, "opinit", "executor.json"))
			require.NoError(t, err)
			require.Equal(t, original, string(data))
			info, err := os.Stat(filepath.Join(home, "opinit", "executor.json"))
			require.NoError(t, err)
			require.Equal(t, fs.FileMode(0640), info.Mode().Perm())
		})
	}
}

// Catches rejection after navigation/state mutation and ensures explicit replacement remains available.
func TestDAPolicyCurrentConfigValidationBeforeNavigation(t *testing.T) {
	for _, withDB := range []bool{false, true} {
		for _, prefix := range []string{"celestia", "other", ""} {
			t.Run(fmt.Sprintf("db=%t/prefix=%q", withDB, prefix), func(t *testing.T) {
				home := t.TempDir()
				t.Setenv("HOME", home)
				selector, _ := daCurrentExecutorSelector(t, home, prefix, withDB)
				require.Equal(t, prefix == "celestia", weavecontext.GetCurrentState[OPInitBotsState](selector.Ctx).daIsCelestia)
				before := weavecontext.GetCurrentState[OPInitBotsState](selector.Ctx).Clone()
				pages := len(weavecontext.GetPageStack[OPInitBotsState](selector.Ctx))
				files := daSnapshot(t, home)
				next, command := selector.Update(tea.KeyMsg{Type: tea.KeyEnter})
				if prefix == "celestia" && features.CelestiaDAEnabled {
					require.IsType(t, &FetchL1StartHeightLoading{}, next)
					require.NotNil(t, command)
				} else {
					require.Same(t, selector, next)
					require.Nil(t, command)
					if prefix == "celestia" {
						require.Contains(t, selector.View(), "Celestia DA is disabled")
					} else {
						require.Contains(t, selector.View(), "unsupported DA prefix")
					}
					require.Equal(t, pages, len(weavecontext.GetPageStack[OPInitBotsState](selector.Ctx)))
					require.Equal(t, before, weavecontext.GetCurrentState[OPInitBotsState](selector.Ctx).Clone())
					state := weavecontext.GetCurrentState[OPInitBotsState](selector.Ctx)
					state.MinitiaConfig = &types.MinitiaConfig{} // Selects the next input page without registry traffic.
					selector.Ctx = weavecontext.SetCurrentState(selector.Ctx, state)
					selector.Update(tea.KeyMsg{Type: tea.KeyDown})
					next, _ = selector.Update(tea.KeyMsg{Type: tea.KeyEnter})
					require.IsType(t, &PrefillMinitiaConfig{}, next)
					require.True(t, weavecontext.GetCurrentState[OPInitBotsState](next.(*PrefillMinitiaConfig).Ctx).ReplaceBotConfig)
				}
				require.Equal(t, files, daSnapshot(t, home))
			})
		}
	}
}

// Catches a missing state/config consistency check at each mutation entrypoint.
func TestDAPolicyExecutorStateGuardsBeforeWork(t *testing.T) {
	for _, tc := range []struct{ prefix, layer string }{{"init", "Celestia"}, {"celestia", "Initia"}, {"", "Initia"}, {"other", "Initia"}} {
		for _, entry := range []string{"setup", "missing", "start", "current"} {
			t.Run(fmt.Sprintf("%s/%q/%s", entry, tc.prefix, tc.layer), func(t *testing.T) {
				home := t.TempDir()
				t.Setenv("HOME", home)
				requests := 0
				daUseTransport(t, func(r *http.Request) (*http.Response, error) { requests++; return daResponse(r, `[]`), nil })
				state := daTestState()
				state.InitExecutorBot = true
				state.botConfig["da_node.bech32_prefix"] = tc.prefix
				state.BotInfos = []BotInfo{{BotName: BridgeExecutor, KeyName: BridgeExecutorKeyName, IsGenerateKey: true}, {BotName: BatchSubmitter, KeyName: BatchSubmitterKeyName, DALayer: tc.layer, IsGenerateKey: true}}
				state.isDeleteDB = true
				state.dbPath = filepath.Join(home, "opinit", "executor.db")
				daWrite(t, filepath.Join(state.dbPath, "sentinel"), "existing DB", 0600)
				ctx := daTestContext(t, state, home)
				before := daSnapshot(t, home)
				if entry == "current" {
					selector, err := NewUseCurrentConfigSelector(ctx, "executor")
					require.NoError(t, err)
					next, command := selector.Update(tea.KeyMsg{Type: tea.KeyEnter})
					require.Same(t, selector, next)
					require.Nil(t, command)
					require.Empty(t, weavecontext.GetPageStack[OPInitBotsState](selector.Ctx))
				} else {
					var msg tea.Msg
					switch entry {
					case "setup":
						msg = WaitSetupOPInitBots(ctx)()
					case "missing":
						msg = WaitSetupOPInitBotsMissingKey(ctx)()
					case "start":
						msg = WaitStartingInitBot(ctx)()
					}
					err := daError(msg)
					require.Error(t, err)
					if features.CelestiaDAEnabled && tc.prefix != "" && tc.prefix != "other" {
						require.Contains(t, err.Error(), "does not match executor DA prefix")
					}
				}
				require.Zero(t, requests)
				require.Equal(t, before, daSnapshot(t, home))
			})
		}
	}
}

// Catches accidentally applying executor DA requirements to Challenger configuration.
func TestDAPolicyChallengerConfigHasNoExecutorDARequirement(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	keys, err := GenerateMnemonicKeyfile([]byte(`{"l1_node":{"chain_id":"l1"},"l2_node":{"chain_id":"l2"}}`), "challenger")
	require.NoError(t, err)
	require.Len(t, keys, 1)
	state := daTestState()
	state.InitChallengerBot = true
	state.BotInfos = []BotInfo{{BotName: BatchSubmitter, DALayer: "Celestia", IsSetup: true, IsNotExist: true}}
	selector, err := NewUseCurrentConfigSelector(daTestContext(t, state, home), "challenger")
	require.NoError(t, err)
	next, command := selector.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.IsType(t, &FetchL1StartHeightLoading{}, next)
	require.NotNil(t, command)
}
