package integration

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	PressEnter = InputStep{Msg: tea.KeyMsg{Type: tea.KeyEnter}}
	PressSpace = InputStep{Msg: tea.KeyMsg{Type: tea.KeySpace}}
	PressTab   = InputStep{Msg: tea.KeyMsg{Type: tea.KeyTab}}
	PressUp    = InputStep{Msg: tea.KeyMsg{Type: tea.KeyUp}}
	PressDown  = InputStep{Msg: tea.KeyMsg{Type: tea.KeyDown}}

	WaitFetching = WaitStep{Check: func() bool {
		time.Sleep(5 * time.Second)
		return true
	}}
)

func TypeText(text string) InputStep {
	return InputStep{Msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)}}
}

// WaitFor receives waitCondition as a parameter, which should return true if the wait should be over.
func WaitFor(waitCondition func() bool) WaitStep {
	return WaitStep{Check: waitCondition}
}
