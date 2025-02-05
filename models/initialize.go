package models

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/config"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/crypto"
	"github.com/initia-labs/weave/styles"
	"github.com/initia-labs/weave/types"
	"github.com/initia-labs/weave/ui"
)

type ExistingCheckerState struct {
	weave             types.WeaveState
	isFirstTime       bool
	generatedMnemonic string
}

func NewExistingCheckerState() ExistingCheckerState {
	return ExistingCheckerState{
		weave:       types.NewWeaveState(),
		isFirstTime: config.IsFirstTimeSetup(),
	}
}

func (e ExistingCheckerState) Clone() ExistingCheckerState {
	return ExistingCheckerState{
		weave:             e.weave.Clone(),
		isFirstTime:       e.isFirstTime,
		generatedMnemonic: e.generatedMnemonic,
	}

}

func InitHeader(isFirstTime bool) string {
	if isFirstTime {
		return styles.FadeText("Welcome to Weave! 🪢\n\n") +
			styles.RenderPrompt("It seems you haven't setup a Gas Station account yet.\nThis account is essential for holding funds to be distributed to OPinit Bots, IBC Relayers, and other services for gas fees.\n\n", []string{"Gas Station account"}, styles.Empty) +
			styles.BoldText("Please note that Weave will not send any transactions without your confirmation.\n", styles.Yellow) +
			styles.Text("Setting up a Gas Station account is required to ensure a smooth and seamless experience.\n", styles.Gray)
	} else {
		return styles.FadeText("Welcome to Weave! 🪢\n\n") +
			styles.RenderPrompt("Since you've previously set up your Gas Station account, you have the option to override it with a new one.\nThis account will continue to hold the necessary funds for the OPinit-bots or relayer to send transactions.\n\n", []string{"Gas Station account"}, styles.Empty) +
			styles.BoldText("Please remember, Weave will only send transactions after your confirmation.\n", styles.Yellow)
	}
}

type GasStationMethodSelect struct {
	weavecontext.BaseModel
	ui.Selector[GasStationMethodOption]
	firstTime bool
}

type GasStationMethodOption string

const (
	Generate GasStationMethodOption = "Generate new account (recommended)"
	Import   GasStationMethodOption = "Import existing account using mnemonic"
)

func NewGasStationMethodSelect(ctx context.Context) *GasStationMethodSelect {
	tooltip := ui.NewTooltipSlice(ui.NewTooltip(
		"Gas Station",
		"Gas Station account is the account from which Weave will use to fund necessary system accounts, enabling easier and more seamless experience while setting up things using Weave.",
		"** Weave will NOT automatically send transactions without asking for your confirmation. **",
		[]string{}, []string{}, []string{}), 2)
	return &GasStationMethodSelect{
		Selector: ui.Selector[GasStationMethodOption]{
			Options: []GasStationMethodOption{
				Generate,
				Import,
			},
			Tooltips:   &tooltip,
			CannotBack: true,
		},
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		firstTime: config.IsFirstTimeSetup(),
	}
}

func (m *GasStationMethodSelect) Init() tea.Cmd {
	return nil
}

func (m *GasStationMethodSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[ExistingCheckerState](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[ExistingCheckerState](m)

		state.weave.PushPreviousResponse(
			styles.RenderPreviousResponse(styles.ArrowSeparator, "How would you like to setup your Gas station account?", []string{"Gas Station account"}, string(*selected)),
		)
		switch *selected {
		case Generate:
			analytics.TrackEvent(analytics.GasStationMethodSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "generate"))
			model := NewGenerateGasStationLoading(weavecontext.SetCurrentState(m.Ctx, state))
			return model, model.Init()
		case Import:
			analytics.TrackEvent(analytics.GasStationMethodSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "import"))
			return NewGasStationMnemonicInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
		}
	}
	return m, cmd
}

func (m *GasStationMethodSelect) View() string {
	state := weavecontext.GetCurrentState[ExistingCheckerState](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	return m.WrapView(InitHeader(state.isFirstTime) + state.weave.Render() +
		styles.RenderPrompt("How would you like to setup your Gas station account?", []string{"Gas Station account"}, styles.Question) +
		m.Selector.View())
}

type GenerateGasStationLoading struct {
	ui.Loading
	weavecontext.BaseModel
}

func NewGenerateGasStationLoading(ctx context.Context) *GenerateGasStationLoading {
	return &GenerateGasStationLoading{
		Loading:   ui.NewLoading("Generating new Gas Station account...", generateGasStationAccount(ctx)),
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
	}
}

func (m *GenerateGasStationLoading) Init() tea.Cmd {
	return m.Loading.Init()
}

func generateGasStationAccount(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		state := weavecontext.GetCurrentState[ExistingCheckerState](ctx)

		mnemonic, err := crypto.GenerateMnemonic()
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to generate gas station mnemonic: %w", err)}
		}
		state.generatedMnemonic = mnemonic

		time.Sleep(1500 * time.Millisecond)

		return ui.EndLoading{
			Ctx: weavecontext.SetCurrentState(ctx, state),
		}
	}
}

func (m *GenerateGasStationLoading) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[ExistingCheckerState](m, msg); handled {
		return model, cmd
	}

	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		m.Ctx = m.Loading.EndContext
		state := weavecontext.PushPageAndGetState[ExistingCheckerState](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.NoSeparator, "Gas Station account has been successfully generated.", []string{}, ""))
		model := NewSystemKeysMnemonicDisplayInput(weavecontext.SetCurrentState(m.Ctx, state))
		return model, model.Init()
	}
	return m, cmd
}

func (m *GenerateGasStationLoading) View() string {
	state := weavecontext.GetCurrentState[ExistingCheckerState](m.Ctx)
	return m.WrapView(InitHeader(state.isFirstTime) + "\n" + state.weave.Render() + "\n" + m.Loading.View())
}

type GasStationMnemonicDisplayInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question string
}

func NewSystemKeysMnemonicDisplayInput(ctx context.Context) *GasStationMnemonicDisplayInput {
	model := &GasStationMnemonicDisplayInput{
		TextInput: ui.NewTextInput(true),
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		question:  "Type `continue` to proceed.",
	}
	model.WithPlaceholder("Type `continue` to continue, Ctrl+C to quit.")
	model.WithValidatorFn(common.ValidateExactString("continue"))
	return model
}

func (m *GasStationMnemonicDisplayInput) GetQuestion() string {
	return m.question
}

func (m *GasStationMnemonicDisplayInput) Init() tea.Cmd {
	return nil
}

func (m *GasStationMnemonicDisplayInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[ExistingCheckerState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[ExistingCheckerState](m)
		model := NewWeaveAppInitialization(m.Ctx, state.generatedMnemonic)
		return model, model.Init()
	}
	m.TextInput = input
	return m, cmd
}

func (m *GasStationMnemonicDisplayInput) View() string {
	state := weavecontext.GetCurrentState[ExistingCheckerState](m.Ctx)
	gasStationAddress, err := crypto.MnemonicToBech32Address("init", state.generatedMnemonic)
	if err != nil {
		m.HandlePanic(fmt.Errorf("failed to convert mnemonic to bech32 address: %w", err))
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		m.HandlePanic(fmt.Errorf("failed to get user home directory: %w", err))
	}

	var mnemonicText string
	mnemonicText += styles.RenderKey("Gas Station", gasStationAddress)

	return m.WrapView(InitHeader(state.isFirstTime) + "\n" + state.weave.Render() + "\n" +
		styles.BoldUnderlineText("Important", styles.Yellow) + "\n" +
		styles.Text(fmt.Sprintf("Note that the mnemonic phrase for Gas Station along with other configuration details will be stored in %s. You can revisit them anytime.", filepath.Join(userHome, common.WeaveConfigFile)), styles.Yellow) + "\n\n" +
		mnemonicText + styles.RenderPrompt(m.GetQuestion(), []string{"`continue`"}, styles.Question) + m.TextInput.View())
}

type GasStationMnemonicInput struct {
	weavecontext.BaseModel
	ui.TextInput
}

func NewGasStationMnemonicInput(ctx context.Context) *GasStationMnemonicInput {
	tooltip := ui.NewTooltip(
		"Gas station account",
		"Gas Station account is the account from which Weave will use to fund necessary system accounts, enabling easier and more seamless experience while setting up things using Weave.",
		"** Weave will NOT automatically send transactions without asking for your confirmation. **",
		[]string{}, []string{}, []string{})
	model := &GasStationMnemonicInput{
		TextInput: ui.NewTextInput(true),
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
	}
	model.WithPlaceholder("Add mnemonic")
	model.WithValidatorFn(common.ValidateMnemonic)
	model.WithTooltip(&tooltip)
	return model
}

func (m *GasStationMnemonicInput) Init() tea.Cmd {
	return nil
}

func (m *GasStationMnemonicInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[ExistingCheckerState](m, msg); handled {
		return model, cmd
	}
	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[ExistingCheckerState](m)

		state.weave.PushPreviousResponse(
			styles.RenderPreviousResponse(styles.DotsSeparator, "Please set up a Gas Station account", []string{"Gas Station account"}, styles.HiddenMnemonicText),
		)
		model := NewWeaveAppInitialization(weavecontext.SetCurrentState(m.Ctx, state), strings.Trim(input.Text, "\n"))
		return model, model.Init()
	}
	m.TextInput = input
	return m, cmd
}

func (m *GasStationMnemonicInput) View() string {
	state := weavecontext.GetCurrentState[ExistingCheckerState](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(InitHeader(state.isFirstTime) + "\n" + state.weave.Render() + styles.RenderPrompt("Please set up a Gas Station account", []string{"Gas Station account"}, styles.Question) + m.TextInput.View())
}

type WeaveAppInitialization struct {
	weavecontext.BaseModel
	ui.Loading
	mnemonic string
}

func NewWeaveAppInitialization(ctx context.Context, mnemonic string) tea.Model {
	return &WeaveAppInitialization{
		Loading:  ui.NewLoading("Initializing Weave...", WaitSetGasStation(mnemonic)),
		mnemonic: mnemonic,
		BaseModel: weavecontext.BaseModel{
			Ctx:        ctx,
			CannotBack: true,
		},
	}
}

func (hi *WeaveAppInitialization) Init() tea.Cmd {
	return hi.Loading.Init()
}

func WaitSetGasStation(mnemonic string) tea.Cmd {
	return func() tea.Msg {
		err := config.SetConfig("common.gas_station_mnemonic", mnemonic)
		if err != nil {
			return ui.ErrorLoading{Err: err}
		}
		time.Sleep(1500 * time.Millisecond)
		return ui.EndLoading{}
	}
}

func (hi *WeaveAppInitialization) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	loader, cmd := hi.Loading.Update(msg)
	hi.Loading = loader
	if hi.Loading.Completing {
		model := NewWeaveAppSettingUpGasStation(hi.Ctx)
		return model, model.Init()
	}
	return hi, cmd
}

func (hi *WeaveAppInitialization) View() string {
	state := weavecontext.GetCurrentState[ExistingCheckerState](hi.Ctx)
	return hi.WrapView(state.weave.Render() + "\n" + hi.Loading.View())
}

type WeaveAppSettingUpGasStation struct {
	weavecontext.BaseModel
	ui.Loading
}

func NewWeaveAppSettingUpGasStation(ctx context.Context) tea.Model {
	return &WeaveAppSettingUpGasStation{
		BaseModel: weavecontext.BaseModel{
			Ctx:        ctx,
			CannotBack: true,
		},
		Loading: ui.NewLoading("Setting up Gas Station account...", ui.DefaultWait()),
	}
}

func (hi *WeaveAppSettingUpGasStation) Init() tea.Cmd {
	return hi.Loading.Init()
}

func (hi *WeaveAppSettingUpGasStation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	loader, cmd := hi.Loading.Update(msg)
	hi.Loading = loader
	if hi.Loading.Completing {
		model := NewWeaveAppSuccessfullyInitialized(hi.Ctx)
		return model, model.Init()
	}
	return hi, cmd
}

func (hi *WeaveAppSettingUpGasStation) View() string {
	state := weavecontext.GetCurrentState[ExistingCheckerState](hi.Ctx)
	return hi.WrapView(state.weave.Render() + "\n" + hi.Loading.View() + "\n")
}

type WeaveAppSuccessfullyInitialized struct {
	weavecontext.BaseModel
	loadingEnded bool
}

func NewWeaveAppSuccessfullyInitialized(ctx context.Context) tea.Model {
	return &WeaveAppSuccessfullyInitialized{
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
	}
}

func (hi *WeaveAppSuccessfullyInitialized) Init() tea.Cmd {
	return ui.DefaultWait()
}

func (hi *WeaveAppSuccessfullyInitialized) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case ui.EndLoading:
		hi.loadingEnded = true // Set loadingEnded to true
		return hi, tea.Quit
	}
	return hi, nil
}

func (hi *WeaveAppSuccessfullyInitialized) View() string {
	if hi.loadingEnded {
		return ""
	}
	state := weavecontext.GetCurrentState[ExistingCheckerState](hi.Ctx)
	return hi.WrapView(state.weave.Render() + "\n" + styles.FadeText("Initia is a network for interwoven rollups ") + "🪢\n")
}
