package utils

import (
	"fmt"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/initia-labs/weave/styles"
)

type TextInput struct {
	Text         string
	Cursor       int // Cursor position within the text
	Placeholder  string
	DefaultValue string
	ValidationFn func(string) error
	IsEntered    bool
	CannotBack   bool
}

func NewTextInput(cannotBack bool) TextInput {
	return TextInput{
		Text:         "",
		Cursor:       0,
		Placeholder:  "",
		DefaultValue: "",
		ValidationFn: NoOps,
		IsEntered:    false,
		CannotBack:   cannotBack,
	}
}

func (ti *TextInput) WithValidatorFn(fn func(string) error) {
	ti.ValidationFn = fn
}

func (ti *TextInput) WithPlaceholder(placeholder string) {
	ti.Placeholder = placeholder
}

func (ti *TextInput) WithDefaultValue(value string) {
	ti.DefaultValue = value
}

func (ti *TextInput) WithPrefillValue(value string) {
	ti.Text = value
	ti.Cursor = len(ti.Text)
}

func (ti TextInput) Update(msg tea.Msg) (TextInput, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		// Handle Enter key
		case msg.Type == tea.KeyEnter:
			ti.IsEntered = true
			return ti, nil, ti.ValidationFn(ti.Text) == nil

		// Handle Backspace key
		case msg.Type == tea.KeyBackspace || msg.Type == tea.KeyCtrlH:
			ti.IsEntered = false
			if ti.Cursor > 0 && len(ti.Text) > 0 {
				ti.Text = ti.Text[:ti.Cursor-1] + ti.Text[ti.Cursor:]
				ti.Cursor--
			}

		case msg.Type == tea.KeyTab:
			ti.IsEntered = false
			if ti.Text == "" {
				ti.Text = ti.DefaultValue
				ti.Cursor = len(ti.Text)
			}

		// Handle Option + Left (move one word left) - Detected as "alt+b"
		case msg.String() == "alt+b":
			ti.IsEntered = false
			ti.Cursor = moveToPreviousWord(ti.Text, ti.Cursor)

		// Handle Option + Right (move one word right) - Detected as "alt+f"
		case msg.String() == "alt+f":
			ti.IsEntered = false
			ti.Cursor = moveToNextWord(ti.Text, ti.Cursor)

		// Handle Arrow Left (move cursor one character left)
		case msg.Type == tea.KeyLeft:
			ti.IsEntered = false
			if ti.Cursor > 0 {
				ti.Cursor--
			}

		// Handle Arrow Right (move cursor one character right)
		case msg.Type == tea.KeyRight:
			ti.IsEntered = false
			if ti.Cursor < len(ti.Text) {
				ti.Cursor++
			}

		// Handle Ctrl+C (quit)
		case msg.Type == tea.KeyCtrlC:
			ti.IsEntered = false
			return ti, tea.Quit, false

		default:
			ti.IsEntered = false

			// Normal typing
			ti.Text = ti.Text[:ti.Cursor] + string(msg.Runes) + ti.Text[ti.Cursor:]
			ti.Cursor += len(msg.Runes)

		}
	}
	return ti, nil, false
}

// Helper function to move the cursor to the beginning of the previous word
func moveToPreviousWord(text string, cursor int) int {
	if cursor == 0 {
		return 0
	}

	// Move cursor left while encountering spaces
	for cursor > 0 && unicode.IsSpace(rune(text[cursor-1])) {
		cursor--
	}
	// Move cursor left until the beginning of the word is found
	for cursor > 0 && !unicode.IsSpace(rune(text[cursor-1])) {
		cursor--
	}
	return cursor
}

// Helper function to move the cursor to the beginning of the next word
func moveToNextWord(text string, cursor int) int {
	if cursor >= len(text) {
		return len(text)
	}

	// Move cursor right while encountering non-space characters (current word)
	for cursor < len(text) && !unicode.IsSpace(rune(text[cursor])) {
		cursor++
	}
	// Move cursor right while encountering spaces
	for cursor < len(text) && unicode.IsSpace(rune(text[cursor])) {
		cursor++
	}
	return cursor
}

func (ti TextInput) View() string {
	var beforeCursor, cursorChar, afterCursor, bottomText string

	if ti.CannotBack {
		bottomText = styles.Text("Press Enter to submit or Ctrl+C to quit.", styles.Gray)
	} else {
		bottomText = styles.Text("Press Enter to submit, Ctrl+Z to go back or Ctrl+C to quit.", styles.Gray)
	}

	feedback := ""
	if ti.IsEntered {
		if err := ti.ValidationFn(ti.Text); err != nil {
			feedback = styles.RenderError(err)
		}
	}

	if len(ti.Text) == 0 {
		return fmt.Sprintf("\n%s %s\n\n%s%s", styles.Text(">", styles.Cyan), styles.Text(ti.Placeholder, styles.Gray), feedback, bottomText)
	} else if ti.Cursor < len(ti.Text) {
		// Cursor is within the text
		beforeCursor = styles.Text(ti.Text[:ti.Cursor], styles.White)
		cursorChar = styles.Cursor(ti.Text[ti.Cursor : ti.Cursor+1])
		afterCursor = styles.Text(ti.Text[ti.Cursor+1:], styles.White)
	} else {
		// Cursor is at the end of the text
		beforeCursor = styles.Text(ti.Text, styles.White)
		cursorChar = styles.Cursor(" ")
	}

	// Compose the full view string
	return fmt.Sprintf("\n%s %s%s%s\n\n%s%s", styles.Text(">", styles.Cyan), beforeCursor, cursorChar, afterCursor, feedback, bottomText)
}

func (ti TextInput) ViewErr(err error) string {
	var beforeCursor, cursorChar, afterCursor string
	bottomText := styles.Text("Press Enter to submit or Ctrl+C to quit.", styles.Gray)
	if len(ti.Text) == 0 {
		return "\n" + styles.Text("> ", styles.Cyan) + styles.Text(ti.Placeholder, styles.Gray) + styles.Cursor(" ") + "\n\n" + styles.RenderError(err) + bottomText
	} else if ti.Cursor < len(ti.Text) {
		// Cursor is within the text
		beforeCursor = styles.Text(ti.Text[:ti.Cursor], styles.White)
		cursorChar = styles.Cursor(ti.Text[ti.Cursor : ti.Cursor+1])
		afterCursor = styles.Text(ti.Text[ti.Cursor+1:], styles.White)
	} else {
		// Cursor is at the end of the text
		beforeCursor = styles.Text(ti.Text, styles.White)
		cursorChar = styles.Cursor(" ")
	}

	feedback := ""
	if ti.IsEntered {
		if err := ti.ValidationFn(ti.Text); err != nil {
			feedback = styles.RenderError(err)
		}
	}

	// Compose the full view string
	return fmt.Sprintf("\n%s %s%s%s\n\n%s%s%s", styles.Text(">", styles.Cyan), beforeCursor, cursorChar, afterCursor, feedback, styles.RenderError(err), bottomText)
}
