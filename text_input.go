package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sammcj/gollama/logging"
)

type textInputModel struct {
	textInput textinput.Model
	oldName   string
	quitting  bool
}

// text_input.go (modified)
func promptForNewName(oldName string) string {
	ti := textinput.New()
	ti.Placeholder = "Enter new name"
	ti.Focus()
	ti.Prompt = "New name for model: "
	ti.CharLimit = 156
	ti.Width = 20

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
		fmt.Println("Error: New name cannot be empty")
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
		"Old name: %s\n%s\n\n%s",
		m.oldName,
		m.textInput.View(),
		"(ctrl+c to cancel)",
	)
}
