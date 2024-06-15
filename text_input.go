// text_input.go contains the textInputModel struct which is used to render the text input view.
package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sammcj/gollama/logging"
)

type textInputModel struct {
	textInput textinput.Model
	oldName   string
	quitting  bool
}

// promptForNewName displays a text input prompt for renaming a model.
func promptForNewName(oldName string) string {
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

	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF"))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF"))
	ti.Cursor.Style = lipgloss.NewStyle().Background(lipgloss.Color("#4E00FF")).Background(lipgloss.Color("#111111"))
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#AD00FF"))

	m := textInputModel{
		textInput: ti,
		oldName:   oldName,
	}

	p := tea.NewProgram(&m)
	if _, err := p.Run(); err != nil {
		logging.ErrorLogger.Printf("Error starting text input program: %v\n", err)
	}

	newName := m.textInput.Value()

	if newName == "" {
		// error handling
		logging.ErrorLogger.Println("No new name entered, returning old name")
		return oldName
	}

	return newName
}

func (m *textInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "enter":
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
		"(ctrl+c to cancel)",
	)
}
