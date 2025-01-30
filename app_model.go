// app_model.go not to be confused with an Ollama model - the app model is the tea.Model for the application.
package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ollama/ollama/api"

	"github.com/sammcj/gollama/logging"
)

const (
	MainView View = iota
	TopView
	HelpView
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

	if m.pulling {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if m.newModelPull {
				switch msg.Type {
				case tea.KeyEnter:
					m.newModelPull = false
					m.pullProgress = 0.01 // Start progress immediately
					return m, tea.Batch(
						m.startPullModel(m.pullInput.Value()),
						m.updateProgressCmd(),
					)
				case tea.KeyCtrlC, tea.KeyEsc:
					m.pulling = false
					m.newModelPull = false
					m.pullInput.Reset()
					return m, nil
				}
				var cmd tea.Cmd
				m.pullInput, cmd = m.pullInput.Update(msg)
				return m, cmd
			} else {
				if msg.Type == tea.KeyCtrlC {
					m.pulling = false
					m.pullProgress = 0
					return m, nil
				}
			}
			if m.comparingModelfile {
				switch msg.String() {
				case "q", "esc":
					m.comparingModelfile = false
					return m, nil
				}
			}
			switch {
			case key.Matches(msg, m.keys.CompareModelfile):
				return m.handleCompareModelfile()
			}
		case pullSuccessMsg:
			return m.handlePullSuccessMsg(msg)
		case pullErrorMsg:
			return m.handlePullErrorMsg(msg)
		case progressMsg:
			if m.pullProgress < 1.0 {
				return m, tea.Batch(
					m.updateProgressCmd(),
					func() tea.Msg {
						return progressMsg{
							modelName: msg.modelName,
							progress:  m.pullProgress,
						}
					},
				)
			}
			return m, nil
		}
	}
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
	case genericMsg:
		return m.handleGenericMsg(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(m.width, m.height)
		return m, nil
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
	case "ctrl+c":
		if m.pulling {
			m.pulling = false
			m.pullProgress = 0
			m.pullInput.Reset()
			return m, nil
		}
		if m.editing {
			m.editing = false
			return m, nil
		}
		return m, tea.Quit
	case "q":
		if m.list.FilterState() == list.FilterApplied {
			logging.DebugLogger.Println("Clearing filter with 'q' key")
			m.list.ResetFilter()
			return m, nil
		}
		if m.view == TopView || m.inspecting || m.view == HelpView {
			m.view = MainView
			m.inspecting = false
			m.editing = false
			return m, nil
		} else {
			return m, tea.Quit
		}
	case "esc":
		if m.list.FilterState() == list.FilterApplied {
			logging.DebugLogger.Println("Clearing filter with 'esc' key")
			m.list.ResetFilter()
			return m, nil
		}
		if m.view == TopView || m.inspecting || m.view == HelpView {
			m.view = MainView
			m.inspecting = false
			m.editing = false
			return m, nil
		} else {
			return m, nil
		}
	}

	if m.confirmDeletion {
		switch {
		case key.Matches(msg, m.keys.ConfirmYes):
			logging.DebugLogger.Println("ConfirmYes key matched")
			for _, selectedModel := range m.selectedModels {
				logging.InfoLogger.Printf("Attempting to delete model: %s\n", selectedModel.Name)
				err := deleteModel(m.client, selectedModel.Name)
				if err != nil {
					logging.ErrorLogger.Println("Error deleting model:", err)
				}
			}
			m.models = removeModels(m.models, m.selectedModels)
			m.refreshList()
			m.confirmDeletion = false
			m.selectedModels = nil
		case key.Matches(msg, m.keys.ConfirmNo):
			logging.DebugLogger.Println("ConfirmNo key matched")
			logging.InfoLogger.Println("Deletion cancelled by user")
			m.confirmDeletion = false
			m.selectedModels = nil
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
	case key.Matches(msg, m.keys.EditModel):
		return m.handleUpdateModelKey()
	case key.Matches(msg, m.keys.UnloadModels):
		return m.handleUnloadModelsKey()
	case key.Matches(msg, m.keys.LinkModel):
		return m.handleLinkModelKey()
	case key.Matches(msg, m.keys.LinkAllModels):
		return m.handleLinkAllModelsKey()
	case key.Matches(msg, m.keys.CopyModel):
		return m.handleCopyModelKey()
	case key.Matches(msg, m.keys.PushModel):
		return m.handlePushModelKey()
	case key.Matches(msg, m.keys.PullModel):
		return m.handlePullModelKey()
	case key.Matches(msg, m.keys.RenameModel):
		return m.handleRenameModelKey()
	case key.Matches(msg, m.keys.PullNewModel):
		return m.handlePullNewModelKey()
	case key.Matches(msg, m.keys.InspectModel):
		return m.handleInspectModelKey()
	case key.Matches(msg, m.keys.Top):
		return m.handleTopKey()
	case key.Matches(msg, m.keys.Help):
		return m.handleHelpKey()
  case key.Matches(msg, m.keys.CompareModelfile):
    return m.handleCompareModelfile()
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

		// Update both the filtered and unfiltered lists
		filteredItems := m.list.Items()
		for i, listItem := range filteredItems {
			if model, ok := listItem.(Model); ok && model.Name == item.Name {
				filteredItems[i] = item
			}
		}

		// Always update the main model list
		for i, model := range m.models {
			if model.Name == item.Name {
				m.models[i] = item
			}
		}

		// Update the items in the list
		m.list.SetItems(filteredItems)

		// If filtering is active, force a refresh of the view
		if m.list.FilterState() == list.Filtering || m.list.FilterState() == list.FilterApplied {
			// Store current cursor position
			currentIndex := m.list.Index()

			// Force a view refresh by temporarily clearing and reapplying items
			tempItems := m.list.Items()
			m.list.SetItems(nil)
			m.list.SetItems(tempItems)

			// Restore cursor position
			m.list.Select(currentIndex)
		}

		logging.DebugLogger.Printf("Toggled selection for model: %s (after: %v)\n", item.Name, item.Selected)
	}
	return m, nil
}

// A function that returns a tea message stating "function not available on remote hosts" if remoteHost == true
func (m *AppModel) isRemoteHost() string {
	if !strings.Contains(m.cfg.OllamaAPIURL, "localhost") && !strings.Contains(m.cfg.OllamaAPIURL, "127.0.0.1") {
		return "Function not available on remote hosts"
	}
	msg := ""
	return msg
}

func (m *AppModel) handleRunFinishedMessage(msg runFinishedMessage) (tea.Model, tea.Cmd) {
	logging.DebugLogger.Printf("Run finished message: %v\n", msg)
	if msg.err != nil {
		logging.ErrorLogger.Printf("Error running model: %v\n", msg.err)
		m.message = fmt.Sprintf("Error running model: %v\n", msg.err)
	}
	return m, nil
}

// TODO: Refactor: Look into making generic handler functions

func (m *AppModel) handleProgressMsg(msg progressMsg) (tea.Model, tea.Cmd) {
	return m, tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return progressMsg{modelName: msg.modelName}
	})
}

func (m *AppModel) handleHelpKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("Help key matched")
	if m.view == HelpView {
		m.view = MainView
		m.message = ""
		return m, tea.Batch(
			tea.ClearScreen,
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)
	}
	m.view = HelpView
	return m, nil
}

func (m *AppModel) handleEditorFinishedMsg(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.message = fmt.Sprintf("Error editing modelfile: %v", msg.err)
		return m, nil
	}
	if item, ok := m.list.SelectedItem().(Model); ok {
		newModelName := promptForNewName(item.Name)
		modelfilePath := fmt.Sprintf("Modelfile-%s", strings.ReplaceAll(newModelName, " ", "_"))
		err := createModelFromModelfile(newModelName, modelfilePath, m.client)
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

func (m *AppModel) handlePullSuccessMsg(msg pullSuccessMsg) (tea.Model, tea.Cmd) {
	m.pulling = false
	m.newModelPull = false
	m.pullProgress = 0
	m.message = fmt.Sprintf("Successfully pulled model: %s", msg.modelName)
	return m, tea.Batch(
		m.refreshModelsAfterPull(),
		func() tea.Msg {
			// This will force a refresh of the main view
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		},
	)
}

func (m *AppModel) handlePullErrorMsg(msg pullErrorMsg) (tea.Model, tea.Cmd) {
	m.pulling = false
	m.pullProgress = 0
	m.message = fmt.Sprintf("Error pulling model: %v", msg.err)
	return m, func() tea.Msg {
		// This will force a refresh of the main view
		return tea.WindowSizeMsg{Width: m.width, Height: m.height}
	}
}

func (m *AppModel) updateProgressCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return progressMsg{
			modelName: m.pullInput.Value(),
			progress:  m.pullProgress,
		}
	})
}

func (m *AppModel) handleGenericMsg(msg genericMsg) (tea.Model, tea.Cmd) {
	if msg.message != "" {
		m.message = msg.message
	}
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
		m.selectedModels = selectedModels
		logging.InfoLogger.Printf("Selected models for deletion: %+v\n", m.selectedModels)
		m.confirmDeletion = true
	} else if item, ok := m.list.SelectedItem().(Model); ok {
		m.selectedModels = []Model{item}
		logging.InfoLogger.Printf("Selected model for deletion: %+v\n", m.selectedModels)
		m.confirmDeletion = true
	}
	return m, nil
}

func (m *AppModel) handleSortByNameKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("SortByName key matched")
	m.cfg.SortOrder = "name"
	sort.Slice(m.models, func(i, j int) bool {
		return m.models[i].Name < m.models[j].Name
	})
	m.refreshList()
	return m, nil
}

func (m *AppModel) handleSortBySizeKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("SortBySize key matched")
	m.cfg.SortOrder = "size"
	sort.Slice(m.models, func(i, j int) bool {
		return m.models[i].Size > m.models[j].Size
	})
	m.refreshList()
	return m, nil
}

func (m *AppModel) handleSortByModifiedKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("SortByModified key matched")
	m.cfg.SortOrder = "modified"
	sort.Slice(m.models, func(i, j int) bool {
		return m.models[i].Modified.After(m.models[j].Modified)
	})
	m.refreshList()
	return m, nil
}

func (m *AppModel) handleSortByQuantKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("SortByQuant key matched")
	m.cfg.SortOrder = "quant"
	sort.Slice(m.models, func(i, j int) bool {
		return m.models[i].QuantizationLevel < m.models[j].QuantizationLevel
	})
	m.refreshList()
	return m, nil
}

func (m *AppModel) handleSortByFamilyKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("SortByFamily key matched")
	m.cfg.SortOrder = "family"
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
		return m, runModel(item.Name, m.cfg)
	}
	return m, nil
}

func (m *AppModel) handleAltScreenKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("AltScreen key matched")
	m.altScreenActive = !m.altScreenActive
	cmd := tea.EnterAltScreen
	if !m.altScreenActive {
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

// top view handler
func (m *AppModel) handleTopKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("Top key matched")
	m.view = TopView
	return m.ToggleTop()
}

func (m *AppModel) handleUpdateModelKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("UpdateModel key matched")
	if item, ok := m.list.SelectedItem().(Model); ok {
		m.editing = true
		message, err := editModelfile(m.client, item.Name)
		if err != nil {
			m.message = fmt.Sprintf("Error updating model: %v", err)
		} else {
			m.message = message
		}
		m.clearScreen()
		m.refreshList()
		return m, nil
	}
	m.refreshList()
	return m, nil
}

func (m *AppModel) handleUnloadModelsKey() (tea.Model, tea.Cmd) {
	return m, func() tea.Msg {
		// get any loaded models
		ctx := context.Background()
		loadedModels, err := m.client.ListRunning(ctx)
		if err != nil {
			return genericMsg{message: fmt.Sprintf("Error listing running models: %v", err)}
		}

		// unload the models
		var unloadedModels []string
		for _, model := range loadedModels.Models {
			_, err := unloadModel(m.client, model.Name)
			if err != nil {
				return genericMsg{message: lipgloss.NewStyle().Foreground(lipgloss.Color("#8B0000")).Render(fmt.Sprintf("Error unloading model %s: %v", model.Name, err))}
			} else {
				unloadedModels = append(unloadedModels, lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB6C1")).Render(model.Name))
				logging.InfoLogger.Printf("Model %s unloaded\n", model.Name)
			}
		}
		return genericMsg{message: lipgloss.NewStyle().Foreground(lipgloss.Color("#EE82EE")).Render(fmt.Sprintf("Models unloaded: %v", unloadedModels))}
	}
}

func (m *AppModel) handleLinkModelKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("LinkModel key matched")
	// Function not available on remote Ollama servers
	if msg := m.isRemoteHost(); msg != "" {
		m.message = msg
		return m, nil
	}
	if item, ok := m.list.SelectedItem().(Model); ok {
		message, err := linkModel(item.Name, m.lmStudioModelsDir, m.noCleanup, false, m.client)
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
	// Function not available on remote Ollama servers
	if msg := m.isRemoteHost(); msg != "" {
		m.message = msg
		return m, nil
	}
	var messages []string
	for _, model := range m.models {
		message, err := linkModel(model.Name, m.lmStudioModelsDir, m.noCleanup, false, m.client)
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
	defer func() {
		m.refreshList()
	}()
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

func (m *AppModel) handlePullModelKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("PullModel key matched")
	if item, ok := m.list.SelectedItem().(Model); ok {
		m.message = lipgloss.NewStyle().Foreground(lipgloss.Color("129")).Render(fmt.Sprintf("Pulling model: %s\n", item.Name))
		m.pulling = true
		m.pullProgress = 0
		return m, m.startPullModel(item.Name)
	}
	return m, nil
}

func (m *AppModel) handlePullNewModelKey() (tea.Model, tea.Cmd) {
	m.pullInput = textinput.New()
	m.pullInput.Placeholder = "Enter model name (e.g. llama3:8b-instruct)"
	m.pullInput.Focus()
	m.pulling = true
	m.newModelPull = true
	return m, textinput.Blink
}

func (m *AppModel) updatePullInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.pullInput, cmd = m.pullInput.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return m, m.startPullModel(m.pullInput.Value())
		case tea.KeyEsc:
			m.pulling = false
			m.pullInput.Reset()
			return m, nil
		}
	}

	return m, cmd
}

func (m *AppModel) startPullNewModel(modelName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		req := &api.PullRequest{Name: modelName}
		err := m.client.Pull(ctx, req, func(resp api.ProgressResponse) error {
			m.pullProgress = float64(resp.Completed) / float64(resp.Total)
			return nil
		})
		if err != nil {
			return pullErrorMsg{err}
		}
		return pullSuccessMsg{modelName}
	}
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

func (m *AppModel) handleRenameModelKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("RenameModel key matched")
	if item, ok := m.list.SelectedItem().(Model); ok {
		newName := promptForNewName(item.Name)
		if newName == "" {
			m.message = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B0000")).Render("Error: name can't be empty")
		} else {
			err := renameModel(m, item.Name, newName)
			if err != nil {
				m.message = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B0000")).Render(fmt.Sprintf("Error renaming model: %v", err))
			} else {
				m.message = lipgloss.NewStyle().Foreground(lipgloss.Color("#EE82EE")).Render(fmt.Sprintf("Model %s renamed to %s", item.Name, newName))
			}
		}
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
	return m, tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		running, err := showRunningModels(m.client)
		if err != nil {
			return fmt.Sprintf("Error showing running models: %v", err)
		}
		return running
	})
}

func (m *AppModel) View() string {
	switch m.view {
	case TopView:
		return m.topView()
	case HelpView:
		return m.printFullHelp()
	default:
		if m.confirmDeletion {
			return m.confirmDeletionView()
		}
		if m.inspecting {
			return m.inspectModelView(m.inspectedModel)
		}
		if m.filtering() {
			return m.filterView()
		}
		if m.comparingModelfile {
			return m.modelfileDiffView()
		}

		if m.pulling {
			if m.newModelPull && m.pullProgress == 0 {
				return fmt.Sprintf(
					"%s\n%s",
					"Enter model name to pull:",
					m.pullInput.View(),
				)
			}
			return fmt.Sprintf(
				"Pulling model: %.0f%%\n%s\n%s",
				m.pullProgress*100,
				m.progress.ViewAs(m.pullProgress),
				"Press Ctrl+C to cancel - Note there is currently bug where you might need to hold a key (e.g. arrow key) to refresh the progress bar",
			)
		}

		view := m.list.View()

		if m.message != "" && m.view != HelpView {
			view += "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(m.message)
		}

		if m.showProgress {
			view += "\n" + m.progress.View()
		}

		return view
	}
}

func (m *AppModel) confirmDeletionView() string {
	defer func() {
		m.refreshList()
	}()
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
	modelParams, _, err := getModelParams(model.Name, m.client)
	if err != nil {
		logging.ErrorLogger.Printf("Error getting model parameters: %v\n", err)
	}

	// similar to above but also includes the model parameters
	rows := []table.Row{
		{"Name", model.Name},
		{"ID", model.ID},
		{"Size (GB)", fmt.Sprintf("%.2f", model.Size)},
		{"quantisation Level", model.QuantizationLevel},
		{"Modified", model.Modified.Format("2006-01-02")},
		{"Family", model.Family},
	}

	// getModelParams returns a map of model parameters, so we need to iterate over the map and add the parameters to the rows
	for key, value := range modelParams {
		rows = append(rows, []string{key, value})
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
	t.Focus()

	// Render the table view
	return "\n" + t.View() + "\nPress 'q' or `esc` to return to the main view."
}

func (m *AppModel) filterView() string {
	m.list.FilterInput.Focus()
	return m.list.View()
}

func (m *AppModel) selectedModelNames() []string {
	var names []string
	for _, model := range m.selectedModels {
		names = append(names, model.Name)
	}
	return names
}

// refreshList updates the list view with the current models
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
	m.refreshList()
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

// FullHelp returns keybindings for the expanded help view. It's part of the key.Map interface.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Space, k.Delete, k.RunModel, k.LinkModel, k.LinkAllModels, k.CopyModel, k.PushModel}, // first column
		{k.SortByName, k.SortBySize, k.SortByModified, k.SortByQuant, k.SortByFamily},           // second column
		{k.Top, k.EditModel, k.InspectModel, k.Quit},                                            // third column
	}
}

// a function that can be called from the man app_model.go file with a hotkey to print the FullHelp as a string
func (m *AppModel) printFullHelp() string {
	if m.view != HelpView {
		m.message = ""
		return m.message
	}

	// Create a new table and use FullHelp() to populate it
	columns := []table.Column{
		{Title: "Key", Width: 10},
		{Title: "Description", Width: 50},
	}

	rows := []table.Row{}
	for _, column := range m.keys.FullHelp() {
		for _, key := range column {
			rows = append(rows, table.Row{key.Help().Key, key.Help().Desc})
		}
	}

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

// Add this method to refresh the model list after pulling:
func (m *AppModel) refreshModelsAfterPull() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := m.client.List(ctx)
		if err != nil {
			return pullErrorMsg{err}
		}
		m.models = parseAPIResponse(resp)
		m.refreshList()
		return nil
	}
}
