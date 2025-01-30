package lmstudio

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/sammcj/gollama/logging"
	"github.com/sammcj/gollama/utils"
)

type Model struct {
	Name     string
	Path     string
	FileType string // e.g., "gguf", "bin", etc.
}

// ModelfileTemplate contains the default template for creating Modelfiles
const ModelfileTemplate = `FROM {{.ModelPath}}
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
`

type ModelfileData struct {
	ModelPath string
	Prompt    string
}

// ScanModels scans the given directory for LM Studio model files
func ScanModels(dirPath string) ([]Model, error) {
	var models []Model

	// First check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("LM Studio models directory does not exist: %s", dirPath)
	}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, walkErr error) error {
		// Handle walk errors immediately
		if walkErr != nil {
			logging.ErrorLogger.Printf("Error accessing path %s: %v", path, walkErr)
			return walkErr
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check for model file extensions
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".gguf" || ext == ".bin" {
			name := strings.TrimSuffix(filepath.Base(path), ext)

			// Basic name validation
			if strings.ContainsAny(name, "/\\:*?\"<>|") {
				logging.ErrorLogger.Printf("Skipping model with invalid characters in name: %s", name)
				return nil
			}

			model := Model{
				Name:     name,
				Path:     path,
				FileType: strings.TrimPrefix(ext, "."),
			}

			logging.DebugLogger.Printf("Found model: %s (%s)", model.Name, model.FileType)
			models = append(models, model)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error scanning directory %s: %w", dirPath, err)
	}

	if len(models) == 0 {
		logging.InfoLogger.Printf("No models found in directory: %s", dirPath)
	} else {
		logging.InfoLogger.Printf("Found %d models in directory: %s", len(models), dirPath)
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
		logging.InfoLogger.Printf("Modelfile already exists for %s, skipping creation", modelName)
		return nil
	}

	tmpl, err := template.New("modelfile").Parse(ModelfileTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse Modelfile template: %w", err)
	}

	data := ModelfileData{
		ModelPath: filepath.Base(modelPath),
		Prompt:    "{{.Prompt}}", // Preserve this as a template variable for Ollama
	}

	file, err := os.OpenFile(modelfilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create Modelfile: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to write Modelfile template: %w", err)
	}

	return nil
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
		if os.IsExist(err) {
			logging.InfoLogger.Printf("Symlink already exists for %s at %s", model.Name, targetPath)
		} else {
			return fmt.Errorf("failed to create symlink for %s to %s: %w", model.Name, targetPath, err)
		}
	}

	// Check if model is already registered with Ollama
	if modelExists(model.Name) {
		return nil
	}

	// Create model-specific Modelfile
	modelfilePath := filepath.Join(filepath.Dir(targetPath), fmt.Sprintf("Modelfile.%s", model.Name))
	if err := createModelfile(model.Name, targetPath); err != nil {
		return fmt.Errorf("failed to create Modelfile for %s: %w", model.Name, err)
	}

	// Create the model in Ollama
	cmd := exec.Command("ollama", "create", model.Name, "-f", modelfilePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up Modelfile on failure
		os.Remove(modelfilePath)
		return fmt.Errorf("failed to create Ollama model %s: %s - %w", model.Name, string(output), err)
	}

	// Clean up the Modelfile after successful creation
	if err := os.Remove(modelfilePath); err != nil {
		logging.ErrorLogger.Printf("Warning: Could not remove temporary Modelfile %s: %v", modelfilePath, err)
	}

	return nil
}
