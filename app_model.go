package main

import (
	"fmt"
	"os"
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
		m.list.SetSize(m.width, m.height-5)
		listHeight := msg.Height - v
		if topRunning {
			listHeight -= 5 // Adjust height when top is running
		}
		m.list.SetSize(msg.Width-h, listHeight)
		logging.DebugLogger.Printf("AppModel received key: %s\n", fmt.Sprintf("%+v", msg)) // Convert the message to a string for logging
	case tea.KeyMsg:

		switch {
		case key.Matches(msg, m.keys.Space) && m.inputAllowed():
			if item, ok := m.list.SelectedItem().(Model); ok {
				logging.DebugLogger.Printf("Toggling selection for model: %s (before: %v)\n", item.Name, item.Selected)
				item.Selected = !item.Selected
				m.models[m.list.Index()] = item
				m.list.SetItem(m.list.Index(), item)
				logging.DebugLogger.Printf("Toggled selection for model: %s (after: %v)\n", item.Name, item.Selected)
			}
		case key.Matches(msg, m.keys.Delete) && m.inputAllowed():
			logging.InfoLogger.Println("Delete key pressed")
			m.selectedForDeletion = getSelectedModels(m.models)
			logging.InfoLogger.Printf("Selected models for deletion: %+v\n", m.selectedForDeletion)
			if len(m.selectedForDeletion) == 0 {
				logging.InfoLogger.Println("No models selected for deletion")
			} else {
				m.confirmDeletion = true
			}

		case key.Matches(msg, m.keys.Top) && m.inputAllowed():
			if m.showTop {
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					running, err := showRunningModels(m.client)
					if err != nil {
						return fmt.Sprintf("Error showing running models: %v", err)
					}
					return running
				})
			}

		case key.Matches(msg, m.keys.ConfirmYes) && m.inputAllowed():
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
		case key.Matches(msg, m.keys.ConfirmNo) && m.inputAllowed():
			if m.confirmDeletion {
				logging.InfoLogger.Println("Deletion cancelled by user")
				m.confirmDeletion = false
				m.selectedForDeletion = nil
			}
		case key.Matches(msg, m.keys.SortByName) && m.inputAllowed():
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].Name < m.models[j].Name
			})
			m.refreshList()
		case key.Matches(msg, m.keys.SortBySize) && m.inputAllowed():
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].Size > m.models[j].Size
			})
			m.refreshList()
		case key.Matches(msg, m.keys.SortByModified) && m.inputAllowed():
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].Modified.After(m.models[j].Modified)
			})
			m.refreshList()
		case key.Matches(msg, m.keys.SortByQuant) && m.inputAllowed():
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].QuantizationLevel < m.models[j].QuantizationLevel
			})
			m.refreshList()
		case key.Matches(msg, m.keys.SortByFamily) && m.inputAllowed():
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].Family < m.models[j].Family
			})
			m.refreshList()
		case key.Matches(msg, m.keys.RunModel) && m.inputAllowed():
			if item, ok := m.list.SelectedItem().(Model); ok {
				runModel(item.Name)
				os.Exit(0) // Exit the application after running the model
			}
		case key.Matches(msg, m.keys.ClearScreen) && m.inputAllowed():
			if m.inspecting {
				return m.clearScreen(), tea.ClearScreen
			}
		case key.Matches(msg, m.keys.LinkModel) && m.inputAllowed():
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
		case key.Matches(msg, m.keys.LinkAllModels) && m.inputAllowed():
			var messages []string
			for _, model := range m.models {
				message, err := linkModel(model.Name, m.lmStudioModelsDir, m.noCleanup)
				// if the message is empty, don't add it to the list
				if err != nil {
					messages = append(messages, fmt.Sprintf("Error linking model %s: %v", model.Name, err))
				} else if message != "" {
					continue
				} else {
					messages = append(messages, fmt.Sprintf("Model %s linked successfully", model.Name))
				}
			}
			// remove any empty messages or duplicates
			for i := 0; i < len(messages); i++ {
				for j := i + 1; j < len(messages); j++ {
					if messages[i] == messages[j] {
						messages = append(messages[:j], messages[j+1:]...)
					}
				}
			}
			messages = append(messages, "Linking complete")
			m.message = strings.Join(messages, "\n")
		case key.Matches(msg, m.keys.CopyModel) && m.inputAllowed():
			if item, ok := m.list.SelectedItem().(Model); ok {
				newName := promptForNewName(item.Name, item) // Pass the selected item as the model
				copyModel(m.client, item.Name, newName)
				m.message = fmt.Sprintf("Model %s copied to %s", item.Name, newName)
			}
		case key.Matches(msg, m.keys.PushModel) && m.inputAllowed():
			if item, ok := m.list.SelectedItem().(Model); ok {
				pushModel(m.client, item.Name)
				m.message = fmt.Sprintf("Model %s pushed successfully", item.Name)
			}

		case key.Matches(msg, m.keys.InspectModel) && m.inputAllowed():
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
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *AppModel) ToggleTop() (tea.Model, tea.Cmd) {
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
	return fmt.Sprintf("\nAre you sure you want to delete the selected models?\n\n%s\n\n%s %s\n%s %s",
		strings.Join(m.selectedModelNames(), "\n"),
		m.keys.ConfirmYes.Help().Key, "Yes",
		m.keys.ConfirmNo.Help().Key, "No",
	)
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

// inputAllowed returns true if the user can interact with the application
func (m *AppModel) inputAllowed() bool {
	if m.inspecting || m.filtering() || m.list.FilterInput.Value() == "" || m.confirmDeletion {
		return true
	}
	return false
}
