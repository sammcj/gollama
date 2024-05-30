package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ollama/ollama/api"
	"golang.org/x/term"
)

var (
	logLevel    = "debug"
	debugLogger *log.Logger
	infoLogger  *log.Logger
	errorLogger *log.Logger
)

func init() {
	f, err := os.OpenFile("gollama.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		os.Exit(1)
	}
	if logLevel == "debug" {
		log.SetOutput(f)
	} else {
		log.SetOutput(io.Discard)
	}
	debugLogger = log.New(f, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	infoLogger = log.New(f, "INFO: ", log.Ldate|log.Ltime)
	errorLogger = log.New(f, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

type Model struct {
	Name              string
	ID                string
	Size              float64
	QuantizationLevel string
	Modified          time.Time
	Selected          bool
}

type KeyMap struct {
	Space          key.Binding
	Delete         key.Binding
	SortByName     key.Binding
	SortBySize     key.Binding
	SortByModified key.Binding
	RunModel       key.Binding
	ConfirmYes     key.Binding
	ConfirmNo      key.Binding
}

func (m Model) Title() string { return m.Name }

func (m Model) Description() string {
	return fmt.Sprintf("ID: %s, Size: %.2f GB, Quant: %s, Modified: %s", m.ID, m.Size, m.QuantizationLevel, m.Modified.Format("2006-01-02"))
}
func (m Model) FilterValue() string { return m.Name }

func NewKeyMap() *KeyMap {
	return &KeyMap{
		Space: key.NewBinding(
			key.WithKeys("space"),
			key.WithHelp("space", "select model"),
		),
		Delete: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "delete selected"),
		),
		SortByName: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sort by name"),
		),
		SortBySize: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "sort by size"),
		),
		SortByModified: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "sort by modified date"),
		),
		RunModel: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "run model"),
		),
		ConfirmYes: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "confirm deletion"),
		),
		ConfirmNo: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "cancel deletion"),
		),
	}
}

type AppModel struct {
	client              *api.Client
	list                list.Model
	keys                *KeyMap
	models              []Model
	width               int
	height              int
	confirmDeletion     bool
	selectedForDeletion []Model
}

func main() {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		errorLogger.Println("Error creating API client:", err)
		return
	}

	ctx := context.Background()
	resp, err := client.List(ctx)
	if err != nil {
		errorLogger.Println("Error fetching models:", err)
		return
	}

	infoLogger.Println("Fetched models from API")
	models := parseAPIResponse(resp)

	// Group models by ID and normalize sizes to GB
	modelMap := make(map[string][]Model)
	for _, model := range models {
		model.Size = normalizeSize(model.Size)
		modelMap[model.ID] = append(modelMap[model.ID], model)
	}

	// Flatten the map into a slice
	groupedModels := make([]Model, 0)

	for _, group := range modelMap {
		groupedModels = append(groupedModels, group...)
	}

	// default to sorting by date modified (newest first)
	sort.Slice(groupedModels, func(i, j int) bool {
		return groupedModels[i].Modified.After(groupedModels[j].Modified)
	})

	items := make([]list.Item, len(groupedModels))
	for i, model := range groupedModels {
		items[i] = model
	}

	keys := NewKeyMap()
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 80, 24 // default size if terminal size can't be determined
	}
	app := AppModel{client: client, keys: keys, models: groupedModels, width: width, height: height}
	l := list.New(items, itemDelegate{appModel: &app}, width, height-5)

	l.Title = "Ollama Models"
	l.InfiniteScrolling = true
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)

	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			keys.Space,
			keys.Delete,
			keys.SortByName,
			keys.SortBySize,
			keys.SortByModified,
			keys.RunModel,
			keys.ConfirmYes,
			keys.ConfirmNo,
		}
	}

	app.list = l
	if _, err := tea.NewProgram(&app, tea.WithAltScreen()).Run(); err != nil {
		errorLogger.Printf("Error: %v", err)
		l.ResetSelected()
	}
}

func (m *AppModel) Init() tea.Cmd {
	return nil
}
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Space):
			if item, ok := m.list.SelectedItem().(Model); ok {
				debugLogger.Printf("Toggling selection for model: %s (before: %v)\n", item.Name, item.Selected)
				item.Selected = !item.Selected
				m.models[m.list.Index()] = item
				m.list.SetItem(m.list.Index(), item)
				debugLogger.Printf("Toggled selection for model: %s (after: %v)\n", item.Name, item.Selected)
			}
		case key.Matches(msg, m.keys.Delete):
			infoLogger.Println("Delete key pressed")
			m.selectedForDeletion = getSelectedModels(m.models)
			infoLogger.Printf("Selected models for deletion: %+v\n", m.selectedForDeletion)
			m.confirmDeletion = true
		case key.Matches(msg, m.keys.ConfirmYes):
			if m.confirmDeletion {
				for _, selectedModel := range m.selectedForDeletion {
					infoLogger.Printf("Attempting to delete model: %s\n", selectedModel.Name)
					err := deleteModel(m.client, selectedModel.Name)
					if err != nil {
						errorLogger.Println("Error deleting model:", err)
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
				infoLogger.Println("Deletion cancelled by user")
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
		case key.Matches(msg, m.keys.RunModel):
			if item, ok := m.list.SelectedItem().(Model); ok {
				runModel(item.Name)
			}
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
		return lipgloss.NewStyle().Bold(true).Render("Are you sure you want to delete the selected models? (y/N): ")
	}

	nameWidth, sizeWidth, quantWidth, modifiedWidth, idWidth := calculateColumnWidths(m.width)

	header := lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
			nameWidth, "Name",
			sizeWidth, "Size",
			quantWidth, "Quant",
			modifiedWidth, "Modified",
			idWidth, "ID",
		),
	)
	return header + "\n" + m.list.View()
}

func parseAPIResponse(resp *api.ListResponse) []Model {
	models := make([]Model, len(resp.Models))
	for i, modelResp := range resp.Models {
		models[i] = Model{
			Name:              modelResp.Name,
			ID:                truncate(modelResp.Digest, 7),                  // Truncate the ID
			Size:              float64(modelResp.Size) / (1024 * 1024 * 1024), // Convert bytes to GB
			QuantizationLevel: modelResp.Details.QuantizationLevel,
			Modified:          modelResp.ModifiedAt,
			Selected:          false,
		}
	}
	return models
}

func getSelectedModels(models []Model) []Model {
	selectedModels := make([]Model, 0)
	for _, model := range models {
		if model.Selected {
			debugLogger.Printf("Model selected for deletion: %s\n", model.Name)
			selectedModels = append(selectedModels, model)
		}
	}
	return selectedModels
}

func (m *AppModel) refreshList() {
	items := make([]list.Item, len(m.models))
	for i, model := range m.models {
		items[i] = model
	}
	m.list.SetItems(items)
}

func normalizeSize(size float64) float64 {
	return size // Sizes are already in GB in the API response
}

func calculateColumnWidths(totalWidth int) (nameWidth, sizeWidth, quantWidth, modifiedWidth, idWidth int) {
	// Calculate column widths
	nameWidth = int(0.6 * float64(totalWidth))
	sizeWidth = int(0.05 * float64(totalWidth))
	quantWidth = int(0.05 * float64(totalWidth))
	modifiedWidth = int(0.05 * float64(totalWidth))
	idWidth = int(0.05 * float64(totalWidth))

	// Set the absolute minimum width for each column
	if nameWidth < 10 {
		nameWidth = 10
	}
	if sizeWidth < 10 {
		sizeWidth = 10
	}
	if quantWidth < 10 {
		quantWidth = 10
	}
	if modifiedWidth < 10 {
		modifiedWidth = 10
	}
	if idWidth < 10 {
		idWidth = 10
	}

	// If the total width is less than the sum of the minimum column widths, adjust the name column width
	if totalWidth < nameWidth+sizeWidth+quantWidth+modifiedWidth+idWidth {
		nameWidth = totalWidth - sizeWidth - quantWidth - modifiedWidth - idWidth + 30
	}

	return
}

// Custom delegate to handle the rendering of the list items without extra spacing
type itemDelegate struct {
	appModel *AppModel
}

func (d itemDelegate) Height() int  { return 1 }
func (d itemDelegate) Spacing() int { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case " ": // space key pressed
			i, ok := m.SelectedItem().(Model)
			if ok {
				debugLogger.Printf("Delegate toggling selection for model: %s (before: %v)\n", i.Name, i.Selected)
				i.Selected = !i.Selected
				m.SetItem(m.Index(), i)
				debugLogger.Printf("Delegate toggled selection for model: %s (after: %v)\n", i.Name, i.Selected)

				// Update the main model list
				d.appModel.models[m.Index()] = i
				debugLogger.Printf("Updated main model list for model: %s (after: %v)\n", i.Name, i.Selected)
			}
		}
	}
	return nil
}

// Render the list items
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	model, ok := item.(Model)
	if !ok {
		return
	}
	var nameStyle, idStyle, sizeStyle, quantStyle, modifiedStyle lipgloss.Style
	if index == m.Index() {
		// Highlight the selected item
		nameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).BorderLeft(true).BorderStyle(lipgloss.OuterHalfBlockBorder())
		sizeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("120"))
		quantStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("150"))
		modifiedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("115"))
		idStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("225"))
	} else {
		nameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("254")).Faint(true)
		idStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("254")).Faint(true)
		sizeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("254")).Faint(true)
		quantStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("254")).Faint(true)
		modifiedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("254")).Faint(true)
	}

	if model.Selected {
		nameStyle = nameStyle.Background(lipgloss.Color("236")).Bold(true).Italic(true)
		idStyle = idStyle.Background(lipgloss.Color("236")).Bold(true).Italic(true)
		sizeStyle = sizeStyle.Background(lipgloss.Color("236")).Bold(true).Italic(true)
		quantStyle = quantStyle.Background(lipgloss.Color("236")).Bold(true).Italic(true)
		modifiedStyle = modifiedStyle.Background(lipgloss.Color("236")).Bold(true).Italic(true)
	}

	nameWidth, sizeWidth, quantWidth, modifiedWidth, idWidth := calculateColumnWidths(m.Width())

	// Ensure the text fits within the terminal width
	name := wrapText(nameStyle.Width(nameWidth).Render(truncate(model.Name, nameWidth)), nameWidth)
	size := wrapText(sizeStyle.Width(sizeWidth).Render(fmt.Sprintf("%.2fGB", model.Size)), sizeWidth)
	quant := wrapText(quantStyle.Width(quantWidth).Render(truncate(model.QuantizationLevel, quantWidth)), quantWidth)
	modified := wrapText(modifiedStyle.Width(modifiedWidth).Render(model.Modified.Format("2006-01-02")), modifiedWidth)
	id := wrapText(idStyle.Width(idWidth).Render(model.ID), idWidth) // Render the truncated ID directly

	fmt.Fprint(w, lipgloss.JoinHorizontal(lipgloss.Top, name, size, quant, modified, id))
}

// truncate ensures the string fits within the specified width
func truncate(text string, width int) string {
	if len(text) > width {
		return text[:width]
	}
	return text
}

// wrapText ensures the text wraps to the next line if it exceeds the column width
func wrapText(text string, width int) string {
	var wrapped string
	for len(text) > width {
		wrapped += text[:width]
		text = text[width:] + " "
	}
	wrapped += text
	return wrapped
}

func runModel(modelName string) {
	// Save the current terminal state
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		errorLogger.Printf("Error saving terminal state: %v\n", err)
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Clear the terminal screen
	fmt.Print("\033[H\033[2J")

	// Run the Ollama model
	cmd := exec.Command("ollama", "run", modelName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		errorLogger.Printf("Error running model: %v\n", err)
	} else {
		infoLogger.Printf("Successfully ran model: %s\n", modelName)
	}

	// Restore the terminal state
	if err := term.Restore(int(os.Stdin.Fd()), oldState); err != nil {
		errorLogger.Printf("Error restoring terminal state: %v\n", err)
	}

	// Clear the terminal screen again to refresh the application view
	fmt.Print("\033[H\033[2J")
}

func deleteModel(client *api.Client, name string) error {
	ctx := context.Background()
	req := &api.DeleteRequest{Name: name}
	debugLogger.Printf("Attempting to delete model: %s\n", name)

	// Log the request details
	debugLogger.Printf("Delete request: %+v\n", req)

	err := client.Delete(ctx, req)
	if err != nil {
		// Print a detailed error message to the console
		errorLogger.Printf("Error deleting model %s: %v\n", name, err)
		// Return an error so that it can be handled by the calling function
		return fmt.Errorf("error deleting model %s: %v", name, err)
	}

	// If we reach this point, the model was deleted successfully
	infoLogger.Printf("Successfully deleted model: %s\n", name)
	return nil
}

func removeModels(models []Model, selectedModels []Model) []Model {
	result := make([]Model, 0)
	for _, model := range models {
		found := false
		for _, selectedModel := range selectedModels {
			if model.Name == selectedModel.Name {
				found = true
				break
			}
		}
		if !found {
			result = append(result, model)
		}
	}
	return result
}
