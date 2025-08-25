package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sammcj/gollama/config"
)

func TestRunModel(t *testing.T) {
	// Determine if we're running in CI
	inCI := os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != ""

	// if we're running in CI, simply skip these tests
	if inCI {
		t.Skip("Skipping operations tests in CI")
	} else {

		tests := []struct {
			name         string
			model        string
			cfg          *config.Config
			expectDocker bool
			expectError  bool
		}{
			{
				name:         "Run with Docker",
				model:        "test-model",
				cfg:          &config.Config{DockerContainer: "test-container"},
				expectDocker: true,
				expectError:  inCI, // Expect error in CI since docker won't be available
			},
			{
				name:         "Run without Docker",
				model:        "test-model",
				cfg:          &config.Config{DockerContainer: ""},
				expectDocker: false,
				expectError:  inCI, // Expect error in CI since ollama won't be available
			},
			{
				name:         "Run with Docker set to false",
				model:        "test-model",
				cfg:          &config.Config{DockerContainer: "false"},
				expectDocker: false,
				expectError:  inCI, // Expect error in CI since ollama won't be available
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cmd := runModel(tt.model, tt.cfg)
				if (cmd == nil) != tt.expectError {
					t.Errorf("runModel() error = %v, expectError %v", cmd == nil, tt.expectError)
					t.Logf("cmd: %v", cmd)
					return
				}
				// Further assertions can be added based on how you want to validate the `tea.Cmd` returned
			})
		}
	}
}

func TestIsGollamaSymlink(t *testing.T) {
	lmStudioModelsDir := "/Users/test/.lmstudio/models"

	tests := []struct {
		name        string
		symlinkPath string
		target      string
		expected    bool
	}{
		{
			name:        "Valid gollama symlink with -GGUF suffix",
			symlinkPath: "/Users/test/.lmstudio/models/unknown/nomic-embed-text-latest-GGUF/nomic-embed-text-latest.gguf",
			target:      "/Users/test/.ollama/models/blobs/sha256-970aa74c0a90ef7482477cf803618e776e173c007bf957f635f1015bfcfef0e6",
			expected:    true,
		},
		{
			name:        "Valid gollama symlink with -GGUF suffix different model",
			symlinkPath: "/Users/test/.lmstudio/models/unknown/qwen3-0.6b-GGUF/qwen3-0.6b.gguf",
			target:      "/Users/test/.ollama/models/blobs/sha256-7f4030143c1c477224c5434f8272c662a8b042079a0a584f0a27a1684fe2e1fa",
			expected:    true,
		},
		{
			name:        "Valid gollama symlink without -GGUF suffix",
			symlinkPath: "/Users/test/.lmstudio/models/testauthor/testmodel/testmodel.gguf",
			target:      "/Users/test/.ollama/models/blobs/sha256-abc123",
			expected:    true,
		},
		{
			name:        "Valid mmproj symlink with -GGUF suffix",
			symlinkPath: "/Users/test/.lmstudio/models/unknown/qwen3-0.6b-GGUF/mmproj-qwen3-0.6b.gguf",
			target:      "/Users/test/.ollama/models/blobs/sha256-def456",
			expected:    true,
		},
		{
			name:        "Non-Ollama blob target",
			symlinkPath: "/Users/test/.lmstudio/models/testauthor/testmodel/testmodel.gguf",
			target:      "/some/other/path/model.gguf",
			expected:    false,
		},
		{
			name:        "Wrong directory structure (too many parts)",
			symlinkPath: "/Users/test/.lmstudio/models/testauthor/testmodel/subdir/testmodel.gguf",
			target:      "/Users/test/.ollama/models/blobs/sha256-abc123",
			expected:    false,
		},
		{
			name:        "Wrong filename (doesn't match model name)",
			symlinkPath: "/Users/test/.lmstudio/models/testauthor/testmodel-GGUF/wrongname.gguf",
			target:      "/Users/test/.ollama/models/blobs/sha256-abc123",
			expected:    false,
		},
		{
			name:        "Non-gguf file",
			symlinkPath: "/Users/test/.lmstudio/models/testauthor/testmodel/testmodel.bin",
			target:      "/Users/test/.ollama/models/blobs/sha256-abc123",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory structure and symlink
			tempDir := t.TempDir()
			testLMDir := filepath.Join(tempDir, "lmstudio", "models")

			// Replace the test path in symlinkPath with our temp path
			relPath, _ := filepath.Rel(lmStudioModelsDir, tt.symlinkPath)
			actualSymlinkPath := filepath.Join(testLMDir, relPath)

			// Create parent directories
			err := os.MkdirAll(filepath.Dir(actualSymlinkPath), 0755)
			if err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}

			// Create the target file (so the symlink won't be broken)
			var actualTarget string
			if strings.Contains(tt.target, ".ollama/models/blobs/") {
				// For Ollama blob targets
				targetDir := filepath.Join(tempDir, ".ollama", "models", "blobs")
				err = os.MkdirAll(targetDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create target directory: %v", err)
				}
				actualTarget = filepath.Join(targetDir, filepath.Base(tt.target))
			} else {
				// For non-Ollama targets, create in a different location
				targetDir := filepath.Join(tempDir, "some", "other", "path")
				err = os.MkdirAll(targetDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create non-Ollama target directory: %v", err)
				}
				actualTarget = filepath.Join(targetDir, filepath.Base(tt.target))
			}

			err = os.WriteFile(actualTarget, []byte("test"), 0644)
			if err != nil {
				t.Fatalf("Failed to create target file: %v", err)
			}

			// Create symlink
			err = os.Symlink(actualTarget, actualSymlinkPath)
			if err != nil {
				t.Fatalf("Failed to create symlink: %v", err)
			}

			result := isGollamaSymlink(actualSymlinkPath, testLMDir)
			if result != tt.expected {
				t.Errorf("isGollamaSymlink() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
