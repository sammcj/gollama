// Package modeledit provides functionality to edit Ollama model parameters and templates
package modeledit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

// Model represents the state of the modeledit UI
type Model struct {
	modelName    string
	manifestPath string
	paramPath    string
	templatePath string
	params       map[string]interface{}
	template     string
	list         list.Model
	textInput    textinput.Model
	currentParam string
	mode         string
	confirmSave  bool
	logger       zerolog.Logger
	statusMsg    string
}

// Initialize sets up the modeledit Model
func Initialize(modelName string) (*Model, error) {
	m := &Model{
		modelName: modelName,
		params:    make(map[string]interface{}),
		mode:      "list",
		logger:    log.With().Str("package", "modeledit").Logger(),
	}

	// Set up paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Try to find the manifest file
	m.manifestPath, err = findManifestPath(homeDir, modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to find manifest file: %w", err)
	}

	// Read and parse manifest
	manifestData, err := os.ReadFile(m.manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %w", err)
	}

	// Find param and template paths
	layers := manifest["layers"].([]interface{})
	for _, layer := range layers {
		layerMap := layer.(map[string]interface{})
		mediaType := layerMap["mediaType"].(string)
		digest := layerMap["digest"].(string)
		digest = strings.Replace(digest, ":", "-", 1)

		blobPath := filepath.Join(homeDir, ".ollama", "models", "blobs", digest)

		switch mediaType {
		case "application/vnd.ollama.image.params":
			m.paramPath = blobPath
		case "application/vnd.ollama.image.template":
			m.templatePath = blobPath
		}
	}

	// Read and parse params
	paramData, err := os.ReadFile(m.paramPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read param file: %w", err)
	}

	if err := json.Unmarshal(paramData, &m.params); err != nil {
		return nil, fmt.Errorf("failed to parse param JSON: %w", err)
	}

	// Read template
	templateData, err := os.ReadFile(m.templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}
	m.template = string(templateData)

	// Initialize list
	m.updateListItems()

	// Initialize text input
	m.textInput = textinput.New()
	m.textInput.CharLimit = 156
	m.textInput.Width = 50

	return m, nil
}

// Init initializes the Bubble Tea program
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles UI events and updates the model state
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c", "q"))):
			return m, tea.Quit

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			return m.handleEnter()

		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			if m.mode != "list" {
				m.mode = "list"
				m.statusMsg = ""
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("s"))):
			if m.mode == "list" {
				m.mode = "confirm_save"
				m.confirmSave = true
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
			if m.mode == "list" {
				m.mode = "add_param"
				m.textInput.SetValue("")
				m.textInput.Focus()
				m.statusMsg = "Enter new parameter name:"
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
			if m.mode == "list" {
				i, ok := m.list.SelectedItem().(item)
				if ok && i.title != "Template" {
					delete(m.params, i.title)
					m.updateListItems()
					m.statusMsg = fmt.Sprintf("Deleted parameter: %s", i.title)
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.textInput, cmd = m.textInput.Update(msg)

	return m, cmd
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.mode {
	case "list":
		i, ok := m.list.SelectedItem().(item)
		if ok {
			if i.title == "Template" {
				m.mode = "edit_template"
				m.textInput.SetValue(m.template)
				m.textInput.Focus()
			} else {
				m.mode = "edit_param"
				m.currentParam = i.title
				m.textInput.SetValue(fmt.Sprintf("%v", m.params[i.title]))
				m.textInput.Focus()
			}
		}
	case "edit_param":
		newValue := m.textInput.Value()
		oldValue := m.params[m.currentParam]
		m.params[m.currentParam] = convertValue(oldValue, newValue)
		m.mode = "list"
		m.updateListItems()
		m.statusMsg = fmt.Sprintf("Updated parameter: %s", m.currentParam)
	case "edit_template":
		m.template = m.textInput.Value()
		m.mode = "list"
		m.updateListItems()
		m.statusMsg = "Updated template"
	case "confirm_save":
		if m.confirmSave {
			if err := m.saveChanges(); err != nil {
				m.logger.Error().Err(err).Msg("Failed to save changes")
				m.statusMsg = "Error: Failed to save changes"
			} else {
				m.statusMsg = "Changes saved successfully"
			}
		}
		m.mode = "list"
	case "add_param":
		newParamName := m.textInput.Value()
		if newParamName != "" {
			m.params[newParamName] = ""
			m.updateListItems()
			m.mode = "list"
			m.statusMsg = fmt.Sprintf("Added new parameter: %s", newParamName)
		}
	}
	return m, nil
}

// View renders the UI
func (m Model) View() string {
	switch m.mode {
	case "list":
		return fmt.Sprintf(
			"%s\n%s\n\n%s",
			m.list.View(),
			m.statusMsg,
			"(a) add param • (d) delete param • (s) save • (q) quit",
		)
	case "edit_param":
		return fmt.Sprintf("Editing parameter: %s\n\n%s\n\n(Enter) to confirm • (Esc) to cancel", m.currentParam, m.textInput.View())
	case "edit_template":
		return fmt.Sprintf("Editing template:\n\n%s\n\n(Enter) to confirm • (Esc) to cancel", m.textInput.View())
	case "confirm_save":
		return "Are you sure you want to save changes? (y/n)"
	case "add_param":
		return fmt.Sprintf("Enter new parameter name:\n\n%s\n\n(Enter) to confirm • (Esc) to cancel", m.textInput.View())
	default:
		return "Unknown mode"
	}
}

// updateListItems updates the list items based on the current params and template
func (m *Model) updateListItems() {
	items := []list.Item{}
	for k, v := range m.params {
		items = append(items, item{title: k, desc: fmt.Sprintf("%v (%T)", v, v)})
	}
	items = append(items, item{title: "Template", desc: "Edit model template"})

	m.list = list.New(items, list.NewDefaultDelegate(), 0, 0)
	m.list.Title = "Model Parameters"
	m.list.SetShowHelp(false)
}

// saveChanges saves the updated params and template to their respective files
func (m *Model) saveChanges() error {
	// Create backup folders if they don't exist
	backupManifestDir := filepath.Join(filepath.Dir(m.manifestPath), "..", "manifests-backup")
	backupBlobDir := filepath.Join(filepath.Dir(m.paramPath), "..", "blobs-backup")

	if err := os.MkdirAll(backupManifestDir, 0755); err != nil {
		return fmt.Errorf("failed to create manifests backup directory: %w", err)
	}
	if err := os.MkdirAll(backupBlobDir, 0755); err != nil {
		return fmt.Errorf("failed to create blobs backup directory: %w", err)
	}

	// Backup files
	if err := copyFile(m.manifestPath, filepath.Join(backupManifestDir, filepath.Base(m.manifestPath))); err != nil {
		return fmt.Errorf("failed to backup manifest file: %w", err)
	}
	if err := copyFile(m.paramPath, filepath.Join(backupBlobDir, filepath.Base(m.paramPath))); err != nil {
		return fmt.Errorf("failed to backup param file: %w", err)
	}
	if err := copyFile(m.templatePath, filepath.Join(backupBlobDir, filepath.Base(m.templatePath))); err != nil {
		return fmt.Errorf("failed to backup template file: %w", err)
	}

	// Save params
	paramData, err := json.MarshalIndent(m.params, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}
	if err := os.WriteFile(m.paramPath, paramData, 0644); err != nil {
		return fmt.Errorf("failed to write param file: %w", err)
	}

	// Save template
	if err := os.WriteFile(m.templatePath, []byte(m.template), 0644); err != nil {
		return fmt.Errorf("failed to write template file: %w", err)
	}

	m.logger.Info().Msg("Changes saved successfully")
	return nil
}

// copyFile is a helper function to copy a file
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0644)
}

// convertValue attempts to convert the new value to the same type as the old value
func convertValue(oldValue interface{}, newValue string) interface{} {
	switch oldValue.(type) {
	case int:
		if i, err := strconv.Atoi(newValue); err == nil {
			return i
		}
	case float64:
		if f, err := strconv.ParseFloat(newValue, 64); err == nil {
			return f
		}
	case bool:
		if b, err := strconv.ParseBool(newValue); err == nil {
			return b
		}
	case []interface{}:
		var slice []interface{}
		if err := json.Unmarshal([]byte(newValue), &slice); err == nil {
			return slice
		}
	}
	return newValue // If conversion fails, return as string
}

// item represents a list item in the UI
type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

// Run starts the Bubble Tea program
func Run(modelName string) error {
	m, err := Initialize(modelName)
	if err != nil {
		return fmt.Errorf("failed to initialize model: %w", err)
	}

	p := tea.NewProgram(m)
	_, err = p.Run()
	return err
}

// findManifestPath attempts to locate the manifest file for the given model
func findManifestPath(homeDir, modelName string) (string, error) {
	basePath := filepath.Join(homeDir, ".ollama", "models", "manifests")

	// We need to recursively search for the manifest file for the given model, it could be many levels deep and we need to handle the model name with different cases and special characters, for example:
	// /Users/samm/.ollama/models/manifests/registry.ollama.ai/library/tinydolphin/1.1b-v2.8-q5_K_M
	// /Users/samm/.ollama/models/manifests/registry.icu.lol/ollama/replete-coder-qwen-1.5b/q6_k_l

	// An example of the tree structure might be:
	// tree ~/.ollama/models/manifests                                                                                                                                                       (main)
	// [ 128]  /Users/samm/.ollama/models/manifests/
	// ├── [  96]  registry.icu.lol/
	// │   └── [ 576]  ollama/
	// │       ├── [  96]  alphex-118b/
	// │       │   └── [ 710]  iq3_xs
	// │       ├── [  96]  command-r-plus/
	// │       │   └── [ 711]  iq2_xs
	// │       ├── [  96]  deepseek-coder-v2-lite-base/
	// │       │   └── [ 709]  q6_k_l
	// │       ├── [  96]  deepseek-coder-v2-lite-instruct/
	// │       │   └── [ 856]  q6_k_l
	// │       ├── [  96]  granite-34b-code-instruct-bartowski/
	// │       │   └── [ 857]  q5_k_m
	// │       ├── [  96]  hermes-2-pro-llama-3-instruct-merged-dpo/
	// │       │   └── [ 708]  q8_0
	// │       ├── [  96]  llama3-llava-next-8b/
	// │       │   └── [ 709]  q8_0
	// │       ├── [  96]  meta-llama-3-70b-instruct-bartowski/
	// │       │   └── [ 710]  q6_k
	// │       ├── [  96]  meta-llama-3-70b-instruct-dpo-maziyarpanahi/
	// │       │   └── [ 710]  q4_k_m
	// │       ├── [  96]  nous-hermes-2-mixtral-8x7b-dpo-imat/
	// │       │   └── [ 709]  q6_k
	// │       ├── [  96]  openchat-3.6-8b-20240522-bartowski/
	// │       │   └── [ 709]  q6_k
	// │       ├── [  96]  phi-3-medium-128k-instruct-bartowski/
	// │       │   └── [ 710]  q6_k
	// │       ├── [  96]  qwen2-1.5b-32k-instruct-maziyarpanahi/
	// │       │   └── [ 709]  q6_k
	// │       ├── [  96]  qwen2-7b-65k-instruct-maziyarpanahi/
	// │       │   └── [ 709]  q6_k
	// │       ├── [  96]  replete-coder-qwen-1.5b/
	// │       │   └── [ 709]  q6_k_l
	// │       └── [  96]  sfr-embedding-mistral/
	// │           └── [ 854]  q4_k_m
	// └── [ 160]  registry.ollama.ai/
	//     ├── [  96]  closex/
	//     │   └── [  96]  neuraldaredevil-8b-abliterated/
	//     │       └── [ 854]  Q6_K
	//     ├── [ 480]  library/
	//     │   ├── [  96]  codestral-tuned-22b-maziyarpanahi/
	//     │   │   └── [ 856]  q6_k
	//     │   ├── [  96]  command-r/
	//     │   │   └── [ 858]  35b-v0.1-q6_K
	//     │   ├── [  96]  command-r-plus/
	//     │   │   └── [ 858]  latest
	//     │   ├── [  96]  deepseek-coder-v2/
	//     │   │   └── [ 713]  16b-lite-base-q4_K_M
	//     │   ├── [  96]  deepseek-coder-v2-instruct/
	//     │   │   └── [ 856]  iq2_xxs
	//     │   ├── [  96]  explain/
	//     │   │   └── [1.2K]  7b
	//     │   ├── [  96]  llava-llama3/
	//     │   │   └── [ 864]  latest
	//     │   ├── [  96]  mxbai-embed-large/
	//     │   │   └── [ 708]  latest
	//     │   ├── [  96]  nomic-embed-text/
	//     │   │   └── [ 708]  latest
	//     │   ├── [  96]  phi3-128k/
	//     │   │   └── [ 710]  Q8_0
	//     │   ├── [ 128]  qwen2/
	//     │   │   ├── [ 857]  72b-instruct-q3_K_L
	//     │   │   └── [ 857]  72b-instruct-q4_K_M
	//     │   ├── [  96]  tinydolphin/
	//     │   │   └── [1001]  1.1b-v2.8-q5_K_M
	//     │   └── [  96]  wizardlm2/
	//     │       └── [1004]  8x22b-q4_0
	//     └── [  96]  sammcj/
	//         └── [ 128]  qwen2/
	//             ├── [ 711]  72b-instruct-q3_K_L
	//             └── [ 711]  72b-instruct-q4_K_M

	// first get a recursive list of all files in the directory
	files, err := filepath.Glob(filepath.Join(basePath, "**"))
	if err != nil {
		return "", fmt.Errorf("failed to list files in directory: %w", err)
	}

	// the manifests we want look like this example: ~/.ollama/models/manifests/registry.icu.lol/ollama/hermes-2-pro-llama-3-instruct-merged-dpo/q8_0 where the full path to q8_0 is the manifest file, note that it could have any name and no extension or any extension.

	// TODO: complete this function to find the manifest file for the given model name

}
