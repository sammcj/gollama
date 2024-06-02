// app_model.go

package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sammcj/gollama/logging"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *AppModel) Init() tea.Cmd {
	return nil
}

var docStyle = lipgloss.NewStyle()

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		logging.DebugLogger.Printf("AppModel received key: %s\n", fmt.Sprintf("%+v", msg)) // Convert the message to a string for logging
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Space) && !m.inspecting && !m.filtering():
			if item, ok := m.list.SelectedItem().(Model); ok {
				logging.DebugLogger.Printf("Toggling selection for model: %s (before: %v)\n", item.Name, item.Selected)
				item.Selected = !item.Selected
				m.models[m.list.Index()] = item
				m.list.SetItem(m.list.Index(), item)
				logging.DebugLogger.Printf("Toggled selection for model: %s (after: %v)\n", item.Name, item.Selected)
			}
		case key.Matches(msg, m.keys.Delete) && !m.inspecting && !m.filtering():
			logging.InfoLogger.Println("Delete key pressed")
			m.selectedForDeletion = getSelectedModels(m.models)
			logging.InfoLogger.Printf("Selected models for deletion: %+v\n", m.selectedForDeletion)
			if len(m.selectedForDeletion) == 0 {
				logging.InfoLogger.Println("No models selected for deletion")
			} else {
				m.confirmDeletion = true
			}
		case key.Matches(msg, m.keys.ConfirmYes) && !m.filtering():
			if m.confirmDeletion {
				for _, selectedModel := range m.selectedForDeletion {
					logging.InfoLogger.Printf("Attempting to delete model: %s\n", selectedModel.Name)
					err := deleteModel(m.client, selectedModel.Name)
					if err != nil {
						logging.ErrorLogger.Println("Error deleting model:", err)
					}
				}

				// Remove the selected models from the slice
				m.models = removeModels(m.models, m.selectedForDeletion)
				m.refreshList()
				m.confirmDeletion = false
				m.selectedForDeletion = nil
			}
		case key.Matches(msg, m.keys.ConfirmNo) && !m.filtering():
			if m.confirmDeletion {
				logging.InfoLogger.Println("Deletion cancelled by user")
				m.confirmDeletion = false
				m.selectedForDeletion = nil
			}
		case key.Matches(msg, m.keys.SortByName) && !m.inspecting && !m.filtering():
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].Name < m.models[j].Name
			})
			m.refreshList()
		case key.Matches(msg, m.keys.SortBySize) && !m.inspecting && !m.filtering():
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].Size > m.models[j].Size
			})
			m.refreshList()
		case key.Matches(msg, m.keys.SortByModified) && !m.inspecting && !m.filtering():
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].Modified.After(m.models[j].Modified)
			})
			m.refreshList()
		case key.Matches(msg, m.keys.SortByQuant) && !m.inspecting && !m.filtering():
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].QuantizationLevel < m.models[j].QuantizationLevel
			})
			m.refreshList()
		case key.Matches(msg, m.keys.SortByFamily) && !m.inspecting && !m.filtering():
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].Family < m.models[j].Family
			})
			m.refreshList()
		case key.Matches(msg, m.keys.RunModel) && !m.inspecting && !m.filtering():
			if item, ok := m.list.SelectedItem().(Model); ok {
				runModel(item.Name)
			}
		case key.Matches(msg, m.keys.ClearScreen) && !m.filtering():
			if m.inspecting {
				return m.clearScreen(), tea.ClearScreen
			}
		case key.Matches(msg, m.keys.LinkModel) && !m.filtering():
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
		case key.Matches(msg, m.keys.LinkAllModels) && !m.inspecting && !m.filtering():
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
		default:
			if key.Matches(msg, m.keys.InspectModel) && !m.inspecting && !m.filtering() {
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
			} else if key.Matches(msg, m.keys.Quit) && !m.inspecting && !m.filtering() {
			} else if key.Matches(msg, m.keys.Quit) && m.inspecting && !m.filtering() {
				m.inspecting = false
				return m, nil // Don't quit the application when exiting inspection mode
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *AppModel) View() string {
	if m.inspecting {
		return m.inspectModelView(m.inspectedModel)
	}
	if m.confirmDeletion {
		selectedModelsList := ""
		for _, model := range m.selectedForDeletion {
			selectedModelsList += fmt.Sprintf("- %s\n", model.Name)
		}
		return fmt.Sprintf("Are you sure you want to delete the following models?\n%s(y/N): ", selectedModelsList)
	}

	nameWidth, sizeWidth, quantWidth, modifiedWidth, idWidth, familyWidth := calculateColumnWidths(m.width)

	header := lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s",
			nameWidth, "Name",
			sizeWidth, "Size",
			quantWidth, "Quant",
			familyWidth, "Family",
			modifiedWidth, "Modified",
			idWidth, "ID",
		),
	)
	message := ""
	if m.message != "" {
		message = lipgloss.NewStyle().Foreground(lipgloss.Color("green")).Render(m.message) + "\n"
	}
	return message + header + "\n" + m.list.View()
}

func (m *AppModel) refreshList() {
	items := make([]list.Item, len(m.models))
	for i, model := range m.models {
		items[i] = model
	}
	m.list.SetItems(items)
}

func (m *AppModel) clearScreen() tea.Model {
	m.list.ResetFilter()
	m.refreshList()
	return m
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

// a function that returns true if the user is currently entering a filter string (and they haven't pressed enter)
func (m *AppModel) filtering() bool {
	return m.list.FilterState() == list.Filtering
}
