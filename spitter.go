// spitter.go contains the functions for copying Ollama models to remote hosts using the spitter package.
package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sammcj/gollama/logging"
	"github.com/sammcj/spitter/spitter"
)

// spitterSuccessMsg is sent when a model is successfully copied to a remote host
type spitterSuccessMsg struct {
	modelName   string
	remoteHost  string
	allModels   bool
	modelsCount int
}

// spitterErrorMsg is sent when there is an error copying a model to a remote host
type spitterErrorMsg struct {
	err        error
	modelName  string
	remoteHost string
}

// syncModelToRemote copies a model to a remote host using the spitter package
func (m *AppModel) syncModelToRemote(modelName, remoteHost string, allModels bool) tea.Cmd {
	logging.InfoLogger.Printf("Syncing model: %s to remote host: %s (all models: %v)\n", modelName, remoteHost, allModels)

	return func() tea.Msg {
		var err error

		if allModels {
			// For all models, use the standard spitter.Sync function
			config := spitter.SyncConfig{
				RemoteServer:   remoteHost,
				CustomModelDir: m.ollamaModelsDir,
				AllModels:      true,
			}

			// If Docker container is specified in the config, use it for the Ollama command
			if m.cfg.DockerContainer != "" && strings.ToLower(m.cfg.DockerContainer) != "false" {
				config.OllamaCommand = fmt.Sprintf("docker exec -it %s ollama", m.cfg.DockerContainer)
			}

			err = spitter.Sync(config)
		} else {
			// For a single model, use our custom sync function that handles template variables
			err = syncSingleModelWithTemplateHandling(modelName, remoteHost, m.ollamaModelsDir, m.cfg.DockerContainer)
		}
		if err != nil {
			return spitterErrorMsg{
				err:        err,
				modelName:  modelName,
				remoteHost: remoteHost,
			}
		}

		// Count the number of models synced if AllModels is true
		modelsCount := 1
		if allModels {
			modelsCount = len(m.models)
		}

		return spitterSuccessMsg{
			modelName:   modelName,
			remoteHost:  remoteHost,
			allModels:   allModels,
			modelsCount: modelsCount,
		}
	}
}

// promptForRemoteHost prompts the user for a remote host URL
func promptForRemoteHost() string {
	// Create a text input for the remote host URL
	input := textinput.New()
	input.Placeholder = "Enter remote host URL (e.g., http://192.168.0.75:11434)"
	input.Focus()

	// Create a simple program to handle the text input
	p := tea.NewProgram(remoteHostInputModel{input: input})
	m, err := p.Run()
	if err != nil {
		logging.ErrorLogger.Printf("Error running text input program: %v\n", err)
		return ""
	}

	// Get the final model and extract the URL
	if m, ok := m.(remoteHostInputModel); ok {
		return m.input.Value()
	}

	return ""
}

// remoteHostInputModel is a simple model for handling remote host URL input
type remoteHostInputModel struct {
	input textinput.Model
}

func (m remoteHostInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m remoteHostInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.input.SetValue("")
			return m, tea.Quit
		}
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m remoteHostInputModel) View() string {
	return fmt.Sprintf(
		"Enter remote host URL (e.g., http://192.168.0.75:11434):\n\n%s\n\n(Enter to confirm, Esc to cancel)",
		m.input.View(),
	)
}

// getModelfileContent gets the modelfile content for a model
func getModelfileContent(modelName, ollamaCommand string) (string, error) {
	var cmd *exec.Cmd
	var output []byte
	var err error

	// Try with the provided custom command if specified
	if ollamaCommand != "" {
		logging.InfoLogger.Printf("Using custom Ollama command: %s\n", ollamaCommand)
		parts := strings.Fields(ollamaCommand)
		if len(parts) == 0 {
			return "", fmt.Errorf("invalid ollama command: %s", ollamaCommand)
		}

		args := append(parts[1:], "show", modelName, "--modelfile")
		cmd = exec.Command(parts[0], args...)
		output, err = cmd.CombinedOutput()
		if err == nil {
			return string(output), nil
		}
		logging.ErrorLogger.Printf("Custom command failed: %v\n", err)
	}

	// Try with the local ollama binary
	logging.InfoLogger.Println("Trying local ollama binary...")
	cmd = exec.Command("ollama", "show", modelName, "--modelfile")
	output, err = cmd.CombinedOutput()
	if err == nil {
		return string(output), nil
	}

	// If all attempts fail, return an error
	return "", fmt.Errorf("could not get ollama Modelfile: %w", err)
}

// processModelfile processes the modelfile to handle template variables
func processModelfile(modelfile string) string {
	// Split the modelfile into lines
	lines := strings.Split(modelfile, "\n")
	var processedLines []string
	inTemplate := false

	for _, line := range lines {
		// Check if we're entering or exiting a template block
		if strings.HasPrefix(strings.TrimSpace(line), "TEMPLATE") {
			inTemplate = true
			processedLines = append(processedLines, line)
			continue
		}
		if inTemplate && (strings.Contains(line, "\"\"\"") || strings.Contains(line, "'''")) {
			inTemplate = false
			processedLines = append(processedLines, line)
			continue
		}

		// If we're in a template block, escape any $ variables
		if inTemplate {
			// Replace $i with ${i} to properly escape it
			line = strings.ReplaceAll(line, "$i", "${i}")
			// Also handle other common template variables
			line = strings.ReplaceAll(line, "$1", "${1}")
			line = strings.ReplaceAll(line, "$2", "${2}")
			line = strings.ReplaceAll(line, "$3", "${3}")
		}

		processedLines = append(processedLines, line)
	}

	return strings.Join(processedLines, "\n")
}

// syncSingleModelWithTemplateHandling syncs a single model to a remote host with special handling for template variables
func syncSingleModelWithTemplateHandling(modelName, remoteHost, customModelDir, dockerContainer string) error {
	logging.InfoLogger.Printf("Syncing model %s to remote host %s with template handling\n", modelName, remoteHost)

	// Create the standard spitter configuration
	config := spitter.SyncConfig{
		LocalModel:     modelName,
		RemoteServer:   remoteHost,
		CustomModelDir: customModelDir,
	}

	// If Docker container is specified, use it for the Ollama command
	if dockerContainer != "" && strings.ToLower(dockerContainer) != "false" {
		config.OllamaCommand = fmt.Sprintf("docker exec -it %s ollama", dockerContainer)
	}

	// First, let's check if the model exists
	checkCmd := exec.Command("ollama", "list")
	checkOutput, err := checkCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error checking models: %w - %s", err, string(checkOutput))
	}

	if !strings.Contains(string(checkOutput), modelName) {
		return fmt.Errorf("model %s not found locally", modelName)
	}

	logging.InfoLogger.Printf("Model %s found locally\n", modelName)

	// Use the standard spitter.Sync function
	// The spitter package will handle the model files and transfer them to the remote host
	// It will only modify the FROM line in the modelfile, replacing it with the files line containing the SHA
	return spitter.Sync(config)
}
