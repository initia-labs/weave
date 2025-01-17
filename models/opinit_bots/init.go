package opinit_bots

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/client"
	"github.com/initia-labs/weave/common"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/cosmosutils"
	"github.com/initia-labs/weave/io"
	"github.com/initia-labs/weave/registry"
	"github.com/initia-labs/weave/service"
	"github.com/initia-labs/weave/styles"
	"github.com/initia-labs/weave/tooltip"
	"github.com/initia-labs/weave/types"
	"github.com/initia-labs/weave/ui"
)

type OPInitBotInitOption string

const (
	ExecutorOPInitBotInitOption   OPInitBotInitOption = "Executor"
	ChallengerOPInitBotInitOption OPInitBotInitOption = "Challenger"
)

type OPInitBotInitSelector struct {
	weavecontext.BaseModel
	ui.Selector[OPInitBotInitOption]
	question string
}

var defaultExecutorFields = []*Field{
	// Listen Address
	{Name: "listen_address", Type: StringField, Question: "Specify listen address of the bot", Highlights: []string{"listen address"}, Placeholder: `Press tab to use "localhost:3000"`, DefaultValue: "localhost:3000", ValidateFn: common.ValidateEmptyString, Tooltip: &tooltip.ListenAddressTooltip},

	// L1 Node Configuration
	{Name: "l1_node.rpc_address", Type: StringField, Question: "Specify L1 RPC endpoint", Highlights: []string{"L1 RPC endpoint"}, Placeholder: "Add RPC address ex. http://localhost:26657", ValidateFn: common.ValidateURL, Tooltip: &tooltip.L1RPCEndpointTooltip},

	// L2 Node Configuration
	{Name: "l2_node.chain_id", Type: StringField, Question: "Specify rollup chain ID", Highlights: []string{"rollup chain ID"}, Placeholder: "Enter chain ID ex. rollup-1", ValidateFn: common.ValidateEmptyString},
	{Name: "l2_node.rpc_address", Type: StringField, Question: "Specify rollup RPC endpoint", Highlights: []string{"rollup RPC endpoint"}, Placeholder: `Press tab to use "http://localhost:26657"`, DefaultValue: "http://localhost:26657", ValidateFn: common.ValidateURL, Tooltip: &tooltip.RollupRPCEndpointTooltip},
	{Name: "l2_node.gas_price", Type: StringField, Question: "Specify rollup gas price", Highlights: []string{"rollup gas price"}, Placeholder: `Press tab to use "0.15umin"`, DefaultValue: "0.15umin", ValidateFn: common.ValidateDecCoin, Tooltip: &tooltip.RollupGasPriceTooltip},
}

var defaultChallengerFields = []*Field{
	// Listen Address
	{Name: "listen_address", Type: StringField, Question: "Specify listen address of the bot", Highlights: []string{"listen address"}, Placeholder: `Press tab to use "localhost:3001"`, DefaultValue: "localhost:3001", ValidateFn: common.ValidateEmptyString, Tooltip: &tooltip.ListenAddressTooltip},

	// L1 Node Configuration
	{Name: "l1_node.rpc_address", Type: StringField, Question: "Specify L1 RPC endpoint", Highlights: []string{"L1 RPC endpoint"}, Placeholder: "Add RPC address ex. http://localhost:26657", ValidateFn: common.ValidateURL, Tooltip: &tooltip.L1RPCEndpointTooltip},

	// L2 Node Configuration
	{Name: "l2_node.chain_id", Type: StringField, Question: "Specify rollup chain ID", Highlights: []string{"rollup chain ID"}, Placeholder: "Enter chain ID ex. rollup-1", ValidateFn: common.ValidateEmptyString},
	{Name: "l2_node.rpc_address", Type: StringField, Question: "Specify rollup RPC endpoint", Highlights: []string{"rollup RPC endpoint"}, Placeholder: `Press tab to use "http://localhost:26657"`, DefaultValue: "http://localhost:26657", ValidateFn: common.ValidateURL, Tooltip: &tooltip.RollupRPCEndpointTooltip},
}

func getField(fields []*Field, name string) (*Field, error) {
	for _, field := range fields {
		if field.Name == name {
			return field, nil
		}
	}
	return nil, fmt.Errorf("field %s not found", name)
}

func setFieldPrefillValue(fields []*Field, name, value string) error {
	field, err := getField(fields, name)
	if err != nil {
		return fmt.Errorf("Error setting prefill value for %s: %v\n", name, err)
	}
	field.PrefillValue = value
	return nil
}

func setFieldPlaceholder(fields []*Field, name, placeholder string) error {
	field, err := getField(fields, name)
	if err != nil {
		return fmt.Errorf("Error setting placeholder for %s: %v\n", name, err)
	}
	field.Placeholder = placeholder
	return nil
}

func NewOPInitBotInitSelector(ctx context.Context) (tea.Model, error) {
	tooltips := []ui.Tooltip{
		ui.NewTooltip("Executor", "Executes cross-chain transactions, ensuring that assets and data move securely between Initia and Minitias.", "", []string{}, []string{}, []string{}),
		ui.NewTooltip("Challenger", "Monitors for potential fraud, submitting proofs to dispute invalid state updates and maintaining network security.", "", []string{}, []string{}, []string{}),
	}
	return &OPInitBotInitSelector{
		Selector: ui.Selector[OPInitBotInitOption]{
			Options:    []OPInitBotInitOption{ExecutorOPInitBotInitOption, ChallengerOPInitBotInitOption},
			CannotBack: true,
			Tooltips:   &tooltips,
		},
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		question:  "Which bot would you like to run?",
	}, nil
}

func (m *OPInitBotInitSelector) GetQuestion() string {
	return m.question
}

func (m *OPInitBotInitSelector) Init() tea.Cmd {
	return nil
}

type BotConfigChainId struct {
	L1Node struct {
		ChainID string `json:"chain_id"`
	} `json:"l1_node"`
	L2Node struct {
		ChainID string `json:"chain_id"`
	} `json:"l2_node"`
	DANode struct {
		ChainID      string `json:"chain_id"`
		Bech32Prefix string `json:"bech32_prefix"`
	} `json:"da_node"`
}

func OPInitBotInitSelectExecutor(ctx context.Context) (tea.Model, error) {
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
	state.InitExecutorBot = true
	minitiaConfigPath, err := weavecontext.GetMinitiaArtifactsConfigJson(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load minitia config path: %v", err)
	}

	if io.FileOrFolderExists(minitiaConfigPath) {
		configData, err := os.ReadFile(minitiaConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read minitia config file: %v", err)
		}

		var minitiaConfig types.MinitiaConfig
		err = json.Unmarshal(configData, &minitiaConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse minitia config file: %v", err)
		}

		state.MinitiaConfig = &minitiaConfig
	}

	opInitHome, err := weavecontext.GetOPInitHome(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get OPinit home directory: %v", err)
	}
	state.dbPath = filepath.Join(opInitHome, "executor.db")
	if io.FileOrFolderExists(state.dbPath) {
		ctx = weavecontext.SetCurrentState(ctx, state)
		return NewDeleteDBSelector(ctx, "executor"), nil
	}

	executorJsonPath := filepath.Join(opInitHome, "executor.json")
	if io.FileOrFolderExists(executorJsonPath) {
		file, err := os.ReadFile(executorJsonPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read executor.json file: %v", err)
		}

		var botConfigChainId BotConfigChainId

		err = json.Unmarshal(file, &botConfigChainId)
		if err != nil {
			return nil, fmt.Errorf("failed to parse executor.json file: %v", err)
		}
		state.botConfig["l1_node.chain_id"] = botConfigChainId.L1Node.ChainID
		state.botConfig["l2_node.chain_id"] = botConfigChainId.L2Node.ChainID
		state.botConfig["da_node.chain_id"] = botConfigChainId.DANode.ChainID
		state.daIsCelestia = botConfigChainId.DANode.Bech32Prefix == "celestia"
		model, err := NewUseCurrentConfigSelector(weavecontext.SetCurrentState(ctx, state), "executor")
		if err != nil {
			return nil, err
		}
		return model, nil
	}

	state.ReplaceBotConfig = true

	if state.MinitiaConfig != nil {
		model, err := NewPrefillMinitiaConfig(weavecontext.SetCurrentState(ctx, state))
		if err != nil {
			return nil, err
		}
		return model, nil
	}

	return NewL1PrefillSelector(weavecontext.SetCurrentState(ctx, state))
}

func OPInitBotInitSelectChallenger(ctx context.Context) (tea.Model, error) {
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
	state.InitChallengerBot = true

	minitiaConfigPath, err := weavecontext.GetMinitiaArtifactsConfigJson(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load minitia config path: %v", err)
	}
	if io.FileOrFolderExists(minitiaConfigPath) {
		configData, err := os.ReadFile(minitiaConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read minitia config file: %v", err)
		}

		var minitiaConfig types.MinitiaConfig
		err = json.Unmarshal(configData, &minitiaConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse minitia config file: %v", err)
		}

		state.MinitiaConfig = &minitiaConfig
	}

	opInitHome, err := weavecontext.GetOPInitHome(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get OPinit home directory: %v", err)
	}
	state.dbPath = filepath.Join(opInitHome, "challenger.db")
	if io.FileOrFolderExists(state.dbPath) {
		ctx = weavecontext.SetCurrentState(ctx, state)
		return NewDeleteDBSelector(ctx, "challenger"), nil
	}

	challengerJsonPath := filepath.Join(opInitHome, "challenger.json")
	if io.FileOrFolderExists(challengerJsonPath) {
		file, err := os.ReadFile(challengerJsonPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read challenger json file: %v", err)
		}

		var botConfigChainId BotConfigChainId

		err = json.Unmarshal(file, &botConfigChainId)
		if err != nil {
			return nil, fmt.Errorf("failed to parse challenger json file: %v", err)
		}
		state.botConfig["l1_node.chain_id"] = botConfigChainId.L1Node.ChainID
		state.botConfig["l2_node.chain_id"] = botConfigChainId.L2Node.ChainID
		model, err := NewUseCurrentConfigSelector(weavecontext.SetCurrentState(ctx, state), "challenger")
		if err != nil {
			return nil, err
		}
		return model, nil
	}

	state.ReplaceBotConfig = true

	if state.MinitiaConfig != nil {
		model, err := NewPrefillMinitiaConfig(weavecontext.SetCurrentState(ctx, state))
		if err != nil {
			return nil, err
		}
		return model, nil
	}
	return NewL1PrefillSelector(weavecontext.SetCurrentState(ctx, state))
}

func (m *OPInitBotInitSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}
	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[OPInitBotsState](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{"bot"}, string(*selected)))
		analytics.AppendGlobalEventProperties(map[string]interface{}{
			analytics.BotTypeKey: string(*selected),
		})
		analytics.TrackEvent(analytics.OPInitBotInitSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, string(*selected)))

		switch *selected {
		case ExecutorOPInitBotInitOption:
			state.InitExecutorBot = true
			keyNames := make(map[string]bool)
			keyNames[BridgeExecutorKeyName] = true
			keyNames[OutputSubmitterKeyName] = true
			keyNames[BatchSubmitterKeyName] = true
			keyNames[OracleBridgeExecutorKeyName] = true

			var err error
			state.BotInfos, err = CheckIfKeysExist(state.BotInfos)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			for idx, botInfo := range state.BotInfos {
				if botInfo.KeyName == OracleBridgeExecutorKeyName && botInfo.IsNotExist {
					state.BotInfos[idx].IsSetup = true
				} else if keyNames[botInfo.KeyName] && botInfo.IsNotExist && !state.AddMinitiaConfig {
					state.BotInfos[idx].IsSetup = true
				} else {
					state.BotInfos[idx].IsSetup = false
				}
			}
			return NextUpdateOpinitBotKey(weavecontext.SetCurrentState(m.Ctx, state))
		case ChallengerOPInitBotInitOption:
			state.InitChallengerBot = true
			keyNames := make(map[string]bool)
			keyNames[ChallengerKeyName] = true

			var err error
			state.BotInfos, err = CheckIfKeysExist(state.BotInfos)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			for idx, botInfo := range state.BotInfos {
				if keyNames[botInfo.KeyName] && botInfo.IsNotExist && !state.AddMinitiaConfig {
					state.BotInfos[idx].IsSetup = true
				} else {
					state.BotInfos[idx].IsSetup = false
				}
			}
			return NextUpdateOpinitBotKey(weavecontext.SetCurrentState(m.Ctx, state))
		}
	}
	return m, cmd
}

func (m *OPInitBotInitSelector) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	m.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"bot"}, styles.Question) + m.Selector.View())
}

type DeleteDBOption string

const (
	DeleteDBOptionNo  = "No"
	DeleteDBOptionYes = "Yes, reset"
)

type DeleteDBSelector struct {
	ui.Selector[DeleteDBOption]
	weavecontext.BaseModel
	question string
	bot      string
}

func NewDeleteDBSelector(ctx context.Context, bot string) *DeleteDBSelector {
	return &DeleteDBSelector{
		Selector: ui.Selector[DeleteDBOption]{
			Options: []DeleteDBOption{
				DeleteDBOptionNo,
				DeleteDBOptionYes,
			},
		},
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  "Would you like to reset the database?",
		bot:       bot,
	}
}

func (m *DeleteDBSelector) GetQuestion() string {
	return m.question
}

func (m *DeleteDBSelector) Init() tea.Cmd {
	return nil
}

func (m *DeleteDBSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}
	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[OPInitBotsState](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{}, string(*selected)))
		switch *selected {
		case DeleteDBOptionNo:
			state.isDeleteDB = false
		case DeleteDBOptionYes:
			state.isDeleteDB = true
		}

		analytics.TrackEvent(analytics.ResetDBSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, state.isDeleteDB))

		opInitHome, err := weavecontext.GetOPInitHome(m.Ctx)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		executorJsonPath := filepath.Join(opInitHome, fmt.Sprintf("%s.json", m.bot))
		if io.FileOrFolderExists(executorJsonPath) {
			file, err := os.ReadFile(executorJsonPath)
			if err != nil {
				return m, m.HandlePanic(err)
			}

			var botConfigChainId BotConfigChainId

			err = json.Unmarshal(file, &botConfigChainId)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			state.botConfig["l1_node.chain_id"] = botConfigChainId.L1Node.ChainID
			state.botConfig["l2_node.chain_id"] = botConfigChainId.L2Node.ChainID
			state.botConfig["da_node.chain_id"] = botConfigChainId.DANode.ChainID
			state.daIsCelestia = botConfigChainId.DANode.Bech32Prefix == "celestia"
			model, err := NewUseCurrentConfigSelector(weavecontext.SetCurrentState(m.Ctx, state), m.bot)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, cmd
		}

		state.ReplaceBotConfig = true
		if state.MinitiaConfig != nil {
			model, err := NewPrefillMinitiaConfig(weavecontext.SetCurrentState(m.Ctx, state))
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, cmd
		}
		model, err := NewL1PrefillSelector(weavecontext.SetCurrentState(m.Ctx, state))
		if err != nil {
			return m, m.HandlePanic(err)
		}
		return model, cmd
	}

	return m, cmd
}

func (m *DeleteDBSelector) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{}, styles.Question) + m.Selector.View())
}

type UseCurrentConfigSelector struct {
	ui.Selector[string]
	weavecontext.BaseModel
	question   string
	configPath string
}

func NewUseCurrentConfigSelector(ctx context.Context, bot string) (*UseCurrentConfigSelector, error) {
	opInitHome, err := weavecontext.GetOPInitHome(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get OPinit Home: %w", err)
	}
	configPath := fmt.Sprintf("%s/%s.json", opInitHome, bot)
	return &UseCurrentConfigSelector{
		Selector: ui.Selector[string]{
			Options: []string{
				"use current file",
				"replace",
			},
		},
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   fmt.Sprintf("Existing %s detected. Would you like to use the current one or replace it?", configPath),
		configPath: configPath,
	}, nil
}

func (m *UseCurrentConfigSelector) GetQuestion() string {
	return m.question
}

func (m *UseCurrentConfigSelector) Init() tea.Cmd {
	return nil
}

func (m *UseCurrentConfigSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}
	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[OPInitBotsState](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{m.configPath}, *selected))
		analytics.TrackEvent(analytics.UseCurrentConfigSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, *selected))

		switch *selected {
		case "use current file":
			state.ReplaceBotConfig = false
			m.Ctx = weavecontext.SetCurrentState(m.Ctx, state)
			model, err := NewFetchL1StartHeightLoading(m.Ctx)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, model.Init()
		case "replace":
			state.ReplaceBotConfig = true
			m.Ctx = weavecontext.SetCurrentState(m.Ctx, state)
			if state.MinitiaConfig != nil {
				model, err := NewPrefillMinitiaConfig(m.Ctx)
				if err != nil {
					return m, m.HandlePanic(err)
				}
				return model, cmd
			}
			if state.InitExecutorBot || state.InitChallengerBot {
				model, err := NewL1PrefillSelector(m.Ctx)
				if err != nil {
					return m, m.HandlePanic(err)
				}
				return model, cmd
			}
			return m, cmd
		}
	}

	return m, cmd
}

func (m *UseCurrentConfigSelector) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{m.configPath}, styles.Question) + m.Selector.View())
}

type PrefillMinitiaConfigOption string

const (
	PrefillMinitiaConfigYes = "Yes, prefill"
	PrefillMinitiaConfigNo  = "No, skip"
)

type PrefillMinitiaConfig struct {
	ui.Selector[PrefillMinitiaConfigOption]
	weavecontext.BaseModel
	question string
}

func NewPrefillMinitiaConfig(ctx context.Context) (*PrefillMinitiaConfig, error) {
	artifactsDir, err := weavecontext.GetMinitiaArtifactsConfigJson(ctx)
	if err != nil {
		return nil, err
	}
	return &PrefillMinitiaConfig{
		Selector: ui.Selector[PrefillMinitiaConfigOption]{
			Options: []PrefillMinitiaConfigOption{
				PrefillMinitiaConfigYes,
				PrefillMinitiaConfigNo,
			},
		},
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  fmt.Sprintf("Existing %s detected. Would you like to use the data in this file to pre-fill some fields?", artifactsDir),
	}, nil
}

func (m *PrefillMinitiaConfig) GetQuestion() string {
	return m.question
}

func (m *PrefillMinitiaConfig) Init() tea.Cmd {
	return nil
}

func (m *PrefillMinitiaConfig) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}
	selected, cmd := m.Select(msg)
	if selected != nil {
		artifactsDir, err := weavecontext.GetMinitiaArtifactsConfigJson(m.Ctx)
		if err != nil {
			return m, m.HandlePanic(err)
		}

		state := weavecontext.PushPageAndGetState[OPInitBotsState](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{artifactsDir}, string(*selected)))

		switch *selected {
		case PrefillMinitiaConfigYes:
			analytics.TrackEvent(analytics.PrefillFromArtifactsSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, true))
			state.UsePrefilledMinitia = true
			minitiaConfig := state.MinitiaConfig
			state.botConfig["l1_node.chain_id"] = minitiaConfig.L1Config.ChainID
			state.botConfig["l1_node.rpc_address"] = minitiaConfig.L1Config.RpcUrl
			state.botConfig["l1_node.gas_price"] = minitiaConfig.L1Config.GasPrices
			if err = setFieldPrefillValue(defaultExecutorFields, "l1_node.rpc_address", minitiaConfig.L1Config.RpcUrl); err != nil {
				return m, m.HandlePanic(err)
			}
			if err = setFieldPrefillValue(defaultExecutorFields, "l2_node.chain_id", minitiaConfig.L2Config.ChainID); err != nil {
				return m, m.HandlePanic(err)
			}
			if err = setFieldPrefillValue(defaultExecutorFields, "l2_node.gas_price", "0.15"+minitiaConfig.L2Config.Denom); err != nil {
				return m, m.HandlePanic(err)
			}
			if err = setFieldPlaceholder(defaultExecutorFields, "l2_node.gas_price", "Press tab to use "+"\"0.15"+minitiaConfig.L2Config.Denom+"\""); err != nil {
				return m, m.HandlePanic(err)
			}

			if err = setFieldPrefillValue(defaultChallengerFields, "l1_node.rpc_address", minitiaConfig.L1Config.RpcUrl); err != nil {
				return m, m.HandlePanic(err)
			}
			if err = setFieldPrefillValue(defaultChallengerFields, "l2_node.chain_id", minitiaConfig.L2Config.ChainID); err != nil {
				return m, m.HandlePanic(err)
			}

			if minitiaConfig.OpBridge.BatchSubmissionTarget == "CELESTIA" {
				var network registry.ChainType
				l1ChainRegistry, err := registry.GetChainRegistry(registry.InitiaL1Testnet)
				if err != nil {
					return m, m.HandlePanic(err)
				}
				if l1ChainRegistry.GetChainId() == minitiaConfig.L1Config.ChainID {
					network = registry.CelestiaTestnet
				} else {
					network = registry.CelestiaMainnet
				}

				chainRegistry, err := registry.GetChainRegistry(network)
				if err != nil {
					return m, m.HandlePanic(err)
				}

				state.botConfig["da_node.chain_id"] = chainRegistry.GetChainId()
				activeRpc, err := chainRegistry.GetActiveRpc()
				if err != nil {
					return m, m.HandlePanic(err)
				}
				state.botConfig["da_node.rpc_address"] = activeRpc
				state.botConfig["da_node.bech32_prefix"] = chainRegistry.GetBech32Prefix()
				state.botConfig["da_node.gas_price"] = DefaultCelestiaGasPrices
				state.daIsCelestia = true
			} else {
				state.botConfig["da_node.chain_id"] = state.botConfig["l1_node.chain_id"]
				state.botConfig["da_node.rpc_address"] = state.botConfig["l1_node.rpc_address"]
				state.botConfig["da_node.bech32_prefix"] = "init"
				state.botConfig["da_node.gas_price"] = state.botConfig["l1_node.gas_price"]
			}
			m.Ctx = weavecontext.SetCurrentState(m.Ctx, state)
			if state.InitExecutorBot {
				return NewFieldInputModel(m.Ctx, defaultExecutorFields, NewFetchL1StartHeightLoading), cmd
			} else if state.InitChallengerBot {
				return NewFieldInputModel(m.Ctx, defaultChallengerFields, NewFetchL1StartHeightLoading), cmd

			}
		case PrefillMinitiaConfigNo:
			analytics.TrackEvent(analytics.PrefillFromArtifactsSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, false))
			model, err := NewL1PrefillSelector(weavecontext.SetCurrentState(m.Ctx, state))
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, cmd
		}

	}

	return m, cmd
}

func (m *PrefillMinitiaConfig) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	artifactsDir, err := weavecontext.GetMinitiaArtifactsConfigJson(m.Ctx)
	if err != nil {
		m.HandlePanic(err)
	}
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{artifactsDir}, styles.Question) + m.Selector.View())
}

type L1PrefillOption string

var (
	L1PrefillOptionTestnet L1PrefillOption = ""
	L1PrefillOptionCustom  L1PrefillOption = "Custom"
)

type L1PrefillSelector struct {
	ui.Selector[L1PrefillOption]
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewL1PrefillSelector(ctx context.Context) (*L1PrefillSelector, error) {
	initiaTestnetRegistry, err := registry.GetChainRegistry(registry.InitiaL1Testnet)
	if err != nil {
		return nil, fmt.Errorf("initia testnet registry: %w", err)
	}
	L1PrefillOptionTestnet = L1PrefillOption(fmt.Sprintf("Testnet (%s)", initiaTestnetRegistry.GetChainId()))
	return &L1PrefillSelector{
		Selector: ui.Selector[L1PrefillOption]{
			Options: []L1PrefillOption{
				L1PrefillOptionTestnet,
				// L1PrefillOptionCustom,
			},
		},
		BaseModel:  weavecontext.BaseModel{Ctx: ctx},
		question:   "Which L1 would you like your rollup to connect to?",
		highlights: []string{"L1", "rollup"},
	}, nil
}

func (m *L1PrefillSelector) GetQuestion() string {
	return m.question
}

func (m *L1PrefillSelector) Init() tea.Cmd {
	return nil
}

func (m *L1PrefillSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}
	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[OPInitBotsState](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), m.highlights, string(*selected)))

		var chainId, rpc, minGasPrice string
		switch *selected {
		case L1PrefillOptionTestnet:
			analytics.TrackEvent(analytics.L1PrefillSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "testnet"))

			chainRegistry, err := registry.GetChainRegistry(registry.InitiaL1Testnet)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			chainId = chainRegistry.GetChainId()
			rpc, err = chainRegistry.GetActiveRpc()
			if err != nil {
				return m, m.HandlePanic(err)
			}
			minGasPrice, err = chainRegistry.GetMinGasPriceByDenom(DefaultInitiaGasDenom)
			if err != nil {
				return m, m.HandlePanic(err)
			}
		case L1PrefillOptionCustom:
			analytics.TrackEvent(analytics.L1PrefillSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "custom"))
		}

		state.botConfig["l1_node.chain_id"] = chainId
		state.botConfig["l1_node.gas_price"] = minGasPrice

		if err := setFieldPrefillValue(defaultExecutorFields, "l1_node.rpc_address", rpc); err != nil {
			return m, m.HandlePanic(err)
		}
		if err := setFieldPrefillValue(defaultChallengerFields, "l1_node.rpc_address", rpc); err != nil {
			return m, m.HandlePanic(err)
		}

		m.Ctx = weavecontext.SetCurrentState(m.Ctx, state)
		if state.InitExecutorBot {
			return NewFieldInputModel(m.Ctx, defaultExecutorFields, NewSetDALayer), cmd
		} else if state.InitChallengerBot {
			return NewFieldInputModel(m.Ctx, defaultChallengerFields, NewFetchL1StartHeightLoading), cmd
		}

	}

	return m, cmd
}

func (m *L1PrefillSelector) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.Selector.View())
}

type DALayerNetwork string

const (
	Initia   DALayerNetwork = "Initia"
	Celestia DALayerNetwork = "Celestia"
	// Add other types as needed
)

type SetDALayer struct {
	ui.Selector[DALayerNetwork]
	weavecontext.BaseModel
	question string

	chainRegistry *registry.ChainRegistry
}

func NewSetDALayer(ctx context.Context) (tea.Model, error) {
	tooltips := []ui.Tooltip{
		tooltip.InitiaDALayerTooltip,
	}
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
	var network registry.ChainType
	l1ChainRegistry, err := registry.GetChainRegistry(registry.InitiaL1Testnet)
	if err != nil {
		return nil, fmt.Errorf("initia testnet registry: %w", err)
	}
	if l1ChainRegistry.GetChainId() == state.botConfig["l1_node.chain_id"] {
		network = registry.CelestiaTestnet
		tooltips = append(tooltips, tooltip.CelestiaTestnetDALayerTooltip)
	} else {
		network = registry.CelestiaMainnet
		tooltips = append(tooltips, tooltip.CelestiaMainnetDALayerTooltip)
	}

	chainRegistry, err := registry.GetChainRegistry(network)
	if err != nil {
		return nil, fmt.Errorf("celestia registry: %w", err)
	}

	return &SetDALayer{
		Selector: ui.Selector[DALayerNetwork]{
			Options: []DALayerNetwork{
				Initia,
				Celestia,
			},
			CannotBack: true,
			Tooltips:   &tooltips,
		},
		BaseModel:     weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		question:      "Which DA Layer would you like to use?",
		chainRegistry: chainRegistry,
	}, nil
}

func (m *SetDALayer) GetQuestion() string {
	return m.question
}

func (m *SetDALayer) Init() tea.Cmd {
	return nil
}

func (m *SetDALayer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}
	selected, cmd := m.Select(msg)
	if selected != nil {

		state := weavecontext.PushPageAndGetState[OPInitBotsState](m)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{"DA Layer"}, string(*selected)))
		analytics.TrackEvent(analytics.DALayerSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, string(*selected)))

		switch *selected {
		case Initia:
			state.botConfig["da_node.chain_id"] = state.botConfig["l1_node.chain_id"]
			state.botConfig["da_node.rpc_address"] = state.botConfig["l1_node.rpc_address"]
			state.botConfig["da_node.bech32_prefix"] = "init"
			state.botConfig["da_node.gas_price"] = state.botConfig["l1_node.gas_price"]
		case Celestia:
			state.botConfig["da_node.chain_id"] = m.chainRegistry.GetChainId()
			activeRpc, err := m.chainRegistry.GetActiveRpc()
			if err != nil {
				return m, m.HandlePanic(err)
			}
			state.botConfig["da_node.rpc_address"] = activeRpc
			state.botConfig["da_node.bech32_prefix"] = m.chainRegistry.GetBech32Prefix()
			state.botConfig["da_node.gas_price"] = DefaultCelestiaGasPrices
			state.daIsCelestia = true
		}
		model, err := NewFetchL1StartHeightLoading(weavecontext.SetCurrentState(m.Ctx, state))
		if err != nil {
			return m, m.HandlePanic(err)
		}
		return model, model.Init()

	}

	return m, cmd
}

func (m *SetDALayer) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"DA Layer"}, styles.Question) + m.Selector.View())
}

type FetchL1StartHeightLoading struct {
	weavecontext.BaseModel
	ui.Loading
}

func NewFetchL1StartHeightLoading(ctx context.Context) (tea.Model, error) {
	return &FetchL1StartHeightLoading{
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		Loading:   ui.NewLoading("Fetching Start Height for L1 ...", waitFetchL1StartHeight(ctx)),
	}, nil
}

func waitFetchL1StartHeight(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
		l2Rpc := state.botConfig["l2_node.rpc_address"]

		minitiadQuerier, err := cosmosutils.NewMinitiadQuerier()
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to initialize opchild querier: %w", err)}
		}

		var network registry.ChainType
		l1ChainRegistry, err := registry.GetChainRegistry(registry.InitiaL1Testnet)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("initia testnet registry: %w", err)}
		}
		if l1ChainRegistry.GetChainId() == state.botConfig["l1_node.chain_id"] {
			network = registry.InitiaL1Testnet
		} else {
			network = registry.InitiaL1Mainnet
		}

		gqlApi, err := registry.GetInitiaGraphQLFromType(network)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("cannot fetch initia GraphQL api: %w", err)}
		}
		gqlClient := client.NewGraphQLClient(gqlApi, client.NewHTTPClient())

		l1NextSequence, err := minitiadQuerier.QueryOPChildNextL1Sequence(l2Rpc)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to query l1 next sequence: %w", err)}
		}

		bridgeInfo, err := minitiadQuerier.QueryOPChildBridgeInfo(l2Rpc)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to query l1 bridge info: %w", err)}
		}

		if l1NextSequence == "1" {
			if state.UsePrefilledMinitia {
				artifactsJson, err := weavecontext.GetMinitiaArtifactsJson(ctx)
				if err != nil {
					return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to get artifacts json path: %w", err)}
				}

				data, err := os.ReadFile(artifactsJson)
				if err != nil {
					return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to read artifacts json: %w", err)}
				}

				var artifacts types.Artifacts
				if err := json.Unmarshal(data, &artifacts); err != nil {
					return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to unmarshal artifacts json: %w", err)}
				}

				state.L1StartHeight, err = strconv.Atoi(artifacts.ExecutorL1MonitorHeight)
				if err != nil {
					return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to parse l1 start height: %w", err)}
				}
			} else {
				state.L1StartHeight, _ = cosmosutils.QueryCreateBridgeHeight(gqlClient, bridgeInfo.BridgeID)
			}
		} else {
			sequence, err := strconv.Atoi(l1NextSequence)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to parse next sequence number: %w", err)}
			}
			state.L1StartHeight, _ = cosmosutils.QueryLatestDepositHeight(gqlClient, bridgeInfo.BridgeID, strconv.Itoa(sequence-1))
		}

		return ui.EndLoading{Ctx: weavecontext.SetCurrentState(ctx, state)}
	}
}

func (m *FetchL1StartHeightLoading) Init() tea.Cmd {
	return m.Loading.Init()
}

func (m *FetchL1StartHeightLoading) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}
	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		m.Ctx = m.Loading.EndContext
		state := weavecontext.PushPageAndGetState[OPInitBotsState](m)
		if state.L1StartHeight == 0 {
			return NewL1StartHeightInput(weavecontext.SetCurrentState(m.Ctx, state)), nil
		}
		model, err := NewStartingInitBot(weavecontext.SetCurrentState(m.Ctx, state))
		if err != nil {
			return m, m.HandlePanic(err)
		}
		return model, model.Init()
	}
	return m, cmd
}

func (m *FetchL1StartHeightLoading) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	return m.WrapView(state.weave.Render() + m.Loading.View())
}

type L1StartHeightInput struct {
	ui.TextInput
	weavecontext.BaseModel
	question   string
	highlights []string
}

func NewL1StartHeightInput(ctx context.Context) *L1StartHeightInput {
	toolTip := tooltip.L1StartHeightTooltip
	model := &L1StartHeightInput{
		TextInput:  ui.NewTextInput(true),
		BaseModel:  weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		question:   "Specify the L1 start height from which the bot should start processing",
		highlights: []string{"L1 start height"},
	}
	model.WithPlaceholder("Enter the start height")
	model.WithValidatorFn(common.IsValidInteger)
	model.WithTooltip(&toolTip)
	return model
}

func (m *L1StartHeightInput) GetQuestion() string {
	return m.question
}

func (m *L1StartHeightInput) Init() tea.Cmd {
	return nil
}

func (m *L1StartHeightInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[OPInitBotsState](m)

		// We can ignore the error here since it has already been validated in the input validator
		state.L1StartHeight, _ = strconv.Atoi(input.Text)
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), m.highlights, input.Text))
		model, err := NewStartingInitBot(weavecontext.SetCurrentState(m.Ctx, state))
		if err != nil {
			return m, m.HandlePanic(err)
		}
		return model, model.Init()
	}
	m.TextInput = input
	return m, cmd
}

func (m *L1StartHeightInput) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	m.TextInput.ViewTooltip(m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), m.highlights, styles.Question) + m.TextInput.View())
}

type StartingInitBot struct {
	weavecontext.BaseModel
	ui.Loading
}

func NewStartingInitBot(ctx context.Context) (tea.Model, error) {
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
	var bot string
	if state.InitExecutorBot {
		bot = "executor"
	} else {
		bot = "challenger"
	}

	return &StartingInitBot{
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		Loading:   ui.NewLoading(fmt.Sprintf("Setting up OPinit bot %s...", bot), WaitStartingInitBot(ctx)),
	}, nil
}

func WaitStartingInitBot(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1500 * time.Millisecond)

		state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
		configMap := state.botConfig

		if state.isDeleteDB {
			err := io.DeleteDirectory(state.dbPath)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to delete db: %w", err)}
			}
		}

		opInitHome, err := weavecontext.GetOPInitHome(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to load opinit home: %w", err)}
		}
		weaveDummyKeyPath := filepath.Join(opInitHome, "weave-dummy")
		l1KeyPath := filepath.Join(opInitHome, configMap["l1_node.chain_id"])
		l2KeyPath := filepath.Join(opInitHome, configMap["l2_node.chain_id"])

		err = io.CopyDirectory(weaveDummyKeyPath, l1KeyPath)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to copy dummy key for l1: %w", err)}
		}
		err = io.CopyDirectory(weaveDummyKeyPath, l2KeyPath)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to copy dummy key for l2: %w", err)}
		}

		if state.daIsCelestia {
			daKeyPath := filepath.Join(opInitHome, configMap["da_node.chain_id"])
			err = io.CopyDirectory(weaveDummyKeyPath, daKeyPath)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to copy dummy key for celestia: %w", err)}
			}
		}

		if state.InitExecutorBot {
			srv, err := service.NewService(service.OPinitExecutor)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to initialize service: %v", err)}
			}

			if err = srv.Create("", opInitHome); err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to create service: %v", err)}
			}

			if !state.ReplaceBotConfig {
				return ui.EndLoading{}
			}

			version, err := registry.GetOPInitBotsSpecVersion(state.botConfig["l1_node.chain_id"])
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to load l1_node.chain_id version: %v", err)}
			}

			config := ExecutorConfig{
				Version: version,
				Server: ServerConfig{
					Address:      configMap["listen_address"],
					AllowOrigins: "*",
					AllowHeaders: "Origin, Content-Type, Accept",
					AllowMethods: "GET",
				},
				L1Node: NodeSettings{
					ChainID:       configMap["l1_node.chain_id"],
					RPCAddress:    configMap["l1_node.rpc_address"],
					Bech32Prefix:  "init",
					GasPrice:      configMap["l1_node.gas_price"],
					GasAdjustment: 1.5,
					TxTimeout:     60,
				},
				L2Node: NodeSettings{
					ChainID:       configMap["l2_node.chain_id"],
					RPCAddress:    configMap["l2_node.rpc_address"],
					Bech32Prefix:  "init",
					GasPrice:      configMap["l2_node.gas_price"],
					GasAdjustment: 1.5,
					TxTimeout:     60,
				},
				DANode: NodeSettings{
					ChainID:       configMap["da_node.chain_id"],
					RPCAddress:    configMap["da_node.rpc_address"],
					Bech32Prefix:  configMap["da_node.bech32_prefix"],
					GasPrice:      configMap["da_node.gas_price"],
					GasAdjustment: 1.5,
					TxTimeout:     60,
				},
				BridgeExecutor:                BridgeExecutorKeyName,
				OracleBridgeExecutor:          OracleBridgeExecutorKeyName,
				MaxChunks:                     5000,
				MaxChunkSize:                  300000,
				MaxSubmissionTime:             3600,
				L1StartHeight:                 state.L1StartHeight,
				L2StartHeight:                 1,
				BatchStartHeight:              1,
				DisableDeleteFutureWithdrawal: false,
				DisableAutoSetL1Height:        true,
				DisableBatchSubmitter:         false,
				DisableOutputSubmitter:        false,
			}
			configBz, err := json.MarshalIndent(config, "", " ")
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to marshal config: %v", err)}
			}

			configFilePath := filepath.Join(opInitHome, "executor.json")
			if err = os.WriteFile(configFilePath, configBz, 0600); err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to write config file: %v", err)}
			}

			userHome, err := os.UserHomeDir()
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to get user home dir: %v", err)}
			}
			binaryPath := filepath.Join(userHome, common.WeaveDataDirectory, fmt.Sprintf("opinitd@%s", OpinitBotBinaryVersion), AppName)
			if address, err := cosmosutils.OPInitGetAddressForKey(binaryPath, OracleBridgeExecutorKeyName, opInitHome); err == nil {
				// TODO: revisit error
				_ = cosmosutils.OPInitGrantOracle(binaryPath, address, opInitHome)
			}
		} else if state.InitChallengerBot {
			srv, err := service.NewService(service.OPinitChallenger)
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to initialize service: %v", err)}
			}

			if err = srv.Create("", opInitHome); err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to create service: %v", err)}
			}

			if !state.ReplaceBotConfig {
				return ui.EndLoading{}
			}

			version, err := registry.GetOPInitBotsSpecVersion(state.botConfig["l1_node.chain_id"])
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to load l1_node.chain_id version: %v", err)}
			}
			config := ChallengerConfig{
				Version: version,
				Server: ServerConfig{
					Address:      configMap["listen_address"],
					AllowOrigins: "*",
					AllowHeaders: "Origin, Content-Type, Accept",
					AllowMethods: "GET",
				},
				L1Node: NodeConfig{
					ChainID:      configMap["l1_node.chain_id"],
					RPCAddress:   configMap["l1_node.rpc_address"],
					Bech32Prefix: "init",
				},
				L2Node: NodeConfig{
					ChainID:      configMap["l2_node.chain_id"],
					RPCAddress:   configMap["l2_node.rpc_address"],
					Bech32Prefix: "init",
				},
				L1StartHeight:          state.L1StartHeight,
				L2StartHeight:          1,
				DisableAutoSetL1Height: true,
			}
			configBz, err := json.MarshalIndent(config, "", " ")
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to marshal config: %v", err)}
			}

			configFilePath := filepath.Join(opInitHome, "challenger.json")
			if err = os.WriteFile(configFilePath, configBz, 0600); err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to write config file: %v", err)}
			}
		}
		return ui.EndLoading{}
	}
}

func (m *StartingInitBot) Init() tea.Cmd {
	return m.Loading.Init()
}

func (m *StartingInitBot) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}
	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		srv, err := service.NewService(service.OPinitExecutor)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		_ = srv.PruneLogs()
		return NewOPinitBotSuccessful(m.Ctx), nil
	}
	return m, cmd
}

func (m *StartingInitBot) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	return m.WrapView(state.weave.Render() + m.Loading.View())
}

type OPinitBotSuccessful struct {
	weavecontext.BaseModel
}

func NewOPinitBotSuccessful(ctx context.Context) *OPinitBotSuccessful {
	return &OPinitBotSuccessful{
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
	}
}

func (m *OPinitBotSuccessful) Init() tea.Cmd {
	return nil
}

func (m *OPinitBotSuccessful) Update(_ tea.Msg) (tea.Model, tea.Cmd) {
	return m, tea.Quit
}

func (m *OPinitBotSuccessful) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)

	botConfigFileName := "executor"
	if state.InitChallengerBot {
		botConfigFileName = "challenger"
	}

	opInitHome, err := weavecontext.GetOPInitHome(m.Ctx)
	if err != nil {
		m.HandlePanic(err)
	}

	return m.WrapView(state.weave.Render() + styles.RenderPrompt(fmt.Sprintf("OPInit bot setup successfully. Config file is saved at %s. Feel free to modify it as needed.", filepath.Join(opInitHome, fmt.Sprintf("%s.json", botConfigFileName))), []string{}, styles.Completed) + "\n" + styles.RenderPrompt("You can start the bot by running `weave opinit start "+botConfigFileName+"`", []string{}, styles.Completed) + "\n")
}

// SetupOPInitBotsMissingKey handles the loading and setup of OPInit bots
type SetupOPInitBotsMissingKey struct {
	weavecontext.BaseModel
	ui.Loading
}

// NewSetupOPInitBotsMissingKey initializes a new SetupOPInitBots with context
func NewSetupOPInitBotsMissingKey(ctx context.Context) *SetupOPInitBotsMissingKey {
	return &SetupOPInitBotsMissingKey{
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		Loading:   ui.NewLoading("Downloading binary and adding keys...", WaitSetupOPInitBotsMissingKey(ctx)),
	}
}

func (m *SetupOPInitBotsMissingKey) Init() tea.Cmd {
	return m.Loading.Init()
}

func handleBotInitSelection(ctx context.Context, state OPInitBotsState) (tea.Model, error) {
	if state.InitExecutorBot {
		return OPInitBotInitSelectExecutor(ctx)
	}
	return OPInitBotInitSelectChallenger(ctx)
}

func (m *SetupOPInitBotsMissingKey) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}
	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		state := weavecontext.GetCurrentState[OPInitBotsState](m.Loading.EndContext)
		oracleBotInfo := GetBotInfo(BotInfos, OracleBridgeExecutor)
		if (state.AddMinitiaConfig && !oracleBotInfo.IsNewKey()) || (!state.AddMinitiaConfig && len(state.SetupOpinitResponses) == 0) {
			model, err := handleBotInitSelection(m.Loading.EndContext, state)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, nil
		}
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "enter" {
				model, err := handleBotInitSelection(m.Loading.EndContext, state)
				if err != nil {
					return m, m.HandlePanic(err)
				}
				return model, nil
			}
		}
	}
	return m, cmd
}

func (m *SetupOPInitBotsMissingKey) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	oracleBotInfo := GetBotInfo(BotInfos, OracleBridgeExecutor)
	if state.AddMinitiaConfig && !oracleBotInfo.IsNewKey() {
		return state.weave.Render() + "\n"
	}
	if len(state.SetupOpinitResponses) > 0 {
		mnemonicText := ""
		for _, botName := range BotNames {
			if res, ok := state.SetupOpinitResponses[botName]; ok {
				keyInfo := strings.Split(res, "\n")
				address := strings.Split(keyInfo[0], ": ")
				mnemonicText += renderMnemonic(string(botName), address[1], keyInfo[1])
			}
		}

		return m.WrapView(state.weave.Render() + "\n" + styles.BoldUnderlineText("Important", styles.Yellow) + "\n" +
			styles.Text("Write down these mnemonic phrases and store them in a safe place. \nIt is the only way to recover your system keys.", styles.Yellow) + "\n\n" +
			mnemonicText + "\nPress enter to go next step\n")
	}
	return m.WrapView(state.weave.Render() + "\n")
}

func WaitSetupOPInitBotsMissingKey(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
		userHome, err := os.UserHomeDir()
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to get user home directory: %v", err)}
		}

		binaryPath := filepath.Join(userHome, common.WeaveDataDirectory, fmt.Sprintf("opinitd@%s", OpinitBotBinaryVersion), AppName)

		opInitHome, err := weavecontext.GetOPInitHome(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to get opinit home directory: %v", err)}
		}
		for _, info := range state.BotInfos {
			if info.Mnemonic != "" {
				res, err := cosmosutils.OPInitRecoverKeyFromMnemonic(binaryPath, info.KeyName, info.Mnemonic, info.DALayer == string(CelestiaLayerOption), opInitHome)
				if err != nil {
					return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to recover key from mnemonic: %v", err)}
				}
				state.SetupOpinitResponses[info.BotName] = res
				continue
			}
			if info.IsGenerateKey {
				res, err := cosmosutils.OPInitAddOrReplace(binaryPath, info.KeyName, info.DALayer == string(CelestiaLayerOption), opInitHome)
				if err != nil {
					return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to add or replace key: %v", err)}

				}
				state.SetupOpinitResponses[info.BotName] = res
				continue
			}
		}

		return ui.EndLoading{
			Ctx: ctx,
		}
	}
}

func InitializeExecutorWithConfig(config ExecutorConfig, keyFile *KeyFile, opInitHome, userHome string) error {
	binaryPath, err := ensureOPInitBotsBinary(userHome)
	if err != nil {
		return err
	}
	_, err = cosmosutils.OPInitRecoverKeyFromMnemonic(binaryPath, BridgeExecutorKeyName, keyFile.BridgeExecutor, false, opInitHome)
	if err != nil {
		return err
	}
	_, err = cosmosutils.OPInitRecoverKeyFromMnemonic(binaryPath, OutputSubmitterKeyName, keyFile.OutputSubmitter, false, opInitHome)
	if err != nil {
		return err
	}
	_, err = cosmosutils.OPInitRecoverKeyFromMnemonic(binaryPath, BatchSubmitterKeyName, keyFile.BatchSubmitter, config.DANode.Bech32Prefix != "init", opInitHome)
	if err != nil {
		return err
	}
	_, err = cosmosutils.OPInitRecoverKeyFromMnemonic(binaryPath, OracleBridgeExecutorKeyName, keyFile.OracleBridgeExecutor, false, opInitHome)
	if err != nil {
		return err
	}

	// File paths and other initialization steps
	weaveDummyKeyPath := filepath.Join(opInitHome, "weave-dummy")
	l1KeyPath := filepath.Join(opInitHome, config.L1Node.ChainID)
	l2KeyPath := filepath.Join(opInitHome, config.L2Node.ChainID)

	err = io.CopyDirectory(weaveDummyKeyPath, l1KeyPath)
	if err != nil {
		return fmt.Errorf("failed to copy dummy key for l1: %w", err)
	}

	err = io.CopyDirectory(weaveDummyKeyPath, l2KeyPath)
	if err != nil {
		return fmt.Errorf("failed to copy dummy key for l2: %w", err)
	}

	// If DA is Celestia, copy keys for DA node
	if config.DANode.Bech32Prefix != "init" {
		daKeyPath := filepath.Join(opInitHome, config.DANode.ChainID)
		err = io.CopyDirectory(weaveDummyKeyPath, daKeyPath)
		if err != nil {
			return fmt.Errorf("failed to copy dummy key for celestia: %w", err)
		}
	}

	// Additional initialization steps for executor
	srv, err := service.NewService(service.OPinitExecutor)
	if err != nil {
		return fmt.Errorf("failed to initialize service: %v", err)
	}

	if err = srv.Create("", opInitHome); err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}

	// Write config to file
	configFilePath := filepath.Join(opInitHome, "executor.json")
	configBz, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err = os.WriteFile(configFilePath, configBz, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	// prune existing logs, ignore error
	_ = srv.PruneLogs()

	return nil
}

func InitializeChallengerWithConfig(config ChallengerConfig, keyFile *KeyFile, opInitHome, userHome string) error {
	binaryPath, err := ensureOPInitBotsBinary(userHome)
	if err != nil {
		return err
	}
	_, err = cosmosutils.OPInitRecoverKeyFromMnemonic(binaryPath, ChallengerKeyName, keyFile.Challenger, false, opInitHome)
	if err != nil {
		return err
	}

	// File paths and other initialization steps
	weaveDummyKeyPath := filepath.Join(opInitHome, "weave-dummy")
	l1KeyPath := filepath.Join(opInitHome, config.L1Node.ChainID)
	l2KeyPath := filepath.Join(opInitHome, config.L2Node.ChainID)

	err = io.CopyDirectory(weaveDummyKeyPath, l1KeyPath)
	if err != nil {
		return fmt.Errorf("failed to copy dummy key for l1: %w", err)
	}

	err = io.CopyDirectory(weaveDummyKeyPath, l2KeyPath)
	if err != nil {
		return fmt.Errorf("failed to copy dummy key for l2: %w", err)
	}

	// Additional initialization steps for executor
	srv, err := service.NewService(service.OPinitChallenger)
	if err != nil {
		return fmt.Errorf("failed to initialize service: %v", err)
	}

	if err = srv.Create("", opInitHome); err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}

	// Write config to file
	configFilePath := filepath.Join(opInitHome, "challenger.json")
	configBz, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err = os.WriteFile(configFilePath, configBz, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	// prune existing logs, ignore error
	_ = srv.PruneLogs()

	return nil
}

func ensureOPInitBotsBinary(userHome string) (string, error) {
	// Define paths
	binaryPath := GetBinaryPath(userHome)
	weaveDataPath := filepath.Join(userHome, common.WeaveDataDirectory)
	tarballPath := filepath.Join(weaveDataPath, "opinitd.tar.gz")
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	extractedPath := filepath.Join(weaveDataPath, fmt.Sprintf("opinitd@%s", OpinitBotBinaryVersion))

	// Check if the binary already exists
	if _, err := os.Stat(binaryPath); err == nil {
		// Binary exists, no need to download
		return binaryPath, nil
	}
	fmt.Printf("Downloading opinit bot\n")
	// If the binary doesn't exist, proceed to download and extract
	// Check if the extracted directory exists, if not, create it
	if _, err := os.Stat(extractedPath); os.IsNotExist(err) {
		err := os.MkdirAll(extractedPath, os.ModePerm)
		if err != nil {
			return binaryPath, fmt.Errorf("failed to create weave data directory: %v", err)
		}
	}

	// Get the binary download URL
	url, err := getBinaryURL(OpinitBotBinaryVersion, goos, goarch)
	if err != nil {
		return binaryPath, fmt.Errorf("failed to get binary URL: %v", err)
	}

	// Download and extract the binary
	if err := io.DownloadAndExtractTarGz(url, tarballPath, extractedPath); err != nil {
		return binaryPath, fmt.Errorf("failed to download and extract binary: %v", err)
	}

	// Set the correct file permissions for the binary
	err = os.Chmod(binaryPath, 0755) // 0755 ensuring read, write, execute permissions for the owner, and read-execute for group/others
	if err != nil {
		return binaryPath, fmt.Errorf("failed to set permissions for binary: %v", err)
	}

	// Create a symlink to the binary (if needed)
	if err := cosmosutils.SetSymlink(binaryPath); err != nil {
		return binaryPath, err
	}
	fmt.Printf("Successfully download opinit bot\n")
	return binaryPath, nil
}

func PrepareExecutorBotKey(ctx context.Context) (tea.Model, error) {
	return PrepareKey(ctx, true)
}

func PrepareChallengerBotKey(ctx context.Context) (tea.Model, error) {
	return PrepareKey(ctx, false)
}

func PrepareKey(ctx context.Context, isExecutor bool) (tea.Model, error) {
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
	keyNames := make(map[string]bool)
	var err error
	if isExecutor {
		state.InitExecutorBot = true
		keyNames[BridgeExecutorKeyName] = true
		keyNames[OutputSubmitterKeyName] = true
		keyNames[BatchSubmitterKeyName] = true
		keyNames[OracleBridgeExecutorKeyName] = true
		state.BotInfos, err = CheckIfKeysExist(state.BotInfos)
		if err != nil {
			return nil, err
		}

		for idx, botInfo := range state.BotInfos {
			if botInfo.KeyName == OracleBridgeExecutorKeyName && botInfo.IsNotExist {
				state.BotInfos[idx].IsSetup = true
			} else if keyNames[botInfo.KeyName] && botInfo.IsNotExist && !state.AddMinitiaConfig {
				state.BotInfos[idx].IsSetup = true
			} else {
				state.BotInfos[idx].IsSetup = false
			}
		}
	} else {
		state.InitChallengerBot = true
		keyNames[ChallengerKeyName] = true
		state.BotInfos, err = CheckIfKeysExist(state.BotInfos)
		if err != nil {
			return nil, err
		}

		for idx, botInfo := range state.BotInfos {
			if keyNames[botInfo.KeyName] && botInfo.IsNotExist && !state.AddMinitiaConfig {
				state.BotInfos[idx].IsSetup = true
			} else {
				state.BotInfos[idx].IsSetup = false
			}
		}
	}

	for idx := 0; idx < len(state.BotInfos); idx++ {
		if state.BotInfos[idx].IsSetup {
			return NewRecoverKeySelector(weavecontext.SetCurrentState(ctx, state), idx), nil
		}
	}
	model := NewSetupOPInitBotsMissingKey(weavecontext.SetCurrentState(ctx, state))
	return model, nil

}
