package models

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/initia-labs/weave/models/weaveinit"
	"github.com/initia-labs/weave/utils"
)

type Homepage struct {
	utils.Selector[HomepageOption]
}

type HomepageOption string

const (
	Init HomepageOption = "Weave Init"
)

func NewHomepage() tea.Model {
	return &Homepage{
		Selector: utils.Selector[HomepageOption]{
			Options: []HomepageOption{
				Init,
			},
			Cursor: 0,
		},
	}
}

func (m *Homepage) Init() tea.Cmd {
	return nil
}

func (m *Homepage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	selected, cmd := m.Select(msg)
	if selected != nil {
		switch *selected {
		case Init:
			return weaveinit.NewInitSelect(&weaveinit.State{}), nil
		}
	}

	return m, cmd
}

func (m *Homepage) View() string {
	view := "Hi 👋🏻 Weave is a CLI for managing Initia deployments.\n"
	for i, option := range m.Options {
		if i == m.Cursor {
			view += "(•) " + string(option) + "\n"
		} else {
			view += "( ) " + string(option) + "\n"
		}
	}
	return view + "\nPress Enter to select, or q to quit."
}
