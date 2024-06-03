package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ollama/ollama/api"
	"github.com/sammcj/gollama/logging"
)

func runModel(model string) tea.Cmd {
	ollamaPath, err := exec.LookPath("ollama")
	if err != nil {
		logging.ErrorLogger.Printf("Error finding ollama binary: %v\n", err)
		return nil
	}
	c := exec.Command(ollamaPath, "run", model)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			logging.ErrorLogger.Printf("Error running model: %v\n", err)
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

func (m *AppModel) startPushModel(modelName string) (tea.Model, tea.Cmd) {
	logging.InfoLogger.Printf("Pushing model: %s\n", modelName)

	// Initialize the progress model
	m.progress = progress.New(progress.WithDefaultGradient())

	return m, tea.Batch(
		tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
			return progressMsg{modelName: modelName}
		}),
		m.pushModelCmd(modelName),
	)
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

func linkModel(modelName, lmStudioModelsDir string, noCleanup bool) (string, error) {
	modelPath, err := getModelPath(modelName)
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

func getModelPath(modelName string) (string, error) {
	cmd := exec.Command("ollama", "show", "--modelfile", modelName)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
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

func copyModel(client *api.Client, oldName string, newName string) {
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

	// Push the new model to the Ollama API
	err = pushModel(client, newName)
	if err != nil {
		logging.ErrorLogger.Printf("Error pushing model: %v\n", err)
	}
}

func pushModel(client *api.Client, modelName string) error {
	ctx := context.Background()
	req := &api.PushRequest{Name: modelName}
	err := client.Push(ctx, req, func(resp api.ProgressResponse) error {
		return nil
	})
	if err != nil {
		return fmt.Errorf("error pushing model: %w", err)
	}
	logging.InfoLogger.Printf("Successfully pushed model: %s\n", modelName)
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
	}

	return runningModels, nil
}
