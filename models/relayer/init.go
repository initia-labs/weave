package relayer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
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
	"github.com/initia-labs/weave/crypto"
	weaveio "github.com/initia-labs/weave/io"
	"github.com/initia-labs/weave/registry"
	"github.com/initia-labs/weave/service"
	"github.com/initia-labs/weave/styles"
	"github.com/initia-labs/weave/tooltip"
	"github.com/initia-labs/weave/types"
	"github.com/initia-labs/weave/ui"
)

var defaultL2ConfigLocal = []*Field{
	{Name: "l2.rpc_address", Type: StringField, Question: "Specify rollup RPC endpoint", Highlights: []string{"rollup RPC endpoint"}, Placeholder: `Press tab to use "http://localhost:26657"`, DefaultValue: "http://localhost:26657", ValidateFn: common.ValidateURL, Tooltip: &tooltip.RollupRPCEndpointTooltip},
	{Name: "l2.grpc_address", Type: StringField, Question: "Specify rollup GRPC endpoint", Highlights: []string{"rollup GRPC endpoint"}, Placeholder: `Press tab to use "http://localhost:9090"`, DefaultValue: "http://localhost:9090", ValidateFn: common.ValidateURL, Tooltip: &tooltip.RollupGRPCEndpointTooltip},
	{Name: "l2.websocket", Type: StringField, Question: "Specify rollup WebSocket endpoint", Highlights: []string{"rollup WebSocket endpoint"}, Placeholder: `Press tab to use "ws://localhost:26657/websocket"`, DefaultValue: "ws://localhost:26657/websocket", ValidateFn: common.ValidateWSURL, Tooltip: &tooltip.RollupWebSocketEndpointTooltip},
}

var defaultL2ConfigManual = []*Field{
	{Name: "l2.chain_id", Type: StringField, Question: "Specify rollup chain ID", Highlights: []string{"rollup chain ID"}, Placeholder: "ex. rollup-1", ValidateFn: common.ValidateEmptyString, Tooltip: &tooltip.RollupChainIdTooltip},
	{Name: "l2.rpc_address", Type: StringField, Question: "Specify rollup RPC endpoint", Highlights: []string{"rollup RPC endpoint"}, Placeholder: "ex. http://localhost:26657", ValidateFn: common.ValidateURL, Tooltip: &tooltip.RollupRPCEndpointTooltip},
	{Name: "l2.grpc_address", Type: StringField, Question: "Specify rollup GRPC endpoint", Highlights: []string{"rollup GRPC endpoint"}, Placeholder: "ex. http://localhost:9090", ValidateFn: common.ValidateURL, Tooltip: &tooltip.RollupGRPCEndpointTooltip},
	{Name: "l2.websocket", Type: StringField, Question: "Specify rollup WebSocket endpoint", Highlights: []string{"rollup WebSocket endpoint"}, Placeholder: "ex. ws://localhost:26657/websocket", ValidateFn: common.ValidateWSURL, Tooltip: &tooltip.RollupWebSocketEndpointTooltip},
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
			analytics.TrackEvent(analytics.RelayerRollupSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "whitelise"))

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

			if minitiaConfig.L1Config.ChainID == InitiaTestnetChainId {
				testnetRegistry, err := registry.GetChainRegistry(registry.InitiaL1Testnet)
				if err != nil {
					return m, m.HandlePanic(err)
				}
				state.Config["l1.chain_id"] = testnetRegistry.GetChainId()
				if state.Config["l1.rpc_address"], err = testnetRegistry.GetActiveRpc(); err != nil {
					return m, m.HandlePanic(err)
				}
				if state.Config["l1.grpc_address"], err = testnetRegistry.GetActiveGrpc(); err != nil {
					return m, m.HandlePanic(err)
				}
				if state.Config["l1.lcd_address"], err = testnetRegistry.GetActiveLcd(); err != nil {
					return m, m.HandlePanic(err)
				}
				if state.Config["l1.websocket"], err = testnetRegistry.GetActiveWebSocket(); err != nil {
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

			} else {
				return m, m.HandlePanic(fmt.Errorf("not support L1"))
			}

			return NewFieldInputModel(weavecontext.SetCurrentState(m.Ctx, state), defaultL2ConfigLocal, NewSelectSettingUpIBCChannelsMethod), nil
		case Manual:
			analytics.TrackEvent(analytics.RelayerRollupSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "manaul"))

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
	if l1RelayerAddress, found := cosmosutils.GetHermesRelayerAddress(state.hermesBinaryPath, l1ChainId); found {
		state.l1RelayerAddress = l1RelayerAddress
		options = append([]L1KeySelectOption{L1ExistingKey}, options...)
	}

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
	if l2RelayerAddress, found := cosmosutils.GetHermesRelayerAddress(state.hermesBinaryPath, l2ChainId); found {
		state.l2RelayerAddress = l2RelayerAddress
		options = append([]L2KeySelectOption{L2ExistingKey}, options...)
	}

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
				model := NewFetchingBalancesLoading(weavecontext.SetCurrentState(m.Ctx, state))
				return model, model.Init()
			case L2SameKey:
				state.l2RelayerAddress = state.l1RelayerAddress
				userHome, err := os.UserHomeDir()
				if err != nil {
					return m, m.HandlePanic(fmt.Errorf("could not get user home directory: %s", err))
				}
				l1ChainId, err := GetL1ChainId(m.Ctx)
				if err != nil {
					return m, m.HandlePanic(err)
				}
				l1ExistingKeyPath := filepath.Join(userHome, HermesKeysDirectory, l1ChainId)
				l2ChainId, err := GetL2ChainId(m.Ctx)
				if err != nil {
					return m, m.HandlePanic(err)
				}
				l2KeyPath := filepath.Join(userHome, HermesKeysDirectory, l2ChainId)
				if err = weaveio.CopyDirectory(l1ExistingKeyPath, l2KeyPath); err != nil {
					return m, m.HandlePanic(fmt.Errorf("could not copy L1 existing key: %s", err))
				}
				model := NewFetchingBalancesLoading(weavecontext.SetCurrentState(m.Ctx, state))
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
		l1ChainId, err := GetL1ChainId(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("get l1 chain id: %w", err)}
		}

		relayerKey, err := cosmosutils.GenerateAndReplaceHermesKey(state.hermesBinaryPath, l1ChainId)
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
			l2ChainId, err := GetL2ChainId(m.Ctx)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			_, err = cosmosutils.RecoverAndReplaceHermesKey(state.hermesBinaryPath, l2ChainId, state.l2RelayerMnemonic)
			if err != nil {
				return m, m.HandlePanic(fmt.Errorf("failed to recover Hermes key for rollup: %w", err))
			}
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
		l2ChainId, err := GetL2ChainId(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("get l2 chain id: %w", err)}
		}

		relayerKey, err := cosmosutils.GenerateAndReplaceHermesKey(state.hermesBinaryPath, l2ChainId)
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
	ui.Clickable
	weavecontext.BaseModel
	question string
}

func NewKeysMnemonicDisplayInput(ctx context.Context) *KeysMnemonicDisplayInput {
	clickable := make([]*ui.ClickableItem, 0)
	state := weavecontext.GetCurrentState[State](ctx)
	if state.l1KeyMethod == string(L1GenerateKey) {
		clickable = append(clickable, ui.NewClickableItem(
			map[bool]string{
				true:  "Copied! Click to copy again",
				false: "Click here to copy",
			},
			func() error {
				return weaveio.CopyToClipboard(state.l1RelayerMnemonic)
			}))
	}
	if state.l2KeyMethod == string(L2GenerateKey) {
		clickable = append(clickable, ui.NewClickableItem(
			map[bool]string{
				true:  "Copied! Click to copy again",
				false: "Click here to copy",
			},
			func() error {
				return weaveio.CopyToClipboard(state.l2RelayerMnemonic)
			}))
	}
	model := &KeysMnemonicDisplayInput{
		TextInput: ui.NewTextInput(true),
		Clickable: *ui.NewClickable(clickable...),
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		question:  "Please type `continue` to proceed after you have securely stored the mnemonic.",
	}
	model.WithPlaceholder("Type `continue` to continue, Ctrl+C to quit.")
	model.WithValidatorFn(common.ValidateExactString("continue"))
	return model
}

func (m *KeysMnemonicDisplayInput) GetQuestion() string {
	return m.question
}

func (m *KeysMnemonicDisplayInput) Init() tea.Cmd {
	return m.Clickable.Init()
}

func (m *KeysMnemonicDisplayInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	err := m.Clickable.ClickableUpdate(msg)
	if err != nil {
		return m, m.HandlePanic(err)
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
			model := NewFetchingBalancesLoading(weavecontext.SetCurrentState(m.Ctx, state))
			return model, tea.Batch(model.Init(), m.Clickable.PostUpdate())
		case L2ImportKey:
			return NewImportL2RelayerKeyInput(weavecontext.SetCurrentState(m.Ctx, state)), m.Clickable.PostUpdate()
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
		mnemonicText += styles.RenderMnemonic(
			styles.RenderPrompt(fmt.Sprintf("Weave Relayer on %s", layerText), []string{layerText}, styles.Empty),
			state.l1RelayerAddress,
			state.l1RelayerMnemonic,
			m.Clickable.ClickableView(0),
		)
	}

	if state.l2KeyMethod == string(L2GenerateKey) {
		mnemonicText += styles.RenderMnemonic(
			styles.RenderPrompt("Weave Relayer on L2", []string{"L2"}, styles.Empty),
			state.l2RelayerAddress,
			state.l2RelayerMnemonic,
			m.Clickable.ClickableView(1),
		)
	}

	viewText := m.WrapView(state.weave.Render() + "\n" +
		styles.BoldUnderlineText("Important", styles.Yellow) + "\n" +
		styles.Text("Write down these mnemonic phrases and store them in a safe place. \nIt is the only way to recover your system keys.", styles.Yellow) + "\n\n" +
		mnemonicText + styles.RenderPrompt(m.GetQuestion(), []string{"`continue`"}, styles.Question) + m.TextInput.View())
	err := m.Clickable.ClickableUpdatePositions(viewText)
	if err != nil {
		m.HandlePanic(err)
	}
	return viewText
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

		l1ChainId, err := GetL1ChainId(m.Ctx)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		relayerKey, err := cosmosutils.RecoverAndReplaceHermesKey(state.hermesBinaryPath, l1ChainId, input.Text)
		if err != nil {
			return m, m.HandlePanic(err)
		}

		state.l1RelayerMnemonic = relayerKey.Mnemonic
		state.l1RelayerAddress = relayerKey.Address
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"relayer account key", m.layerText}, styles.HiddenMnemonicText))

		switch L2KeySelectOption(state.l2KeyMethod) {
		case L2ExistingKey:
			model := NewFetchingBalancesLoading(weavecontext.SetCurrentState(m.Ctx, state))
			return model, model.Init()
		case L2SameKey:
			state.l2RelayerAddress = relayerKey.Address
			state.l2RelayerMnemonic = relayerKey.Mnemonic
			model := NewFetchingBalancesLoading(weavecontext.SetCurrentState(m.Ctx, state))
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

		l2ChainId, err := GetL2ChainId(m.Ctx)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		relayerKey, err := cosmosutils.RecoverAndReplaceHermesKey(state.hermesBinaryPath, l2ChainId, input.Text)
		if err != nil {
			return m, m.HandlePanic(err)
		}

		state.l2RelayerMnemonic = relayerKey.Mnemonic
		state.l2RelayerAddress = relayerKey.Address
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"relayer account key", "L2"}, styles.HiddenMnemonicText))

		model := NewFetchingBalancesLoading(weavecontext.SetCurrentState(m.Ctx, state))
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
		l1Balances, err := cosmosutils.QueryBankBalances(l1Rest, state.l1RelayerAddress)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("cannot fetch balance for l1: %v", err)}
		}
		state.l1NeedsFunding = l1Balances.IsZero()

		querier, err := cosmosutils.NewInitiadQuerier(l1Rest)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("cannot initialize initiad querier: %v", err)}
		}
		l2ActiveRpc, err := GetL2ActiveRpc(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("cannot get l2 active rpc: %v", err)}
		}
		l2Balances, err := querier.QueryBankBalances(state.l2RelayerAddress, l2ActiveRpc)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("cannot fetch balance for l2: %v", err)}
		}
		state.l2NeedsFunding = l2Balances.IsZero()

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
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.NoSeparator, "Your relayer has been setup successfully. 🎉", []string{}, ""))
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

var (
	FundingDefaultPreset FundingAmountSelectOption = ""
)

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
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.NoSeparator, "Your relayer has been set up successfully! 🎉", []string{}, ""))
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
}

func NewFundDefaultPresetConfirmationInput(ctx context.Context) (*FundDefaultPresetConfirmationInput, error) {
	gasStationMnemonic := config.GetGasStationMnemonic()
	initiaGasStationAddress, err := crypto.MnemonicToBech32Address("init", gasStationMnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to recover initia gas station key: %w", err)
	}

	model := &FundDefaultPresetConfirmationInput{
		TextInput:               ui.NewTextInput(false),
		BaseModel:               weavecontext.BaseModel{Ctx: ctx},
		initiaGasStationAddress: initiaGasStationAddress,
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
		_ = weavecontext.PushPageAndGetState[State](m)
		model := NewFundDefaultPresetBroadcastLoading(m.Ctx)
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
	return m.WrapView(state.weave.Render() + "\n" +
		styles.Text("i ", styles.Yellow) +
		styles.RenderPrompt(
			styles.BoldUnderlineText("Weave will now broadcast the following transactions", styles.Yellow),
			[]string{}, styles.Empty,
		) + "\n\n" +
		l1FundingText[state.l1FundingAmount == "0"] +
		l2FundingText[state.l2FundingAmount == "0"] +
		styles.RenderPrompt(m.GetQuestion(), []string{}, styles.Question) + m.TextInput.View())
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
		gasStationMnemonic := config.GetGasStationMnemonic()
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
				gasStationMnemonic,
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
				gasStationMnemonic,
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
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.NoSeparator, "Your relayer has been setup successfully. 🎉", []string{}, ""))

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
	model.WithValidatorFn(common.IsValidInteger)
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
	model.WithValidatorFn(common.IsValidInteger)
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
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"Relayer account", "L2"}, input.Text))

		if state.l1FundingAmount == "0" && state.l2FundingAmount == "0" {
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.NoSeparator, "Your relayer has been setup successfully. 🎉", []string{}, ""))
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
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"Relayer account", "L2"}, styles.Question) + m.TextInput.View())
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
	//mainnetRegistry := registry.MustGetChainRegistry(registry.InitiaL1Mainnet)
	Testnet = NetworkSelectOption(fmt.Sprintf("Testnet (%s)", testnetRegistry.GetChainId()))
	//Mainnet = NetworkSelectOption(fmt.Sprintf("Mainnet (%s)", mainnetRegistry.GetChainId()))
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
			testnetRegistry, err := registry.GetChainRegistry(registry.InitiaL1Testnet)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			state.Config["l1.chain_id"] = testnetRegistry.GetChainId()
			if state.Config["l1.rpc_address"], err = testnetRegistry.GetActiveRpc(); err != nil {
				return m, m.HandlePanic(err)
			}
			if state.Config["l1.grpc_address"], err = testnetRegistry.GetActiveGrpc(); err != nil {
				return m, m.HandlePanic(err)
			}
			if state.Config["l1.websocket"], err = testnetRegistry.GetActiveWebSocket(); err != nil {
				return m, m.HandlePanic(err)
			}
			if state.Config["l1.lcd_address"], err = testnetRegistry.GetActiveLcd(); err != nil {
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
	question string
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

	// Handle selection logic
	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[State](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{"rollup network"}, *selected))
		m.Ctx = weavecontext.SetCurrentState(m.Ctx, state)

		re := regexp.MustCompile(`\(([^)]+)\)`)
		chainId := re.FindStringSubmatch(m.Options[m.Cursor])[1]
		l2Registry, err := registry.GetL2Registry(registry.InitiaL1Testnet, chainId)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		lcdAddress, err := l2Registry.GetActiveLcd()
		if err != nil {
			return m, m.HandlePanic(err)
		}
		httpClient := client.NewHTTPClient()
		var res types.ChannelsResponse
		_, err = httpClient.Get(lcdAddress, "/ibc/core/channel/v1/channels", nil, &res)
		if err != nil {
			return m, m.HandlePanic(err)
		}

		pairs := make([]types.IBCChannelPair, 0)
		for _, channel := range res.Channels {
			pairs = append(pairs, types.IBCChannelPair{
				L1: channel.Counterparty,
				L2: types.Channel{
					PortID:    channel.PortID,
					ChannelID: channel.ChannelID,
				},
			})
		}

		l2DefaultFeeToken, err := l2Registry.GetDefaultFeeToken()
		if err != nil {
			return m, m.HandlePanic(err)
		}
		l2Rpc, err := l2Registry.GetActiveRpc()
		if err != nil {
			return m, m.HandlePanic(err)
		}
		analytics.TrackEvent(analytics.RelayerL2Selected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, chainId))

		state.Config["l2.chain_id"] = chainId
		state.Config["l2.gas_price.denom"] = l2DefaultFeeToken.Denom
		state.Config["l2.gas_price.price"] = strconv.FormatFloat(l2DefaultFeeToken.FixedMinGasPrice, 'f', -1, 64)
		state.Config["l2.rpc_address"] = l2Rpc
		if state.Config["l2.grpc_address"], err = l2Registry.GetActiveGrpc(); err != nil {
			return m, m.HandlePanic(err)
		}
		if state.Config["l2.websocket"], err = l2Registry.GetActiveWebSocket(); err != nil {
			return m, m.HandlePanic(err)
		}

		return NewIBCChannelsCheckbox(weavecontext.SetCurrentState(m.Ctx, state), pairs), nil
	}

	return m, cmd
}

func (m *SelectingL2Network) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"rollup network"}, styles.Question) + m.Selector.View())
}

type TerminalState struct {
	weavecontext.BaseModel
}

func NewTerminalState(ctx context.Context) tea.Model {
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
	return m.WrapView(state.weave.Render())
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
	//mainnetRegistry := registry.MustGetChainRegistry(registry.InitiaL1Mainnet)
	Testnet = NetworkSelectOption(fmt.Sprintf("Testnet (%s)", testnetRegistry.GetChainId()))
	//Mainnet = NetworkSelectOption(fmt.Sprintf("Mainnet (%s)", mainnetRegistry.GetChainId()))
	tooltips := ui.NewTooltipSlice(tooltip.RelayerL1NetworkSelectTooltip, 2)
	return &SelectingL1NetworkRegistry{
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
			testnetRegistry, err := registry.GetChainRegistry(registry.InitiaL1Testnet)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			state.Config["l1.chain_id"] = testnetRegistry.GetChainId()
			if state.Config["l1.rpc_address"], err = testnetRegistry.GetActiveRpc(); err != nil {
				return m, m.HandlePanic(err)
			}
			if state.Config["l1.grpc_address"], err = testnetRegistry.GetActiveGrpc(); err != nil {
				return m, m.HandlePanic(err)
			}
			if state.Config["l1.lcd_address"], err = testnetRegistry.GetActiveLcd(); err != nil {
				return m, m.HandlePanic(err)
			}
			if state.Config["l1.websocket"], err = testnetRegistry.GetActiveWebSocket(); err != nil {
				return m, m.HandlePanic(err)
			}
			if state.Config["l1.gas_price.price"], err = testnetRegistry.GetFixedMinGasPriceByDenom(DefaultGasPriceDenom); err != nil {
				return m, m.HandlePanic(err)
			}
			state.Config["l1.gas_price.denom"] = DefaultGasPriceDenom

			model, err := NewSelectingL2Network(weavecontext.SetCurrentState(m.Ctx, state), registry.InitiaL1Testnet)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, nil
		case Mainnet:
			mainnetRegistry, err := registry.GetChainRegistry(registry.InitiaL1Mainnet)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			state.Config["l1.chain_id"] = mainnetRegistry.GetChainId()
			if state.Config["l1.rpc_address"], err = mainnetRegistry.GetActiveRpc(); err != nil {
				return m, m.HandlePanic(err)
			}
			if state.Config["l1.grpc_address"], err = mainnetRegistry.GetActiveGrpc(); err != nil {
				return m, m.HandlePanic(err)
			}
			if state.Config["l1.lcd_address"], err = mainnetRegistry.GetActiveLcd(); err != nil {
				return m, m.HandlePanic(err)
			}
			if state.Config["l1.websocket"], err = mainnetRegistry.GetActiveWebSocket(); err != nil {
				return m, m.HandlePanic(err)
			}
			if state.Config["l1.gas_price.price"], err = mainnetRegistry.GetMinGasPriceByDenom(DefaultGasPriceDenom); err != nil {
				return m, m.HandlePanic(err)
			}
			state.Config["l1.gas_price.denom"] = DefaultGasPriceDenom

			model, err := NewSelectingL2Network(weavecontext.SetCurrentState(m.Ctx, state), registry.InitiaL1Testnet)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, nil
		}
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
	FillFromLCD SettingUpIBCChannelOption = "Fill in rollup LCD endpoint to detect all available IBC Channels"
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
			var networkRegistry *registry.ChainRegistry
			l1Registry, err := registry.GetChainRegistry(registry.InitiaL1Testnet)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			if state.Config["l1.chain_id"] == l1Registry.GetChainId() {
				networkRegistry, err = registry.GetChainRegistry(registry.InitiaL1Testnet)
				if err != nil {
					return m, m.HandlePanic(err)
				}
				info, err := networkRegistry.GetOpinitBridgeInfo(artifacts.BridgeID)
				if err != nil {
					return m, m.HandlePanic(err)
				}
				metadata, err = types.DecodeBridgeMetadata(info.BridgeConfig.Metadata)
				if err != nil {
					return m, m.HandlePanic(err)
				}
			} else {
				return m, m.HandlePanic(fmt.Errorf("not support for l1 %s", state.Config["l1.chain_id"]))
			}
			channelPairs := make([]types.IBCChannelPair, 0)
			for _, channel := range metadata.PermChannels {
				counterparty, err := networkRegistry.GetCounterPartyIBCChannel(channel.PortID, channel.ChannelID)
				if err != nil {
					return m, m.HandlePanic(err)
				}
				channelPairs = append(channelPairs, types.IBCChannelPair{
					L1: channel,
					L2: counterparty,
				})
			}
			return NewIBCChannelsCheckbox(weavecontext.SetCurrentState(m.Ctx, state), channelPairs), nil
		case FillFromLCD:
			analytics.TrackEvent(analytics.SettingUpIBCChannelsMethodSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "lcd"))
			return NewFillL2LCD(weavecontext.SetCurrentState(m.Ctx, state)), nil
		case Manually:
			analytics.TrackEvent(analytics.SettingUpIBCChannelsMethodSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "mannaul"))
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
		model := NewSettingUpRelayer(weavecontext.SetCurrentState(m.Ctx, state))
		return model, model.Init()
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
	tooltip := tooltip.RelayerRollupLCDTooltip
	m := &FillL2LCD{
		TextInput: ui.NewTextInput(false),
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  fmt.Sprintf("Specify rollup LCD endpoint %s", extra),
		extra:     extra,
	}
	m.WithTooltip(&tooltip)
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
			m.err = fmt.Errorf("unable to call the LCD endpoint '%s'. Please verify that the address is correct and reachablem", input.Text)
			return m, cmd
		}

		pairs := make([]types.IBCChannelPair, 0)
		for _, channel := range res.Channels {
			pairs = append(pairs, types.IBCChannelPair{
				L1: channel.Counterparty,
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

// getHermesBinaryURL is a utility function for determining Hermes binary URL
func getHermesBinaryURL(version string) (string, error) {
	baseURL := "https://github.com/informalsystems/hermes/releases/download"
	goarch := runtime.GOARCH
	goos := runtime.GOOS

	var binaryType string
	switch goarch {
	case "amd64":
		goarch = "x86_64"
	case "arm64":
		goarch = "aarch64"
	default:
		return "", fmt.Errorf("unsupported architecture: %s", goarch)
	}

	switch goos {
	case "darwin":
		binaryType = "apple-darwin"
	case "linux":
		binaryType = "unknown-linux-gnu"
	default:
		return "", fmt.Errorf("unsupported operating system: %s", goos)
	}

	fileName := fmt.Sprintf("hermes-%s-%s-%s.tar.gz", version, goarch, binaryType)
	return fmt.Sprintf("%s/%s/%s", baseURL, version, fileName), nil
}

func WaitSettingUpRelayer(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		state := weavecontext.GetCurrentState[State](ctx)

		// Get Hermes binary URL based on OS and architecture
		hermesURL, err := getHermesBinaryURL(HermesVersion)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("unable to get hermes binary URL: %s", err)}
		}

		// Get the user's home directory
		userHome, err := os.UserHomeDir()
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to get user home directory: %v", err)}
		}

		// Define paths
		weaveDataPath := filepath.Join(userHome, common.WeaveDataDirectory)
		tarballPath := filepath.Join(weaveDataPath, fmt.Sprintf("hermes-%s.tar.gz", HermesVersion))
		state.hermesBinaryPath = filepath.Join(weaveDataPath, "hermes")

		// Ensure the data directory exists
		if err = os.MkdirAll(weaveDataPath, 0755); err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to create data directory: %v", err)}
		}

		// Download and extract Hermes tarball
		if err = weaveio.DownloadAndExtractTarGz(hermesURL, tarballPath, weaveDataPath); err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to download and extract Hermes: %v", err)}
		}

		// Make the Hermes binary executable
		if err = os.Chmod(state.hermesBinaryPath, 0755); err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to set executable permissions for Hermes: %v", err)}
		}

		// Remove quarantine attribute on macOS
		if runtime.GOOS == "darwin" {
			if err = removeQuarantineAttribute(state.hermesBinaryPath); err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to remove quarantine attribute on macOS: %v", err)}
			}
		}

		// Create Hermes configuration
		err = createHermesConfig(state)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to create hermes config: %v", err)}
		}

		srv, err := service.NewService(service.Relayer)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to initialize service: %v", err)}
		}

		if err = srv.Create("", filepath.Join(userHome, HermesHome)); err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to create service: %v", err)}
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
		model, err := NewL1KeySelect(weavecontext.SetCurrentState(m.Ctx, state))
		if err != nil {
			return m, m.HandlePanic(err)
		}
		return model, nil
	}
	return m, cmd
}

func (m *SettingUpRelayer) View() string {
	state := weavecontext.GetCurrentState[State](m.Ctx)
	return m.WrapView(state.weave.Render() + "\n" + m.Loading.View())
}

func removeQuarantineAttribute(filePath string) error {
	cmd := exec.Command("xattr", "-l", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If listing attributes fails, assume no quarantine attribute
		return nil
	}

	// Check if com.apple.quarantine exists
	if !containsQuarantine(string(output)) {
		return nil
	}

	// Remove the quarantine attribute
	cmd = exec.Command("xattr", "-d", "com.apple.quarantine", filePath)
	return cmd.Run()
}

func containsQuarantine(attrs string) bool {
	return stringContains(attrs, "com.apple.quarantine")
}

func stringContains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && haystack[len(haystack)-len(needle):] == needle
}
