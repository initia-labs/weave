package minitia

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initia-labs/weave/common"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/internal/features"
	"github.com/initia-labs/weave/registry"
	"github.com/initia-labs/weave/types"
	"github.com/test-go/testify/require"
)

// Catches absent nil/target validation and an accidentally enabled Celestia build.
func TestDAPolicyRollupConfig(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config *types.MinitiaConfig
		valid  bool
	}{
		{"nil", nil, false}, {"missing bridge", &types.MinitiaConfig{}, false},
		{"empty", &types.MinitiaConfig{OpBridge: &types.OpBridge{}}, false},
		{"unknown", &types.MinitiaConfig{OpBridge: &types.OpBridge{BatchSubmissionTarget: "OTHER"}}, false},
		{"whitespace", &types.MinitiaConfig{OpBridge: &types.OpBridge{BatchSubmissionTarget: " INITIA"}}, false},
		{"lowercase", &types.MinitiaConfig{OpBridge: &types.OpBridge{BatchSubmissionTarget: "initia"}}, false},
		{"Initia", &types.MinitiaConfig{OpBridge: &types.OpBridge{BatchSubmissionTarget: "INITIA"}}, true},
		{"Celestia", &types.MinitiaConfig{OpBridge: &types.OpBridge{BatchSubmissionTarget: "CELESTIA"}}, features.CelestiaDAEnabled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateLaunchDA(tc.config)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			if tc.name == "Celestia" && !tc.valid {
				require.True(t, errors.Is(err, common.ErrCelestiaDADisabled), "policy error: %v", err)
			}
		})
	}
}

// Catches the default Celestia choice and stale Celestia state after reselection.
func TestDAFlowRollupSelection(t *testing.T) {
	daIsolateViper(t)
	wantOptions := []OpBridgeBatchSubmissionTargetOption{Initia}
	if features.CelestiaDAEnabled {
		wantOptions = []OpBridgeBatchSubmissionTargetOption{Celestia, Initia}
	}
	for i, option := range wantOptions {
		t.Run(string(option), func(t *testing.T) {
			state := *NewLaunchState()
			state.batchSubmissionIsCelestia = true
			state.opBridgeBatchSubmissionTarget = "CELESTIA"
			m := NewOpBridgeBatchSubmissionTargetSelect(weavecontext.NewAppContext(state))
			require.Equal(t, wantOptions, m.Options)
			require.Len(t, *m.Tooltips, len(wantOptions))
			require.Zero(t, m.Cursor)
			m.Update(tea.KeyMsg{Type: tea.KeyUp})
			require.True(t, m.Cursor >= 0 && m.Cursor < len(m.Options))
			m.Update(tea.KeyMsg{Type: tea.KeyDown})
			require.Zero(t, m.Cursor)
			m.Cursor = i
			next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			result := weavecontext.GetCurrentState[LaunchState](next.(*OracleEnableSelect).Ctx)
			wantTarget := "INITIA"
			if option == Celestia {
				wantTarget = "CELESTIA"
			}
			require.Equal(t, wantTarget, result.opBridgeBatchSubmissionTarget)
			require.Equal(t, option == Celestia, result.batchSubmissionIsCelestia)
			m.Ctx = weavecontext.SetTooltip(m.Ctx, true)
			m.Ctx = weavecontext.SetWindowWidth(m.Ctx, 100)
			view := m.View()
			if !features.CelestiaDAEnabled {
				require.NotContains(t, view, "recommend Celestia")
			}
		})
	}
}

type daRoundTripFunc func(*http.Request) (*http.Response, error)

func (f daRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func daJSONResponse(r *http.Request, body string) *http.Response {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}
}
func daRegistryFixture(t *testing.T) {
	old := registry.LoadedChainRegistry
	oldTestnet, oldMainnet := Testnet, Mainnet
	registry.LoadedChainRegistry = map[registry.ChainType]*registry.ChainRegistry{
		registry.InitiaL1Testnet: {ChainId: "initia-test-1", ActiveRpcs: []string{"https://test-rpc.invalid"}, ActiveLcds: []string{"https://test-lcd.invalid"}},
		registry.InitiaL1Mainnet: {ChainId: "initia-main-1", ActiveRpcs: []string{"https://main-rpc.invalid"}, ActiveLcds: []string{"https://main-lcd.invalid"}},
	}
	t.Cleanup(func() { registry.LoadedChainRegistry = old; Testnet, Mainnet = oldTestnet, oldMainnet })
}

// Catches unconditional Celestia initialization without hiding requests in cache.
func TestDAFlowRollupNetworkDoesNotFetchCelestia(t *testing.T) {
	daIsolateViper(t)
	for _, index := range []int{0, 1} {
		t.Run([]string{"testnet", "mainnet"}[index], func(t *testing.T) {
			daRegistryFixture(t)
			old := http.DefaultTransport
			calls := []string{}
			http.DefaultTransport = daRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls = append(calls, r.URL.String())
				return daJSONResponse(r, `{"chain_id":"celestia-fixture","ActiveRpcs":["https://celestia-rpc.invalid"]}`), nil
			})
			t.Cleanup(func() { http.DefaultTransport = old })
			if features.CelestiaDAEnabled {
				registry.LoadedChainRegistry[registry.CelestiaTestnet] = &registry.ChainRegistry{ChainId: "celestia-test-1", ActiveRpcs: []string{"https://celestia-test-rpc.invalid"}}
				registry.LoadedChainRegistry[registry.CelestiaMainnet] = &registry.ChainRegistry{ChainId: "celestia-main-1", ActiveRpcs: []string{"https://celestia-main-rpc.invalid"}}
			}
			m, err := NewNetworkSelect(weavecontext.NewAppContext(*NewLaunchState()))
			require.NoError(t, err)
			m.Cursor = index
			next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			require.IsType(t, &VMTypeSelect{}, next)
			state := weavecontext.GetCurrentState[LaunchState](next.(*VMTypeSelect).Ctx)
			require.Equal(t, []string{"initia-test-1", "initia-main-1"}[index], state.l1ChainId)
			require.Equal(t, []string{"https://test-rpc.invalid", "https://main-rpc.invalid"}[index], state.l1RPC)
			require.Equal(t, []string{InitiaScanTestnetURL, InitiaScanMainnetURL}[index], state.scanLink)
			require.Empty(t, calls, "network selection must not request Celestia in the default build")
			if !features.CelestiaDAEnabled {
				require.Empty(t, state.daRPC)
				require.Empty(t, state.daChainId)
			} else {
				require.Equal(t, []string{"celestia-test-1", "celestia-main-1"}[index], state.daChainId)
			}
		})
	}
}

func daLaunchState() LaunchState {
	s := *NewLaunchState()
	s.l1ChainId = "initia-test-1"
	s.l1RPC = "https://test-rpc.invalid"
	s.chainId = "rollup-1"
	s.gasDenom = "umin"
	s.moniker = "test-rollup"
	s.vmType = "Move"
	s.minitiadVersion = "v1.2.3"
	s.opBridgeSubmissionInterval = "60s"
	s.opBridgeOutputFinalizationPeriod = "3600s"
	s.opBridgeBatchSubmissionTarget = "INITIA"
	s.enableOracle = true
	s.systemKeyOperatorMnemonic = "operator words"
	s.systemKeyOperatorAddress = "init1operator"
	s.systemKeyBridgeExecutorMnemonic = "bridge words"
	s.systemKeyBridgeExecutorAddress = "init1bridge"
	s.systemKeyOutputSubmitterMnemonic = "output words"
	s.systemKeyOutputSubmitterAddress = "init1output"
	s.systemKeyBatchSubmitterMnemonic = "batch words"
	s.systemKeyBatchSubmitterAddress = "init1batch"
	s.systemKeyChallengerMnemonic = "challenger words"
	s.systemKeyChallengerAddress = "init1challenger"
	s.genesisAccounts = types.GenesisAccounts{{Address: "init1genesis", Coins: "42umin"}}
	return s
}

// Catches schema/address regressions in the builder used by the real launch path.
func TestDAFlowRollupConfigJSON(t *testing.T) {
	for _, target := range []string{"INITIA", "CELESTIA"} {
		t.Run(target, func(t *testing.T) {
			if target == "CELESTIA" && !features.CelestiaDAEnabled {
				return
			}
			s := daLaunchState()
			s.opBridgeBatchSubmissionTarget = target
			batch := "init1batch"
			if target == "CELESTIA" {
				batch = "celestia1batch"
				s.systemKeyBatchSubmitterAddress = batch
				s.batchSubmissionIsCelestia = true
			}
			cfg := buildLaunchConfig(s)
			require.NotNil(t, cfg)
			require.NoError(t, ValidateLaunchDA(cfg))
			bz, err := json.Marshal(cfg)
			require.NoError(t, err)
			expected := `{"l1_config":{"chain_id":"initia-test-1","rpc_url":"https://test-rpc.invalid","gas_prices":"0.015uinit"},"l2_config":{"chain_id":"rollup-1","denom":"umin","moniker":"test-rollup"},"op_bridge":{"output_submission_interval":"60s","output_finalization_period":"3600s","output_submission_start_height":1,"batch_submission_target":"` + target + `","enable_oracle":true},"system_keys":{"validator":{"l1_address":"init1operator","l2_address":"init1operator","mnemonic":"operator words"},"bridge_executor":{"l1_address":"init1bridge","l2_address":"init1bridge","mnemonic":"bridge words"},"output_submitter":{"l1_address":"init1output","l2_address":"init1output","mnemonic":"output words"},"batch_submitter":{"da_address":"` + batch + `","mnemonic":"batch words"},"challenger":{"l1_address":"init1challenger","l2_address":"init1challenger","mnemonic":"challenger words"}},"genesis_accounts":[{"address":"init1genesis","coins":"42umin"}]}`
			require.JSONEq(t, expected, string(bz))
			require.Empty(t, cfg.SystemKeys.BatchSubmitter.L1Address)
		})
	}
}

// Catches production advice for a provider unavailable in this build.
func TestDAFlowRollupTooltip(t *testing.T) {
	daIsolateViper(t)
	ctx := weavecontext.SetWindowWidth(weavecontext.SetTooltip(weavecontext.NewAppContext(*NewLaunchState()), true), 110)
	m := NewOpBridgeBatchSubmissionTargetSelect(ctx)
	view := m.View()
	require.Contains(t, view, "Initia L1")
	if features.CelestiaDAEnabled {
		require.Contains(t, view, "recommend Celestia")
	} else {
		require.NotContains(t, view, "recommend Celestia")
		require.Contains(t, view, "supports submissions to Initia L1.")
	}
}
