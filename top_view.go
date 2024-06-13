// top_view.go contains the TopModel struct which is used to render the top view of the application.
package main

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ollama/ollama/api"
	"github.com/sammcj/gollama/logging"
)

type TopModel struct {
	client   *api.Client
	table    table.Model
	quitting bool
}

func NewTopModel(client *api.Client) *TopModel {
	columns := []table.Column{
		{Title: "Name", Width: 40},
		{Title: "Size (GB)", Width: 10},
		{Title: "VRAM (GB)", Width: 10},
		{Title: "Until", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithHeight(10),
	)

	t.KeyMap.LineUp.SetKeys("up")
	t.KeyMap.LineDown.SetKeys("down")
	t.Focused()

	return &TopModel{
		client: client,
		table:  t,
	}
}

func (m *TopModel) Init() tea.Cmd {
	return m.updateRunningModels()
}

func (m *TopModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(msg.Height)

	case tea.Msg:
		return m, m.updateRunningModels()
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *TopModel) View() string {
	if m.quitting {
		return "Returning to main view...\n"
	}
	return lipgloss.NewStyle().Render(m.table.View())
}

func (m *TopModel) updateRunningModels() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		running, err := showRunningModels(m.client)
		if err != nil {
			logging.ErrorLogger.Printf("Error showing running models: %v", err)
			return fmt.Sprintf("Error showing running models: %v", err)
		}
		m.table.SetRows(running)
		return nil
	})
}
