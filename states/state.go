package states

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Example Reset function for all states:
func ResetStates() {
	HomePageInstance = nil
	InitiaInitInstance = nil
	RunL1NodeInstance = nil
	LaunchNewMinitiaInstance = nil
}

type State interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (tea.Model, tea.Cmd)
	View() string
	GetName() string
}

type BaseState struct {
	Transitions []State
	Cursor      int
	Global      *GlobalStorage
}

func (bs *BaseState) CommonUpdate(msg tea.Msg, currentState State) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "down", "j":
			bs.Cursor = (bs.Cursor + 1) % len(bs.Transitions)
			return currentState, nil
		case "up", "k":
			bs.Cursor = (bs.Cursor - 1 + len(bs.Transitions)) % len(bs.Transitions)
			return currentState, nil
		case "q", "ctrl+c":
			return currentState, tea.Quit
		case "enter":
			// Transition to the selected state
			return bs.Transitions[bs.Cursor], nil
		}
	}
	return currentState, nil
}
