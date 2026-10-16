package opinit_bots

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/internal/features"
	"github.com/initia-labs/weave/types"
)

func TestProcessingMinitiaConfig_Update_AddKeys(t *testing.T) {
	ctx := weavecontext.NewAppContext(daFlowState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)

	// Set up mock BotInfos and MinitiaConfig in the state
	state.BotInfos = []BotInfo{
		{BotName: BridgeExecutor},
		{BotName: OutputSubmitter},
		{BotName: BatchSubmitter},
		{BotName: Challenger},
		{BotName: OracleBridgeExecutor},
	}
	state.MinitiaConfig = &types.MinitiaConfig{
		SystemKeys: &types.SystemKeys{
			BridgeExecutor:  &types.SystemAccount{Mnemonic: daMnemonic},
			OutputSubmitter: &types.SystemAccount{Mnemonic: daMnemonic},
			BatchSubmitter:  &types.SystemAccount{Mnemonic: daMnemonic, DAAddress: daInitAddress},
			Challenger:      &types.SystemAccount{Mnemonic: daMnemonic},
		},
	}

	// Update context with modified state
	ctx = weavecontext.SetCurrentState(ctx, state)

	// Initialize the ProcessingMinitiaConfig model
	model, _ := NewProcessingMinitiaConfig(ctx, func(ctx context.Context) (tea.Model, error) {
		return NewSetupBotCheckbox(ctx)
	})

	// Simulate selecting "Yes" to add keys
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection

	// Check that the model transitions to SetupOPInitBots
	if setupModel, ok := nextModel.(*SetupBotCheckbox); ok {
		state := weavecontext.GetCurrentState[OPInitBotsState](setupModel.Ctx)

		// Validate that BotInfos have been updated
		assert.Equal(t, daMnemonic, state.BotInfos[0].Mnemonic)
		assert.Equal(t, daMnemonic, state.BotInfos[1].Mnemonic)
		assert.Equal(t, daMnemonic, state.BotInfos[2].Mnemonic)
		assert.Equal(t, daMnemonic, state.BotInfos[3].Mnemonic)
		assert.Equal(t, string(InitiaLayerOption), state.BotInfos[2].DALayer) // BatchSubmitter DA Layer check
	} else {
		t.Errorf("Expected model to be of type *SetupBotCheckbox, but got %T", nextModel)
	}
}

func TestProcessingMinitiaConfig_Update_SkipKeys(t *testing.T) {
	ctx := weavecontext.NewAppContext(daFlowState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)

	// Initialize state with empty BotInfos and MinitiaConfig
	state.BotInfos = []BotInfo{
		{BotName: BridgeExecutor},
		{BotName: OutputSubmitter},
		{BotName: BatchSubmitter},
		{BotName: Challenger},
		{BotName: OracleBridgeExecutor},
	}
	ctx = weavecontext.SetCurrentState(ctx, state)

	// Initialize the ProcessingMinitiaConfig model
	model, _ := NewProcessingMinitiaConfig(ctx, func(ctx context.Context) (tea.Model, error) {
		return NewSetupBotCheckbox(ctx)
	})

	// Simulate selecting "No" to skip adding keys
	model.Update(tea.KeyMsg{Type: tea.KeyDown})                  // Navigate to "No" option
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection

	// Check that the model transitions to NewSetupBotCheckbox
	if checkboxModel, ok := nextModel.(*SetupBotCheckbox); ok {
		state := weavecontext.GetCurrentState[OPInitBotsState](checkboxModel.Ctx)

		// Ensure that BotInfos remain unchanged (no mnemonic added)
		assert.Empty(t, state.BotInfos[0].Mnemonic)
		assert.Empty(t, state.BotInfos[1].Mnemonic)
		assert.Empty(t, state.BotInfos[2].Mnemonic)
		assert.Empty(t, state.BotInfos[3].Mnemonic)
		assert.Empty(t, state.BotInfos[4].Mnemonic)
	} else {
		t.Errorf("Expected model to be of type *SetupBotCheckbox, but got %T", nextModel)
	}
}

func TestSetupBotCheckbox_SelectBots(t *testing.T) {
	ctx := weavecontext.NewAppContext(daFlowState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)

	// Setup initial BotInfos in state
	state.BotInfos = []BotInfo{
		{BotName: BridgeExecutor},
		{BotName: OutputSubmitter},
		{BotName: BatchSubmitter},
		{BotName: Challenger},
		{BotName: OracleBridgeExecutor},
	}
	ctx = weavecontext.SetCurrentState(ctx, state)

	// Initialize SetupBotCheckbox model
	model, _ := NewSetupBotCheckbox(ctx)

	// Simulate selecting two bots (e.g., BridgeExecutor and BatchSubmitter)
	model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model.Update(tea.KeyMsg{Type: tea.KeySpace})

	// Press Enter to confirm selection
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Check that the model transitions to NextUpdateOpinitBotKey
	if opInitModel, ok := nextModel.(*RecoverKeySelector); ok {
		state := weavecontext.GetCurrentState[OPInitBotsState](opInitModel.Ctx)

		// Verify that selected bots have IsSetup set to true
		assert.True(t, state.BotInfos[0].IsSetup)  // BridgeExecutor
		assert.False(t, state.BotInfos[1].IsSetup) // OutputSubmitter
		assert.True(t, state.BotInfos[2].IsSetup)  // BatchSubmitter
		assert.False(t, state.BotInfos[3].IsSetup) // Challenger
		assert.False(t, state.BotInfos[4].IsSetup) // Challenger
	} else {
		t.Errorf("Expected model to transition to *NextUpdateOpinitBotKey, but got %T", nextModel)
	}
}

func TestSetupBotCheckbox_NoSelection(t *testing.T) {
	ctx := weavecontext.NewAppContext(daFlowState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)

	// Setup initial BotInfos in state
	state.BotInfos = []BotInfo{
		{BotName: BridgeExecutor},
		{BotName: OutputSubmitter},
		{BotName: BatchSubmitter},
		{BotName: Challenger},
		{BotName: OracleBridgeExecutor},
	}
	ctx = weavecontext.SetCurrentState(ctx, state)

	// Initialize SetupBotCheckbox model with no selection
	model, _ := NewSetupBotCheckbox(ctx)

	// Press Enter without making any choice
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify that the model returns to SetupOPInitBots when no bots are selected
	if opInitModel, ok := nextModel.(*SetupOPInitBots); ok {
		state := weavecontext.GetCurrentState[OPInitBotsState](opInitModel.Ctx)

		// Ensure that none of the bots have IsSetup set to true
		for _, botInfo := range state.BotInfos {
			assert.False(t, botInfo.IsSetup)
		}
	} else {
		t.Errorf("Expected model to transition to *SetupOPInitBots, but got %T", nextModel)
	}
}

func TestRecoverKeySelector_GenerateNewKey(t *testing.T) {
	ctx := weavecontext.NewAppContext(daFlowState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)

	// Set up BotInfos in state
	state.BotInfos = []BotInfo{
		{BotName: BridgeExecutor},
		{BotName: BatchSubmitter},
	}
	ctx = weavecontext.SetCurrentState(ctx, state)

	// Initialize RecoverKeySelector for the first bot (BridgeExecutor)
	model := NewRecoverKeySelector(ctx, 0)

	// Simulate selecting "Generate new system key"
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection

	// Verify that the state is updated and the transition happens
	if opInitModel, ok := nextModel.(*SetupOPInitBots); ok {
		state := weavecontext.GetCurrentState[OPInitBotsState](opInitModel.Ctx)

		assert.True(t, state.BotInfos[0].IsGenerateKey)
		assert.Equal(t, "", state.BotInfos[0].Mnemonic)
		assert.False(t, state.BotInfos[0].IsSetup)
	} else {
		t.Errorf("Expected model to transition to *SetupOPInitBots, but got %T", nextModel)
	}
}

func TestRecoverKeySelector_GenerateNewKey_BatchSubmitter(t *testing.T) {
	ctx := weavecontext.NewAppContext(daFlowState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)

	// Set up BotInfos in state
	state.BotInfos = []BotInfo{
		{BotName: BatchSubmitter},
	}
	ctx = weavecontext.SetCurrentState(ctx, state)

	// Initialize RecoverKeySelector for BatchSubmitter
	model := NewRecoverKeySelector(ctx, 0)

	// Simulate selecting "Generate new system key"
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection

	// Verify that the state is updated and transitions to NewDALayerSelector
	if dalayerModel, ok := nextModel.(*DALayerSelector); ok {
		state := weavecontext.GetCurrentState[OPInitBotsState](dalayerModel.Ctx)

		assert.True(t, state.BotInfos[0].IsGenerateKey)
		assert.Equal(t, "", state.BotInfos[0].Mnemonic)
		assert.False(t, state.BotInfos[0].IsSetup)
	} else {
		t.Errorf("Expected model to transition to *DALayerSelector, but got %T", nextModel)
	}
}

func TestRecoverKeySelector_ImportExistingKey(t *testing.T) {
	ctx := weavecontext.NewAppContext(daFlowState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)

	// Set up BotInfos in state
	state.BotInfos = []BotInfo{
		{BotName: Challenger},
	}
	ctx = weavecontext.SetCurrentState(ctx, state)

	// Initialize RecoverKeySelector for Challenger
	model := NewRecoverKeySelector(ctx, 0)

	// Simulate selecting "Import existing key"
	model.Update(tea.KeyMsg{Type: tea.KeyDown})                  // Navigate to "Import existing key"
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection

	// Verify that the state is updated and transitions to NewRecoverFromMnemonic
	if recoverModel, ok := nextModel.(*RecoverFromMnemonic); ok {
		state := weavecontext.GetCurrentState[OPInitBotsState](recoverModel.Ctx)

		assert.False(t, state.BotInfos[0].IsGenerateKey)
		assert.Empty(t, state.BotInfos[0].Mnemonic) // Should not be set yet
	} else {
		t.Errorf("Expected model to transition to *RecoverFromMnemonic, but got %T", nextModel)
	}
}

func TestRecoverFromMnemonic_ValidInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(daFlowState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)

	// Set up BotInfos in state
	state.BotInfos = []BotInfo{
		{BotName: BridgeExecutor},
		{BotName: BatchSubmitter},
	}
	ctx = weavecontext.SetCurrentState(ctx, state)

	// Initialize RecoverFromMnemonic for the first bot (BridgeExecutor)
	model := NewRecoverFromMnemonic(ctx, 0)

	// Simulate entering a valid mnemonic
	validMnemonic := "use cost town major cram over ordinary great into armed razor train caught exhaust position mass juice quit dizzy balance mango sphere anxiety domain"
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validMnemonic)})
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Submit input

	// Verify that the state is updated and transitions to SetupOPInitBots
	if opInitModel, ok := nextModel.(*SetupOPInitBots); ok {
		state := weavecontext.GetCurrentState[OPInitBotsState](opInitModel.Ctx)

		assert.Equal(t, strings.Trim(validMnemonic, "\n"), state.BotInfos[0].Mnemonic)
		assert.False(t, state.BotInfos[0].IsSetup)
	} else {
		t.Errorf("Expected model to transition to *SetupOPInitBots, but got %T", nextModel)
	}
}

func TestRecoverFromMnemonic_ValidInput_BatchSubmitter(t *testing.T) {
	ctx := weavecontext.NewAppContext(daFlowState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)

	// Set up BotInfos in state
	state.BotInfos = []BotInfo{
		{BotName: BatchSubmitter},
	}
	ctx = weavecontext.SetCurrentState(ctx, state)

	// Initialize RecoverFromMnemonic for BatchSubmitter
	model := NewRecoverFromMnemonic(ctx, 0)

	// Simulate entering a valid mnemonic
	validMnemonic := "use cost town major cram over ordinary great into armed razor train caught exhaust position mass juice quit dizzy balance mango sphere anxiety domain"
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validMnemonic)})
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Submit input

	// Verify that the state is updated and transitions to NewDALayerSelector
	if dalayerModel, ok := nextModel.(*DALayerSelector); ok {
		state := weavecontext.GetCurrentState[OPInitBotsState](dalayerModel.Ctx)

		assert.Equal(t, strings.Trim(validMnemonic, "\n"), state.BotInfos[0].Mnemonic)
		assert.False(t, state.BotInfos[0].IsSetup)
	} else {
		t.Errorf("Expected model to transition to *DALayerSelector, but got %T", nextModel)
	}
}

func TestRecoverFromMnemonic_InvalidInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(daFlowState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)

	// Set up BotInfos in state
	state.BotInfos = []BotInfo{
		{BotName: Challenger},
	}
	ctx = weavecontext.SetCurrentState(ctx, state)

	// Initialize RecoverFromMnemonic for Challenger
	model := NewRecoverFromMnemonic(ctx, 0)

	// Simulate entering an invalid mnemonic
	invalidMnemonic := "invalid mnemonic"
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(invalidMnemonic)})
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Submit input

	// Verify that no transition occurs
	if recoverModel, ok := nextModel.(*RecoverFromMnemonic); ok {
		state := weavecontext.GetCurrentState[OPInitBotsState](recoverModel.Ctx)

		assert.Empty(t, state.BotInfos[0].Mnemonic) // Mnemonic should not be set
	} else {
		t.Errorf("Expected model to remain *RecoverFromMnemonic, but got %T", nextModel)
	}
}

func TestDALayerSelector_SelectInitiaLayer(t *testing.T) {
	ctx := weavecontext.NewAppContext(daFlowState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)

	// Set up BotInfos in state
	state.BotInfos = []BotInfo{
		{BotName: BatchSubmitter},
	}
	ctx = weavecontext.SetCurrentState(ctx, state)

	// Initialize DALayerSelector for the first bot (BatchSubmitter)
	model := NewDALayerSelector(ctx, 0)

	// Simulate selecting InitiaLayerOption
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection

	// Verify the state and transition
	if opInitModel, ok := nextModel.(*SetupOPInitBots); ok {
		state := weavecontext.GetCurrentState[OPInitBotsState](opInitModel.Ctx)

		assert.Equal(t, string(InitiaLayerOption), state.BotInfos[0].DALayer) // Check that DALayer is set to Initia
	} else {
		t.Errorf("Expected model to transition to *SetupOPInitBots, but got %T", nextModel)
	}
}

func TestDALayerSelector_SelectCelestiaLayer(t *testing.T) {
	ctx := weavecontext.NewAppContext(daFlowState())
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)

	// Set up BotInfos in state
	state.BotInfos = []BotInfo{
		{BotName: BatchSubmitter},
	}
	ctx = weavecontext.SetCurrentState(ctx, state)

	// Initialize DALayerSelector for the first bot (BatchSubmitter)
	model := NewDALayerSelector(ctx, 0)

	// Simulate selecting CelestiaLayerOption
	model.Update(tea.KeyMsg{Type: tea.KeyDown})                  // Navigate to CelestiaLayerOption
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection

	// Verify the state and transition
	if opInitModel, ok := nextModel.(*SetupOPInitBots); ok {
		state := weavecontext.GetCurrentState[OPInitBotsState](opInitModel.Ctx)

		if features.CelestiaDAEnabled {
			assert.Equal(t, "Celestia", state.BotInfos[0].DALayer)
		} else {
			assert.Equal(t, "Initia", state.BotInfos[0].DALayer)
		} // Check that DALayer is set to Celestia
	} else {
		t.Errorf("Expected model to transition to *SetupOPInitBots, but got %T", nextModel)
	}
}
