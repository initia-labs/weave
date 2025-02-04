package opinit_bots

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/initia-labs/weave/analytics"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/registry"
	"github.com/initia-labs/weave/types"
)

func TestMain(m *testing.M) {
	analytics.Client = &analytics.NoOpClient{}
	exitCode := m.Run()
	os.Exit(exitCode)
}
func TestUseCurrentConfigSelector_Update_UseCurrentFile(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewOPInitBotsState())
	model, _ := NewUseCurrentConfigSelector(ctx, "test-bot")

	// Simulate selecting "use current file"
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection with Enter

	// Expect transition to FetchL1StartHeightLoading
	if m, ok := nextModel.(*FetchL1StartHeightLoading); !ok {
		t.Errorf("Expected model to be of type *FetchL1StartHeightLoading, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
		assert.False(t, state.ReplaceBotConfig) // Verify ReplaceBotConfig is set to false
	}
}

func TestUseCurrentConfigSelector_Update_ReplaceWithMinitiaConfig(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewOPInitBotsState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
	state.MinitiaConfig = &types.MinitiaConfig{} // Assume MinitiaConfig is defined
	ctx = weavecontext.SetCurrentState(ctx, state)
	model, _ := NewUseCurrentConfigSelector(ctx, "test-bot")

	// Simulate selecting "replace"
	model.Update(tea.KeyMsg{Type: tea.KeyDown})                  // Navigate to "replace"
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection with Enter

	// Expect transition to PrefillMinitiaConfig
	if m, ok := nextModel.(*PrefillMinitiaConfig); !ok {
		t.Errorf("Expected model to be of type *PrefillMinitiaConfig, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
		assert.True(t, state.ReplaceBotConfig) // Verify ReplaceBotConfig is set to true
	}
}

func TestUseCurrentConfigSelector_Update_ReplaceWithL1PrefillSelector(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewOPInitBotsState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
	state.InitExecutorBot = true // Example setup, assume InitExecutorBot is true
	ctx = weavecontext.SetCurrentState(ctx, state)
	model, _ := NewUseCurrentConfigSelector(ctx, "test-bot")

	// Simulate selecting "replace"
	model.Update(tea.KeyMsg{Type: tea.KeyDown})                  // Navigate to "replace"
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection with Enter

	// Expect transition to L1PrefillSelector
	if m, ok := nextModel.(*L1PrefillSelector); !ok {
		t.Errorf("Expected model to be of type *L1PrefillSelector, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
		assert.True(t, state.ReplaceBotConfig) // Verify ReplaceBotConfig is set to true
	}
}

func TestPrefillMinitiaConfig_Update_PrefillYes(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewOPInitBotsState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
	state.MinitiaConfig = &types.MinitiaConfig{
		L1Config: &types.L1Config{
			ChainID:   "l1-chain-id",
			RpcUrl:    "http://l1-rpc-url",
			GasPrices: "0.01",
		},
		L2Config: &types.L2Config{
			ChainID: "l2-chain-id",
			Denom:   "denom",
		},
		OpBridge: &types.OpBridge{
			BatchSubmissionTarget: "CELESTIA",
		},
	}
	state.InitExecutorBot = true
	ctx = weavecontext.SetCurrentState(ctx, state)

	model, _ := NewPrefillMinitiaConfig(ctx)

	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection with Enter

	// Expect transition to FieldInputModel for Executor with defaultExecutorFields
	if m, ok := nextModel.(*FieldInputModel); !ok {
		t.Errorf("Expected model to be of type *FieldInputModel, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
		gasField, _ := getField(defaultExecutorFields, "l2_node.gas_denom")
		assert.Equal(t, "l1-chain-id", state.botConfig["l1_node.chain_id"])
		assert.Equal(t, "http://l1-rpc-url", state.botConfig["l1_node.rpc_address"])
		assert.Equal(t, "0.01", state.botConfig["l1_node.gas_price"])
		assert.Equal(t, "denom", gasField.PrefillValue)
		assert.True(t, state.daIsCelestia)
	}
}

func TestPrefillMinitiaConfig_Update_PrefillNo(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewOPInitBotsState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
	state.MinitiaConfig = &types.MinitiaConfig{}
	ctx = weavecontext.SetCurrentState(ctx, state)

	model, _ := NewPrefillMinitiaConfig(ctx)

	// Simulate selecting "No" for a prefilled option
	model.Update(tea.KeyMsg{Type: tea.KeyDown})                  // Navigate to "PrefillMinitiaConfigNo"
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection with Enter

	// Expect transition to L1PrefillSelector
	if m, ok := nextModel.(*L1PrefillSelector); !ok {
		t.Errorf("Expected model to be of type *L1PrefillSelector, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
		assert.Empty(t, state.botConfig) // Ensure botConfig is not prefilled
	}
}

func TestPrefillMinitiaConfig_Update_PrefillYes_NonCelestia(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewOPInitBotsState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
	state.MinitiaConfig = &types.MinitiaConfig{
		L1Config: &types.L1Config{
			ChainID:   "l1-chain-id",
			RpcUrl:    "http://l1-rpc-url",
			GasPrices: "0.01",
		},
		L2Config: &types.L2Config{
			ChainID: "l2-chain-id",
			Denom:   "denom",
		},
		OpBridge: &types.OpBridge{
			BatchSubmissionTarget: "INITIA", // Set to non-Celestia target
		},
	}
	state.InitExecutorBot = true
	ctx = weavecontext.SetCurrentState(ctx, state)

	model, _ := NewPrefillMinitiaConfig(ctx)

	// Simulate selecting "Yes" for a prefilled option
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection with Enter

	// Expect transition to FieldInputModel for Executor with defaultExecutorFields
	if m, ok := nextModel.(*FieldInputModel); !ok {
		t.Errorf("Expected model to be of type *FieldInputModel, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)

		// Verify that botConfig fields are set according to the non-Celestia case
		assert.Equal(t, "l1-chain-id", state.botConfig["l1_node.chain_id"])
		assert.Equal(t, "http://l1-rpc-url", state.botConfig["l1_node.rpc_address"])
		assert.Equal(t, "0.01", state.botConfig["l1_node.gas_price"])

		// Verify non-Celestia values in botConfig for da_node fields
		assert.Equal(t, state.botConfig["l1_node.chain_id"], state.botConfig["da_node.chain_id"])
		assert.Equal(t, state.botConfig["l1_node.rpc_address"], state.botConfig["da_node.rpc_address"])
		assert.Equal(t, "init", state.botConfig["da_node.bech32_prefix"])
		assert.Equal(t, state.botConfig["l1_node.gas_price"], state.botConfig["da_node.gas_price"])

		// Check that daIsCelestia is false
		assert.False(t, state.daIsCelestia)
	}
}

func TestSetDALayer_Update_SelectInitia(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewOPInitBotsState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
	state.botConfig = map[string]string{
		"l1_node.chain_id":    "initiation-2",
		"l1_node.rpc_address": "http://l1-rpc-url",
		"l1_node.gas_price":   "0.01",
	}
	ctx = weavecontext.SetCurrentState(ctx, state)

	model, _ := NewSetDALayer(ctx) // Assuming NewSetDALayer initializes SetDALayer

	// Simulate selecting "Initia"
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection with Enter

	// Expect transition to FetchL1StartHeightLoading
	if m, ok := nextModel.(*FetchL1StartHeightLoading); !ok {
		t.Errorf("Expected model to be of type *FetchL1StartHeightLoading, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
		assert.Equal(t, state.botConfig["l1_node.chain_id"], state.botConfig["da_node.chain_id"])
		assert.Equal(t, "init", state.botConfig["da_node.bech32_prefix"])
		assert.Equal(t, state.botConfig["l1_node.gas_price"], state.botConfig["da_node.gas_price"])
		assert.False(t, state.daIsCelestia) // Ensure daIsCelestia is false for Initia
	}
}

func TestSetDALayer_Update_SelectCelestia(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewOPInitBotsState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
	state.botConfig = map[string]string{}
	state.botConfig["l1_node.chain_id"] = "initiation-2"
	ctx = weavecontext.SetCurrentState(ctx, state)

	chainRegistry, _ := registry.GetChainRegistry(registry.CelestiaTestnet)
	model, _ := NewSetDALayer(ctx)

	// Simulate selecting "Celestia"
	model.Update(tea.KeyMsg{Type: tea.KeyDown})                  // Navigate to "Celestia" option
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection with Enter

	// Expect transition to FetchL1StartHeightLoading
	if m, ok := nextModel.(*FetchL1StartHeightLoading); !ok {
		t.Errorf("Expected model to be of type *FetchL1StartHeightLoading, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
		assert.Equal(t, chainRegistry.ChainId, state.botConfig["da_node.chain_id"])
		assert.Equal(t, chainRegistry.Bech32Prefix, state.botConfig["da_node.bech32_prefix"])
		assert.Equal(t, DefaultCelestiaGasPrices, state.botConfig["da_node.gas_price"])
		assert.True(t, state.daIsCelestia) // Ensure daIsCelestia is true for Celestia
	}
}
