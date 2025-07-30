package minitia

import (
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"
	"github.com/test-go/testify/assert"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/config"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/styles"
	"github.com/initia-labs/weave/types"
	"github.com/initia-labs/weave/ui"
)

func TestMain(m *testing.M) {
	analytics.Client = &analytics.NoOpClient{}
	exitCode := m.Run()
	os.Exit(exitCode)
}

func InitializeViperForTest(t *testing.T) {
	viper.Reset()

	viper.SetConfigType("json")
	err := viper.ReadConfig(strings.NewReader(config.DefaultConfigTemplate))

	if err != nil {
		t.Fatalf("failed to initialize viper: %v", err)
	}
}

func TestNewExistingMinitiaChecker(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	model := NewExistingMinitiaChecker(ctx)

	assert.NotNil(t, model, "Expected ExistingMinitiaChecker to be created")
	assert.NotNil(t, model.Init(), "Expected Init command to be returned")
	assert.Contains(t, model.View(), "Checking for an existing rollup app...")
}

func TestExistingMinitiaChecker_Update(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	model := NewExistingMinitiaChecker(ctx)
	model.Loading.EndContext = model.Ctx
	model.Loading.Completing = true

	nextModel, _ := model.Update(&tea.KeyMsg{})

	if _, ok := nextModel.(*NetworkSelect); !ok {
		t.Errorf("Expected model to be of type *NetworkSelect, but got %T", nextModel)
	}

	model = NewExistingMinitiaChecker(ctx)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	state.existingMinitiaApp = true
	ctx = weavecontext.SetCurrentState(ctx, state)
	model.Loading.EndContext = ctx
	model.Loading.Completing = true

	nextModel, _ = model.Update(&tea.KeyMsg{})
	if _, ok := nextModel.(*DeleteExistingMinitiaInput); !ok {
		t.Errorf("Expected model to be of type *DeleteExistingMinitiaInput, but got %T", nextModel)
	}
}

func TestExistingMinitiaChecker_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	model := NewExistingMinitiaChecker(ctx)

	view := model.View()
	assert.Contains(t, view, "When launching a rollup,", "Expected the view to contain the launch message")
}

func TestExistingMinitiaChecker(t *testing.T) {
	testCases := []struct {
		name            string
		existingMinitia bool
		expectedModel   interface{}
	}{
		{
			name:            "No .minitia",
			existingMinitia: false,
			expectedModel:   &NetworkSelect{},
		},
		{
			name:            "With .minitia",
			existingMinitia: true,
			expectedModel:   &DeleteExistingMinitiaInput{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockState := &LaunchState{
				existingMinitiaApp: tc.existingMinitia,
			}
			ctx := weavecontext.NewAppContext(*mockState)

			model := NewExistingMinitiaChecker(ctx)

			model.Loading.EndContext = ctx
			model.Loading.Completing = true
			m, _ := model.Update(&tea.KeyMsg{})

			assert.IsType(t, tc.expectedModel, m)
		})
	}
}

func TestNewDeleteExistingMinitiaInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())
	model := NewDeleteExistingMinitiaInput(ctx)

	assert.Nil(t, model.Init(), "Expected Init command to be returned")
	assert.NotNil(t, model, "Expected DeleteExistingMinitiaInput to be created")
	assert.Equal(t, "Type `delete` to delete the .minitia folder and proceed with weave rollup launch", model.GetQuestion())
	assert.NotNil(t, model.TextInput, "Expected TextInput to be initialized")
	assert.Equal(t, "Type `delete` to delete, Ctrl+C to keep the folder and quit this command.", model.TextInput.Placeholder, "Expected placeholder to be set correctly")
	assert.NotNil(t, model.TextInput.ValidationFn, "Expected validation function to be set")
}

func TestDeleteExistingMinitiaInput_Update(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())
	model := NewDeleteExistingMinitiaInput(ctx)

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("incorrect input")})
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Contains(t, updatedModel.View(), "please type `delete` to proceed")
	assert.IsType(t, &DeleteExistingMinitiaInput{}, updatedModel, "Expected model to stay in DeleteExistingMinitiaInput")
}

func TestDeleteExistingMinitiaInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())
	ctx = weavecontext.SetMinitiaHome(ctx, "~/.minitia")
	model := NewDeleteExistingMinitiaInput(ctx)

	view := model.View()
	assert.Contains(t, view, "🚨 Existing ~/.minitia folder detected.", "Expected warning message for existing folder")
	assert.Contains(t, view, "permanently deleted and cannot be reversed.", "Expected deletion warning")
	assert.Contains(t, view, "Type `delete` to delete", "Expected prompt for deletion confirmation")
}

func TestNewNetworkSelect(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	model, _ := NewNetworkSelect(ctx)

	assert.Nil(t, model.Init(), "Expected Init command to be returned")
	assert.NotNil(t, model, "Expected NetworkSelect to be created")
	assert.Equal(t, "Select the Initia L1 network you want to connect your rollup to", model.GetQuestion())
	assert.Contains(t, model.Selector.Options, Testnet, "Expected Testnet to be available as a network option")
	assert.Contains(t, model.Selector.Options, Mainnet, "Expected Mainnet to be available as a network option")
}

func TestNetworkSelect_Update_Selection(t *testing.T) {
	InitializeViperForTest(t)

	ctx := weavecontext.NewAppContext(*NewLaunchState())
	state := weavecontext.GetCurrentState[LaunchState](ctx)
	state.weave = types.WeaveState{}
	model, _ := NewNetworkSelect(ctx)

	msg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, cmd := model.Update(msg)

	msg = tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd = updatedModel.Update(msg)

	assert.IsType(t, &VMTypeSelect{}, updatedModel, "Expected model to transition to VMTypeSelect after network selection")
	assert.Nil(t, cmd, "Expected no command after network selection")
}

func TestNetworkSelect_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	model, _ := NewNetworkSelect(ctx)

	view := model.View()
	assert.Contains(t, view, "Select the Initia L1 network you want to connect your rollup to", "Expected question prompt in the view")
	assert.Contains(t, view, "Testnet", "Expected Testnet option to be displayed")
	assert.Contains(t, view, "Mainnet", "Expected Mainnet option to be displayed")
}

func TestNewVMTypeSelect(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	model := NewVMTypeSelect(ctx)

	assert.NotNil(t, model, "Expected VMTypeSelect to be created")
	assert.Equal(t, "Select the Virtual Machine (VM) for your rollup", model.GetQuestion())
	assert.Contains(t, model.Selector.Options, Move, "Expected Move to be an available VM option")
	assert.Contains(t, model.Selector.Options, Wasm, "Expected Wasm to be an available VM option")
	assert.Contains(t, model.Selector.Options, EVM, "Expected EVM to be an available VM option")
}

func TestVMTypeSelect_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	model := NewVMTypeSelect(ctx)

	assert.Nil(t, model.Init(), "Expected Init command to return nil")
}

func TestVMTypeSelect_Update(t *testing.T) {
	testCases := []struct {
		name           string
		keyPresses     []tea.KeyMsg
		expectedVMType string
		expectedModel  interface{}
	}{
		{
			name:           "Select Move VM type",
			keyPresses:     []tea.KeyMsg{{Type: tea.KeyEnter}},
			expectedVMType: "Move",
			expectedModel:  &LatestVersionLoading{},
		},
		{
			name:           "Select Wasm VM type",
			keyPresses:     []tea.KeyMsg{{Type: tea.KeyDown}, {Type: tea.KeyEnter}},
			expectedVMType: "Wasm",
			expectedModel:  &LatestVersionLoading{},
		},
		{
			name:           "Select EVM VM type",
			keyPresses:     []tea.KeyMsg{{Type: tea.KeyDown}, {Type: tea.KeyDown}, {Type: tea.KeyEnter}},
			expectedVMType: "EVM",
			expectedModel:  &LatestVersionLoading{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := weavecontext.NewAppContext(*NewLaunchState())

			model := NewVMTypeSelect(ctx)

			var m tea.Model = model
			var cmd tea.Cmd
			for _, keyPress := range tc.keyPresses {
				m, cmd = m.Update(keyPress)
				if cmd != nil {
					cmd()
				}
			}

			nextModel := m.(*LatestVersionLoading)
			state := weavecontext.GetCurrentState[LaunchState](nextModel.Ctx)
			assert.Equal(t, tc.expectedVMType, state.vmType, "Expected vmType to be set correctly")

			assert.IsType(t, tc.expectedModel, m, "Expected model to transition to the correct type after VM type selection")
			assert.NotNil(t, cmd, "Expected Init command after VM type selection")
		})
	}
}

func TestVMTypeSelect_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	model := NewVMTypeSelect(ctx)

	view := model.View()
	assert.Contains(t, view, "Select the Virtual Machine (VM) for your rollup", "Expected question prompt in the view")
	assert.Contains(t, view, "Move", "Expected Move option to be displayed")
	assert.Contains(t, view, "Wasm", "Expected Wasm option to be displayed")
	assert.Contains(t, view, "EVM", "Expected EVM option to be displayed")
}

func TestNetworkSelect_SaveToState(t *testing.T) {
	InitializeViperForTest(t)
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	networkSelect, _ := NewNetworkSelect(ctx)
	//m, _ := networkSelect.Update(tea.KeyMsg{Type: tea.KeyEnter})
	//
	//assert.Equal(t, "", mockState.l1ChainId)
	//assert.Equal(t, "", mockState.l1RPC)
	//
	//assert.IsType(t, m, &VMTypeSelect{})
	//
	//_, _ = networkSelect.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ := networkSelect.Update(tea.KeyMsg{Type: tea.KeyEnter})

	nextModel := m.(*VMTypeSelect)
	state := weavecontext.GetCurrentState[LaunchState](nextModel.Ctx)
	assert.Equal(t, "initiation-2", state.l1ChainId)
	assert.Equal(t, "https://rpc.testnet.initia.xyz:443", state.l1RPC)

	assert.IsType(t, m, &VMTypeSelect{})
}

func TestChainIdInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewChainIdInput(ctx)

	cmd := input.Init()
	assert.Nil(t, cmd, "Expected Init to return nil")
}

func TestChainIdInput_Update(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewChainIdInput(ctx)

	typedInput := "test-chain-id"
	keyPress := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(typedInput)}
	updatedModel, _ := input.Update(keyPress)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, finalCmd := updatedModel.Update(enterPress)

	nextModel := finalModel.(*GasDenomInput)
	state := weavecontext.GetCurrentState[LaunchState](nextModel.Ctx)
	assert.Equal(t, typedInput, state.chainId, "Expected chainId to be set correctly")
	assert.IsType(t, &GasDenomInput{}, finalModel, "Expected model to transition to GasDenomInput")
	assert.Nil(t, finalCmd, "Expected no command after input")
}

func TestChainIdInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewChainIdInput(ctx)

	view := input.View()
	assert.Contains(t, view, "Specify rollup chain ID", "Expected question prompt in the view")
	assert.Contains(t, view, "Enter your chain ID ex. local-rollup-1", "Expected placeholder in the view")
}

func TestGasDenomInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewGasDenomInput(ctx)

	cmd := input.Init()
	assert.Nil(t, cmd, "Expected Init to return nil")
}

func TestGasDenomInput_Update(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewGasDenomInput(ctx)

	typedInput := "test-denom"
	keyPress := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(typedInput)}
	updatedModel, _ := input.Update(keyPress)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, finalCmd := updatedModel.Update(enterPress)

	nextModel := finalModel.(*MonikerInput)
	state := weavecontext.GetCurrentState[LaunchState](nextModel.Ctx)
	assert.Equal(t, typedInput, state.gasDenom, "Expected gasDenom to be set correctly")
	assert.IsType(t, &MonikerInput{}, finalModel, "Expected model to transition to MonikerInput")
	assert.Nil(t, finalCmd, "Expected no command after input")
}

func TestGasDenomInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewGasDenomInput(ctx)

	view := input.View()
	assert.Contains(t, view, "Specify rollup gas denom", "Expected question prompt in the view")
	assert.Contains(t, view, `Press tab to use "umin"`, "Expected placeholder in the view")
}

func TestMonikerInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewMonikerInput(ctx)

	cmd := input.Init()
	assert.Nil(t, cmd, "Expected Init to return nil")
}

func TestMonikerInput_Update(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewMonikerInput(ctx)

	typedInput := "test-moniker"
	keyPress := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(typedInput)}
	updatedModel, _ := input.Update(keyPress)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, finalCmd := updatedModel.Update(enterPress)

	nextModel := finalModel.(*OpBridgeSubmissionIntervalInput)
	state := weavecontext.GetCurrentState[LaunchState](nextModel.Ctx)
	assert.Equal(t, typedInput, state.moniker, "Expected moniker to be set correctly")
	assert.IsType(t, &OpBridgeSubmissionIntervalInput{}, finalModel, "Expected model to transition to OpBridgeSubmissionIntervalInput")
	assert.Nil(t, finalCmd, "Expected no command after input")
}

func TestMonikerInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewMonikerInput(ctx)

	view := input.View()
	assert.Contains(t, view, "Specify rollup node moniker", "Expected question prompt in the view")
	assert.Contains(t, view, `Press tab to use "operator"`, "Expected placeholder in the view")
}

func TestNewOpBridgeSubmissionIntervalInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewOpBridgeSubmissionIntervalInput(ctx)

	assert.NotNil(t, input)
	assert.Equal(t, "Specify OP bridge config: Submission Interval (format s, m or h - ex. 30s, 5m, 12h)", input.GetQuestion())
	assert.Equal(t, "Press tab to use “1m”", input.TextInput.Placeholder)
	assert.Equal(t, "1m", input.TextInput.DefaultValue)
	assert.NotNil(t, input.TextInput.ValidationFn)
}

func TestOpBridgeSubmissionIntervalInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewOpBridgeSubmissionIntervalInput(ctx)

	cmd := input.Init()
	assert.Nil(t, cmd)
}

func TestOpBridgeSubmissionIntervalInput_Update(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewOpBridgeSubmissionIntervalInput(ctx)

	typedInput := "5m"
	keyPress := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(typedInput)}
	updatedModel, _ := input.Update(keyPress)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, cmd := updatedModel.Update(enterPress)

	nextModel := finalModel.(*OpBridgeOutputFinalizationPeriodInput)
	state := weavecontext.GetCurrentState[LaunchState](nextModel.Ctx)
	assert.IsType(t, &OpBridgeOutputFinalizationPeriodInput{}, finalModel)
	assert.Equal(t, "5m", state.opBridgeSubmissionInterval)
	assert.Nil(t, cmd)
}

func TestOpBridgeSubmissionIntervalInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewOpBridgeSubmissionIntervalInput(ctx)

	view := input.View()
	assert.Contains(t, view, "Specify OP bridge config: Submission Interval (format s, m or h - ex. 30s, 5m, 12h)", "Expected question prompt in the view")
	assert.Contains(t, view, "Press tab to use “1m”", "Expected placeholder in the view")
	assert.Contains(t, view, "1m", "Expected default value in the view") // Ensure the default value is displayed in the view
}

func TestNewOpBridgeOutputFinalizationPeriodInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewOpBridgeOutputFinalizationPeriodInput(ctx)

	assert.NotNil(t, input)
	assert.Equal(t, "Specify OP bridge config: Output Finalization Period (format s, m or h - ex. 30s, 5m, 12h)", input.GetQuestion())
	assert.Equal(t, "Press tab to use “168h” (7 days)", input.TextInput.Placeholder)
	assert.Equal(t, "168h", input.TextInput.DefaultValue)
	assert.NotNil(t, input.TextInput.ValidationFn)
}

func TestOpBridgeOutputFinalizationPeriodInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewOpBridgeOutputFinalizationPeriodInput(ctx)

	cmd := input.Init()
	assert.Nil(t, cmd)
}

func TestOpBridgeOutputFinalizationPeriodInput_Update(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewOpBridgeOutputFinalizationPeriodInput(ctx)

	typedInput := "12h"
	keyPress := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(typedInput)}
	updatedModel, _ := input.Update(keyPress)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, cmd := updatedModel.Update(enterPress)

	nextModel := finalModel.(*OracleEnableSelect)
	state := weavecontext.GetCurrentState[LaunchState](nextModel.Ctx)
	assert.IsType(t, &OracleEnableSelect{}, finalModel)
	assert.Equal(t, "12h", state.opBridgeOutputFinalizationPeriod)
	assert.Nil(t, cmd)
}

func TestOpBridgeOutputFinalizationPeriodInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewOpBridgeOutputFinalizationPeriodInput(ctx)

	view := input.View()
	assert.Contains(t, view, "Specify OP bridge config: Output Finalization Period (format s, m or h - ex. 30s, 5m, 12h)", "Expected question prompt in the view")
	assert.Contains(t, view, "Press tab to use “168h”", "Expected placeholder in the view")
	assert.Contains(t, view, "168h", "Expected default value in the view")
}

func TestNewOpBridgeBatchSubmissionTargetSelect(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewOpBridgeBatchSubmissionTargetSelect(ctx)

	assert.NotNil(t, input)
	assert.Equal(t, "Where should the rollup blocks and transaction data be submitted?", input.GetQuestion())
}

func TestOpBridgeBatchSubmissionTargetSelect_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewOpBridgeBatchSubmissionTargetSelect(ctx)

	cmd := input.Init()
	assert.Nil(t, cmd)
}

func TestOpBridgeBatchSubmissionTargetSelect_Update(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewOpBridgeBatchSubmissionTargetSelect(ctx)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	thisModel, cmd := input.Update(enterPress)

	nextModel := thisModel.(*OracleEnableSelect)
	state := weavecontext.GetCurrentState[LaunchState](nextModel.Ctx)
	assert.IsType(t, &OracleEnableSelect{}, thisModel)
	assert.Equal(t, "CELESTIA", state.opBridgeBatchSubmissionTarget)
	assert.Nil(t, cmd)

	ctx = weavecontext.NewAppContext(state)
	input = NewOpBridgeBatchSubmissionTargetSelect(ctx)

	downPress := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := input.Update(downPress)

	enterPress = tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, cmd := updatedModel.Update(enterPress)

	nextModel = finalModel.(*OracleEnableSelect)
	state = weavecontext.GetCurrentState[LaunchState](nextModel.Ctx)
	assert.IsType(t, &OracleEnableSelect{}, finalModel)
	assert.Equal(t, "INITIA", state.opBridgeBatchSubmissionTarget)

}

func TestOpBridgeBatchSubmissionTargetSelect_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewOpBridgeBatchSubmissionTargetSelect(ctx)

	view := input.View()
	assert.Contains(t, view, "Where should the rollup blocks and transaction data be submitted?", "Expected question prompt in the view")
}

func TestNewOracleEnableSelect(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	selectInput := NewOracleEnableSelect(ctx)

	assert.NotNil(t, selectInput)
	assert.Equal(t, "Would you like to enable oracle price feed from L1?", selectInput.GetQuestion())
	assert.Equal(t, 2, len(selectInput.Options))
}

func TestOracleEnableSelect_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	selectInput := NewOracleEnableSelect(ctx)

	cmd := selectInput.Init()
	assert.Nil(t, cmd)
}

func TestOracleEnableSelect_Update(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	selectInput := NewOracleEnableSelect(ctx)

	downPress := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := selectInput.Update(downPress)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, cmd := updatedModel.Update(enterPress)

	nextModel := finalModel.(*SystemKeysSelect)
	state := weavecontext.GetCurrentState[LaunchState](nextModel.Ctx)
	assert.IsType(t, &SystemKeysSelect{}, finalModel)
	assert.False(t, state.enableOracle)
	assert.Nil(t, cmd)

	ctx = weavecontext.NewAppContext(state)
	selectInput = NewOracleEnableSelect(ctx)
	downPress = tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = selectInput.Update(downPress)
	upPress := tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ = updatedModel.Update(upPress)

	enterPress = tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, cmd = updatedModel.Update(enterPress)

	nextModel = finalModel.(*SystemKeysSelect)
	state = weavecontext.GetCurrentState[LaunchState](nextModel.Ctx)
	assert.IsType(t, &SystemKeysSelect{}, finalModel)
	assert.True(t, state.enableOracle)
	assert.Nil(t, cmd)
}

func TestOracleEnableSelect_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	selectInput := NewOracleEnableSelect(ctx)

	view := selectInput.View()
	assert.Contains(t, view, "Would you like to enable oracle price feed from L1?", "Expected question prompt in the view")
}

func TestNewSystemKeysSelect(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	selectInput := NewSystemKeysSelect(ctx)

	assert.NotNil(t, selectInput)
	assert.Equal(t, "Select a setup method for the system keys", selectInput.GetQuestion())
	assert.Equal(t, 2, len(selectInput.Options))
}

func TestSystemKeysSelect_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	selectInput := NewSystemKeysSelect(ctx)

	cmd := selectInput.Init()
	assert.Nil(t, cmd)
}

func TestSystemKeysSelect_Update(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	selectInput := NewSystemKeysSelect(ctx)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, _ := selectInput.Update(enterPress)

	nextModel := finalModel.(*ExistingGasStationChecker)
	state := weavecontext.GetCurrentState[LaunchState](nextModel.Ctx)
	assert.IsType(t, &ExistingGasStationChecker{}, finalModel)
	assert.True(t, state.generateKeys)

	ctx = weavecontext.NewAppContext(*NewLaunchState())
	selectInput = NewSystemKeysSelect(ctx)
	downPress := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := selectInput.Update(downPress)
	finalModel, _ = updatedModel.Update(enterPress)

	model := finalModel.(*SystemKeyOperatorMnemonicInput)
	state = weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.IsType(t, &SystemKeyOperatorMnemonicInput{}, finalModel)
	assert.False(t, state.generateKeys)
}

func TestSystemKeysSelect_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	selectInput := NewSystemKeysSelect(ctx)

	view := selectInput.View()
	assert.Contains(t, view, "System keys are required for each of the following roles:", "Expected roles prompt in the view")
	assert.Contains(t, view, "Select a setup method for the system keys", "Expected question prompt in the view")
}

func TestSystemKeyOperatorMnemonicInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyOperatorMnemonicInput(ctx)

	cmd := input.Init()
	assert.Nil(t, cmd, "Init should return nil, as it has no side-effect commands")
}

func TestSystemKeyOperatorMnemonicInput_Update(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyOperatorMnemonicInput(ctx)

	validMnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validMnemonic)}
	nextModel, _ := input.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, _ := nextModel.Update(enterPress)

	model := finalModel.(*SystemKeyBridgeExecutorMnemonicInput)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.IsType(t, &SystemKeyBridgeExecutorMnemonicInput{}, finalModel)
	assert.Equal(t, validMnemonic, state.systemKeyOperatorMnemonic)
	assert.Contains(t, state.weave.PreviousResponse, styles.RenderPreviousResponse(
		styles.DotsSeparator, input.GetQuestion(), []string{"Operator"}, styles.HiddenMnemonicText))

	ctx = weavecontext.NewAppContext(*NewLaunchState())
	input = NewSystemKeyOperatorMnemonicInput(ctx)
	invalidMnemonic := "invalid mnemonic phrase"
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(invalidMnemonic)}
	nextModel, _ = input.Update(keyMsg)
	finalModel, _ = input.Update(enterPress)

	checkModel := finalModel.(*SystemKeyOperatorMnemonicInput)
	state = weavecontext.GetCurrentState[LaunchState](checkModel.Ctx)
	assert.NotEqual(t, invalidMnemonic, state.systemKeyOperatorMnemonic)
	assert.NotContains(t, state.weave.PreviousResponse, styles.RenderPreviousResponse(
		styles.DotsSeparator, input.GetQuestion(), []string{"Operator"}, styles.HiddenMnemonicText))
}

func TestSystemKeyOperatorMnemonicInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyOperatorMnemonicInput(ctx)

	view := input.View()
	assert.Contains(t, view, input.GetQuestion())
	assert.Contains(t, view, "Enter the mnemonic")
}

func TestNewSystemKeyBridgeExecutorMnemonicInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyBridgeExecutorMnemonicInput(ctx)

	assert.NotNil(t, input, "Expected non-nil input model")
	assert.Equal(t, "Specify the mnemonic for the bridge executor", input.GetQuestion())
	assert.Equal(t, "Enter the mnemonic", input.TextInput.Placeholder)
}

func TestSystemKeyBridgeExecutorMnemonicInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyBridgeExecutorMnemonicInput(ctx)

	cmd := input.Init()
	assert.Nil(t, cmd, "Init should return nil, as it has no side-effect commands")
}

func TestSystemKeyBridgeExecutorMnemonicInput_Update(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyBridgeExecutorMnemonicInput(ctx)

	validMnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validMnemonic)}
	nextModel, _ := input.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, _ := nextModel.Update(enterPress)

	model := finalModel.(*SystemKeyOutputSubmitterMnemonicInput)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.IsType(t, &SystemKeyOutputSubmitterMnemonicInput{}, finalModel)
	assert.Equal(t, validMnemonic, state.systemKeyBridgeExecutorMnemonic)
	assert.Contains(t, state.weave.PreviousResponse, styles.RenderPreviousResponse(
		styles.DotsSeparator, input.GetQuestion(), []string{"bridge executor"}, styles.HiddenMnemonicText))

	ctx = weavecontext.NewAppContext(*NewLaunchState())

	input = NewSystemKeyBridgeExecutorMnemonicInput(ctx)
	invalidMnemonic := "invalid mnemonic phrase"
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(invalidMnemonic)}
	nextModel, _ = input.Update(keyMsg)
	finalModel, _ = input.Update(enterPress)

	checkModel := finalModel.(*SystemKeyBridgeExecutorMnemonicInput)
	state = weavecontext.GetCurrentState[LaunchState](checkModel.Ctx)
	assert.NotEqual(t, invalidMnemonic, state.systemKeyBridgeExecutorMnemonic)
	assert.NotContains(t, state.weave.PreviousResponse, styles.RenderPreviousResponse(
		styles.DotsSeparator, input.GetQuestion(), []string{"bridge executor"}, styles.HiddenMnemonicText))
}

func TestSystemKeyBridgeExecutorMnemonicInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyBridgeExecutorMnemonicInput(ctx)

	view := input.View()
	assert.Contains(t, view, input.GetQuestion())
	assert.Contains(t, view, "Enter the mnemonic")
}

func TestNewSystemKeyOutputSubmitterMnemonicInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyOutputSubmitterMnemonicInput(ctx)

	assert.NotNil(t, input, "Expected non-nil input model")
	assert.Equal(t, "Specify the mnemonic for the output submitter", input.GetQuestion())
	assert.Equal(t, "Enter the mnemonic", input.TextInput.Placeholder)
}

func TestSystemKeyOutputSubmitterMnemonicInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyOutputSubmitterMnemonicInput(ctx)

	cmd := input.Init()
	assert.Nil(t, cmd, "Init should return nil, as it has no side-effect commands")
}

func TestSystemKeyOutputSubmitterMnemonicInput_Update(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyOutputSubmitterMnemonicInput(ctx)

	validMnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validMnemonic)}
	nextModel, _ := input.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, _ := nextModel.Update(enterPress)

	model := finalModel.(*SystemKeyBatchSubmitterMnemonicInput)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.IsType(t, &SystemKeyBatchSubmitterMnemonicInput{}, finalModel)
	assert.Equal(t, validMnemonic, state.systemKeyOutputSubmitterMnemonic)
	assert.Contains(t, state.weave.PreviousResponse, styles.RenderPreviousResponse(
		styles.DotsSeparator, input.GetQuestion(), []string{"output submitter"}, styles.HiddenMnemonicText))

	ctx = weavecontext.NewAppContext(*NewLaunchState())
	input = NewSystemKeyOutputSubmitterMnemonicInput(ctx)
	invalidMnemonic := "invalid mnemonic phrase"
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(invalidMnemonic)}
	nextModel, _ = input.Update(keyMsg)
	finalModel, _ = input.Update(enterPress)

	checkerModel := finalModel.(*SystemKeyOutputSubmitterMnemonicInput)
	state = weavecontext.GetCurrentState[LaunchState](checkerModel.Ctx)
	assert.NotEqual(t, invalidMnemonic, state.systemKeyOutputSubmitterMnemonic)
	assert.NotContains(t, state.weave.PreviousResponse, styles.RenderPreviousResponse(
		styles.DotsSeparator, input.GetQuestion(), []string{"output submitter"}, styles.HiddenMnemonicText))
}

func TestSystemKeyOutputSubmitterMnemonicInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyOutputSubmitterMnemonicInput(ctx)

	view := input.View()
	assert.Contains(t, view, input.GetQuestion())
	assert.Contains(t, view, "Enter the mnemonic")
}

func TestNewSystemKeyBatchSubmitterMnemonicInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyBatchSubmitterMnemonicInput(ctx)

	assert.NotNil(t, input, "Expected non-nil input model")
	assert.Equal(t, "Specify the mnemonic for the batch submitter", input.GetQuestion())
	assert.Equal(t, "Enter the mnemonic", input.TextInput.Placeholder)
}

func TestSystemKeyBatchSubmitterMnemonicInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyBatchSubmitterMnemonicInput(ctx)

	cmd := input.Init()
	assert.Nil(t, cmd, "Init should return nil, as it has no side-effect commands")
}

func TestSystemKeyBatchSubmitterMnemonicInput_Update(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyBatchSubmitterMnemonicInput(ctx)

	// Test valid mnemonic input
	validMnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validMnemonic)}
	nextModel, _ := input.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, _ := nextModel.Update(enterPress)

	model := finalModel.(*SystemKeyChallengerMnemonicInput)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.IsType(t, &SystemKeyChallengerMnemonicInput{}, finalModel)
	assert.Equal(t, validMnemonic, state.systemKeyBatchSubmitterMnemonic)
	assert.Contains(t, state.weave.PreviousResponse, styles.RenderPreviousResponse(
		styles.DotsSeparator, input.GetQuestion(), []string{"batch submitter"}, styles.HiddenMnemonicText))

	// Test invalid mnemonic input
	ctx = weavecontext.NewAppContext(*NewLaunchState())
	input = NewSystemKeyBatchSubmitterMnemonicInput(ctx)
	invalidMnemonic := "invalid mnemonic phrase"
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(invalidMnemonic)}
	nextModel, _ = input.Update(keyMsg)
	finalModel, _ = input.Update(enterPress)

	checkerModel := finalModel.(*SystemKeyBatchSubmitterMnemonicInput)
	state = weavecontext.GetCurrentState[LaunchState](checkerModel.Ctx)
	assert.NotEqual(t, invalidMnemonic, state.systemKeyBatchSubmitterMnemonic)
	assert.NotContains(t, state.weave.PreviousResponse, styles.RenderPreviousResponse(
		styles.DotsSeparator, input.GetQuestion(), []string{"batch submitter"}, styles.HiddenMnemonicText))
}

func TestSystemKeyBatchSubmitterMnemonicInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyBatchSubmitterMnemonicInput(ctx)

	view := input.View()
	assert.Contains(t, view, input.GetQuestion(), "Expected question prompt in the view")
	assert.Contains(t, view, "Enter the mnemonic", "Expected mnemonic prompt in the view")
}

func TestNewSystemKeyChallengerMnemonicInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyChallengerMnemonicInput(ctx)

	assert.NotNil(t, input, "Expected non-nil input model")
	assert.Equal(t, "Specify the mnemonic for the challenger", input.GetQuestion())
	assert.Equal(t, "Enter the mnemonic", input.TextInput.Placeholder)
}

func TestSystemKeyChallengerMnemonicInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyChallengerMnemonicInput(ctx)

	cmd := input.Init()
	assert.Nil(t, cmd, "Init should return nil, as it has no side-effect commands")
}

func TestSystemKeyChallengerMnemonicInput_Update(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyChallengerMnemonicInput(ctx)

	// Test valid mnemonic input
	validMnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validMnemonic)}
	nextModel, _ := input.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, _ := nextModel.Update(enterPress)

	model := finalModel.(*ExistingGasStationChecker)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.IsType(t, &ExistingGasStationChecker{}, finalModel)
	assert.Equal(t, validMnemonic, state.systemKeyChallengerMnemonic)
	assert.Contains(t, state.weave.PreviousResponse, styles.RenderPreviousResponse(
		styles.DotsSeparator, input.GetQuestion(), []string{"challenger"}, styles.HiddenMnemonicText))

	// Test invalid mnemonic input
	ctx = weavecontext.NewAppContext(*NewLaunchState())
	input = NewSystemKeyChallengerMnemonicInput(ctx)
	invalidMnemonic := "invalid mnemonic phrase"
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(invalidMnemonic)}
	nextModel, _ = input.Update(keyMsg)
	finalModel, _ = input.Update(enterPress)

	checkerModel := finalModel.(*SystemKeyChallengerMnemonicInput)
	state = weavecontext.GetCurrentState[LaunchState](checkerModel.Ctx)
	assert.NotEqual(t, invalidMnemonic, state.systemKeyChallengerMnemonic)
	assert.NotContains(t, state.weave.PreviousResponse, styles.RenderPreviousResponse(
		styles.DotsSeparator, input.GetQuestion(), []string{"challenger"}, styles.HiddenMnemonicText))
}

func TestSystemKeyChallengerMnemonicInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewSystemKeyChallengerMnemonicInput(ctx)

	view := input.View()
	assert.Contains(t, view, input.GetQuestion(), "Expected question prompt in the view")
	assert.Contains(t, view, "Enter the mnemonic", "Expected mnemonic prompt in the view")
}

func TestNewExistingGasStationChecker(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	checker := NewExistingGasStationChecker(ctx)

	assert.NotNil(t, checker, "Expected non-nil ExistingGasStationChecker")
	assert.Contains(t, checker.Loading.Text, "Checking for gas station account...", "Expected loading message to be set")
}

func TestExistingGasStationChecker_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	checker := NewExistingGasStationChecker(ctx)

	cmd := checker.Init()
	assert.NotNil(t, cmd, "Expected non-nil command for loading initialization")
}

func TestWaitExistingGasStationChecker_FirstTimeSetup(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	cmd := waitExistingGasStationChecker(ctx)
	msg := cmd()

	state := weavecontext.GetCurrentState[LaunchState](ctx)
	assert.IsType(t, ui.EndLoading{}, msg, "Expected to receive EndLoading message")
	assert.False(t, state.gasStationExist, "Expected gasStationExist to be false in first-time setup")
}

func TestWaitExistingGasStationChecker_ExistingSetup(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	cmd := waitExistingGasStationChecker(ctx)
	msg := cmd()

	state := weavecontext.GetCurrentState[LaunchState](ctx)
	assert.IsType(t, ui.EndLoading{}, msg, "Expected to receive EndLoading message")
	assert.False(t, state.gasStationExist, "Expected gasStationExist to be true in existing setup")
}

func TestWaitExistingGasStationChecker_NonExistingSetup(t *testing.T) {
	InitializeViperForTest(t)
	key, _ := config.RecoverGasStationKey("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon")
	viper.Set("common.gas_station", key)
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	cmd := waitExistingGasStationChecker(ctx)
	msg := cmd()

	endLoading := msg.(ui.EndLoading)
	state := weavecontext.GetCurrentState[LaunchState](endLoading.Ctx)
	assert.IsType(t, ui.EndLoading{}, msg, "Expected to receive EndLoading message")
	assert.True(t, state.gasStationExist, "Expected gasStationExist to be true in existing setup")
}

func TestExistingGasStationChecker_Update_LoadingIncomplete(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	checker := NewExistingGasStationChecker(ctx)
	mockMsg := ui.TickMsg{}

	updatedModel, cmd := checker.Update(mockMsg)

	assert.Equal(t, checker, updatedModel, "Expected to return the same model while loading is not complete")
	assert.NotNil(t, cmd, "Expected a command during loading update")
}

func TestExistingGasStationChecker_Update_LoadingComplete_NoGasStation(t *testing.T) {
	state := &LaunchState{gasStationExist: false}
	ctx := weavecontext.NewAppContext(*state)

	checker := NewExistingGasStationChecker(ctx)
	checker.Loading.EndContext = ctx
	checker.Loading.Completing = true

	updatedModel, cmd := checker.Update(&tea.KeyMsg{})

	assert.IsType(t, &GasStationMnemonicInput{}, updatedModel, "Expected to transition to GasStationMnemonicInput when no gas station exists")
	assert.Nil(t, cmd, "Expected no additional command after transition")
}

func TestExistingGasStationChecker_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	checker := NewExistingGasStationChecker(ctx)

	view := checker.View()

	assert.Contains(t, view, "Checking for gas station account...", "Expected the view to contain the loading message")
}

func TestNewGasStationMnemonicInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewGasStationMnemonicInput(ctx)

	assert.NotNil(t, input)
	assert.Contains(t, input.GetQuestion(), "Please set up a gas station account")
	assert.NotNil(t, input.TextInput)
	assert.Contains(t, input.question, "Please set up a gas station account", "Expected question prompt in the input")
}

func TestGasStationMnemonicInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewGasStationMnemonicInput(ctx)

	cmd := input.Init()
	assert.Nil(t, cmd, "Init should return nil, as it has no side-effect commands")
}

func TestGasStationMnemonicInput_Update_Invalid(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewGasStationMnemonicInput(ctx)

	invalidMnemonic := ""
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(invalidMnemonic)}
	nextModel, _ := input.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, _ := nextModel.Update(enterPress)

	assert.IsType(t, input, finalModel)
}

func TestGasStationMnemonicInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	input := NewGasStationMnemonicInput(ctx)

	view := input.View()
	assert.Contains(t, view, "Please set up a gas station account", "Expected question prompt in the view")
	assert.Contains(t, view, "Enter the mnemonic", "Expected placeholder prompt in the view")
}

func TestNewSystemKeyL1BridgeExecutorBalanceInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	balanceInput := NewSystemKeyL1BridgeExecutorBalanceInput(ctx)

	assert.NotNil(t, balanceInput)
	assert.Equal(t, "Specify the amount to fund the bridge executor on L1 (uinit)", balanceInput.GetQuestion())
}

func TestSystemKeyL1BridgeExecutorBalanceInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	balanceInput := NewSystemKeyL1BridgeExecutorBalanceInput(ctx)

	cmd := balanceInput.Init()
	assert.Nil(t, cmd)
}

func TestSystemKeyL1BridgeExecutorBalanceInput_Update_Valid(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	balanceInput := NewSystemKeyL1BridgeExecutorBalanceInput(ctx)

	validInput := "2000"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validInput)}
	nextModel, _ := balanceInput.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, _ := nextModel.Update(enterPress)

	model := finalModel.(*SystemKeyL1OutputSubmitterBalanceInput)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.IsType(t, &SystemKeyL1OutputSubmitterBalanceInput{}, finalModel)
	assert.Equal(t, validInput, state.systemKeyL1BridgeExecutorBalance)
	assert.Contains(t, state.weave.PreviousResponse, styles.RenderPreviousResponse(
		styles.DotsSeparator, balanceInput.GetQuestion(), []string{"bridge executor", "L1"}, validInput))
}

func TestSystemKeyL1BridgeExecutorBalanceInput_Update_Invalid(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	balanceInput := NewSystemKeyL1BridgeExecutorBalanceInput(ctx)

	invalidInput := "xyz"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(invalidInput)}
	nextModel, _ := balanceInput.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, cmd := nextModel.Update(enterPress)

	model := finalModel.(*SystemKeyL1BridgeExecutorBalanceInput)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.Equal(t, balanceInput, finalModel)
	assert.Nil(t, cmd)
	assert.NotEqual(t, invalidInput, state.systemKeyL1BridgeExecutorBalance)
}

func TestSystemKeyL1BridgeExecutorBalanceInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	balanceInput := NewSystemKeyL1BridgeExecutorBalanceInput(ctx)

	view := balanceInput.View()
	assert.Contains(t, view, "Specify the amount to fund the bridge executor on L1 (uinit)", "Expected question prompt in the view")
	assert.Contains(t, view, "Enter a positive amount", "Expected placeholder in the view")
}

func TestNewSystemKeyL1OutputSubmitterBalanceInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	outputSubmitterInput := NewSystemKeyL1OutputSubmitterBalanceInput(ctx)

	assert.NotNil(t, outputSubmitterInput)
	assert.Equal(t, "Specify the amount to fund the output submitter on L1 (uinit)", outputSubmitterInput.GetQuestion())
}

func TestSystemKeyL1OutputSubmitterBalanceInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	outputSubmitterInput := NewSystemKeyL1OutputSubmitterBalanceInput(ctx)

	cmd := outputSubmitterInput.Init()
	assert.Nil(t, cmd)
}

func TestSystemKeyL1OutputSubmitterBalanceInput_Update_Valid(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	outputSubmitterInput := NewSystemKeyL1OutputSubmitterBalanceInput(ctx)

	validInput := "3000"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validInput)}
	nextModel, _ := outputSubmitterInput.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, _ := nextModel.Update(enterPress)

	model := finalModel.(*SystemKeyL1BatchSubmitterBalanceInput)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.IsType(t, &SystemKeyL1BatchSubmitterBalanceInput{}, finalModel)
	assert.Equal(t, validInput, state.systemKeyL1OutputSubmitterBalance)
	assert.Contains(t, state.weave.PreviousResponse, styles.RenderPreviousResponse(
		styles.DotsSeparator, outputSubmitterInput.GetQuestion(), []string{"output submitter", "L1"}, validInput))
}

func TestSystemKeyL1OutputSubmitterBalanceInput_Update_Invalid(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	outputSubmitterInput := NewSystemKeyL1OutputSubmitterBalanceInput(ctx)

	invalidInput := "abc"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(invalidInput)}
	nextModel, _ := outputSubmitterInput.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, cmd := nextModel.Update(enterPress)

	model := finalModel.(*SystemKeyL1OutputSubmitterBalanceInput)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.Equal(t, outputSubmitterInput, finalModel)
	assert.Nil(t, cmd)
	assert.NotEqual(t, invalidInput, state.systemKeyL1OutputSubmitterBalance)
}

func TestSystemKeyL1OutputSubmitterBalanceInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	outputSubmitterInput := NewSystemKeyL1OutputSubmitterBalanceInput(ctx)

	view := outputSubmitterInput.View()
	assert.Contains(t, view, "Specify the amount to fund the output submitter on L1 (uinit)", "Expected question prompt in the view")
	assert.Contains(t, view, "Enter a positive amount", "Expected placeholder in the view")
}

func TestNewSystemKeyL1BatchSubmitterBalanceInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	batchSubmitterInput := NewSystemKeyL1BatchSubmitterBalanceInput(ctx)

	assert.NotNil(t, batchSubmitterInput)
	assert.Equal(t, "Specify the amount to fund the batch submitter on L1 (uinit)", batchSubmitterInput.GetQuestion())
}

func TestSystemKeyL1BatchSubmitterBalanceInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	batchSubmitterInput := NewSystemKeyL1BatchSubmitterBalanceInput(ctx)

	cmd := batchSubmitterInput.Init()
	assert.Nil(t, cmd)
}

func TestSystemKeyL1BatchSubmitterBalanceInput_Update_Valid(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	batchSubmitterInput := NewSystemKeyL1BatchSubmitterBalanceInput(ctx)

	validInput := "5000"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validInput)}
	nextModel, _ := batchSubmitterInput.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, _ := nextModel.Update(enterPress)

	model := finalModel.(*SystemKeyL1ChallengerBalanceInput)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.IsType(t, &SystemKeyL1ChallengerBalanceInput{}, finalModel)
	assert.Equal(t, validInput, state.systemKeyL1BatchSubmitterBalance)
	assert.Contains(t, state.weave.PreviousResponse, styles.RenderPreviousResponse(
		styles.DotsSeparator, batchSubmitterInput.GetQuestion(), []string{"batch submitter", "L1"}, validInput))
}

func TestSystemKeyL1BatchSubmitterBalanceInput_Update_Invalid(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	batchSubmitterInput := NewSystemKeyL1BatchSubmitterBalanceInput(ctx)

	invalidInput := "xyz"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(invalidInput)}
	nextModel, _ := batchSubmitterInput.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, cmd := nextModel.Update(enterPress)

	model := finalModel.(*SystemKeyL1BatchSubmitterBalanceInput)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.Equal(t, batchSubmitterInput, finalModel)
	assert.Nil(t, cmd)
	assert.NotEqual(t, invalidInput, state.systemKeyL1BatchSubmitterBalance)
}

func TestSystemKeyL1BatchSubmitterBalanceInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	batchSubmitterInput := NewSystemKeyL1BatchSubmitterBalanceInput(ctx)

	view := batchSubmitterInput.View()
	assert.Contains(t, view, "Specify the amount to fund the batch submitter on L1 (uinit)", "Expected question prompt in the view")
	assert.Contains(t, view, "Enter a positive amount", "Expected placeholder in the view")
}

func TestNewSystemKeyL1ChallengerBalanceInput(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	challengerInput := NewSystemKeyL1ChallengerBalanceInput(ctx)

	assert.NotNil(t, challengerInput)
	assert.Equal(t, "Specify the amount to fund the challenger on L1 (uinit)", challengerInput.GetQuestion())
}

func TestSystemKeyL1ChallengerBalanceInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	challengerInput := NewSystemKeyL1ChallengerBalanceInput(ctx)

	cmd := challengerInput.Init()
	assert.Nil(t, cmd)
}

func TestSystemKeyL1ChallengerBalanceInput_Update_Invalid(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	challengerInput := NewSystemKeyL1ChallengerBalanceInput(ctx)

	invalidInput := "abc"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(invalidInput)}
	nextModel, _ := challengerInput.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, cmd := nextModel.Update(enterPress)

	model := finalModel.(*SystemKeyL1ChallengerBalanceInput)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.Equal(t, challengerInput, finalModel)
	assert.Nil(t, cmd)
	assert.NotEqual(t, invalidInput, state.systemKeyL1ChallengerBalance)
}

func TestSystemKeyL1ChallengerBalanceInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	challengerInput := NewSystemKeyL1ChallengerBalanceInput(ctx)

	view := challengerInput.View()
	assert.Contains(t, view, "Specify the amount to fund the challenger on L1 (uinit)", "Expected question prompt in the view")
	assert.Contains(t, view, "Enter a positive amount", "Expected placeholder in the view")
}

func TestAddGenesisAccountsSelect_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	selectInput := NewAddGenesisAccountsSelect(false, ctx)

	cmd := selectInput.Init()
	assert.Nil(t, cmd)
}

func TestAddGenesisAccountsSelect_Update_Yes(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	selectInput := NewAddGenesisAccountsSelect(false, ctx)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, _ := selectInput.Update(enterPress)

	model := finalModel.(*GenesisAccountsAddressInput)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.IsType(t, &GenesisAccountsAddressInput{}, finalModel)
	assert.Contains(t, state.weave.PreviousResponse[0], "Would you like to add genesis accounts? > Yes")
}

func TestAddGenesisAccountsSelect_Update_No(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	selectInput := NewAddGenesisAccountsSelect(false, ctx)

	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	nextModel, _ := selectInput.Update(downMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, _ := nextModel.Update(enterPress)

	model := finalModel.(*DownloadMinitiaBinaryLoading)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.IsType(t, &DownloadMinitiaBinaryLoading{}, finalModel)
	assert.Contains(t, state.weave.PreviousResponse[0], "Would you like to add genesis accounts? > No")
}

func TestAddGenesisAccountsSelect_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	selectInput := NewAddGenesisAccountsSelect(false, ctx)

	view := selectInput.View()
	assert.Contains(t, view, "Would you like to add genesis accounts?", "Expected question prompt in the view")
	assert.Contains(t, view, "> Yes", "Expected choice prompt in the view")
}

func TestGenesisAccountsAddressInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	addressInput := NewGenesisAccountsAddressInput(ctx)

	cmd := addressInput.Init()
	assert.Nil(t, cmd)
}

func TestGenesisAccountsAddressInput_Update_Valid(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	addressInput := NewGenesisAccountsAddressInput(ctx)

	validInput := "init1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpqr5e3d"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validInput)}
	nextModel, _ := addressInput.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, _ := nextModel.Update(enterPress)

	assert.IsType(t, &GenesisAccountsBalanceInput{}, finalModel)
}

func TestGenesisAccountsAddressInput_Update_Invalid(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	addressInput := NewGenesisAccountsAddressInput(ctx)

	invalidInput := "invalidAddress"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(invalidInput)}
	nextModel, _ := addressInput.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, cmd := nextModel.Update(enterPress)

	model := finalModel.(*GenesisAccountsAddressInput)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.Equal(t, addressInput, finalModel)
	assert.Nil(t, cmd)
	assert.NotContains(t, state.weave.PreviousResponse, invalidInput)
}

func TestGenesisAccountsAddressInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	addressInput := NewGenesisAccountsAddressInput(ctx)

	view := addressInput.View()
	assert.Contains(t, view, "Specify a genesis account address", "Expected question prompt in the view")
	assert.Contains(t, view, "Enter a valid address", "Expected placeholder in the view")
}

func TestGenesisAccountsBalanceInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	balanceInput := NewGenesisAccountsBalanceInput("init1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpqr5e3d", ctx)

	cmd := balanceInput.Init()
	assert.Nil(t, cmd)
}

func TestGenesisAccountsBalanceInput_Update_Valid(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	balanceInput := NewGenesisAccountsBalanceInput("init1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpqr5e3d", ctx)

	validBalance := "1000"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validBalance)}
	nextModel, _ := balanceInput.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, _ := nextModel.Update(enterPress)

	model := finalModel.(*AddGenesisAccountsSelect)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.IsType(t, &AddGenesisAccountsSelect{}, finalModel)
	assert.Equal(t, 1, len(state.genesisAccounts))
	assert.Equal(t, "init1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpqr5e3d", state.genesisAccounts[0].Address)
	assert.Equal(t, "1000"+state.gasDenom, state.genesisAccounts[0].Coins)
	assert.Contains(t, state.weave.PreviousResponse[0], validBalance)
}

func TestGenesisAccountsBalanceInput_Update_Invalid(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	balanceInput := NewGenesisAccountsBalanceInput("init1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpqr5e3d", ctx)

	invalidBalance := "notANumber"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(invalidBalance)}
	nextModel, _ := balanceInput.Update(keyMsg)

	enterPress := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, cmd := nextModel.Update(enterPress)

	model := finalModel.(*GenesisAccountsBalanceInput)
	state := weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.Equal(t, balanceInput, finalModel)
	assert.Nil(t, cmd)
	assert.Equal(t, 0, len(state.genesisAccounts))
}

func TestGenesisAccountsBalanceInput_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	balanceInput := NewGenesisAccountsBalanceInput("init1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpqr5e3d", ctx)

	view := balanceInput.View()
	assert.Contains(t, view, "Specify the genesis balance for init1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpqr5e3d", "Expected question prompt in the view")
	assert.Contains(t, view, "Enter a positive amount", "Expected placeholder in the view")
}

func TestAddGenesisAccountsSelect_Update_RecurringWithAccounts(t *testing.T) {
	state := LaunchState{
		genesisAccounts: []types.GenesisAccount{
			{Address: "address1", Coins: "100token"},
			{Address: "address2", Coins: "200token"},
		},
	}
	ctx := weavecontext.NewAppContext(state)

	model := NewAddGenesisAccountsSelect(true, ctx)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(msg)

	finalModel := updatedModel.(*GenesisAccountsAddressInput)
	state = weavecontext.GetCurrentState[LaunchState](finalModel.Ctx)
	assert.IsType(t, &GenesisAccountsAddressInput{}, updatedModel)
	assert.Nil(t, cmd)
	assert.Len(t, state.weave.PreviousResponse, 1)
	assert.Equal(t, "Would you like to add another genesis account?", model.recurringQuestion)
	assert.Contains(t, state.weave.PreviousResponse[0], "Would you like to add another genesis account?")
	assert.Contains(t, state.weave.PreviousResponse[0], "Yes")
}

func TestAddGenesisAccountsSelect_Update_NoRecurringWithAccounts(t *testing.T) {
	state := LaunchState{
		genesisAccounts: []types.GenesisAccount{
			{Address: "address1", Coins: "100token"},
			{Address: "address2", Coins: "200token"},
		},
	}

	ctx := weavecontext.NewAppContext(state)

	model := NewAddGenesisAccountsSelect(true, ctx)

	msg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, cmd := model.Update(msg)

	msg = tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd = model.Update(msg)

	finalModel := updatedModel.(*DownloadMinitiaBinaryLoading)
	state = weavecontext.GetCurrentState[LaunchState](finalModel.Ctx)
	assert.IsType(t, &DownloadMinitiaBinaryLoading{}, updatedModel)
	assert.NotNil(t, cmd)
	assert.Len(t, state.weave.PreviousResponse, 2)
	assert.Contains(t, state.weave.PreviousResponse[0], "Would you like to add genesis accounts?")
	assert.Contains(t, state.weave.PreviousResponse[1], "List of extra Genesis Accounts")
	assert.Contains(t, state.weave.PreviousResponse[1], "address1")
	assert.Contains(t, state.weave.PreviousResponse[1], "100token")
	assert.Contains(t, state.weave.PreviousResponse[1], "address2")
	assert.Contains(t, state.weave.PreviousResponse[1], "200token")
}

func TestNewDownloadMinitiaBinaryLoading(t *testing.T) {
	state := LaunchState{
		vmType:           "TestVM",
		minitiadVersion:  "v1.0.0",
		minitiadEndpoint: "https://example.com/minitia.tar.gz",
	}
	ctx := weavecontext.NewAppContext(state)

	loadingModel := NewDownloadMinitiaBinaryLoading(ctx)

	nextState := weavecontext.GetCurrentState[LaunchState](loadingModel.Ctx)
	assert.NotNil(t, loadingModel)
	assert.Equal(t, state, nextState)
	assert.Contains(t, loadingModel.Loading.Text, "Downloading Minitestvm binary <v1.0.0>")
}

func TestDownloadMinitiaBinaryLoading_Init(t *testing.T) {
	state := LaunchState{
		vmType:           "TestVM",
		minitiadVersion:  "v1.0.0",
		minitiadEndpoint: "https://example.com/minitia.tar.gz",
	}
	ctx := weavecontext.NewAppContext(state)

	loadingModel := NewDownloadMinitiaBinaryLoading(ctx)

	cmd := loadingModel.Init()
	assert.NotNil(t, cmd)
}

func TestDownloadMinitiaBinaryLoading_Update_Complete(t *testing.T) {
	state := LaunchState{
		vmType:           "TestVM",
		minitiadVersion:  "v1.0.0",
		minitiadEndpoint: "https://example.com/minitia.tar.gz",
	}
	ctx := weavecontext.NewAppContext(state)

	loadingModel := NewDownloadMinitiaBinaryLoading(ctx)
	loadingModel.Loading.Completing = true
	loadingModel.Loading.EndContext = ctx

	finalModel, cmd := loadingModel.Update(&tea.KeyMsg{})

	assert.NotNil(t, cmd)
	assert.IsType(t, &GenerateOrRecoverSystemKeysLoading{}, finalModel)
}

func TestDownloadMinitiaBinaryLoading_Update_DownloadSuccess(t *testing.T) {
	state := LaunchState{
		vmType:              "TestVM",
		minitiadVersion:     "v1.0.0",
		downloadedNewBinary: true,
	}
	ctx := weavecontext.NewAppContext(state)

	loadingModel := NewDownloadMinitiaBinaryLoading(ctx)

	stillLoadingMsg := ui.TickMsg{}
	nextModel, _ := loadingModel.Update(stillLoadingMsg)

	assert.IsType(t, &DownloadMinitiaBinaryLoading{}, nextModel)

	model := nextModel.(*DownloadMinitiaBinaryLoading)
	model.Loading.Completing = true
	model.Loading.EndContext = model.Ctx
	finalModel, _ := nextModel.Update(&tea.KeyMsg{})

	assert.IsType(t, &GenerateOrRecoverSystemKeysLoading{}, finalModel)
}

func TestDownloadMinitiaBinaryLoading_View(t *testing.T) {
	state := LaunchState{
		vmType:           "TestVM",
		minitiadVersion:  "v1.0.0",
		minitiadEndpoint: "https://example.com/minitia.tar.gz",
	}
	ctx := weavecontext.NewAppContext(state)

	loadingModel := NewDownloadMinitiaBinaryLoading(ctx)
	view := loadingModel.View()

	loadingModel.Loading.Completing = true
	loadingModel.Loading.EndContext = ctx

	assert.Contains(t, view, state.weave.Render())
	assert.Contains(t, view, loadingModel.Loading.View())
}

func TestGetCelestiaBinaryURL(t *testing.T) {
	tests := []struct {
		os     string
		arch   string
		expect string
	}{
		{"darwin", "amd64", "https://github.com/celestiaorg/celestia-app/releases/download/v1.0.0/celestia-app_Darwin_x86_64.tar.gz"},
		{"linux", "arm64", "https://github.com/celestiaorg/celestia-app/releases/download/v1.0.0/celestia-app_Linux_arm64.tar.gz"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.os, tt.arch), func(t *testing.T) {
			url, _ := getCelestiaBinaryURL("1.0.0", tt.os, tt.arch)
			assert.Equal(t, tt.expect, url)
		})
	}
}

func TestNewGenerateOrRecoverSystemKeysLoading_Generate(t *testing.T) {
	state := LaunchState{
		generateKeys: true,
	}
	ctx := weavecontext.NewAppContext(state)

	loadingModel := NewGenerateOrRecoverSystemKeysLoading(ctx)

	assert.NotNil(t, loadingModel)
	assert.Contains(t, loadingModel.Loading.Text, "Generating new system keys...")
}

func TestNewGenerateOrRecoverSystemKeysLoading_Recover(t *testing.T) {
	state := LaunchState{
		generateKeys: false,
	}
	ctx := weavecontext.NewAppContext(state)

	loadingModel := NewGenerateOrRecoverSystemKeysLoading(ctx)

	assert.NotNil(t, loadingModel)
	assert.Contains(t, loadingModel.Loading.Text, "Recovering system keys...")
}

func TestGenerateOrRecoverSystemKeysLoading_Init(t *testing.T) {
	state := LaunchState{
		generateKeys: true,
	}
	ctx := weavecontext.NewAppContext(state)

	loadingModel := NewGenerateOrRecoverSystemKeysLoading(ctx)

	cmd := loadingModel.Init()
	assert.NotNil(t, cmd)
}

func TestGenerateOrRecoverSystemKeysLoading_Update_Generate(t *testing.T) {
	state := LaunchState{
		generateKeys: true,
		binaryPath:   "test/path",
	}
	ctx := weavecontext.NewAppContext(state)

	loadingModel := NewGenerateOrRecoverSystemKeysLoading(ctx)
	loadingModel.Loading.Completing = true
	loadingModel.Loading.EndContext = ctx
	finalModel, cmd := loadingModel.Update(&tea.KeyMsg{})

	model := finalModel.(*SystemKeysMnemonicDisplayInput)
	state = weavecontext.GetCurrentState[LaunchState](model.Ctx)
	assert.Nil(t, cmd)
	assert.IsType(t, &SystemKeysMnemonicDisplayInput{}, finalModel)
	assert.Contains(t, state.weave.PreviousResponse[0], "System keys have been successfully generated.")
}

func TestGenerateOrRecoverSystemKeysLoading_View(t *testing.T) {
	state := LaunchState{
		generateKeys: true,
	}
	ctx := weavecontext.NewAppContext(state)

	loadingModel := NewGenerateOrRecoverSystemKeysLoading(ctx)
	view := loadingModel.View()

	assert.Contains(t, view, state.weave.Render())
	assert.Contains(t, view, loadingModel.Loading.View())
}

func TestNewSystemKeysMnemonicDisplayInput(t *testing.T) {
	state := LaunchState{
		systemKeyOperatorAddress:         "operator_address",
		systemKeyOperatorMnemonic:        "operator_mnemonic",
		systemKeyBridgeExecutorAddress:   "bridge_executor_address",
		systemKeyBridgeExecutorMnemonic:  "bridge_executor_mnemonic",
		systemKeyOutputSubmitterAddress:  "output_submitter_address",
		systemKeyOutputSubmitterMnemonic: "output_submitter_mnemonic",
		systemKeyBatchSubmitterAddress:   "batch_submitter_address",
		systemKeyBatchSubmitterMnemonic:  "batch_submitter_mnemonic",
		systemKeyChallengerAddress:       "challenger_address",
		systemKeyChallengerMnemonic:      "challenger_mnemonic",
	}
	ctx := weavecontext.NewAppContext(state)

	inputModel := NewSystemKeysMnemonicDisplayInput(ctx)

	checkState := weavecontext.GetCurrentState[LaunchState](inputModel.Ctx)
	assert.NotNil(t, inputModel)
	assert.Equal(t, state, checkState)
	assert.Equal(t, "Type `continue` to proceed.", inputModel.question)
	assert.Contains(t, inputModel.TextInput.Placeholder, "Type `continue` to continue, Ctrl+C to quit.")
}

func TestSystemKeysMnemonicDisplayInput_GetQuestion(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	inputModel := NewSystemKeysMnemonicDisplayInput(ctx)

	question := inputModel.GetQuestion()

	assert.Equal(t, "Type `continue` to proceed.", question)
}

func TestSystemKeysMnemonicDisplayInput_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	inputModel := NewSystemKeysMnemonicDisplayInput(ctx)

	cmd := inputModel.Init()

	assert.Nil(t, cmd)
}

func TestSystemKeysMnemonicDisplayInput_Update_NotDone(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	inputModel := NewSystemKeysMnemonicDisplayInput(ctx)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i dont wanna continue")}
	nextModel, _ := inputModel.Update(msg)

	msg = tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, cmd := nextModel.Update(msg)

	assert.Equal(t, inputModel, finalModel)
	assert.Nil(t, cmd)
}

func TestSystemKeysMnemonicDisplayInput_View(t *testing.T) {
	state := LaunchState{
		systemKeyOperatorAddress:         "operator_address",
		systemKeyOperatorMnemonic:        "operator_mnemonic",
		systemKeyBridgeExecutorAddress:   "bridge_executor_address",
		systemKeyBridgeExecutorMnemonic:  "bridge_executor_mnemonic",
		systemKeyOutputSubmitterAddress:  "output_submitter_address",
		systemKeyOutputSubmitterMnemonic: "output_submitter_mnemonic",
		systemKeyBatchSubmitterAddress:   "batch_submitter_address",
		systemKeyBatchSubmitterMnemonic:  "batch_submitter_mnemonic",
		systemKeyChallengerAddress:       "challenger_address",
		systemKeyChallengerMnemonic:      "challenger_mnemonic",
	}
	ctx := weavecontext.NewAppContext(state)

	inputModel := NewSystemKeysMnemonicDisplayInput(ctx)

	view := inputModel.View()

	assert.Contains(t, view, "Important")
	assert.Contains(t, view, "Note that these mnemonic phrases along with other configuration details will be stored")
	assert.Contains(t, view, "Key Name: Operator")
	assert.Contains(t, view, "continue")
	assert.Contains(t, view, inputModel.TextInput.View())
}

func TestNewFundGasStationBroadcastLoading(t *testing.T) {
	state := LaunchState{
		systemKeyOperatorAddress:          "operator_address",
		systemKeyBridgeExecutorAddress:    "bridge_executor_address",
		systemKeyOutputSubmitterAddress:   "output_submitter_address",
		systemKeyBatchSubmitterAddress:    "batch_submitter_address",
		systemKeyChallengerAddress:        "challenger_address",
		systemKeyL1BridgeExecutorBalance:  "2000",
		systemKeyL1OutputSubmitterBalance: "3000",
		systemKeyL1BatchSubmitterBalance:  "4000",
		systemKeyL1ChallengerBalance:      "5000",
		binaryPath:                        "/path/to/binary",
		l1RPC:                             "http://localhost:8545",
		l1ChainId:                         "1",
	}
	ctx := weavecontext.NewAppContext(state)

	loadingModel := NewFundGasStationBroadcastLoading(ctx)

	checkState := weavecontext.GetCurrentState[LaunchState](loadingModel.Ctx)
	assert.NotNil(t, loadingModel)
	assert.Equal(t, state, checkState)
	assert.Equal(t, "Broadcasting transactions...", loadingModel.Loading.Text)
}

func TestFundGasStationBroadcastLoading_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	loadingModel := NewFundGasStationBroadcastLoading(ctx)

	cmd := loadingModel.Init()

	assert.NotNil(t, cmd)
}

func TestFundGasStationBroadcastLoading_Update_Complete(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	loadingModel := NewFundGasStationBroadcastLoading(ctx)
	loadingModel.Loading.Completing = true
	loadingModel.Loading.EndContext = ctx
	finalModel, cmd := loadingModel.Update(&tea.KeyMsg{})

	assert.IsType(t, &LaunchingNewMinitiaLoading{}, finalModel)
	assert.NotNil(t, cmd)
}

func TestFundGasStationBroadcastLoading_Update_Incomplete(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	loadingModel := NewFundGasStationBroadcastLoading(ctx)

	msg := ui.TickMsg{}
	finalModel, cmd := loadingModel.Update(msg)

	assert.Equal(t, loadingModel, finalModel)
	assert.NotNil(t, cmd)
}

func TestFundGasStationBroadcastLoading_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	loadingModel := NewFundGasStationBroadcastLoading(ctx)

	view := loadingModel.View()

	state := weavecontext.GetCurrentState[LaunchState](loadingModel.Ctx)
	assert.Contains(t, view, "Broadcasting transactions...")
	assert.Contains(t, view, state.weave.Render())
}

func TestNewTerminalState(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	terminalState := NewTerminalState(ctx)

	assert.NotNil(t, terminalState)
}

func TestTerminalState_Init(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	terminalState := NewTerminalState(ctx)

	cmd := terminalState.Init()
	assert.Nil(t, cmd, "Init should return nil, as it has no side-effect commands")
}

func TestTerminalState_Update(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	terminalState := NewTerminalState(ctx)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	finalModel, cmd := terminalState.Update(msg)

	assert.Equal(t, terminalState, finalModel, "Expected the same terminal state to be returned")
	assert.Nil(t, cmd, "Update should return nil command since there are no state changes")
}

func TestTerminalState_View(t *testing.T) {
	ctx := weavecontext.NewAppContext(*NewLaunchState())

	terminalState := NewTerminalState(ctx)

	view := terminalState.View()
	state := weavecontext.GetCurrentState[LaunchState](terminalState.Ctx)
	assert.Contains(t, view, state.weave.Render(), "Expected view to contain the rendered output from the weave")
}
