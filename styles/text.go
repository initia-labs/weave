package styles

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	FooterLine         = BoldText("│ ", Gray)
	FooterCommands     = []string{"Enter", "Ctrl+c", "q", "Ctrl+z", "Ctrl+t", "Space", "arrow-keys"}
	HiddenMnemonicText = Text("*Mnemonic has been entered and is now hidden for security purposes.*", Ivory)
)

func SetColor(text string, color HexColor) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	return style.Render(text)
}

// Text applies a hex color to a given string and returns the styled string
func Text(text string, color HexColor) string {
	coloredText := ""

	texts := strings.Split(text, "\n")
	for idx, t := range texts {
		coloredText += SetColor(t, color)
		if idx != len(texts)-1 {
			coloredText += "\n"
		}
	}
	return coloredText
}

func SetBoldColor(text string, color HexColor) string {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color))
	return style.Render(text)
}

// BoldText applies bold and color styling to a string
func BoldText(text string, color HexColor) string {
	styledText := ""

	texts := strings.Split(text, "\n")
	for idx, t := range texts {
		styledText += SetBoldColor(t, color)
		if idx != len(texts)-1 {
			styledText += "\n"
		}
	}
	return styledText
}

func SetBoldUnderlineColor(text string, color HexColor) string {
	style := lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color(color))
	return style.Render(text)
}

func BoldUnderlineText(text string, color HexColor) string {
	styledText := ""

	texts := strings.Split(text, "\n")
	for idx, t := range texts {
		styledText += SetBoldUnderlineColor(t, color)
		if idx != len(texts)-1 {
			styledText += "\n"
		}
	}
	return styledText
}

func Cursor(cursorChar string) string {
	cursorStyle := lipgloss.NewStyle().
		Bold(true).                        // Make the cursor bold
		Reverse(true).                     // Reverse the foreground and background colors
		Background(lipgloss.Color(Black)). // Black background
		Foreground(lipgloss.Color(White))  // White foreground

	return cursorStyle.Render(cursorChar)
}

func FadeText(text string) string {
	fadedColors := []HexColor{
		"#27D8FF", // Cyan
		"#25ccf0", // Intermediate color 1
		"#22bfe2", // Intermediate color 2
		"#20b3d3", // Intermediate color 3
		"#1ea7c5", // Intermediate color 4
		"#1c9bb6", // Intermediate color 5
		"#198ea8", // Intermediate color 6
		"#178299", // DarkCyan
	}

	result := ""

	for i, char := range text {
		color := fadedColors[i*len(fadedColors)/len(text)]
		result += Text(string(char), color)
	}

	return result
}

type PromptStatus string

const (
	Empty       PromptStatus = "empty"
	Checked     PromptStatus = "checked"
	Question    PromptStatus = "question"
	Information PromptStatus = "information"
	Completed   PromptStatus = "completed"
)

var (
	QuestionMark    = Text("? ", Cyan)
	CorrectMark     = Text("✓ ", Green)
	InformationMark = Text("i ", Cyan)
	SelectorCursor  = Text("> ", Cyan)
)

// RenderPrompt highlights phrases in the text if they match any phrase in the highlight list
func RenderPrompt(text string, highlights []string, status PromptStatus) string {
	prompt := ""
	if status == Question {
		prompt += "\n"
	}
	switch status {
	case Question:
		prompt += QuestionMark
	case Checked:
		prompt += CorrectMark
	case Information:
		prompt += InformationMark
	case Completed:
		prompt += CorrectMark
	}

	if status != Completed {
		for _, highlight := range highlights {
			if strings.Contains(text, highlight) {
				text = strings.ReplaceAll(text, highlight, BoldText(highlight, Cyan))
			}
		}
		text = DefaultTextWithoutOverridingStyledText(text)
	} else {
		text = FadeText(text)
	}
	return prompt + text
}

func TextWithoutOverridingStyledText(text string, color HexColor) string {
	styledText := ""
	for _, line := range strings.Split(text, "\n") {
		// Split the line by ANSI escape codes to detect already styled substrings
		parts := splitPreservingANSI(line)
		for _, part := range parts {
			if !containsANSI(part) {
				// Only style the parts that don't already have styling
				part = Text(part, color)
			}
			styledText += part
		}
		styledText += "\n"
	}
	return strings.TrimSuffix(styledText, "\n")
}

// DefaultTextWithoutOverridingStyledText is a helper function
// to apply default styling without overriding existing styles
func DefaultTextWithoutOverridingStyledText(text string) string {
	return TextWithoutOverridingStyledText(text, Ivory)
}

// Utility functions to handle ANSI escape codes
func splitPreservingANSI(text string) []string {
	// Implement splitting that preserves ANSI codes
	// This is a placeholder; you'll need to implement or use a library
	return []string{text}
}

func containsANSI(text string) bool {
	// Check if the text contains ANSI escape codes
	// This is a placeholder; you'll need to implement or use a library
	return strings.Contains(text, "\x1b[")
}

func StripANSI(text string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(text, "")
}

var (
	NoSeparator    = ""
	ArrowSeparator = Text(" > ", Gray)
	DotsSeparator  = Text(" ... ", Gray)
)

func RenderPreviousResponse(separator string, question string, highlights []string, answer string) string {
	return RenderPrompt(question, highlights, Checked) + separator + answer + "\n"
}

func RenderError(err error) string {
	return TextWithoutOverridingStyledText(fmt.Sprintf("%v\n", err), Yellow)
}

func RenderFooter(text string) string {
	words := strings.Fields(text)

	for i, word := range words {
		if slices.Contains(FooterCommands, word) {
			words[i] = BoldText(word, LightGray)
		} else {
			words[i] = Text(word, Gray)
		}
	}

	styledText := strings.Join(words, " ")
	return FooterLine + styledText
}

func RenderKey(keyName, address string) string {
	text := BoldText("Key Name: ", Ivory) + keyName + "\n" +
		BoldText("Address: ", Ivory) + address + "\n"
	return text
}

func CreateFrame(text string, maxWidth int) string {
	lines := strings.Split(text, "\n")
	top := "┌" + strings.Repeat("─", maxWidth+2) + "┐"
	bottom := "└" + strings.Repeat("─", maxWidth+2) + "┘"

	var framedContent strings.Builder
	for _, line := range lines {
		framedContent.WriteString(fmt.Sprintf("│ %-*s │\n", maxWidth, line))
	}

	return fmt.Sprintf("%s\n%s%s", top, framedContent.String(), bottom)
}
