package relayer

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/styles"
	"github.com/initia-labs/weave/ui"
)

type FieldType int

const (
	StringField FieldType = iota
	NumberField
	// Add other types as needed
)

type Field struct {
	Name         string
	Type         FieldType
	Question     string
	Placeholder  string
	DefaultValue string
	PrefillValue string
	ValidateFn   func(string) error
	Tooltip      *ui.Tooltip
	Highlights   []string
}

type SubModel struct {
	ui.TextInput
	field      Field
	CannotBack bool
	highlights []string
}

func NewSubModel(field Field) SubModel {
	textInput := ui.NewTextInput(false)
	textInput.WithPlaceholder(field.Placeholder)
	textInput.WithDefaultValue(field.DefaultValue)
	textInput.WithPrefillValue(field.PrefillValue)
	textInput.WithValidatorFn(field.ValidateFn)
	textInput.WithTooltip(field.Tooltip)
	switch field.Type {
	case NumberField:
		textInput.WithValidatorFn(func(input string) error {
			if _, err := strconv.Atoi(input); err != nil {
				return fmt.Errorf("please enter a valid number")
			}
			return nil
		})
	}

	s := SubModel{
		TextInput: textInput,
		field:     field,
	}
	if field.Highlights != nil {
		s.highlights = field.Highlights
	} else {
		s.highlights = []string{"L1", "Rollup", "rollup"}
	}

	return s
}

// Init is a common Init method for all field models
func (m *SubModel) Init() tea.Cmd {
	return nil
}

func (m *SubModel) Update(_ tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *SubModel) UpdateWithContext(ctx context.Context, parent weavecontext.BaseModelInterface, msg tea.Msg) (context.Context, *SubModel, tea.Cmd) {
	input, cmd, done := m.TextInput.Update(msg)
	if done {
		state := weavecontext.PushPageAndGetState[State](parent)
		res := strings.TrimSpace(input.Text)
		state.Config[m.field.Name] = res
		state.weave.PushPreviousResponse(styles.RenderPreviousResponse(styles.DotsSeparator, m.field.Question, m.highlights, res))
		ctx = weavecontext.SetCurrentState(ctx, state)
		return ctx, nil, nil // Done with this field, signal completion
	}
	m.TextInput = input
	return ctx, m, cmd
}

// View is a common View method for all field models
func (m *SubModel) View() string {
	return styles.RenderPrompt(m.field.Question, m.highlights, styles.Question) + m.TextInput.View()
}

func GetField(fields []*Field, name string) (*Field, error) {
	for _, field := range fields {
		if field.Name == name {
			return field, nil
		}
	}
	return nil, fmt.Errorf("field %s not found", name)
}
