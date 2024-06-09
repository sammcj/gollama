// app_model.go not to be confused with an Ollama model - the app model is the tea.Model for the application.
package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sammcj/gollama/logging"
)

type View int

const (
	MainView View = iota
	TopView
)

func (m *AppModel) Init() tea.Cmd {
	if m.showTop {
		return m.startTopTicker()
	}
	return nil
}

func (m *AppModel) FilterValue() string {
	return m.filterInput.View()
}

// var docStyle = lipgloss.NewStyle()
var topRunning = false

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case runFinishedMessage:
		return m.handleRunFinishedMessage(msg)
	case progressMsg:
		return m.handleProgressMsg(msg)
	case editorFinishedMsg:
		return m.handleEditorFinishedMsg(msg)
	case pushSuccessMsg:
		return m.handlePushSuccessMsg(msg)
	case pushErrorMsg:
		return m.handlePushErrorMsg(msg)
	default:
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
}

func (m *AppModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	logging.DebugLogger.Printf("Received key: %s\n", msg.String())

	// Log the current filter state
	logging.DebugLogger.Printf("Current filter state: %v\n", m.list.FilterState())

	// Handle the space key separately to ensure it works even when filtering
	if key.Matches(msg, m.keys.Space) {
		return m.handleSpaceKey()
	}

	// Handle filtering state
	if m.list.FilterState() == list.Filtering {
		var cmd tea.Cmd
		switch msg.String() {
		case "enter":
			// Confirm the filter
			logging.DebugLogger.Println("Confirming filter")
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		case "esc":
			// Clear the filter
			logging.DebugLogger.Println("Clearing filter")
			m.list.ResetFilter()
			return m, nil
		default:
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
	}

	// Handle other keys
	switch msg.String() {
	case "q":
		if m.list.FilterState() == list.Filtering {
			logging.DebugLogger.Println("Clearing filter with 'q' key")
			m.list.ResetFilter()
			return m, nil
		}
		if m.view == TopView || m.inspecting {
			m.view = MainView
			m.inspecting = false
			m.editing = false
			return m, nil
		} else {
			return m, tea.Quit
		}
	case "esc":
		if m.list.FilterState() == list.Filtering {
			logging.DebugLogger.Println("Clearing filter with 'esc' key")
			m.list.ResetFilter()
			return m, nil
		}
		if m.view == TopView || m.inspecting {
			m.view = MainView
			m.inspecting = false
			m.editing = false
			return m, nil
		} else {
			return m, tea.Quit
		}
	case "ctrl+c":
		if m.editing {
			m.editing = false
			return m, nil
		}
		return m, tea.Quit
	}

	if m.confirmDeletion {
		switch {
		case key.Matches(msg, m.keys.ConfirmYes):
			logging.DebugLogger.Println("ConfirmYes key matched")
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
		case key.Matches(msg, m.keys.ConfirmNo):
			logging.DebugLogger.Println("ConfirmNo key matched")
			logging.InfoLogger.Println("Deletion cancelled by user")
			m.confirmDeletion = false
			m.selectedForDeletion = nil
		}
		return m, nil
	}

	var cmd tea.Cmd // Define the cmd variable
	switch {
	case key.Matches(msg, m.keys.Delete):
		return m.handleDeleteKey()
	case key.Matches(msg, m.keys.SortByName):
		return m.handleSortByNameKey()
	case key.Matches(msg, m.keys.SortBySize):
		return m.handleSortBySizeKey()
	case key.Matches(msg, m.keys.SortByModified):
		return m.handleSortByModifiedKey()
	case key.Matches(msg, m.keys.SortByQuant):
		return m.handleSortByQuantKey()
	case key.Matches(msg, m.keys.SortByFamily):
		return m.handleSortByFamilyKey()
	case key.Matches(msg, m.keys.RunModel):
		return m.handleRunModelKey()
	case key.Matches(msg, m.keys.AltScreen):
		return m.handleAltScreenKey()
	case key.Matches(msg, m.keys.ClearScreen):
		return m.handleClearScreenKey()
	case key.Matches(msg, m.keys.UpdateModel):
		return m.handleUpdateModelKey()
	case key.Matches(msg, m.keys.LinkModel):
		return m.handleLinkModelKey()
	case key.Matches(msg, m.keys.LinkAllModels):
		return m.handleLinkAllModelsKey()
	case key.Matches(msg, m.keys.CopyModel):
		return m.handleCopyModelKey()
	case key.Matches(msg, m.keys.PushModel):
		return m.handlePushModelKey()
	case key.Matches(msg, m.keys.InspectModel):
		return m.handleInspectModelKey()
	default:
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
}

func (m *AppModel) handleSpaceKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("Space key matched")
	if item, ok := m.list.SelectedItem().(Model); ok {
		logging.DebugLogger.Printf("Toggling selection for model: %s (before: %v)\n", item.Name, item.Selected)
		item.Selected = !item.Selected

		// Update the item in the list's items
		items := m.list.Items()
		for i, listItem := range items {
			if model, ok := listItem.(Model); ok && model.Name == item.Name {
				items[i] = item
				break
			}
		}

		m.list.SetItems(items)

		// Find the index of the actual item in the full list and update it
		for i, model := range m.models {
			if model.Name == item.Name {
				m.models[i] = item
				break
			}
		}

		logging.DebugLogger.Printf("Toggled selection for model: %s (after: %v)\n", item.Name, item.Selected)
	}
	return m, nil
}

func (m *AppModel) handleRunFinishedMessage(msg runFinishedMessage) (tea.Model, tea.Cmd) {
	logging.DebugLogger.Printf("Run finished message: %v\n", msg)
	if msg.err != nil {
		logging.ErrorLogger.Printf("Error running model: %v\n", msg.err)
		m.message = fmt.Sprintf("Error running model: %v\n", msg.err)
	}
	return m, nil
}

func (m *AppModel) handleProgressMsg(msg progressMsg) (tea.Model, tea.Cmd) {
	return m, tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return progressMsg{modelName: msg.modelName}
	})
}

func (m *AppModel) handleEditorFinishedMsg(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.message = fmt.Sprintf("Error editing modelfile: %v", msg.err)
		return m, nil
	}
	if item, ok := m.list.SelectedItem().(Model); ok {
		newModelName := promptForNewName(item.Name)
		modelfilePath := fmt.Sprintf("Modelfile-%s", strings.ReplaceAll(newModelName, " ", "_"))
		err := createModelFromModelfile(newModelName, modelfilePath)
		if err != nil {
			m.message = fmt.Sprintf("Error creating model: %v", err)
			return m, nil
		}
		m.message = fmt.Sprintf("Model %s created successfully", newModelName)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *AppModel) handlePushSuccessMsg(msg pushSuccessMsg) (tea.Model, tea.Cmd) {
	m.message = fmt.Sprintf("Successfully pushed model: %s\n", msg.modelName)
	m.showProgress = false // Hide progress bar
	return m, nil
}

func (m *AppModel) handlePushErrorMsg(msg pushErrorMsg) (tea.Model, tea.Cmd) {
	logging.ErrorLogger.Printf("Error pushing model: %v\n", msg.err)
	m.message = fmt.Sprintf("Error pushing model: %v\n", msg.err)
	m.showProgress = false // Hide progress bar
	return m, nil
}

func (m *AppModel) handleDeleteKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("Delete key matched")
	logging.InfoLogger.Println("Delete key pressed")

	// Collect all selected models for deletion
	var selectedModels []Model
	for _, item := range m.list.Items() {
		if model, ok := item.(Model); ok && model.Selected {
			selectedModels = append(selectedModels, model)
		}
	}

	if len(selectedModels) > 0 {
		m.selectedForDeletion = selectedModels
		logging.InfoLogger.Printf("Selected models for deletion: %+v\n", m.selectedForDeletion)
		m.confirmDeletion = true
	} else if item, ok := m.list.SelectedItem().(Model); ok {
		m.selectedForDeletion = []Model{item}
		logging.InfoLogger.Printf("Selected model for deletion: %+v\n", m.selectedForDeletion)
		m.confirmDeletion = true
	}
	return m, nil
}

func (m *AppModel) handleSortByNameKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("SortByName key matched")
	sort.Slice(m.models, func(i, j int) bool {
		return m.models[i].Name < m.models[j].Name
	})
	m.refreshList()
	return m, nil
}

func (m *AppModel) handleSortBySizeKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("SortBySize key matched")
	sort.Slice(m.models, func(i, j int) bool {
		return m.models[i].Size > m.models[j].Size
	})
	m.refreshList()
	return m, nil
}

func (m *AppModel) handleSortByModifiedKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("SortByModified key matched")
	sort.Slice(m.models, func(i, j int) bool {
		return m.models[i].Modified.After(m.models[j].Modified)
	})
	m.refreshList()
	return m, nil
}

func (m *AppModel) handleSortByQuantKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("SortByQuant key matched")
	sort.Slice(m.models, func(i, j int) bool {
		return m.models[i].QuantizationLevel < m.models[j].QuantizationLevel
	})
	m.refreshList()
	return m, nil
}

func (m *AppModel) handleSortByFamilyKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("SortByFamily key matched")
	sort.Slice(m.models, func(i, j int) bool {
		return m.models[i].Family < m.models[j].Family
	})
	m.refreshList()
	return m, nil
}

func (m *AppModel) handleRunModelKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("RunModel key matched")
	if item, ok := m.list.SelectedItem().(Model); ok {
		logging.InfoLogger.Printf("Running model: %s\n", item.Name)
		return m, runModel(item.Name)
	}
	return m, nil
}

func (m *AppModel) handleAltScreenKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("AltScreen key matched")
	m.altscreenActive = !m.altscreenActive
	cmd := tea.EnterAltScreen
	if !m.altscreenActive {
		cmd = tea.ExitAltScreen
	}
	return m, cmd
}

func (m *AppModel) handleClearScreenKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("ClearScreen key matched")
	m.refreshList()
	if m.inspecting {
		return m.clearScreen(), tea.ClearScreen
	}
	return m, nil
}

func (m *AppModel) handleUpdateModelKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("UpdateModel key matched")
	if item, ok := m.list.SelectedItem().(Model); ok {
		m.editing = true
		modelfilePath, err := copyModelfile(item.Name, item.Name)
		if err != nil {
			m.message = fmt.Sprintf("Error copying modelfile: %v", err)
			return m, nil
		}
		return m, openEditor(modelfilePath)
	}
	return m, nil
}

func (m *AppModel) handleLinkModelKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("LinkModel key matched")
	if item, ok := m.list.SelectedItem().(Model); ok {
		message, err := linkModel(item.Name, m.lmStudioModelsDir, m.noCleanup)
		if err != nil {
			m.message = fmt.Sprintf("Error linking model: %v", err)
		} else if message != "" {
			m.message = message
		} else {
			m.message = fmt.Sprintf("Model %s linked successfully", item.Name)
		}
	}
	return m, nil
}

func (m *AppModel) handleLinkAllModelsKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("LinkAllModels key matched")
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
	return m, nil
}

func (m *AppModel) handleCopyModelKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("CopyModel key matched")
	if item, ok := m.list.SelectedItem().(Model); ok {
		newName := promptForNewName(item.Name) // Pass the selected item as the model
		if newName == "" {
			m.message = "Error: name can't be empty"
		} else {
			copyModel(m, m.client, item.Name, newName)
			m.message = fmt.Sprintf("Model %s copied to %s", item.Name, newName)
		}
	}
	return m, nil
}

func (m *AppModel) handlePushModelKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("PushModel key matched")
	if item, ok := m.list.SelectedItem().(Model); ok {
		m.message = lipgloss.NewStyle().Foreground(lipgloss.Color("129")).Render(fmt.Sprintf("Pushing model: %s\n", item.Name))
		m.showProgress = true // Show progress bar
		return m, m.startPushModel(item.Name)
	}
	return m, nil
}

func (m *AppModel) handleInspectModelKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("InspectModel key matched")
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
	return m, nil
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
	if m.view == TopView {
		return m.topView()
	}

	if m.confirmDeletion {
		return m.confirmDeletionView()
	}

	if m.inspecting {
		return m.inspectModelView(m.inspectedModel)
	}
	if m.filtering() {
		return m.filterView()
	}

	view := m.list.View() + "\n" + m.message

	if m.showProgress {
		view += "\n" + m.progress.View()
	}

	return view
}

func (m *AppModel) confirmDeletionView() string {
	logging.DebugLogger.Println("Confirm deletion function")
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

	// Use getModelParams to get the model parameters and add them to the rows
	modelParams, err := getModelParams(model.Name)
	if err != nil {
		logging.ErrorLogger.Printf("Error getting model parameters: %v\n", err)
	}

	// similar to above but also includes the model parameters
	rows := []table.Row{
		{"Name", model.Name},
		{"ID", model.ID},
		{"Size (GB)", fmt.Sprintf("%.2f", model.Size)},
		{"Quantization Level", model.QuantizationLevel},
		{"Modified", model.Modified.Format("2006-01-02")},
		{"Family", model.Family},
	}

	// getModelParams returns a map of model parameters, so we need to iterate over the map and add the parameters to the rows
	for key, value := range modelParams {
		// rows = append(rows, []string{key, value[0]})
		// this works, but there may be multiple values for a single key, so we need to join them
		rows = append(rows, []string{key, strings.Join(value, ", ")})

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
	return "\n" + t.View() + "\nPress 'q' or `esc` to return to the main view."
}

func (m *AppModel) filterView() string {
	return m.list.View()
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
	m.editing = false
	m.showProgress = false
	m.table = table.New()
	return m
}

func (m *AppModel) filtering() bool {
	return m.list.FilterState() == list.Filtering
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

func (m *AppModel) topView() string {
	runningModels, err := showRunningModels(m.client)
	if err != nil {
		return fmt.Sprintf("Error showing running models: %v", err)
	}

	columns := []table.Column{
		{Title: "Name", Width: 40},
		{Title: "Size (GB)", Width: 10},
		{Title: "VRAM (GB)", Width: 10},
		{Title: "Until", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(runningModels),
		table.WithFocused(true),
		table.WithHeight(len(runningModels)+1),
	)

	// Set the table styles
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57"))
	t.SetStyles(s)

	// Render the table view
	return "\n" + t.View() + "\nPress 'q' or `esc` to return to the main view."
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Space, k.Delete, k.RunModel, k.LinkModel, k.LinkAllModels, k.CopyModel, k.PushModel}, // first column
		{k.SortByName, k.SortBySize, k.SortByModified, k.SortByQuant, k.SortByFamily},           // second column
		{k.Top, k.UpdateModel, k.InspectModel, k.Quit},                                          // third column
	}
}

// a function that can be called from the man app_model.go file with a hotkey to print the FullHelp as a string
func (m *AppModel) printFullHelp() string {
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("129")).Render("Help")
	for _, column := range m.keys.FullHelp() {
		for _, key := range column {
			help += fmt.Sprintf(" %s: %s\n", key.Help().Key, key.Help().Desc)
		}
	}
	return help
}
