// app_model.go not to be confused with an Ollama model - the app model is the tea.Model for the application.
package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
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
	"github.com/sammcj/gollama/styles"
)

const (
	MainView View = iota
	TopView
	HelpView
	ExternalEditorView
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
		if m.view == TopView || m.inspecting || m.view == HelpView || m.view == ExternalEditorView {
			if m.view == ExternalEditorView {
				m.resetExternalEditorState()
				return m, nil
			}
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
		if m.view == TopView || m.inspecting || m.view == HelpView || m.view == ExternalEditorView {
			if m.view == ExternalEditorView {
				m.resetExternalEditorState()
				return m, nil
			}
			m.view = MainView
			m.inspecting = false
			m.editing = false
			return m, nil
		} else {
			return m, nil
		}
	}

	// Handle external editor workflow
	if m.view == ExternalEditorView {
		switch msg.String() {
		case "s":
			// Save the modelfile
			message, err := finishExternalEdit(m.client, m.externalEditorModel, m.externalEditorFile)
			if err != nil {
				m.message = fmt.Sprintf("Error saving model: %v", err)
			} else {
				m.message = message
			}
			// Return to main view
			m.resetExternalEditorState()
			m.clearScreen()
			m.refreshList()
			return m, nil
		}
		// Let q and esc keys fall through to the general handlers above
		return m, nil
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
	case key.Matches(msg, m.keys.SortByParamSize):
		return m.handleSortByParamSizeKey()
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
	case key.Matches(msg, m.keys.CopyModel):
		return m.handleCopyModelKey()
	case key.Matches(msg, m.keys.PushModel):
		return m.handlePushModelKey()
	case key.Matches(msg, m.keys.PullModel):
		return m.handlePullModelKey()
	case key.Matches(msg, m.keys.PullKeepConfig):
		return m.handlePullKeepConfigKey()
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
		newModelName, cancelled := promptForNewName(item.Name)
		if cancelled {
			m.message = "Model creation cancelled"
			return m, nil
		}
		if strings.TrimSpace(newModelName) == "" {
			m.message = "Model name cannot be empty"
			return m, nil
		}
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
	m.refreshListWithSort(func(i, j int) bool {
		return m.models[i].Name < m.models[j].Name
	})
	return m, nil
}

func (m *AppModel) handleSortBySizeKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("SortBySize key matched")
	m.cfg.SortOrder = "size"
	m.refreshListWithSort(func(i, j int) bool {
		return m.models[i].Size > m.models[j].Size
	})
	return m, nil
}

func (m *AppModel) handleSortByModifiedKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("SortByModified key matched")
	m.cfg.SortOrder = "modified"
	m.refreshListWithSort(func(i, j int) bool {
		return m.models[i].Modified.After(m.models[j].Modified)
	})
	return m, nil
}

func (m *AppModel) handleSortByQuantKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("SortByQuant key matched")
	m.cfg.SortOrder = "quant"
	m.refreshListWithSort(func(i, j int) bool {
		return m.models[i].QuantizationLevel < m.models[j].QuantizationLevel
	})
	return m, nil
}

func (m *AppModel) handleSortByFamilyKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("SortByFamily key matched")
	m.cfg.SortOrder = "family"
	m.refreshListWithSort(func(i, j int) bool {
		return m.models[i].Family < m.models[j].Family
	})
	return m, nil
}

func (m *AppModel) handleSortByParamSizeKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("SortByParamSize key matched")
	m.cfg.SortOrder = "paramsize"

	// Helper function to extract numeric value from parameter size strings
	getParamSizeValue := func(paramSize string) float64 {
		if paramSize == "" {
			return 0
		}

		// Remove the "B" suffix if present
		numStr := paramSize
		if len(paramSize) > 0 && paramSize[len(paramSize)-1] == 'B' {
			numStr = paramSize[:len(paramSize)-1]
		}

		// Parse the numeric part
		size, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0
		}
		return size
	}

	// Sort models by parameter size (largest first)
	m.refreshListWithSort(func(i, j int) bool {
		sizeI := getParamSizeValue(m.models[i].ParameterSize)
		sizeJ := getParamSizeValue(m.models[j].ParameterSize)
		return sizeI > sizeJ
	})

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
	m.table = table.New()
	return m.ToggleTop()
}

func (m *AppModel) handleUpdateModelKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("UpdateModel key matched")
	if item, ok := m.list.SelectedItem().(Model); ok {
		editor := getEditor()

		logging.DebugLogger.Printf("Editor configured: %s", editor)
		logging.DebugLogger.Printf("Is external editor: %v", isExternalEditor(editor))

		if isExternalEditor(editor) {
			// Handle external editor workflow
			tempFilePath, err := startExternalEditor(m.client, item.Name)
			if err != nil {
				m.message = fmt.Sprintf("Error starting external editor: %v", err)
				return m, nil
			}

			// Set state for external editing
			m.externalEditing = true
			m.externalEditorFile = tempFilePath
			m.externalEditorModel = item.Name
			m.view = ExternalEditorView
			return m, nil
		} else {
			// Handle vim/terminal editor workflow (existing code)
			m.editing = true
			message, err := editModelfile(m.client, item.Name)
			if err != nil {
				m.message = fmt.Sprintf("Error updating model: %v", err)
			} else {
				m.message = message
				// Automatically return to main view after editing
				m.view = MainView
				m.editing = false
			}
			m.clearScreen()
			m.refreshList()
			return m, nil
		}
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
				return genericMsg{message: styles.ErrorStyle().Render(fmt.Sprintf("Error unloading model %s: %v", model.Name, err))}
			} else {
				unloadedModels = append(unloadedModels, styles.WarningStyle().Render(model.Name))
				logging.InfoLogger.Printf("Model %s unloaded\n", model.Name)
			}
		}
		return genericMsg{message: styles.SuccessStyle().Render(fmt.Sprintf("Models unloaded: %v", unloadedModels))}
	}
}

func (m *AppModel) handleCopyModelKey() (tea.Model, tea.Cmd) {
	defer func() {
		m.refreshList()
	}()
	logging.DebugLogger.Println("CopyModel key matched")
	if item, ok := m.list.SelectedItem().(Model); ok {
		newName, cancelled := promptForNewName(item.Name)
		if cancelled {
			m.message = styles.InfoStyle().Render("Copy cancelled")
		} else if newName == "" {
			m.message = styles.ErrorStyle().Render("Error: name can't be empty")
		} else if newName == item.Name {
			m.message = styles.ErrorStyle().Render("Error: new name must be different from the original name")
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
		m.message = styles.InfoStyle().Render(fmt.Sprintf("Pushing model: %s\n", item.Name))
		m.showProgress = true // Show progress bar
		return m, m.startPushModel(item.Name)
	}
	return m, nil
}

func (m *AppModel) handlePullModelKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("PullModel key matched")
	if item, ok := m.list.SelectedItem().(Model); ok {
		m.message = styles.InfoStyle().Render(fmt.Sprintf("Pulling model: %s\n", item.Name))
		m.pulling = true
		m.pullProgress = 0
		return m, m.startPullModel(item.Name)
	}
	return m, nil
}

// handlePullKeepConfigKey handles the shift+p key to pull a model while preserving user config
func (m *AppModel) handlePullKeepConfigKey() (tea.Model, tea.Cmd) {
	logging.DebugLogger.Println("PullKeepConfig key matched")
	if item, ok := m.list.SelectedItem().(Model); ok {
		m.message = styles.InfoStyle().Render(fmt.Sprintf("Pulling model & preserving config: %s\n", item.Name))
		m.pulling = true
		m.pullProgress = 0
		return m, m.startPullModelPreserveConfig(item.Name)
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
		newName, cancelled := promptForNewName(item.Name)
		if cancelled {
			m.message = styles.InfoStyle().Render("Rename cancelled")
		} else if newName == "" {
			m.message = styles.ErrorStyle().Render("Error: name can't be empty")
		} else if newName == item.Name {
			// User explicitly confirmed the same name, no action needed
			m.message = styles.InfoStyle().Render("Name unchanged")
		} else {
			err := renameModel(m, item.Name, newName)
			if err != nil {
				m.message = styles.ErrorStyle().Render(fmt.Sprintf("Error renaming model: %v", err))
			} else {
				m.message = styles.SuccessStyle().Render(fmt.Sprintf("Model %s renamed to %s", item.Name, newName))
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
	case ExternalEditorView:
		return m.externalEditorView()
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
			view += "\n\n" + styles.InfoStyle().Render(m.message)
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
		{Title: "Property", Width: 25},
		{Title: "Value", Width: 50},
	}

	// Get enhanced model information using the new API
	enhancedInfo, err := getEnhancedModelInfo(model.Name, m.client)
	if err != nil {
		logging.ErrorLogger.Printf("Error getting enhanced model info: %v\n", err)
		// Fall back to basic information if enhanced info fails
		enhancedInfo = &EnhancedModelInfo{}
	}

	// Use getModelParams to get the model parameters for backward compatibility
	modelParams, _, paramErr := getModelParams(model.Name, m.client)
	if paramErr != nil {
		logging.ErrorLogger.Printf("Error getting model parameters: %v\n", paramErr)
	}

	// Start with basic model information
	rows := []table.Row{
		{"Name", model.Name},
		{"ID", model.ID},
		{"Size (GB)", fmt.Sprintf("%.2f", model.Size)},
	}

	// Add enhanced information if available, only showing fields that have values
	if enhancedInfo.ParameterSize != "" {
		rows = append(rows, table.Row{"Parameter Size", enhancedInfo.ParameterSize})
	}

	if enhancedInfo.QuantizationLevel != "" {
		rows = append(rows, table.Row{"Quantisation Level", enhancedInfo.QuantizationLevel})
	} else if model.QuantizationLevel != "" {
		rows = append(rows, table.Row{"Quantisation Level", model.QuantizationLevel})
	}

	if enhancedInfo.Format != "" {
		rows = append(rows, table.Row{"Format", enhancedInfo.Format})
	}

	if enhancedInfo.Family != "" {
		rows = append(rows, table.Row{"Family", enhancedInfo.Family})
	} else if model.Family != "" {
		rows = append(rows, table.Row{"Family", model.Family})
	}

	if enhancedInfo.ContextLength > 0 {
		rows = append(rows, table.Row{"Context Length", fmt.Sprintf("%d", enhancedInfo.ContextLength)})
	}

	if enhancedInfo.EmbeddingLength > 0 {
		rows = append(rows, table.Row{"Embedding Length", fmt.Sprintf("%d", enhancedInfo.EmbeddingLength)})
	}

	if enhancedInfo.RopeDimensionCount > 0 {
		rows = append(rows, table.Row{"Rope Dimension Count", fmt.Sprintf("%d", enhancedInfo.RopeDimensionCount)})
	}

	if enhancedInfo.RopeFreqBase > 0 {
		rows = append(rows, table.Row{"Rope Freq Base", fmt.Sprintf("%.0f", enhancedInfo.RopeFreqBase)})
	}

	if enhancedInfo.VocabSize > 0 {
		rows = append(rows, table.Row{"Vocab Size", fmt.Sprintf("%d", enhancedInfo.VocabSize)})
	}

	if len(enhancedInfo.Capabilities) > 0 {
		rows = append(rows, table.Row{"Capabilities", strings.Join(enhancedInfo.Capabilities, ", ")})
	}

	// Add modified date
	rows = append(rows, table.Row{"Modified", model.Modified.Format("2006-01-02")})

	// Add any additional model parameters from the legacy function for completeness
	for key, value := range modelParams {
		// Skip parameters we've already shown in enhanced info
		if key != "parameter_size" && key != "quantization_level" && key != "format" && key != "family" {
			rows = append(rows, []string{key, value})
		}
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
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(styles.GetTheme().GetColour(styles.GetTheme().Colours.HeaderBorder))
	s.Selected = styles.SelectedItemStyle()
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

// refreshListWithSort updates the list view after sorting, preserving active filter state
func (m *AppModel) refreshListWithSort(sortFunc func(i, j int) bool) {
	// Always sort the master models list
	sort.Slice(m.models, sortFunc)

	// Check if a filter is currently active
	filterActive := m.list.FilterState() == list.Filtering || m.list.FilterState() == list.FilterApplied

	if filterActive {
		// Get the currently visible (filtered) items
		filteredItems := m.list.Items()

		// Early return if no filtered items
		if len(filteredItems) == 0 {
			m.refreshList()
			return
		}

		currentIndex := m.list.Index()

		// Create a map for O(1) lookup instead of nested loop
		modelIndexMap := make(map[string]int, len(m.models))
		for idx, model := range m.models {
			modelIndexMap[model.Name] = idx
		}

		// Sort the filtered items using the same sort function
		sort.Slice(filteredItems, func(i, j int) bool {
			modelI, okI := filteredItems[i].(Model)
			modelJ, okJ := filteredItems[j].(Model)
			if !okI || !okJ {
				return false
			}
			// Find the indices in the sorted m.models to determine order
			idxI, foundI := modelIndexMap[modelI.Name]
			idxJ, foundJ := modelIndexMap[modelJ.Name]
			if !foundI || !foundJ {
				return false
			}
			return idxI < idxJ
		})

		// Update the list with sorted filtered items (avoid nil assignment to prevent flicker)
		m.list.SetItems(filteredItems)

		// Restore cursor position (clamped to valid range)
		if currentIndex >= len(filteredItems) {
			currentIndex = len(filteredItems) - 1
		}
		if currentIndex >= 0 {
			m.list.Select(currentIndex)
		}
	} else {
		// No filter active, just refresh with all models
		m.refreshList()
	}
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
	headerStyle := styles.HeaderStyle()

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
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(styles.GetTheme().GetColour(styles.GetTheme().Colours.HeaderBorder))
	s.Selected = styles.SelectedItemStyle()
	t.SetStyles(s)

	// Render the table view
	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerStyle.Render(fmt.Sprintf("Connected to Ollama at: %s", m.cfg.OllamaAPIURL)),
		"\n"+t.View()+"\nPress 'q' or `esc` to return to the main view.",
	)
}

// FullHelp returns keybindings for the expanded help view. It's part of the key.Map interface.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Space, k.Delete, k.RunModel, k.CopyModel, k.PushModel},                                        // first column
		{k.SortByName, k.SortBySize, k.SortByModified, k.SortByQuant, k.SortByFamily, k.SortByParamSize}, // second column
		{k.Top, k.EditModel, k.InspectModel, k.Quit},                                                     // third column
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
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(styles.GetTheme().GetColour(styles.GetTheme().Colours.HeaderBorder))
	s.Selected = styles.SelectedItemStyle()
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

func (m *AppModel) externalEditorView() string {
	headerStyle := styles.HeaderStyle()
	infoStyle := styles.InfoStyle()
	helpStyle := styles.HelpTextStyle()
	warningStyle := styles.WarningStyle()

	// Create a styled border box around the content
	title := headerStyle.Render("🎨 External Editor Mode")

	modelInfo := fmt.Sprintf("Editing modelfile for: %s", m.externalEditorModel)
	editorInfo := fmt.Sprintf("Temporary file: %s", m.externalEditorFile)

	instructions := []string{
		warningStyle.Render("📝 Edit the Modelfile in your configured editor."),
		"",
		helpStyle.Render("💾 Press 's' to save after making your changes"),
		helpStyle.Render("🚪 Press any other key to return to the application"),
	}

	content := []string{
		"",
		title,
		"",
		infoStyle.Render("🎯 " + modelInfo),
		infoStyle.Render("📁 " + editorInfo),
		"",
	}

	for _, instruction := range instructions {
		content = append(content, instruction)
	}

	content = append(content, "")

	return strings.Join(content, "\n")
}

// resetExternalEditorState resets all external editor state to initial values
func (m *AppModel) resetExternalEditorState() {
	m.externalEditing = false
	m.externalEditorFile = ""
	m.externalEditorModel = ""
	m.view = MainView
}
