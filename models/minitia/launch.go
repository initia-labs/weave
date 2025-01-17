package minitia

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/client"
	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/config"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/cosmosutils"
	"github.com/initia-labs/weave/crypto"
	"github.com/initia-labs/weave/io"
	"github.com/initia-labs/weave/registry"
	"github.com/initia-labs/weave/service"
	"github.com/initia-labs/weave/styles"
	"github.com/initia-labs/weave/tooltip"
	"github.com/initia-labs/weave/types"
	"github.com/initia-labs/weave/ui"
)

type ExistingMinitiaChecker struct {
	weavecontext.BaseModel
	ui.Loading
}

func NewExistingMinitiaChecker(ctx context.Context) *ExistingMinitiaChecker {
	return &ExistingMinitiaChecker{
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		Loading:   ui.NewLoading("Checking for an existing rollup app...", waitExistingMinitiaChecker(ctx)),
	}
}

func (m *ExistingMinitiaChecker) Init() tea.Cmd {
	return m.Loading.Init()
}

func waitExistingMinitiaChecker(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		state := weavecontext.GetCurrentState[LaunchState](ctx)

		minitiaPath, err := weavecontext.GetMinitiaHome(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: err}
		}
		time.Sleep(1500 * time.Millisecond)

		if !io.FileOrFolderExists(minitiaPath) {
			state.existingMinitiaApp = false
			return ui.EndLoading{Ctx: weavecontext.SetCurrentState(ctx, state)}
		} else {
			state.existingMinitiaApp = true
			return ui.EndLoading{Ctx: weavecontext.SetCurrentState(ctx, state)}
		}
	}
}

func (m *ExistingMinitiaChecker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		m.Ctx = m.Loading.EndContext
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		if !state.existingMinitiaApp {
			if state.launchFromExistingConfig {
				model := NewDownloadMinitiaBinaryLoading(weavecontext.SetCurrentState(m.Ctx, state))
				return model, model.Init()
			}

			model, err := NewNetworkSelect(weavecontext.SetCurrentState(m.Ctx, state))
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, cmd
		} else {
			return NewDeleteExistingMinitiaInput(weavecontext.SetCurrentState(m.Ctx, state)), cmd
		}
	}
	return m, cmd
}

func (m *ExistingMinitiaChecker) View() string {
	return m.WrapView(styles.Text("🪢 When launching a rollup, after all configurations are set,\nthe rollup process will run for a few blocks to establish the necessary components.\nThis process may take some time.\n\n", styles.Ivory) +
		m.Loading.View())
}

type DeleteExistingMinitiaInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question string
}

func NewDeleteExistingMinitiaInput(ctx context.Context) *DeleteExistingMinitiaInput {
	model := &DeleteExistingMinitiaInput{
		TextInput: ui.NewTextInput(true),
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		question:  "Type `delete` to delete the .minitia folder and proceed with weave rollup launch",
	}
	model.WithPlaceholder("Type `delete` to delete, Ctrl+C to keep the folder and quit this command.")
	model.WithValidatorFn(common.ValidateExactString("delete"))
	return model
}

func (m *DeleteExistingMinitiaInput) GetQuestion() string {
	return m.question
}

func (m *DeleteExistingMinitiaInput) Init() tea.Cmd {
	return nil
}

func (m *DeleteExistingMinitiaInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		minitiaHome, err := weavecontext.GetMinitiaHome(m.Ctx)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		if err := io.DeleteDirectory(minitiaHome); err != nil {
			return m, m.HandlePanic(fmt.Errorf("failed to delete .minitia: %v", err))
		}

		if state.launchFromExistingConfig {
			model := NewDownloadMinitiaBinaryLoading(weavecontext.SetCurrentState(m.Ctx, state))
			return model, model.Init()
		}

		model, err := NewNetworkSelect(weavecontext.SetCurrentState(m.Ctx, state))
		if err != nil {
			return m, m.HandlePanic(err)
		}
		return model, nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *DeleteExistingMinitiaInput) View() string {
	minitiaHome, err := weavecontext.GetMinitiaHome(m.Ctx)
	if err != nil {
		m.HandlePanic(err)
	}
	return m.WrapView(styles.RenderPrompt(fmt.Sprintf("🚨 Existing %s folder detected.\n", minitiaHome), []string{minitiaHome}, styles.Empty) +
		styles.RenderPrompt("To proceed with weave rollup launch, you must confirm the deletion of the .minitia folder.\nIf you do not confirm the deletion, the command will not run, and you will be returned to the homepage.\n\n", []string{".minitia", "weave rollup launch"}, styles.Empty) +
		styles.Text("Please note that once you delete, all configurations, state, keys, and other data will be \n", styles.Yellow) + styles.BoldText("permanently deleted and cannot be reversed.\n", styles.Yellow) +
		styles.RenderPrompt(m.GetQuestion(), []string{"`delete`", ".minitia", "weave rollup launch"}, styles.Question) + m.TextInput.View())
}

type NetworkSelect struct {
	ui.Selector[NetworkSelectOption]
	weavecontext.BaseModel
	question   string
	highlights []string
}

type NetworkSelectOption string

func (n NetworkSelectOption) ToChainType() (registry.ChainType, error) {
	switch n {
	case Mainnet:
		return registry.InitiaL1Mainnet, nil
	case Testnet:
		return registry.InitiaL1Testnet, nil
	default:
		return 0, fmt.Errorf("invalid case for NetworkSelectOption: %v", n)
	}
}

var (
	Testnet NetworkSelectOption = ""
	Mainnet NetworkSelectOption = ""
)

func NewNetworkSelect(ctx context.Context) (*NetworkSelect, error) {
	testnetRegistry, err := registry.GetChainRegistry(registry.InitiaL1Testnet)
	if err != nil {
		return nil, err
	}
	//mainnetRegistry := registry.MustGetChainRegistry(registry.InitiaL1Mainnet)
	Testnet = NetworkSelectOption(fmt.Sprintf("Testnet (%s)", testnetRegistry.GetChainId()))
	//Mainnet = NetworkSelectOption(fmt.Sprintf("Mainnet (%s)", mainnetRegistry.GetChainId()))
	return &NetworkSelect{
		Selector: ui.Selector[NetworkSelectOption]{
			Options: []NetworkSelectOption{
				Testnet,
				//Mainnet,
			},
			CannotBack: true,
		},
		BaseModel:  weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		question:   "Select the Initia L1 network you want to connect your rollup to",
		highlights: []string{"Initia L1 network"},
	}, nil
}

func (m *NetworkSelect) GetQuestion() string {
	return m.question
}

func (m *NetworkSelect) Init() tea.Cmd {
	return nil
}

func (m *NetworkSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		events := analytics.NewEmptyEvent()
		events.Add(analytics.OptionEventKey, string(*selected))
		analytics.TrackEvent(analytics.L1NetworkSelected, events)
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), m.highlights, string(*selected)))
		chainType, err := selected.ToChainType()
		if err != nil {
			return m, m.HandlePanic(err)
		}
		chainRegistry, err := registry.GetChainRegistry(chainType)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		state.l1ChainId = chainRegistry.GetChainId()
		activeRpc, err := chainRegistry.GetActiveRpc()
		if err != nil {
			return m, m.HandlePanic(err)
		}
		state.l1RPC = activeRpc

		return NewVMTypeSelect(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}

	return m, cmd
}

func (m *NetworkSelect) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	return m.WrapView(styles.Text("🪢 When launching a rollup, after all configurations are set,\nthe rollup process will run for a few blocks to establish the necessary components.\nThis process may take some time.\n\n", styles.Ivory) +
		state.weave.Render() + styles.RenderPrompt(
		m.GetQuestion(),
		m.highlights,
		styles.Question,
	) + m.Selector.View())
}

type VMTypeSelect struct {
	ui.Selector[VMTypeSelectOption]
	weavecontext.BaseModel
	question   string
	highlights []string
}

type VMTypeSelectOption string

const (
	Move VMTypeSelectOption = "Move"
	Wasm VMTypeSelectOption = "Wasm"
	EVM  VMTypeSelectOption = "EVM"
)

func ParseVMType(vmType string) (VMTypeSelectOption, error) {
	switch vmType {
	case "move":
		return Move, nil
	case "wasm":
		return Wasm, nil
	case "evm":
		return EVM, nil
	default:
		return "", fmt.Errorf("invalid VM type: %s", vmType)
	}
}

func NewVMTypeSelect(ctx context.Context) *VMTypeSelect {
	tooltips := ui.NewTooltipSlice(
		ui.NewTooltip(
			"Smart Contract VM",
			"We currently supports three VMs - Move, Wasm, and EVM. By selecting a VM, Weave will automatically use the latest version available for that VM, ensuring compatibility and access to recent updates.",
			"", []string{}, []string{}, []string{},
		), 3,
	)
	return &VMTypeSelect{
		Selector: ui.Selector[VMTypeSelectOption]{
			Options: []VMTypeSelectOption{
				Move,
				Wasm,
				EVM,
			},
			Tooltips: &tooltips,
		},
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   "Select the Virtual Machine (VM) for your rollup",
		highlights: []string{"Virtual Machine (VM)"},
	}
}

func (m *VMTypeSelect) GetQuestion() string {
	return m.question
}

func (m *VMTypeSelect) Init() tea.Cmd {
	return nil
}

func (m *VMTypeSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		analytics.TrackEvent(analytics.VmTypeSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, string(*selected)))
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), m.highlights, string(*selected)))
		state.vmType = string(*selected)
		model := NewLatestVersionLoading(weavecontext.SetCurrentState(m.Ctx, state))

		return model, model.Init()
	}

	return m, cmd
}

func (m *VMTypeSelect) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(
		m.GetQuestion(),
		m.highlights,
		styles.Question,
	) + m.Selector.View())
}

type LatestVersionLoading struct {
	weavecontext.BaseModel
	ui.Loading
	vmType string
}

func NewLatestVersionLoading(ctx context.Context) *LatestVersionLoading {
	state := weavecontext.GetCurrentState[LaunchState](ctx)
	vmType := strings.ToLower(state.vmType)
	return &LatestVersionLoading{
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		Loading:   ui.NewLoading(fmt.Sprintf("Fetching the latest release for Mini%s...", vmType), waitLatestVersionLoading(ctx, vmType)),
		vmType:    vmType,
	}
}

func (m *LatestVersionLoading) Init() tea.Cmd {
	return m.Loading.Init()
}

func waitLatestVersionLoading(ctx context.Context, vmType string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1500 * time.Millisecond)

		state := weavecontext.GetCurrentState[LaunchState](ctx)

		version, downloadURL, err := cosmosutils.GetLatestMinitiaVersion(vmType)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: err}
		}
		state.minitiadVersion = version
		state.minitiadEndpoint = downloadURL

		return ui.EndLoading{Ctx: weavecontext.SetCurrentState(ctx, state)}
	}
}

func (m *LatestVersionLoading) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		m.Ctx = m.Loading.EndContext
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		vmText := fmt.Sprintf("Mini%s version", m.vmType)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, fmt.Sprintf("Using the latest %s", vmText), []string{vmText}, state.minitiadVersion))
		return NewChainIdInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	return m, cmd
}

func (m *LatestVersionLoading) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" + m.Loading.View())
}

type ChainIdInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewChainIdInput(ctx context.Context) *ChainIdInput {
	toolTip := tooltip.RollupChainIdTooltip
	model := &ChainIdInput{
		TextInput:  ui.NewTextInput(true),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		question:   "Specify rollup chain ID",
		highlights: []string{"rollup chain ID"},
	}
	model.WithPlaceholder("Enter your chain ID ex. local-rollup-1")
	model.WithValidatorFn(common.ValidateNonEmptyAndLengthString("Chain ID", MaxChainIDLength))
	model.WithTooltip(&toolTip)
	return model
}

func (m *ChainIdInput) GetQuestion() string {
	return m.question
}

func (m *ChainIdInput) Init() tea.Cmd {
	return nil
}

func (m *ChainIdInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.chainId = input.Text
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), m.highlights, input.Text))
		return NewGasDenomInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *ChainIdInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.TextInput.View())
}

type GasDenomInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewGasDenomInput(ctx context.Context) *GasDenomInput {
	toolTip := tooltip.RollupGasDenomTooltip
	model := &GasDenomInput{
		TextInput:  ui.NewTextInput(false),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   "Specify rollup gas denom",
		highlights: []string{"rollup gas denom"},
	}
	model.WithPlaceholder(`Press tab to use "umin"`)
	model.WithDefaultValue("umin")
	model.WithValidatorFn(common.ValidateDenom)
	model.WithTooltip(&toolTip)
	return model
}

func (m *GasDenomInput) GetQuestion() string {
	return m.question
}

func (m *GasDenomInput) Init() tea.Cmd {
	return nil
}

func (m *GasDenomInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.gasDenom = input.Text
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), m.highlights, input.Text))
		return NewMonikerInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *GasDenomInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.TextInput.View())
}

type MonikerInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewMonikerInput(ctx context.Context) *MonikerInput {
	toolTip := tooltip.MonikerTooltip
	model := &MonikerInput{
		TextInput:  ui.NewTextInput(false),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   "Specify rollup node moniker",
		highlights: []string{"rollup node moniker"},
	}
	model.WithPlaceholder(`Press tab to use "operator"`)
	model.WithDefaultValue("operator")
	model.WithValidatorFn(common.ValidateNonEmptyAndLengthString("Moniker", MaxMonikerLength))
	model.WithTooltip(&toolTip)
	return model
}

func (m *MonikerInput) GetQuestion() string {
	return m.question
}

func (m *MonikerInput) Init() tea.Cmd {
	return nil
}

func (m *MonikerInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.moniker = input.Text
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), m.highlights, input.Text))
		return NewOpBridgeSubmissionIntervalInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *MonikerInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.TextInput.View())
}

type OpBridgeSubmissionIntervalInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewOpBridgeSubmissionIntervalInput(ctx context.Context) *OpBridgeSubmissionIntervalInput {
	toolTip := tooltip.OpBridgeSubmissionIntervalTooltip
	model := &OpBridgeSubmissionIntervalInput{
		TextInput:  ui.NewTextInput(false),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   "Specify OP bridge config: Submission Interval (format s, m or h - ex. 30s, 5m, 12h)",
		highlights: []string{"Submission Interval"},
	}
	model.WithPlaceholder("Press tab to use “1m”")
	model.WithDefaultValue("1m")
	model.WithValidatorFn(common.IsValidTimestamp)
	model.WithTooltip(&toolTip)
	return model
}

func (m *OpBridgeSubmissionIntervalInput) GetQuestion() string {
	return m.question
}

func (m *OpBridgeSubmissionIntervalInput) Init() tea.Cmd {
	return nil
}

func (m *OpBridgeSubmissionIntervalInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.opBridgeSubmissionInterval = input.Text
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), m.highlights, input.Text))
		return NewOpBridgeOutputFinalizationPeriodInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *OpBridgeSubmissionIntervalInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.TextInput.View())
}

type OpBridgeOutputFinalizationPeriodInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewOpBridgeOutputFinalizationPeriodInput(ctx context.Context) *OpBridgeOutputFinalizationPeriodInput {
	toolTip := tooltip.OpBridgeOutputFinalizationPeriodTooltip
	model := &OpBridgeOutputFinalizationPeriodInput{
		TextInput:  ui.NewTextInput(false),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   "Specify OP bridge config: Output Finalization Period (format s, m or h - ex. 30s, 5m, 12h)",
		highlights: []string{"Output Finalization Period"},
	}
	model.WithPlaceholder("Press tab to use “168h” (7 days)")
	model.WithDefaultValue("168h")
	model.WithValidatorFn(common.IsValidTimestamp)
	model.WithTooltip(&toolTip)
	return model
}

func (m *OpBridgeOutputFinalizationPeriodInput) GetQuestion() string {
	return m.question
}

func (m *OpBridgeOutputFinalizationPeriodInput) Init() tea.Cmd {
	return nil
}

func (m *OpBridgeOutputFinalizationPeriodInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.opBridgeOutputFinalizationPeriod = input.Text
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), m.highlights, input.Text))
		return NewOpBridgeBatchSubmissionTargetSelect(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *OpBridgeOutputFinalizationPeriodInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.TextInput.View())
}

type OpBridgeBatchSubmissionTargetSelect struct {
	ui.Selector[OpBridgeBatchSubmissionTargetOption]
	weavecontext.BaseModel
	question   string
	highlights []string
}

type OpBridgeBatchSubmissionTargetOption string

const (
	Celestia OpBridgeBatchSubmissionTargetOption = "Celestia"
	Initia   OpBridgeBatchSubmissionTargetOption = "Initia L1"
)

func NewOpBridgeBatchSubmissionTargetSelect(ctx context.Context) *OpBridgeBatchSubmissionTargetSelect {
	tooltips := ui.NewTooltipSlice(tooltip.OpBridgeBatchSubmissionTargetTooltip, 2)
	return &OpBridgeBatchSubmissionTargetSelect{
		Selector: ui.Selector[OpBridgeBatchSubmissionTargetOption]{
			Options: []OpBridgeBatchSubmissionTargetOption{
				Celestia,
				Initia,
			},
			Tooltips: &tooltips,
		},
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   "Where should the rollup blocks and transaction data be submitted?",
		highlights: []string{"rollup"},
	}
}

func (m *OpBridgeBatchSubmissionTargetSelect) GetQuestion() string {
	return m.question
}

func (m *OpBridgeBatchSubmissionTargetSelect) Init() tea.Cmd {
	return nil
}

func (m *OpBridgeBatchSubmissionTargetSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		analytics.TrackEvent(analytics.OpBridgeBatchSubmissionTargetSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, string(*selected)))
		state := weavecontext.PushPageAndGetState[LaunchState](m)
		state.opBridgeBatchSubmissionTarget = common.TransformFirstWordUpperCase(string(*selected))
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), m.highlights, string(*selected)))
		if *selected == Celestia {
			state.batchSubmissionIsCelestia = true
		}
		return NewOracleEnableSelect(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}

	return m, cmd
}

func (m *OpBridgeBatchSubmissionTargetSelect) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(
		m.GetQuestion(),
		m.highlights,
		styles.Question,
	) + m.Selector.View())
}

type OracleEnableSelect struct {
	ui.Selector[OracleEnableOption]
	weavecontext.BaseModel
	question   string
	highlights []string
}

type OracleEnableOption string

const (
	Enable  OracleEnableOption = "Enable"
	Disable OracleEnableOption = "Disable"
)

func NewOracleEnableSelect(ctx context.Context) *OracleEnableSelect {
	tooltips := ui.NewTooltipSlice(tooltip.EnableOracleTooltip, 2)
	return &OracleEnableSelect{
		Selector: ui.Selector[OracleEnableOption]{
			Options: []OracleEnableOption{
				Enable,
				Disable,
			},
			Tooltips: &tooltips,
		},
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   "Would you like to enable oracle price feed from L1?",
		highlights: []string{"oracle price feed", "L1"},
	}
}

func (m *OracleEnableSelect) GetQuestion() string {
	return m.question
}

func (m *OracleEnableSelect) Init() tea.Cmd {
	return nil
}

func (m *OracleEnableSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		analytics.TrackEvent(analytics.EnableOracleSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, *selected == Enable))
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		if *selected == Enable {
			state.enableOracle = true
		} else {
			state.enableOracle = false
		}
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), m.highlights, string(*selected)))
		return NewSystemKeysSelect(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}

	return m, cmd
}

func (m *OracleEnableSelect) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(
		m.GetQuestion(),
		m.highlights,
		styles.Question,
	) + m.Selector.View())
}

type SystemKeysSelect struct {
	ui.Selector[SystemKeysOption]
	weavecontext.BaseModel
	question   string
	highlights []string
}

type SystemKeysOption string

const (
	Generate SystemKeysOption = "Generate new system keys (Will be done at the end of the flow)"
	Import   SystemKeysOption = "Import existing keys"
)

func (s *SystemKeysOption) toString() string {
	switch *s {
	case Generate:
		return "generate"
	case Import:
		return "import"
	}
	return ""
}

func NewSystemKeysSelect(ctx context.Context) *SystemKeysSelect {
	return &SystemKeysSelect{
		Selector: ui.Selector[SystemKeysOption]{
			Options: []SystemKeysOption{
				Generate,
				Import,
			},
		},
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   "Select a setup method for the system keys",
		highlights: []string{"system keys"},
	}
}

func (m *SystemKeysSelect) GetQuestion() string {
	return m.question
}

func (m *SystemKeysSelect) Init() tea.Cmd {
	return nil
}

func (m *SystemKeysSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[LaunchState](m)
		analytics.TrackEvent(analytics.SystemKeysSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, selected.toString()))

		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), m.highlights, string(*selected)))
		switch *selected {
		case Generate:
			state.generateKeys = true
			model := NewExistingGasStationChecker(weavecontext.SetCurrentState(m.Ctx, state))
			return model, model.Init()
		case Import:
			return NewSystemKeyOperatorMnemonicInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
		}
	}

	return m, cmd
}

func (m *SystemKeysSelect) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" +
		styles.RenderPrompt(
			"System keys are required for each of the following roles:\nRollup Operator, Bridge Executor, Output Submitter, Batch Submitter, Challenger",
			[]string{"System keys"},
			styles.Information,
		) + "\n" +
		styles.RenderPrompt(
			m.GetQuestion(),
			m.highlights,
			styles.Question,
		) + m.Selector.View())
}

type SystemKeyOperatorMnemonicInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewSystemKeyOperatorMnemonicInput(ctx context.Context) *SystemKeyOperatorMnemonicInput {
	toolTip := tooltip.SystemKeyOperatorMnemonicTooltip
	model := &SystemKeyOperatorMnemonicInput{
		TextInput:  ui.NewTextInput(false),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   "Specify the mnemonic for the rollup operator",
		highlights: []string{"rollup operator"},
	}
	model.WithPlaceholder("Enter the mnemonic")
	model.WithValidatorFn(common.ValidateMnemonic)
	model.WithTooltip(&toolTip)
	return model
}

func (m *SystemKeyOperatorMnemonicInput) GetQuestion() string {
	return m.question
}

func (m *SystemKeyOperatorMnemonicInput) Init() tea.Cmd {
	return nil
}

func (m *SystemKeyOperatorMnemonicInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		// TODO: Check if duplicate
		state.systemKeyOperatorMnemonic = input.Text
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), m.highlights, styles.HiddenMnemonicText))
		return NewSystemKeyBridgeExecutorMnemonicInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *SystemKeyOperatorMnemonicInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.TextInput.View())
}

type SystemKeyBridgeExecutorMnemonicInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewSystemKeyBridgeExecutorMnemonicInput(ctx context.Context) *SystemKeyBridgeExecutorMnemonicInput {
	toolTip := tooltip.SystemKeyBridgeExecutorMnemonicTooltip
	model := &SystemKeyBridgeExecutorMnemonicInput{
		TextInput:  ui.NewTextInput(false),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   "Specify the mnemonic for the bridge executor",
		highlights: []string{"bridge executor"},
	}
	model.WithPlaceholder("Enter the mnemonic")
	model.WithValidatorFn(common.ValidateMnemonic)
	model.WithTooltip(&toolTip)
	return model
}

func (m *SystemKeyBridgeExecutorMnemonicInput) GetQuestion() string {
	return m.question
}

func (m *SystemKeyBridgeExecutorMnemonicInput) Init() tea.Cmd {
	return nil
}

func (m *SystemKeyBridgeExecutorMnemonicInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.systemKeyBridgeExecutorMnemonic = input.Text
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), m.highlights, styles.HiddenMnemonicText))
		return NewSystemKeyOutputSubmitterMnemonicInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *SystemKeyBridgeExecutorMnemonicInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.TextInput.View())
}

type SystemKeyOutputSubmitterMnemonicInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewSystemKeyOutputSubmitterMnemonicInput(ctx context.Context) *SystemKeyOutputSubmitterMnemonicInput {
	toolTip := tooltip.SystemKeyOutputSubmitterMnemonicTooltip
	model := &SystemKeyOutputSubmitterMnemonicInput{
		TextInput:  ui.NewTextInput(false),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   "Specify the mnemonic for the output submitter",
		highlights: []string{"output submitter"},
	}
	model.WithPlaceholder("Enter the mnemonic")
	model.WithValidatorFn(common.ValidateMnemonic)
	model.WithTooltip(&toolTip)
	return model
}

func (m *SystemKeyOutputSubmitterMnemonicInput) GetQuestion() string {
	return m.question
}

func (m *SystemKeyOutputSubmitterMnemonicInput) Init() tea.Cmd {
	return nil
}

func (m *SystemKeyOutputSubmitterMnemonicInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.systemKeyOutputSubmitterMnemonic = input.Text
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), m.highlights, styles.HiddenMnemonicText))
		return NewSystemKeyBatchSubmitterMnemonicInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *SystemKeyOutputSubmitterMnemonicInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.TextInput.View())
}

type SystemKeyBatchSubmitterMnemonicInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewSystemKeyBatchSubmitterMnemonicInput(ctx context.Context) *SystemKeyBatchSubmitterMnemonicInput {
	toolTip := tooltip.SystemKeyBatchSubmitterMnemonicTooltip
	model := &SystemKeyBatchSubmitterMnemonicInput{
		TextInput:  ui.NewTextInput(false),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   "Specify the mnemonic for the batch submitter",
		highlights: []string{"batch submitter"},
	}
	model.WithPlaceholder("Enter the mnemonic")
	model.WithValidatorFn(common.ValidateMnemonic)
	model.WithTooltip(&toolTip)
	return model
}

func (m *SystemKeyBatchSubmitterMnemonicInput) GetQuestion() string {
	return m.question
}

func (m *SystemKeyBatchSubmitterMnemonicInput) Init() tea.Cmd {
	return nil
}

func (m *SystemKeyBatchSubmitterMnemonicInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.systemKeyBatchSubmitterMnemonic = input.Text
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), m.highlights, styles.HiddenMnemonicText))
		return NewSystemKeyChallengerMnemonicInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *SystemKeyBatchSubmitterMnemonicInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.TextInput.View())
}

type SystemKeyChallengerMnemonicInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewSystemKeyChallengerMnemonicInput(ctx context.Context) *SystemKeyChallengerMnemonicInput {
	toolTip := tooltip.SystemKeyChallengerMnemonicTooltip
	model := &SystemKeyChallengerMnemonicInput{
		TextInput:  ui.NewTextInput(false),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   "Specify the mnemonic for the challenger",
		highlights: []string{"challenger"},
	}
	model.WithPlaceholder("Enter the mnemonic")
	model.WithValidatorFn(common.ValidateMnemonic)
	model.WithTooltip(&toolTip)
	return model
}

func (m *SystemKeyChallengerMnemonicInput) GetQuestion() string {
	return m.question
}

func (m *SystemKeyChallengerMnemonicInput) Init() tea.Cmd {
	return nil
}

func (m *SystemKeyChallengerMnemonicInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.systemKeyChallengerMnemonic = input.Text
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), m.highlights, styles.HiddenMnemonicText))
		model := NewExistingGasStationChecker(weavecontext.SetCurrentState(m.Ctx, state))
		return model, model.Init()
	}
	m.TextInput = input
	return m, cmd
}

func (m *SystemKeyChallengerMnemonicInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.TextInput.View())
}

type ExistingGasStationChecker struct {
	ui.Loading
	weavecontext.BaseModel
}

func NewExistingGasStationChecker(ctx context.Context) *ExistingGasStationChecker {
	return &ExistingGasStationChecker{
		Loading:   ui.NewLoading("Checking for gas station account...", waitExistingGasStationChecker(ctx)),
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
	}
}

func (m *ExistingGasStationChecker) Init() tea.Cmd {
	return m.Loading.Init()
}

func waitExistingGasStationChecker(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1500 * time.Millisecond)

		state := weavecontext.GetCurrentState[LaunchState](ctx)
		if config.IsFirstTimeSetup() {
			state.gasStationExist = false
			return ui.EndLoading{
				Ctx: weavecontext.SetCurrentState(ctx, state),
			}
		} else {
			state.gasStationExist = true
			return ui.EndLoading{
				Ctx: weavecontext.SetCurrentState(ctx, state),
			}
		}
	}
}

func (m *ExistingGasStationChecker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.Completing {
		m.Ctx = m.Loading.EndContext
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		if !state.gasStationExist {
			return NewGasStationMnemonicInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
		} else {
			model, err := NewAccountsFundingPresetSelect(weavecontext.SetCurrentState(m.Ctx, state))
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, nil
		}
	}
	return m, cmd
}

func (m *ExistingGasStationChecker) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" + m.Loading.View())
}

type GasStationMnemonicInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewGasStationMnemonicInput(ctx context.Context) *GasStationMnemonicInput {
	toolTip := tooltip.GasStationMnemonicTooltip
	model := &GasStationMnemonicInput{
		TextInput:  ui.NewTextInput(true),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		question:   fmt.Sprintf("Please set up a gas station account %s\n%s", styles.Text("(The account that will hold the funds required by the OPinit-bots or relayer to send transactions)", styles.Gray), styles.BoldText("Weave will not send any transactions without your confirmation.", styles.Yellow)),
		highlights: []string{"gas station account"},
	}
	model.WithPlaceholder("Enter the mnemonic")
	model.WithValidatorFn(common.ValidateMnemonic)
	model.WithTooltip(&toolTip)
	return model
}

func (m *GasStationMnemonicInput) GetQuestion() string {
	return m.question
}

func (m *GasStationMnemonicInput) Init() tea.Cmd {
	return nil
}

func (m *GasStationMnemonicInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		err := config.SetConfig("common.gas_station_mnemonic", input.Text)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), m.highlights, styles.HiddenMnemonicText))
		model, err := NewAccountsFundingPresetSelect(weavecontext.SetCurrentState(m.Ctx, state))
		if err != nil {
			return m, m.HandlePanic(err)
		}
		return model, nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *GasStationMnemonicInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" +
		styles.RenderPrompt(fmt.Sprintf("%s %s", styles.BoldUnderlineText("Please note that", styles.Yellow), styles.Text("you will need to set up a Gas Station account to fund the following accounts in order to run the weave rollup launch command:\n  • Bridge Executor\n  • Output Submitter\n  • Batch Submitter\n  • Challenger", styles.Yellow)), []string{}, styles.Information) + "\n" +
		styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.TextInput.View())
}

type AccountsFundingPresetSelect struct {
	ui.Selector[AccountsFundingPresetOption]
	weavecontext.BaseModel
	question string
}

type AccountsFundingPresetOption string

var DefaultPreset AccountsFundingPresetOption = ""

const ManuallyFill AccountsFundingPresetOption = "○ Fill in an amount for each account manually"

func (p *AccountsFundingPresetOption) toString() string {
	switch *p {
	case DefaultPreset:
		return "default"
	case ManuallyFill:
		return "manually"
	}
	return ""
}

func NewAccountsFundingPresetSelect(ctx context.Context) (*AccountsFundingPresetSelect, error) {
	state := weavecontext.GetCurrentState[LaunchState](ctx)
	tooltips := ui.NewTooltipSlice(
		tooltip.SystemAccountsFundingPresetTooltip, 2,
	)

	gasStationMnemonic := config.GetGasStationMnemonic()
	initiaGasStationAddress, err := crypto.MnemonicToBech32Address("init", gasStationMnemonic)
	if err != nil {
		return nil, fmt.Errorf("cannot recover gas station for init: %v", err)
	}
	var batchSubmitterDenom, batchSubmitterText, initiaNeededBalance, celestiaNeededBalance string
	if state.batchSubmissionIsCelestia {
		batchSubmitterDenom = DefaultCelestiaGasDenom
		batchSubmitterText = " on Celestia"
		initiaNeededBalance = DefaultL1InitiaNeededBalanceIfCelestiaDA
		var celestiaChainId string
		l1Registry, err := registry.GetChainRegistry(registry.InitiaL1Testnet)
		if err != nil {
			return nil, err
		}
		if state.l1ChainId == l1Registry.GetChainId() {
			celestiaRegistry, err := registry.GetChainRegistry(registry.CelestiaTestnet)
			if err != nil {
				return nil, err
			}
			celestiaChainId = celestiaRegistry.GetChainId()
		} else {
			celestiaRegistry, err := registry.GetChainRegistry(registry.CelestiaMainnet)
			if err != nil {
				return nil, err
			}
			celestiaChainId = celestiaRegistry.GetChainId()
		}
		celestiaGasStationAddress, err := crypto.MnemonicToBech32Address("celestia", gasStationMnemonic)
		if err != nil {
			return nil, fmt.Errorf("cannot recover gas station for celestia: %v", err)
		}
		celestiaNeededBalance = fmt.Sprintf("%s %s (%s)\n    ", styles.Text(fmt.Sprintf("• Celestia (%s):", celestiaChainId), styles.Cyan), styles.BoldText(fmt.Sprintf("%s%s", DefaultL1BatchSubmitterBalance, DefaultCelestiaGasDenom), styles.White), celestiaGasStationAddress)
	} else {
		batchSubmitterDenom = DefaultL1GasDenom
		batchSubmitterText = " on L1"
		initiaNeededBalance = DefaultL1InitiaNeededBalanceIfInitiaDA
	}
	separator := styles.Text("------------------------------------------------------------------------------------", styles.Gray)
	DefaultPreset = AccountsFundingPresetOption(fmt.Sprintf(
		"○ Use the default preset\n    %s\n    %s\n    %s %s on L1\n    %s %s on L1\n    %s %s%s\n    %s %s on L1\n    %s\n    %s\n    %s %s (%s)\n    %s%s\n",
		separator,
		styles.BoldText("• Executor", styles.Cyan),
		styles.BoldText("  • Bridge Executor:", styles.Cyan),
		styles.BoldText(fmt.Sprintf("%s%s", DefaultL1BridgeExecutorBalance, DefaultL1GasDenom), styles.White),
		styles.BoldText("  • Output Submitter:", styles.Cyan),
		styles.BoldText(fmt.Sprintf("%s%s", DefaultL1OutputSubmitterBalance, DefaultL1GasDenom), styles.White),
		styles.BoldText("  • Batch Submitter:", styles.Cyan),
		styles.BoldText(fmt.Sprintf("%s%s", DefaultL1BatchSubmitterBalance, batchSubmitterDenom), styles.White),
		batchSubmitterText,
		styles.BoldText("• Challenger:", styles.Cyan),
		styles.BoldText(fmt.Sprintf("%s%s", DefaultL1ChallengerBalance, DefaultL1GasDenom), styles.White),
		separator,
		styles.Text("Total amount required from the Gas Station account:", styles.Ivory),
		styles.Text(fmt.Sprintf("• L1 (%s):", state.l1ChainId), styles.Cyan),
		styles.BoldText(fmt.Sprintf("%s%s", initiaNeededBalance, DefaultL1GasDenom), styles.White),
		initiaGasStationAddress,
		celestiaNeededBalance,
		separator,
	))
	return &AccountsFundingPresetSelect{
		Selector: ui.Selector[AccountsFundingPresetOption]{
			Options: []AccountsFundingPresetOption{
				DefaultPreset,
				ManuallyFill,
			},
			CannotBack: true,
			Tooltips:   &tooltips,
		},
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		question:  "Select system accounts funding option",
	}, nil
}

func (m *AccountsFundingPresetSelect) GetQuestion() string {
	return m.question
}

func (m *AccountsFundingPresetSelect) Init() tea.Cmd {
	return nil
}

func (m *AccountsFundingPresetSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		analytics.TrackEvent(analytics.AccountsFundingPresetSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, selected.toString()))
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		switch *selected {
		case DefaultPreset:
			state.FillDefaultBalances()
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{}, "Use the default preset"))
			return NewFeeWhitelistAccountsInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
		case ManuallyFill:
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{}, "Fill in an amount for each account manually"))
			return NewSystemKeyL1BridgeExecutorBalanceInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
		}
	}

	return m, cmd
}

func (m *AccountsFundingPresetSelect) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	var l1Batch, celestiaBatch string
	if state.batchSubmissionIsCelestia {
		l1Batch = ""
		celestiaBatch = "\n  Celestia:\n  • Batch Submitter"
	} else {
		l1Batch = "• Batch Submitter\n  "
		celestiaBatch = ""
	}
	return m.WrapView(state.weave.Render() + "\n" +
		styles.RenderPrompt(
			fmt.Sprintf("You will need to fund the following accounts on ...\n  L1:\n  • Bridge Executor\n  • Output Submitter\n  %s• Challenger%s", l1Batch, celestiaBatch),
			[]string{"L1", "Rollup", "Celestia"},
			styles.Information,
		) + "\n" +
		styles.RenderPrompt(
			m.GetQuestion(),
			[]string{},
			styles.Question,
		) + m.Selector.View())
}

type SystemKeyL1BridgeExecutorBalanceInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewSystemKeyL1BridgeExecutorBalanceInput(ctx context.Context) *SystemKeyL1BridgeExecutorBalanceInput {
	state := weavecontext.GetCurrentState[LaunchState](ctx)
	state.preL1BalancesResponsesCount = len(state.weave.PreviousResponse)
	model := &SystemKeyL1BridgeExecutorBalanceInput{
		TextInput:  ui.NewTextInput(true),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		question:   "Specify the amount to fund the bridge executor on L1 (uinit)",
		highlights: []string{"bridge executor", "L1"},
	}
	model.WithPlaceholder("Enter a positive amount")
	model.WithValidatorFn(common.IsValidInteger)
	model.Ctx = weavecontext.SetCurrentState(model.Ctx, state)
	return model
}

func (m *SystemKeyL1BridgeExecutorBalanceInput) GetQuestion() string {
	return m.question
}

func (m *SystemKeyL1BridgeExecutorBalanceInput) Init() tea.Cmd {
	return nil
}

func (m *SystemKeyL1BridgeExecutorBalanceInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.systemKeyL1BridgeExecutorBalance = input.Text
		state.weave.PushPreviousResponse(fmt.Sprintf("\n%s\n", styles.RenderPrompt("Please fund the following accounts on L1:\n  • Bridge Executor\n  • Output Submitter\n  • Batch Submitter\n  • Challenger\n", []string{"L1"}, styles.Information)))
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), m.highlights, input.Text))
		return NewSystemKeyL1OutputSubmitterBalanceInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *SystemKeyL1BridgeExecutorBalanceInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" +
		styles.RenderPrompt("Please fund the following accounts on L1:\n  • Bridge Executor\n  • Output Submitter\n  • Batch Submitter\n  • Challenger", []string{"L1"}, styles.Information) + "\n" +
		styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.TextInput.View())
}

type SystemKeyL1OutputSubmitterBalanceInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewSystemKeyL1OutputSubmitterBalanceInput(ctx context.Context) *SystemKeyL1OutputSubmitterBalanceInput {
	model := &SystemKeyL1OutputSubmitterBalanceInput{
		TextInput:  ui.NewTextInput(false),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   "Specify the amount to fund the output submitter on L1 (uinit)",
		highlights: []string{"output submitter", "L1"},
	}
	model.WithPlaceholder("Enter a positive amount")
	model.WithValidatorFn(common.IsValidInteger)
	return model
}

func (m *SystemKeyL1OutputSubmitterBalanceInput) GetQuestion() string {
	return m.question
}

func (m *SystemKeyL1OutputSubmitterBalanceInput) Init() tea.Cmd {
	return nil
}

func (m *SystemKeyL1OutputSubmitterBalanceInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.systemKeyL1OutputSubmitterBalance = input.Text
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), m.highlights, input.Text))
		return NewSystemKeyL1BatchSubmitterBalanceInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *SystemKeyL1OutputSubmitterBalanceInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	return m.WrapView(state.weave.Render() +
		styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.TextInput.View())
}

type SystemKeyL1BatchSubmitterBalanceInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewSystemKeyL1BatchSubmitterBalanceInput(ctx context.Context) *SystemKeyL1BatchSubmitterBalanceInput {
	state := weavecontext.GetCurrentState[LaunchState](ctx)
	var denom, network string
	if state.batchSubmissionIsCelestia {
		denom = DefaultCelestiaGasDenom
		network = "Celestia Testnet"
	} else {
		denom = DefaultL1GasDenom
		network = "L1"
	}

	model := &SystemKeyL1BatchSubmitterBalanceInput{
		TextInput:  ui.NewTextInput(false),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   fmt.Sprintf("Specify the amount to fund the batch submitter on %s (%s)", network, denom),
		highlights: []string{"batch submitter", "L1", "Celestia Testnet"},
	}
	model.WithPlaceholder("Enter a positive amount")
	model.WithValidatorFn(common.IsValidInteger)
	return model
}

func (m *SystemKeyL1BatchSubmitterBalanceInput) GetQuestion() string {
	return m.question
}

func (m *SystemKeyL1BatchSubmitterBalanceInput) Init() tea.Cmd {
	return nil
}

func (m *SystemKeyL1BatchSubmitterBalanceInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.systemKeyL1BatchSubmitterBalance = input.Text
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), m.highlights, input.Text))
		return NewSystemKeyL1ChallengerBalanceInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *SystemKeyL1BatchSubmitterBalanceInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	return m.WrapView(state.weave.Render() +
		styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.TextInput.View())
}

type SystemKeyL1ChallengerBalanceInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewSystemKeyL1ChallengerBalanceInput(ctx context.Context) *SystemKeyL1ChallengerBalanceInput {
	model := &SystemKeyL1ChallengerBalanceInput{
		TextInput:  ui.NewTextInput(false),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   "Specify the amount to fund the challenger on L1 (uinit)",
		highlights: []string{"challenger", "L1"},
	}
	model.WithPlaceholder("Enter a positive amount")
	model.WithValidatorFn(common.IsValidInteger)
	return model
}

func (m *SystemKeyL1ChallengerBalanceInput) GetQuestion() string {
	return m.question
}

func (m *SystemKeyL1ChallengerBalanceInput) Init() tea.Cmd {
	return nil
}

func (m *SystemKeyL1ChallengerBalanceInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.systemKeyL1ChallengerBalance = input.Text
		state.weave.PopPreviousResponseAtIndex(state.preL1BalancesResponsesCount)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), m.highlights, input.Text))
		return NewFeeWhitelistAccountsInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *SystemKeyL1ChallengerBalanceInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	return m.WrapView(state.weave.Render() +
		styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.TextInput.View())
}

type AddGasStationToGenesisSelect struct {
	ui.Selector[AddGasStationToGenesisOption]
	weavecontext.BaseModel
	question   string
	highlights []string
}

type AddGasStationToGenesisOption string

const (
	Add     AddGasStationToGenesisOption = "Yes"
	DontAdd AddGasStationToGenesisOption = "No"
)

func NewAddGasStationToGenesisSelect(ctx context.Context) *AddGasStationToGenesisSelect {
	state := weavecontext.GetCurrentState[LaunchState](ctx)

	tooltips := ui.NewTooltipSlice(
		tooltip.GasStationInRollupGenesisTooltip, 2,
	)

	return &AddGasStationToGenesisSelect{
		Selector: ui.Selector[AddGasStationToGenesisOption]{
			Options: []AddGasStationToGenesisOption{
				Add,
				DontAdd,
			},
			CannotBack: true,
			Tooltips:   &tooltips,
		},
		BaseModel:  weavecontext.BaseModel{Ctx: weavecontext.SetCurrentState(ctx, state), CannotBack: true},
		question:   "Would you like to add the gas station account to genesis accounts?",
		highlights: []string{"gas station", "genesis"},
	}
}

func (m *AddGasStationToGenesisSelect) GetQuestion() string {
	return m.question
}

func (m *AddGasStationToGenesisSelect) Init() tea.Cmd {
	return nil
}

func (m *AddGasStationToGenesisSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		analytics.TrackEvent(analytics.AddGasStationToGenesisSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, string(*selected)))
		state := weavecontext.PushPageAndGetState[LaunchState](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), m.highlights, string(*selected)))

		switch *selected {
		case Add:
			model, err := NewGenesisGasStationBalanceInput(weavecontext.SetCurrentState(m.Ctx, state))
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, nil
		case DontAdd:
			return NewAddGenesisAccountsSelect(false, weavecontext.SetCurrentState(m.Ctx, state)), nil
		}
	}

	return m, cmd
}

func (m *AddGasStationToGenesisSelect) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" +
		styles.RenderPrompt("Adding a gas station account to the rollup genesis ensures that when running relayer init you would have funds to distribute to the relayer account.", []string{"gas station", "relayer init"}, styles.Information) + "\n" +
		styles.RenderPrompt(
			m.GetQuestion(),
			m.highlights,
			styles.Question,
		) + m.Selector.View())
}

type GenesisGasStationBalanceInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
	address    string
}

func NewGenesisGasStationBalanceInput(ctx context.Context) (*GenesisGasStationBalanceInput, error) {
	toolTip := tooltip.GasStationBalanceOnRollupGenesisTooltip
	state := weavecontext.GetCurrentState[LaunchState](ctx)
	gasStationAddress, err := crypto.MnemonicToBech32Address("init", config.GetGasStationMnemonic())
	if err != nil {
		return nil, fmt.Errorf("cannot recover gas station for init: %v", err)
	}

	model := &GenesisGasStationBalanceInput{
		TextInput:  ui.NewTextInput(false),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   fmt.Sprintf("Specify the genesis balance for the gas station account (%s)", state.gasDenom),
		highlights: []string{"gas station"},
		address:    gasStationAddress,
	}
	model.WithPlaceholder("Enter a positive amount")
	model.WithValidatorFn(common.IsValidInteger)
	model.WithTooltip(&toolTip)
	return model, nil
}

func (m *GenesisGasStationBalanceInput) GetQuestion() string {
	return m.question
}

func (m *GenesisGasStationBalanceInput) Init() tea.Cmd {
	return nil
}

func (m *GenesisGasStationBalanceInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.genesisAccounts = append(state.genesisAccounts, types.GenesisAccount{
			Coins:   fmt.Sprintf("%s%s", input.Text, state.gasDenom),
			Address: m.address,
		})

		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{m.address}, input.Text))
		return NewAddGenesisAccountsSelect(false, weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *GenesisGasStationBalanceInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{m.address}, styles.Question) + m.TextInput.View())
}

type AddGenesisAccountsSelect struct {
	ui.Selector[AddGenesisAccountsOption]
	weavecontext.BaseModel
	recurring         bool
	firstTimeQuestion string
	recurringQuestion string
}

type AddGenesisAccountsOption string

const (
	Yes AddGenesisAccountsOption = "Yes"
	No  AddGenesisAccountsOption = "No"
)

func NewAddGenesisAccountsSelect(recurring bool, ctx context.Context) *AddGenesisAccountsSelect {
	state := weavecontext.GetCurrentState[LaunchState](ctx)
	if !recurring {
		state.preGenesisAccountsResponsesCount = len(state.weave.PreviousResponse)
	}

	tooltips := ui.NewTooltipSlice(
		tooltip.GenesisAccountSelectTooltip, 2,
	)

	return &AddGenesisAccountsSelect{
		Selector: ui.Selector[AddGenesisAccountsOption]{
			Options: []AddGenesisAccountsOption{
				Yes,
				No,
			},
			CannotBack: true,
			Tooltips:   &tooltips,
		},
		BaseModel:         weavecontext.BaseModel{Ctx: weavecontext.SetCurrentState(ctx, state), CannotBack: true},
		recurring:         recurring,
		firstTimeQuestion: "Would you like to add genesis accounts?",
		recurringQuestion: "Would you like to add another genesis account?",
	}
}

func (m *AddGenesisAccountsSelect) GetQuestionAndHighlight() (string, string) {
	if m.recurring {
		return m.recurringQuestion, "genesis account"
	}
	return m.firstTimeQuestion, "genesis accounts"
}

func (m *AddGenesisAccountsSelect) Init() tea.Cmd {
	return nil
}

func (m *AddGenesisAccountsSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		analytics.TrackEvent(analytics.AddGenesisAccountsSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, string(*selected)))
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		switch *selected {
		case Yes:
			question, highlight := m.GetQuestionAndHighlight()
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, question, []string{highlight}, string(*selected)))
			return NewGenesisAccountsAddressInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
		case No:
			question := m.firstTimeQuestion
			highlight := "genesis accounts"
			if len(state.genesisAccounts) > 0 {
				state.weave.PreviousResponse = state.weave.PreviousResponse[:state.preGenesisAccountsResponsesCount]
				state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, question, []string{highlight}, string(Yes)))
				currentResponse := "  List of extra Genesis Accounts (excluding OPinit bots)\n"
				for _, account := range state.genesisAccounts {
					currentResponse += styles.Text(fmt.Sprintf("  %s\tInitial Balance: %s\n", account.Address, account.Coins), styles.Gray)
				}
				state.weave.PushPreviousResponse(currentResponse)
			} else {
				state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, question, []string{highlight}, string(No)))
			}
			model := NewDownloadMinitiaBinaryLoading(weavecontext.SetCurrentState(m.Ctx, state))
			return model, model.Init()
		}
	}

	return m, cmd
}

func (m *AddGenesisAccountsSelect) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	preText := ""
	if !m.recurring {
		preText += "\n" + styles.RenderPrompt("You can add extra genesis accounts by first entering the addresses, then assigning the initial balance one by one.", []string{"genesis accounts"}, styles.Information) + "\n"
	}
	question, highlight := m.GetQuestionAndHighlight()
	return m.WrapView(state.weave.Render() + preText + styles.RenderPrompt(
		question,
		[]string{highlight},
		styles.Question,
	) + m.Selector.View())
}

type FeeWhitelistAccountsInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question string
}

func NewFeeWhitelistAccountsInput(ctx context.Context) *FeeWhitelistAccountsInput {
	tooltip := tooltip.FeeWhitelistAccountsInputTooltip
	model := &FeeWhitelistAccountsInput{
		TextInput: ui.NewTextInput(true),
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  "Specify fee whitelist addresses",
	}
	model.WithTooltip(&tooltip)
	model.WithPlaceholder("Enter whitelist address (or leave this empty to skip), You can add multiple addresses by separating them with a comma (,)")
	model.WithValidatorFn(common.IsValidAddresses)
	return model
}

func (m *FeeWhitelistAccountsInput) GetQuestion() string {
	return m.question
}

func (m *FeeWhitelistAccountsInput) Init() tea.Cmd {
	return nil
}

func (m *FeeWhitelistAccountsInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)
		accs := strings.Trim(input.Text, "\n")
		var prevAnswer string
		if accs == "" {
			prevAnswer = "None"
		} else {
			prevAnswer = accs
		}
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"fee whitelist"}, prevAnswer))
		state.feeWhitelistAccounts = accs
		model := NewAddGasStationToGenesisSelect(weavecontext.SetCurrentState(m.Ctx, state))
		return model, model.Init()
	}
	m.TextInput = input
	return m, cmd
}

func (m *FeeWhitelistAccountsInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"fee whitelist"}, styles.Question) + m.TextInput.View())
}

type GenesisAccountsAddressInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question string
}

func NewGenesisAccountsAddressInput(ctx context.Context) *GenesisAccountsAddressInput {
	model := &GenesisAccountsAddressInput{
		TextInput: ui.NewTextInput(false),
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  "Specify a genesis account address",
	}
	model.WithPlaceholder("Enter a valid address")
	model.WithValidatorFn(common.IsValidAddress)
	return model
}

func (m *GenesisAccountsAddressInput) GetQuestion() string {
	return m.question
}

func (m *GenesisAccountsAddressInput) Init() tea.Cmd {
	return nil
}

func (m *GenesisAccountsAddressInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"genesis account address"}, input.Text))
		return NewGenesisAccountsBalanceInput(input.Text, weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *GenesisAccountsAddressInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"moniker"}, styles.Question) + m.TextInput.View())
}

type GenesisAccountsBalanceInput struct {
	ui.TextInput
	weavecontext.BaseModel
	address  string
	question string
}

func NewGenesisAccountsBalanceInput(address string, ctx context.Context) *GenesisAccountsBalanceInput {
	toolTip := tooltip.GenesisBalanceInputTooltip
	state := weavecontext.GetCurrentState[LaunchState](ctx)
	model := &GenesisAccountsBalanceInput{
		TextInput: ui.NewTextInput(false),
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		address:   address,
		question:  fmt.Sprintf("Specify the genesis balance for %s (%s)", address, state.gasDenom),
	}
	model.WithPlaceholder("Enter a positive amount")
	model.WithValidatorFn(common.IsValidInteger)
	model.WithTooltip(&toolTip)
	return model
}

func (m *GenesisAccountsBalanceInput) GetQuestion() string {
	return m.question
}

func (m *GenesisAccountsBalanceInput) Init() tea.Cmd {
	return nil
}

func (m *GenesisAccountsBalanceInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		state.genesisAccounts = append(state.genesisAccounts, types.GenesisAccount{
			Address: m.address,
			Coins:   fmt.Sprintf("%s%s", input.Text, state.gasDenom),
		})
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{m.address}, input.Text))
		return NewAddGenesisAccountsSelect(true, weavecontext.SetCurrentState(m.Ctx, state)), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *GenesisAccountsBalanceInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{m.address}, styles.Question) + m.TextInput.View())
}

type DownloadMinitiaBinaryLoading struct {
	ui.Loading
	weavecontext.BaseModel
}

func NewDownloadMinitiaBinaryLoading(ctx context.Context) *DownloadMinitiaBinaryLoading {
	state := weavecontext.GetCurrentState[LaunchState](ctx)
	latest := map[bool]string{true: "latest ", false: ""}
	return &DownloadMinitiaBinaryLoading{
		Loading:   ui.NewLoading(fmt.Sprintf("Downloading %sMini%s binary <%s>", latest[state.launchFromExistingConfig], strings.ToLower(state.vmType), state.minitiadVersion), downloadMinitiaApp(ctx)),
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
	}
}

func (m *DownloadMinitiaBinaryLoading) Init() tea.Cmd {
	return m.Loading.Init()
}

func downloadMinitiaApp(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		state := weavecontext.GetCurrentState[LaunchState](ctx)

		userHome, err := os.UserHomeDir()
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to get user home directory: %v", err)}
		}
		weaveDataPath := filepath.Join(userHome, common.WeaveDataDirectory)
		tarballPath := filepath.Join(weaveDataPath, "minitia.tar.gz")
		extractedPath := filepath.Join(weaveDataPath, fmt.Sprintf("mini%s@%s", strings.ToLower(state.vmType), state.minitiadVersion))

		var binaryPath string
		switch runtime.GOOS {
		case "linux":
			binaryPath = filepath.Join(extractedPath, fmt.Sprintf("mini%s_%s", strings.ToLower(state.vmType), state.minitiadVersion), AppName)
		case "darwin":
			binaryPath = filepath.Join(extractedPath, AppName)
		default:
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("unsupported OS: %v", runtime.GOOS)}
		}
		state.binaryPath = binaryPath

		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			if _, err := os.Stat(extractedPath); os.IsNotExist(err) {
				err := os.MkdirAll(extractedPath, os.ModePerm)
				if err != nil {
					return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to create weave data directory: %v", err)}
				}
			}

			if err = io.DownloadAndExtractTarGz(state.minitiadEndpoint, tarballPath, extractedPath); err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to download and extract binary: %v", err)}
			}

			err = os.Chmod(binaryPath, 0755)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to set permissions for binary: %v", err)}
			}

			state.downloadedNewBinary = true
		}

		if state.vmType == string(Move) || state.vmType == string(Wasm) {
			err = io.SetLibraryPaths(filepath.Dir(binaryPath))
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to set library path: %v", err)}
			}
		}

		return ui.EndLoading{
			Ctx: weavecontext.SetCurrentState(ctx, state),
		}
	}
}

func (m *DownloadMinitiaBinaryLoading) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		m.Ctx = m.Loading.EndContext
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		if state.downloadedNewBinary {
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.NoSeparator, fmt.Sprintf("Mini%s binary has been successfully downloaded.", strings.ToLower(state.vmType)), []string{}, ""))
		}

		if state.launchFromExistingConfig {
			model := NewLaunchingNewMinitiaLoading(weavecontext.SetCurrentState(m.Ctx, state))
			return model, model.Init()
		}

		if state.batchSubmissionIsCelestia {
			model, err := NewDownloadCelestiaBinaryLoading(weavecontext.SetCurrentState(m.Ctx, state))
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, model.Init()
		}

		model := NewGenerateOrRecoverSystemKeysLoading(weavecontext.SetCurrentState(m.Ctx, state))
		return model, model.Init()
	}
	return m, cmd
}

func (m *DownloadMinitiaBinaryLoading) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" + m.Loading.View())
}

type DownloadCelestiaBinaryLoading struct {
	ui.Loading
	weavecontext.BaseModel
}

func NewDownloadCelestiaBinaryLoading(ctx context.Context) (*DownloadCelestiaBinaryLoading, error) {
	celestiaMainnetRegistry, err := registry.GetChainRegistry(registry.CelestiaMainnet)
	if err != nil {
		return nil, err
	}
	httpClient := client.NewHTTPClient()

	activeLcd, err := celestiaMainnetRegistry.GetActiveLcd()
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	_, err = httpClient.Get(
		activeLcd,
		"/cosmos/base/tendermint/v1beta1/node_info",
		nil,
		&result,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch node info: %v", err)
	}

	applicationVersion, ok := result["application_version"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to get node version")
	}
	version := applicationVersion["version"].(string)
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	binaryUrl, err := getCelestiaBinaryURL(version, goos, goarch)
	if err != nil {
		return nil, fmt.Errorf("failed to get celestia binary url: %v", err)
	}
	return &DownloadCelestiaBinaryLoading{
		Loading:   ui.NewLoading(fmt.Sprintf("Downloading Celestia binary <%s>", version), downloadCelestiaApp(ctx, version, binaryUrl)),
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
	}, nil
}

func getCelestiaBinaryURL(version, os, arch string) (string, error) {
	switch os {
	case "darwin":
		switch arch {
		case "amd64":
			return fmt.Sprintf("https://github.com/celestiaorg/celestia-app/releases/download/v%s/celestia-app_Darwin_x86_64.tar.gz", version), nil
		case "arm64":
			return fmt.Sprintf("https://github.com/celestiaorg/celestia-app/releases/download/v%s/celestia-app_Darwin_arm64.tar.gz", version), nil
		}
	case "linux":
		switch arch {
		case "amd64":
			return fmt.Sprintf("https://github.com/celestiaorg/celestia-app/releases/download/v%s/celestia-app_Linux_x86_64.tar.gz", version), nil
		case "arm64":
			return fmt.Sprintf("https://github.com/celestiaorg/celestia-app/releases/download/v%s/celestia-app_Linux_arm64.tar.gz", version), nil
		}
	}
	return "", fmt.Errorf("unsupported OS or architecture: %v %v", os, arch)
}

func (m *DownloadCelestiaBinaryLoading) Init() tea.Cmd {
	return m.Loading.Init()
}

func downloadCelestiaApp(ctx context.Context, version, binaryUrl string) tea.Cmd {
	return func() tea.Msg {
		state := weavecontext.GetCurrentState[LaunchState](ctx)
		userHome, err := os.UserHomeDir()
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to get user home directory: %v", err)}
		}
		weaveDataPath := filepath.Join(userHome, common.WeaveDataDirectory)
		tarballPath := filepath.Join(weaveDataPath, "celestia.tar.gz")
		extractedPath := filepath.Join(weaveDataPath, fmt.Sprintf("celestia@%s", version))
		binaryPath := filepath.Join(extractedPath, CelestiaAppName)
		state.celestiaBinaryPath = binaryPath

		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			if _, err := os.Stat(extractedPath); os.IsNotExist(err) {
				err := os.MkdirAll(extractedPath, os.ModePerm)
				if err != nil {
					return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to create weave data directory: %v", err)}
				}
			}

			if err = io.DownloadAndExtractTarGz(binaryUrl, tarballPath, extractedPath); err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to download and extract binary: %v", err)}
			}

			err = os.Chmod(binaryPath, 0755)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to set permissions for binary: %v", err)}
			}

			state.downloadedNewCelestiaBinary = true
		}

		return ui.EndLoading{
			Ctx: weavecontext.SetCurrentState(ctx, state),
		}
	}
}

func (m *DownloadCelestiaBinaryLoading) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		m.Ctx = m.Loading.EndContext
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		if state.downloadedNewCelestiaBinary {
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.NoSeparator, "Celestia binary has been successfully downloaded.", []string{}, ""))
		}
		model := NewGenerateOrRecoverSystemKeysLoading(weavecontext.SetCurrentState(m.Ctx, state))
		return model, model.Init()
	}
	return m, cmd
}

func (m *DownloadCelestiaBinaryLoading) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" + m.Loading.View())
}

type GenerateOrRecoverSystemKeysLoading struct {
	ui.Loading
	weavecontext.BaseModel
}

func NewGenerateOrRecoverSystemKeysLoading(ctx context.Context) *GenerateOrRecoverSystemKeysLoading {
	state := weavecontext.GetCurrentState[LaunchState](ctx)
	var loadingText string
	if state.generateKeys {
		loadingText = "Generating new system keys..."
	} else {
		loadingText = "Recovering system keys..."
	}
	return &GenerateOrRecoverSystemKeysLoading{
		Loading:   ui.NewLoading(loadingText, generateOrRecoverSystemKeys(ctx)),
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
	}
}

func (m *GenerateOrRecoverSystemKeysLoading) Init() tea.Cmd {
	return m.Loading.Init()
}

func generateOrRecoverSystemKeys(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		state := weavecontext.GetCurrentState[LaunchState](ctx)
		if state.generateKeys {
			operatorKey, err := cosmosutils.GenerateNewKeyInfo(state.binaryPath, OperatorKeyName)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to generate operator key: %v", err)}
			}
			state.systemKeyOperatorMnemonic = operatorKey.Mnemonic
			state.systemKeyOperatorAddress = operatorKey.Address

			bridgeExecutorKey, err := cosmosutils.GenerateNewKeyInfo(state.binaryPath, BridgeExecutorKeyName)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to generate bridge executor key: %v", err)}
			}
			state.systemKeyBridgeExecutorMnemonic = bridgeExecutorKey.Mnemonic
			state.systemKeyBridgeExecutorAddress = bridgeExecutorKey.Address

			outputSubmitterKey, err := cosmosutils.GenerateNewKeyInfo(state.binaryPath, OutputSubmitterKeyName)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to generate output submitter key: %v", err)}
			}
			state.systemKeyOutputSubmitterMnemonic = outputSubmitterKey.Mnemonic
			state.systemKeyOutputSubmitterAddress = outputSubmitterKey.Address

			if state.batchSubmissionIsCelestia {
				batchSubmitterKey, err := cosmosutils.GenerateNewKeyInfo(state.celestiaBinaryPath, BatchSubmitterKeyName)
				if err != nil {
					return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to generate celestia batch submitter key: %v", err)}
				}
				state.systemKeyBatchSubmitterMnemonic = batchSubmitterKey.Mnemonic
				state.systemKeyBatchSubmitterAddress = batchSubmitterKey.Address
			} else {
				batchSubmitterKey, err := cosmosutils.GenerateNewKeyInfo(state.binaryPath, BatchSubmitterKeyName)
				if err != nil {
					return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to generate initia batch submitter key: %v", err)}
				}
				state.systemKeyBatchSubmitterMnemonic = batchSubmitterKey.Mnemonic
				state.systemKeyBatchSubmitterAddress = batchSubmitterKey.Address
			}

			challengerKey, err := cosmosutils.GenerateNewKeyInfo(state.binaryPath, ChallengerKeyName)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to generate challenger key: %v", err)}
			}
			state.systemKeyChallengerMnemonic = challengerKey.Mnemonic
			state.systemKeyChallengerAddress = challengerKey.Address
		} else {
			var err error
			state.systemKeyOperatorAddress, err = cosmosutils.GetAddressFromMnemonic(state.binaryPath, state.systemKeyOperatorMnemonic)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to recover key operator address: %v", err)}
			}
			state.systemKeyBridgeExecutorAddress, err = cosmosutils.GetAddressFromMnemonic(state.binaryPath, state.systemKeyBridgeExecutorMnemonic)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to recover key bridge executor address: %v", err)}
			}
			state.systemKeyOutputSubmitterAddress, err = cosmosutils.GetAddressFromMnemonic(state.binaryPath, state.systemKeyOutputSubmitterMnemonic)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to recover key output submitter address: %v", err)}
			}
			if state.batchSubmissionIsCelestia {
				state.systemKeyBatchSubmitterAddress, err = cosmosutils.GetAddressFromMnemonic(state.celestiaBinaryPath, state.systemKeyBatchSubmitterMnemonic)
				if err != nil {
					return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to recover celestia batch submitter address: %v", err)}
				}
			} else {
				state.systemKeyBatchSubmitterAddress, err = cosmosutils.GetAddressFromMnemonic(state.binaryPath, state.systemKeyBatchSubmitterMnemonic)
				if err != nil {
					return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to recover initia batch submitter address: %v", err)}
				}
			}
			state.systemKeyChallengerAddress, err = cosmosutils.GetAddressFromMnemonic(state.binaryPath, state.systemKeyChallengerMnemonic)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to recover challenger address: %v", err)}
			}
		}

		state.FinalizeGenesisAccounts()
		time.Sleep(1500 * time.Millisecond)

		return ui.EndLoading{
			Ctx: weavecontext.SetCurrentState(ctx, state),
		}
	}
}

func (m *GenerateOrRecoverSystemKeysLoading) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		m.Ctx = m.Loading.EndContext
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		if state.generateKeys {
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.NoSeparator, "System keys have been successfully generated.", []string{}, ""))
			model := NewSystemKeysMnemonicDisplayInput(weavecontext.SetCurrentState(m.Ctx, state))
			return model, model.Init()
		} else {
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.NoSeparator, "System keys have been successfully recovered.", []string{}, ""))
			model, err := NewFundGasStationConfirmationInput(weavecontext.SetCurrentState(m.Ctx, state))
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, nil
		}
	}
	return m, cmd
}

func (m *GenerateOrRecoverSystemKeysLoading) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" + m.Loading.View())
}

type SystemKeysMnemonicDisplayInput struct {
	ui.TextInput
	//ui.Clickable
	weavecontext.BaseModel
	question string
}

// NewSystemKeysMnemonicDisplayInput
// TODO: Switch to the comment version if we manage to fix the height content overflow issue
func NewSystemKeysMnemonicDisplayInput(ctx context.Context) *SystemKeysMnemonicDisplayInput {
	//state := weavecontext.GetCurrentState[LaunchState](ctx)
	model := &SystemKeysMnemonicDisplayInput{
		TextInput: ui.NewTextInput(true),
		//Clickable: *ui.NewClickable([]*ui.ClickableItem{
		//	ui.NewClickableItem(map[bool]string{
		//		true:  "Copied! Click to copy again",
		//		false: "Click here to copy",
		//	}, func() error {
		//		return io.CopyToClipboard(state.systemKeyOperatorMnemonic)
		//	}),
		//	ui.NewClickableItem(map[bool]string{
		//		true:  "Copied! Click to copy again",
		//		false: "Click here to copy",
		//	}, func() error {
		//		return io.CopyToClipboard(state.systemKeyBridgeExecutorMnemonic)
		//	}),
		//	ui.NewClickableItem(map[bool]string{
		//		true:  "Copied! Click to copy again",
		//		false: "Click here to copy",
		//	}, func() error {
		//		return io.CopyToClipboard(state.systemKeyOutputSubmitterMnemonic)
		//	}),
		//	ui.NewClickableItem(map[bool]string{
		//		true:  "Copied! Click to copy again",
		//		false: "Click here to copy",
		//	}, func() error {
		//		return io.CopyToClipboard(state.systemKeyBatchSubmitterMnemonic)
		//	}),
		//	ui.NewClickableItem(map[bool]string{
		//		true:  "Copied! Click to copy again",
		//		false: "Click here to copy",
		//	}, func() error {
		//		return io.CopyToClipboard(state.systemKeyChallengerMnemonic)
		//	}),
		//}...),
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		question:  "Type `continue` to proceed after you have securely stored the mnemonic.",
	}
	model.WithPlaceholder("Type `continue` to continue, Ctrl+C to quit.")
	model.WithValidatorFn(common.ValidateExactString("continue"))
	return model
}

func (m *SystemKeysMnemonicDisplayInput) GetQuestion() string {
	return m.question
}

func (m *SystemKeysMnemonicDisplayInput) Init() tea.Cmd {
	//return m.Clickable.Init()
	return nil
}

func (m *SystemKeysMnemonicDisplayInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	//err := m.Clickable.ClickableUpdate(msg)
	//if err != nil {
	//	return m, m.HandlePanic(err)
	//}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		_ = weavecontext.PushPageAndGetState[LaunchState](m)
		model, err := NewFundGasStationConfirmationInput(m.Ctx)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		//return model, m.Clickable.PostUpdate()
		return model, nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *SystemKeysMnemonicDisplayInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)

	var mnemonicText string
	//mnemonicText += styles.RenderMnemonic("Operator", state.systemKeyOperatorAddress, state.systemKeyOperatorMnemonic, m.Clickable.ClickableView(0))
	//mnemonicText += styles.RenderMnemonic("Bridge Executor", state.systemKeyBridgeExecutorAddress, state.systemKeyBridgeExecutorMnemonic, m.Clickable.ClickableView(1))
	//mnemonicText += styles.RenderMnemonic("Output Submitter", state.systemKeyOutputSubmitterAddress, state.systemKeyOutputSubmitterMnemonic, m.Clickable.ClickableView(2))
	//mnemonicText += styles.RenderMnemonic("Batch Submitter", state.systemKeyBatchSubmitterAddress, state.systemKeyBatchSubmitterMnemonic, m.Clickable.ClickableView(3))
	//mnemonicText += styles.RenderMnemonic("Challenger", state.systemKeyChallengerAddress, state.systemKeyChallengerMnemonic, m.Clickable.ClickableView(4))
	mnemonicText += styles.RenderMnemonic("Operator", state.systemKeyOperatorAddress, state.systemKeyOperatorMnemonic, "")
	mnemonicText += styles.RenderMnemonic("Bridge Executor", state.systemKeyBridgeExecutorAddress, state.systemKeyBridgeExecutorMnemonic, "")
	mnemonicText += styles.RenderMnemonic("Output Submitter", state.systemKeyOutputSubmitterAddress, state.systemKeyOutputSubmitterMnemonic, "")
	mnemonicText += styles.RenderMnemonic("Batch Submitter", state.systemKeyBatchSubmitterAddress, state.systemKeyBatchSubmitterMnemonic, "")
	mnemonicText += styles.RenderMnemonic("Challenger", state.systemKeyChallengerAddress, state.systemKeyChallengerMnemonic, "")

	userHome, err := os.UserHomeDir()
	if err != nil {
		m.HandlePanic(err)
	}
	configFilePath := filepath.Join(userHome, common.WeaveDataDirectory, LaunchConfigFilename)
	viewText := m.WrapView(state.weave.Render() + "\n" +
		styles.BoldUnderlineText("Important", styles.Yellow) + "\n" +
		styles.Text(fmt.Sprintf("Write down these mnemonic phrases and store them in a safe place. \nIt is the only way to recover your system keys.\n\nNote that before launching, these mnemonic phrases along with other configuration details will be stored\nin %s. You can revisit them anytime.", configFilePath), styles.Yellow) + "\n\n" +
		mnemonicText + styles.RenderPrompt(m.GetQuestion(), []string{"`continue`"}, styles.Question) + m.TextInput.View())
	//err = m.Clickable.ClickableUpdatePositions(viewText)
	if err != nil {
		m.HandlePanic(err)
	}
	return viewText
}

type FundGasStationConfirmationInput struct {
	ui.TextInput
	weavecontext.BaseModel
	initiaGasStationAddress   string
	celestiaGasStationAddress string
	question                  string
}

func NewFundGasStationConfirmationInput(ctx context.Context) (*FundGasStationConfirmationInput, error) {
	state := weavecontext.GetCurrentState[LaunchState](ctx)
	gasStationMnemonic := config.GetGasStationMnemonic()
	var celestiaGasStationAddress string
	if state.batchSubmissionIsCelestia {
		var err error
		celestiaGasStationAddress, err = cosmosutils.GetAddressFromMnemonic(state.celestiaBinaryPath, gasStationMnemonic)
		if err != nil {
			return nil, fmt.Errorf("failed to get the celestia gas station address: %w", err)
		}
	}
	initiaGasStationAddress, err := cosmosutils.GetAddressFromMnemonic(state.binaryPath, gasStationMnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to get the initia gas station address: %w", err)
	}
	model := &FundGasStationConfirmationInput{
		TextInput:                 ui.NewTextInput(false),
		BaseModel:                 weavecontext.BaseModel{Ctx: ctx},
		initiaGasStationAddress:   initiaGasStationAddress,
		celestiaGasStationAddress: celestiaGasStationAddress,
		question:                  "Confirm to proceed with signing and broadcasting the following transactions? [y]:",
	}
	model.WithPlaceholder("Type `y` to confirm")
	model.WithValidatorFn(common.ValidateExactString("y"))
	return model, nil
}

func (m *FundGasStationConfirmationInput) GetQuestion() string {
	return m.question
}

func (m *FundGasStationConfirmationInput) Init() tea.Cmd {
	return nil
}

func (m *FundGasStationConfirmationInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		_ = weavecontext.PushPageAndGetState[LaunchState](m)
		model := NewFundGasStationBroadcastLoading(m.Ctx)
		return model, model.Init()
	}
	m.TextInput = input
	return m, cmd
}

func (m *FundGasStationConfirmationInput) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	formatSendMsg := func(coins, denom, keyName, address string) string {
		return fmt.Sprintf(
			"> Send %s to %s %s\n",
			styles.BoldText(coins+denom, styles.Ivory),
			styles.BoldText(keyName, styles.Ivory),
			styles.Text(fmt.Sprintf("(%s)", address), styles.Gray))
	}
	headerText := map[bool]string{
		true:  "Weave will now broadcast the following transactions",
		false: "Weave will now broadcast the following transaction",
	}
	batchSubmitterText := map[bool]string{
		true:  "",
		false: formatSendMsg(state.systemKeyL1BatchSubmitterBalance, "uinit", "Batch Submitter on Initia L1", state.systemKeyBatchSubmitterAddress),
	}
	celestiaText := map[bool]string{
		true:  fmt.Sprintf("\nSending tokens from the Gas Station account on Celestia Testnet %s ⛽️\n%s", styles.Text(fmt.Sprintf("(%s)", m.celestiaGasStationAddress), styles.Gray), formatSendMsg(state.systemKeyL1BatchSubmitterBalance, DefaultCelestiaGasDenom, "Batch Submitter on Celestia Testnet", state.systemKeyBatchSubmitterAddress)),
		false: "",
	}
	return m.WrapView(state.weave.Render() + "\n" +
		styles.Text("i ", styles.Yellow) +
		styles.RenderPrompt(
			styles.BoldUnderlineText(headerText[state.batchSubmissionIsCelestia], styles.Yellow),
			[]string{}, styles.Empty,
		) + "\n\n" +
		fmt.Sprintf("Sending tokens from the Gas Station account on Initia L1 %s ⛽️\n", styles.Text(fmt.Sprintf("(%s)", m.initiaGasStationAddress), styles.Gray)) +
		formatSendMsg(state.systemKeyL1BridgeExecutorBalance, "uinit", "Bridge Executor on Initia L1", state.systemKeyBridgeExecutorAddress) +
		formatSendMsg(state.systemKeyL1OutputSubmitterBalance, "uinit", "Output Submitter on Initia L1", state.systemKeyOutputSubmitterAddress) +
		batchSubmitterText[state.batchSubmissionIsCelestia] +
		formatSendMsg(state.systemKeyL1ChallengerBalance, "uinit", "Challenger on Initia L1", state.systemKeyChallengerAddress) +
		celestiaText[state.batchSubmissionIsCelestia] +
		styles.RenderPrompt(m.GetQuestion(), []string{"`continue`"}, styles.Question) + m.TextInput.View())
}

type FundGasStationBroadcastLoading struct {
	ui.Loading
	weavecontext.BaseModel
}

func NewFundGasStationBroadcastLoading(ctx context.Context) *FundGasStationBroadcastLoading {
	return &FundGasStationBroadcastLoading{
		Loading:   ui.NewLoading("Broadcasting transactions...", broadcastFundingFromGasStation(ctx)),
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
	}
}

func (m *FundGasStationBroadcastLoading) Init() tea.Cmd {
	return m.Loading.Init()
}

func broadcastFundingFromGasStation(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		state := weavecontext.GetCurrentState[LaunchState](ctx)
		systemKeys := NewL1SystemKeys(
			&types.GenesisAccount{
				Address: state.systemKeyBridgeExecutorAddress,
				Coins:   state.systemKeyL1BridgeExecutorBalance,
			},
			&types.GenesisAccount{
				Address: state.systemKeyOutputSubmitterAddress,
				Coins:   state.systemKeyL1OutputSubmitterBalance,
			},
			&types.GenesisAccount{
				Address: state.systemKeyBatchSubmitterAddress,
				Coins:   state.systemKeyL1BatchSubmitterBalance,
			},
			&types.GenesisAccount{
				Address: state.systemKeyChallengerAddress,
				Coins:   state.systemKeyL1ChallengerBalance,
			},
		)
		txResult, err := systemKeys.FundAccountsWithGasStation(&state)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: err}
		}

		if txResult.CelestiaTx != nil {
			state.systemKeyCelestiaFundingTxHash = txResult.CelestiaTx.TxHash
		}
		state.systemKeyL1FundingTxHash = txResult.InitiaTx.TxHash
		time.Sleep(1500 * time.Millisecond)

		return ui.EndLoading{
			Ctx: weavecontext.SetCurrentState(ctx, state),
		}
	}
}

func (m *FundGasStationBroadcastLoading) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		m.Ctx = m.Loading.EndContext
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		if state.systemKeyCelestiaFundingTxHash != "" {
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, "Batch Submitter on Celestia funded via Gas Station, with Tx Hash", []string{}, state.systemKeyCelestiaFundingTxHash))
		}
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, "System keys on Initia L1 funded via Gas Station, with Tx Hash", []string{}, state.systemKeyL1FundingTxHash))
		model := NewLaunchingNewMinitiaLoading(weavecontext.SetCurrentState(m.Ctx, state))
		return model, model.Init()
	}
	return m, cmd
}

func (m *FundGasStationBroadcastLoading) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" + m.Loading.View())
}

type ScanPayload struct {
	Vm          string  `json:"vm"`
	ChainId     string  `json:"chainId"`
	MinGasPrice float64 `json:"minGasPrice"`
	Denom       string  `json:"denom"`
	Lcd         string  `json:"lcd"`
	Rpc         string  `json:"rpc"`
	JsonRpc     string  `json:"jsonRpc,omitempty"`
}

func (sp *ScanPayload) EncodeToBase64() (string, error) {
	jsonBytes, err := json.Marshal(sp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal struct: %w", err)
	}

	base64String := base64.StdEncoding.EncodeToString(jsonBytes)
	return base64String, nil
}

type LaunchingNewMinitiaLoading struct {
	ui.Loading
	weavecontext.BaseModel
	streamingLogs *[]string
}

func NewLaunchingNewMinitiaLoading(ctx context.Context) *LaunchingNewMinitiaLoading {
	newLogs := make([]string, 0)
	return &LaunchingNewMinitiaLoading{
		Loading: ui.NewLoading(
			styles.RenderPrompt(
				"Running `minitiad launch` with the specified config...",
				[]string{"`minitiad launch`"},
				styles.Empty,
			), launchingMinitia(ctx, &newLogs)),
		BaseModel:     weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		streamingLogs: &newLogs,
	}
}

func (m *LaunchingNewMinitiaLoading) Init() tea.Cmd {
	return m.Loading.Init()
}

var timestampRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z`)
var initPrefixRegex = regexp.MustCompile(`^init1`)

func isJSONLog(line string) bool {
	return timestampRegex.MatchString(line) || initPrefixRegex.MatchString(line)
}

func launchingMinitia(ctx context.Context, streamingLogs *[]string) tea.Cmd {
	return func() tea.Msg {
		state := weavecontext.GetCurrentState[LaunchState](ctx)
		userHome, err := os.UserHomeDir()
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to get user home directory: %v", err)}
		}
		var configFilePath string
		if state.launchFromExistingConfig {
			configFilePath = state.existingConfigPath
		} else {

			minitiaConfig := &types.MinitiaConfig{
				L1Config: &types.L1Config{
					ChainID:   state.l1ChainId,
					RpcUrl:    state.l1RPC,
					GasPrices: DefaultL1GasPrices,
				},
				L2Config: &types.L2Config{
					ChainID: state.chainId,
					Denom:   state.gasDenom,
					Moniker: state.moniker,
				},
				OpBridge: &types.OpBridge{
					OutputSubmissionInterval:    state.opBridgeSubmissionInterval,
					OutputFinalizationPeriod:    state.opBridgeOutputFinalizationPeriod,
					OutputSubmissionStartHeight: 1,
					BatchSubmissionTarget:       state.opBridgeBatchSubmissionTarget,
					EnableOracle:                state.enableOracle,
				},
				SystemKeys: &types.SystemKeys{
					Validator: types.NewSystemAccount(
						state.systemKeyOperatorMnemonic,
						state.systemKeyOperatorAddress,
					),
					BridgeExecutor: types.NewSystemAccount(
						state.systemKeyBridgeExecutorMnemonic,
						state.systemKeyBridgeExecutorAddress,
					),
					OutputSubmitter: types.NewSystemAccount(
						state.systemKeyOutputSubmitterMnemonic,
						state.systemKeyOutputSubmitterAddress,
					),
					BatchSubmitter: types.NewBatchSubmitterAccount(
						state.systemKeyBatchSubmitterMnemonic,
						state.systemKeyBatchSubmitterAddress,
					),
					Challenger: types.NewSystemAccount(
						state.systemKeyChallengerMnemonic,
						state.systemKeyChallengerAddress,
					),
				},
				GenesisAccounts: &state.genesisAccounts,
			}

			configBz, err := json.MarshalIndent(minitiaConfig, "", " ")
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to marshal config: %v", err)}
			}

			configFilePath = filepath.Join(userHome, common.WeaveDataDirectory, LaunchConfigFilename)
			if err = os.WriteFile(configFilePath, configBz, 0600); err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to write config file: %v", err)}
			}
		}

		minitiaHome, err := weavecontext.GetMinitiaHome(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to get minitia home directory: %v", err)}
		}
		launchCmd := exec.Command(state.binaryPath, "launch", "--with-config", configFilePath, "--home", minitiaHome)

		stdout, err := launchCmd.StdoutPipe()
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to capture stdout: %v", err)}
		}
		stderr, err := launchCmd.StderrPipe()
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to capture stderr: %v", err)}
		}

		if err = launchCmd.Start(); err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to start command: %v", err)}
		}

		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				if !isJSONLog(line) {
					*streamingLogs = append(*streamingLogs, line)
					if len(*streamingLogs) > 10 {
						*streamingLogs = (*streamingLogs)[1:]
					}
				}
			}
		}()

		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				if !isJSONLog(line) {
					*streamingLogs = append(*streamingLogs, line)
					if len(*streamingLogs) > 10 {
						*streamingLogs = (*streamingLogs)[1:]
					}
				}
			}
		}()

		if err = launchCmd.Wait(); err != nil {
			if err != nil {
				*streamingLogs = append(*streamingLogs, fmt.Sprintf("Launch command finished with error: %v", err))
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("command execution failed: %v", err)}
			}
		}

		appConfigPath := filepath.Join(userHome, common.MinitiaConfigPath, "app.toml")
		if err = config.UpdateTomlValue(appConfigPath, "inter-block-cache", "false"); err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to set inter-block-cache: %v", err)}
		}
		if err = config.UpdateTomlValue(appConfigPath, "minimum-gas-prices", fmt.Sprintf("0%s", state.gasDenom)); err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to set minimum-gas-prices: %v", err)}
		}

		srv, err := service.NewService(service.Minitia)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to initialize service: %v", err)}
		}

		if err = srv.Create(fmt.Sprintf("mini%s@%s", strings.ToLower(state.vmType), state.minitiadVersion), minitiaHome); err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to create service: %v", err)}
		}

		// prune existing logs, ignore error
		_ = srv.PruneLogs()

		return ui.EndLoading{
			Ctx: weavecontext.SetCurrentState(ctx, state),
		}
	}
}

func (m *LaunchingNewMinitiaLoading) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[LaunchState](m, msg); handled {
		return model, cmd
	}

	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		m.Ctx = m.Loading.EndContext
		state := weavecontext.PushPageAndGetState[LaunchState](m)

		artifactsConfigJsonDir, err := weavecontext.GetMinitiaArtifactsConfigJson(m.Ctx)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		artifactsJsonDir, err := weavecontext.GetMinitiaArtifactsJson(m.Ctx)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		state.weave.PushPreviousResponse(
			styles.RenderPreviousResponse(
				styles.NoSeparator,
				fmt.Sprintf("New rollup has been launched. (More details about your rollup in %s and %s)", artifactsJsonDir, artifactsConfigJsonDir),
				[]string{artifactsJsonDir, artifactsConfigJsonDir},
				"",
			),
		)

		var jsonRpc string
		if state.vmType == string(EVM) {
			jsonRpc = DefaultMinitiaJsonRPC
		}

		payload := &ScanPayload{
			Vm:          strings.ToLower(state.vmType),
			ChainId:     state.chainId,
			MinGasPrice: 0,
			Denom:       state.gasDenom,
			Lcd:         DefaultMinitiaLCD,
			Rpc:         DefaultMinitiaRPC,
			JsonRpc:     jsonRpc,
		}

		encodedPayload, err := payload.EncodeToBase64()
		if err != nil {
			return m, m.HandlePanic(fmt.Errorf("failed to encode payload: %v", err))
		}

		state.scanLink = fmt.Sprintf("%s/custom-network/add/link?config=%s", InitiaScanURL, encodedPayload)
		scanText := fmt.Sprintf(
			"\n✨ %s 🪄 (We already started the rollup app for you)\n",
			styles.BoldText("Explore your new rollup here", styles.White),
		)

		state.weave.PushPreviousResponse(scanText)

		srv, err := service.NewService(service.Minitia)
		if err != nil {
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.NoSeparator, "Invalid OS: only Linux and Darwin are supported", []string{}, fmt.Sprintf("%v", err)))
		}

		if err = srv.Start(); err != nil {
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.NoSeparator, "Failed to start rollup service", []string{}, fmt.Sprintf("%v", err)))
		}

		if state.feeWhitelistAccounts != "" {
			time.Sleep(1 * time.Second)

			cache := make(map[string]bool)
			for _, acc := range strings.Split(state.feeWhitelistAccounts, ",") {
				cache[acc] = true
			}

			userHome, err := os.UserHomeDir()
			if err != nil {
				return m, m.HandlePanic(fmt.Errorf("failed to get user home directory: %v", err))
			}
			messageJsonPath := filepath.Join(userHome, common.WeaveDataDirectory, "messages.json")

			params, err := cosmosutils.QueryOPChildParams("http://localhost:1317")
			if err != nil {
				return m, m.HandlePanic(fmt.Errorf("failed to query params: %v", err))

			}
			for _, acc := range params.FeeWhitelist {
				cache[acc] = true
			}

			updatedFeeWhitelistAccounts := make([]string, 0)
			for acc := range cache {
				updatedFeeWhitelistAccounts = append(updatedFeeWhitelistAccounts, acc)
			}
			params.FeeWhitelist = updatedFeeWhitelistAccounts

			err = cosmosutils.CreateOPChildUpdateParamsMsg(messageJsonPath, params)
			if err != nil {
				return m, m.HandlePanic(fmt.Errorf("failed to create update params message: %v", err))

			}

			runCmd := exec.Command(state.binaryPath, "tx", "opchild", "execute-messages", messageJsonPath,
				"--from", "Validator", "--keyring-backend", "test",
				"--chain-id", state.chainId, "-y",
			)
			if err := runCmd.Run(); err != nil {
				return m, m.HandlePanic(fmt.Errorf("failed to update params message: %v", err))
			}
		}

		return NewTerminalState(weavecontext.SetCurrentState(m.Ctx, state)), tea.Quit
	}
	return m, cmd
}

func (m *LaunchingNewMinitiaLoading) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" + m.Loading.View() + "\n" + strings.Join(*m.streamingLogs, "\n"))
}

type TerminalState struct {
	weavecontext.BaseModel
}

func NewTerminalState(ctx context.Context) *TerminalState {
	analytics.TrackCompletedEvent(analytics.RollupLaunchFeature)
	return &TerminalState{
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
	}
}

func (m *TerminalState) Init() tea.Cmd {
	return nil
}

func (m *TerminalState) Update(_ tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *TerminalState) View() string {
	state := weavecontext.GetCurrentState[LaunchState](m.Ctx)
	return m.WrapView(state.weave.Render()) + state.scanLink + fmt.Sprintf(
		"\n\n%s %s\n%s\n",
		styles.Text("i", styles.Yellow),
		styles.BoldUnderlineText("Important", styles.Yellow),
		styles.Text("Open this in Chrome is recommended because some browsers may not support localhost access from a different host, or edit your browser's settings to allow it if necessary.", styles.Yellow),
	)
}
