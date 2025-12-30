// operations.go contains the functions that perform the operations on the models.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ollama/ollama/api"
	"github.com/sammcj/gollama/v2/config"
	"github.com/sammcj/gollama/v2/logging"
	"github.com/sammcj/gollama/v2/styles"
)

func runModel(model string, cfg *config.Config) tea.Cmd {
	if cfg.DockerContainer != "" && strings.ToLower(cfg.DockerContainer) != "false" {
		return runDocker(cfg.DockerContainer, model)
	}

	ollamaPath, err := exec.LookPath("ollama")
	if err != nil {
		logging.ErrorLogger.Printf("error finding ollama binary: %v\n", err)
		logging.ErrorLogger.Printf("If you're running Ollama in a container, make sure you updated the config file with the container name\n")
		return nil
	}
	c := exec.Command(ollamaPath, "run", model)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			logging.ErrorLogger.Printf("error running model: %v\n", err)
		}
		return runFinishedMessage{err}
	})
}

func runDocker(container string, model string) tea.Cmd {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		logging.ErrorLogger.Printf("error finding docker binary: %v\n", err)
		return nil
	}

	args := []string{"exec", "-it", container, "ollama", "run", model}
	c := exec.Command(dockerPath, args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			logging.ErrorLogger.Printf("error running model in docker container: %v\n", err)
		}
		return runFinishedMessage{err}
	})
}

func deleteModel(client *api.Client, name string) error {
	ctx := context.Background()
	req := &api.DeleteRequest{Name: name}
	logging.DebugLogger.Printf("Attempting to delete model: %s\n", name)

	err := client.Delete(ctx, req)
	if err != nil {
		logging.ErrorLogger.Printf("Error deleting model %s: %v\n", name, err)
		return fmt.Errorf("error deleting model %s: %v", name, err)
	}

	logging.InfoLogger.Printf("Successfully deleted model: %s\n", name)
	return nil
}

func copyModel(m *AppModel, client *api.Client, oldName string, newName string) {
	ctx := context.Background()
	req := &api.CopyRequest{
		Source:      oldName,
		Destination: newName,
	}
	err := client.Copy(ctx, req)
	if err != nil {
		// Check if error indicates cancellation
		if strings.Contains(err.Error(), "operation was canceled") {
			logging.InfoLogger.Printf("Copy operation cancelled for model: %s\n", oldName)
			return
		}
		logging.ErrorLogger.Printf("Error copying model: %v\n", err)
		return
	}

	logging.InfoLogger.Printf("Successfully copied model: %s to %s\n", oldName, newName)

	resp, err := client.List(ctx)
	if err != nil {
		logging.ErrorLogger.Printf("Error fetching models: %v\n", err)
		return
	}
	m.models = parseAPIResponse(resp)
	m.refreshList()
}

func renameModel(m *AppModel, oldName string, newName string) error {
	if newName == "" {
		return fmt.Errorf("no new name provided")
	}
	copyModel(m, m.client, oldName, newName)
	deleteModel(m.client, oldName)
	for i, model := range m.models {
		if model.Name == oldName {
			m.models = append(m.models[:i], m.models[i+1:]...)
			break
		}
	}

	message := fmt.Sprintf("Successfully renamed model %s to %s", oldName, newName)
	logging.InfoLogger.Printf("%s", message)
	return nil
}

func showRunningModels(client *api.Client) ([]table.Row, error) {
	ctx := context.Background()
	resp, err := client.ListRunning(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching running models: %v", err)
	}

	var runningModels []table.Row
	for _, model := range resp.Models {
		name := model.Name
		size := float64(model.Size) / 1024 / 1024 / 1024
		vram := float64(model.SizeVRAM) / 1024 / 1024 / 1024
		until := model.ExpiresAt.Format("2006-01-02 15:04:05")

		runningModels = append(runningModels, table.Row{name, fmt.Sprintf("%.2f GB", size), fmt.Sprintf("%.2f GB", vram), until})
		logging.DebugLogger.Printf("Running model: %s\n", name)
	}

	return runningModels, nil
}

func unloadModel(client *api.Client, modelName string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("invalid API client: client is nil")
	}

	ctx := context.Background()

	if strings.Contains(modelName, "embed") {
		req := &api.EmbeddingRequest{
			Model:     modelName,
			KeepAlive: &api.Duration{Duration: 0},
		}
		logging.DebugLogger.Printf("Attempting to unload embedding model: %s\n", modelName)

		_, err := client.Embeddings(ctx, req)
		if err != nil {
			logging.ErrorLogger.Printf("Failed to unload embedding model: %v\n", err)
			return modelName, err
		}
	} else {
		req := &api.GenerateRequest{
			Model:     modelName,
			Prompt:    "",
			KeepAlive: &api.Duration{Duration: 0},
		}
		logging.DebugLogger.Printf("Attempting to unload model: %s\n", modelName)

		response := func(resp api.GenerateResponse) error {
			logging.DebugLogger.Printf("done")
			return nil
		}
		err := client.Generate(ctx, req, response)
		if err != nil {
			logging.ErrorLogger.Printf("Failed to unload model: %v\n", err)
			return "", err
		}
	}

	return modelName, nil
}

func getModelParams(modelName string, client *api.Client) (map[string]string, string, error) {
	logging.InfoLogger.Printf("Getting parameters for model: %s\n", modelName)
	ctx := context.Background()
	req := &api.ShowRequest{Name: modelName}
	resp, err := client.Show(ctx, req)
	if err != nil {
		logging.ErrorLogger.Printf("Error getting parameters for model %s: %v\n", modelName, err)
		return nil, "", err
	}
	output := []byte(resp.Modelfile)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	params := make(map[string]string)
	var template string

	for _, line := range lines {
		if strings.HasPrefix(line, "TEMPLATE") {
			template = strings.TrimPrefix(line, "TEMPLATE ")
			template = strings.Trim(template, "\"")
		} else if strings.HasPrefix(line, "PARAMETER") {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				key := parts[1]
				value := strings.TrimSpace(parts[1])
				params[key] = value
			}
		}
	}
	return params, template, nil
}

// getEnhancedModelInfo fetches detailed model information using the Ollama show API
func getEnhancedModelInfo(modelName string, client *api.Client) (*EnhancedModelInfo, error) {
	logging.InfoLogger.Printf("Getting enhanced model information for: %s\n", modelName)
	ctx := context.Background()
	req := &api.ShowRequest{Name: modelName}
	resp, err := client.Show(ctx, req)
	if err != nil {
		logging.ErrorLogger.Printf("Error getting enhanced model info for %s: %v\n", modelName, err)
		return nil, err
	}

	info := &EnhancedModelInfo{}

	// Extract information from Details
	info.ParameterSize = resp.Details.ParameterSize
	info.QuantizationLevel = resp.Details.QuantizationLevel
	info.Format = resp.Details.Format
	info.Family = resp.Details.Family

	// Extract information from ModelInfo
	if resp.ModelInfo != nil {
		// Log all available keys for debugging
		logging.DebugLogger.Printf("Available ModelInfo keys for %s:", modelName)
		for key := range resp.ModelInfo {
			logging.DebugLogger.Printf("  - %s", key)
		}

		// Generic field extraction - search for patterns in ModelInfo keys
		for key, value := range resp.ModelInfo {
			if val, ok := value.(float64); ok {
				// Context length - look for keys ending with context_length
				if strings.HasSuffix(key, "context_length") && info.ContextLength == 0 {
					info.ContextLength = int64(val)
					logging.DebugLogger.Printf("Found context length: %d (key: %s)", info.ContextLength, key)
				}

				// Embedding length - look for keys ending with embedding_length
				if strings.HasSuffix(key, "embedding_length") && info.EmbeddingLength == 0 {
					info.EmbeddingLength = int64(val)
					logging.DebugLogger.Printf("Found embedding length: %d (key: %s)", info.EmbeddingLength, key)
				}

				// Rope dimension count - look for keys containing rope and dimension
				if (strings.Contains(key, "rope") && strings.Contains(key, "dimension")) ||
					strings.HasSuffix(key, "attention.key_length") ||
					strings.HasSuffix(key, "attention.value_length") {
					if info.RopeDimensionCount == 0 {
						info.RopeDimensionCount = int64(val)
						logging.DebugLogger.Printf("Found rope dimension count: %d (key: %s)", info.RopeDimensionCount, key)
					}
				}

				// Rope frequency base - look for keys containing rope and freq
				if strings.Contains(key, "rope") && strings.Contains(key, "freq") && info.RopeFreqBase == 0 {
					info.RopeFreqBase = val
					logging.DebugLogger.Printf("Found rope freq base: %.0f (key: %s)", info.RopeFreqBase, key)
				}

				// Vocabulary size - look for keys containing vocab
				if strings.Contains(key, "vocab") && strings.Contains(key, "size") && info.VocabSize == 0 {
					info.VocabSize = int64(val)
					logging.DebugLogger.Printf("Found vocab size: %d (key: %s)", info.VocabSize, key)
				}
			}

			// Handle special cases for vocab size from token arrays
			if strings.Contains(key, "tokens") && info.VocabSize == 0 {
				if tokens, ok := value.([]interface{}); ok {
					info.VocabSize = int64(len(tokens))
					logging.DebugLogger.Printf("Found vocab size from tokens array: %d (key: %s)", info.VocabSize, key)
				}
			}
		}

		// If vocab size still not found, try to estimate from parameter count and embedding length
		if info.VocabSize == 0 {
			if paramCount, exists := resp.ModelInfo["general.parameter_count"]; exists {
				if paramVal, ok := paramCount.(float64); ok && info.EmbeddingLength > 0 {
					// Rough estimation: vocab_size ≈ (param_count - other_params) / embedding_length
					// This is a very rough estimate and may not be accurate
					estimatedVocab := int64(paramVal / float64(info.EmbeddingLength) / 100) // Rough divisor
					if estimatedVocab > 10000 && estimatedVocab < 200000 {                  // Reasonable range
						info.VocabSize = estimatedVocab
						logging.DebugLogger.Printf("Estimated vocab size: %d", info.VocabSize)
					}
				}
			}
		}
	}

	// Extract capabilities - convert from []model.Capability to []string
	if resp.Capabilities != nil {
		capabilities := make([]string, len(resp.Capabilities))
		for i, cap := range resp.Capabilities {
			capabilities[i] = string(cap)
		}
		info.Capabilities = capabilities
	}

	return info, nil
}

func editModelfile(client *api.Client, modelName string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("error: Client is nil")
	}
	ctx := context.Background()

	// Fetch the current modelfile from the server
	showResp, err := client.Show(ctx, &api.ShowRequest{Name: modelName})
	if err != nil {
		return "", fmt.Errorf("error fetching modelfile for %s: %v", modelName, err)
	}
	modelfileContent := showResp.Modelfile

	// Get editor from environment or config
	editor := getEditor()

	logging.DebugLogger.Printf("Using editor: %s for model: %s\n", editor, modelName)

	// Create a secure temporary file with random name
	tempFile, err := os.CreateTemp("", fmt.Sprintf("gollama_%s_*.modelfile", strings.ReplaceAll(strings.ReplaceAll(modelName, ":", "_"), "/", "_")))
	if err != nil {
		return "", fmt.Errorf("error creating temporary file: %v", err)
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	newModelfilePath := tempFile.Name()

	// Write the content to the temporary file
	_, err = tempFile.Write([]byte(modelfileContent))
	if err != nil {
		return "", fmt.Errorf("error writing modelfile to temp file: %v", err)
	}

	// Open the local modelfile in the editor
	cmd := exec.Command(editor, newModelfilePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running editor: %v", err)
	}

	// Read the edited content from the local file
	newModelfileContent, err := os.ReadFile(newModelfilePath)
	if err != nil {
		return "", fmt.Errorf("error reading edited modelfile: %v", err)
	}

	// If there were no changes, return early
	if string(newModelfileContent) == modelfileContent {
		return fmt.Sprintf("No changes made to model %s", modelName), nil
	}

	// Extract TEMPLATE, SYSTEM, and parameters from both original and new content
	var origTemplate, origSystem, newTemplate, newSystem string

	// Create request with base fields
	createReq := &api.CreateRequest{
		Model: modelName, // The model to update
		From:  modelName, // Required: use the model's own name as the base
	}

	origTemplate, origSystem = extractTemplateAndSystem(modelfileContent)
	newTemplate, newSystem = extractTemplateAndSystem(string(newModelfileContent))

	// Only include template if it was changed
	if newTemplate != origTemplate {
		logging.DebugLogger.Printf("Template was modified for model %s", modelName)
		createReq.Template = newTemplate
	}

	if newSystem != origSystem {
		logging.DebugLogger.Printf("System prompt was modified for model %s", modelName)
		createReq.System = newSystem
	}

	// Add parameters if any were found
	parameters := extractParameters(string(newModelfileContent))
	if len(parameters) > 0 {
		createReq.Parameters = parameters
	}

	logging.DebugLogger.Printf("Updating model %s with changes:\n", modelName)
	if newTemplate != origTemplate {
		logging.DebugLogger.Printf("- Modified template\n")
	}
	if newSystem != origSystem {
		logging.DebugLogger.Printf("- Modified system prompt\n")
	}
	if len(parameters) > 0 {
		logging.DebugLogger.Printf("- Modified parameters: %+v\n", parameters)
	}

	reqJson, jsonErr := json.Marshal(createReq)
	if jsonErr == nil {
		logging.DebugLogger.Printf("Create request: %s", string(reqJson))
	}

	err = client.Create(ctx, createReq, func(resp api.ProgressResponse) error {
		logging.DebugLogger.Printf("Create progress: Status=%s, Digest=%s, Total=%d, Completed=%d\n",
			resp.Status, resp.Digest, resp.Total, resp.Completed)
		return nil
	})
	if err != nil {
		errMsg := fmt.Sprintf("Failed to update model %s", modelName)
		if strings.Contains(err.Error(), "error getting blobs path") {
			errMsg += fmt.Sprintf(": error updating model parameters. This may occur with remote Ollama instances")
		} else if strings.Contains(err.Error(), "no such file or directory") {
			errMsg += fmt.Sprintf(": model file not found. This may occur with remote Ollama instances")
		} else {
			errMsg += fmt.Sprintf(": %v", err)
		}
		return "", fmt.Errorf("%s (check debug logs for details)", errMsg)
	}

	// log to the console if we're not in a tea app
	fmt.Printf("Model %s updated successfully\n", modelName)

	return fmt.Sprintf("Model %s updated successfully", modelName), nil
}

func isLocalhost(url string) bool {
	return strings.Contains(url, "localhost") || strings.Contains(url, "127.0.0.1")
}

func parseContextSize(input string) (int, error) {
	input = strings.ToLower(strings.TrimSpace(input))
	multiplier := 1

	if strings.HasSuffix(input, "k") {
		multiplier = 1024
		input = strings.TrimSuffix(input, "k")
	} else if strings.HasSuffix(input, "m") {
		multiplier = 1024 * 1024
		input = strings.TrimSuffix(input, "m")
	}

	value, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("invalid context size: %s", input)
	}

	return value * multiplier, nil
}

func getEditor() string {
	// First check if editor is explicitly configured in config file
	cfg, err := config.LoadConfig()
	if err != nil {
		logging.ErrorLogger.Printf("Error loading config for editor: %v\n", err)
	} else if cfg.Editor != "" {
		// Config file has an editor explicitly set, use it
		return cfg.Editor
	}

	// Fallback to environment variable
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	// Final fallback to vim
	return "vim"
}

func isExternalEditor(editor string) bool {
	// Check if the editor is not vim/vi/nano (terminal-based editors)
	switch editor {
	case "vim", "vi", "nvim", "nano", "emacs", "":
		return false
	default:
		// Check if it's a command with arguments (like "code --wait")
		// External editors often need flags to wait for the file to be closed
		return true
	}
}

func startExternalEditor(client *api.Client, modelName string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("error: Client is nil")
	}
	ctx := context.Background()

	// Fetch the current modelfile from the server
	showResp, err := client.Show(ctx, &api.ShowRequest{Name: modelName})
	if err != nil {
		return "", fmt.Errorf("error fetching modelfile for %s: %v", modelName, err)
	}
	modelfileContent := showResp.Modelfile

	// Get editor from environment or config
	originalEditor := getEditor()

	// Prepare editor command with flags if needed (don't modify original config)
	editor := originalEditor
	if strings.Contains(editor, "code") && !strings.Contains(editor, "--wait") {
		editor = editor + " --wait"
	}

	logging.DebugLogger.Printf("Starting external editor: %s for model: %s\n", editor, modelName)

	// Create a secure temporary file with random name
	tempFile, err := os.CreateTemp("", fmt.Sprintf("gollama_%s_*.modelfile", strings.ReplaceAll(strings.ReplaceAll(modelName, ":", "_"), "/", "_")))
	if err != nil {
		return "", fmt.Errorf("error creating temporary file: %v", err)
	}
	defer tempFile.Close()

	newModelfilePath := tempFile.Name()

	// Write the content to the temporary file
	_, err = tempFile.Write([]byte(modelfileContent))
	if err != nil {
		os.Remove(newModelfilePath) // Clean up on error
		return "", fmt.Errorf("error writing modelfile to temp file: %v", err)
	}

	// Try to launch the external editor immediately to catch any errors
	logging.DebugLogger.Printf("Launching editor command: %s %s", editor, newModelfilePath)

	// Build the command properly handling spaces and arguments
	var cmd *exec.Cmd
	var fullCommand string

	// Handle complex editor paths and arguments
	if strings.Contains(editor, " ") {
		// For VS Code path with arguments, we need to properly separate the executable from args
		// Special handling for VS Code paths that end with "code --wait" or similar
		if strings.Contains(editor, "/code --wait") || strings.Contains(editor, "/code ") {
			// Split at the last occurrence of "/code " to separate path from args
			parts := strings.Split(editor, "/code ")
			if len(parts) == 2 {
				codePath := parts[0] + "/code"
				args := strings.TrimSpace(parts[1])
				if args != "" {
					fullCommand = fmt.Sprintf(`"%s" %s "%s"`, codePath, args, newModelfilePath)
				} else {
					fullCommand = fmt.Sprintf(`"%s" "%s"`, codePath, newModelfilePath)
				}
			} else {
				// Fallback: quote the whole thing
				fullCommand = fmt.Sprintf(`"%s" "%s"`, editor, newModelfilePath)
			}
		} else {
			// For other commands with spaces, quote the whole thing
			fullCommand = fmt.Sprintf(`"%s" "%s"`, editor, newModelfilePath)
		}
		cmd = exec.Command("sh", "-c", fullCommand)
		logging.DebugLogger.Printf("Using shell execution with command: sh -c '%s'", fullCommand)
	} else {
		// Simple command without spaces
		cmd = exec.Command(editor, newModelfilePath)
		logging.DebugLogger.Printf("Using direct execution: %s %s", editor, newModelfilePath)
	}

	// Start the command to validate it works
	err = cmd.Start()
	if err != nil {
		os.Remove(newModelfilePath) // Clean up temp file on error
		return "", fmt.Errorf("failed to start editor: %v", err)
	}

	logging.DebugLogger.Printf("Successfully started editor process with PID: %d", cmd.Process.Pid)

	// Let it run in background but don't wait for completion here
	go func() {
		err := cmd.Wait()
		if err != nil {
			logging.ErrorLogger.Printf("Editor process ended with error: %v", err)
		}
		logging.DebugLogger.Printf("External editor process completed")
	}()

	return newModelfilePath, nil
}

func finishExternalEdit(client *api.Client, modelName, tempFilePath string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("error: Client is nil")
	}
	ctx := context.Background()

	// Read the edited content from the local file
	newModelfileContent, err := os.ReadFile(tempFilePath)
	if err != nil {
		return "", fmt.Errorf("error reading edited modelfile: %v", err)
	}

	// Clean up the temporary file
	defer os.Remove(tempFilePath)

	// Fetch the current modelfile from the server to compare
	showResp, err := client.Show(ctx, &api.ShowRequest{Name: modelName})
	if err != nil {
		return "", fmt.Errorf("error fetching modelfile for %s: %v", modelName, err)
	}
	originalContent := showResp.Modelfile

	// If there were no changes, return early
	if string(newModelfileContent) == originalContent {
		return fmt.Sprintf("No changes made to model %s", modelName), nil
	}

	// Extract TEMPLATE, SYSTEM, and parameters from both original and new content
	var origTemplate, origSystem, newTemplate, newSystem string

	// Create request with base fields
	createReq := &api.CreateRequest{
		Model: modelName, // The model to update
		From:  modelName, // Required: use the model's own name as the base
	}

	origTemplate, origSystem = extractTemplateAndSystem(originalContent)
	newTemplate, newSystem = extractTemplateAndSystem(string(newModelfileContent))

	// Only include template if it was changed
	if newTemplate != origTemplate {
		logging.DebugLogger.Printf("Template was modified for model %s", modelName)
		createReq.Template = newTemplate
	}

	if newSystem != origSystem {
		logging.DebugLogger.Printf("System prompt was modified for model %s", modelName)
		createReq.System = newSystem
	}

	// Parse new content for parameters
	newLines := strings.Split(string(newModelfileContent), "\n")
	params := make(map[string]interface{})
	for _, line := range newLines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Skip TEMPLATE and SYSTEM lines
		if strings.HasPrefix(strings.ToUpper(line), "TEMPLATE") ||
			strings.HasPrefix(strings.ToUpper(line), "SYSTEM") ||
			strings.HasPrefix(strings.ToUpper(line), "FROM") {
			continue
		}

		if strings.HasPrefix(strings.ToUpper(line), "PARAMETER") {
			parts := strings.SplitN(line, " ", 3)
			if len(parts) >= 3 {
				paramName := strings.ToLower(parts[1])
				paramValue := strings.TrimSpace(parts[2])

				if floatVal, err := strconv.ParseFloat(paramValue, 64); err == nil {
					params[paramName] = floatVal
				} else if intVal, err := strconv.Atoi(paramValue); err == nil {
					params[paramName] = intVal
				} else {
					params[paramName] = paramValue
				}
			}
		}
	}

	if len(params) > 0 {
		createReq.Parameters = params
	}

	// Stream the model creation
	err = client.Create(ctx, createReq, func(resp api.ProgressResponse) error {
		logging.DebugLogger.Printf("Create response: %s\n", resp.Status)
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error updating model: %v", err)
	}

	return fmt.Sprintf("Successfully updated model %s", modelName), nil
}

func createModelFromModelfile(modelName, modelfilePath string, client *api.Client) error {
	ctx := context.Background()
	content, err := os.ReadFile(modelfilePath)
	if err != nil {
		return fmt.Errorf("error reading modelfile %s: %v", modelfilePath, err)
	}

	req := &api.CreateRequest{
		Model: modelName,
		Files: map[string]string{
			"modelfile": string(content),
		},
	}

	err = client.Create(ctx, req, nil)
	if err != nil {
		logging.ErrorLogger.Printf("Error creating model from modelfile %s: %v\n", modelfilePath, err)
		return fmt.Errorf("error creating model from modelfile %s: %v", modelfilePath, err)
	}
	logging.InfoLogger.Printf("Successfully created model from modelfile: %s\n", modelfilePath)
	return nil
}

func (m *AppModel) startPushModel(modelName string) tea.Cmd {
	logging.InfoLogger.Printf("Pushing model: %s\n", modelName)
	return func() tea.Msg {
		ctx := context.Background()
		req := &api.PushRequest{Name: modelName}
		err := m.client.Push(ctx, req, func(resp api.ProgressResponse) error {
			return nil
		})
		if err != nil {
			return pushErrorMsg{err}
		}
		return pushSuccessMsg{modelName}
	}
}

// ModelFiles represents the files associated with a model
type ModelFiles struct {
	MainModel string // Primary model file (usually .gguf)
	Projector string // Vision projector file (mmproj, if present)
}

// getModelFiles returns all files associated with a model (main model + any projector files)
func getModelFiles(modelName string, client *api.Client) (ModelFiles, error) {
	ctx := context.Background()
	req := &api.ShowRequest{Name: modelName}
	resp, err := client.Show(ctx, req)
	if err != nil {
		return ModelFiles{}, err
	}

	output := []byte(resp.Modelfile)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	var files ModelFiles
	var fromPaths []string

	// Collect all FROM lines
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "FROM ") {
			path := strings.TrimSpace(line[5:])
			fromPaths = append(fromPaths, path)
		}
	}

	if len(fromPaths) == 0 {
		message := "failed to get model path for %s: no FROM line in output"
		logging.ErrorLogger.Printf(message+"\n", modelName)
		return files, fmt.Errorf(message, modelName)
	}

	// First FROM line is always the main model
	files.MainModel = fromPaths[0]

	// If there are additional FROM lines, the second one is typically the projector
	// (This handles vision models with separate projector files)
	if len(fromPaths) > 1 {
		files.Projector = fromPaths[1]
		logging.DebugLogger.Printf("Found multi-file model %s: main=%s, projector=%s",
			modelName, files.MainModel, files.Projector)
	}

	return files, nil
}

func getModelPath(modelName string, client *api.Client) (string, error) {
	ctx := context.Background()
	req := &api.ShowRequest{Name: modelName}
	resp, err := client.Show(ctx, req)
	if err != nil {
		return "", err
	}

	output := []byte(resp.Modelfile)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "FROM ") {
			return strings.TrimSpace(line[5:]), nil
		}
	}
	message := "failed to get model path for %s: no FROM line in output"
	logging.ErrorLogger.Printf(message+"\n", modelName)
	return "", fmt.Errorf(message, modelName)
}

func getOriginalModelName(client *api.Client, modelName string) (string, error) {
	ctx := context.Background()
	req := &api.ShowRequest{Name: modelName}
	resp, err := client.Show(ctx, req)
	if err != nil {
		return "", err
	}

	output := []byte(resp.Modelfile)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "FROM ") {
			// Extract the original model name from the FROM directive
			fromName := strings.TrimSpace(line[5:])
			if !strings.Contains(fromName, "/root/.ollama/models/blobs/") {
				return fromName, nil
			}
		}
	}
	message := "failed to get original model name for %s: no valid FROM line in output"
	logging.ErrorLogger.Printf(message+"\n", modelName)
	return "", fmt.Errorf(message, modelName)
}

func isValidSymlink(symlinkPath, targetPath string) bool {
	// Check if the symlink matches the expected naming convention
	expectedSuffix := ".gguf"
	if !strings.HasSuffix(filepath.Base(symlinkPath), expectedSuffix) {
		return false
	}

	// Check if the target file exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return false
	}

	// Check if the symlink target is a file (not a directory or another symlink)
	fileInfo, err := os.Lstat(targetPath)
	if err != nil || fileInfo.Mode()&os.ModeSymlink != 0 || fileInfo.IsDir() {
		logging.DebugLogger.Printf("Symlink target is not a file: %s\n", targetPath)
		return false
	}

	return true
}

func (m *AppModel) startPullModel(modelName string) tea.Cmd {
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

// startPullModelPreserveConfig pulls a model while preserving user-modified configuration
func (m *AppModel) startPullModelPreserveConfig(modelName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Step 1: Extract current parameters and template before pulling
		logging.InfoLogger.Printf("Extracting parameters for model %s before pulling", modelName)
		currentParams, currentTemplate, systemPrompt, err := getModelParamsWithSystem(modelName, m.client)
		if err != nil {
			logging.ErrorLogger.Printf("Error extracting parameters for model %s: %v", modelName, err)
			return pullErrorMsg{fmt.Errorf("failed to extract parameters: %v", err)}
		}

		// Step 2: Pull the updated model
		logging.InfoLogger.Printf("Pulling updated model: %s", modelName)
		req := &api.PullRequest{Name: modelName}
		err = m.client.Pull(ctx, req, func(resp api.ProgressResponse) error {
			m.pullProgress = float64(resp.Completed) / float64(resp.Total)
			return nil
		})
		if err != nil {
			return pullErrorMsg{err}
		}

		// Step 3: Apply the saved configuration back to the updated model
		logging.InfoLogger.Printf("Restoring configuration for model: %s", modelName)

		// Create request with base fields
		createReq := &api.CreateRequest{
			Model: modelName, // The model to update
			From:  modelName, // Use the same model name as base (it's now been updated)
		}

		// Add template if it exists
		if currentTemplate != "" {
			createReq.Template = currentTemplate
		}

		// Add system prompt if it exists
		if systemPrompt != "" {
			createReq.System = systemPrompt
		}

		// Add parameters if any were found
		if len(currentParams) > 0 {
			// Convert map[string]string to map[string]any
			parameters := make(map[string]any)
			for k, v := range currentParams {
				// Try to convert numeric values
				if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
					parameters[k] = floatVal
				} else if intVal, err := strconv.Atoi(v); err == nil {
					parameters[k] = intVal
				} else {
					parameters[k] = v
				}
			}
			createReq.Parameters = parameters
		}

		// Apply the configuration
		err = m.client.Create(ctx, createReq, func(resp api.ProgressResponse) error {
			return nil
		})
		if err != nil {
			return pullErrorMsg{fmt.Errorf("failed to restore configuration: %v", err)}
		}

		return pullSuccessMsg{modelName}
	}
}

type editorFinishedMsg struct{ err error }

func searchModels(models []Model, searchTerms ...string) {
	logging.InfoLogger.Printf("Searching for models with terms: %v\n", searchTerms)

	var searchResults []Model
	for _, model := range models {
		if containsAllTerms(model.Name, searchTerms...) {
			searchResults = append(searchResults, model)
		}
	}

	sort.Slice(searchResults, func(i, j int) bool {
		return strings.ToLower(searchResults[i].Name) < strings.ToLower(searchResults[j].Name)
	})

	baseStyle, highlightStyle, headerStyle := styles.SearchHighlightStyle(), styles.SearchTextStyle(), styles.SearchHeaderStyle()

	for i, model := range searchResults {
		colourisedName := model.Name
		for _, term := range searchTerms {
			andTerms := strings.Split(term, "&")
			colourisedName = highlightTerms(colourisedName, baseStyle, highlightStyle, andTerms)
		}
		searchResults[i].Name = colourisedName
	}

	fmt.Println(headerStyle.Render("Search results for: " + highlightStyle.Render(strings.Join(searchTerms, " "))))
	fmt.Println(headerStyle.Render("-------------------"))
	if len(searchResults) == 0 {
		fmt.Println("No matching models found.")
		logging.InfoLogger.Println("No matching models found.")
	} else {
		for _, model := range searchResults {
			fmt.Println(model.Name)
		}
		logging.InfoLogger.Printf("Found %d matching models\n", len(searchResults))
	}
}

func highlightTerms(modelName string, baseStyle, highlightStyle lipgloss.Style, searchTerms []string) string {
	lowercaseName := strings.ToLower(modelName)
	var highlights []struct{ start, end int }

	for _, term := range searchTerms {
		orTerms := strings.Split(term, "|")
		for _, orTerm := range orTerms {
			lowercaseOrTerm := strings.ToLower(orTerm)
			start := 0
			for {
				index := strings.Index(lowercaseName[start:], lowercaseOrTerm)
				if index == -1 {
					break
				}
				highlights = append(highlights, struct{ start, end int }{start + index, start + index + len(orTerm)})
				start += index + len(orTerm)
			}
		}
	}

	if len(highlights) == 0 {
		return baseStyle.Render(modelName)
	}

	sort.Slice(highlights, func(i, j int) bool {
		return highlights[i].start < highlights[j].start
	})

	var result strings.Builder
	lastEnd := 0
	for _, h := range highlights {
		if h.start < lastEnd {
			continue // Skip overlapping highlights
		}
		result.WriteString(baseStyle.Render(modelName[lastEnd:h.start]))
		result.WriteString(highlightStyle.Render(modelName[h.start:h.end]))
		lastEnd = h.end
	}
	result.WriteString(baseStyle.Render(modelName[lastEnd:]))

	return result.String()
}

func containsAllTerms(s string, terms ...string) bool {
	lowercaseS := strings.ToLower(s)
	for _, term := range terms {
		andTerms := strings.Split(term, "&")
		if !containsAllAndTerms(lowercaseS, andTerms...) {
			return false
		}
	}
	return true
}

func containsAllAndTerms(s string, terms ...string) bool {
	for _, term := range terms {
		orTerms := strings.Split(term, "|")
		if !containsAnyTerm(s, orTerms...) {
			return false
		}
	}
	return true
}

func containsAnyTerm(s string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(s, strings.ToLower(term)) {
			return true
		}
	}
	return false
}

// extractTemplateAndSystem extracts TEMPLATE and SYSTEM values from modelfile content
// preserving any template syntax within them
func extractTemplateAndSystem(content string) (template string, system string) {
	lines := strings.Split(content, "\n")
	var templateLines []string
	inTemplate := false
	inMultilineTemplate := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle TEMPLATE directive
		if strings.HasPrefix(trimmed, "TEMPLATE") {
			if strings.Contains(trimmed, `"""`) {
				// Multi-line template
				templateContent := strings.TrimPrefix(trimmed, "TEMPLATE ")
				templateContent = strings.TrimSpace(templateContent)
				if strings.HasPrefix(templateContent, `"""`) {
					templateContent = strings.TrimPrefix(templateContent, `"""`)
				}
				inTemplate = true
				inMultilineTemplate = true
				if templateContent != "" {
					templateLines = append(templateLines, templateContent)
				}
			} else {
				// Single-line template
				template = strings.TrimPrefix(trimmed, "TEMPLATE ")
				template = strings.Trim(template, `"`)
			}
		} else if inTemplate {
			if inMultilineTemplate && strings.HasSuffix(trimmed, `"""`) {
				line = strings.TrimSuffix(line, `"""`)
				if line != "" {
					templateLines = append(templateLines, line)
				}
				inTemplate = false
				inMultilineTemplate = false
			} else {
				templateLines = append(templateLines, line)
			}
		} else if strings.HasPrefix(trimmed, "SYSTEM") {
			system = strings.TrimPrefix(trimmed, "SYSTEM ")
			// Remove surrounding quotes if present
			system = strings.Trim(system, `"`)
		}
	}

	if len(templateLines) > 0 {
		template = strings.Join(templateLines, "\n")
	}

	return template, system
}

// getModelParamsWithSystem extracts parameters, template, and system prompt from a model's modelfile
func getModelParamsWithSystem(modelName string, client *api.Client) (map[string]string, string, string, error) {
	logging.InfoLogger.Printf("Getting parameters and system prompt for model: %s\n", modelName)
	ctx := context.Background()
	req := &api.ShowRequest{Name: modelName}
	resp, err := client.Show(ctx, req)
	if err != nil {
		logging.ErrorLogger.Printf("Error getting modelfile for %s: %v\n", modelName, err)
		return nil, "", "", err
	}
	output := []byte(resp.Modelfile)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	params := make(map[string]string)
	var template string
	var system string

	inTemplate := false
	inMultilineTemplate := false
	var templateLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle TEMPLATE directive
		if strings.HasPrefix(trimmed, "TEMPLATE") {
			if strings.Contains(trimmed, `"""`) {
				// Multi-line template
				templateContent := strings.TrimPrefix(trimmed, "TEMPLATE ")
				templateContent = strings.TrimSpace(templateContent)
				if strings.HasPrefix(templateContent, `"""`) {
					templateContent = strings.TrimPrefix(templateContent, `"""`)
				}
				inTemplate = true
				inMultilineTemplate = true
				if templateContent != "" {
					templateLines = append(templateLines, templateContent)
				}
			} else {
				// Single-line template
				template = strings.TrimPrefix(trimmed, "TEMPLATE ")
				template = strings.Trim(template, `"`)
			}
		} else if inTemplate {
			if inMultilineTemplate && strings.HasSuffix(trimmed, `"""`) {
				line = strings.TrimSuffix(line, `"""`)
				if line != "" {
					templateLines = append(templateLines, line)
				}
				inTemplate = false
				inMultilineTemplate = false
			} else {
				templateLines = append(templateLines, line)
			}
		} else if strings.HasPrefix(trimmed, "SYSTEM") {
			system = strings.TrimPrefix(trimmed, "SYSTEM ")
			// Remove surrounding quotes if present
			system = strings.Trim(system, `"`)
		} else if strings.HasPrefix(trimmed, "PARAMETER") {
			parts := strings.SplitN(trimmed, " ", 3)
			if len(parts) >= 3 {
				key := parts[1]
				value := strings.TrimSpace(parts[2])
				params[key] = value
			}
		}
	}

	if len(templateLines) > 0 {
		template = strings.Join(templateLines, "\n")
	}

	return params, template, system, nil
}

// extractParameters extracts parameter values from modelfile content
func extractParameters(content string) map[string]any {
	parameters := make(map[string]any)
	lines := strings.Split(content, "\n")

	// Handle multiple stop parameters
	var stopValues []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "PARAMETER") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				paramName := parts[1]
				paramValue := strings.Join(parts[2:], " ")

				// Special handling for stop parameters
				if paramName == "stop" {
					stopValues = append(stopValues, paramValue)
					continue
				}

				// Convert numeric values to appropriate types
				if num, err := strconv.Atoi(paramValue); err == nil {
					parameters[paramName] = num
				} else if num, err := strconv.ParseFloat(paramValue, 64); err == nil {
					parameters[paramName] = num
				} else {
					parameters[paramName] = paramValue
				}
			}
		}
	}

	// Add stop values as an array if any were found
	if len(stopValues) > 0 {
		parameters["stop"] = stopValues
	}

	return parameters
}
