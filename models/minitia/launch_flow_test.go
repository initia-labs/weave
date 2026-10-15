package minitia

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/common"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/models/opinit_bots"
	"github.com/initia-labs/weave/service"
	"github.com/initia-labs/weave/ui"
	"github.com/test-go/testify/require"
)

type daImportTerminal struct{ initialized bool }

func (m *daImportTerminal) Init() tea.Cmd                       { m.initialized = true; return nil }
func (m *daImportTerminal) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *daImportTerminal) View() string                        { return "import complete" }

// Catches producer/consumer schema mismatches and swapped or omitted imported roles.
// The consumer receives the fake external launch's unchanged artifact, not a
// separate hand-authored consumer fixture.
func TestDAFlowLaunchArtifactConsumedByOPInit(t *testing.T) {
	daIsolateViper(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	oldClient, oldProperties := analytics.Client, analytics.GlobalEventProperties
	analytics.Client = &analytics.NoOpClient{}
	analytics.GlobalEventProperties = make(map[string]interface{})
	t.Cleanup(func() { analytics.Client, analytics.GlobalEventProperties = oldClient, oldProperties })
	oldTransport := http.DefaultTransport
	http.DefaultTransport = daRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		t.Errorf("unexpected artifact import HTTP request: %s", r.URL)
		return daJSONResponse(r, `{}`), nil
	})
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	require.NoError(t, os.MkdirAll(filepath.Join(home, common.WeaveDataDirectory), 0700))
	rollupHome := filepath.Join(home, "rollup")
	binary := filepath.Join(home, "bin", "minimove")
	daWriteFile(t, binary, `#!/bin/sh
set -eu
[ "$#" = 5 ]
[ "$1" = launch ]
[ "$2" = --with-config ]
[ "$4" = --home ]
mkdir -p "$5/config" "$5/artifacts"
cp "$3" "$5/artifacts/config.json"
printf '%s\n' '{"BRIDGE_ID":"42","EXECUTOR_L1_MONITOR_HEIGHT":"10","EXECUTOR_L2_MONITOR_HEIGHT":"1"}' > "$5/artifacts/artifacts.json"
printf '%s\n' 'inter-block-cache = true' 'minimum-gas-prices = "99umin"' > "$5/config/app.toml"
`, 0700)
	// Public BIP39 test vectors, assigned before producing the launch config.
	const bridge = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	const output = "legal winner thank year wave sausage worth useful legal winner thank yellow"
	const batch = "letter advice cage absurd amount doctor acoustic avoid letter advice cage above"
	const challenger = "zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo wrong"
	s := daLaunchState()
	s.binaryPath, s.scanLink = binary, InitiaScanTestnetURL
	s.systemKeyBridgeExecutorMnemonic, s.systemKeyOutputSubmitterMnemonic = bridge, output
	s.systemKeyBatchSubmitterMnemonic, s.systemKeyChallengerMnemonic = batch, challenger
	s.systemKeyBatchSubmitterAddress = "init16pawh0v7w996jrmtzugz3hmhq0wx6ndq5pp0dr"
	fake := &daService{}
	oldService := newService
	newService = func(command service.CommandName, vm string) (service.Service, error) {
		require.Equal(t, service.Minitia, command)
		require.Equal(t, "Move", vm)
		return fake, nil
	}
	t.Cleanup(func() { newService = oldService })
	t.Cleanup(func() { daWaitForRollytics(t, home) })
	ctx := weavecontext.SetMinitiaHome(weavecontext.NewAppContext(s), rollupHome)
	logs := []string{}
	var mu sync.Mutex
	msg := launchingMinitia(ctx, &logs, &mu)()
	end, ok := msg.(ui.EndLoading)
	require.True(t, ok, "launch returned %T: %+v", msg, msg)
	launchModel := NewLaunchingNewMinitiaLoading(ctx)
	finished, _ := launchModel.Update(end)
	require.IsType(t, &TerminalState{}, finished)
	artifactPath := filepath.Join(rollupHome, "artifacts", "config.json")
	artifact, err := os.ReadFile(artifactPath)
	require.NoError(t, err)
	launchConfig, err := os.ReadFile(filepath.Join(home, common.WeaveDataDirectory, LaunchConfigFilename))
	require.NoError(t, err)
	require.Equal(t, launchConfig, artifact)

	state := opinit_bots.OPInitBotsState{
		BotInfos: []opinit_bots.BotInfo{
			{BotName: opinit_bots.BridgeExecutor, KeyName: opinit_bots.BridgeExecutorKeyName, IsNotExist: true},
			{BotName: opinit_bots.OutputSubmitter, KeyName: opinit_bots.OutputSubmitterKeyName, IsNotExist: true},
			{BotName: opinit_bots.BatchSubmitter, KeyName: opinit_bots.BatchSubmitterKeyName, IsNotExist: true},
			{BotName: opinit_bots.Challenger, KeyName: opinit_bots.ChallengerKeyName, IsNotExist: true},
			{BotName: opinit_bots.OracleBridgeExecutor, KeyName: opinit_bots.OracleBridgeExecutorKeyName, IsNotExist: true},
		},
		SetupOpinitResponses: make(map[opinit_bots.BotName]string),
	}
	importCtx := weavecontext.SetMinitiaHome(weavecontext.NewAppContext(state), rollupHome)
	callbackCalled := false
	terminal := &daImportTerminal{}
	model, err := opinit_bots.ProcessMinitiaConfig(importCtx, func(nextCtx context.Context) (tea.Model, error) {
		callbackCalled = true
		got := weavecontext.GetCurrentState[opinit_bots.OPInitBotsState](nextCtx)
		require.True(t, got.AddMinitiaConfig)
		require.Equal(t, "INITIA", got.MinitiaConfig.OpBridge.BatchSubmissionTarget)
		require.Empty(t, got.MinitiaConfig.SystemKeys.BatchSubmitter.L1Address)
		require.Equal(t, "init16pawh0v7w996jrmtzugz3hmhq0wx6ndq5pp0dr", got.MinitiaConfig.SystemKeys.BatchSubmitter.DAAddress)
		for i, want := range []string{bridge, output, batch, challenger} {
			require.Equal(t, want, got.BotInfos[i].Mnemonic, "role %s", got.BotInfos[i].BotName)
			require.False(t, got.BotInfos[i].IsNotExist)
			require.True(t, got.BotInfos[i].IsNewKey())
		}
		require.Equal(t, "Initia", got.BotInfos[2].DALayer)
		require.Empty(t, got.BotInfos[4].Mnemonic)
		require.True(t, got.BotInfos[4].IsNotExist)
		return terminal, nil
	})
	require.NoError(t, err)
	importer, ok := model.(*opinit_bots.ProcessingMinitiaConfig)
	require.True(t, ok, "actual artifact discovery returned %T", model)
	next, cmd := importer.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.True(t, callbackCalled, "Yes import did not reach the next model: %s", importer.View())
	require.True(t, next == terminal)
	require.True(t, terminal.initialized)
	require.Nil(t, cmd)
	after, err := os.ReadFile(artifactPath)
	require.NoError(t, err)
	require.Equal(t, artifact, after)
	daWaitForRollytics(t, home)
}

type daService struct {
	versions, homes []string
	starts, prunes  int
	startArgs       [][]string
}

func (s *daService) Create(v, h string) error {
	s.versions = append(s.versions, v)
	s.homes = append(s.homes, h)
	return nil
}
func (s *daService) Start(args ...string) error {
	s.starts++
	s.startArgs = append(s.startArgs, args)
	return nil
}
func (s *daService) PruneLogs() error                                 { s.prunes++; return nil }
func (s *daService) Log(int) error                                    { return nil }
func (s *daService) Stop() error                                      { return nil }
func (s *daService) Restart() error                                   { return nil }
func (s *daService) GetServiceFile() (string, error)                  { return "", nil }
func (s *daService) GetServiceBinaryAndHome() (string, string, error) { return "", "", nil }
func daWriteFile(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
	require.NoError(t, os.WriteFile(path, []byte(body), mode))
}

// Catches post-launch updates to the wrong home and incomplete Create/Start finalization.
func TestDAFlowInitiaLaunchCompletes(t *testing.T) {
	for _, homeKind := range []string{"default", "custom"} {
		for _, existing := range []bool{false, true} {
			name := homeKind + "/new"
			if existing {
				name = homeKind + "/existing"
			}
			t.Run(name, func(t *testing.T) {
				daIsolateViper(t)
				home := t.TempDir()
				t.Setenv("HOME", home)
				defaultHome := filepath.Join(home, ".minitia")
				rollupHome := defaultHome
				if homeKind == "custom" {
					rollupHome = filepath.Join(home, "custom-rollup")
					daWriteFile(t, filepath.Join(defaultHome, "sentinel"), "preserve", 0600)
				}
				require.NoError(t, os.MkdirAll(filepath.Join(home, common.WeaveDataDirectory), 0700))
				binary := filepath.Join(home, "bin", "minimove")
				daWriteFile(t, binary, `#!/bin/sh
set -eu
[ "$#" = 5 ]
[ "$1" = launch ]
[ "$2" = --with-config ]
[ "$4" = --home ]
mkdir -p "$5/config" "$5/artifacts"
cp "$3" "$5/artifacts/config.json"
printf '%s\n' '{"BRIDGE_ID":"42","EXECUTOR_L1_MONITOR_HEIGHT":"10","EXECUTOR_L2_MONITOR_HEIGHT":"1"}' > "$5/artifacts/artifacts.json"
printf '%s\n' 'inter-block-cache = true' 'minimum-gas-prices = "99umin"' > "$5/config/app.toml"
`, 0700)
				s := daLaunchState()
				s.binaryPath = binary
				s.scanLink = InitiaScanTestnetURL
				configPath := filepath.Join(home, common.WeaveDataDirectory, LaunchConfigFilename)
				var original []byte
				if existing {
					s.launchFromExistingConfig = true
					s.existingConfigPath = filepath.Join(home, "existing.json")
					original = []byte(`{"l1_config":{"chain_id":"initia-test-1","rpc_url":"https://test-rpc.invalid"},"l2_config":{"chain_id":"rollup-1","denom":"umin"},"op_bridge":{"batch_submission_target":"INITIA"},"future_field":{"preserved":true}}`)
					daWriteFile(t, s.existingConfigPath, string(original), 0600)
					configPath = s.existingConfigPath
				}
				fake := &daService{}
				old := newService
				factories := 0
				newService = func(command service.CommandName, vm string) (service.Service, error) {
					require.Equal(t, service.Minitia, command)
					require.Equal(t, "Move", vm)
					factories++
					return fake, nil
				}
				t.Cleanup(func() { newService = old })
				ctx := weavecontext.SetMinitiaHome(weavecontext.NewAppContext(s), rollupHome)
				logs := []string{}
				var mu sync.Mutex
				// Cleanup also waits when a later assertion fails, before HOME restoration.
				t.Cleanup(func() { daWaitForRollytics(t, home) })
				msg := launchingMinitia(ctx, &logs, &mu)()
				end, ok := msg.(ui.EndLoading)
				require.True(t, ok, "launch returned %T: %+v", msg, msg)
				m := NewLaunchingNewMinitiaLoading(ctx)
				next, _ := m.Update(end)
				require.IsType(t, &TerminalState{}, next)
				require.Equal(t, []string{"minimove@v1.2.3"}, fake.versions)
				require.Equal(t, []string{rollupHome}, fake.homes)
				require.Equal(t, 1, fake.starts)
				require.Len(t, fake.startArgs, 1)
				require.Empty(t, fake.startArgs[0])
				require.Equal(t, 1, fake.prunes)
				require.Equal(t, 2, factories)
				app, err := os.ReadFile(filepath.Join(rollupHome, "config", "app.toml"))
				require.NoError(t, err)
				require.Contains(t, string(app), `inter-block-cache = "false"`)
				require.Contains(t, string(app), `minimum-gas-prices = "0umin"`)
				cfg, err := os.ReadFile(configPath)
				require.NoError(t, err)
				require.Contains(t, string(cfg), `"batch_submission_target"`)
				require.Contains(t, string(cfg), `"INITIA"`)
				info, err := os.Stat(configPath)
				require.NoError(t, err)
				require.Equal(t, os.FileMode(0600), info.Mode().Perm())
				artifact, err := os.ReadFile(filepath.Join(rollupHome, "artifacts", "config.json"))
				require.NoError(t, err)
				require.Equal(t, cfg, artifact)
				artifact, err = os.ReadFile(filepath.Join(rollupHome, "artifacts", "artifacts.json"))
				require.NoError(t, err)
				require.Contains(t, string(artifact), `"BRIDGE_ID":"42"`)
				if existing {
					require.Equal(t, original, cfg)
				}
				if homeKind == "custom" {
					sentinel, err := os.ReadFile(filepath.Join(defaultHome, "sentinel"))
					require.NoError(t, err)
					require.Equal(t, "preserve", string(sentinel))
				}
				daWaitForRollytics(t, home)
				mu.Lock()
				logText := strings.Join(logs, "\n")
				mu.Unlock()
				require.NotContains(t, logText, "error")
				require.NotContains(t, next.View(), "Failed to start")
				require.NotContains(t, next.View(), "Invalid OS")
			})
		}
	}
}
func daWaitForRollytics(t *testing.T, home string) {
	t.Helper()
	completed := func() bool {
		bz, err := os.ReadFile(filepath.Join(home, common.RollyticsConfigPath))
		return err == nil && strings.HasSuffix(string(bz), "# METRICS_PORT=9090                     # default: 9090\n")
	}
	deadline := time.Now().Add(3 * time.Second)
	for !completed() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	require.True(t, completed(), "Rollytics config was not completed")
}

// Catches validation bypasses in both launch entry paths before binary/service side effects.
func TestDAPolicyRollupLaunchRevalidates(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "new", true: "existing"}[existing], func(t *testing.T) {
			daIsolateViper(t)
			home := t.TempDir()
			t.Setenv("HOME", home)
			require.NoError(t, os.MkdirAll(filepath.Join(home, common.WeaveDataDirectory), 0700))
			s := daLaunchState()
			s.opBridgeBatchSubmissionTarget = "OTHER"
			s.binaryPath = filepath.Join(home, "must-not-run")
			marker := filepath.Join(home, "binary-called")
			daWriteFile(t, s.binaryPath, "#!/bin/sh\ntouch '"+marker+"'\nexit 1\n", 0700)
			if existing {
				s.launchFromExistingConfig = true
				s.existingConfigPath = filepath.Join(home, "existing.json")
				daWriteFile(t, s.existingConfigPath, `{"op_bridge":{"batch_submission_target":"OTHER"},"future":42}`, 0600)
			}
			ctx := weavecontext.SetMinitiaHome(weavecontext.NewAppContext(s), filepath.Join(home, "rollup"))
			logs := []string{}
			var mu sync.Mutex
			msg := launchingMinitia(ctx, &logs, &mu)()
			errMsg, ok := msg.(ui.NonRetryableErrorLoading)
			require.True(t, ok, "%T", msg)
			require.Contains(t, errMsg.Err.Error(), "DA target")
			_, err := os.Stat(marker)
			require.True(t, os.IsNotExist(err))
			_, err = os.Stat(filepath.Join(home, common.WeaveDataDirectory, LaunchConfigFilename))
			require.True(t, os.IsNotExist(err))
		})
	}
}
