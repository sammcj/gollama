package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sammcj/gollama/logging"
)

func TestGenerateDefaultConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempConfigPath := filepath.Join(tempDir, "config.json")

	err = generateDefaultConfig(tempConfigPath)
	if err != nil {
		t.Fatalf("Failed to generate default config: %v", err)
	}

	if _, err := os.Stat(tempConfigPath); os.IsNotExist(err) {
		t.Fatalf("Expected config file to be created at %s", tempConfigPath)
	}

	file, err := os.Open(tempConfigPath)
	if err != nil {
		t.Fatalf("Failed to open generated config file: %v", err)
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		t.Fatalf("Generated config is not valid JSON: %v", err)
	}

	if !compareConfigs(config, defaultConfig) {
		t.Errorf("Generated config does not match default config. Got: %v, Expected: %v", config, defaultConfig)
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name          string
		prepFunc      func(configPath string) error
		expected      Config
		expectedError bool
	}{
		{
			name: "Config file exists and is valid",
			prepFunc: func(configPath string) error {
				// Ensure environment variables are unset for this test
				os.Unsetenv("OLLAMA_HOST")
				os.Unsetenv("OLLAMA_API_URL")
				config := Config{
					Columns:           []string{"Name", "Size"},
					OllamaAPIKey:      "testkey",
					OllamaAPIURL:      "http://testurl",
					OllamaModelsDir:   GetOllamaModelDir(),
					LMStudioFilePaths: "/test/path",
					LogLevel:          "debug",
					LogFilePath:       "/test/path/gollama.log",
					SortOrder:         "name",
					StripString:       "strip",
					Editor:            "vim",
					DockerContainer:   "testcontainer",
				}
				return saveTempConfig(configPath, config)
			},
			expected: Config{
				Columns:           []string{"Name", "Size"},
				OllamaAPIKey:      "testkey",
				OllamaAPIURL:      "http://testurl",
				OllamaModelsDir:   GetOllamaModelDir(),
				LMStudioFilePaths: "/test/path",
				LogLevel:          "debug",
				LogFilePath:       "/test/path/gollama.log",
				SortOrder:         "name",
				StripString:       "strip",
				Editor:            "vim",
				DockerContainer:   "testcontainer",
			},
			expectedError: false,
		},
		{
			name: "Config file does not exist (no env vars)",
			prepFunc: func(configPath string) error {
				// Ensure both environment variables are unset for this test
				os.Unsetenv("OLLAMA_HOST")
				os.Unsetenv("OLLAMA_API_URL")
				return nil // No prep needed for this test case
			},
			expected: Config{
				Columns:           []string{"Name", "Size", "Quant", "Family", "Modified", "ID"},
				OllamaAPIKey:      "",
				OllamaAPIURL:      "http://127.0.0.1:11434", // Default URL when no env vars are set
				OllamaModelsDir:   GetOllamaModelDir(),
				LMStudioFilePaths: "",
				LogLevel:          "info",
				SortOrder:         "modified",
				StripString:       "",
				Editor:            "/usr/bin/vim",
				DockerContainer:   "",
			},
			expectedError: false,
		},
		{
			name: "Config file does not exist (with OLLAMA_HOST)",
			prepFunc: func(configPath string) error {
				// Set OLLAMA_HOST but unset OLLAMA_API_URL
				os.Unsetenv("OLLAMA_API_URL")
				os.Setenv("OLLAMA_HOST", "ollama.icu.lol")
				return nil
			},
			expected: Config{
				Columns:           []string{"Name", "Size", "Quant", "Family", "Modified", "ID"},
				OllamaAPIKey:      "",
				OllamaAPIURL:      "http://ollama.icu.lol",
				LMStudioFilePaths: "",
				OllamaModelsDir:   GetOllamaModelDir(),
				LogLevel:          "info",
				SortOrder:         "modified",
				StripString:       "",
				Editor:            "/usr/bin/vim",
				DockerContainer:   "",
			},
			expectedError: false,
		},
		{
			name: "Config file is invalid",
			prepFunc: func(configPath string) error {
				// Ensure environment variables are unset for this test
				os.Unsetenv("OLLAMA_HOST")
				os.Unsetenv("OLLAMA_API_URL")
				return os.WriteFile(configPath, []byte("invalid json"), 0644)
			},
			expected:      Config{},
			expectedError: true,
		},
		{
			name: "Empty ollama_api_url with OLLAMA_HOST set",
			prepFunc: func(configPath string) error {
				os.Unsetenv("OLLAMA_API_URL")
				os.Setenv("OLLAMA_HOST", "test.example.com:1234")
				config := Config{
					Columns:      []string{"Name", "Size"},
					OllamaAPIURL: "", // Empty URL to test fallback
					LogLevel:     "debug",
				}
				return saveTempConfig(configPath, config)
			},
			expected: Config{
				Columns:      []string{"Name", "Size"},
				OllamaAPIURL: "http://test.example.com:1234",
				LogLevel:     "debug",
			},
			expectedError: false,
		},
		{
			name: "Empty ollama_api_url with no environment variables",
			prepFunc: func(configPath string) error {
				os.Unsetenv("OLLAMA_HOST")
				os.Unsetenv("OLLAMA_API_URL")
				config := Config{
					Columns:      []string{"Name", "Size"},
					OllamaAPIURL: "", // Empty URL to test default
					LogLevel:     "debug",
				}
				return saveTempConfig(configPath, config)
			},
			expected: Config{
				Columns:      []string{"Name", "Size"},
				OllamaAPIURL: "http://127.0.0.1:11434",
				LogLevel:     "debug",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "config_test")
			if err != nil {
				t.Fatalf("Failed to create temporary directory: %v", err)
			}
			defer func() {
				os.RemoveAll(tempDir)
				// Clean up environment variables after each test
				if tt.name != "Empty ollama_api_url with OLLAMA_HOST set" {
					os.Unsetenv("OLLAMA_API_URL")
					os.Unsetenv("OLLAMA_HOST")
				}
			}()

			tempConfigPath := filepath.Join(tempDir, "config.json")
			if tt.prepFunc != nil {
				if err := tt.prepFunc(tempConfigPath); err != nil {
					t.Fatalf("prepFunc failed: %v", err)
				}
			}

			got, err := loadConfigFromPath(tempConfigPath)
			if (err != nil) != tt.expectedError {
				t.Errorf("LoadConfig() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if !tt.expectedError && !compareConfigs(got, tt.expected) {
				t.Errorf("LoadConfig() got = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSaveConfig(t *testing.T) {
	tests := []struct {
		name          string
		input         Config
		expectedError bool
	}{
		{
			name: "Valid config save",
			input: Config{
				Columns:           []string{"Name", "Size"},
				OllamaAPIKey:      "testkey",
				OllamaAPIURL:      "http://testurl",
				LMStudioFilePaths: "/test/path",
				LogLevel:          "debug",
				LogFilePath:       "/test/path/gollama.log",
				SortOrder:         "name",
				StripString:       "strip",
				Editor:            "vim",
				DockerContainer:   "testcontainer",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "config_test")
			if err != nil {
				t.Fatalf("Failed to create temporary directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			tempConfigPath := filepath.Join(tempDir, "config.json")

			err = saveConfigToPath(tempConfigPath, tt.input)
			if (err != nil) != tt.expectedError {
				t.Errorf("SaveConfig() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			if !tt.expectedError {
				var got Config
				file, err := os.Open(tempConfigPath)
				if err != nil {
					t.Fatalf("Failed to open config file: %v", err)
				}
				defer file.Close()

				if err := json.NewDecoder(file).Decode(&got); err != nil {
					t.Fatalf("Failed to decode config file: %v", err)
				}

				if !compareConfigs(got, tt.input) {
					t.Errorf("Config in file = %v, want %v", got, tt.input)
				}
			}
		})
	}
}

func saveTempConfig(testConfigPath string, config Config) error {
	if err := os.MkdirAll(filepath.Dir(testConfigPath), 0755); err != nil {
		logging.ErrorLogger.Printf("Failed to create config directory: %v\n", err)
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.Create(testConfigPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Set indentation for better readability

	return encoder.Encode(config)
}

func compareConfigs(a, b Config) bool {
	aBytes, _ := json.Marshal(a)
	bBytes, _ := json.Marshal(b)
	return string(aBytes) == string(bBytes)
}

func loadConfigFromPath(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config with API URL fallback
			config := defaultConfig
			config.OllamaAPIURL = getAPIUrl()
			return config, nil
		}
		return Config{}, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return Config{}, fmt.Errorf("failed to decode config file: %w", err)
	}

	// Only use getAPIUrl() if no URL is set in the config file
	if config.OllamaAPIURL == "" {
		config.OllamaAPIURL = getAPIUrl()
	}

	return config, nil
}

func saveConfigToPath(path string, config Config) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Set indentation for better readability

	return encoder.Encode(config)
}

func generateDefaultConfig(path string) error {
	return saveConfigToPath(path, defaultConfig)
}
