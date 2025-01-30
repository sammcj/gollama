package lmstudio

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sammcj/gollama/logging"
	"github.com/sammcj/gollama/utils"
)

type Model struct {
	Name     string
	Path     string
	FileType string // e.g., "gguf", "bin", etc.
}

// ScanModels scans the given directory for LM Studio model files
func ScanModels(dirPath string) ([]Model, error) {
	var models []Model

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check for model file extensions
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".gguf" || ext == ".bin" {
			name := strings.TrimSuffix(filepath.Base(path), ext)
			models = append(models, Model{
				Name:     name,
				Path:     path,
				FileType: strings.TrimPrefix(ext, "."),
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error scanning directory: %w", err)
	}

	return models, nil
}

// GetOllamaModelDir returns the default Ollama models directory for the current OS
func GetOllamaModelDir() string {
	homeDir := utils.GetHomeDir()
	if runtime.GOOS == "darwin" {
		return filepath.Join(homeDir, ".ollama", "models")
	} else if runtime.GOOS == "linux" {
		return "/usr/share/ollama/models"
	}
	// Add Windows path if needed
	return filepath.Join(homeDir, ".ollama", "models")
}

// modelExists checks if a model is already registered with Ollama
func modelExists(modelName string) bool {
	cmd := exec.Command("ollama", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), modelName)
}

// createModelfile creates a Modelfile for the given model
func createModelfile(modelName string, modelPath string) error {
	modelfilePath := filepath.Join(filepath.Dir(modelPath), fmt.Sprintf("Modelfile.%s", modelName))

	// Check if Modelfile already exists
	if _, err := os.Stat(modelfilePath); err == nil {
		return nil
	}

	modelfileContent := fmt.Sprintf(`FROM %s
PARAMETER temperature 0.7
PARAMETER top_k 40
PARAMETER top_p 0.4
PARAMETER repeat_penalty 1.1
PARAMETER repeat_last_n 64
PARAMETER seed 0
PARAMETER stop "Human:" "Assistant:"
TEMPLATE """
{{.Prompt}}
Assistant: """
SYSTEM """You are a helpful AI assistant."""
`, filepath.Base(modelPath))

	return os.WriteFile(modelfilePath, []byte(modelfileContent), 0644)
}

// LinkModelToOllama links an LM Studio model to Ollama
func LinkModelToOllama(model Model) error {
	ollamaDir := GetOllamaModelDir()

	// Create Ollama models directory if it doesn't exist
	if err := os.MkdirAll(ollamaDir, 0755); err != nil {
		return fmt.Errorf("failed to create Ollama models directory: %w", err)
	}

	targetPath := filepath.Join(ollamaDir, filepath.Base(model.Path))

	// Create symlink for model file
	if err := os.Symlink(model.Path, targetPath); err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to create symlink: %w", err)
		}
	}

	// Check if model is already registered with Ollama
	if modelExists(model.Name) {
		return nil
	}

	// Create model-specific Modelfile
	modelfilePath := filepath.Join(filepath.Dir(targetPath), fmt.Sprintf("Modelfile.%s", model.Name))
	if err := createModelfile(model.Name, targetPath); err != nil {
		return fmt.Errorf("failed to create Modelfile: %w", err)
	}

	cmd := exec.Command("ollama", "create", model.Name, "-f", modelfilePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create Ollama model: %s\n%w", string(output), err)
	}

	// Clean up the Modelfile after successful creation
	if err := os.Remove(modelfilePath); err != nil {
		logging.ErrorLogger.Printf("Warning: Could not remove temporary Modelfile %s: %v\n", modelfilePath, err)
	}

	return nil
}
