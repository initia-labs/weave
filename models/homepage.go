package models

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/initia-labs/weave/models/weaveinit"
	"github.com/initia-labs/weave/styles"
	"github.com/initia-labs/weave/utils"
)

type Homepage struct {
	utils.Selector[HomepageOption]
	TextInput        utils.TextInput
	isFirstTimeSetup bool
}

type HomepageOption string

const (
	InitOption HomepageOption = "Weave Init"
)

func NewHomepage() tea.Model {
	return &Homepage{
		Selector: utils.Selector[HomepageOption]{
			Options: []HomepageOption{
				InitOption,
			},
			Cursor: 0,
		},
		isFirstTimeSetup: utils.IsFirstTimeSetup(),
		TextInput:        utils.NewTextInput(),
	}
}

func (m *Homepage) Init() tea.Cmd {
	return nil
}

func (m *Homepage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.isFirstTimeSetup {
		input, cmd, done := m.TextInput.Update(msg)
		if done {
			err := utils.SetConfig("common.gas_station_mnemonic", input.Text)
			if err != nil {
				return nil, nil
			}
			view := styles.RenderPrompt("Please set up a Gas Station account", []string{"Gas Station account"}, styles.Question) +
				" " + styles.Text("(The account that will hold the funds required by the OPinit-bots or relayer to send transactions)", styles.Gray) +
				"\n" + styles.Text(input.Text, styles.White)
			model := NewHomepageInitialization(view)
			return model, model.Init()
		}
		m.TextInput = input
		return m, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		switch *selected {
		case InitOption:
			return weaveinit.NewWeaveInit(), nil
		}
	}

	return m, cmd
}

func (m *Homepage) View() string {
	view := "Hi 👋🏻 Weave is a CLI for managing Initia deployments.\n"

	if m.isFirstTimeSetup {
		view += styles.RenderPrompt("Please set up a Gas Station account", []string{"Gas Station account"}, styles.Question) +
			" " + styles.Text("(The account that will hold the funds required by the OPinit-bots or relayer to send transactions)", styles.Gray)
		// view += styles.FadeText("Initia is a network for interwoven rollups ") + "🪢\n"
		view += m.TextInput.View()

	} else {
		view += styles.RenderPrompt("What would you like to do today?", []string{}, styles.Question) + m.Selector.View()
	}

	return view
}

type HomepageInitialization struct {
	previousResponse string
	loading          utils.Loading
}

func NewHomepageInitialization(prevRes string) tea.Model {
	return &HomepageInitialization{
		previousResponse: prevRes,
		loading:          utils.NewLoading("Downloading ...", utils.Wait()),
	}
}

func (hi *HomepageInitialization) Init() tea.Cmd {
	return hi.loading.Init()
}

func (hi *HomepageInitialization) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	loader, cmd := hi.loading.Update(msg)
	hi.loading = loader
	if hi.loading.Completing {
		return hi, tea.Quit
	}
	return hi, cmd
}

func (hi *HomepageInitialization) View() string {
	return hi.previousResponse + "\n" + hi.loading.View()
}
