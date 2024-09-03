package states

import (
	tea "github.com/charmbracelet/bubbletea"
)

var _ State = &HomePage{}
var _ tea.Model = &HomePage{}

type HomePage struct {
	BaseState
}

func NewHomePage(transitions []State) *HomePage {
	return &HomePage{
		BaseState: BaseState{
			Transitions: transitions,
			Name:        "home page",
		},
	}
}

func (hp *HomePage) Init() tea.Cmd {
	return nil
}

func (hp *HomePage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return hp.CommonUpdate(msg, hp)
}

func (hp *HomePage) View() string {
	view := "Which action would you like to do?\n"
	for i, transition := range hp.Transitions {
		if i == hp.Cursor {
			view += "(•) " + transition.GetName() + "\n"
		} else {
			view += "( ) " + transition.GetName() + "\n"
		}
	}
	return view + "\nPress Enter to go to the selected page, or Q to quit."
}
