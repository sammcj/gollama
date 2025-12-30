// text_input.go contains the textInputModel struct which is used to render the text input view.
package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sammcj/gollama/v2/logging"
	"github.com/sammcj/gollama/v2/styles"
)

type textInputModel struct {
	textInput textinput.Model
	oldName   string
	quitting  bool
	cancelled bool
}

// promptForNewName displays a text input prompt for renaming a model.
// Returns the new name and a boolean indicating if the operation was cancelled.
func promptForNewName(oldName string) (string, bool) {
	ti := textinput.New()
	// print 'renaming oldName' to the console with the oldName in purple
	ti.Prompt = oldName + "\n" + "Name for new model: "
	ti.Placeholder = oldName
	ti.Focus()

	ti.KeyMap.AcceptSuggestion = key.NewBinding(key.WithKeys("tab"))
	ti.SetSuggestions([]string{oldName})
	ti.ShowSuggestions = true
	ti.CharLimit = 300
	ti.Width = 140

	ti.PromptStyle = styles.PromptStyle()
	ti.TextStyle = styles.InputTextStyle()
	ti.Cursor.Style = styles.CursorStyle()
	ti.PlaceholderStyle = styles.PlaceholderStyle()

	m := textInputModel{
		textInput: ti,
		oldName:   oldName,
	}

	p := tea.NewProgram(&m)
	if _, err := p.Run(); err != nil {
		logging.ErrorLogger.Printf("Error starting text input program: %v\n", err)
	}

	// If the user cancelled (Ctrl-C or Esc), return empty string and cancelled=true
	if m.cancelled {
		logging.InfoLogger.Println("Rename cancelled by user")
		return "", true
	}

	newName := m.textInput.Value()

	// If user pressed enter without typing anything, return empty string
	if newName == "" {
		logging.DebugLogger.Println("No new name entered, returning empty string")
		return "", false
	}

	return newName, false
}

func (m *textInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			m.quitting = true
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m textInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m textInputModel) View() string {
	if m.quitting {
		return ""
	}
	return fmt.Sprintf(
		"\n%s\n\n%s",
		m.textInput.View(),
		"(esc or ctrl+c to cancel)",
	)
}
