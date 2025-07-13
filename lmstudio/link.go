package lmstudio

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
// TODO: Make the default Modelfile template configurable
const ModelfileTemplate = `### MODEL IMPORTED FROM LM-STUDIO BY GOLLAMA ###

# Tune the below inference, model load parameters and template to your needs
# The template and stop parameters are currently set to the default for models that use the ChatML format
# If required update these match the prompt format your model expects
# You can look at existing similar models on the Ollama model hub for examples
# See https://github.com/ollama/ollama/blob/main/docs/modelfile.md for a complete reference

FROM {{.ModelPath}}

### Model Load Parameters ###
PARAMETER num_ctx 4096

### Inference Parameters ####
PARAMETER temperature 0.4
PARAMETER top_p 0.6

### Chat Template Parameters ###

TEMPLATE """
{{.Prompt}}
"""

PARAMETER stop "<|im_start|>"
PARAMETER stop "<|im_end|>"
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

// modelExists checks if a model is already registered with Ollama
func modelExists(modelName string) bool {
	cmd := exec.Command("ollama", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), modelName)
}

// generateModelfileContent generates the Modelfile content as a string
func generateModelfileContent(modelName string, modelPath string) (string, error) {
	tmpl, err := template.New("modelfile").Parse(ModelfileTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse Modelfile template: %w", err)
	}

	data := ModelfileData{
		ModelPath: modelPath,     // Use full path instead of just the base name
		Prompt:    "{{.Prompt}}", // Preserve this as a template variable for Ollama
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute Modelfile template: %w", err)
	}

	return buf.String(), nil
}

// createModelfile creates a Modelfile for the given model
func createModelfile(modelName string, modelPath string) error {
	modelfilePath := filepath.Join(filepath.Dir(modelPath), fmt.Sprintf("Modelfile.%s", modelName))

	// Check if Modelfile already exists
	if _, err := os.Stat(modelfilePath); err == nil {
		logging.InfoLogger.Printf("Modelfile already exists for %s, skipping creation", modelName)
		return nil
	}

	// Generate Modelfile content using the helper function
	content, err := generateModelfileContent(modelName, modelPath)
	if err != nil {
		return fmt.Errorf("failed to generate Modelfile content: %w", err)
	}

	logging.DebugLogger.Printf("Creating Modelfile at: %s with model path: %s", modelfilePath, modelPath)

	// Write the content to file
	if err := os.WriteFile(modelfilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write Modelfile: %w", err)
	}

	// Log the content of the created Modelfile for debugging
	logging.DebugLogger.Printf("Created Modelfile content:\n%s", content)

	return nil
}

// LinkModelToOllama links an LM Studio model to Ollama
// If dryRun is true, it will only print what would happen without making any changes
func LinkModelToOllama(model Model, dryRun bool, ollamaHost string, ollamaDir string) error {
	// Check if we're connecting to a local Ollama instance
	if !utils.IsLocalhost(ollamaHost) {
		return fmt.Errorf("linking LM Studio models to Ollama is only supported when connecting to a local Ollama instance (got %s)", ollamaHost)
	}

	if dryRun {
		fmt.Printf("[DRY RUN] Would create Ollama models directory at: %s\n", ollamaDir)
	} else if err := os.MkdirAll(ollamaDir, 0755); err != nil {
		return fmt.Errorf("failed to create Ollama models directory: %w", err)
	}

	targetPath := filepath.Join(ollamaDir, filepath.Base(model.Path))

	if dryRun {
		fmt.Printf("[DRY RUN] Would create symlink from %s to %s\n", model.Path, targetPath)
	} else if err := os.Symlink(model.Path, targetPath); err != nil {
		if os.IsExist(err) {
			logging.InfoLogger.Printf("Symlink already exists for %s at %s", model.Name, targetPath)
		} else {
			return fmt.Errorf("failed to create symlink for %s to %s: %w", model.Name, targetPath, err)
		}
	}

	// Check if model is already registered with Ollama
	if !dryRun && modelExists(model.Name) {
		return nil
	}

	// Create model-specific Modelfile
	modelfilePath := filepath.Join(filepath.Dir(targetPath), fmt.Sprintf("Modelfile.%s", model.Name))
	if dryRun {
		fmt.Printf("\n\n[DRY RUN] *** Would create Modelfile at: %s ***\n", modelfilePath)

		// Generate and display the Modelfile content that would be created
		modelfileContent, err := generateModelfileContent(model.Name, targetPath)
		if err != nil {
			fmt.Printf("[DRY RUN] Error generating Modelfile content: %v\n", err)
		} else {
			fmt.Printf("[DRY RUN] Modelfile content that would be created:\n")
			fmt.Printf("--- BEGIN MODELFILE ---\n")
			fmt.Printf("%s", modelfileContent)
			fmt.Printf("--- END MODELFILE ---\n")
		}

		fmt.Printf("[DRY RUN] Would create Ollama model: %s using Modelfile\n", model.Name)
		return nil
	}

	if err := createModelfile(model.Name, targetPath); err != nil {
		return fmt.Errorf("failed to create Modelfile for %s: %w", model.Name, err)
	}

	// Create the model in Ollama
	logging.DebugLogger.Printf("Creating Ollama model %s using Modelfile at: %s", model.Name, modelfilePath)
	cmd := exec.Command("ollama", "create", model.Name, "-f", modelfilePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Log the error output for debugging
		logging.ErrorLogger.Printf("Ollama create command output: %s", string(output))
		// Clean up Modelfile on failure
		os.Remove(modelfilePath)
		return fmt.Errorf("failed to create Ollama model %s: %s - %w", model.Name, string(output), err)
	}
	logging.DebugLogger.Printf("Successfully created Ollama model %s", model.Name)

	// Clean up the Modelfile after successful creation
	if err := os.Remove(modelfilePath); err != nil {
		logging.ErrorLogger.Printf("Warning: Could not remove temporary Modelfile %s: %v", modelfilePath, err)
	}

	return nil
}
