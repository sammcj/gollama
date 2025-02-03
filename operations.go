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
	"github.com/sammcj/gollama/config"
	"github.com/sammcj/gollama/logging"
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
	logging.InfoLogger.Printf(message)
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
	if editor == "" {
		editor = "vim" // Default fallback
	}

	logging.DebugLogger.Printf("Using editor: %s for model: %s\n", editor, modelName)

	// Write the fetched content to a temporary file
	tempDir := os.TempDir()
	newModelfilePath := filepath.Join(tempDir, fmt.Sprintf("%s_modelfile.txt", modelName))
	err = os.WriteFile(newModelfilePath, []byte(modelfileContent), 0644)
	if err != nil {
		return "", fmt.Errorf("error writing modelfile to temp file: %v", err)
	}
	defer os.Remove(newModelfilePath)

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
		Model: modelName,    // The model to update
		From:  modelName,    // Required: use the model's own name as the base
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

	return fmt.Sprintf("Model %s updated successfully, Press 'q' to return to the models list", modelName), nil
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
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		logging.ErrorLogger.Printf("Error loading config for editor: %v\n", err)
		return ""
	}

	return cfg.Editor
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

func cleanBrokenSymlinks(lmStudioModelsDir string) {
	err := filepath.Walk(lmStudioModelsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			files, err := os.ReadDir(path)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				logging.InfoLogger.Printf("Removing empty directory: %s\n", path)
				err = os.Remove(path)
				if err != nil {
					return err
				}
			}
		} else if info.Mode()&os.ModeSymlink != 0 {
			linkPath, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if !isValidSymlink(path, linkPath) {
				logging.InfoLogger.Printf("Removing invalid symlink: %s\n", path)
				err = os.Remove(path)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		logging.ErrorLogger.Printf("Error walking LM Studio models directory: %v\n", err)
		return
	}
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

type editorFinishedMsg struct{ err error }

func cleanupSymlinkedModels(lmStudioModelsDir string) {
	for {
		hasEmptyDir := false
		err := filepath.Walk(lmStudioModelsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				files, err := os.ReadDir(path)
				if err != nil {
					return err
				}
				if len(files) == 0 {
					logging.InfoLogger.Printf("Removing empty directory: %s\n", path)
					err = os.Remove(path)
					if err != nil {
						return err
					}
					hasEmptyDir = true
				}
			} else if info.Mode()&os.ModeSymlink != 0 {
				logging.InfoLogger.Printf("Removing symlinked model: %s\n", path)
				err = os.Remove(path)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			logging.ErrorLogger.Printf("Error walking LM Studio models directory: %v\n", err)
			return
		}
		if !hasEmptyDir {
			break
		}
	}
}

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

	baseStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#5000D3", Dark: "#FF60FF"})
	highlightStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"})
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#AAEE9A"})

	for i, model := range searchResults {
		colorizedName := model.Name
		for _, term := range searchTerms {
			andTerms := strings.Split(term, "&")
			colorizedName = highlightTerms(colorizedName, baseStyle, highlightStyle, andTerms)
		}
		searchResults[i].Name = colorizedName
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

func linkModel(modelName, lmStudioModelsDir string, noCleanup bool, dryRun bool, client *api.Client) (string, error) {
	modelPath, err := getModelPath(modelName, client)
	if err != nil {
		return "", fmt.Errorf("error getting model path for %s: %v", modelName, err)
	}

	parts := strings.Split(modelName, ":")
	author := "unknown"
	if len(parts) > 1 {
		author = strings.ReplaceAll(parts[0], "/", "-")
	}

	lmStudioModelName := strings.ReplaceAll(strings.ReplaceAll(modelName, ":", "-"), "_", "-")
	lmStudioModelDir := filepath.Join(lmStudioModelsDir, author, lmStudioModelName+"-GGUF")

	// Check if the model path is a valid file
	fileInfo, err := os.Stat(modelPath)
	if err != nil || fileInfo.IsDir() {
		return "", fmt.Errorf("invalid model path for %s: %s", modelName, modelPath)
	}

	// Check if the symlink already exists and is valid
	lmStudioModelPath := filepath.Join(lmStudioModelDir, filepath.Base(lmStudioModelName)+".gguf")
	if _, err := os.Lstat(lmStudioModelPath); err == nil {
		if isValidSymlink(lmStudioModelPath, modelPath) {
			message := "Model %s is already symlinked to %s"
			logging.InfoLogger.Printf(message+"\n", modelName, lmStudioModelPath)
			return "", nil
		}
		// Remove the invalid symlink
		err = os.Remove(lmStudioModelPath)
		if err != nil {
			message := "failed to remove invalid symlink %s: %v"
			logging.ErrorLogger.Printf(message+"\n", lmStudioModelPath, err)
			return "", fmt.Errorf(message, lmStudioModelPath, err)
		}
	}

	if dryRun {
		message := "[DRY RUN] Would create directory %s and symlink %s to %s"
		logging.InfoLogger.Printf(message+"\n", lmStudioModelDir, modelName, lmStudioModelPath)
		return fmt.Sprintf(message, lmStudioModelDir, modelName, lmStudioModelPath), nil
	} else {
		// Create the symlink
		err = os.MkdirAll(lmStudioModelDir, os.ModePerm)
		if err != nil {
			message := "failed to create directory %s: %v"
			logging.ErrorLogger.Printf(message+"\n", lmStudioModelDir, err)
			return "", fmt.Errorf(message, lmStudioModelDir, err)
		}
		err = os.Symlink(modelPath, lmStudioModelPath)
		if err != nil {
			message := "failed to symlink %s: %v"
			logging.ErrorLogger.Printf(message+"\n", modelName, err)
			return "", fmt.Errorf(message, modelName, err)
		}
		if !noCleanup {
			cleanBrokenSymlinks(lmStudioModelsDir)
		}
		message := "Symlinked %s to %s"
		logging.InfoLogger.Printf(message+"\n", modelName, lmStudioModelPath)
		return "", nil
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
				inTemplate = true
				inMultilineTemplate = true
				templateLines = append(templateLines, strings.TrimPrefix(trimmed, "TEMPLATE "))
			} else {
				// Single-line template
				template = strings.TrimPrefix(trimmed, "TEMPLATE ")
				template = strings.Trim(template, `"`)
			}
		} else if inTemplate {
			if inMultilineTemplate && strings.Contains(trimmed, `"""`) {
				inTemplate = false
				inMultilineTemplate = false
			}
			templateLines = append(templateLines, line)
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
