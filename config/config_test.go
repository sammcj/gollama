// config_test.go
package config

import (
	"encoding/json"
	"os"
	"testing"
)

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
				config := Config{
					DefaultSort:       "Name",
					Columns:           []string{"Name", "Size"},
					OllamaAPIKey:      "testkey",
					OllamaAPIURL:      "http://testurl",
					LMStudioFilePaths: "/test/path",
					LogLevel:          "debug",
					LogFilePath:       "/test/path/gollama.log",
					SortOrder:         "name",
					StripString:       "strip",
					Editor:            "nano",
					DockerContainer:   "testcontainer",
				}
				return saveTempConfig(configPath, config)
			},
			expected: Config{
				DefaultSort:       "Name",
				Columns:           []string{"Name", "Size"},
				OllamaAPIKey:      "testkey",
				OllamaAPIURL:      "http://testurl",
				LMStudioFilePaths: "/test/path",
				LogLevel:          "debug",
				LogFilePath:       "/test/path/gollama.log",
				SortOrder:         "name",
				StripString:       "strip",
				Editor:            "nano",
				DockerContainer:   "testcontainer",
				LastSortSelection: "name",
			},
			expectedError: false,
		},
		{
			name: "Config file does not exist",
			prepFunc: func(configPath string) error {
				return nil // No prep needed for this test case
			},
			expected:      defaultConfig,
			expectedError: false,
		},
		{
			name: "Config file is invalid",
			prepFunc: func(configPath string) error {
				return os.WriteFile(configPath, []byte("invalid json"), 0644)
			},
			expected:      Config{},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := getConfigPath()
			os.Remove(configPath) // Ensure config file does not exist at start
			if tt.prepFunc != nil {
				if err := tt.prepFunc(configPath); err != nil {
					t.Fatalf("prepFunc failed: %v", err)
				}
			}

			got, err := LoadConfig()
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
				DefaultSort:       "Name",
				Columns:           []string{"Name", "Size"},
				OllamaAPIKey:      "testkey",
				OllamaAPIURL:      "http://testurl",
				LMStudioFilePaths: "/test/path",
				LogLevel:          "debug",
				LogFilePath:       "/test/path/gollama.log",
				SortOrder:         "name",
				StripString:       "strip",
				Editor:            "nano",
				DockerContainer:   "testcontainer",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := getConfigPath()
			os.Remove(configPath) // Ensure config file does not exist at start

			err := SaveConfig(tt.input)
			if (err != nil) != tt.expectedError {
				t.Errorf("SaveConfig() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			if !tt.expectedError {
				var got Config
				file, err := os.Open(configPath)
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

func saveTempConfig(configPath string, config Config) error {
	file, err := os.Create(configPath)
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
