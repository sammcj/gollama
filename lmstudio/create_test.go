package lmstudio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTestFile creates a temporary file with specified content and returns its path and SHA256 hash
func createTestFile(t *testing.T, dir, name, content string) (string, string) {
	t.Helper()
	filePath := filepath.Join(dir, name)
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash := sha256.Sum256([]byte(content))
	return filePath, hex.EncodeToString(hash[:])
}

func TestCreateBlobSymlink(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	ollamaModelsDir := filepath.Join(tempDir, "ollama_models")
	sourceDir := filepath.Join(tempDir, "source")

	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create a test file
	sourcePath, hash := createTestFile(t, sourceDir, "test.gguf", "test content")

	// Test symlink creation
	symlinkPath, err := createBlobSymlink(sourcePath, hash, ollamaModelsDir)
	if err != nil {
		t.Fatalf("createBlobSymlink failed: %v", err)
	}

	// Verify symlink was created
	expectedPath := filepath.Join(ollamaModelsDir, "blobs", "sha256-"+hash)
	if symlinkPath != expectedPath {
		t.Errorf("Expected symlink path %s, got %s", expectedPath, symlinkPath)
	}

	// Verify it's actually a symlink
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("Failed to stat symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("Expected file to be a symlink")
	}

	// Verify symlink target
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}
	if target != sourcePath {
		t.Errorf("Expected symlink target %s, got %s", sourcePath, target)
	}

	// Test creating symlink again (should replace existing)
	symlinkPath2, err := createBlobSymlink(sourcePath, hash, ollamaModelsDir)
	if err != nil {
		t.Fatalf("createBlobSymlink failed on second call: %v", err)
	}
	if symlinkPath2 != symlinkPath {
		t.Error("Second call should return same path")
	}
}

func TestCleanupBlobSymlinks(t *testing.T) {
	tempDir := t.TempDir()
	ollamaModelsDir := filepath.Join(tempDir, "ollama_models")
	sourceDir := filepath.Join(tempDir, "source")

	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create test files and symlinks
	var symlinks []string
	for i := 0; i < 3; i++ {
		sourcePath, hash := createTestFile(t, sourceDir, fmt.Sprintf("%s-%d-test.gguf", filepath.Base(t.Name()), i), fmt.Sprintf("content-%d", i))
		symlinkPath, err := createBlobSymlink(sourcePath, hash, ollamaModelsDir)
		if err != nil {
			t.Fatalf("Failed to create symlink: %v", err)
		}
		symlinks = append(symlinks, symlinkPath)
	}

	// Verify symlinks exist
	for _, symlink := range symlinks {
		if _, err := os.Lstat(symlink); err != nil {
			t.Errorf("Symlink should exist before cleanup: %v", err)
		}
	}

	// Clean up symlinks
	cleanupBlobSymlinks(symlinks)

	// Verify symlinks are removed
	for _, symlink := range symlinks {
		if _, err := os.Lstat(symlink); !os.IsNotExist(err) {
			t.Errorf("Symlink should be removed after cleanup: %s", symlink)
		}
	}
}

func TestCalculateSHA256(t *testing.T) {
	tempDir := t.TempDir()
	testContent := "test content for hashing"
	filePath, expectedHash := createTestFile(t, tempDir, "test.txt", testContent)

	hash, err := calculateSHA256(filePath)
	if err != nil {
		t.Fatalf("calculateSHA256 failed: %v", err)
	}

	if hash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, hash)
	}
}

func TestCalculateSHA256_NonExistentFile(t *testing.T) {
	_, err := calculateSHA256("/nonexistent/file.gguf")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestModelNameGeneration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"publisher/model", "publisher-model"},
		{"Publisher/Model", "publisher-model"},
		{"a/b/c", "a-b-c"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := strings.ToLower(strings.ReplaceAll(tt.input, "/", "-"))
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSymlinkIntegration(t *testing.T) {
	// This test validates the full symlink creation and cleanup flow
	tempDir := t.TempDir()
	ollamaModelsDir := filepath.Join(tempDir, "ollama_models")
	sourceDir := filepath.Join(tempDir, "source")

	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create test files
	modelPath, modelHash := createTestFile(t, sourceDir, "model.gguf", "model content")
	visionPath, visionHash := createTestFile(t, sourceDir, "mmproj.gguf", "vision content")

	var symlinks []string

	// Create symlinks
	modelSymlink, err := createBlobSymlink(modelPath, modelHash, ollamaModelsDir)
	if err != nil {
		t.Fatalf("Failed to create model symlink: %v", err)
	}
	symlinks = append(symlinks, modelSymlink)

	visionSymlink, err := createBlobSymlink(visionPath, visionHash, ollamaModelsDir)
	if err != nil {
		t.Fatalf("Failed to create vision symlink: %v", err)
	}
	symlinks = append(symlinks, visionSymlink)

	// Verify both symlinks exist
	for i, symlink := range symlinks {
		if _, err := os.Lstat(symlink); err != nil {
			t.Errorf("Symlink %d should exist: %v", i, err)
		}

		// Verify it's actually a symlink
		info, err := os.Lstat(symlink)
		if err != nil {
			t.Fatalf("Failed to stat symlink %d: %v", i, err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("File %d should be a symlink", i)
		}
	}

	// Clean up symlinks
	cleanupBlobSymlinks(symlinks)

	// Verify both symlinks are removed
	for i, symlink := range symlinks {
		if _, err := os.Lstat(symlink); !os.IsNotExist(err) {
			t.Errorf("Symlink %d should be removed after cleanup", i)
		}
	}
}
