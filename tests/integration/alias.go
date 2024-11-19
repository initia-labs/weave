package integration

import tea "github.com/charmbracelet/bubbletea"

var (
	pressEnter = InputStep{Msg: tea.KeyMsg{Type: tea.KeyEnter}}
	pressSpace = InputStep{Msg: tea.KeyMsg{Type: tea.KeySpace}}
	pressTab   = InputStep{Msg: tea.KeyMsg{Type: tea.KeyTab}}
	pressUp    = InputStep{Msg: tea.KeyMsg{Type: tea.KeyUp}}
	pressDown  = InputStep{Msg: tea.KeyMsg{Type: tea.KeyDown}}
)

func typeText(text string) InputStep {
	return InputStep{Msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)}}
}

// waitFor receives waitCondition as a parameter, which should return true if the wait should be over.
func waitFor(waitCondition func() bool) WaitStep {
	return WaitStep{Check: waitCondition}
}
