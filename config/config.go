// config.go contains the Config struct and functions to load and save the configuration file.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sammcj/gollama/logging"
)

type Config struct {
	DefaultSort       string   `json:"default_sort"`
	Columns           []string `json:"columns"`
	OllamaAPIKey      string   `json:"ollama_api_key"`
	OllamaAPIURL      string   `json:"ollama_api_url"`
	LMStudioFilePaths string   `json:"lm_studio_file_paths"`
	LogLevel          string   `json:"log_level"`
	LogFilePath       string   `json:"log_file_path"`
	SortOrder         string   `json:"sort_order"`   // Current sort order
	LastSortSelection string   `json:"-"`            // Temporary field to hold the last sort selection
	StripString       string   `json:"strip_string"` // Optional string to strip from model names in the TUI (e.g. a private registry URL)
	Editor            string   `json:"editor"`
	DockerContainer   string   `json:"docker_container"` // Optionally specify a docker container to run the ollama commands in
}

var defaultConfig = Config{
	DefaultSort:       "Size",
	Columns:           []string{"Name", "Size", "Quant", "Family", "Modified", "ID"},
	OllamaAPIKey:      "",
	OllamaAPIURL:      os.Getenv("OLLAMA_HOST"),
	LMStudioFilePaths: "",
	LogLevel:          "info",
	LogFilePath:       os.Getenv("HOME") + "/.config/gollama/gollama.log",
	SortOrder:         "modified", // Default sort order
	StripString:       "",
	Editor:            "vim",
	DockerContainer:   "",
}

func LoadConfig() (Config, error) {
	// Initialize loggers with desired parameters.
	if err := logging.Init("debug", "gollama.log"); err != nil {
		fmt.Println("Failed to initialize loggers:", err)
		os.Exit(1)
	}
	configPath := getConfigPath()
	logging.InfoLogger.Printf("Loading config from: %s\n", configPath)

	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			logging.DebugLogger.Println("Config file does not exist, creating with default values")
			fmt.Println("Config file does not exist, creating with default values")

			// Create the config directory if it doesn't exist
			if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
				logging.ErrorLogger.Printf("Failed to create config directory: %v\n", err)
				return Config{}, fmt.Errorf("failed to create config directory: %w", err)
			}

			// Save the default config
			if err := SaveConfig(defaultConfig); err != nil {
				logging.ErrorLogger.Printf("Failed to save default config: %v\n", err)
				return Config{}, fmt.Errorf("failed to save default config: %w", err)
			}
			// now continue to open the file
			file, err = os.Open(configPath)
			if err != nil {
				return Config{}, fmt.Errorf("failed to open config file: %w", err)
			}
		} else {
			logging.ErrorLogger.Printf("Failed to open config file: %v\n", err)
			return Config{}, fmt.Errorf("failed to open config file: %w", err)
		}
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		logging.ErrorLogger.Printf("Failed to decode config file: %v\n", err)
		return Config{}, fmt.Errorf("failed to decode config file: %w", err)
	}

	// Set the last sort selection to the current sort order
	config.LastSortSelection = config.SortOrder

	if config.LogLevel == "debug" {
		logging.DebugLogger.Println("Config loaded:", config)
	}
	return config, nil
}

func SaveConfig(config Config) error {
	configPath := getConfigPath()
	logging.DebugLogger.Printf("Saving config to: %s\n", configPath)

	// if the config file doesn't exist, create it
	file, err := os.Create(configPath)
	if err != nil {
		logging.ErrorLogger.Printf("Failed to create config file: %v\n", err)
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Set indentation for better readability

	if err := encoder.Encode(config); err != nil {
		logging.ErrorLogger.Printf("Failed to encode config to file: %v\n", err)
		return fmt.Errorf("failed to encode config to file: %w", err)
	}
	return nil
}

func getConfigPath() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "gollama", "config.json")
}
