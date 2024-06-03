package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sammcj/gollama/logging"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *AppModel) Init() tea.Cmd {
	if m.showTop {
		return m.startTopTicker()
	}
	return nil
}

var docStyle = lipgloss.NewStyle()
var topRunning = false

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.width, m.height = msg.Width, msg.Height
		listHeight := m.height - v - 5 // Adjust for potential additional UI elements
		if topRunning {
			listHeight -= 5 // Adjust height when top is running
		}
		m.list.SetSize(m.width-h, listHeight)
		logging.DebugLogger.Printf("AppModel received key: %s\n", fmt.Sprintf("%+v", msg)) // Convert the message to a string for logging
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			// If filtering, do not process other keybindings
			break
		}

		switch {
		case key.Matches(msg, m.keys.Space):
			if item, ok := m.list.SelectedItem().(Model); ok {
				logging.DebugLogger.Printf("Toggling selection for model: %s (before: %v)\n", item.Name, item.Selected)
				item.Selected = !item.Selected
				m.models[m.list.Index()] = item
				m.list.SetItem(m.list.Index(), item)
				logging.DebugLogger.Printf("Toggled selection for model: %s (after: %v)\n", item.Name, item.Selected)
			}
		case key.Matches(msg, m.keys.Delete):
			logging.InfoLogger.Println("Delete key pressed")
			m.selectedForDeletion = getSelectedModels(m.models)
			logging.InfoLogger.Printf("Selected models for deletion: %+v\n", m.selectedForDeletion)
			if len(m.selectedForDeletion) == 0 {
				logging.InfoLogger.Println("No models selected for deletion")
			} else {
				m.confirmDeletion = true
			}
		case key.Matches(msg, m.keys.Top):
			var cmd tea.Cmd
			m, cmd = m.ToggleTop()
			return m, cmd
		case key.Matches(msg, m.keys.ConfirmYes):
			if m.confirmDeletion {
				for _, selectedModel := range m.selectedForDeletion {
					logging.InfoLogger.Printf("Attempting to delete model: %s\n", selectedModel.Name)
					err := deleteModel(m.client, selectedModel.Name)
					if err != nil {
						logging.ErrorLogger.Println("Error deleting model:", err)
					}
				}
				m.models = removeModels(m.models, m.selectedForDeletion)
				m.refreshList()
				m.confirmDeletion = false
				m.selectedForDeletion = nil
			}
		case key.Matches(msg, m.keys.ConfirmNo):
			if m.confirmDeletion {
				logging.InfoLogger.Println("Deletion cancelled by user")
				m.confirmDeletion = false
				m.selectedForDeletion = nil
			}
		case key.Matches(msg, m.keys.SortByName):
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].Name < m.models[j].Name
			})
			m.refreshList()
		case key.Matches(msg, m.keys.SortBySize):
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].Size > m.models[j].Size
			})
			m.refreshList()
		case key.Matches(msg, m.keys.SortByModified):
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].Modified.After(m.models[j].Modified)
			})
			m.refreshList()
		case key.Matches(msg, m.keys.SortByQuant):
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].QuantizationLevel < m.models[j].QuantizationLevel
			})
			m.refreshList()
		case key.Matches(msg, m.keys.SortByFamily):
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].Family < m.models[j].Family
			})
			m.refreshList()
		case key.Matches(msg, m.keys.RunModel):
			if item, ok := m.list.SelectedItem().(Model); ok {
				logging.InfoLogger.Printf("Running model: %s\n", item.Name)
				return m, runModel(item.Name)
			}
		case key.Matches(msg, m.keys.AltScreen):
			m.altscreenActive = !m.altscreenActive
			cmd := tea.EnterAltScreen
			if !m.altscreenActive {
				cmd = tea.ExitAltScreen
			}
			return m, cmd
		case key.Matches(msg, m.keys.ClearScreen):
			if m.inspecting {
				return m.clearScreen(), tea.ClearScreen
			}
		case key.Matches(msg, m.keys.LinkModel):
			if item, ok := m.list.SelectedItem().(Model); ok {
				message, err := linkModel(item.Name, m.lmStudioModelsDir, m.noCleanup)
				if err != nil {
					m.message = fmt.Sprintf("Error linking model: %v", err)
				} else if message != "" {
					break
				} else {
					m.message = fmt.Sprintf("Model %s linked successfully", item.Name)
				}
			}
		case key.Matches(msg, m.keys.LinkAllModels):
			var messages []string
			for _, model := range m.models {
				message, err := linkModel(model.Name, m.lmStudioModelsDir, m.noCleanup)
				if err != nil {
					messages = append(messages, fmt.Sprintf("Error linking model %s: %v", model.Name, err))
				} else if message != "" {
					continue
				} else {
					messages = append(messages, fmt.Sprintf("Model %s linked successfully", model.Name))
				}
			}
			messages = append(messages, "Linking complete")
			m.message = strings.Join(messages, "\n")
		case key.Matches(msg, m.keys.CopyModel):
			if item, ok := m.list.SelectedItem().(Model); ok {
				newName := promptForNewName(item.Name) // Pass the selected item as the model

				if newName == "" {
					m.message = "Error: name can't be empty"
				} else {
					copyModel(m.client, item.Name, newName)
					m.message = fmt.Sprintf("Model %s copied to %s", item.Name, newName)
				}
			}
		case key.Matches(msg, m.keys.PushModel):
			if item, ok := m.list.SelectedItem().(Model); ok {
				m.message = lipgloss.NewStyle().Foreground(lipgloss.Color("129")).Render(fmt.Sprintf("Pushing model: %s\n", item.Name))
				return m.startPushModel(item.Name)
			}
		case key.Matches(msg, m.keys.InspectModel):
			selectedItem := m.list.SelectedItem()
			if selectedItem != nil {
				model, ok := selectedItem.(Model)
				if !ok {
					return m, nil // This should never happen
				}
				m.inspecting = true
				m.inspectedModel = model                                     // Ensure inspectedModel is set correctly
				logging.DebugLogger.Printf("Inspecting model: %+v\n", model) // Log the inspected model
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			if m.inspecting {
				m.inspecting = false
				return m, nil
			}
		}
	case runFinishedMessage:
		logging.DebugLogger.Printf("Run finished message: %v\n", msg)
		if msg.err != nil {
			logging.ErrorLogger.Printf("Error running model: %v\n", msg.err)
			m.message = fmt.Sprintf("Error running model: %v\n", msg.err)
			return m, nil
		}
	case progressMsg:
		// Just trigger the next tick for progress updates
		return m, tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
			return progressMsg{modelName: msg.modelName}
		})

	case pushSuccessMsg:
		m.message = fmt.Sprintf("Successfully pushed model: %s\n", msg.modelName)
		return m, nil

	case pushErrorMsg:
		logging.ErrorLogger.Printf("Error pushing model: %v\n", msg.err)
		m.message = fmt.Sprintf("Error pushing model: %v\n", msg.err)
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *AppModel) ToggleTop() (*AppModel, tea.Cmd) {
	if topRunning {
		m.message = ""
		topRunning = false
		m.list.SetSize(m.width, m.height) // Reset list size when top is not running
		return m, nil
	}

	topRunning = true
	m.list.SetSize(m.width, m.height-5) // Adjust list size when top is running
	return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		running, err := showRunningModels(m.client)
		if err != nil {
			return fmt.Sprintf("Error showing running models: %v", err)
		}
		return running
	})
}

func (m *AppModel) View() string {
	if m.confirmDeletion {
		return m.confirmDeletionView()
	}

	if m.inspecting {
		return m.inspectModelView(m.inspectedModel)
	}
	if m.filtering() {
		return m.filterView()
	}
	return m.list.View() + "\n" + m.message
}

func (m *AppModel) confirmDeletionView() string {
	return fmt.Sprintf("\nAre you sure you want to delete the selected models? (Y/N)\n\n%s\n\n%s\n%s",
		strings.Join(m.selectedModelNames(), "\n"),
		m.keys.ConfirmYes.Help().Key,
		m.keys.ConfirmNo.Help().Key)
}

func (m *AppModel) inspectModelView(model Model) string {
	logging.DebugLogger.Printf("Inspecting model view: %+v\n", model) // Log the model being inspected

	columns := []table.Column{
		{Title: "Property", Width: 20},
		{Title: "Value", Width: 50},
	}

	rows := []table.Row{
		{"Name", model.Name},
		{"ID", model.ID},
		{"Size (GB)", fmt.Sprintf("%.2f", model.Size)},
		{"Quantization Level", model.QuantizationLevel},
		{"Modified", model.Modified.Format("2006-01-02")},
		{"Family", model.Family},
	}

	// Log the rows to ensure they are being populated correctly
	for _, row := range rows {
		logging.DebugLogger.Printf("Row: %v\n", row)
	}

	// Create the table with the columns and rows
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(rows)+1),
	)

	// Set the table styles
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57"))
	t.SetStyles(s)

	// Render the table view
	return "\n" + t.View() + "\nPress escape to return to the main view."
}

func (m *AppModel) filterView() string {
	return m.filterInput.View()
}

func (m *AppModel) selectedModelNames() []string {
	var names []string
	for _, model := range m.selectedForDeletion {
		names = append(names, model.Name)
	}
	return names
}

func (m *AppModel) refreshList() {
	items := make([]list.Item, len(m.models))
	for i, model := range m.models {
		items[i] = model
	}
	m.list.SetItems(items)
}

func (m *AppModel) clearScreen() tea.Model {
	m.inspecting = false
	m.table = table.New()
	return m
}

func (m *AppModel) filtering() bool {
	// Implement the filtering logic for your application
	return false
}

func (m *AppModel) startTopTicker() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		running, err := showRunningModels(m.client)
		if err != nil {
			return fmt.Sprintf("Error showing running models: %v", err)
		}
		return running
	})
}
