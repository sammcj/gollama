package lmstudio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ollama/ollama/api"
	"github.com/sammcj/gollama/logging"
	"github.com/sammcj/gollama/utils"
)

// LMStudioModel represents an LM Studio model with enhanced metadata
type LMStudioModel struct {
	Name        string
	Path        string
	FileType    string
	VisionFiles []string // mmproj files for vision models
	IsSymlinked bool     // Skip if already linked from Ollama
	Publisher   string   // Extract from directory structure
	ModelDir    string   // Publisher/model directory path
	Size        int64    // File size in bytes
}

// ModelConfig contains configuration parameters for the model
type ModelConfig struct {
	NumCtx      int     `json:"num_ctx"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
	MinP        float64 `json:"min_p"`
}

// OllamaManifest represents the OCI-compliant manifest structure
type OllamaManifest struct {
	SchemaVersion int             `json:"schemaVersion"`
	MediaType     string          `json:"mediaType"`
	Config        ManifestConfig  `json:"config"`
	Layers        []ManifestLayer `json:"layers"`
}

// ManifestConfig represents the config section of the manifest
type ManifestConfig struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

// ManifestLayer represents a layer in the manifest
type ManifestLayer struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

const (
	// Media types for different layer types
	MediaTypeModel     = "application/vnd.ollama.image.model"
	MediaTypeParams    = "application/vnd.ollama.image.params"
	MediaTypeTemplate  = "application/vnd.ollama.image.template"
	MediaTypeSystem    = "application/vnd.ollama.image.system"
	MediaTypeProjector = "application/vnd.ollama.image.projector"
	MediaTypeLicense   = "application/vnd.ollama.image.license"

	// Manifest media type
	ManifestMediaType = "application/vnd.docker.distribution.manifest.v2+json"
)

// Default configuration values
var defaultConfig = ModelConfig{
	NumCtx:      16384,
	Temperature: 0.6,
	TopP:        0.95,
	MinP:        0.01,
}

// calculateSHA256 calculates the SHA256 hash of a file
func calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate hash for %s: %w", filePath, err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// isSymlink checks if a file is a symbolic link
func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// extractPublisherAndModel extracts publisher and model name from LM Studio path structure
func extractPublisherAndModel(modelPath, lmStudioDir string) (string, string, error) {
	// Remove the LM Studio models directory prefix
	relPath, err := filepath.Rel(lmStudioDir, modelPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to get relative path: %w", err)
	}

	// Split the path to get publisher/model structure
	parts := strings.Split(filepath.Dir(relPath), string(filepath.Separator))
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid LM Studio model path structure: %s", relPath)
	}

	publisher := parts[0]
	model := parts[1]

	// Strip common suffixes like -GGUF, -gguf, etc.
	model = strings.TrimSuffix(model, "-GGUF")
	model = strings.TrimSuffix(model, "-gguf")
	model = strings.TrimSuffix(model, "-Gguf")

	return publisher, model, nil
}

// isVisionModel checks if a model directory contains vision projection files
func isVisionModel(modelDir string) (bool, []string, error) {
	var visionFiles []string

	err := filepath.Walk(modelDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if info.IsDir() {
			return nil
		}

		fileName := strings.ToLower(filepath.Base(path))
		// Look for mmproj files (vision projection files)
		if strings.Contains(fileName, "mmproj") && strings.HasSuffix(fileName, ".gguf") {
			visionFiles = append(visionFiles, path)
		}

		return nil
	})

	if err != nil {
		return false, nil, fmt.Errorf("failed to scan for vision files: %w", err)
	}

	return len(visionFiles) > 0, visionFiles, nil
}

// ScanUnlinkedModels scans for LM Studio models that are not symlinked from Ollama
func ScanUnlinkedModels(lmStudioDir string) ([]LMStudioModel, error) {
	var models []LMStudioModel
	seenDirs := make(map[string]bool)

	// First check if directory exists
	if _, err := os.Stat(lmStudioDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("LM Studio models directory does not exist: %s", lmStudioDir)
	}

	err := filepath.Walk(lmStudioDir, func(path string, info os.FileInfo, walkErr error) error {
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
			// Skip if this is a symlink (already linked from Ollama)
			if isSymlink(path) {
				logging.DebugLogger.Printf("Skipping symlinked file: %s", path)
				return nil
			}

			// Skip mmproj files as they'll be handled as vision files
			if strings.Contains(strings.ToLower(filepath.Base(path)), "mmproj") {
				return nil
			}

			name := strings.TrimSuffix(filepath.Base(path), ext)

			// Basic name validation
			if strings.ContainsAny(name, "/\\:*?\"<>|") {
				logging.ErrorLogger.Printf("Skipping model with invalid characters in name: %s", name)
				return nil
			}

			// Extract publisher and model information
			publisher, modelName, err := extractPublisherAndModel(path, lmStudioDir)
			if err != nil {
				logging.ErrorLogger.Printf("Skipping model with invalid path structure: %s (%v)", path, err)
				return nil
			}

			// Check for vision model files
			modelDir := filepath.Dir(path)

			// Skip if we've already processed this model directory
			modelKey := fmt.Sprintf("%s/%s", publisher, modelName)
			if seenDirs[modelKey] {
				logging.DebugLogger.Printf("Skipping duplicate model in same directory: %s", path)
				return nil
			}
			seenDirs[modelKey] = true

			isVision, visionFiles, err := isVisionModel(modelDir)
			if err != nil {
				logging.ErrorLogger.Printf("Error checking for vision files in %s: %v", modelDir, err)
				// Continue processing as non-vision model
			}

			model := LMStudioModel{
				Name:        modelKey,
				Path:        path,
				FileType:    strings.TrimPrefix(ext, "."),
				VisionFiles: visionFiles,
				IsSymlinked: false,
				Publisher:   publisher,
				ModelDir:    modelDir,
				Size:        info.Size(),
			}

			if isVision {
				logging.DebugLogger.Printf("Found vision model: %s with %d vision files", model.Name, len(visionFiles))
			} else {
				logging.DebugLogger.Printf("Found model: %s (%s)", model.Name, model.FileType)
			}

			models = append(models, model)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error scanning directory %s: %w", lmStudioDir, err)
	}

	if len(models) == 0 {
		logging.InfoLogger.Printf("No unlinked models found in directory: %s", lmStudioDir)
	} else {
		logging.InfoLogger.Printf("Found %d unlinked models in directory: %s", len(models), lmStudioDir)
	}

	return models, nil
}

// generateManifest creates an OCI-compliant manifest for the model
func generateManifest(model LMStudioModel, hashes map[string]string, config ModelConfig) (OllamaManifest, error) {
	var layers []ManifestLayer

	// Add main model layer
	mainHash, exists := hashes[model.Path]
	if !exists {
		return OllamaManifest{}, fmt.Errorf("missing hash for main model file: %s", model.Path)
	}

	layers = append(layers, ManifestLayer{
		MediaType: MediaTypeModel,
		Size:      model.Size,
		Digest:    fmt.Sprintf("sha256:%s", mainHash),
	})

	// Add vision projection layers if this is a vision model
	for _, visionFile := range model.VisionFiles {
		visionHash, exists := hashes[visionFile]
		if !exists {
			return OllamaManifest{}, fmt.Errorf("missing hash for vision file: %s", visionFile)
		}

		visionInfo, err := os.Stat(visionFile)
		if err != nil {
			return OllamaManifest{}, fmt.Errorf("failed to get vision file info: %w", err)
		}

		layers = append(layers, ManifestLayer{
			MediaType: MediaTypeProjector,
			Size:      visionInfo.Size(),
			Digest:    fmt.Sprintf("sha256:%s", visionHash),
		})
	}

	// Create config layer (contains model parameters)
	configData, err := json.Marshal(config)
	if err != nil {
		return OllamaManifest{}, fmt.Errorf("failed to marshal config: %w", err)
	}

	configHash := sha256.Sum256(configData)
	configHashStr := hex.EncodeToString(configHash[:])

	layers = append(layers, ManifestLayer{
		MediaType: MediaTypeParams,
		Size:      int64(len(configData)),
		Digest:    fmt.Sprintf("sha256:%s", configHashStr),
	})

	// Create manifest config
	manifestConfig := ManifestConfig{
		MediaType: MediaTypeModel,
		Size:      int64(len(configData)),
		Digest:    fmt.Sprintf("sha256:%s", configHashStr),
	}

	manifest := OllamaManifest{
		SchemaVersion: 2,
		MediaType:     ManifestMediaType,
		Config:        manifestConfig,
		Layers:        layers,
	}

	return manifest, nil
}

// createBlobSymlink creates a symlink in Ollama's blob directory
func createBlobSymlink(sourcePath, hash, ollamaModelsDir string) (string, error) {
	blobsDir := filepath.Join(ollamaModelsDir, "blobs")

	// Ensure blobs directory exists
	if err := os.MkdirAll(blobsDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create blobs directory: %w", err)
	}

	// Create symlink with the format sha256-<hash>
	blobPath := filepath.Join(blobsDir, fmt.Sprintf("sha256-%s", hash))

	// Remove existing symlink if it exists
	_ = os.Remove(blobPath)

	// Create the symlink
	if err := os.Symlink(sourcePath, blobPath); err != nil {
		return "", fmt.Errorf("failed to create symlink from %s to %s: %w", sourcePath, blobPath, err)
	}

	logging.DebugLogger.Printf("Created blob symlink: %s -> %s", blobPath, sourcePath)
	return blobPath, nil
}

// cleanupBlobSymlinks removes symlinks created for the model
func cleanupBlobSymlinks(symlinks []string) {
	for _, symlink := range symlinks {
		if err := os.Remove(symlink); err != nil {
			logging.ErrorLogger.Printf("Failed to remove symlink %s: %v", symlink, err)
		} else {
			logging.DebugLogger.Printf("Removed blob symlink: %s", symlink)
		}
	}
}

// CreateOllamaModel creates an Ollama model from an LM Studio model
func CreateOllamaModel(model LMStudioModel, dryRun bool, ollamaHost string, client *api.Client) error {
	// Check if we're connecting to a local Ollama instance
	if !utils.IsLocalhost(ollamaHost) {
		return fmt.Errorf("creating Ollama models from LM Studio is only supported when connecting to a local Ollama instance (got %s)", ollamaHost)
	}

	modelName := strings.ToLower(strings.ReplaceAll(model.Name, "/", "-"))

	if dryRun {
		logging.InfoLogger.Printf("[DRY RUN] Would create Ollama model: %s", modelName)
		logging.InfoLogger.Printf("[DRY RUN] Source file: %s", model.Path)
		if len(model.VisionFiles) > 0 {
			logging.InfoLogger.Printf("[DRY RUN] Vision files: %v", model.VisionFiles)
		}
		return nil
	}

	logging.InfoLogger.Printf("Creating Ollama model: %s from LM Studio model: %s", modelName, model.Name)

	// Get Ollama models directory
	ollamaModelsDir := os.Getenv("OLLAMA_MODELS")
	if ollamaModelsDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		ollamaModelsDir = filepath.Join(homeDir, ".ollama", "models")
	}

	// Step 1: Calculate SHA256 hashes for all files
	hashes := make(map[string]string)
	var createdSymlinks []string
	cleanupOnError := true

	defer func() {
		// Only clean up on error, keep symlinks on success so Ollama can access the files
		if cleanupOnError && len(createdSymlinks) > 0 {
			logging.DebugLogger.Printf("Cleaning up symlinks due to error")
			cleanupBlobSymlinks(createdSymlinks)
		}
	}()

	// Hash main model file
	logging.DebugLogger.Printf("Calculating SHA256 for main file: %s", model.Path)
	mainHash, err := calculateSHA256(model.Path)
	if err != nil {
		return fmt.Errorf("failed to calculate hash for main file: %w", err)
	}
	hashes[model.Path] = mainHash

	// Create symlink for main model file
	symlinkPath, err := createBlobSymlink(model.Path, mainHash, ollamaModelsDir)
	if err != nil {
		return fmt.Errorf("failed to create blob symlink for main file: %w", err)
	}
	createdSymlinks = append(createdSymlinks, symlinkPath)

	// Hash vision files if present
	for _, visionFile := range model.VisionFiles {
		logging.DebugLogger.Printf("Calculating SHA256 for vision file: %s", visionFile)
		visionHash, err := calculateSHA256(visionFile)
		if err != nil {
			return fmt.Errorf("failed to calculate hash for vision file %s: %w", visionFile, err)
		}
		hashes[visionFile] = visionHash

		// Create symlink for vision file
		visionSymlinkPath, err := createBlobSymlink(visionFile, visionHash, ollamaModelsDir)
		if err != nil {
			return fmt.Errorf("failed to create blob symlink for vision file: %w", err)
		}
		createdSymlinks = append(createdSymlinks, visionSymlinkPath)
	}

	// Step 2: Generate manifest (for informational purposes)
	config := defaultConfig
	manifest, err := generateManifest(model, hashes, config)
	if err != nil {
		return fmt.Errorf("failed to generate manifest: %w", err)
	}

	logging.DebugLogger.Printf("Generated manifest with %d layers", len(manifest.Layers))

	// Step 3: Create the model using Ollama's API with Files parameter
	logging.DebugLogger.Printf("Creating Ollama model: %s using blob symlinks", modelName)

	// Build the files map with relative paths and digests
	files := make(map[string]string)

	// Use a simple relative path for the main model with sha256: prefix
	mainFileName := filepath.Base(model.Path)
	files[mainFileName] = fmt.Sprintf("sha256:%s", mainHash)

	// Add vision files if present
	for _, visionFile := range model.VisionFiles {
		visionFileName := filepath.Base(visionFile)
		files[visionFileName] = fmt.Sprintf("sha256:%s", hashes[visionFile])
	}

	// Use the Ollama API to create the model with Files instead of From
	createRequest := api.CreateRequest{
		Model: modelName,
		Files: files,
		Parameters: map[string]any{
			"num_ctx":     config.NumCtx,
			"temperature": config.Temperature,
			"top_p":       config.TopP,
			"min_p":       config.MinP,
		},
	}

	err = client.Create(context.Background(), &createRequest, func(resp api.ProgressResponse) error {
		if resp.Status != "" {
			logging.DebugLogger.Printf("Create progress: %s", resp.Status)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create Ollama model %s: %w", modelName, err)
	}

	// Success - don't clean up symlinks, Ollama needs them to access the model files
	cleanupOnError = false
	logging.InfoLogger.Printf("Successfully created Ollama model: %s", modelName)
	logging.DebugLogger.Printf("Created %d permanent symlinks in Ollama blob store", len(createdSymlinks))
	return nil
}
