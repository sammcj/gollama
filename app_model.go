package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sammcj/gollama/logging"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *AppModel) Init() tea.Cmd {
	return nil
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		logging.DebugLogger.Printf("AppModel received key: %s\n", msg.String())
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
				break
			}
			m.confirmDeletion = true
		case key.Matches(msg, m.keys.ConfirmYes):
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
			m.keys.SortOrder = "name"
		case key.Matches(msg, m.keys.SortBySize):
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].Size > m.models[j].Size
			})
			m.refreshList()
			m.keys.SortOrder = "size"
		case key.Matches(msg, m.keys.SortByModified):
			sort.Slice(m.models, func(i, j int) bool {
				return m.models[i].Modified.After(m.models[j].Modified)
			})
			m.refreshList()
			m.keys.SortOrder = "modified"
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
			m.keys.SortOrder = "family"
		case key.Matches(msg, m.keys.RunModel):
			if item, ok := m.list.SelectedItem().(Model); ok {
				runModel(item.Name)
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
			return m.clearScreen(), tea.ClearScreen
		case key.Matches(msg, m.keys.LinkAllModels):
			var messages []string
			for _, model := range m.models {
				message, err := linkModel(model.Name, m.lmStudioModelsDir, m.noCleanup)
				// if the message is empty, don't add it to the list
				if err != nil {
					messages = append(messages, fmt.Sprintf("Error linking model %s: %v", model.Name, err))
				} else if message != "" {
					continue
				} else {
					messages = append(messages, message)
				}
			}
			// remove any empty messages or duplicates
			for i := 0; i < len(messages); i++ {
				for j := i + 1; j < len(messages); j++ {
					if messages[i] == messages[j] {
						messages = append(messages[:j], messages[j+1:]...)
						j--
					}
				}
			}
			messages = append(messages, "Linking complete")
			m.message = strings.Join(messages, "\n")
			return m.clearScreen(), tea.ClearScreen
		case key.Matches(msg, m.keys.ClearScreen):
			return m.clearScreen(), tea.ClearScreen
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(m.width, m.height-5)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *AppModel) View() string {
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
