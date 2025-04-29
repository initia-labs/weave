package initia

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/common"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/cosmosutils"
	"github.com/initia-labs/weave/registry"
	"github.com/initia-labs/weave/ui"
)

func TestMain(m *testing.M) {
	analytics.Client = &analytics.NoOpClient{}
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestGetNextModelByExistingApp(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())

	// Test case when existingApp is true
	model, _ := GetNextModelByExistingApp(ctx, true)
	if _, ok := model.(*ExistingAppReplaceSelect); !ok {
		t.Errorf("Expected model to be of type *ExistingAppReplaceSelect when existingApp is true, but got %T", model)
	}

	// Test case when existingApp is false
	model, _ = GetNextModelByExistingApp(ctx, false)
	if _, ok := model.(*RunL1NodeMonikerInput); !ok {
		t.Errorf("Expected model to be of type *RunL1NodeMonikerInput when existingApp is false, but got %T", model)
	}
}

func TestRunL1NodeNetworkSelectInitialization(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	model, _ := NewRunL1NodeNetworkSelect(ctx)

	assert.Equal(t, "Which network will your node participate in?", model.GetQuestion())
	assert.Contains(t, model.Selector.Options, Mainnet)
	assert.Contains(t, model.Selector.Options, Testnet)
	// assert.Contains(t, model.Selector.Options, Local)
}

// func TestRunL1NodeNetworkSelectLocalSelection(t *testing.T) {
// 	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
// 	model := NewRunL1NodeNetworkSelect(ctx)

// 	// Simulate pressing down to move to "Local" option and select it
// 	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
// 	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

// 	// Verify that the next model is of the expected type for Local selection
// 	if m, ok := nextModel.(*RunL1NodeVersionSelect); !ok {
// 		t.Errorf("Expected next model to be of type *RunL1NodeVersionSelect, but got %T", nextModel)
// 	} else {
// 		state := weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)

// 		// Verify the state after selecting "Local"
// 		assert.Equal(t, string(Local), state.network)
// 		assert.Nil(t, state.chainRegistry)
// 		assert.Empty(t, state.chainId)
// 		assert.Empty(t, state.genesisEndpoint)
// 	}
// }

func TestRunL1NodeVersionSelectUpdate(t *testing.T) {
	// Set up a mock context and initial state
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())

	// Define mock versions
	mockVersions := cosmosutils.BinaryVersionWithDownloadURL{
		"v1.0.0": "http://example.com/download/v1.0.0",
		"v2.0.0": "http://example.com/download/v2.0.0",
	}

	// Initialize the model with mock versions and a question
	model := &RunL1NodeVersionSelect{
		Selector: ui.Selector[string]{Options: []string{"v1.0.0", "v2.0.0"}},
		BaseModel: weavecontext.BaseModel{
			Ctx: ctx,
		},
		versions: mockVersions,
		question: "Select the version of initiad to download",
	}

	// Simulate pressing down to move to "v2.0.0" and selecting it
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify the next model type
	if m, ok := nextModel.(*RunL1NodeChainIdInput); !ok {
		t.Errorf("Expected next model to be of type *RunL1NodeChainIdInput, but got %T", nextModel)
	} else {
		// Retrieve the updated state from the next model's context
		state := weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)

		// Verify state fields after selection
		assert.Equal(t, "v2.0.0", state.initiadVersion)
		assert.Equal(t, "http://example.com/download/v2.0.0", state.initiadEndpoint)
		assert.Equal(t, "Select the version of initiad to download", model.question)
	}
}

func TestExistingAppReplaceSelectUseCurrentAppLocal(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	model, _ := NewExistingAppReplaceSelect(ctx)

	// Set network to Local
	state := weavecontext.GetCurrentState[RunL1NodeState](model.Ctx)
	state.network = string(Local)
	model.Ctx = weavecontext.SetCurrentState(model.Ctx, state)

	nextModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Check that nextModel is ExistingGenesisChecker and verify state
	if m, ok := nextModel.(*ExistingGenesisChecker); !ok {
		t.Errorf("Expected model to be of type *ExistingGenesisChecker, but got %T", nextModel)
	} else {
		state = weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
		assert.False(t, state.replaceExistingApp)
		assert.Equal(t, "Local", state.network)
	}
	assert.NotNil(t, cmd)
}

func TestExistingAppReplaceSelectUseCurrentAppMainnet(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	model, _ := NewExistingAppReplaceSelect(ctx)

	// Set network to Mainnet
	state := weavecontext.GetCurrentState[RunL1NodeState](model.Ctx)
	state.network = string(Mainnet)
	model.Ctx = weavecontext.SetCurrentState(model.Ctx, state)

	// Simulate selecting "UseCurrentApp"
	nextModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Check that nextModel is CosmovisorAutoUpgradeSelector and verify state
	if m, ok := nextModel.(*CosmovisorAutoUpgradeSelector); !ok {
		t.Errorf("Expected model to be of type *CosmovisorAutoUpgradeSelector, but got %T", nextModel)
	} else {
		state = weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
		assert.False(t, state.replaceExistingApp)
	}
	assert.Nil(t, cmd)
}

func TestExistingAppReplaceSelectReplaceApp(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	model, _ := NewExistingAppReplaceSelect(ctx)

	// Set network to Mainnet for consistency
	state := weavecontext.GetCurrentState[RunL1NodeState](model.Ctx)
	state.network = string(Mainnet)
	model.Ctx = weavecontext.SetCurrentState(model.Ctx, state)

	// Simulate selecting "ReplaceApp"
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown}) // Move to ReplaceApp
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Check that nextModel is RunL1NodeMonikerInput and verify state
	if m, ok := nextModel.(*RunL1NodeMonikerInput); !ok {
		t.Errorf("Expected model to be of type *RunL1NodeMonikerInput, but got %T", nextModel)
	} else {
		state = weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
		assert.True(t, state.replaceExistingApp)
	}
}

func TestRunL1NodeMonikerInputUpdateLocalNetwork(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	model := NewRunL1NodeMonikerInput(ctx)

	// Set the network to Local
	state := weavecontext.GetCurrentState[RunL1NodeState](model.Ctx)
	state.network = string(Local)
	model.Ctx = weavecontext.SetCurrentState(model.Ctx, state)

	// Simulate entering the moniker "Node1"
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Node1")})
	assert.Equal(t, "Node1", model.TextInput.Text) // Verify input is set

	// Simulate pressing Enter to submit the moniker
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify the next model type is MinGasPriceInput and retrieve updated state
	if m, ok := nextModel.(*MinGasPriceInput); !ok {
		t.Errorf("Expected model to be of type *MinGasPriceInput, but got %T", nextModel)
	} else {
		state = weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
	}
	assert.Equal(t, "Node1", state.moniker)
	assert.Empty(t, state.minGasPrice) // minGasPrice should be empty for Local
}

func TestRunL1NodeMonikerInputUpdateTestnetNetwork(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	model := NewRunL1NodeMonikerInput(ctx)

	// Set the network to Testnet
	state := weavecontext.GetCurrentState[RunL1NodeState](model.Ctx)
	state.network = string(Testnet)
	state.chainRegistry, _ = registry.GetChainRegistry(registry.InitiaL1Testnet)
	model.Ctx = weavecontext.SetCurrentState(model.Ctx, state)

	// Simulate entering the moniker "NodeTest"
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("NodeTest")})
	assert.Equal(t, "NodeTest", model.TextInput.Text) // Verify input is set

	// Simulate pressing Enter to submit the moniker
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify the next model type is EnableFeaturesCheckbox and retrieve updated state
	if m, ok := nextModel.(*EnableFeaturesCheckbox); !ok {
		t.Errorf("Expected model to be of type *EnableFeaturesCheckbox, but got %T", nextModel)
	} else {
		state = weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
		assert.Equal(t, "NodeTest", state.moniker)
	}
}

func TestRunL1NodeMonikerInputUpdateMainnetNetwork(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	model := NewRunL1NodeMonikerInput(ctx)

	// Set the network to Mainnet
	state := weavecontext.GetCurrentState[RunL1NodeState](model.Ctx)
	state.network = string(Mainnet)

	state.chainRegistry, _ = registry.GetChainRegistry(registry.InitiaL1Mainnet)
	model.Ctx = weavecontext.SetCurrentState(model.Ctx, state)

	// Simulate entering the moniker "NodeMain"
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("NodeMain")})
	assert.Equal(t, "NodeMain", model.TextInput.Text) // Verify input is set

	// Simulate pressing Enter to submit the moniker
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify the next model type is EnableFeaturesCheckbox and retrieve updated state
	if m, ok := nextModel.(*EnableFeaturesCheckbox); !ok {
		t.Errorf("Expected model to be of type *EnableFeaturesCheckbox, but got %T", nextModel)
	} else {
		state = weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
		assert.Equal(t, "NodeMain", state.moniker)
		assert.NotEmpty(t, state.minGasPrice) // minGasPrice should be set for Mainnet
	}
}

func TestMinGasPriceInputUpdate(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())

	// Define test cases
	tests := []struct {
		name                string
		input               string
		expectTransition    bool
		expectedMinGasPrice string
	}{
		{
			name:                "ValidInput",
			input:               "0.01uatom",
			expectTransition:    true,
			expectedMinGasPrice: "0.01uatom",
		},
		{
			name:                "InvalidInput",
			input:               "invalid-input",
			expectTransition:    false,
			expectedMinGasPrice: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			model := NewMinGasPriceInput(ctx)

			// Simulate entering the input
			model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.input)})
			assert.Equal(t, tc.input, model.TextInput.Text) // Verify input is set

			// Simulate pressing Enter to submit the input
			nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

			// Check if transition should occur
			if tc.expectTransition {
				// Expect transition to EnableFeaturesCheckbox
				if m, ok := nextModel.(*EnableFeaturesCheckbox); !ok {
					t.Errorf("Expected model to be of type *EnableFeaturesCheckbox, but got %T", nextModel)
				} else {
					state := weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
					assert.Equal(t, tc.expectedMinGasPrice, state.minGasPrice) // Verify min-gas-price is saved in the state
				}
			} else {
				// Expect no transition
				assert.Equal(t, model, nextModel) // Should remain in MinGasPriceInput

				// Verify that minGasPrice is not set in the state
				state := weavecontext.GetCurrentState[RunL1NodeState](model.Ctx)
				assert.Empty(t, state.minGasPrice) // minGasPrice should remain empty due to validation failure
			}
		})
	}
}

func TestEnableFeaturesCheckboxUpdate(t *testing.T) {
	// Define test cases with actions for each scenario
	tests := []struct {
		name       string
		action     []tea.KeyMsg
		expectLCD  bool
		expectGRPC bool
	}{
		{
			name: "EnableBoth",
			action: []tea.KeyMsg{
				{Type: tea.KeyDown},  // Navigate to LCD
				{Type: tea.KeySpace}, // Select LCD
				{Type: tea.KeyDown},  // Navigate to gRPC
				{Type: tea.KeySpace}, // Select gRPC
			},
			expectLCD:  true,
			expectGRPC: true,
		},
		{
			name: "EnableLCDOnly",
			action: []tea.KeyMsg{
				{Type: tea.KeyDown},
				{Type: tea.KeyDown},
				{Type: tea.KeySpace},
			},
			expectLCD:  true,
			expectGRPC: false,
		},
		{
			name: "EnableGRPCOnly",
			action: []tea.KeyMsg{
				{Type: tea.KeyDown},
				{Type: tea.KeySpace},
			},
			expectLCD:  false,
			expectGRPC: true,
		},
		{
			name:       "EnableNone",
			action:     []tea.KeyMsg{}, // No selection
			expectLCD:  false,
			expectGRPC: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := weavecontext.NewAppContext(NewRunL1NodeState())
			// Set the network to Mainnet
			state := weavecontext.GetCurrentState[RunL1NodeState](ctx)
			state.network = string(Mainnet)

			state.chainRegistry, _ = registry.GetChainRegistry(registry.InitiaL1Testnet)
			ctx = weavecontext.SetCurrentState(ctx, state)
			model := NewEnableFeaturesCheckbox(ctx)

			// Execute actions defined in the test case
			for _, msg := range tc.action {
				model.Update(msg)
			}

			// Simulate pressing Enter to submit the selections
			nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

			// Verify the next model type is NewSeedsInput
			if m, ok := nextModel.(*SeedsInput); !ok {
				t.Errorf("Expected model to be of type *SeedsInput, but got %T", nextModel)
			} else {
				state := weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
				assert.Equal(t, tc.expectLCD, state.enableLCD)   // Check enableLCD state
				assert.Equal(t, tc.expectGRPC, state.enableGRPC) // Check enableGRPC state
			}
		})
	}
}

func TestSeedsInputUpdate(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	state := weavecontext.GetCurrentState[RunL1NodeState](ctx)

	state.chainType = registry.InitiaL1Testnet
	state.chainRegistry, _ = registry.GetChainRegistry(registry.InitiaL1Testnet)
	ctx = weavecontext.SetCurrentState(ctx, state)

	// Define test cases
	tests := []struct {
		name               string
		input              string
		expectTransition   bool
		expectedSeeds      string
		expectedPrevAnswer string
	}{
		{
			name:               "ValidInput",
			input:              "7a4f10fbdedbb50354163bd43ea6bc4357dd2632@34.87.159.32:26656",
			expectTransition:   true,
			expectedSeeds:      "7a4f10fbdedbb50354163bd43ea6bc4357dd2632@34.87.159.32:26656",
			expectedPrevAnswer: "7a4f10fbdedbb50354163bd43ea6bc4357dd2632@34.87.159.32:26656",
		},
		{
			name:               "InvalidInput",
			input:              "invalid-seed",
			expectTransition:   false,
			expectedSeeds:      "",
			expectedPrevAnswer: "",
		},
		{
			name:               "EmptyInput",
			input:              "",
			expectTransition:   true,
			expectedSeeds:      "",
			expectedPrevAnswer: "None",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			model := NewSeedsInput(ctx)

			// Simulate entering the input
			model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.input)})
			assert.Equal(t, tc.input, model.TextInput.Text) // Verify input is set

			// Simulate pressing Enter to submit the input
			nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

			// Check if transition should occur
			if tc.expectTransition {
				// Expect transition to NewPersistentPeersInput
				if m, ok := nextModel.(*PersistentPeersInput); !ok {
					t.Errorf("Expected model to be of type *PersistentPeersInput, but got %T", nextModel)
				} else {
					state := weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
					assert.Equal(t, tc.expectedSeeds, state.seeds)                  // Verify seeds in state
					assert.Contains(t, state.weave.Render(), tc.expectedPrevAnswer) // Check previous response
				}
			} else {
				// Expect no transition
				assert.Equal(t, model, nextModel) // Should remain in SeedsInput

				// Verify that seeds are not set in the state
				state := weavecontext.GetCurrentState[RunL1NodeState](model.Ctx)
				assert.Empty(t, state.seeds) // Seeds should remain empty due to validation failure
			}
		})
	}
}

func TestPersistentPeersInputUpdate_ValidInput_LocalNetwork(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	state := weavecontext.GetCurrentState[RunL1NodeState](ctx)
	state.network = string(Local)
	ctx = weavecontext.SetCurrentState(ctx, state)

	model, err := NewPersistentPeersInput(ctx)
	assert.Nil(t, err)

	// Simulate entering a valid persistent peer input
	validPeer := "7a4f10fbdedbb50354163bd43ea6bc4357dd2632@34.87.159.32:26656"
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validPeer)})
	assert.Equal(t, validPeer, model.TextInput.Text) // Verify input is set

	// Simulate pressing Enter to submit the valid input
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Expect transition to SelectingPruningStrategy for Local network
	if m, ok := nextModel.(*SelectingPruningStrategy); !ok {
		t.Errorf("Expected model to be of type *SelectingPruningStrategy, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
		assert.Equal(t, validPeer, state.persistentPeers)   // Verify persistent peers in state
		assert.Contains(t, state.weave.Render(), validPeer) // Check previous response
	}
}

func TestPersistentPeersInputUpdate_ValidInput_MainnetNetwork(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	state := weavecontext.GetCurrentState[RunL1NodeState](ctx)
	state.network = string(Mainnet)
	state.chainType = registry.InitiaL1Testnet
	state.chainRegistry, _ = registry.GetChainRegistry(registry.InitiaL1Testnet)
	ctx = weavecontext.SetCurrentState(ctx, state)

	model, err := NewPersistentPeersInput(ctx)
	assert.Nil(t, err)

	// Simulate entering a valid persistent peer input
	validPeer := "7a4f10fbdedbb50354163bd43ea6bc4357dd2632@34.87.159.32:26656"
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validPeer)})
	assert.Equal(t, validPeer, model.TextInput.Text) // Verify input is set

	// Simulate pressing Enter to submit the valid input
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Expect transition to SelectingPruningStrategy for Mainnet network
	if m, ok := nextModel.(*SelectingPruningStrategy); !ok {
		t.Errorf("Expected model to be of type *SelectingPruningStrategy, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
		assert.Equal(t, validPeer, state.persistentPeers)   // Verify persistent peers in state
		assert.Contains(t, state.weave.Render(), validPeer) // Check previous response
	}
}

func TestPersistentPeersInputUpdate_InvalidInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	state := weavecontext.GetCurrentState[RunL1NodeState](ctx)
	state.network = string(Testnet)
	state.chainType = registry.InitiaL1Testnet
	state.chainRegistry, _ = registry.GetChainRegistry(registry.InitiaL1Testnet)
	ctx = weavecontext.SetCurrentState(ctx, state)

	model, err := NewPersistentPeersInput(ctx)
	assert.Nil(t, err)

	// Simulate entering an invalid persistent peer input
	invalidPeer := "invalid-peer"
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(invalidPeer)})
	assert.Equal(t, invalidPeer, model.TextInput.Text) // Verify input is set

	// Simulate pressing Enter to submit the invalid input
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Expect no transition, should remain in PersistentPeersInput
	assert.Equal(t, model, nextModel) // Should remain in PersistentPeersInput

	// Verify that persistent peers are not set in the state
	weavecontext.GetCurrentState[RunL1NodeState](model.Ctx)
	assert.Empty(t, state.persistentPeers) // persistent_peers should remain empty due to validation failure
}

func TestPersistentPeersInputUpdate_EmptyInput_LocalNetwork(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	state := weavecontext.GetCurrentState[RunL1NodeState](ctx)
	state.network = string(Local)
	state.chainRegistry, _ = registry.GetChainRegistry(registry.InitiaL1Testnet)
	ctx = weavecontext.SetCurrentState(ctx, state)

	model, err := NewPersistentPeersInput(ctx)
	assert.Nil(t, err)

	// Simulate entering an empty input
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("")})
	assert.Equal(t, "", model.TextInput.Text) // Verify input is empty

	// Simulate pressing Enter to submit the empty input
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Expect transition to SelectingPruningStrategy for Local network
	if m, ok := nextModel.(*SelectingPruningStrategy); !ok {
		t.Errorf("Expected model to be of type *SelectingPruningStrategy, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
		assert.Equal(t, "", state.persistentPeers)       // Verify persistent_peers in state as empty
		assert.Contains(t, state.weave.Render(), "None") // Check the previous response as "None"
	}
}

func TestExistingGenesisCheckerUpdate_NoExistingGenesis_LocalNetwork(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	state := weavecontext.GetCurrentState[RunL1NodeState](ctx)
	state.existingGenesis = false
	state.network = string(Local)
	state.chainRegistry, _ = registry.GetChainRegistry(registry.InitiaL1Testnet)

	ctx = weavecontext.SetCurrentState(ctx, state)

	model := NewExistingGenesisChecker(ctx)

	// Simulate the loading completion
	model.Loading.EndContext = ctx
	model.Loading.Completing = true
	nextModel, _ := model.Update(tea.KeyMsg{})

	// Expect transition to InitializingAppLoading for Local network
	if _, ok := nextModel.(*CosmovisorAutoUpgradeSelector); !ok {
		t.Errorf("Expected model to be of type *CosmovisorAutoUpgradeSelector, but got %T", nextModel)
	}
}

func TestExistingGenesisCheckerUpdate_NoExistingGenesis_MainnetNetwork(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	state := weavecontext.GetCurrentState[RunL1NodeState](ctx)
	state.existingGenesis = false
	state.network = string(Mainnet)
	ctx = weavecontext.SetCurrentState(ctx, state)

	model := NewExistingGenesisChecker(ctx)

	// Simulate the loading completion
	model.Loading.EndContext = ctx
	model.Loading.Completing = true
	nextModel, _ := model.Update(tea.KeyMsg{})

	// Expect transition to GenesisEndpointInput for Mainnet network
	if _, ok := nextModel.(*GenesisEndpointInput); !ok {
		t.Errorf("Expected model to be of type *GenesisEndpointInput, but got %T", nextModel)
	}
}

func TestExistingGenesisCheckerUpdate_ExistingGenesis(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	state := weavecontext.GetCurrentState[RunL1NodeState](ctx)
	state.existingGenesis = true
	ctx = weavecontext.SetCurrentState(ctx, state)

	model := NewExistingGenesisChecker(ctx)

	// Simulate the loading completion
	model.Loading.EndContext = ctx
	model.Loading.Completing = true
	nextModel, _ := model.Update(tea.KeyMsg{})

	// Expect transition to ExistingGenesisReplaceSelect when existingGenesis is true
	if _, ok := nextModel.(*ExistingGenesisReplaceSelect); !ok {
		t.Errorf("Expected model to be of type *ExistingGenesisReplaceSelect, but got %T", nextModel)
	}
}

func TestExistingGenesisReplaceSelect_Update_UseCurrentGenesis_LocalNetwork(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	state := weavecontext.GetCurrentState[RunL1NodeState](ctx)
	state.network = string(Local)
	ctx = weavecontext.SetCurrentState(ctx, state)

	model, _ := NewExistingGenesisReplaceSelect(ctx)

	// Simulate selecting "UseCurrentGenesis" option
	model.Update(tea.KeyMsg{Type: tea.KeyDown})                  // Navigate to UseCurrentGenesis
	model.Update(tea.KeyMsg{Type: tea.KeySpace})                 // Select UseCurrentGenesis
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection with Enter

	// Expect transition to CosmovisorAutoUpgradeSelector for Local network
	if _, ok := nextModel.(*CosmovisorAutoUpgradeSelector); !ok {
		t.Errorf("Expected model to be of type *CosmovisorAutoUpgradeSelector, but got %T", nextModel)
	}
}

func TestExistingGenesisReplaceSelect_Update_ReplaceGenesis_LocalNetwork(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	state := weavecontext.GetCurrentState[RunL1NodeState](ctx)
	state.network = string(Local)
	ctx = weavecontext.SetCurrentState(ctx, state)

	model, _ := NewExistingGenesisReplaceSelect(ctx)

	// Simulate selecting "ReplaceGenesis" option
	model.Update(tea.KeyMsg{Type: tea.KeyDown})                  // Navigate to ReplaceGenesis
	model.Update(tea.KeyMsg{Type: tea.KeySpace})                 // Select ReplaceGenesis
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection with Enter

	// Expect transition to CosmovisorAutoUpgradeSelector and state update
	if m, ok := nextModel.(*CosmovisorAutoUpgradeSelector); !ok {
		t.Errorf("Expected model to be of type *CosmovisorAutoUpgradeSelector, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
		assert.True(t, state.replaceExistingGenesisWithDefault) // Verify a flag is set for Local network
	}
}

func TestExistingGenesisReplaceSelect_Update_ReplaceGenesis_MainnetNetwork(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	state := weavecontext.GetCurrentState[RunL1NodeState](ctx)
	state.network = string(Mainnet)
	ctx = weavecontext.SetCurrentState(ctx, state)

	model, _ := NewExistingGenesisReplaceSelect(ctx)

	// Simulate selecting "ReplaceGenesis" option
	model.Update(tea.KeyMsg{Type: tea.KeyDown})                  // Navigate to ReplaceGenesis
	model.Update(tea.KeyMsg{Type: tea.KeySpace})                 // Select ReplaceGenesis
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection with Enter

	// Expect transition to GenesisEndpointInput for Mainnet network
	if _, ok := nextModel.(*GenesisEndpointInput); !ok {
		t.Errorf("Expected model to be of type *GenesisEndpointInput, but got %T", nextModel)
	}
}

func TestGenesisEndpointInputUpdate_ValidInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	model := NewGenesisEndpointInput(ctx)

	// Simulate entering a valid URL
	validURL := "https://example.com/genesis.json"
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validURL)})
	assert.Equal(t, validURL, model.TextInput.Text) // Verify input is set

	// Simulate pressing Enter to submit the valid URL
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Expect transition to CosmovisorAutoUpgradeSelector for valid URL
	if m, ok := nextModel.(*CosmovisorAutoUpgradeSelector); !ok {
		t.Errorf("Expected model to be of type *CosmovisorAutoUpgradeSelector, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
		assert.Equal(t, validURL, state.genesisEndpoint)                       // Verify genesis endpoint in state
		assert.Contains(t, state.weave.Render(), common.CleanString(validURL)) // Check previous response
		assert.Nil(t, model.err)                                               // Error should be nil for valid URL
	}
}

func TestGenesisEndpointInputUpdate_InvalidInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	model := NewGenesisEndpointInput(ctx)

	// Simulate entering an invalid URL
	invalidURL := "invalid-url"
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(invalidURL)})
	assert.Equal(t, invalidURL, model.TextInput.Text) // Verify input is set

	// Simulate pressing Enter to submit the invalid URL
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Expect no transition for invalid URL, should remain in GenesisEndpointInput
	assert.Equal(t, model, nextModel) // Should remain in GenesisEndpointInput

	// Verify error is set and the state is not updated
	assert.NotNil(t, model.err) // Error should be set for invalid URL
	state := weavecontext.GetCurrentState[RunL1NodeState](model.Ctx)
	assert.Empty(t, state.genesisEndpoint) // genesisEndpoint should remain empty due to validation failure
}

func TestInitializingAppLoading_Update_LocalNetwork(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	state := weavecontext.GetCurrentState[RunL1NodeState](ctx)
	state.network = string(Local)
	ctx = weavecontext.SetCurrentState(ctx, state)

	model := NewInitializingAppLoading(ctx)

	// Simulate loading completion
	model.Loading.Completing = true
	model.Loading.EndContext = ctx
	nextModel, _ := model.Update(tea.KeyMsg{})

	// Expect the model to quit for Local network
	if _, ok := nextModel.(*TerminalState); !ok {
		t.Errorf("Expected model to be of type *TerminalState, but got %T", nextModel)
	}
	// Verify a final state message
	state = weavecontext.GetCurrentState[RunL1NodeState](model.Ctx)
	assert.Contains(t, state.weave.Render(), "Initialization successful.\n")
}

func TestInitializingAppLoading_Update_MainnetNetwork(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	state := weavecontext.GetCurrentState[RunL1NodeState](ctx)
	state.network = string(Mainnet)
	ctx = weavecontext.SetCurrentState(ctx, state)

	model := NewInitializingAppLoading(ctx)

	// Simulate loading completion
	model.Loading.Completing = true
	model.Loading.EndContext = ctx
	nextModel, _ := model.Update(tea.KeyMsg{})

	// Expect transition to SyncMethodSelect for Mainnet network
	if m, ok := nextModel.(*SyncMethodSelect); !ok {
		t.Errorf("Expected model to be of type *SyncMethodSelect, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
		assert.Contains(t, state.weave.Render(), "Initialization successful.\n") // Verify a final state message
	}
}

func TestInitializingAppLoading_Update_TestnetNetwork(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	state := weavecontext.GetCurrentState[RunL1NodeState](ctx)
	state.network = string(Testnet)
	ctx = weavecontext.SetCurrentState(ctx, state)

	model := NewInitializingAppLoading(ctx)

	// Simulate loading completion
	model.Loading.Completing = true
	model.Loading.EndContext = ctx
	nextModel, _ := model.Update(tea.KeyMsg{})

	// Expect transition to SyncMethodSelect for Testnet network
	if m, ok := nextModel.(*SyncMethodSelect); !ok {
		t.Errorf("Expected model to be of type *SyncMethodSelect, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
		assert.Contains(t, state.weave.Render(), "Initialization successful.\n") // Verify a final state message
	}
}

func TestSyncMethodSelect_Update_NoSync(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	model := NewSyncMethodSelect(ctx)

	// Simulate selecting "NoSync" option
	model.Update(tea.KeyMsg{Type: tea.KeyDown})                  // Navigate to NoSync
	model.Update(tea.KeyMsg{Type: tea.KeyDown})                  // Navigate to NoSync
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection with Enter

	// Expect transition to TerminalState and command to quit
	if _, ok := nextModel.(*TerminalState); !ok {
		t.Errorf("Expected model to be of type *TerminalState, but got %T", nextModel)
	}
}

func TestSyncMethodSelect_Update_Snapshot(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	model := NewSyncMethodSelect(ctx)

	// Simulate selecting "Snapshot" option
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection with Enter

	// Expect transition to ExistingDataChecker
	if m, ok := nextModel.(*ExistingDataChecker); !ok {
		t.Errorf("Expected model to be of type *ExistingDataChecker, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
		assert.Equal(t, string(Snapshot), state.syncMethod) // Verify sync method in state
	}
}

func TestSyncMethodSelect_Update_StateSync(t *testing.T) {
	ctx := weavecontext.NewAppContext(NewRunL1NodeState())
	model := NewSyncMethodSelect(ctx)

	// Simulate selecting "StateSync" option
	model.Update(tea.KeyMsg{Type: tea.KeyDown})                  // Move down to StateSync
	nextModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Confirm selection with Enter

	// Expect transition to ExistingDataChecker
	if m, ok := nextModel.(*ExistingDataChecker); !ok {
		t.Errorf("Expected model to be of type *ExistingDataChecker, but got %T", nextModel)
	} else {
		state := weavecontext.GetCurrentState[RunL1NodeState](m.Ctx)
		assert.Equal(t, string(StateSync), state.syncMethod) // Verify sync method in state
	}
}
