// operations.go contains the functions that perform the operations on the models.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ollama/ollama/api"
	"github.com/sammcj/gollama/config"
	"github.com/sammcj/gollama/logging"
)

func runModel(model string, cfg *config.Config) tea.Cmd {
	// if config is set to run in docker container, run the mode using runDocker
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

	// parse the params into a list of arguments to supply to docker exec
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

func (m *AppModel) startPushModel(modelName string) tea.Cmd {
	logging.InfoLogger.Printf("Pushing model: %s\n", modelName)

	// Initialize the progress model
	m.progress = progress.New(progress.WithDefaultGradient())

	return tea.Batch(
		tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
			return progressMsg{modelName: modelName}
		}),
		m.pushModelCmd(modelName),
	)
}

func (m *AppModel) startPullModel(modelName string) tea.Cmd {
	logging.InfoLogger.Printf("Pulling model: %s\n", modelName)
	return m.pullModelCmd(modelName)
}

func (m *AppModel) pushModelCmd(modelName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		req := &api.PushRequest{Name: modelName}
		err := m.client.Push(ctx, req, func(resp api.ProgressResponse) error {
			m.progress.SetPercent(float64(resp.Completed) / float64(resp.Total))
			return nil
		})
		if err != nil {
			return pushErrorMsg{err}
		}
		return pushSuccessMsg{modelName}
	}
}

func (m *AppModel) pullModelCmd(modelName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		req := &api.PullRequest{Name: modelName}
		err := m.client.Pull(ctx, req, func(resp api.ProgressResponse) error {
			m.progress.SetPercent(float64(resp.Completed) / float64(resp.Total))
			return nil
		})
		// If the progress is 100%, the model has been successfully pulled, return a success message
		if err == nil {
			return pullSuccessMsg{modelName}
		}
		// If the progress is not 100%, the model has not been successfully pulled, return an error message
		return pullErrorMsg{err}
	}
}

func linkModel(modelName, lmStudioModelsDir string, noCleanup bool, client *api.Client) (string, error) {
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

	// Check if the model is already symlinked in another location
	var existingSymlinkPath string
	err = filepath.Walk(lmStudioModelsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			linkPath, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if linkPath == modelPath {
				existingSymlinkPath = path
				return nil
			}
		}
		return nil
	})
	if err != nil {
		message := "error walking LM Studio models directory: %v"
		logging.ErrorLogger.Printf(message+"\n", err)
		return "", fmt.Errorf(message, err)
	}

	if existingSymlinkPath != "" {
		// Remove the duplicated model directory
		err = os.RemoveAll(lmStudioModelDir)
		if err != nil {
			message := "failed to remove duplicated model directory %s: %v"
			logging.ErrorLogger.Printf(message+"\n", lmStudioModelDir, err)
			return "", fmt.Errorf(message, lmStudioModelDir, err)
		}
		return fmt.Sprintf("Removed duplicated model directory %s", lmStudioModelDir), nil
	}

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
	message := "failed to get model path for %s: no 'FROM' line in output"
	logging.ErrorLogger.Printf(message+"\n", modelName)
	return "", fmt.Errorf(message, modelName)
}

func getModelParams(modelName string, client *api.Client) (map[string][]string, error) {
	logging.InfoLogger.Printf("Getting parameters for model: %s\n", modelName)
	ctx := context.Background()
	req := &api.ShowRequest{Name: modelName}
	resp, err := client.Show(ctx, req)
	if err != nil {
		logging.ErrorLogger.Printf("Error getting parameters for model %s: %v\n", modelName, err)
		return nil, err
	}
	output := []byte(resp.Modelfile)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	// loop through all lines and for each line containing PARAMETER <key> <value> add the key value pair to the map
	params := make(map[string][]string)
	for _, line := range lines {
		if strings.HasPrefix(line, "PARAMETER") {
			logging.DebugLogger.Printf("Found parameter line: %s\n", line)
			parts := strings.Split(line, " ")
			key := parts[1]
			value := strings.Join(parts[2:], " ")
			params[key] = append(params[key], value)
			logging.DebugLogger.Printf("Added parameter: %s: %s\n", key, value)
		}
	}
	return params, nil
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

	// Although the model has been copied, the model list has not been updated as the API does not return the new model which is annoying
	resp, err := client.List(ctx)
	if err != nil {
		logging.ErrorLogger.Printf("Error fetching models: %v\n", err)
		return
	}
	m.models = parseAPIResponse(resp)
	m.refreshList()

}

// A function that returns a list of models that contain a search term (case insensitive) in their name, for use by the cli flag -s
func searchModels(models []Model, searchTerms ...string) {
	logging.InfoLogger.Printf("Searching for models with terms: %v\n", searchTerms)

	var searchResults []Model
	for _, model := range models {
		if containsAllTerms(model.Name, searchTerms...) {
			searchResults = append(searchResults, model)
		}
	}

	// Sort the results alphabetically case insensitive
	sort.Slice(searchResults, func(i, j int) bool {
		return strings.ToLower(searchResults[i].Name) < strings.ToLower(searchResults[j].Name)
	})

	// Define adaptive styles
	baseStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#5000D3", Dark: "#FF60FF"})
	highlightStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"})
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#AAEE9A"})

	// Colorize the matching parts of the model name
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

// Helper function to check if a string contains all search terms
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

// A renameModel function that takes a selected model, prompts for a new name then calls copyModel, then deleteModel
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

// Adding a new function get use client to get the running models
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

func copyModelfile(modelName, newModelName string, client *api.Client) (string, error) {
	logging.InfoLogger.Printf("Copying modelfile for model: %s\n", modelName)

	ctx := context.Background()
	req := &api.ShowRequest{Name: modelName}
	resp, err := client.Show(ctx, req)
	if err != nil {
		logging.ErrorLogger.Printf("Error copying modelfile for model %s: %v\n", modelName, err)
		return "", err
	}

	output := []byte(resp.Modelfile)

	err = os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".config", "gollama", "modelfiles"), os.ModePerm)
	if err != nil {
		logging.ErrorLogger.Printf("Error creating modelfiles directory: %v\n", err)
		return "", err
	}

	// replace any slashes, colons, or underscores with dashes
	newModelName = strings.ReplaceAll(newModelName, "/", "-")
	newModelName = strings.ReplaceAll(newModelName, ":", "-")

	newModelfilePath := filepath.Join(os.Getenv("HOME"), ".config", "gollama", "modelfiles", newModelName+".modelfile")

	err = os.WriteFile(newModelfilePath, output, 0644)
	if err != nil {
		logging.ErrorLogger.Printf("Error writing modelfile for model %s: %v\n", modelName, err)
		return "", err
	}
	logging.InfoLogger.Printf("Copied modelfile to: %s\n", newModelfilePath)

	return newModelfilePath, nil
}

type editorFinishedMsg struct{ err error }

func openEditor(filePath string) tea.Cmd {
	logging.DebugLogger.Printf("Opening editor for file: %s\n", filePath)
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	c := exec.Command(editor, filePath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err}
	})
}

func createModelFromModelfile(modelName, modelfilePath string, client *api.Client) error {
	ctx := context.Background()
	req := &api.CreateRequest{
		Name:      modelName,
		Modelfile: modelfilePath,
	}
	// TODO: complete progress bar
	// progressResponse := func(resp api.ProgressResponse) error {
	// 	logging.DebugLogger.Printf("Progress: %d/%d\n", resp.Completed, resp.Total)
	// 	progress := progress.New(progress.WithDefaultGradient())
	// 	// update the progress bar
	// 	progress.SetPercent(float64(resp.Completed) / float64(resp.Total))
	// 	// render the progress bar
	// 	fmt.Println(progress.View())

	// 	return nil
	// }
	err := client.Create(ctx, req, nil) //TODO: add working progress bar
	if err != nil {
		logging.ErrorLogger.Printf("Error creating model from modelfile %s: %v\n", modelfilePath, err)
		return fmt.Errorf("error creating model from modelfile %s: %v", modelfilePath, err)
	}
	logging.InfoLogger.Printf("Successfully created model from modelfile: %s\n", modelfilePath)
	return nil

}

func unloadModel(client *api.Client, modelName string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("invalid API client: client is nil")
	}

	ctx := context.Background()

	// if the model is an embedding model, we can't call generaterequest on it, we have to call embeddingrequest
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

	if os.Getenv("EDITOR") == "" {
		os.Setenv("EDITOR", "vim")
	}

	logging.DebugLogger.Printf("Editing modelfile for model: %s\n", modelName)

	// Write the fetched content to a temporary file
	tempDir := os.TempDir()
	newModelfilePath := filepath.Join(tempDir, fmt.Sprintf("%s_modelfile.txt", modelName))
	err = os.WriteFile(newModelfilePath, []byte(modelfileContent), 0644)
	if err != nil {
		return "", fmt.Errorf("error writing modelfile to temp file: %v", err)
	}
	defer os.Remove(newModelfilePath)

	// Open the local modelfile in the editor
	cmd := exec.Command(os.Getenv("EDITOR"), newModelfilePath)
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

	// Update the model on the server with the new modelfile content
	createReq := &api.CreateRequest{
		Name:      modelName,
		Modelfile: string(newModelfileContent),
	}

	err = client.Create(ctx, createReq, func(resp api.ProgressResponse) error {
		logging.InfoLogger.Printf("Create progress: %s\n", resp.Status)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("error updating model with new modelfile: %v", err)
	}

	// log to the console if we're not in a tea app
	fmt.Printf("Model %s updated successfully\n", modelName)

	return fmt.Sprintf("Model %s updated successfully, Press 'q' to return to the models list", modelName), nil
}

func isLocalhost(url string) bool {
	return strings.Contains(url, "localhost") || strings.Contains(url, "127.0.0.1")
}
