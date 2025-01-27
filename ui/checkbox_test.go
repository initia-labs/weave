package ui

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/initia-labs/weave/analytics"
)

func TestMain(m *testing.M) {
	analytics.Client = &analytics.NoOpClient{}
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestCheckBoxNavigationAndSelection(t *testing.T) {
	options := []string{"Option 1", "Option 2", "Option 3"}
	cb := NewCheckBox(options)

	cb, _, _ = cb.Select(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	assert.Equal(t, cb.Cursor, 1)

	cb, _, _ = cb.Select(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	assert.Equal(t, cb.Cursor, 0)

	cb, _, _ = cb.Select(tea.KeyMsg{Type: tea.KeySpace})
	assert.Equal(t, cb.GetSelected(), []string{"Option 1"})

	cb, _, _ = cb.Select(tea.KeyMsg{Type: tea.KeySpace})
	assert.Equal(t, cb.GetSelected(), []string{})

	cb, _, _ = cb.Select(tea.KeyMsg{Type: tea.KeySpace})

	cb, _, _ = cb.Select(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	cb, _, _ = cb.Select(tea.KeyMsg{Type: tea.KeySpace})
	assert.Equal(t, cb.GetSelected(), []string{"Option 1", "Option 2"})

	_, _, entered := cb.Select(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, entered)
}

func TestCheckBoxQuit(t *testing.T) {
	options := []string{"Option 1", "Option 2", "Option 3"}
	cb := NewCheckBox(options)

	_, cmd, _ := cb.Select(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	assert.NotNil(t, cmd)
}

func TestCheckBoxNavigationWrapping(t *testing.T) {
	options := []string{"Option 1", "Option 2", "Option 3"}
	cb := NewCheckBox(options)

	for i := 0; i < len(options)+1; i++ {
		cb, _, _ = cb.Select(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	}
	assert.Equal(t, 1, cb.Cursor)

	cb.Cursor = 0
	for i := 0; i < len(options)+1; i++ {
		cb, _, _ = cb.Select(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	}
	assert.Equal(t, len(options)-1, cb.Cursor)
}

func TestCheckBoxSimultaneousSelectionsAndDeselections(t *testing.T) {
	options := []string{"Option 1", "Option 2", "Option 3"}
	cb := NewCheckBox(options)

	for i := 0; i < len(options); i++ {
		cb, _, _ = cb.Select(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		cb, _, _ = cb.Select(tea.KeyMsg{Type: tea.KeySpace})
	}

	expectedAllSelected := []string{"Option 1", "Option 2", "Option 3"}
	assert.ElementsMatch(t, expectedAllSelected, cb.GetSelected())

	for i := 0; i < len(options); i++ {
		cb, _, _ = cb.Select(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		cb, _, _ = cb.Select(tea.KeyMsg{Type: tea.KeySpace})
	}

	var expectedNoneSelected []string
	assert.ElementsMatch(t, expectedNoneSelected, cb.GetSelected())
}
