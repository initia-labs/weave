package weaveinit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/initia-labs/weave/styles"
	"github.com/initia-labs/weave/utils"
)

type RunL1NodeNetworkSelect struct {
	utils.Selector[L1NodeNetworkOption]
	state    *RunL1NodeState
	question string
}

type L1NodeNetworkOption string

const (
	Mainnet L1NodeNetworkOption = "Mainnet"
	Testnet L1NodeNetworkOption = "Testnet"
	Local   L1NodeNetworkOption = "Local"
)

func NewRunL1NodeNetworkSelect(state *RunL1NodeState) *RunL1NodeNetworkSelect {
	return &RunL1NodeNetworkSelect{
		Selector: utils.Selector[L1NodeNetworkOption]{
			Options: []L1NodeNetworkOption{
				Mainnet,
				Testnet,
				Local,
			},
		},
		state:    state,
		question: "Which network will your node participate in?",
	}
}

func (m *RunL1NodeNetworkSelect) GetQuestion() string {
	return m.question
}

func (m *RunL1NodeNetworkSelect) Init() tea.Cmd {
	return nil
}

func (m *RunL1NodeNetworkSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	selected, cmd := m.Select(msg)
	if selected != nil {
		selectedString := string(*selected)
		m.state.network = selectedString
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{}, selectedString)
		switch *selected {
		case Mainnet, Testnet:
			return NewRunL1NodeMonikerInput(m.state), cmd
		case Local:
			return NewRunL1NodeVersionInput(m.state), nil
		}
		return m, tea.Quit
	}

	return m, cmd
}

func (m *RunL1NodeNetworkSelect) View() string {
	return styles.RenderPrompt("Which network will your node participate in?", []string{"network"}, styles.Question) + m.Selector.View()
}

type RunL1NodeVersionInput struct {
	utils.TextInput
	state    *RunL1NodeState
	question string
}

func NewRunL1NodeVersionInput(state *RunL1NodeState) *RunL1NodeVersionInput {
	return &RunL1NodeVersionInput{
		TextInput: utils.NewTextInput(),
		state:     state,
		question:  "Please specify the initiad version",
	}
}

func (m *RunL1NodeVersionInput) GetQuestion() string {
	return m.question
}

func (m *RunL1NodeVersionInput) Init() tea.Cmd {
	return nil
}

func (m *RunL1NodeVersionInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	input, cmd, done := m.TextInput.Update(msg)
	if done {
		m.state.initiadVersion = input.Text
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"initiad version"}, input.Text)
		return NewRunL1NodeChainIdInput(m.state), cmd
	}
	m.TextInput = input
	return m, cmd
}

func (m *RunL1NodeVersionInput) View() string {
	return m.state.weave.PreviousResponse + styles.RenderPrompt(m.GetQuestion(), []string{"initiad version"}, styles.Question) + m.TextInput.View()
}

type RunL1NodeChainIdInput struct {
	utils.TextInput
	state    *RunL1NodeState
	question string
}

func NewRunL1NodeChainIdInput(state *RunL1NodeState) *RunL1NodeChainIdInput {
	return &RunL1NodeChainIdInput{
		TextInput: utils.NewTextInput(),
		state:     state,
		question:  "Please specify the chain id",
	}
}

func (m *RunL1NodeChainIdInput) GetQuestion() string {
	return m.question
}

func (m *RunL1NodeChainIdInput) Init() tea.Cmd {
	return nil
}

func (m *RunL1NodeChainIdInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	input, cmd, done := m.TextInput.Update(msg)
	if done {
		m.state.chainId = input.Text
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"chain id"}, input.Text)
		return NewRunL1NodeMonikerInput(m.state), cmd
	}
	m.TextInput = input
	return m, cmd
}

func (m *RunL1NodeChainIdInput) View() string {
	return m.state.weave.PreviousResponse + styles.RenderPrompt(m.GetQuestion(), []string{"chain id"}, styles.Question) + m.TextInput.View()
}

type RunL1NodeMonikerInput struct {
	utils.TextInput
	state    *RunL1NodeState
	question string
}

func NewRunL1NodeMonikerInput(state *RunL1NodeState) *RunL1NodeMonikerInput {
	return &RunL1NodeMonikerInput{
		TextInput: utils.NewTextInput(),
		state:     state,
		question:  "Please specify the moniker",
	}
}

func (m *RunL1NodeMonikerInput) GetQuestion() string {
	return m.question
}

func (m *RunL1NodeMonikerInput) Init() tea.Cmd {
	return nil
}

func (m *RunL1NodeMonikerInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	input, cmd, done := m.TextInput.Update(msg)
	if done {
		m.state.moniker = input.Text
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"moniker"}, input.Text)
		model := NewExistingAppChecker(m.state)
		return model, model.Init()
	}
	m.TextInput = input
	return m, cmd
}

func (m *RunL1NodeMonikerInput) View() string {
	return m.state.weave.PreviousResponse + styles.RenderPrompt(m.GetQuestion(), []string{"moniker"}, styles.Question) + m.TextInput.View()
}

type ExistingAppChecker struct {
	state   *RunL1NodeState
	loading utils.Loading
}

func NewExistingAppChecker(state *RunL1NodeState) *ExistingAppChecker {
	return &ExistingAppChecker{
		state:   state,
		loading: utils.NewLoading("Checking for an existing Initia app...", WaitExistingAppChecker(state)),
	}
}

func (m *ExistingAppChecker) Init() tea.Cmd {
	return m.loading.Init()
}

func WaitExistingAppChecker(state *RunL1NodeState) tea.Cmd {
	return func() tea.Msg {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return utils.ErrorLoading{Err: err}
		}

		initiaConfigPath := filepath.Join(homeDir, utils.InitiaConfigDirectory)
		appTomlPath := filepath.Join(initiaConfigPath, "app.toml")
		configTomlPath := filepath.Join(initiaConfigPath, "config.toml")
		time.Sleep(1500 * time.Millisecond)
		if !utils.FileOrFolderExists(configTomlPath) || !utils.FileOrFolderExists(appTomlPath) {
			state.existingApp = false
			return utils.EndLoading{}
		} else {
			state.existingApp = true
			return utils.EndLoading{}
		}
	}
}

func (m *ExistingAppChecker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	loader, cmd := m.loading.Update(msg)
	m.loading = loader
	if m.loading.Completing {

		if !m.state.existingApp {
			return NewMinGasPriceInput(m.state), nil
		} else {
			return NewExistingAppReplaceSelect(m.state), nil
		}
	}
	return m, cmd
}

func (m *ExistingAppChecker) View() string {
	return m.state.weave.PreviousResponse + "\n" + m.loading.View()
}

type ExistingAppReplaceSelect struct {
	utils.Selector[ExistingAppReplaceOption]
	state    *RunL1NodeState
	question string
}

type ExistingAppReplaceOption string

const (
	UseCurrentApp ExistingAppReplaceOption = "Use current files"
	ReplaceApp    ExistingAppReplaceOption = "Replace"
)

func NewExistingAppReplaceSelect(state *RunL1NodeState) *ExistingAppReplaceSelect {
	return &ExistingAppReplaceSelect{
		Selector: utils.Selector[ExistingAppReplaceOption]{
			Options: []ExistingAppReplaceOption{
				UseCurrentApp,
				ReplaceApp,
			},
		},
		state:    state,
		question: "Existing config/app.toml and config/config.toml detected. Would you like to use the current files or replace them",
	}
}

func (m *ExistingAppReplaceSelect) GetQuestion() string {
	return m.question
}

func (m *ExistingAppReplaceSelect) Init() tea.Cmd {
	return nil
}

func (m *ExistingAppReplaceSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	selected, cmd := m.Select(msg)
	if selected != nil {
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{"config/app.toml", "config/config.toml"}, string(*selected))
		switch *selected {
		case UseCurrentApp:
			m.state.replaceExistingApp = false
			return NewExistingGenesisChecker(m.state), utils.DoTick()
		case ReplaceApp:
			m.state.replaceExistingApp = true
			return NewMinGasPriceInput(m.state), nil
		}
		return m, tea.Quit
	}

	return m, cmd
}

func (m *ExistingAppReplaceSelect) View() string {
	return m.state.weave.PreviousResponse + styles.RenderPrompt("Existing config/app.toml and config/config.toml detected. Would you like to use the current files or replace them", []string{"config/app.toml", "config/config.toml"}, styles.Question) + m.Selector.View()
}

type MinGasPriceInput struct {
	utils.TextInput
	state    *RunL1NodeState
	question string
}

func NewMinGasPriceInput(state *RunL1NodeState) *MinGasPriceInput {
	model := &MinGasPriceInput{
		TextInput: utils.NewTextInput(),
		state:     state,
		question:  "Please specify min-gas-price (uinit)",
	}
	model.WithValidatorFn(utils.ValidateDecCoin)
	return model
}

func (m *MinGasPriceInput) GetQuestion() string {
	return m.question
}

func (m *MinGasPriceInput) Init() tea.Cmd {
	return nil
}

func (m *MinGasPriceInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	input, cmd, done := m.TextInput.Update(msg)
	if done {
		m.state.minGasPrice = input.Text
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"min-gas-price"}, input.Text)
		return NewEnableFeaturesCheckbox(m.state), cmd
	}
	m.TextInput = input
	return m, cmd
}

func (m *MinGasPriceInput) View() string {
	preText := ""
	if !m.state.existingApp {
		preText += styles.RenderPrompt("There is no config/app.toml or config/config.toml available. You will need to enter the required information to proceed.\n\n", []string{"config/app.toml", "config/config.toml"}, styles.Information)
	}
	return m.state.weave.PreviousResponse + preText + styles.RenderPrompt(m.GetQuestion(), []string{"min-gas-price"}, styles.Question) + m.TextInput.View()
}

type EnableFeaturesCheckbox struct {
	utils.CheckBox[EnableFeaturesOption]
	state    *RunL1NodeState
	question string
}

type EnableFeaturesOption string

const (
	LCD  EnableFeaturesOption = "LCD API"
	gRPC EnableFeaturesOption = "gRPC"
)

func NewEnableFeaturesCheckbox(state *RunL1NodeState) *EnableFeaturesCheckbox {
	return &EnableFeaturesCheckbox{
		CheckBox: *utils.NewCheckBox([]EnableFeaturesOption{LCD, gRPC}),
		state:    state,
		question: "Would you like to enable the following options?",
	}
}

func (m *EnableFeaturesCheckbox) GetQuestion() string {
	return m.question
}

func (m *EnableFeaturesCheckbox) Init() tea.Cmd {
	return nil
}

func (m *EnableFeaturesCheckbox) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cb, cmd, done := m.Select(msg)
	if done {
		empty := true
		for idx, isSelected := range cb.Selected {
			if isSelected {
				empty = false
				switch cb.Options[idx] {
				case LCD:
					m.state.enableLCD = true
				case gRPC:
					m.state.enableGRPC = true
				}
			}
		}
		if empty {
			m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{}, "None")
		} else {
			m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{}, cb.GetSelectedString())
		}
		return NewSeedsInput(m.state), nil
	}

	return m, cmd
}

func (m *EnableFeaturesCheckbox) View() string {
	return m.state.weave.PreviousResponse + "\n" + styles.RenderPrompt(m.GetQuestion(), []string{}, styles.Question) + "\n" + m.CheckBox.View()
}

type SeedsInput struct {
	utils.TextInput
	state    *RunL1NodeState
	question string
}

func NewSeedsInput(state *RunL1NodeState) *SeedsInput {
	model := &SeedsInput{
		TextInput: utils.NewTextInput(),
		state:     state,
		question:  "Please specify the seeds",
	}
	model.WithValidatorFn(utils.IsValidPeerOrSeed)
	return model
}

func (m *SeedsInput) GetQuestion() string {
	return m.question
}

func (m *SeedsInput) Init() tea.Cmd {
	return nil
}

func (m *SeedsInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	input, cmd, done := m.TextInput.Update(msg)
	if done {
		m.state.seeds = input.Text
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"seeds"}, input.Text)
		return NewPersistentPeersInput(m.state), cmd
	}
	m.TextInput = input
	return m, cmd
}

func (m *SeedsInput) View() string {
	return m.state.weave.PreviousResponse + styles.RenderPrompt(m.GetQuestion(), []string{"seeds"}, styles.Question) + m.TextInput.View()
}

type PersistentPeersInput struct {
	utils.TextInput
	state    *RunL1NodeState
	question string
}

func NewPersistentPeersInput(state *RunL1NodeState) *PersistentPeersInput {
	model := &PersistentPeersInput{
		TextInput: utils.NewTextInput(),
		state:     state,
		question:  "Please specify the persistent_peers",
	}
	model.WithValidatorFn(utils.IsValidPeerOrSeed)
	return model
}

func (m *PersistentPeersInput) GetQuestion() string {
	return m.question
}

func (m *PersistentPeersInput) Init() tea.Cmd {
	return nil
}

func (m *PersistentPeersInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	input, cmd, done := m.TextInput.Update(msg)
	if done {
		m.state.persistentPeers = input.Text
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"persistent_peers"}, input.Text)
		model := NewExistingGenesisChecker(m.state)
		return model, model.Init()
	}
	m.TextInput = input
	return m, cmd
}

func (m *PersistentPeersInput) View() string {
	return m.state.weave.PreviousResponse + styles.RenderPrompt(m.GetQuestion(), []string{"persistent_peers"}, styles.Question) + m.TextInput.View()
}

type ExistingGenesisChecker struct {
	state   *RunL1NodeState
	loading utils.Loading
}

func NewExistingGenesisChecker(state *RunL1NodeState) *ExistingGenesisChecker {
	return &ExistingGenesisChecker{
		state:   state,
		loading: utils.NewLoading("Checking for an existing Initia genesis file...", WaitExistingGenesisChecker(state)),
	}
}

func (m *ExistingGenesisChecker) Init() tea.Cmd {
	return m.loading.Init()
}

func WaitExistingGenesisChecker(state *RunL1NodeState) tea.Cmd {
	return func() tea.Msg {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("[error] Failed to get user home directory: %v\n", err)
			return utils.ErrorLoading{Err: err}
		}
		initiaConfigPath := filepath.Join(homeDir, utils.InitiaConfigDirectory)
		genesisFilePath := filepath.Join(initiaConfigPath, "genesis.json")

		time.Sleep(1500 * time.Millisecond)

		if !utils.FileOrFolderExists(genesisFilePath) {
			state.existingGenesis = false
			return utils.EndLoading{}
		} else {
			state.existingGenesis = true
			return utils.EndLoading{}
		}
	}
}

func (m *ExistingGenesisChecker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	loader, cmd := m.loading.Update(msg)
	m.loading = loader
	if m.loading.Completing {
		if !m.state.existingGenesis {
			if m.state.network == string(Local) {
				newLoader := NewInitializingAppLoading(m.state)
				return newLoader, newLoader.Init()
			}
			return NewGenesisEndpointInput(m.state), nil
		} else {
			return NewExistingGenesisReplaceSelect(m.state), nil
		}
	}
	return m, cmd
}

func (m *ExistingGenesisChecker) View() string {
	return m.state.weave.PreviousResponse + "\n" + m.loading.View()
}

type ExistingGenesisReplaceSelect struct {
	utils.Selector[ExistingGenesisReplaceOption]
	state    *RunL1NodeState
	question string
}

type ExistingGenesisReplaceOption string

const (
	UseCurrentGenesis ExistingGenesisReplaceOption = "Use current one"
	ReplaceGenesis    ExistingGenesisReplaceOption = "Replace"
)

func NewExistingGenesisReplaceSelect(state *RunL1NodeState) *ExistingGenesisReplaceSelect {
	return &ExistingGenesisReplaceSelect{
		Selector: utils.Selector[ExistingGenesisReplaceOption]{
			Options: []ExistingGenesisReplaceOption{
				UseCurrentGenesis,
				ReplaceGenesis,
			},
		},
		state:    state,
		question: "Existing config/genesis.json detected. Would you like to use the current one or replace it?",
	}
}

func (m *ExistingGenesisReplaceSelect) GetQuestion() string {
	return m.question
}

func (m *ExistingGenesisReplaceSelect) Init() tea.Cmd {
	return nil
}

func (m *ExistingGenesisReplaceSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	selected, cmd := m.Select(msg)
	if selected != nil {
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{"config/genesis.json"}, string(*selected))
		switch *selected {
		case UseCurrentGenesis:
			newLoader := NewInitializingAppLoading(m.state)
			return newLoader, newLoader.Init()
		case ReplaceGenesis:
			return NewGenesisEndpointInput(m.state), nil
		}
		return m, tea.Quit
	}

	return m, cmd
}

func (m *ExistingGenesisReplaceSelect) View() string {
	return m.state.weave.PreviousResponse + styles.RenderPrompt(
		m.GetQuestion(),
		[]string{"config/genesis.json"},
		styles.Question,
	) + m.Selector.View()
}

type GenesisEndpointInput struct {
	utils.TextInput
	state    *RunL1NodeState
	question string
}

func NewGenesisEndpointInput(state *RunL1NodeState) *GenesisEndpointInput {
	return &GenesisEndpointInput{
		TextInput: utils.NewTextInput(),
		state:     state,
		question:  "Please specify the endpoint to fetch genesis.json",
	}
}

func (m *GenesisEndpointInput) GetQuestion() string {
	return m.question
}

func (m *GenesisEndpointInput) Init() tea.Cmd {
	return nil
}

func (m *GenesisEndpointInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	input, cmd, done := m.TextInput.Update(msg)
	if done {
		m.state.genesisEndpoint = input.Text
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"endpoint"}, input.Text)
		newLoader := NewInitializingAppLoading(m.state)
		return newLoader, newLoader.Init()
	}
	m.TextInput = input
	return m, cmd
}

func (m *GenesisEndpointInput) View() string {
	preText := ""
	if !m.state.existingApp {
		preText += styles.RenderPrompt("There is no config/genesis.json available. You will need to enter the required information to proceed.\n\n", []string{"config/genesis.json"}, styles.Information)
	}
	return m.state.weave.PreviousResponse + preText + styles.RenderPrompt(m.GetQuestion(), []string{"endpoint"}, styles.Question) + m.TextInput.View()
}

type InitializingAppLoading struct {
	utils.Loading
	state *RunL1NodeState
}

func NewInitializingAppLoading(state *RunL1NodeState) *InitializingAppLoading {
	return &InitializingAppLoading{
		Loading: utils.NewLoading("Initializing Initia App...", utils.DefaultWait()),
		state:   state,
	}
}

func (m *InitializingAppLoading) Init() tea.Cmd {
	return m.Loading.Init()
}

func (m *InitializingAppLoading) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.Completing {
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.NoSeparator, "Initialization successful.", []string{}, "")
		switch m.state.network {
		case string(Local):
			return m, tea.Quit
		case string(Mainnet), string(Testnet):
			return NewSyncMethodSelect(m.state), nil
		}
	}
	return m, cmd
}

func (m *InitializingAppLoading) View() string {
	if m.Completing {
		return m.state.weave.PreviousResponse
	}
	return m.state.weave.PreviousResponse + m.Loading.View()
}

type SyncMethodSelect struct {
	utils.Selector[SyncMethodOption]
	state    *RunL1NodeState
	question string
}

type SyncMethodOption string

const (
	Snapshot  SyncMethodOption = "Snapshot"
	StateSync SyncMethodOption = "State Sync"
)

func NewSyncMethodSelect(state *RunL1NodeState) *SyncMethodSelect {
	return &SyncMethodSelect{
		Selector: utils.Selector[SyncMethodOption]{
			Options: []SyncMethodOption{
				Snapshot,
				StateSync,
			},
		},
		state:    state,
		question: "Please select a sync option",
	}
}

func (m *SyncMethodSelect) GetQuestion() string {
	return m.question
}

func (m *SyncMethodSelect) Init() tea.Cmd {
	return nil
}

func (m *SyncMethodSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	selected, cmd := m.Select(msg)
	if selected != nil {
		m.state.syncMethod = string(*selected)
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{""}, string(*selected))
		model := NewExistingDataChecker(m.state)
		return model, model.Init()
	}

	return m, cmd
}

func (m *SyncMethodSelect) View() string {
	return m.state.weave.PreviousResponse + styles.RenderPrompt(
		m.GetQuestion(),
		[]string{""},
		styles.Question,
	) + m.Selector.View()
}

type ExistingDataChecker struct {
	state   *RunL1NodeState
	loading utils.Loading
}

func NewExistingDataChecker(state *RunL1NodeState) *ExistingDataChecker {
	return &ExistingDataChecker{
		state:   state,
		loading: utils.NewLoading("Checking for an existing Initia data...", WaitExistingDataChecker(state)),
	}
}

func (m *ExistingDataChecker) Init() tea.Cmd {
	return m.loading.Init()
}

func WaitExistingDataChecker(state *RunL1NodeState) tea.Cmd {
	return func() tea.Msg {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("[error] Failed to get user home directory: %v\n", err)
			return utils.ErrorLoading{Err: err}
		}

		initiaDataPath := filepath.Join(homeDir, utils.InitiaDataDirectory)
		time.Sleep(1500 * time.Millisecond)

		if !utils.FileOrFolderExists(initiaDataPath) {
			state.existingData = false
			return utils.EndLoading{}
		} else {
			state.existingData = true
			return utils.EndLoading{}
		}
	}
}

func (m *ExistingDataChecker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	loader, cmd := m.loading.Update(msg)
	m.loading = loader
	if m.loading.Completing {
		if !m.state.existingData {
			switch m.state.syncMethod {
			case string(Snapshot):
				return NewSnapshotEndpointInput(m.state), nil
			case string(StateSync):
				return NewStateSyncEndpointInput(m.state), nil
			}
			return m, tea.Quit
		} else {
			m.state.existingData = true
			return NewExistingDataReplaceSelect(m.state), nil
		}
	}
	return m, cmd
}

func (m *ExistingDataChecker) View() string {
	return m.state.weave.PreviousResponse + "\n" + m.loading.View()
}

type ExistingDataReplaceSelect struct {
	utils.Selector[ExistingDataReplaceOption]
	state    *RunL1NodeState
	question string
}

type ExistingDataReplaceOption string

const (
	UseCurrentData ExistingDataReplaceOption = "Use current one"
	ReplaceData    ExistingDataReplaceOption = "Replace"
)

func NewExistingDataReplaceSelect(state *RunL1NodeState) *ExistingDataReplaceSelect {
	return &ExistingDataReplaceSelect{
		Selector: utils.Selector[ExistingDataReplaceOption]{
			Options: []ExistingDataReplaceOption{
				UseCurrentData,
				ReplaceData,
			},
		},
		state:    state,
		question: fmt.Sprintf("Existing %s detected. Would you like to use the current one or replace it", utils.InitiaDataDirectory),
	}
}

func (m *ExistingDataReplaceSelect) GetQuestion() string {
	return m.question
}

func (m *ExistingDataReplaceSelect) Init() tea.Cmd {
	return nil
}

func (m *ExistingDataReplaceSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	selected, cmd := m.Select(msg)
	if selected != nil {
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.ArrowSeparator, m.GetQuestion(), []string{utils.InitiaDataDirectory}, string(*selected))
		switch *selected {
		case UseCurrentData:
			m.state.replaceExistingData = false
			return m, tea.Quit
		case ReplaceData:
			m.state.replaceExistingData = true
			switch m.state.syncMethod {
			case string(Snapshot):
				return NewSnapshotEndpointInput(m.state), nil
			case string(StateSync):
				return NewStateSyncEndpointInput(m.state), nil
			}
		}
		return m, tea.Quit
	}

	return m, cmd
}

func (m *ExistingDataReplaceSelect) View() string {
	return m.state.weave.PreviousResponse + styles.RenderPrompt(m.GetQuestion(), []string{utils.InitiaDataDirectory}, styles.Question) + m.Selector.View()
}

type SnapshotEndpointInput struct {
	utils.TextInput
	state    *RunL1NodeState
	question string
}

func NewSnapshotEndpointInput(state *RunL1NodeState) *SnapshotEndpointInput {
	return &SnapshotEndpointInput{
		TextInput: utils.NewTextInput(),
		state:     state,
		question:  "Please specify the snapshot url to download",
	}
}

func (m *SnapshotEndpointInput) GetQuestion() string {
	return m.question
}

func (m *SnapshotEndpointInput) Init() tea.Cmd {
	return nil
}

func (m *SnapshotEndpointInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	input, cmd, done := m.TextInput.Update(msg)
	if done {
		m.state.snapshotEndpoint = input.Text
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"snapshot url"}, input.Text)
		snapshotDownload := NewSnapshotDownloadLoading(m.state)
		return snapshotDownload, snapshotDownload.Init()
	}
	m.TextInput = input
	return m, cmd
}

func (m *SnapshotEndpointInput) View() string {
	// TODO: Correctly render terminal output
	return m.state.weave.PreviousResponse + styles.RenderPrompt(m.GetQuestion(), []string{"snapshot url"}, styles.Question) + m.TextInput.View()
}

type StateSyncEndpointInput struct {
	utils.TextInput
	state    *RunL1NodeState
	question string
}

func NewStateSyncEndpointInput(state *RunL1NodeState) *StateSyncEndpointInput {
	return &StateSyncEndpointInput{
		TextInput: utils.NewTextInput(),
		state:     state,
		question:  "Please specify the state sync RPC server url",
	}
}

func (m *StateSyncEndpointInput) GetQuestion() string {
	return m.question
}

func (m *StateSyncEndpointInput) Init() tea.Cmd {
	return nil
}

func (m *StateSyncEndpointInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	input, cmd, done := m.TextInput.Update(msg)
	if done {
		m.state.stateSyncEndpoint = input.Text
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.DotsSeparator, m.GetQuestion(), []string{"state sync RPC"}, input.Text)
		// TODO: Continue
		return m, tea.Quit
	}
	m.TextInput = input
	return m, cmd
}

func (m *StateSyncEndpointInput) View() string {
	// TODO: Correctly render terminal output
	return m.state.weave.PreviousResponse + styles.RenderPrompt(m.GetQuestion(), []string{"state sync RPC"}, styles.Question) + m.TextInput.View()
}

type SnapshotDownloadLoading struct {
	utils.Downloader
	state *RunL1NodeState
}

func NewSnapshotDownloadLoading(state *RunL1NodeState) *SnapshotDownloadLoading {
	userHome, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("[error] Failed to get user home: %v\n", err)
		// TODO: Return error
	}

	return &SnapshotDownloadLoading{
		Downloader: *utils.NewDownloader(
			"Downloading snapshot from the provided URL",
			state.snapshotEndpoint,
			fmt.Sprintf("%s/%s/%s", userHome, utils.WeaveDataDirectory, utils.SnapshotFilename),
		),
		state: state,
	}
}

func (m *SnapshotDownloadLoading) Init() tea.Cmd {
	return m.Downloader.Init()
}

func (m *SnapshotDownloadLoading) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.GetCompletion() {
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.NoSeparator, "Snapshot download completed.", []string{}, "")
		newLoader := NewSnapshotExtractLoading(m.state)
		return newLoader, newLoader.Init()
	}
	downloader, cmd := m.Downloader.Update(msg)
	m.Downloader = *downloader
	return m, cmd
}

func (m *SnapshotDownloadLoading) View() string {
	return m.state.weave.PreviousResponse + m.Downloader.View()
}

type SnapshotExtractLoading struct {
	utils.Loading
	state *RunL1NodeState
}

func NewSnapshotExtractLoading(state *RunL1NodeState) *SnapshotExtractLoading {
	return &SnapshotExtractLoading{
		Loading: utils.NewLoading("Extracting downloaded snapshot...", snapshotExtractor()),
		state:   state,
	}
}

func (m *SnapshotExtractLoading) Init() tea.Cmd {
	return m.Loading.Init()
}

func (m *SnapshotExtractLoading) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	loader, cmd := m.Loading.Update(msg)
	m.Loading = loader
	if m.Loading.Completing {
		m.state.weave.PreviousResponse += styles.RenderPreviousResponse(styles.NoSeparator, fmt.Sprintf("Snapshot extracted to %s successfully.", utils.InitiaDataDirectory), []string{}, "")
		return m, tea.Quit
	}
	return m, cmd
}

func (m *SnapshotExtractLoading) View() string {
	if m.Completing {
		return m.state.weave.PreviousResponse
	}
	return m.state.weave.PreviousResponse + m.Loading.View()
}

func snapshotExtractor() tea.Cmd {
	return func() tea.Msg {
		userHome, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("[error] Failed to get user home: %v\n", err)
			// TODO: Return error
		}

		targetDir := filepath.Join(userHome, utils.InitiaDirectory)
		cmd := exec.Command("bash", "-c", fmt.Sprintf("lz4 -c -d %s | tar -x -C %s", filepath.Join(userHome, utils.WeaveDataDirectory, utils.SnapshotFilename), targetDir))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err = cmd.Run()
		if err != nil {
			fmt.Printf("[error] Failed to extract snapshot: %v\n", err)
			// TODO: Return error
		}
		return utils.EndLoading{}
	}
}
