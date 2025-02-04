package opinit_bots

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/common"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/cosmosutils"
	"github.com/initia-labs/weave/io"
	"github.com/initia-labs/weave/styles"
	"github.com/initia-labs/weave/types"
	"github.com/initia-labs/weave/ui"
)

func ProcessMinitiaConfig(ctx context.Context, nextModelFunc func(ctx context.Context) (tea.Model, error)) (tea.Model, error) {
	minitiaConfigPath, err := weavecontext.GetMinitiaArtifactsConfigJson(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load minitia config json: %w", err)
	}
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)

	// no config file, proceed to next model
	if !io.FileOrFolderExists(minitiaConfigPath) {
		model, err := nextModelFunc(weavecontext.SetCurrentState(ctx, state))
		if err != nil {
			return nil, err
		}
		return model, nil
	}

	// Load the config if found
	configData, err := os.ReadFile(minitiaConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read minitia config: %w", err)
	}

	var minitiaConfig types.MinitiaConfig
	err = json.Unmarshal(configData, &minitiaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal minitia config: %w", err)
	}

	// Set the loaded config to the state variable
	state.MinitiaConfig = &minitiaConfig
	return NewProcessingMinitiaConfig(weavecontext.SetCurrentState(ctx, state), nextModelFunc)
}

type AddMinitiaKeyOption string

const (
	YesAddMinitiaKeyOption AddMinitiaKeyOption = "Yes, use detected keys"
	NoAddMinitiaKeyOption  AddMinitiaKeyOption = "No, skip"
)

type ProcessingMinitiaConfig struct {
	weavecontext.BaseModel
	ui.Selector[AddMinitiaKeyOption]
	question      string
	nextModelFunc func(ctx context.Context) (tea.Model, error)
}

func assignBotInfo(botInfo *BotInfo, minitiaConfig *types.MinitiaConfig) {
	botInfo.IsNotExist = false
	botInfo.Mnemonic = getMnemonicForBot(botInfo.BotName, minitiaConfig)

	// Set DA Layer for BatchSubmitter
	if botInfo.BotName == BatchSubmitter {
		botInfo.DALayer = getDALayer(minitiaConfig.SystemKeys.BatchSubmitter.L1Address)
	}
}

func getMnemonicForBot(botName BotName, minitiaConfig *types.MinitiaConfig) string {
	switch botName {
	case BridgeExecutor:
		return minitiaConfig.SystemKeys.BridgeExecutor.Mnemonic
	case OutputSubmitter:
		return minitiaConfig.SystemKeys.OutputSubmitter.Mnemonic
	case BatchSubmitter:
		return minitiaConfig.SystemKeys.BatchSubmitter.Mnemonic
	case Challenger:
		return minitiaConfig.SystemKeys.Challenger.Mnemonic
	default:
		return ""
	}
}

func getDALayer(address string) string {
	if strings.HasPrefix(address, "initia") {
		return string(InitiaLayerOption)
	}
	return string(CelestiaLayerOption)
}

func NewProcessingMinitiaConfig(ctx context.Context, nextModelFunc func(ctx context.Context) (tea.Model, error)) (*ProcessingMinitiaConfig, error) {
	artifactsDir, err := weavecontext.GetMinitiaArtifactsConfigJson(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load artifacts directory: %w", err)
	}
	return &ProcessingMinitiaConfig{
		Selector: ui.Selector[AddMinitiaKeyOption]{
			Options: []AddMinitiaKeyOption{
				YesAddMinitiaKeyOption,
				NoAddMinitiaKeyOption,
			},
		},
		BaseModel:     weavecontext.BaseModel{Ctx: ctx},
		question:      fmt.Sprintf("Existing keys in %s detected. Would you like to add these to the keyring before proceeding?", artifactsDir),
		nextModelFunc: nextModelFunc,
	}, nil
}

func (m *ProcessingMinitiaConfig) GetQuestion() string {
	return m.question
}

func (m *ProcessingMinitiaConfig) Init() tea.Cmd {
	return nil
}

func (m *ProcessingMinitiaConfig) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}

	// Handle selection logic
	selected, cmd := m.Select(msg)
	if selected != nil {
		artifactsDir, err := weavecontext.GetMinitiaArtifactsConfigJson(m.Ctx)
		if err != nil {
			return m, m.HandlePanic(err)
		}

		state := weavecontext.PushPageAndGetState[OPInitBotsState](m)
		state.weave.PushPreviousResponse(
			styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{artifactsDir}, string(*selected)),
		)

		switch *selected {
		case YesAddMinitiaKeyOption:
			analytics.TrackEvent(analytics.ImportKeysFromArtifactsSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "yes"))
			// Iterate through botInfos and add relevant keys
			for idx := range state.BotInfos {
				if state.BotInfos[idx].BotName != OracleBridgeExecutor {
					assignBotInfo(&state.BotInfos[idx], state.MinitiaConfig)
				}
			}
			state.AddMinitiaConfig = true
			nextModel, err := m.nextModelFunc(weavecontext.SetCurrentState(m.Ctx, state))
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return nextModel, nextModel.Init()

		case NoAddMinitiaKeyOption:
			analytics.TrackEvent(analytics.ImportKeysFromArtifactsSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "no"))
			nextModel, err := m.nextModelFunc(weavecontext.SetCurrentState(m.Ctx, state))
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return nextModel, nextModel.Init()
		}
	}

	return m, cmd
}

func (m *ProcessingMinitiaConfig) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	m.Selector.ViewTooltip(m.Ctx)
	artifactsDir, err := weavecontext.GetMinitiaArtifactsConfigJson(m.Ctx)
	if err != nil {
		m.HandlePanic(err)
	}
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{artifactsDir}, styles.Question) + m.Selector.View())
}

func NextUpdateOpinitBotKey(ctx context.Context) (tea.Model, tea.Cmd) {
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
	for idx := 0; idx < len(state.BotInfos); idx++ {
		if state.BotInfos[idx].IsSetup {
			return NewRecoverKeySelector(ctx, idx), nil
		}
	}
	if state.InitExecutorBot || state.InitChallengerBot {
		model := NewSetupOPInitBotsMissingKey(ctx)
		return model, model.Init()
	}

	model := NewSetupOPInitBots(ctx)
	return model, model.Init()
}

type SetupBotCheckbox struct {
	weavecontext.BaseModel
	ui.CheckBox[string]
	question string
}

func NewSetupBotCheckbox(ctx context.Context) (tea.Model, error) {
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
	checkBoxOptions := make([]string, 0)
	for idx, botInfo := range state.BotInfos {
		if botInfo.IsNotExist {
			checkBoxOptions = append(checkBoxOptions, string(BotNames[idx]))
		} else {
			checkBoxOptions = append(checkBoxOptions, fmt.Sprintf("%s (key exists)", BotNames[idx]))
		}
	}

	question := "Which bots would you like to set/override?"
	tooltips := []ui.Tooltip{
		ui.NewTooltip("Bridge Executor", "Monitors the L1 and rollup transactions, facilitates token bridging and withdrawals between the minitia and Initia L1 chain, and also relays oracle price feed to rollup.", "", []string{}, []string{}, []string{}),
		ui.NewTooltip("Output Submitter", "Submits rollup output roots to L1 for verification and potential challenges. If the submitted output remains unchallenged beyond the output finalization period, it is considered finalized and immutable.", "", []string{}, []string{}, []string{}),
		ui.NewTooltip("Batch Submitter", "Submits block and transactions data in batches into a chain to ensure Data Availability. Currently, submissions can be made to Initia L1 or Celestia.", "", []string{}, []string{}, []string{}),
		ui.NewTooltip("Challenger", "Prevents misconduct and invalid minitia state submissions by monitoring for output proposals and challenging any that are invalid.", "", []string{}, []string{}, []string{}),
		ui.NewTooltip("Oracle Bridge Executor", "Relays oracle transaction from L1 to rollup. If rollup is using oracle, you need to set this field.", "", []string{}, []string{}, []string{}),
	}

	checkBox := ui.NewCheckBox(checkBoxOptions)
	checkBox.WithTooltip(&tooltips)
	return &SetupBotCheckbox{
		CheckBox:  *checkBox,
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  question,
	}, nil
}

func (m *SetupBotCheckbox) GetQuestion() string {
	return m.question
}

func (m *SetupBotCheckbox) Init() tea.Cmd {
	return nil
}

func (m *SetupBotCheckbox) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}

	cb, cmd, done := m.Select(msg)
	if done {
		artifactsDir, err := weavecontext.GetMinitiaArtifactsConfigJson(m.Ctx)
		if err != nil {
			return m, m.HandlePanic(err)
		}

		state := weavecontext.PushPageAndGetState[OPInitBotsState](m)
		state.weave.PushPreviousResponse(
			styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{"bots", "set", "override", artifactsDir}, cb.GetSelectedString()),
		)

		empty := true
		// Update the state based on the user's selections
		for idx, isSelected := range cb.Selected {
			if isSelected {
				empty = false
				state.BotInfos[idx].IsSetup = true
			}
		}

		m.Ctx = weavecontext.SetCurrentState(m.Ctx, state)
		// If no bots were selected, return to SetupOPInitBots
		if empty {
			model := NewSetupOPInitBots(m.Ctx)
			return model, model.Init()
		}

		// Proceed to the next step
		return NextUpdateOpinitBotKey(m.Ctx)
	}

	return m, cmd
}

// View renders the current prompt and selection options
func (m *SetupBotCheckbox) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	m.CheckBox.ViewTooltip(m.Ctx)
	artifactsDir, err := weavecontext.GetMinitiaArtifactsConfigJson(m.Ctx)
	if err != nil {
		m.HandlePanic(err)
	}
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"bots", "set", "override", artifactsDir}, styles.Question) + "\n\n" + m.CheckBox.ViewWithBottom("For bots with an existing key, selecting them will override the key."))
}

type RecoverKeySelector struct {
	weavecontext.BaseModel
	ui.Selector[string]
	idx      int
	question string
}

func NewRecoverKeySelector(ctx context.Context, idx int) *RecoverKeySelector {
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
	return &RecoverKeySelector{
		Selector: ui.Selector[string]{
			Options: []string{
				"Generate new system key",
				"Import existing key " + styles.Text("(you will be prompted to enter your mnemonic)", styles.Gray),
			},
		},
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		idx:       idx,
		question:  fmt.Sprintf(`Please select an option for the system key for %s`, state.BotInfos[idx].BotName),
	}
}

func (m *RecoverKeySelector) GetQuestion() string {
	return m.question
}

func (m *RecoverKeySelector) Init() tea.Cmd {
	return nil
}

func (m *RecoverKeySelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		state := weavecontext.PushPageAndGetState[OPInitBotsState](m)

		if *selected == "Generate new system key" {
			analytics.TrackEvent(analytics.RecoverKeySelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "generate"))
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{string(state.BotInfos[m.idx].BotName)}, *selected))

			state.BotInfos[m.idx].IsGenerateKey = true
			state.BotInfos[m.idx].Mnemonic = ""
			state.BotInfos[m.idx].IsSetup = false

			m.Ctx = weavecontext.SetCurrentState(m.Ctx, state)
			if state.BotInfos[m.idx].BotName == BatchSubmitter {
				return NewDALayerSelector(m.Ctx, m.idx), nil
			}

			return NextUpdateOpinitBotKey(m.Ctx)
		} else {
			analytics.TrackEvent(analytics.RecoverKeySelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, "import"))
			state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{string(state.BotInfos[m.idx].BotName)}, "Import existing key"))
			return NewRecoverFromMnemonic(weavecontext.SetCurrentState(m.Ctx, state), m.idx), nil
		}
	}

	return m, cmd
}

func (m *RecoverKeySelector) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{string(state.BotInfos[m.idx].BotName)}, styles.Question) + m.Selector.View())
}

type RecoverFromMnemonic struct {
	weavecontext.BaseModel
	ui.TextInput
	question string
	idx      int
}

func NewRecoverFromMnemonic(ctx context.Context, idx int) *RecoverFromMnemonic {
	state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
	model := &RecoverFromMnemonic{
		TextInput: ui.NewTextInput(false),
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  fmt.Sprintf("Please add mnemonic for new %s", state.BotInfos[idx].BotName),
		idx:       idx,
	}
	model.WithValidatorFn(common.ValidateMnemonic)
	model.WithPlaceholder("Enter in your mnemonic")
	return model
}

func (m *RecoverFromMnemonic) GetQuestion() string {
	return m.question
}

func (m *RecoverFromMnemonic) Init() tea.Cmd {
	return nil
}

func (m *RecoverFromMnemonic) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}

	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[OPInitBotsState](m)

		// Save the response with hidden mnemonic text
		state.weave.PushPreviousResponse(
			styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{string(state.BotInfos[m.idx].BotName)}, styles.HiddenMnemonicText),
		)

		// Update the state with the input mnemonic
		state.BotInfos[m.idx].Mnemonic = strings.Trim(input.Text, "\n")
		state.BotInfos[m.idx].IsSetup = false

		m.Ctx = weavecontext.SetCurrentState(m.Ctx, state)
		// Check if the bot is of type BatchSubmitter and move to the next step accordingly
		if state.BotInfos[m.idx].BotName == BatchSubmitter {
			return NewDALayerSelector(m.Ctx, m.idx), nil
		}
		return NextUpdateOpinitBotKey(m.Ctx)
	}

	m.TextInput = input
	return m, cmd
}

func (m *RecoverFromMnemonic) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{string(state.BotInfos[m.idx].BotName)}, styles.Question) + m.TextInput.View())
}

// SetupOPInitBots handles the loading and setup of OPInit bots
type SetupOPInitBots struct {
	weavecontext.BaseModel
	ui.Loading
}

// NewSetupOPInitBots initializes a new SetupOPInitBots with context
func NewSetupOPInitBots(ctx context.Context) *SetupOPInitBots {
	return &SetupOPInitBots{
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		Loading:   ui.NewLoading("Downloading binary and adding keys...", WaitSetupOPInitBots(ctx)),
	}
}

func (m *SetupOPInitBots) Init() tea.Cmd {
	return m.Loading.Init()
}

func (m *SetupOPInitBots) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}

	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		state := weavecontext.PushPageAndGetState[OPInitBotsState](m)

		keyFilePath, err := weavecontext.GetOPInitKeyFileJson(m.Ctx)
		if err != nil {
			return m, m.HandlePanic(fmt.Errorf("failed to get key file path for OPinit: %w", err))
		}

		keyFile := io.NewKeyFile()
		err = keyFile.Load(keyFilePath)
		if err != nil {
			return m, m.HandlePanic(fmt.Errorf("failed to load key file for OPinit: %w", err))
		}

		for botName, res := range state.SetupOpinitResponses {
			keyInfo := strings.Split(res, "\n")
			address := strings.Split(keyInfo[0], ": ")
			mnemonic := keyInfo[1]
			keyFile.AddWallet(string(BotNameToKeyName[botName]), io.NewWallet(address[1], mnemonic))
		}

		err = keyFile.Write(keyFilePath)
		if err != nil {
			return m, m.HandlePanic(fmt.Errorf("failed to write key file: %w", err))
		}

		return NewTerminalState(m.Loading.EndContext), tea.Quit
	}
	return m, cmd
}

func (m *SetupOPInitBots) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)

	if m.Loading.Completing {
		// Handle WaitSetupOPInitBots error
		if len(state.SetupOpinitResponses) > 0 {
			keyFilePath, err := weavecontext.GetOPInitKeyFileJson(m.Ctx)
			if err != nil {
				m.HandlePanic(fmt.Errorf("failed to get key file path for OPinit: %w", err))
			}

			addressesText := ""
			for botName, res := range state.SetupOpinitResponses {
				keyInfo := strings.Split(res, "\n")
				address := strings.Split(keyInfo[0], ": ")
				addressesText += renderKey(string(botName), address[1]) + "\n"
			}

			return m.WrapView(state.weave.Render() + "\n" + styles.RenderPrompt("Download binary and add keys successfully.", []string{}, styles.Completed) + "\n\n" +
				styles.BoldUnderlineText("Important", styles.Yellow) + "\n" +
				styles.Text(fmt.Sprintf("Note that the mnemonic phrases will be stored in %s. You can revisit them anytime.", keyFilePath), styles.Yellow) + "\n\n" +
				addressesText)
		} else {
			return m.WrapView(state.weave.Render() + "\n" + styles.RenderPrompt("Download binary and add keys successfully.", []string{}, styles.Completed))
		}
	}

	return m.WrapView(state.weave.Render() + m.Loading.View())
}

func renderKey(keyName, address string) string {
	return styles.BoldText("Key Name: ", styles.Ivory) + keyName + "\n" +
		styles.BoldText("Address: ", styles.Ivory) + address + "\n"
}

// DALayerOption defines options for Data Availability Layers
type DALayerOption string

const (
	InitiaLayerOption   DALayerOption = "Initia"
	CelestiaLayerOption DALayerOption = "Celestia"
)

// DALayerSelector handles the selection of the DA Layer for a specific bot
type DALayerSelector struct {
	weavecontext.BaseModel
	ui.Selector[DALayerOption]
	question string
	idx      int
}

// NewDALayerSelector initializes a new DALayerSelector with context
func NewDALayerSelector(ctx context.Context, idx int) *DALayerSelector {
	return &DALayerSelector{
		Selector: ui.Selector[DALayerOption]{
			Options: []DALayerOption{
				InitiaLayerOption,
				CelestiaLayerOption,
			},
		},
		BaseModel: weavecontext.BaseModel{Ctx: ctx},
		question:  "Which DA Layer would you like to use?",
		idx:       idx,
	}
}

func (m *DALayerSelector) GetQuestion() string {
	return m.question
}

func (m *DALayerSelector) Init() tea.Cmd {
	return nil
}

func (m *DALayerSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		analytics.TrackEvent(analytics.DALayerSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, string(*selected)))
		state := weavecontext.PushPageAndGetState[OPInitBotsState](m)

		// Update the DA Layer for the specific bot
		state.BotInfos[m.idx].DALayer = string(*selected)

		// Save the response for the selected DA Layer
		state.weave.PushPreviousResponse(
			styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{"DA Layer"}, state.BotInfos[m.idx].DALayer),
		)

		// Proceed to the next step
		return NextUpdateOpinitBotKey(weavecontext.SetCurrentState(m.Ctx, state))
	}

	return m, cmd
}

func (m *DALayerSelector) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	return m.WrapView(state.weave.Render() + styles.RenderPrompt(m.GetQuestion(), []string{"DA Layer"}, styles.Question) + m.Selector.View())
}

func getBinaryURL(version, os, arch string) (string, error) {
	switch os {
	case "darwin":
		switch arch {
		case "amd64":
			return fmt.Sprintf("https://github.com/initia-labs/opinit-bots/releases/download/%s/opinitd_%s_Darwin_x86_64.tar.gz", version, version), nil
		case "arm64":
			return fmt.Sprintf("https://github.com/initia-labs/opinit-bots/releases/download/%s/opinitd_%s_Darwin_aarch64.tar.gz", version, version), nil
		}
	case "linux":
		switch arch {
		case "amd64":
			return fmt.Sprintf("https://github.com/initia-labs/opinit-bots/releases/download/%s/opinitd_%s_Linux_x86_64.tar.gz", version, version), nil
		case "arm64":
			return fmt.Sprintf("https://github.com/initia-labs/opinit-bots/releases/download/%s/opinitd_%s_Linux_aarch64.tar.gz", version, version), nil
		}
	}
	return "", fmt.Errorf("unsupported OS or architecture: %v %v", os, arch)
}

func GetBinaryPath(userHome string) string {
	return filepath.Join(userHome, common.WeaveDataDirectory, fmt.Sprintf("opinitd@%s", OpinitBotBinaryVersion), AppName)
}

func EnsureOPInitBotsBinary(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to get user home directory: %v", err)}
		}

		binaryPath := GetBinaryPath(userHome)
		_, err = cosmosutils.GetBinaryVersion(binaryPath)
		if err == nil {
			return ui.EndLoading{
				Ctx: ctx,
			}
		}

		weaveDataPath := filepath.Join(userHome, common.WeaveDataDirectory)
		tarballPath := filepath.Join(weaveDataPath, "opinitd.tar.gz")

		goos := runtime.GOOS
		goarch := runtime.GOARCH
		url, err := getBinaryURL(OpinitBotBinaryVersion, goos, goarch)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to get binary url: %v", err)}
		}

		extractedPath := filepath.Join(weaveDataPath, fmt.Sprintf("opinitd@%s", OpinitBotBinaryVersion))

		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {

			if _, err := os.Stat(extractedPath); os.IsNotExist(err) {
				err := os.MkdirAll(extractedPath, os.ModePerm)
				if err != nil {
					return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to create weave data directory: %v", err)}
				}
			}

			if err = io.DownloadAndExtractTarGz(url, tarballPath, extractedPath); err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to download and extract binary: %v", err)}
			}
			err = os.Chmod(binaryPath, 0755) // 0755 ensuring read, write, execute permissions for the owner, and read-execute for group/others
			if err != nil {
				return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to set permissions for binary: %v", err)}
			}
		}

		err = cosmosutils.SetSymlink(binaryPath)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: err}
		}

		return ui.EndLoading{
			Ctx: ctx,
		}
	}
}

func WaitSetupOPInitBots(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		state := weavecontext.GetCurrentState[OPInitBotsState](ctx)
		userHome, err := os.UserHomeDir()
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to get user home directory: %v", err)}
		}

		binaryPath := GetBinaryPath(userHome)
		opInitHome, err := weavecontext.GetOPInitHome(ctx)
		if err != nil {
			return ui.NonRetryableErrorLoading{Err: fmt.Errorf("failed to get OPinit home: %v", err)}
		}
		for _, info := range state.BotInfos {
			if info.Mnemonic != "" {
				res, err := cosmosutils.OPInitRecoverKeyFromMnemonic(binaryPath, info.KeyName, info.Mnemonic, info.DALayer == string(CelestiaLayerOption), opInitHome)
				if err != nil {
					return ui.ErrorLoading{Err: err}
				}
				state.SetupOpinitResponses[info.BotName] = res
				continue
			}
			if info.IsGenerateKey {
				res, err := cosmosutils.OPInitAddOrReplace(binaryPath, info.KeyName, info.DALayer == string(CelestiaLayerOption), opInitHome)
				if err != nil {
					return ui.ErrorLoading{Err: err}

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

type TerminalState struct {
	weavecontext.BaseModel
}

func NewTerminalState(ctx context.Context) *TerminalState {
	analytics.TrackCompletedEvent(analytics.SetupOPinitKeysFeature)
	return &TerminalState{
		weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
	}
}

func (m *TerminalState) Init() tea.Cmd {
	return nil
}

func (m *TerminalState) Update(_ tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *TerminalState) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	if len(state.SetupOpinitResponses) > 0 {
		addressesText := ""
		for _, botName := range BotNames {
			if res, ok := state.SetupOpinitResponses[botName]; ok {
				keyInfo := strings.Split(res, "\n")
				address := strings.Split(keyInfo[0], ": ")
				addressesText += renderKey(string(botName), address[1]) + "\n"
			}
		}

		keyFile, err := weavecontext.GetOPInitKeyFileJson(m.Ctx)
		if err != nil {
			m.HandlePanic(fmt.Errorf("failed to get OPinit key file: %v", err))
		}

		return m.WrapView(state.weave.Render() + "\n" + styles.RenderPrompt("Setup keys successfully.", []string{}, styles.Completed) + "\n\n" +
			styles.BoldUnderlineText("Important", styles.Yellow) + "\n" +
			styles.Text(fmt.Sprintf("Note that the mnemonic phrases will be stored in %s. You can revisit them anytime.", keyFile), styles.Yellow) + "\n\n" +
			addressesText)
	}
	return m.WrapView(state.weave.Render() + "\n")
}

type EnsureOPInitBotsBinaryLoadingModel struct {
	weavecontext.BaseModel
	ui.Loading
	nextModelFunc func(ctx context.Context) (tea.Model, error)
}

func NewEnsureOPInitBotsBinaryLoadingModel(ctx context.Context, nextModelFunc func(ctx context.Context) (tea.Model, error)) tea.Model {
	return &EnsureOPInitBotsBinaryLoadingModel{
		BaseModel:     weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		Loading:       ui.NewLoading("Downloading OPinit bot ...", EnsureOPInitBotsBinary(ctx)),
		nextModelFunc: nextModelFunc,
	}
}

func (m *EnsureOPInitBotsBinaryLoadingModel) Init() tea.Cmd {
	return m.Loading.Init()
}

func (m *EnsureOPInitBotsBinaryLoadingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[OPInitBotsState](m, msg); handled {
		return model, cmd
	}

	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.NonRetryableErr != nil {
		return m, m.HandlePanic(m.Loading.NonRetryableErr)
	}
	if m.Loading.Completing {
		nextModel, err := m.nextModelFunc(m.Ctx)
		if err != nil {
			return m, m.HandlePanic(err)
		}
		return nextModel, nextModel.Init()
	}
	return m, cmd
}

func (m *EnsureOPInitBotsBinaryLoadingModel) View() string {
	state := weavecontext.GetCurrentState[OPInitBotsState](m.Ctx)
	return m.WrapView(state.weave.Render() + m.Loading.View())
}
