package opinit_bots

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/internal/features"
	"github.com/initia-labs/weave/registry"
	"github.com/initia-labs/weave/types"
	"github.com/stretchr/testify/require"
)

const daMnemonic = "imitate sick vibrant bonus weather spice pave announce direct impulse strategy math"
const daOutputMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
const daChallengerMnemonic = "letter advice cage absurd amount doctor acoustic avoid letter advice cage above"
const daInitAddress = "init16pawh0v7w996jrmtzugz3hmhq0wx6ndq5pp0dr"
const daCelestiaAddress = "celestia16pawh0v7w996jrmtzugz3hmhq0wx6ndqtasvuv"

func daArtifact() *types.MinitiaConfig {
	return &types.MinitiaConfig{
		L1Config: &types.L1Config{ChainID: "artifact-l1", RpcUrl: "https://artifact-l1.invalid", GasPrices: "0.025uinit"},
		L2Config: &types.L2Config{ChainID: "artifact-l2", Denom: "umin"},
		OpBridge: &types.OpBridge{BatchSubmissionTarget: "INITIA"},
		SystemKeys: &types.SystemKeys{
			BridgeExecutor:  &types.SystemAccount{Mnemonic: daMnemonic, L1Address: daInitAddress},
			OutputSubmitter: &types.SystemAccount{Mnemonic: daOutputMnemonic, L1Address: daInitAddress},
			BatchSubmitter:  &types.SystemAccount{Mnemonic: daMnemonic, DAAddress: daInitAddress},
			Challenger:      &types.SystemAccount{Mnemonic: daChallengerMnemonic, L1Address: daInitAddress},
		},
	}
}

func daFlowState() OPInitBotsState {
	state := daTestState()
	state.BotInfos = append([]BotInfo(nil), BotInfos...)
	return state
}

// No Celestia cache is installed in the default build; any eager HTTP lookup fails.
func daFlowFixtures(t *testing.T) {
	t.Helper()
	daUseTransport(t, func(r *http.Request) (*http.Response, error) {
		t.Errorf("unexpected HTTP request: %s", r.URL)
		return nil, fmt.Errorf("offline fixture")
	})
	old := registry.LoadedChainRegistry
	registry.LoadedChainRegistry = map[registry.ChainType]*registry.ChainRegistry{
		registry.InitiaL1Testnet: {ChainId: "test-l1", Bech32Prefix: "init", ActiveRpcs: []string{"https://test-l1.invalid"}, Fees: registry.Fees{FeeTokens: []registry.FeeTokens{{Denom: "uinit", FixedMinGasPrice: 0.015}}}},
		registry.InitiaL1Mainnet: {ChainId: "main-l1", Bech32Prefix: "init", ActiveRpcs: []string{"https://main-l1.invalid"}, Fees: registry.Fees{FeeTokens: []registry.FeeTokens{{Denom: "uinit", FixedMinGasPrice: 0.025}}}},
	}
	t.Cleanup(func() { registry.LoadedChainRegistry = old })
	oldTestnet, oldMainnet := L1PrefillOptionTestnet, L1PrefillOptionMainnet
	oldExecutor, oldChallenger := make([]Field, len(defaultExecutorFields)), make([]Field, len(defaultChallengerFields))
	for i, f := range defaultExecutorFields {
		oldExecutor[i] = *f
	}
	for i, f := range defaultChallengerFields {
		oldChallenger[i] = *f
	}
	t.Cleanup(func() {
		L1PrefillOptionTestnet, L1PrefillOptionMainnet = oldTestnet, oldMainnet
		for i, f := range oldExecutor {
			*defaultExecutorFields[i] = f
		}
		for i, f := range oldChallenger {
			*defaultChallengerFields[i] = f
		}
	})
}

// Catches real-shaped modern artifacts being treated as Celestia and invalid imports accepted.
func TestDAFlowArtifactImport(t *testing.T) {
	for _, tc := range []struct {
		name  string
		edit  func(*types.MinitiaConfig)
		layer string
		valid bool
	}{
		{"modern Initia", func(*types.MinitiaConfig) {}, "Initia", true},
		{"modern Celestia", func(c *types.MinitiaConfig) {
			c.OpBridge.BatchSubmissionTarget = "CELESTIA"
			c.SystemKeys.BatchSubmitter.DAAddress = daCelestiaAddress
		}, "Celestia", features.CelestiaDAEnabled},
		{"legacy DAAddress", func(c *types.MinitiaConfig) { c.OpBridge = nil }, "Initia", true},
		{"legacy L1Address", func(c *types.MinitiaConfig) {
			c.OpBridge = nil
			c.SystemKeys.BatchSubmitter.DAAddress = ""
			c.SystemKeys.BatchSubmitter.L1Address = daInitAddress
		}, "Initia", true},
		{"Celestia target overrides legacy Initia L1", func(c *types.MinitiaConfig) {
			c.OpBridge.BatchSubmissionTarget = "CELESTIA"
			c.SystemKeys.BatchSubmitter.DAAddress = ""
			c.SystemKeys.BatchSubmitter.L1Address = daInitAddress
		}, "Celestia", features.CelestiaDAEnabled},
		{"legacy Celestia DAAddress", func(c *types.MinitiaConfig) {
			c.OpBridge = nil
			c.SystemKeys.BatchSubmitter.DAAddress = daCelestiaAddress
		}, "Celestia", features.CelestiaDAEnabled},
		{"legacy DAAddress before L1", func(c *types.MinitiaConfig) {
			c.OpBridge = nil
			c.SystemKeys.BatchSubmitter.L1Address = daCelestiaAddress
		}, "Initia", true},
		{"Initia target ignores legacy Celestia L1", func(c *types.MinitiaConfig) { c.SystemKeys.BatchSubmitter.L1Address = daCelestiaAddress }, "Initia", true},
		{"missing system keys", func(c *types.MinitiaConfig) { c.SystemKeys = nil }, "", false},
		{"missing executor", func(c *types.MinitiaConfig) { c.SystemKeys.BridgeExecutor = nil }, "", false},
		{"missing output", func(c *types.MinitiaConfig) { c.SystemKeys.OutputSubmitter = nil }, "", false},
		{"missing batch", func(c *types.MinitiaConfig) { c.SystemKeys.BatchSubmitter = nil }, "", false},
		{"missing challenger", func(c *types.MinitiaConfig) { c.SystemKeys.Challenger = nil }, "", false},
		{"empty mnemonic", func(c *types.MinitiaConfig) { c.SystemKeys.BatchSubmitter.Mnemonic = "" }, "", false},
		{"invalid mnemonic", func(c *types.MinitiaConfig) { c.SystemKeys.BatchSubmitter.Mnemonic = "invalid words" }, "", false},
		{"unknown target", func(c *types.MinitiaConfig) { c.OpBridge.BatchSubmissionTarget = "unknown" }, "", false},
		{"damaged DAAddress", func(c *types.MinitiaConfig) { c.SystemKeys.BatchSubmitter.DAAddress = "init1damaged" }, "", false},
		{"conflicting target", func(c *types.MinitiaConfig) { c.SystemKeys.BatchSubmitter.DAAddress = daCelestiaAddress }, "", false},
		{"legacy no address", func(c *types.MinitiaConfig) { c.OpBridge = nil; c.SystemKeys.BatchSubmitter.DAAddress = "" }, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			daFlowFixtures(t)
			state := daFlowState()
			state.MinitiaConfig = daArtifact()
			tc.edit(state.MinitiaConfig)
			m, err := NewProcessingMinitiaConfig(daTestContext(t, state, t.TempDir()), NewSetupBotCheckbox)
			require.NoError(t, err)
			var next tea.Model
			var cmd tea.Cmd
			require.NotPanics(t, func() { next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) })
			if !tc.valid {
				require.Same(t, m, next)
				require.Nil(t, cmd)
				require.Empty(t, m.PanicText)
				require.Empty(t, weavecontext.GetPageStack[OPInitBotsState](m.Ctx))
				return
			}
			checkbox, ok := next.(*SetupBotCheckbox)
			require.True(t, ok, "%T", next)
			got := weavecontext.GetCurrentState[OPInitBotsState](checkbox.Ctx)
			require.True(t, got.AddMinitiaConfig)
			require.Equal(t, tc.layer, GetBotInfo(got.BotInfos, BatchSubmitter).DALayer)
			for _, info := range got.BotInfos {
				if info.BotName != OracleBridgeExecutor {
					wantMnemonic := daMnemonic
					if info.BotName == OutputSubmitter {
						wantMnemonic = daOutputMnemonic
					}
					if info.BotName == Challenger {
						wantMnemonic = daChallengerMnemonic
					}
					require.Equal(t, wantMnemonic, info.Mnemonic)
					require.False(t, info.IsNotExist)
				}
			}
		})
	}
}

// Catches partial replacement of earlier keys when a later batch fails validation.
func TestDAFlowImportIsAtomic(t *testing.T) {
	daFlowFixtures(t)
	state := daFlowState()
	state.MinitiaConfig = daArtifact()
	state.MinitiaConfig.SystemKeys.BatchSubmitter.DAAddress = "init1damaged"
	state.BotInfos[0].Mnemonic = "original executor"
	state.SetupOpinitResponses[BridgeExecutor] = "original response"
	before := state.Clone()
	m, err := NewProcessingMinitiaConfig(daTestContext(t, state, t.TempDir()), NewSetupBotCheckbox)
	require.NoError(t, err)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Same(t, m, next)
	require.Nil(t, cmd)
	require.Equal(t, before, weavecontext.GetCurrentState[OPInitBotsState](m.Ctx))
	require.Equal(t, before, state)
	require.Empty(t, weavecontext.GetPageStack[OPInitBotsState](m.Ctx))
}

// Catches nil node dereferences before prefill and a rejection that cannot be skipped.
func TestDAFlowPrefillMissingNodesCanSkip(t *testing.T) {
	for _, executor := range []bool{false, true} {
		for _, missing := range []string{"l1", "l2"} {
			t.Run(fmt.Sprintf("executor=%t/%s", executor, missing), func(t *testing.T) {
				daFlowFixtures(t)
				state := daFlowState()
				state.InitExecutorBot = executor
				state.InitChallengerBot = !executor
				state.MinitiaConfig = daArtifact()
				if missing == "l1" {
					state.MinitiaConfig.L1Config = nil
				} else {
					state.MinitiaConfig.L2Config = nil
				}
				before := state.Clone()
				m, err := NewPrefillMinitiaConfig(daTestContext(t, state, t.TempDir()))
				require.NoError(t, err)
				var next tea.Model
				var cmd tea.Cmd
				require.NotPanics(t, func() { next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) })
				require.Same(t, m, next)
				require.Nil(t, cmd)
				require.Contains(t, m.View(), missing+"_config")
				require.Empty(t, m.PanicText)
				require.Equal(t, before, weavecontext.GetCurrentState[OPInitBotsState](m.Ctx))
				require.Empty(t, weavecontext.GetPageStack[OPInitBotsState](m.Ctx))
				m.Update(tea.KeyMsg{Type: tea.KeyDown})
				next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				require.IsType(t, &L1PrefillSelector{}, next)
				require.NotContains(t, m.View(), "missing "+missing+"_config")
			})
		}
	}
}

// Catches discovery/skip incorrectly activating batch keys or consulting Celestia for Challenger.
func TestDAFlowChallengerAfterSkippingArtifact(t *testing.T) {
	daFlowFixtures(t)
	state := daFlowState()
	state.InitChallengerBot = true
	state.MinitiaConfig = daArtifact()
	state.MinitiaConfig.OpBridge.BatchSubmissionTarget = "CELESTIA"
	state.MinitiaConfig.SystemKeys.BatchSubmitter.DAAddress = daCelestiaAddress
	m, err := NewProcessingMinitiaConfig(daTestContext(t, state, t.TempDir()), func(ctx context.Context) (tea.Model, error) { return NewPrefillMinitiaConfig(ctx) })
	require.NoError(t, err)
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	prefill, ok := next.(*PrefillMinitiaConfig)
	require.True(t, ok)
	next, _ = prefill.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fields, ok := next.(*FieldInputModel)
	require.True(t, ok, "%T", next)
	got := weavecontext.GetCurrentState[OPInitBotsState](fields.Ctx)
	require.False(t, GetBotInfo(got.BotInfos, BatchSubmitter).IsNewKey())
	require.Empty(t, got.botConfig["da_node.bech32_prefix"])
	require.Equal(t, "artifact-l1", got.botConfig["l1_node.chain_id"])
}

// Catches all-key rejection quitting instead of allowing Skip into Challenger setup.
func TestDAFlowRejectedImportCanSkip(t *testing.T) {
	daFlowFixtures(t)
	state := daFlowState()
	state.InitChallengerBot = true
	state.MinitiaConfig = daArtifact()
	state.MinitiaConfig.SystemKeys.BatchSubmitter.DAAddress = "broken"
	m, err := NewProcessingMinitiaConfig(daTestContext(t, state, t.TempDir()), OPInitBotInitSelectChallenger)
	require.NoError(t, err)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Same(t, m, next)
	require.Nil(t, cmd)
	require.Contains(t, m.View(), "address")
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.IsType(t, &PrefillMinitiaConfig{}, next)
	require.NotContains(t, m.View(), "invalid")
}

func daCompleteFields(t *testing.T, fields *FieldInputModel) tea.Model {
	t.Helper()
	values := map[string]string{"listen_address": "localhost:3000", "l1_node.rpc_address": "https://final-l1.invalid:26657/custom", "l2_node.chain_id": "final-l2", "l2_node.rpc_address": "https://final-l2.invalid", "l2_node.gas_denom": "umin"}
	var next tea.Model = fields
	for i := range fields.subModels {
		fields.subModels[i].Text = values[fields.subModels[i].field.Name]
		next, _ = fields.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	return next
}

// Catches the hidden Celestia default and DA RPC copied before the user edits L1 RPC.
func TestDAFlowExecutorUsesFinalL1RPC(t *testing.T) {
	for _, artifact := range []bool{false, true} {
		for _, pending := range []bool{false, true} {
			t.Run(fmt.Sprintf("artifact=%t/pending=%t", artifact, pending), func(t *testing.T) {
				daFlowFixtures(t)
				state := daFlowState()
				state.InitExecutorBot = true
				state.daIsCelestia = true
				state.BotInfos[2].DALayer = "Initia"
				state.BotInfos[2].IsGenerateKey = pending
				ctx := daTestContext(t, state, t.TempDir())
				var next tea.Model
				if artifact {
					state.MinitiaConfig = daArtifact()
					ctx = weavecontext.SetCurrentState(ctx, state)
					m, err := NewPrefillMinitiaConfig(ctx)
					require.NoError(t, err)
					next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				} else {
					m, err := NewL1PrefillSelector(ctx)
					require.NoError(t, err)
					next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				}
				fields, ok := next.(*FieldInputModel)
				require.True(t, ok, "%T", next)
				next = daCompleteFields(t, fields)
				if !artifact {
					selector, ok := next.(*SetDALayer)
					require.True(t, ok, "%T", next)
					next, _ = selector.Update(tea.KeyMsg{Type: tea.KeyEnter})
				}
				loading, ok := next.(*FetchL1StartHeightLoading)
				require.True(t, ok, "%T", next)
				got := weavecontext.GetCurrentState[OPInitBotsState](loading.Ctx)
				require.Equal(t, "https://final-l1.invalid:26657/custom", got.botConfig["da_node.rpc_address"])
				if artifact {
					require.Equal(t, "artifact-l1", got.botConfig["da_node.chain_id"])
					require.Equal(t, "0.025uinit", got.botConfig["da_node.gas_price"])
				} else {
					require.Equal(t, "test-l1", got.botConfig["da_node.chain_id"])
					require.Equal(t, "0.015uinit", got.botConfig["da_node.gas_price"])
				}
				config := buildExecutorConfig(got, 7)
				require.Equal(t, "https://final-l1.invalid:26657/custom", config.DANode.RPCAddress)
				require.Equal(t, "init", got.botConfig["da_node.bech32_prefix"])
				require.False(t, got.daIsCelestia)
			})
		}
	}
}

// Catches eager constructor lookup, unavailable options, and mismatches accepted before loading.
func TestDAFlowSetDALayerRejectThenCorrect(t *testing.T) {
	for _, layer := range []string{"Initia", "Celestia"} {
		t.Run(layer, func(t *testing.T) {
			if layer == "Celestia" && !features.CelestiaDAEnabled {
				t.Skip("correcting to Celestia requires tagged build")
			}
			daFlowFixtures(t)
			state := daFlowState()
			state.InitExecutorBot = true
			state.BotInfos[2].DALayer = layer
			state.BotInfos[2].IsGenerateKey = true
			state.botConfig["l1_node.chain_id"] = "test-l1"
			state.botConfig["l1_node.rpc_address"] = "https://final-l1.invalid"
			state.botConfig["l1_node.gas_price"] = "0.015uinit"
			model, err := NewSetDALayer(daTestContext(t, state, t.TempDir()))
			require.NoError(t, err)
			m := model.(*SetDALayer)
			if features.CelestiaDAEnabled {
				require.Equal(t, []DALayerNetwork{Initia, Celestia}, m.Options)
			} else {
				require.Equal(t, []DALayerNetwork{Initia}, m.Options)
			}
			require.Len(t, *m.Tooltips, len(m.Options))
			require.True(t, m.CanGoPreviousPage())
			require.False(t, m.Selector.CannotBack)
			before := state.Clone()
			m.Options = []DALayerNetwork{Celestia, Initia}
			if layer == "Celestia" {
				m.Options = []DALayerNetwork{Initia, Celestia}
			}
			m.Cursor = 0
			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			require.Same(t, m, next)
			require.Nil(t, cmd)
			require.Empty(t, m.PanicText)
			require.Equal(t, before, weavecontext.GetCurrentState[OPInitBotsState](m.Ctx))
			require.Empty(t, weavecontext.GetPageStack[OPInitBotsState](m.Ctx))
			if features.CelestiaDAEnabled {
				require.Contains(t, m.View(), "does not match")
			} else {
				require.Contains(t, m.View(), "Celestia DA is disabled")
			}
			m.Update(tea.KeyMsg{Type: tea.KeyDown})
			if layer == "Celestia" {
				daCelestiaRegistryFixture()
			}
			next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			require.IsType(t, &FetchL1StartHeightLoading{}, next)
			require.NotContains(t, m.View(), "does not match")
			require.NotContains(t, m.View(), "Celestia DA is disabled")
		})
	}
}

// Catches previous-field editing becoming inaccessible once DA selection is reached.
func TestDAFlowSetDALayerBackToInput(t *testing.T) {
	daFlowFixtures(t)
	state := daFlowState()
	state.InitExecutorBot = true
	m, err := NewL1PrefillSelector(daTestContext(t, state, t.TempDir()))
	require.NoError(t, err)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fields, ok := next.(*FieldInputModel)
	require.True(t, ok)
	next = daCompleteFields(t, fields)
	selector, ok := next.(*SetDALayer)
	require.True(t, ok, "%T", next)
	next, _ = selector.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	require.Same(t, fields, next)
	require.NotPanics(t, func() { next.View() })
	// Walk back across three completed inputs and change L1 RPC, then resubmit.
	for i := 0; i < 3; i++ {
		next, _ = fields.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
		require.Same(t, fields, next)
		require.NotPanics(t, func() { next.View() })
	}
	require.Equal(t, "l1_node.rpc_address", fields.subModels[fields.currentIndex].field.Name)
	values := []string{"https://edited-l1.invalid", "edited-l2", "https://edited-l2.invalid", "uother"}
	for _, value := range values {
		fields.subModels[fields.currentIndex].Text = value
		next, _ = fields.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	selector, ok = next.(*SetDALayer)
	require.True(t, ok)
	next, _ = selector.Update(tea.KeyMsg{Type: tea.KeyEnter})
	loading, ok := next.(*FetchL1StartHeightLoading)
	require.True(t, ok)
	got := weavecontext.GetCurrentState[OPInitBotsState](loading.Ctx)
	require.Equal(t, "https://edited-l1.invalid", got.botConfig["da_node.rpc_address"])
	require.Equal(t, "uother", got.botConfig["l2_node.gas_denom"])
}

func daCelestiaRegistryFixture() {
	registry.LoadedChainRegistry[registry.CelestiaTestnet] = &registry.ChainRegistry{ChainId: "celestia-test", Bech32Prefix: "celestia", ActiveRpcs: []string{"https://celestia-test.invalid"}}
	registry.LoadedChainRegistry[registry.CelestiaMainnet] = &registry.ChainRegistry{ChainId: "celestia-main", Bech32Prefix: "celestia", ActiveRpcs: []string{"https://celestia-main.invalid"}}
}

// Catches prefill mutating shared state/defaults before a DA policy or pending-key mismatch.
func TestDAFlowPrefillRejectThenCorrect(t *testing.T) {
	for _, target := range []string{"INITIA", "CELESTIA", "unknown"} {
		t.Run(target, func(t *testing.T) {
			daFlowFixtures(t)
			state := daFlowState()
			state.InitExecutorBot = true
			state.MinitiaConfig = daArtifact()
			state.MinitiaConfig.OpBridge.BatchSubmissionTarget = target
			state.MinitiaConfig.SystemKeys = nil // Prefill needs the explicit target, not key-import accounts.
			state.BotInfos[2].IsGenerateKey = true
			state.BotInfos[2].DALayer = "Initia"
			if target == "INITIA" {
				state.BotInfos[2].DALayer = "Celestia"
			}
			state.botConfig["da_node.rpc_address"] = "original-da-rpc"
			before := state.Clone()
			executorBefore := make([]Field, len(defaultExecutorFields))
			for idx, f := range defaultExecutorFields {
				executorBefore[idx] = *f
			}
			m, err := NewPrefillMinitiaConfig(daTestContext(t, state, t.TempDir()))
			require.NoError(t, err)
			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			require.Same(t, m, next)
			require.Nil(t, cmd)
			require.Empty(t, m.PanicText)
			require.Equal(t, before, state)
			require.Equal(t, before, weavecontext.GetCurrentState[OPInitBotsState](m.Ctx))
			require.Empty(t, weavecontext.GetPageStack[OPInitBotsState](m.Ctx))
			for idx, f := range defaultExecutorFields {
				require.Equal(t, executorBefore[idx].PrefillValue, f.PrefillValue)
				require.Equal(t, executorBefore[idx].Placeholder, f.Placeholder)
			}
			// Correcting the pending key/artifact lets the same model continue and clears its error.
			corrected := state.Clone()
			corrected.MinitiaConfig.OpBridge.BatchSubmissionTarget = "INITIA"
			corrected.BotInfos[2].DALayer = "Initia"
			m.Ctx = weavecontext.SetCurrentState(m.Ctx, corrected)
			next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			require.IsType(t, &FieldInputModel{}, next)
			require.Nil(t, m.validationErr)
		})
	}
}

// Catches a rejected existing Celestia/unknown config being silently used or trapping replacement.
func TestDAFlowRejectedCurrentConfigCanReplaceWithInitia(t *testing.T) {
	daFlowFixtures(t)
	state := daFlowState()
	state.InitExecutorBot = true
	state.botConfig["da_node.bech32_prefix"] = "unknown"
	m, err := NewUseCurrentConfigSelector(daTestContext(t, state, t.TempDir()), "executor")
	require.NoError(t, err)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Same(t, m, next)
	require.Nil(t, cmd)
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, m.validationErr)
	l1, ok := next.(*L1PrefillSelector)
	require.True(t, ok)
	next, _ = l1.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fields := next.(*FieldInputModel)
	next = daCompleteFields(t, fields)
	selector := next.(*SetDALayer)
	next, _ = selector.Update(tea.KeyMsg{Type: tea.KeyEnter})
	loading := next.(*FetchL1StartHeightLoading)
	got := weavecontext.GetCurrentState[OPInitBotsState](loading.Ctx)
	require.True(t, got.ReplaceBotConfig)
	require.Equal(t, "init", got.botConfig["da_node.bech32_prefix"])
}

// Catches accidentally preserving the disabled behavior in a Celestia-enabled build.
func TestDAFlowTaggedCelestiaSelection(t *testing.T) {
	if !features.CelestiaDAEnabled {
		t.Skip("positive Celestia path requires tagged build")
	}
	for _, artifact := range []bool{false, true} {
		for _, pending := range []bool{false, true} {
			t.Run(fmt.Sprintf("artifact=%t/pending=%t", artifact, pending), func(t *testing.T) {
				daFlowFixtures(t)
				state := daFlowState()
				state.InitExecutorBot = true
				state.botConfig["l1_node.chain_id"] = "test-l1"
				state.BotInfos[2].DALayer = "Celestia"
				state.BotInfos[2].IsGenerateKey = pending
				ctx := daTestContext(t, state, t.TempDir())
				var next tea.Model
				if artifact {
					state.MinitiaConfig = daArtifact()
					state.MinitiaConfig.OpBridge.BatchSubmissionTarget = "CELESTIA"
					state.MinitiaConfig.SystemKeys = nil
					daCelestiaRegistryFixture()
					m, err := NewPrefillMinitiaConfig(weavecontext.SetCurrentState(ctx, state))
					require.NoError(t, err)
					next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
					require.IsType(t, &FieldInputModel{}, next)
					next = daCompleteFields(t, next.(*FieldInputModel))
				} else {
					model, err := NewSetDALayer(ctx)
					require.NoError(t, err) // Still no Celestia cache or HTTP access.
					selector := model.(*SetDALayer)
					if pending {
						require.Equal(t, 1, selector.Cursor)
					} else {
						require.Zero(t, selector.Cursor)
						selector.Update(tea.KeyMsg{Type: tea.KeyDown})
					}
					daCelestiaRegistryFixture()
					next, _ = selector.Update(tea.KeyMsg{Type: tea.KeyEnter})
				}
				loading, ok := next.(*FetchL1StartHeightLoading)
				require.True(t, ok, "%T", next)
				got := weavecontext.GetCurrentState[OPInitBotsState](loading.Ctx)
				require.True(t, got.daIsCelestia)
				require.Equal(t, "celestia", got.botConfig["da_node.bech32_prefix"])
				if artifact {
					require.Equal(t, "celestia-main", got.botConfig["da_node.chain_id"])
					require.Equal(t, "https://celestia-main.invalid", got.botConfig["da_node.rpc_address"])
				} else {
					require.Equal(t, "celestia-test", got.botConfig["da_node.chain_id"])
					require.Equal(t, "https://celestia-test.invalid", got.botConfig["da_node.rpc_address"])
				}
			})
		}
	}
}

// Catches DA values or executor defaults omitted by the actual config writer's builder.
func TestDAFlowExecutorConfigJSON(t *testing.T) {
	for _, tc := range []struct{ prefix, chain, rpc, gas string }{
		{"init", "l1-final", "https://l1-final.invalid", "0.015uinit"},
		{"celestia", "da-final", "https://da-final.invalid", "0.004utia"},
	} {
		prefix := tc.prefix
		t.Run(prefix, func(t *testing.T) {
			if prefix == "celestia" && !features.CelestiaDAEnabled {
				t.Skip("positive Celestia config requires tagged build")
			}
			state := daTestState()
			state.L1StartHeight = 123
			state.botConfig = map[string]string{
				"listen_address": "localhost:3000", "l1_node.chain_id": "l1-final", "l1_node.rpc_address": "https://l1-final.invalid", "l1_node.gas_price": "0.015uinit",
				"l2_node.chain_id": "l2-final", "l2_node.rpc_address": "https://l2-final.invalid", "l2_node.gas_denom": "umin",
				"da_node.chain_id": tc.chain, "da_node.rpc_address": tc.rpc, "da_node.bech32_prefix": prefix, "da_node.gas_price": tc.gas,
			}
			raw, err := json.Marshal(buildExecutorConfig(state, 7))
			require.NoError(t, err)
			require.JSONEq(t, fmt.Sprintf(`{
   "version":7, "server":{"address":"localhost:3000","allow_origins":"*","allow_headers":"Origin, Content-Type, Accept","allow_methods":"GET"},
   "l1_node":{"chain_id":"l1-final","bech32_prefix":"init","rpc_address":"https://l1-final.invalid","gas_price":"0.015uinit","gas_adjustment":1.5,"tx_timeout":60},
   "l2_node":{"chain_id":"l2-final","bech32_prefix":"init","rpc_address":"https://l2-final.invalid","gas_price":"0umin","gas_adjustment":1.5,"tx_timeout":60},
   "da_node":{"chain_id":%q,"bech32_prefix":%q,"rpc_address":%q,"gas_price":%q,"gas_adjustment":1.5,"tx_timeout":60},
   "bridge_executor":"weave_bridge_executor", "oracle_bridge_executor":"weave_oracle_bridge_executor",
   "disable_output_submitter":false, "disable_batch_submitter":false, "max_chunks":5000, "max_chunk_size":300000, "max_submission_time":3600,
   "disable_auto_set_l1_height":true, "l1_start_height":123, "l2_start_height":1, "batch_start_height":1, "disable_delete_future_withdrawal":false
  }`, tc.chain, prefix, tc.rpc, tc.gas), string(raw))
		})
	}
}
