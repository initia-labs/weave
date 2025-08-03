package relayer

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/client"
	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/config"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/cosmosutils"
	weaveio "github.com/initia-labs/weave/io"
	"github.com/initia-labs/weave/registry"
	"github.com/initia-labs/weave/styles"
	"github.com/initia-labs/weave/tooltip"
	"github.com/initia-labs/weave/types"
	"github.com/initia-labs/weave/ui"
)

var defaultL2ConfigLocal = []*Field{
	{Name: "l2.rpc_address", Type: StringField, Question: "Specify rollup RPC endpoint", Highlights: []string{"rollup RPC endpoint"}, Placeholder: `Press tab to use "http://localhost:26657"`, DefaultValue: "http://localhost:26657", ValidateFn: common.ValidateURL, Tooltip: &tooltip.RollupRPCEndpointTooltip},
	{Name: "l2.lcd_address", Type: StringField, Question: "Specify rollup REST endpoint", Highlights: []string{"rollup REST endpoint"}, Placeholder: `Press tab to use "http://localhost:1317"`, DefaultValue: "http://localhost:1317", ValidateFn: common.ValidateURLWithPort, Tooltip: &tooltip.RollupRESTEndpointTooltip},
}

var defaultL2ConfigManual = []*Field{
	{Name: "l2.chain_id", Type: StringField, Question: "Specify rollup chain ID", Highlights: []string{"rollup chain ID"}, Placeholder: "ex. rollup-1", ValidateFn: common.ValidateEmptyString, Tooltip: &tooltip.RollupChainIdTooltip},
	{Name: "l2.rpc_address", Type: StringField, Question: "Specify rollup RPC endpoint", Highlights: []string{"rollup RPC endpoint"}, Placeholder: "ex. http://localhost:26657", ValidateFn: common.ValidateURL, Tooltip: &tooltip.RollupRPCEndpointTooltip},
	{Name: "l2.lcd_address", Type: StringField, Question: "Specify rollup REST endpoint", Highlights: []string{"rollup REST endpoint"}, Placeholder: "ex. http://localhost:1317", ValidateFn: common.ValidateURLWithPort, Tooltip: &tooltip.RollupRESTEndpointTooltip},
	{Name: "l2.gas_price.denom", Type: StringField, Question: "Specify rollup gas denom", Highlights: []string{"rollup gas denom"}, Placeholder: "ex. umin", ValidateFn: common.ValidateDenom, Tooltip: &tooltip.RollupGasDenomTooltip},
	{Name: "l2.gas_price.price", Type: StringField, Question: "Specify rollup gas price", Highlights: []string{"rollup gas price"}, Placeholder: "ex. 0.15", ValidateFn: common.ValidateDecFromStr, Tooltip: &tooltip.RollupGasPriceTooltip},
}

type RollupSelect struct {
	ui.Selector[RollupSelectOption]
	weavecontext.BaseModel
	question string
}

type RollupSelectOption string

const (
	Whitelisted RollupSelectOption = "Whitelisted Rollup"
	Manual      RollupSelectOption = "Manual Relayer Setup"
)

var Local RollupSelectOption = "Local Rollup"

func NewRollupSelect(ctx context.Context) (*RollupSelect, error) {
	options := make([]RollupSelectOption, 0)
	tooltips := make([]ui.Tooltip, 0)
	minitiaConfigPath, err := weavecontext.GetMinitiaArtifactsConfigJson(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load minitia artifacts config: %v", err)
	}
	if weaveio.FileOrFolderExists(minitiaConfigPath) {
		configData, err := os.ReadFile(minitiaConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read minitia config: %v", err)
		}

		var minitiaConfig types.MinitiaConfig
		err = json.Unmarshal(configData, &minitiaConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal minitia config: %v", err)
		}

		Local = RollupSelectOption(fmt.Sprintf("%s (%s)", Local, minitiaConfig.L2Config.ChainID))

		options = append(options, Local, Whitelisted, Manual)
		tooltips = append(tooltips,
			tooltip.RelayerRollupSelectLocalTooltip,
			tooltip.RelayerRollupSelectWhitelistedTooltip,
			tooltip.RelayerRollupSelectManualTooltip,
		)
	} else {
		options = append(options, Whitelisted, Manual)
		tooltips = append(tooltips,
			tooltip.RelayerRollupSelectWhitelistedTooltip,
			tooltip.RelayerRollupSelectManualTooltip,
		)
	}

	return &RollupSelect{
		Selector: ui.Selector[RollupSelectOption]{
			Options:    options,
			CannotBack: true,
			Tooltips:   &tooltips,
		},
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		question:  "Select the type of Interwoven rollup you want to relay",
	}, nil
}

func (m *RollupSelect) GetQuestion() string {
	return m.question
}

func (m *RollupSelect) Init() tea.Cmd {
	return nil
}

func (m *RollupSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[State](m)

		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{}, string(*selected)))
		switch *selected {
		case Whitelisted:
			analytics.TrackEvent(analytics.RelayerRollupSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "whitelist"))

			model, err := NewSelectingL1NetworkRegistry(weavecontext.SetCurrentState(m.Ctx, state))
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, nil
		case Local:
			analytics.TrackEvent(analytics.RelayerRollupSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "local"))

			minitiaConfigPath, err := weavecontext.GetMinitiaArtifactsConfigJson(m.Ctx)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			configData, err := os.ReadFile(minitiaConfigPath)
			if err != nil {
				return m, m.HandlePanic(err)
			}

			var minitiaConfig types.MinitiaConfig
			err = json.Unmarshal(configData, &minitiaConfig)
			if err != nil {
				return m, m.HandlePanic(err)
			}

			state.feeWhitelistAccounts = append(state.feeWhitelistAccounts, minitiaConfig.SystemKeys.Challenger.L2Address)

			state.minitiaConfig = &minitiaConfig
			switch minitiaConfig.L1Config.ChainID {
			case InitiaTestnetChainId:
				state.chainType = registry.InitiaL1Testnet
				testnetRegistry, err := registry.GetChainRegistry(registry.InitiaL1Testnet)
				if err != nil {
					return m, m.HandlePanic(err)
				}
				state.Config["l1.chain_id"] = testnetRegistry.GetChainId()
				if state.Config["l1.rpc_address"], err = testnetRegistry.GetFirstActiveRpc(); err != nil {
					return m, m.HandlePanic(err)
				}
				if state.Config["l1.lcd_address"], err = testnetRegistry.GetFirstActiveLcd(); err != nil {
					return m, m.HandlePanic(err)
				}
				if state.Config["l1.gas_price.price"], err = testnetRegistry.GetFixedMinGasPriceByDenom(DefaultGasPriceDenom); err != nil {
					return m, m.HandlePanic(err)
				}
				state.Config["l1.gas_price.denom"] = DefaultGasPriceDenom

				state.Config["l2.chain_id"] = minitiaConfig.L2Config.ChainID
				state.Config["l2.gas_price.denom"] = minitiaConfig.L2Config.Denom
				state.Config["l2.gas_price.price"] = DefaultGasPriceAmount
				state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, "L1 network is auto-detected", []string{}, minitiaConfig.L1Config.ChainID))

			case InitiaMainnetChainId:
				state.chainType = registry.InitiaL1Mainnet
				mainnetRegistry, err := registry.GetChainRegistry(registry.InitiaL1Mainnet)
				if err != nil {
					return m, m.HandlePanic(err)
				}
				state.Config["l1.chain_id"] = mainnetRegistry.GetChainId()
				if state.Config["l1.rpc_address"], err = mainnetRegistry.GetFirstActiveRpc(); err != nil {
					return m, m.HandlePanic(err)
				}
				if state.Config["l1.lcd_address"], err = mainnetRegistry.GetFirstActiveLcd(); err != nil {
					return m, m.HandlePanic(err)
				}
				if state.Config["l1.gas_price.price"], err = mainnetRegistry.GetFixedMinGasPriceByDenom(DefaultGasPriceDenom); err != nil {
					return m, m.HandlePanic(err)
				}
				state.Config["l1.gas_price.denom"] = DefaultGasPriceDenom

				state.Config["l2.chain_id"] = minitiaConfig.L2Config.ChainID
				state.Config["l2.gas_price.denom"] = minitiaConfig.L2Config.Denom
				state.Config["l2.gas_price.price"] = DefaultGasPriceAmount
				state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, "L1 network is auto-detected", []string{}, minitiaConfig.L1Config.ChainID))
			default:
				return m, m.HandlePanic(fmt.Errorf("not support L1"))
			}

			return NewFieldInputModel(weavecontext.SetCurrentState(m.Ctx, state), defaultL2ConfigLocal, NewSelectSettingUpIBCChannelsMethod), nil
		case Manual:
			analytics.TrackEvent(analytics.RelayerRollupSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "manual"))

			model, err := NewSelectingL1Network(weavecontext.SetCurrentState(m.Ctx, state))
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, nil
		}
		return m, tea.Quit
	}

	return m, cmd
}

func (m *RollupSelect) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(
		m.GetQuestion(),
		[]string{"Interwoven rollup"},
		styles.Question,
	) + m.Selector.View())
}

type L1KeySelect struct {
	ui.Selector[L1KeySelectOption]
	weavecontext.BaseModel
	question string
	chainId  string
}

type L1KeySelectOption string

const (
	L1GenerateKey L1KeySelectOption = "Generate new system key"
)

var (
	L1ExistingKey = L1KeySelectOption("Use an existing key " + styles.Text("(previously setup with Weave)", styles.Gray))
	L1ImportKey   = L1KeySelectOption("Import existing key " + styles.Text("(you will be prompted to enter your mnemonic)", styles.Gray))
)

func NewL1KeySelect(ctx context.Context) (*L1KeySelect, error) {
	l1ChainId, err := GetL1ChainId(ctx)
	if err != nil {
		return nil, fmt.Errorf("get l1 chain id: %w", err)
	}
	options := []L1KeySelectOption{
		L1GenerateKey,
		L1ImportKey,
	}
	state := weavecontext.GetCurrentState[State](ctx)
	// TODO: find a way or remove
	// if l1RelayerAddress, found := cosmosutils.GetHermesRelayerAddress(state.hermesBinaryPath, l1ChainId); found {
	// 	state.l1RelayerAddress = l1RelayerAddress
	// 	options = append([]L1KeySelectOption{L1ExistingKey}, options...)
	// }

	tooltips := ui.NewTooltipSlice(tooltip.RelayerL1KeySelectTooltip, len(options))

	return &L1KeySelect{
		Selector: ui.Selector[L1KeySelectOption]{
			Options:    options,
			CannotBack: true,
			Tooltips:   &tooltips,
		},
		BaseModel: weavecontext.BaseModel{Ctx: weavecontext.SetCurrentState(ctx, state), CannotBack: true},
		question:  fmt.Sprintf("Select an option for setting up the relayer account key on L1 (%s)", l1ChainId),
		chainId:   l1ChainId,
	}, nil
}

func (m *L1KeySelect) GetQuestion() string {
	return m.question
}

func (m *L1KeySelect) Init() tea.Cmd {
	return nil
}

func (m *L1KeySelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[State](m)

		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{"relayer account key", fmt.Sprintf("L1 (%s)", m.chainId)}, string(*selected)))
		state.l1KeyMethod = string(*selected)
		model, err := NewL2KeySelect(weavecontext.SetCurrentState(m.Ctx, state))
		if err != nil {
			return m, m.HandlePanic(err)
		}
		return model, nil
	}

	return m, cmd
}

func (m *L1KeySelect) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" + styles.InformationMark + styles.BoldText(
		"Relayer account keys with funds",
		styles.White,
	) + " are required to setup and run the relayer properly." + "\n" + styles.RenderPrompt(
		m.GetQuestion(),
		[]string{"relayer account key", fmt.Sprintf("L1 (%s)", m.chainId)},
		styles.Question,
	) + m.Selector.View())
}

type L2KeySelect struct {
	ui.Selector[L2KeySelectOption]
	weavecontext.BaseModel
	question string
	chainId  string
}

type L2KeySelectOption string

const (
	L2SameKey     L2KeySelectOption = "Use the same key with L1"
	L2GenerateKey L2KeySelectOption = "Generate new system key"
)

var (
	L2ExistingKey = L2KeySelectOption("Use an existing key " + styles.Text("(previously setup in Weave)", styles.Gray))
	L2ImportKey   = L2KeySelectOption("Import existing key " + styles.Text("(you will be prompted to enter your mnemonic)", styles.Gray))
)

func NewL2KeySelect(ctx context.Context) (*L2KeySelect, error) {
	l2ChainId, err := GetL2ChainId(ctx)
	if err != nil {
		return nil, fmt.Errorf("get l2 chain id: %w", err)
	}
	options := []L2KeySelectOption{
		L2SameKey,
		L2GenerateKey,
		L2ImportKey,
	}
	state := weavecontext.GetCurrentState[State](ctx)
	// TODO: find a way or remove
	// if l2RelayerAddress, found := cosmosutils.GetHermesRelayerAddress(state.hermesBinaryPath, l2ChainId); found {
	// 	state.l2RelayerAddress = l2RelayerAddress
	// 	options = append([]L2KeySelectOption{L2ExistingKey}, options...)
	// }

	tooltips := ui.NewTooltipSlice(tooltip.RelayerRollupKeySelectTooltip, len(options))

	return &L2KeySelect{
		Selector: ui.Selector[L2KeySelectOption]{
			Options:  options,
			Tooltips: &tooltips,
		},
		BaseModel: weavecontext.BaseModel{Ctx: weavecontext.SetCurrentState(ctx, state)},
		question:  fmt.Sprintf("Select an option for setting up the relayer account key on rollup (%s)", l2ChainId),
		chainId:   l2ChainId,
	}, nil
}

func (m *L2KeySelect) GetQuestion() string {
	return m.question
}

func (m *L2KeySelect) Init() tea.Cmd {
	return nil
}

func (m *L2KeySelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[State](m)

		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{"relayer account key", fmt.Sprintf("rollup (%s)", m.chainId)}, string(*selected)))
		state.l2KeyMethod = string(*selected)

		switch L1KeySelectOption(state.l1KeyMethod) {
		case L1ExistingKey:
			switch *selected {
			case L2ExistingKey:
				model := NewSettingUpRelayer(weavecontext.SetCurrentState(m.Ctx, state))
				return model, model.Init()
			case L2SameKey:
				state.l2RelayerAddress = state.l1RelayerAddress
				state.l2RelayerMnemonic = state.l1RelayerMnemonic
				model := NewSettingUpRelayer(weavecontext.SetCurrentState(m.Ctx, state))
				return model, model.Init()
			case L2GenerateKey:
				model := NewGenerateL2RelayerKeyLoading(weavecontext.SetCurrentState(m.Ctx, state))
				return model, model.Init()
			case L2ImportKey:
				return NewImportL2RelayerKeyInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
			}
		case L1GenerateKey:
			model := NewGenerateL1RelayerKeyLoading(weavecontext.SetCurrentState(m.Ctx, state))
			return model, model.Init()
		case L1ImportKey:
			return NewImportL1RelayerKeyInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
		}
	}

	return m, cmd
}

func (m *L2KeySelect) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(
		m.GetQuestion(),
		[]string{"relayer account key", fmt.Sprintf("rollup (%s)", m.chainId)},
		styles.Question,
	) + m.Selector.View())
}

type GenerateL1RelayerKeyLoading struct {
	ui.Loading
	weavecontext.BaseModel
}

func NewGenerateL1RelayerKeyLoading(ctx context.Context) *GenerateL1RelayerKeyLoading {
	state := weavecontext.GetCurrentState[State](ctx)
	layerText := "L1"
	if state.l1KeyMethod == string(L1GenerateKey) && state.l2KeyMethod == string(L2SameKey) {
		layerText = "L1 and rollup"
	}

	return &GenerateL1RelayerKeyLoading{
		Loading:   ui.NewLoading(fmt.Sprintf("Generating new relayer account key for %s ...", layerText), waitGenerateL1RelayerKeyLoading(ctx)),
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
	}
}

func (m *GenerateL1RelayerKeyLoading) Init() tea.Cmd {
	return m.Loading.Init()
}

func waitGenerateL1RelayerKeyLoading(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1500 * time.Millisecond)

		state := weavecontext.GetCurrentState[State](ctx)

		relayerKey, err := weaveio.GenerateKey("init")
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("could not generate hermes key: %s", err)}
		}
		state.l1RelayerAddress = relayerKey.Address
		state.l1RelayerMnemonic = relayerKey.Mnemonic

		return ui.EndLoading{
			Ctx: weavecontext.SetCurrentState(ctx, state),
		}
	}
}

func (m *GenerateL1RelayerKeyLoading) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		m.Ctx = m.Loading.EndContext
		state := weavecontext.PushPageAndGetState[State](m)

		switch L2KeySelectOption(state.l2KeyMethod) {
		case L2ExistingKey, L2ImportKey:
			model := NewKeysMnemonicDisplayInput(weavecontext.SetCurrentState(m.Ctx, state))
			return model, model.Init()
		case L2SameKey:
			state.l2RelayerAddress = state.l1RelayerAddress
			state.l2RelayerMnemonic = state.l1RelayerMnemonic
			model := NewKeysMnemonicDisplayInput(weavecontext.SetCurrentState(m.Ctx, state))
			return model, model.Init()
		case L2GenerateKey:
			model := NewGenerateL2RelayerKeyLoading(weavecontext.SetCurrentState(m.Ctx, state))
			return model, model.Init()
		}
	}
	return m, cmd
}

func (m *GenerateL1RelayerKeyLoading) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" + m.Loading.View())
}

type GenerateL2RelayerKeyLoading struct {
	ui.Loading
	weavecontext.BaseModel
}

func NewGenerateL2RelayerKeyLoading(ctx context.Context) *GenerateL2RelayerKeyLoading {
	return &GenerateL2RelayerKeyLoading{
		Loading:   ui.NewLoading("Generating new relayer account key for rollup...", waitGenerateL2RelayerKeyLoading(ctx)),
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
	}
}

func (m *GenerateL2RelayerKeyLoading) Init() tea.Cmd {
	return m.Loading.Init()
}

func waitGenerateL2RelayerKeyLoading(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1500 * time.Millisecond)

		state := weavecontext.GetCurrentState[State](ctx)

		relayerKey, err := weaveio.GenerateKey("init")
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("could not generate hermes key: %s", err)}
		}
		state.l2RelayerAddress = relayerKey.Address
		state.l2RelayerMnemonic = relayerKey.Mnemonic

		return ui.EndLoading{
			Ctx: weavecontext.SetCurrentState(ctx, state),
		}
	}
}

func (m *GenerateL2RelayerKeyLoading) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		m.Ctx = m.Loading.EndContext
		state := weavecontext.PushPageAndGetState[State](m)
		model := NewKeysMnemonicDisplayInput(weavecontext.SetCurrentState(m.Ctx, state))
		return model, model.Init()
	}
	return m, cmd
}

func (m *GenerateL2RelayerKeyLoading) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" + m.Loading.View())
}

type KeysMnemonicDisplayInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question string
}

func NewKeysMnemonicDisplayInput(ctx context.Context) *KeysMnemonicDisplayInput {
	model := &KeysMnemonicDisplayInput{
		TextInput: ui.NewTextInput(true),
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		question:  "Please type `continue` to proceed.",
	}
	model.WithPlaceholder("Type `continue` to continue, Ctrl+C to quit.")
	model.WithValidatorFn(common.ValidateExactString("continue"))
	return model
}

func (m *KeysMnemonicDisplayInput) GetQuestion() string {
	return m.question
}

func (m *KeysMnemonicDisplayInput) Init() tea.Cmd {
	return nil
}

func (m *KeysMnemonicDisplayInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[State](m)

		extraText := " has"
		if state.l1KeyMethod == string(L1GenerateKey) && state.l2KeyMethod == string(L2GenerateKey) {
			extraText = "s have"
		}
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.NoSeparator, fmt.Sprintf("Relayer key%s been successfully generated.", extraText), []string{}, ""))

		switch L2KeySelectOption(state.l2KeyMethod) {
		case L2ExistingKey, L2GenerateKey, L2SameKey:
			model := NewSettingUpRelayer(weavecontext.SetCurrentState(m.Ctx, state))
			return model, model.Init()
		case L2ImportKey:
			return NewImportL2RelayerKeyInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
		}
	}
	m.TextInput = input
	return m, cmd
}

func (m *KeysMnemonicDisplayInput) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	var mnemonicText string

	if state.l1KeyMethod == string(L1GenerateKey) {
		layerText := "L1"
		if state.l2KeyMethod == string(L2SameKey) {
			layerText = "L1 and rollup"
		}
		mnemonicText += styles.RenderKey(
			styles.RenderPrompt(fmt.Sprintf("Weave Relayer on %s", layerText), []string{layerText}, styles.Empty),
			state.l1RelayerAddress,
		)
	}

	if state.l2KeyMethod == string(L2GenerateKey) {
		if mnemonicText != "" {
			mnemonicText += "\n"
		}
		mnemonicText += styles.RenderKey(
			styles.RenderPrompt("Weave Relayer on L2", []string{"L2"}, styles.Empty),
			state.l2RelayerAddress,
		)
	}

	return m.WrapView(state.weave.Render() + "\n" +
		styles.BoldUnderlineText("Important", styles.Yellow) + "\n" +
		styles.Text(fmt.Sprintf("Note that the mnemonic phrases for Relayer will be stored in %s. You can revisit them anytime.", common.RelayerConfigPath), styles.Yellow) + "\n\n" +
		mnemonicText + styles.RenderPrompt(m.GetQuestion(), []string{"`continue`"}, styles.Question) + m.TextInput.View())
}

type ImportL1RelayerKeyInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question  string
	layerText string
}

func NewImportL1RelayerKeyInput(ctx context.Context) *ImportL1RelayerKeyInput {
	state := weavecontext.GetCurrentState[State](ctx)
	layerText := "L1"
	if state.l1KeyMethod == string(L1ImportKey) && state.l2KeyMethod == string(L2SameKey) {
		layerText = "L1 and rollup"
	}
	model := &ImportL1RelayerKeyInput{
		TextInput: ui.NewTextInput(false),
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  fmt.Sprintf("Please add mnemonic for relayer account key on %s", layerText),
		layerText: layerText,
	}
	model.WithPlaceholder("Enter the mnemonic")
	model.WithValidatorFn(common.ValidateMnemonic)
	return model
}

func (m *ImportL1RelayerKeyInput) GetQuestion() string {
	return m.question
}

func (m *ImportL1RelayerKeyInput) Init() tea.Cmd {
	return nil
}

func (m *ImportL1RelayerKeyInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[State](m)

		relayerKey, err := weaveio.RecoverKey("init", input.Text)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		state.l1RelayerMnemonic = relayerKey.Mnemonic
		state.l1RelayerAddress = relayerKey.Address
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"relayer account key", m.layerText}, styles.HiddenMnemonicText))

		switch L2KeySelectOption(state.l2KeyMethod) {
		case L2ExistingKey:
			model := NewSettingUpRelayer(weavecontext.SetCurrentState(m.Ctx, state))
			return model, model.Init()
		case L2SameKey:
			state.l2RelayerAddress = relayerKey.Address
			state.l2RelayerMnemonic = relayerKey.Mnemonic
			model := NewSettingUpRelayer(weavecontext.SetCurrentState(m.Ctx, state))
			return model, model.Init()
		case L2GenerateKey:
			model := NewGenerateL2RelayerKeyLoading(weavecontext.SetCurrentState(m.Ctx, state))
			return model, model.Init()
		case L2ImportKey:
			return NewImportL2RelayerKeyInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
		}
	}
	m.TextInput = input
	return m, cmd
}

func (m *ImportL1RelayerKeyInput) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"relayer account key", m.layerText}, styles.Question) + m.TextInput.View())
}

type ImportL2RelayerKeyInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question string
}

func NewImportL2RelayerKeyInput(ctx context.Context) *ImportL2RelayerKeyInput {
	model := &ImportL2RelayerKeyInput{
		TextInput: ui.NewTextInput(false),
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  "Please add mnemonic for relayer account key on L2",
	}
	model.WithPlaceholder("Enter the mnemonic")
	model.WithValidatorFn(common.ValidateMnemonic)
	return model
}

func (m *ImportL2RelayerKeyInput) GetQuestion() string {
	return m.question
}

func (m *ImportL2RelayerKeyInput) Init() tea.Cmd {
	return nil
}

func (m *ImportL2RelayerKeyInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[State](m)

		relayerKey, err := weaveio.RecoverKey("init", input.Text)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		state.l2RelayerMnemonic = relayerKey.Mnemonic
		state.l2RelayerAddress = relayerKey.Address
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"relayer account key", "L2"}, styles.HiddenMnemonicText))

		model := NewSettingUpRelayer(weavecontext.SetCurrentState(m.Ctx, state))
		return model, model.Init()
	}
	m.TextInput = input
	return m, cmd
}

func (m *ImportL2RelayerKeyInput) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"relayer account key", "L2"}, styles.Question) + m.TextInput.View())
}

type FetchingBalancesLoading struct {
	ui.Loading
	weavecontext.BaseModel
}

func NewFetchingBalancesLoading(ctx context.Context) *FetchingBalancesLoading {
	return &FetchingBalancesLoading{
		Loading:   ui.NewLoading("Fetching relayer account balances ...", waitFetchingBalancesLoading(ctx)),
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
	}
}

func (m *FetchingBalancesLoading) Init() tea.Cmd {
	return m.Loading.Init()
}

func waitFetchingBalancesLoading(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		state := weavecontext.GetCurrentState[State](ctx)

		l1Rest, err := GetL1ActiveLcd(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("cannot load l1 active lcd: %w", err)}
		}
		l1Balances, err := cosmosutils.QueryBankBalances([]string{l1Rest}, state.l1RelayerAddress)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("cannot fetch balance for l1: %v", err)}
		}
		state.l1NeedsFunding = l1Balances.IsZero()

		l2Rest, err := GetL2ActiveLcd(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("cannot load l2 active lcd: %w", err)}
		}
		l2Balances, err := cosmosutils.QueryBankBalances([]string{l2Rest}, state.l2RelayerAddress)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("cannot fetch balance for l2: %v", err)}
		}
		state.l2NeedsFunding = l2Balances.IsZero()
		if slices.Contains(state.feeWhitelistAccounts, state.l2RelayerAddress) {
			state.l2NeedsFunding = false
		}

		return ui.EndLoading{
			Ctx: weavecontext.SetCurrentState(ctx, state),
		}
	}
}

func (m *FetchingBalancesLoading) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		m.Ctx = m.Loading.EndContext
		state := weavecontext.PushPageAndGetState[State](m)

		if !state.l1NeedsFunding && !state.l2NeedsFunding {
			return NewTerminalState(weavecontext.SetCurrentState(m.Ctx, state)), tea.Quit
		}

		model, err := NewFundingAmountSelect(weavecontext.SetCurrentState(m.Ctx, state))
		if err != nil {
			return m, m.HandlePanic(err)
		}
		return model, nil
	}
	return m, cmd
}

func (m *FetchingBalancesLoading) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" + m.Loading.View())
}

type FundingAmountSelect struct {
	ui.Selector[FundingAmountSelectOption]
	weavecontext.BaseModel
	question string
}

type FundingAmountSelectOption string

const (
	FundingFillManually FundingAmountSelectOption = "○ Fill in an amount manually to fund from Gas Station Account"
	FundingUserTransfer FundingAmountSelectOption = "○ Skip funding from Gas station"
)

var FundingDefaultPreset FundingAmountSelectOption = ""

func NewFundingAmountSelect(ctx context.Context) (*FundingAmountSelect, error) {
	state := weavecontext.GetCurrentState[State](ctx)
	l1ChainId, err := GetL1ChainId(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get l1 chain id: %w", err)
	}
	l1GasDenom, err := GetL1GasDenom(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get l1 gas denom: %w", err)
	}
	l2ChainId, err := GetL2ChainId(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get l2 chain id: %w", err)
	}
	l2GasDenom, err := GetL2GasDenom(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get l2 gas denom: %w", err)
	}
	FundingDefaultPreset = FundingAmountSelectOption(fmt.Sprintf(
		"○ Use the default preset\n    Total amount that will be transferred from Gas Station account:\n    %s %s on L1 %s\n    %s %s on L2 %s",
		styles.BoldText(fmt.Sprintf("• L1 (%s):", l1ChainId), styles.Cyan),
		styles.BoldText(fmt.Sprintf("%s%s", DefaultL1RelayerBalance, l1GasDenom), styles.White),
		styles.Text(fmt.Sprintf("(%s)", state.l1RelayerAddress), styles.Gray),
		styles.BoldText(fmt.Sprintf("• Rollup (%s):", l2ChainId), styles.Cyan),
		styles.BoldText(fmt.Sprintf("%s%s", DefaultL2RelayerBalance, l2GasDenom), styles.White),
		styles.Text(fmt.Sprintf("(%s)", state.l2RelayerAddress), styles.Gray),
	))
	options := []FundingAmountSelectOption{
		FundingDefaultPreset,
		FundingFillManually,
		FundingUserTransfer,
	}
	tooltips := ui.NewTooltipSlice(
		tooltip.RelayerFundingAmountSelectTooltip, len(options),
	)
	return &FundingAmountSelect{
		Selector: ui.Selector[FundingAmountSelectOption]{
			Options:    options,
			CannotBack: true,
			Tooltips:   &tooltips,
		},
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		question:  "Select the filling amount option",
	}, nil
}

func (m *FundingAmountSelect) GetQuestion() string {
	return m.question
}

func (m *FundingAmountSelect) Init() tea.Cmd {
	return nil
}

func (m *FundingAmountSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[State](m)

		switch *selected {
		case FundingDefaultPreset:
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{}, "Use the default preset"))
			state.l1FundingAmount = DefaultL1RelayerBalance
			state.l2FundingAmount = DefaultL2RelayerBalance
			model, err := NewFundDefaultPresetConfirmationInput(weavecontext.SetCurrentState(m.Ctx, state))
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, nil
		case FundingFillManually:
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{}, "Fill in an amount manually to fund from Gas Station Account"))
			model, err := NewFundManuallyL1BalanceInput(weavecontext.SetCurrentState(m.Ctx, state))
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, nil
		case FundingUserTransfer:
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{}, "Transfer funds manually from other account"))
			state.weave.PushPreviousResponse(fmt.Sprintf(
				"%s %s\n  %s\n%s\n\n",
				styles.Text("i", styles.Yellow),
				styles.BoldUnderlineText("Important", styles.Yellow),
				styles.Text("To ensure the relayer functions properly, make sure these accounts are funded.", styles.Yellow),
				styles.CreateFrame(fmt.Sprintf(
					"%s %s    \n%s %s",
					styles.BoldText("• Relayer key on L1", styles.White),
					styles.Text(fmt.Sprintf("(%s)", state.l1RelayerAddress), styles.Gray),
					styles.BoldText("• Relayer key on rollup", styles.White),
					styles.Text(fmt.Sprintf("(%s)", state.l2RelayerAddress), styles.Gray),
				), 69),
			))
			return NewTerminalState(weavecontext.SetCurrentState(m.Ctx, state)), tea.Quit
		}
	}

	return m, cmd
}

func (m *FundingAmountSelect) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)

	var informationLayer, warningLayer string
	if state.l1NeedsFunding && state.l2NeedsFunding {
		informationLayer = "both L1 and rollup"
		warningLayer = "L1 and rollup have"
	} else if state.l1NeedsFunding {
		informationLayer = "L1"
		warningLayer = "L1 has"
	} else if state.l2NeedsFunding {
		informationLayer = "Rollup"
		warningLayer = "Rollup has"
	}

	return m.WrapView(state.weave.Render() + "\n" +
		styles.RenderPrompt(
			fmt.Sprintf("You will need to fund the relayer account on %s.\n  You can either transfer funds from created Gas Station Account or transfer manually.", informationLayer),
			[]string{informationLayer},
			styles.Information,
		) + "\n\n" +
		styles.BoldUnderlineText("Important", styles.Yellow) + "\n" +
		styles.Text(fmt.Sprintf("The relayer account on %s no funds.\nYou will need to fund the account in order to run the relayer properly.", warningLayer), styles.Yellow) + "\n\n" +
		styles.RenderPrompt(
			m.GetQuestion(),
			[]string{},
			styles.Question,
		) + m.Selector.View())
}

type FundDefaultPresetConfirmationInput struct {
	ui.TextInput
	weavecontext.BaseModel
	initiaGasStationAddress string
	question                string
	err                     error
}

func NewFundDefaultPresetConfirmationInput(ctx context.Context) (*FundDefaultPresetConfirmationInput, error) {
	gasStationKey, err := config.GetGasStationKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get gas station key: %v", err)
	}
	model := &FundDefaultPresetConfirmationInput{
		TextInput:               ui.NewTextInput(false),
		BaseModel:               weavecontext.BaseModel{Ctx: ctx},
		initiaGasStationAddress: gasStationKey.InitiaAddress,
		question:                "Confirm to proceed with signing and broadcasting the following transactions? [y]:",
	}
	model.WithPlaceholder("Type `y` to confirm")
	model.WithValidatorFn(common.ValidateExactString("y"))
	return model, nil
}

func (m *FundDefaultPresetConfirmationInput) GetQuestion() string {
	return m.question
}

func (m *FundDefaultPresetConfirmationInput) Init() tea.Cmd {
	return nil
}

func (m *FundDefaultPresetConfirmationInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.GetCurrentState[State](m.Ctx)

		// Check gas station balances
		gasStationKey, err := config.GetGasStationKey()
		if err != nil {
			return nil, m.HandlePanic(fmt.Errorf("failed to get gas station key: %v", err))
		}

		// Check L1 balance if needed
		if state.l1FundingAmount != "0" {
			l1Rest, err := GetL1ActiveLcd(m.Ctx)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			l1Balances, err := cosmosutils.QueryBankBalances([]string{l1Rest}, gasStationKey.InitiaAddress)
			if err != nil {
				return m, m.HandlePanic(err)
			}

			l1Required := new(big.Int)
			if _, ok := l1Required.SetString(state.l1FundingAmount, 10); !ok {
				return m, m.HandlePanic(fmt.Errorf("invalid L1 funding amount: %s", state.l1FundingAmount))
			}

			// Get the available balance for the default denom
			l1GasDenom, err := GetL1GasDenom(m.Ctx)
			if err != nil {
				m.HandlePanic(err)
			}
			l1Available := new(big.Int)
			for _, coin := range *l1Balances {
				if coin.Denom == l1GasDenom {
					if _, ok := l1Available.SetString(coin.Amount, 10); !ok {
						return m, m.HandlePanic(fmt.Errorf("invalid L1 balance amount: %s", coin.Amount))
					}
				}
			}
			if l1Available.Cmp(l1Required) < 0 {
				m.err = fmt.Errorf("insufficient balance in gas station on L1. Required: %s%s, Available: %s%s",
					l1Required.String(), l1GasDenom, l1Available.String(), l1GasDenom)
				return m, cmd
			}
		}

		// Check L2 balance if needed
		if state.l2FundingAmount != "0" {
			l2Rest, err := GetL2ActiveLcd(m.Ctx)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			l2Balances, err := cosmosutils.QueryBankBalances([]string{l2Rest}, gasStationKey.InitiaAddress)
			if err != nil {
				return m, m.HandlePanic(err)
			}

			l2Required := new(big.Int)
			if _, ok := l2Required.SetString(state.l2FundingAmount, 10); !ok {
				return m, m.HandlePanic(fmt.Errorf("invalid L2 funding amount: %s", state.l2FundingAmount))
			}

			// Get the gas denom for L2
			l2GasDenom, err := GetL2GasDenom(m.Ctx)
			if err != nil {
				return m, m.HandlePanic(fmt.Errorf("failed to get L2 gas denom: %w", err))
			}

			// Find the balance for the required denom
			l2Available := new(big.Int)
			for _, coin := range *l2Balances {
				if coin.Denom == l2GasDenom {
					if _, ok := l2Available.SetString(coin.Amount, 10); !ok {
						return m, m.HandlePanic(fmt.Errorf("invalid L2 balance amount: %s", coin.Amount))
					}
					break
				}
			}

			if l2Available.Cmp(l2Required) < 0 {
				m.err = fmt.Errorf("insufficient balance in gas station on L2. Required: %s%s, Available: %s%s",
					l2Required.String(), l2GasDenom, l2Available.String(), l2GasDenom)
				return m, cmd
			}
		}

		state = weavecontext.PushPageAndGetState[State](m)
		model := NewFundDefaultPresetBroadcastLoading(weavecontext.SetCurrentState(m.Ctx, state))
		return model, model.Init()
	}
	m.TextInput = input
	return m, cmd
}

func (m *FundDefaultPresetConfirmationInput) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	l1GasDenom, err := GetL1GasDenom(m.Ctx)
	if err != nil {
		m.HandlePanic(err)
	}
	l2GasDenom, err := GetL2GasDenom(m.Ctx)
	if err != nil {
		m.HandlePanic(err)
	}
	formatSendMsg := func(coins, denom, keyName, address string) string {
		return fmt.Sprintf(
			"> Send %s to %s %s\n",
			styles.BoldText(coins+denom, styles.Ivory),
			styles.BoldText(keyName, styles.Ivory),
			styles.Text(fmt.Sprintf("(%s)", address), styles.Gray))
	}
	l1FundingText := map[bool]string{
		true: "",
		false: fmt.Sprintf("Sending tokens from the Gas Station account on L1 %s ⛽️\n", styles.Text(fmt.Sprintf("(%s)", m.initiaGasStationAddress), styles.Gray)) +
			formatSendMsg(state.l1FundingAmount, l1GasDenom, "Relayer key on L1", state.l1RelayerAddress) + "\n",
	}
	l2FundingText := map[bool]string{
		true: "",
		false: fmt.Sprintf("Sending tokens from the Gas Station account on L2 %s ⛽️\n", styles.Text(fmt.Sprintf("(%s)", m.initiaGasStationAddress), styles.Gray)) +
			formatSendMsg(state.l2FundingAmount, l2GasDenom, "Relayer key on L2", state.l2RelayerAddress),
	}

	textInputView := ""
	if m.err != nil {
		textInputView = m.TextInput.ViewErr(m.err)
	} else {
		textInputView = m.TextInput.View()
	}

	return m.WrapView(state.weave.Render() + "\n" +
		styles.Text("i ", styles.Yellow) +
		styles.RenderPrompt(
			styles.BoldUnderlineText("Weave will now broadcast the following transactions", styles.Yellow),
			[]string{}, styles.Empty,
		) + "\n\n" +
		l1FundingText[state.l1FundingAmount == "0"] +
		l2FundingText[state.l2FundingAmount == "0"] +
		styles.RenderPrompt(m.GetQuestion(), []string{}, styles.Question) + textInputView)
}

type FundDefaultPresetBroadcastLoading struct {
	ui.Loading
	weavecontext.BaseModel
}

func NewFundDefaultPresetBroadcastLoading(ctx context.Context) *FundDefaultPresetBroadcastLoading {
	return &FundDefaultPresetBroadcastLoading{
		Loading:   ui.NewLoading("Broadcasting transactions...", broadcastDefaultPresetFromGasStation(ctx)),
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
	}
}

func (m *FundDefaultPresetBroadcastLoading) Init() tea.Cmd {
	return m.Loading.Init()
}

func broadcastDefaultPresetFromGasStation(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		state := weavecontext.GetCurrentState[State](ctx)
		gasStationKey, err := config.GetGasStationKey()
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to get gas station key: %v", err)}
		}
		l1ActiveLcd, err := GetL1ActiveLcd(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: err}
		}
		cliTx, err := cosmosutils.NewInitiadTxExecutor(l1ActiveLcd)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: err}
		}

		l1GasDenom, err := GetL1GasDenom(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: err}
		}
		l1GasPrices, err := GetL1GasPrices(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: err}
		}
		l1ActiveRpc, err := GetL1ActiveRpc(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: err}
		}
		l1ChainId, err := GetL1ChainId(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: err}
		}
		if state.l1FundingAmount != "0" {
			res, err := cliTx.BroadcastMsgSend(
				gasStationKey.Mnemonic,
				state.l1RelayerAddress,
				fmt.Sprintf("%s%s", state.l1FundingAmount, l1GasDenom),
				l1GasPrices,
				l1ActiveRpc,
				l1ChainId,
			)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: err}
			}
			state.l1FundingTxHash = res.TxHash
		}

		l2GasDenom, err := GetL2GasDenom(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: err}
		}
		l2GasPrices, err := GetL2GasPrices(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: err}
		}
		l2ActiveRpc, err := GetL2ActiveRpc(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: err}
		}
		l2ChainId, err := GetL2ChainId(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: err}
		}
		if state.l2FundingAmount != "0" {
			res, err := cliTx.BroadcastMsgSend(
				gasStationKey.Mnemonic,
				state.l2RelayerAddress,
				fmt.Sprintf("%s%s", state.l2FundingAmount, l2GasDenom),
				l2GasPrices,
				l2ActiveRpc,
				l2ChainId,
			)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: err}
			}
			state.l2FundingTxHash = res.TxHash
		}

		return ui.EndLoading{
			Ctx: weavecontext.SetCurrentState(ctx, state),
		}
	}
}

func (m *FundDefaultPresetBroadcastLoading) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		m.Ctx = m.Loading.EndContext
		state := weavecontext.PushPageAndGetState[State](m)
		if state.l1FundingTxHash != "" {
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, "The relayer account has been funded on L1, with Tx Hash", []string{}, state.l1FundingTxHash))
		}
		if state.l2FundingTxHash != "" {
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, "The relayer account has been funded on L2, with Tx Hash", []string{}, state.l2FundingTxHash))
		}

		return NewTerminalState(weavecontext.SetCurrentState(m.Ctx, state)), tea.Quit
	}
	return m, cmd
}

func (m *FundDefaultPresetBroadcastLoading) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" + m.Loading.View())
}

type FundManuallyL1BalanceInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question string
}

func NewFundManuallyL1BalanceInput(ctx context.Context) (*FundManuallyL1BalanceInput, error) {
	l1GasDenom, err := GetL1GasDenom(ctx)
	if err != nil {
		return nil, fmt.Errorf("get l1 gas denom: %w", err)
	}
	model := &FundManuallyL1BalanceInput{
		TextInput: ui.NewTextInput(false),
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  fmt.Sprintf("Specify the amount that would be transferred to Relayer account on L1 (%s)", l1GasDenom),
	}
	model.WithPlaceholder("Enter the amount (or 0 to skip)")
	model.WithValidatorFn(common.ValidatePositiveBigIntOrZero)
	return model, nil
}

func (m *FundManuallyL1BalanceInput) GetQuestion() string {
	return m.question
}

func (m *FundManuallyL1BalanceInput) Init() tea.Cmd {
	return nil
}

func (m *FundManuallyL1BalanceInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[State](m)
		state.l1FundingAmount = input.Text
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"Relayer account", "L1"}, input.Text))

		model, err := NewFundManuallyL2BalanceInput(weavecontext.SetCurrentState(m.Ctx, state))
		if err != nil {
			return m, m.HandlePanic(err)
		}
		return model, nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *FundManuallyL1BalanceInput) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"Relayer account", "L1"}, styles.Question) + m.TextInput.View())
}

type FundManuallyL2BalanceInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question string
}

func NewFundManuallyL2BalanceInput(ctx context.Context) (*FundManuallyL2BalanceInput, error) {
	l2GasDenom, err := GetL2GasDenom(ctx)
	if err != nil {
		return nil, fmt.Errorf("get l2 gas denom: %w", err)
	}
	model := &FundManuallyL2BalanceInput{
		TextInput: ui.NewTextInput(false),
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  fmt.Sprintf("Specify the amount that would be transferred to Relayer account on rollup (%s)", l2GasDenom),
	}
	model.WithPlaceholder("Enter the amount (or 0 to skip)")
	model.WithValidatorFn(common.ValidatePositiveBigIntOrZero)
	return model, nil
}

func (m *FundManuallyL2BalanceInput) GetQuestion() string {
	return m.question
}

func (m *FundManuallyL2BalanceInput) Init() tea.Cmd {
	return nil
}

func (m *FundManuallyL2BalanceInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[State](m)
		state.l2FundingAmount = input.Text
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"Relayer account", "rollup"}, input.Text))

		if state.l1FundingAmount == "0" && state.l2FundingAmount == "0" {
			return NewTerminalState(weavecontext.SetCurrentState(m.Ctx, state)), tea.Quit
		}

		model, err := NewFundDefaultPresetConfirmationInput(weavecontext.SetCurrentState(m.Ctx, state))
		if err != nil {
			return m, m.HandlePanic(err)
		}
		return model, nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *FundManuallyL2BalanceInput) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"Relayer account", "rollup"}, styles.Question) + m.TextInput.View())
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

type SelectingL1Network struct {
	ui.Selector[NetworkSelectOption]
	weavecontext.BaseModel
	question string
}

func NewSelectingL1Network(ctx context.Context) (*SelectingL1Network, error) {
	testnetRegistry, err := registry.GetChainRegistry(registry.InitiaL1Testnet)
	if err != nil {
		return nil, fmt.Errorf("get testnet registry: %w", err)
	}
	// mainnetRegistry := registry.MustGetChainRegistry(registry.InitiaL1Mainnet)
	Testnet = NetworkSelectOption(fmt.Sprintf("Testnet (%s)", testnetRegistry.GetChainId()))
	// Mainnet = NetworkSelectOption(fmt.Sprintf("Mainnet (%s)", mainnetRegistry.GetChainId()))
	tooltips := ui.NewTooltipSlice(tooltip.RelayerL1NetworkSelectTooltip, 2)
	return &SelectingL1Network{
		Selector: ui.Selector[NetworkSelectOption]{
			Options: []NetworkSelectOption{
				Testnet,
				// Mainnet,
			},
			Tooltips: &tooltips,
		},
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  "Select the Initia L1 network you want to connect your rollup to",
	}, nil
}

func (m *SelectingL1Network) GetQuestion() string {
	return m.question
}

func (m *SelectingL1Network) Init() tea.Cmd {
	return nil
}

func (m *SelectingL1Network) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	// Handle selection logic
	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[State](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{"Initia L1 network"}, string(*selected)))
		switch *selected {
		case Testnet:
			state.chainType = registry.InitiaL1Testnet
			testnetRegistry, err := registry.GetChainRegistry(registry.InitiaL1Testnet)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			state.Config["l1.chain_id"] = testnetRegistry.GetChainId()
			if state.Config["l1.rpc_address"], err = testnetRegistry.GetFirstActiveRpc(); err != nil {
				return m, m.HandlePanic(err)
			}
			if state.Config["l1.lcd_address"], err = testnetRegistry.GetFirstActiveLcd(); err != nil {
				return m, m.HandlePanic(err)
			}
			if state.Config["l1.gas_price.price"], err = testnetRegistry.GetFixedMinGasPriceByDenom(DefaultGasPriceDenom); err != nil {
				return m, m.HandlePanic(err)
			}
			state.Config["l1.gas_price.denom"] = DefaultGasPriceDenom

			return NewFieldInputModel(m.Ctx, defaultL2ConfigManual, NewSelectSettingUpIBCChannelsMethod), nil
		}
	}

	return m, cmd
}

func (m *SelectingL1Network) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"Initia L1 network"}, styles.Question) + m.Selector.View())
}

type SelectingL2Network struct {
	ui.Selector[string]
	weavecontext.BaseModel
	question  string
	chainType registry.ChainType
}

func NewSelectingL2Network(ctx context.Context, chainType registry.ChainType) (*SelectingL2Network, error) {
	networks, err := registry.GetAllL2AvailableNetwork(chainType)
	if err != nil {
		return nil, fmt.Errorf("get all l2 available networks: %w", err)
	}

	options := make([]string, 0)
	for _, network := range networks {
		options = append(options, fmt.Sprintf("%s (%s)", network.PrettyName, network.ChainId))
	}
	sort.Slice(options, func(i, j int) bool { return strings.ToLower(options[i]) < strings.ToLower(options[j]) })

	tooltips := ui.NewTooltipSlice(tooltip.RelayerRollupSelectWhitelistedTooltip, len(options))

	return &SelectingL2Network{
		Selector: ui.Selector[string]{
			Options:  options,
			Tooltips: &tooltips,
		},
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  "Specify rollup network",
		chainType: chainType,
	}, nil
}

func (m *SelectingL2Network) Init() tea.Cmd {
	return nil
}

func (m *SelectingL2Network) GetQuestion() string {
	return m.question
}

func (m *SelectingL2Network) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	if len(m.Options) == 0 {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "ctrl+c":
				return nil, tea.Quit
			}
			return m, nil
		}
	}

	// Handle selection logic
	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[State](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{"rollup network"}, *selected))
		m.Ctx = weavecontext.SetCurrentState(m.Ctx, state)

		re := regexp.MustCompile(`\(([^)]+)\)`)
		chainId := re.FindStringSubmatch(m.Options[m.Cursor])[1]

		l1NetworkRegistry, err := registry.GetChainRegistry(state.chainType)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		l2NetworkRegistry, err := registry.GetL2Registry(state.chainType, chainId)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		lcdAddresses, err := l2NetworkRegistry.GetActiveLcds()
		if err != nil {
			return m, m.HandlePanic(err)
		}
		res, err := cosmosutils.QueryChannels(lcdAddresses)
		if err != nil {
			return m, m.HandlePanic(fmt.Errorf("failed to get channels from any active LCD endpoints"))
		}

		params, err := cosmosutils.QueryOPChildParams(lcdAddresses)
		if err != nil {
			return m, m.HandlePanic(fmt.Errorf("failed to get OP child params from any active LCD endpoints"))
		}
		state.feeWhitelistAccounts = append(state.feeWhitelistAccounts, params.FeeWhitelist...)

		pairs := make([]types.IBCChannelPair, 0)
		for _, channel := range res.Channels {
			l1Response, err := l1NetworkRegistry.GetIBCChannelInfo(channel.Counterparty.PortID, channel.Counterparty.ChannelID)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			l2Response, err := l2NetworkRegistry.GetIBCChannelInfo(channel.PortID, channel.ChannelID)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			pairs = append(pairs, types.IBCChannelPair{
				L1ConnectionID: l1Response.Channel.ConnectionHops[0],
				L1:             channel.Counterparty,
				L2ConnectionID: l2Response.Channel.ConnectionHops[0],
				L2: types.Channel{
					PortID:    channel.PortID,
					ChannelID: channel.ChannelID,
				},
			})
		}

		l2DefaultFeeToken, err := l2NetworkRegistry.GetDefaultFeeToken()
		if err != nil {
			return m, m.HandlePanic(err)
		}
		l2Rpc, err := l2NetworkRegistry.GetFirstActiveRpc()
		if err != nil {
			return m, m.HandlePanic(err)
		}
		l2Rest, err := l2NetworkRegistry.GetFirstActiveLcd()
		if err != nil {
			return m, m.HandlePanic(err)
		}
		analytics.TrackEvent(analytics.RelayerL2Selected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, chainId))

		state.Config["l2.chain_id"] = chainId
		state.Config["l2.gas_price.denom"] = l2DefaultFeeToken.Denom
		state.Config["l2.gas_price.price"] = strconv.FormatFloat(l2DefaultFeeToken.FixedMinGasPrice, 'f', -1, 64)
		state.Config["l2.rpc_address"] = l2Rpc
		state.Config["l2.lcd_address"] = l2Rest

		return NewIBCChannelsCheckbox(weavecontext.SetCurrentState(m.Ctx, state), pairs), nil
	}

	return m, cmd
}

func (m *SelectingL2Network) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	if len(m.Options) == 0 {
		return m.WrapView(state.weave.Render() + styles.RenderPrompt(fmt.Sprintf("No rollups found for chain type %s in initia-registry.", m.chainType), []string{"rollup network"}, styles.Information) +
			"\n" + styles.RenderFooter("Ctrl+z to go back, Ctrl+c or q to quit."))
	}
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"rollup network"}, styles.Question) + m.Selector.View())
}

type TerminalState struct {
	weavecontext.BaseModel
}

func NewTerminalState(ctx context.Context) tea.Model {
	analytics.TrackCompletedEvent(analytics.SetupRelayerFeature)
	return &TerminalState{
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
	}
}

func (m *TerminalState) Init() tea.Cmd {
	return nil
}

func (m *TerminalState) Update(_ tea.Msg) (tea.Model, tea.Cmd) {
	return m, tea.Quit
}

func (m *TerminalState) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	return m.WrapView(state.weave.Render()) + getRelayerSetSuccessMessage()
}

type SelectingL1NetworkRegistry struct {
	ui.Selector[NetworkSelectOption]
	weavecontext.BaseModel
	question string
}

func NewSelectingL1NetworkRegistry(ctx context.Context) (*SelectingL1NetworkRegistry, error) {
	testnetRegistry, err := registry.GetChainRegistry(registry.InitiaL1Testnet)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain registry: %w", err)
	}
	mainnetRegistry, err := registry.GetChainRegistry(registry.InitiaL1Mainnet)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain registry: %w", err)
	}
	Testnet = NetworkSelectOption(fmt.Sprintf("Testnet (%s)", testnetRegistry.GetChainId()))
	Mainnet = NetworkSelectOption(fmt.Sprintf("Mainnet (%s)", mainnetRegistry.GetChainId()))
	tooltips := ui.NewTooltipSlice(tooltip.RelayerL1NetworkSelectTooltip, 2)
	return &SelectingL1NetworkRegistry{
		Selector: ui.Selector[NetworkSelectOption]{
			Options: []NetworkSelectOption{
				Testnet,
				Mainnet,
			},
			Tooltips: &tooltips,
		},
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  "Select the Initia L1 network you want to connect your rollup to",
	}, nil
}

func (m *SelectingL1NetworkRegistry) GetQuestion() string {
	return m.question
}

func (m *SelectingL1NetworkRegistry) Init() tea.Cmd {
	return nil
}

func (m *SelectingL1NetworkRegistry) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	// Handle selection logic
	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[State](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{"Initia L1 network"}, string(*selected)))

		switch *selected {
		case Testnet:
			state.chainType = registry.InitiaL1Testnet
		case Mainnet:
			state.chainType = registry.InitiaL1Mainnet
		}

		chainRegistry, err := registry.GetChainRegistry(state.chainType)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		state.Config["l1.chain_id"] = chainRegistry.GetChainId()
		if state.Config["l1.rpc_address"], err = chainRegistry.GetFirstActiveRpc(); err != nil {
			return m, m.HandlePanic(err)
		}
		if state.Config["l1.lcd_address"], err = chainRegistry.GetFirstActiveLcd(); err != nil {
			return m, m.HandlePanic(err)
		}
		if state.Config["l1.gas_price.price"], err = chainRegistry.GetFixedMinGasPriceByDenom(DefaultGasPriceDenom); err != nil {
			return m, m.HandlePanic(err)
		}
		state.Config["l1.gas_price.denom"] = DefaultGasPriceDenom

		model, err := NewSelectingL2Network(weavecontext.SetCurrentState(m.Ctx, state), state.chainType)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		return model, nil
	}

	return m, cmd
}

func (m *SelectingL1NetworkRegistry) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"Initia L1 network"}, styles.Question) + m.Selector.View())
}

type SettingUpIBCChannelOption string

var (
	Basic       SettingUpIBCChannelOption = "Subscribe to only `transfer` and `nft-transfer` IBC Channels (minimal setup)"
	FillFromLCD SettingUpIBCChannelOption = "Fill in rollup REST endpoint to detect all available IBC Channels"
	Manually    SettingUpIBCChannelOption = "Setup IBC Channels manually"
)

type SelectSettingUpIBCChannelsMethod struct {
	ui.Selector[SettingUpIBCChannelOption]
	weavecontext.BaseModel
	question string
}

func NewSelectSettingUpIBCChannelsMethod(ctx context.Context) (tea.Model, error) {
	options := make([]SettingUpIBCChannelOption, 0)
	tooltips := make([]ui.Tooltip, 0)
	artifactsDir, err := weavecontext.GetMinitiaArtifactsJson(ctx)
	if err != nil {
		return nil, fmt.Errorf("get artifacts dir: %w", err)
	}
	if weaveio.FileOrFolderExists(artifactsDir) {
		options = append(options, Basic)
		tooltips = append(tooltips, tooltip.RelayerIBCMinimalSetupTooltip)
	}
	options = append(options, FillFromLCD, Manually)
	tooltips = append(
		tooltips,
		tooltip.RelayerIBCFillFromLCDTooltip,
		tooltip.RelayerIBCManualSetupTooltip,
	)

	return &SelectSettingUpIBCChannelsMethod{
		Selector: ui.Selector[SettingUpIBCChannelOption]{
			Options:  options,
			Tooltips: &tooltips,
		},
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  "Select method to setup IBC channels for the relayer.",
	}, nil
}

func (m *SelectSettingUpIBCChannelsMethod) GetQuestion() string {
	return m.question
}

func (m *SelectSettingUpIBCChannelsMethod) Init() tea.Cmd {
	return nil
}

func (m *SelectSettingUpIBCChannelsMethod) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	// Handle selection logic
	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[State](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{}, string(*selected)))
		switch *selected {
		case Basic:
			analytics.TrackEvent(analytics.SettingUpIBCChannelsMethodSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "basic"))
			artifactsJson, err := weavecontext.GetMinitiaArtifactsJson(m.Ctx)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			// Read the file content
			data, err := os.ReadFile(artifactsJson)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			// Decode the JSON into a struct
			var artifacts types.Artifacts
			if err := json.Unmarshal(data, &artifacts); err != nil {
				return m, m.HandlePanic(err)
			}

			var metadata types.Metadata
			l1NetworkRegistry, err := registry.GetChainRegistry(state.chainType)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			info, err := l1NetworkRegistry.GetOpinitBridgeInfo(artifacts.BridgeID)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			metadata, err = types.DecodeBridgeMetadata(info.BridgeConfig.Metadata)
			if err != nil {
				return m, m.HandlePanic(err)
			}

			channelPairs := make([]types.IBCChannelPair, 0)
			for _, channel := range metadata.PermChannels {
				l1Response, err := l1NetworkRegistry.GetIBCChannelInfo(channel.PortID, channel.ChannelID)
				if err != nil {
					return m, m.HandlePanic(err)
				}
				l2Response, err := GetL2IBCChannelInfo(m.Ctx, l1Response.Channel.Counterparty.PortID, l1Response.Channel.Counterparty.ChannelID)
				if err != nil {
					return m, m.HandlePanic(err)
				}
				channelPairs = append(channelPairs, types.IBCChannelPair{
					L1ConnectionID: l1Response.Channel.ConnectionHops[0],
					L1:             channel,
					L2ConnectionID: l2Response.Channel.ConnectionHops[0],
					L2:             l1Response.Channel.Counterparty,
				})
			}
			return NewIBCChannelsCheckbox(weavecontext.SetCurrentState(m.Ctx, state), channelPairs), nil
		case FillFromLCD:
			analytics.TrackEvent(analytics.SettingUpIBCChannelsMethodSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "lcd"))
			return NewFillL2LCD(weavecontext.SetCurrentState(m.Ctx, state)), nil
		case Manually:
			analytics.TrackEvent(analytics.SettingUpIBCChannelsMethodSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "manaul"))
			return NewFillPortOnL1(weavecontext.SetCurrentState(m.Ctx, state), 0), nil
		}
	}

	return m, cmd
}

func (m *SelectSettingUpIBCChannelsMethod) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{}, styles.Question) + m.Selector.View())
}

func GetL1ChainId(ctx context.Context) (string, error) {
	state := weavecontext.GetCurrentState[State](ctx)
	if chainId, ok := state.Config["l1.chain_id"]; ok {
		return chainId, nil
	}
	return "", fmt.Errorf("l1.chain_id not found in state")
}

func GetL2ChainId(ctx context.Context) (string, error) {
	state := weavecontext.GetCurrentState[State](ctx)
	if chainId, found := state.Config["l2.chain_id"]; found {
		return chainId, nil
	}
	return "", fmt.Errorf("l2.chain_id not found in state")
}

func GetL1ActiveLcd(ctx context.Context) (string, error) {
	state := weavecontext.GetCurrentState[State](ctx)
	if lcd, found := state.Config["l1.lcd_address"]; found {
		return lcd, nil
	}
	return "", fmt.Errorf("l1.lcd_address not found in state")
}

func GetL1ActiveRpc(ctx context.Context) (string, error) {
	state := weavecontext.GetCurrentState[State](ctx)
	if rpc, found := state.Config["l1.rpc_address"]; found {
		return rpc, nil
	}
	return "", fmt.Errorf("l1.rpc_address not found in state")
}

func GetL2ActiveLcd(ctx context.Context) (string, error) {
	state := weavecontext.GetCurrentState[State](ctx)
	if lcd, found := state.Config["l2.lcd_address"]; found {
		return lcd, nil
	}
	return "", fmt.Errorf("l2.lcd_address not found in state")
}

func GetL2ActiveRpc(ctx context.Context) (string, error) {
	state := weavecontext.GetCurrentState[State](ctx)
	if rpc, found := state.Config["l2.rpc_address"]; found {
		return rpc, nil
	}
	return "", fmt.Errorf("l2.rpc_address not found in state")
}

func GetL1GasDenom(ctx context.Context) (string, error) {
	state := weavecontext.GetCurrentState[State](ctx)
	if denom, found := state.Config["l1.gas_price.denom"]; found {
		return denom, nil
	}

	return "", fmt.Errorf("l1.gas_price.denom not found in state")
}

func GetL2GasDenom(ctx context.Context) (string, error) {
	state := weavecontext.GetCurrentState[State](ctx)
	if denom, found := state.Config["l2.gas_price.denom"]; found {
		return denom, nil
	}

	return "", fmt.Errorf("l2.gas_price.denom not found in state")
}

func GetL1GasPrices(ctx context.Context) (string, error) {
	denom, err := GetL1GasDenom(ctx)
	if err != nil {
		return "", err
	}
	state := weavecontext.GetCurrentState[State](ctx)
	price, ok := state.Config["l1.gas_price.price"]
	if !ok {
		return "", fmt.Errorf("cannot get l1 gas price from state")
	}

	return fmt.Sprintf("%s%s", price, denom), nil
}

func GetL2GasPrices(ctx context.Context) (string, error) {
	denom, err := GetL2GasDenom(ctx)
	if err != nil {
		return "", err
	}
	state := weavecontext.GetCurrentState[State](ctx)
	amount, ok := state.Config["l2.gas_price.price"]
	if !ok {
		return "", fmt.Errorf("cannot get l2 gas denom from state")
	}

	return fmt.Sprintf("%s%s", amount, denom), nil
}

func GetL2IBCChannelInfo(ctx context.Context, port, channel string) (types.ChannelResponse, error) {
	rest, err := GetL2ActiveLcd(ctx)
	if err != nil {
		return types.ChannelResponse{}, err
	}

	httpClient := client.NewHTTPClient()

	var response types.ChannelResponse
	if _, err := httpClient.Get(rest, fmt.Sprintf("/ibc/core/channel/v1/channels/%s/ports/%s", channel, port), nil, &response); err != nil {
		return types.ChannelResponse{}, err
	}

	if len(response.Channel.ConnectionHops) == 0 {
		return types.ChannelResponse{}, fmt.Errorf("no connection ID found")
	}

	return response, nil
}

type FillPortOnL1 struct {
	weavecontext.BaseModel
	ui.TextInput
	idx      int
	question string
	extra    string
}

func NewFillPortOnL1(ctx context.Context, idx int) *FillPortOnL1 {
	extra := ""
	if chainId, err := GetL1ChainId(ctx); err == nil {
		extra = fmt.Sprintf("(%s)", chainId)
	}
	model := &FillPortOnL1{
		TextInput: ui.NewTextInput(false),
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  fmt.Sprintf("Specify the port ID on L1 %s", extra),
		idx:       idx,
		extra:     extra,
	}
	relayerTooltip := tooltip.RelayerL1IBCPortIDTooltip
	model.WithTooltip(&relayerTooltip)
	model.WithPlaceholder("ex. transfer")
	return model
}

func (m *FillPortOnL1) GetQuestion() string {
	return m.question
}

func (m *FillPortOnL1) Init() tea.Cmd {
	return nil
}

func (m *FillPortOnL1) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[State](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"L1", m.extra}, m.TextInput.Text))
		state.IBCChannels = append(state.IBCChannels, types.IBCChannelPair{})
		state.IBCChannels[m.idx].L1.PortID = m.TextInput.Text
		return NewFillChannelL1(weavecontext.SetCurrentState(m.Ctx, state), m.TextInput.Text, m.idx), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *FillPortOnL1) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"L1", m.extra}, styles.Question) + m.TextInput.View())
}

type FillChannelL1 struct {
	weavecontext.BaseModel
	ui.TextInput
	idx      int
	port     string
	question string
	extra    string
}

func NewFillChannelL1(ctx context.Context, port string, idx int) *FillChannelL1 {
	extra := ""
	if chainId, err := GetL1ChainId(ctx); err == nil {
		extra = fmt.Sprintf("(%s)", chainId)
	}
	model := &FillChannelL1{
		TextInput: ui.NewTextInput(false),
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  fmt.Sprintf("Specify the channel ID that associated with `%s` port on L1 %s", port, extra),
		idx:       idx,
		port:      port,
		extra:     extra,
	}
	relayerTooltip := tooltip.RelayerL1IBCChannelIDTooltip
	model.WithTooltip(&relayerTooltip)
	model.WithPlaceholder("ex. channel-1")
	return model
}

func (m *FillChannelL1) GetQuestion() string {
	return m.question
}

func (m *FillChannelL1) Init() tea.Cmd {
	return nil
}

func (m *FillChannelL1) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[State](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"L1", m.port, m.extra}, m.TextInput.Text))
		state.IBCChannels[m.idx].L1.ChannelID = m.TextInput.Text
		return NewFillPortOnL2(weavecontext.SetCurrentState(m.Ctx, state), m.idx, CounterParty{Port: m.port, Channel: m.TextInput.Text}), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *FillChannelL1) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"L1", m.port, m.extra}, styles.Question) + m.TextInput.View())
}

type CounterParty struct {
	Port    string
	Channel string
}

type FillPortOnL2 struct {
	weavecontext.BaseModel
	ui.TextInput
	idx          int
	counterParty CounterParty
	question     string
	extra        string
}

func NewFillPortOnL2(ctx context.Context, idx int, counterParty CounterParty) *FillPortOnL2 {
	extra := ""
	if chainId, err := GetL2ChainId(ctx); err == nil {
		extra = fmt.Sprintf("(%s)", chainId)
	}
	model := &FillPortOnL2{
		TextInput:    ui.NewTextInput(false),
		BaseModel:    weavecontext.BaseModel{Ctx: ctx},
		question:     fmt.Sprintf("Specify the port ID on rollup that associated with `%s:%s` on L1 %s", counterParty.Port, counterParty.Channel, extra),
		idx:          idx,
		counterParty: counterParty,
		extra:        extra,
	}
	relayerTooltip := tooltip.RelayerRollupIBCPortIDTooltip
	model.WithTooltip(&relayerTooltip)
	model.WithPlaceholder("ex. transfer")
	return model
}

func (m *FillPortOnL2) GetQuestion() string {
	return m.question
}

func (m *FillPortOnL2) Init() tea.Cmd {
	return nil
}

func (m *FillPortOnL2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[State](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"L2", m.extra}, m.TextInput.Text))
		state.IBCChannels[m.idx].L2.PortID = m.TextInput.Text
		return NewFillChannelL2(weavecontext.SetCurrentState(m.Ctx, state), m.TextInput.Text, m.idx), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *FillPortOnL2) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"L2", m.extra}, styles.Question) + m.TextInput.View())
}

type FillChannelL2 struct {
	weavecontext.BaseModel
	ui.TextInput
	idx      int
	port     string
	extra    string
	question string
}

func NewFillChannelL2(ctx context.Context, port string, idx int) *FillChannelL2 {
	extra := ""
	if chainId, err := GetL2ChainId(ctx); err == nil {
		extra = fmt.Sprintf("(%s)", chainId)
	}
	model := &FillChannelL2{
		TextInput: ui.NewTextInput(false),
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  fmt.Sprintf("Specify the channel on rollup network that associated with `%s` port on rollup network %s", port, extra),
		idx:       idx,
		port:      port,
		extra:     extra,
	}
	relayerTooltip := tooltip.RelayerRollupIBCChannelIDTooltip
	model.WithTooltip(&relayerTooltip)
	model.WithPlaceholder("ex. channel-1")
	return model
}

func (m *FillChannelL2) GetQuestion() string {
	return m.question
}

func (m *FillChannelL2) Init() tea.Cmd {
	return nil
}

func (m *FillChannelL2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[State](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"L2", m.port, m.extra}, m.TextInput.Text))
		state.IBCChannels[m.idx].L2.ChannelID = m.TextInput.Text
		return NewAddMoreIBCChannels(weavecontext.SetCurrentState(m.Ctx, state), m.idx), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *FillChannelL2) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"L2", m.port, m.extra}, styles.Question) + m.TextInput.View())
}

type AddMoreIBCChannels struct {
	ui.Selector[string]
	weavecontext.BaseModel
	question string
	idx      int
}

func NewAddMoreIBCChannels(ctx context.Context, idx int) *AddMoreIBCChannels {
	return &AddMoreIBCChannels{
		Selector: ui.Selector[string]{
			Options: []string{
				"Yes",
				"No",
			},
		},
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  "Do you want to add more IBC Channel pairs?",
		idx:       idx,
	}
}

func (m *AddMoreIBCChannels) GetQuestion() string {
	return m.question
}

func (m *AddMoreIBCChannels) Init() tea.Cmd {
	return nil
}

func (m *AddMoreIBCChannels) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[State](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{}, *selected))

		return NewFillPortOnL1(weavecontext.SetCurrentState(m.Ctx, state), m.idx), nil
	}
	return m, cmd
}

func (m *AddMoreIBCChannels) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{}, styles.Question) + m.Selector.View())
}

type IBCChannelsCheckbox struct {
	ui.CheckBox[string]
	weavecontext.BaseModel
	question string
	pairs    []types.IBCChannelPair
	alert    bool
}

func NewIBCChannelsCheckbox(ctx context.Context, pairs []types.IBCChannelPair) *IBCChannelsCheckbox {
	prettyPairs := []string{"Relay all IBC channels"}
	for _, pair := range pairs {
		prettyPairs = append(prettyPairs, fmt.Sprintf("(L1) %s : %s ◀ ▶︎ (L2) %s : %s", pair.L1.PortID, pair.L1.ChannelID, pair.L2.PortID, pair.L2.ChannelID))
	}
	cb := ui.NewCheckBox(prettyPairs)
	tooltips := ui.NewTooltipSlice(tooltip.RelayerIBCChannelsTooltip, len(prettyPairs))
	cb.WithTooltip(&tooltips)
	cb.EnableSelectAll()
	pairs = append([]types.IBCChannelPair{pairs[0]}, pairs...)
	return &IBCChannelsCheckbox{
		CheckBox:  *cb,
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  "Select the IBC channels you would like to relay",
		pairs:     pairs,
	}
}

func (m *IBCChannelsCheckbox) GetQuestion() string {
	return m.question
}

func (m *IBCChannelsCheckbox) Init() tea.Cmd {
	return nil
}

func (m *IBCChannelsCheckbox) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}
	cb, cmd, done := m.Select(msg)
	_ = cb
	if done {
		analytics.TrackEvent(analytics.IBCChannelsSelected, analytics.NewEmptyEvent().Add("select-all", m.Selected[0]))

		state := weavecontext.PushPageAndGetState[State](m)
		ibcChannels := make([]types.IBCChannelPair, 0)
		for idx := 1; idx < len(m.pairs); idx++ {
			if m.Selected[idx] {
				ibcChannels = append(ibcChannels, m.pairs[idx])
			}
			state.IBCChannels = ibcChannels
		}
		response := ""
		channelCount := len(state.IBCChannels)
		if channelCount == 0 {
			m.alert = true
			return m, cmd
		}

		if channelCount == 1 {
			response = "1 IBC channel subscribed"
		} else {
			response = fmt.Sprintf("%d IBC channels subscribed", channelCount)
		}

		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{}, response))

		if state.minitiaConfig != nil {
			return NewAddChallengerKeyToRelayer(weavecontext.SetCurrentState(m.Ctx, state)), nil
		}

		model, err := NewL1KeySelect(weavecontext.SetCurrentState(m.Ctx, state))
		if err != nil {
			return m, m.HandlePanic(err)
		}
		return model, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case " ":
			m.alert = false
		}
	}

	return m, cmd
}

func (m *IBCChannelsCheckbox) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	m.CheckBox.ViewTooltip(m.Ctx)
	if m.alert {
		return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{}, styles.Question) + "\n" + m.CheckBox.View() + "\n" + styles.Text("Select at least one IBC channel to proceed to the next step.", styles.Yellow) + "\n")
	}
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{}, styles.Question) + "\n" + m.CheckBox.View())
}

type FillL2LCD struct {
	weavecontext.BaseModel
	ui.TextInput
	question string
	extra    string
	err      error
}

func NewFillL2LCD(ctx context.Context) *FillL2LCD {
	chainId, _ := GetL2ChainId(ctx)
	extra := fmt.Sprintf("(%s)", chainId)
	toolTip := tooltip.RelayerRollupLCDTooltip
	m := &FillL2LCD{
		TextInput: ui.NewTextInput(false),
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  fmt.Sprintf("Specify rollup LCD endpoint %s", extra),
		extra:     extra,
	}
	m.WithTooltip(&toolTip)
	m.WithPlaceholder("ex. http://localhost:1317")
	return m
}

func (m *FillL2LCD) GetQuestion() string {
	return m.question
}

func (m *FillL2LCD) Init() tea.Cmd {
	return nil
}

func (m *FillL2LCD) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[State](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"L2", "LCD_address", m.extra}, m.TextInput.Text))

		// TODO: should have loading state for this
		httpClient := client.NewHTTPClient()
		var res types.ChannelsResponse
		_, err := httpClient.Get(input.Text, "/ibc/core/channel/v1/channels", nil, &res)
		if err != nil {
			m.err = fmt.Errorf("unable to call the LCD endpoint '%s'. Please verify that the address is correct and reachable", input.Text)
			return m, cmd
		}

		if params, err := cosmosutils.QueryOPChildParams([]string{input.Text}); err == nil {
			state.feeWhitelistAccounts = append(state.feeWhitelistAccounts, params.FeeWhitelist...)
		}

		l1NetworkRegistry, err := registry.GetChainRegistry(state.chainType)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		l2NetworkRegistry, err := registry.GetL2Registry(state.chainType, state.Config["l2.chain_id"])
		if err != nil {
			return m, m.HandlePanic(err)
		}

		pairs := make([]types.IBCChannelPair, 0)
		for _, channel := range res.Channels {
			l1Response, err := l1NetworkRegistry.GetIBCChannelInfo(channel.Counterparty.PortID, channel.Counterparty.ChannelID)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			l2Response, err := l2NetworkRegistry.GetIBCChannelInfo(channel.PortID, channel.ChannelID)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			pairs = append(pairs, types.IBCChannelPair{
				L1ConnectionID: l1Response.Channel.ConnectionHops[0],
				L1:             channel.Counterparty,
				L2ConnectionID: l2Response.Channel.ConnectionHops[0],
				L2: types.Channel{
					PortID:    channel.PortID,
					ChannelID: channel.ChannelID,
				},
			})
		}

		return NewIBCChannelsCheckbox(weavecontext.SetCurrentState(m.Ctx, state), pairs), nil
	}
	m.TextInput = input
	return m, cmd
}

func (m *FillL2LCD) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	if m.err != nil {
		return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"L2", "LCD_address", m.extra}, styles.Question) + m.TextInput.ViewErr(m.err))
	}
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"L2", "LCD_address", m.extra}, styles.Question) + m.TextInput.View())
}

type SettingUpRelayer struct {
	ui.Loading
	weavecontext.BaseModel
}

func NewSettingUpRelayer(ctx context.Context) *SettingUpRelayer {
	return &SettingUpRelayer{
		Loading:   ui.NewLoading("Setting up relayer...", WaitSettingUpRelayer(ctx)),
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
	}
}

func WaitSettingUpRelayer(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		state := weavecontext.GetCurrentState[State](ctx)
		// Create Rapid Relayer configuration
		err := createRapidRelayerConfig(state)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to create rapid relayer config: %v", err)}
		}

		// Return updated state
		return ui.EndLoading{Ctx: weavecontext.SetCurrentState(ctx, state)}
	}
}

func (m *SettingUpRelayer) Init() tea.Cmd {
	return m.Loading.Init()
}

func (m *SettingUpRelayer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		m.Ctx = m.Loading.EndContext
		state := weavecontext.PushPageAndGetState[State](m)
		model := NewFetchingBalancesLoading(weavecontext.SetCurrentState(m.Ctx, state))
		return model, model.Init()
	}
	return m, cmd
}

func (m *SettingUpRelayer) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" + m.Loading.View())
}

type AddChallengerKeyToRelayer struct {
	ui.Selector[AddChallengerKeyToRelayerOption]
	weavecontext.BaseModel
	question string
}

type AddChallengerKeyToRelayerOption string

const (
	YesAddChallengerKeyToRelayerOption AddChallengerKeyToRelayerOption = "Yes (recommended, open the tooltip to see the details)"
	NoAddChallengerKeyToRelayerOption  AddChallengerKeyToRelayerOption = "No, I want to setup relayer with a separate key"
)

func NewAddChallengerKeyToRelayer(ctx context.Context) *AddChallengerKeyToRelayer {
	tooltips := ui.NewTooltipSlice(tooltip.RelayerChallengerKeyTooltip, 2)
	return &AddChallengerKeyToRelayer{
		Selector: ui.Selector[AddChallengerKeyToRelayerOption]{
			Options: []AddChallengerKeyToRelayerOption{
				YesAddChallengerKeyToRelayerOption,
				NoAddChallengerKeyToRelayerOption,
			},
			Tooltips: &tooltips,
		},
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		question:  "Do you want to setup relayer with the challenger key",
	}
}

func (m *AddChallengerKeyToRelayer) GetQuestion() string {
	return m.question
}

func (m *AddChallengerKeyToRelayer) Init() tea.Cmd {
	return nil
}

func (m *AddChallengerKeyToRelayer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[State](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{"challenger key"}, string(*selected)))
		switch *selected {
		case YesAddChallengerKeyToRelayerOption:
			analytics.TrackEvent(analytics.UseChallengerKeySelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, true))
			state.l1RelayerAddress = state.minitiaConfig.SystemKeys.Challenger.L1Address
			state.l1RelayerMnemonic = state.minitiaConfig.SystemKeys.Challenger.Mnemonic

			state.l2RelayerAddress = state.minitiaConfig.SystemKeys.Challenger.L2Address
			state.l2RelayerMnemonic = state.minitiaConfig.SystemKeys.Challenger.Mnemonic

			model := NewSettingUpRelayer(weavecontext.SetCurrentState(m.Ctx, state))
			return model, model.Init()
		case NoAddChallengerKeyToRelayerOption:
			analytics.TrackEvent(analytics.UseChallengerKeySelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, false))
			model, err := NewL1KeySelect(weavecontext.SetCurrentState(m.Ctx, state))
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, nil
		}

	}
	return m, cmd
}

func getRelayerSetSuccessMessage() string {
	userHome, _ := os.UserHomeDir()
	relayerHome := filepath.Join(userHome, common.RelayerDirectory)
	s := "\n" + styles.RenderPrompt("Rapid relayer config is generated successfully!", []string{}, styles.Completed)
	s += "\n" + styles.RenderPrompt(fmt.Sprintf("Config file is saved at %s/config.json. You can modify it as needed.", relayerHome), []string{}, styles.Information)
	s += "\n" + styles.RenderPrompt("To start relaying:", []string{}, styles.Empty)
	s += "\n" + styles.RenderPrompt("1. git clone https://github.com/initia-labs/rapid-relayer && cd rapid-relayer && npm install", []string{}, styles.Empty)
	s += "\n" + styles.RenderPrompt(fmt.Sprintf("2. cp %s/config.json ./config.json", relayerHome), []string{}, styles.Empty)
	s += "\n" + styles.RenderPrompt("3. npm start", []string{}, styles.Empty) + "\n"
	return s
}

func (m *AddChallengerKeyToRelayer) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(
		m.GetQuestion(),
		[]string{"challenger key"},
		styles.Question,
	) + m.Selector.View())
}
