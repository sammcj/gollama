package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sammcj/gollama/logging"
	"github.com/spf13/viper"
)

type Config struct {
	Columns           []string `mapstructure:"columns"`
	OllamaAPIKey      string   `mapstructure:"ollama_api_key"`
	OllamaAPIURL      string   `mapstructure:"ollama_api_url"`
	LMStudioFilePaths string   `mapstructure:"lm_studio_file_paths"`
	LogLevel          string   `mapstructure:"log_level"`
	LogFilePath       string   `mapstructure:"log_file_path"`
	SortOrder         string   `mapstructure:"sort_order"`   // Current sort order
	StripString       string   `mapstructure:"strip_string"` // Optional string to strip from model names in the TUI (e.g. a private registry URL)
	Editor            string   `mapstructure:"editor"`
	DockerContainer   string   `mapstructure:"docker_container"` // Optionally specify a docker container to run the ollama commands in
}

var defaultConfig = Config{
	Columns:           []string{"Name", "Size", "Quant", "Family", "Modified", "ID"},
	OllamaAPIKey:      "",
	OllamaAPIURL:      getAPIUrl(),
	LMStudioFilePaths: "",
	LogLevel:          "info",
	LogFilePath:       getDefaultLogPath(),
	SortOrder:         "modified",
	StripString:       "",
	Editor:            "/usr/bin/vim",
	DockerContainer:   "",
}

// getAPIUrl determines the API URL based on environment variables.
func getAPIUrl() string {
	if apiUrl := os.Getenv("OLLAMA_API_URL"); apiUrl != "" {
		return apiUrl
	}
	if host := os.Getenv("OLLAMA_HOST"); host != "" {
		if host[:7] != "http://" && host[:8] != "https://" {
			host = "http://" + host
		}
		return host
	}
	return "http://127.0.0.1:11434"
}

// getDefaultLogPath returns the default log file path based on the home directory.
func getDefaultLogPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logging.ErrorLogger.Printf("Failed to get user home directory: %v\n", err)
	}
	return filepath.Join(homeDir, ".config", "gollama", "gollama.log")
}

func LoadConfig() (Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(getConfigPath())

	// Set default values
	viper.SetDefault("columns", defaultConfig.Columns)
	viper.SetDefault("ollama_api_key", defaultConfig.OllamaAPIKey)
	viper.SetDefault("ollama_api_url", defaultConfig.OllamaAPIURL)
	viper.SetDefault("lm_studio_file_paths", defaultConfig.LMStudioFilePaths)
	viper.SetDefault("log_level", defaultConfig.LogLevel)
	viper.SetDefault("log_file_path", defaultConfig.LogFilePath)
	viper.SetDefault("sort_order", defaultConfig.SortOrder)
	viper.SetDefault("strip_string", defaultConfig.StripString)
	viper.SetDefault("editor", defaultConfig.Editor)
	viper.SetDefault("docker_container", defaultConfig.DockerContainer)

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, create it with default values
			if err := SaveConfig(defaultConfig); err != nil {
				return Config{}, fmt.Errorf("failed to save default config: %w", err)
			}
		} else {
			return Config{}, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	config.OllamaAPIURL = getAPIUrl() // Ensure the API URL is set correctly from environment variables or defaults

	if config.LogLevel == "debug" {
		logging.DebugLogger.Println("Config loaded:", config)
	}

	return config, nil
}

func SaveConfig(config Config) error {
	viper.Set("columns", config.Columns)
	viper.Set("ollama_api_key", config.OllamaAPIKey)
	viper.Set("ollama_api_url", config.OllamaAPIURL)
	viper.Set("lm_studio_file_paths", config.LMStudioFilePaths)
	viper.Set("log_level", config.LogLevel)
	viper.Set("log_file_path", config.LogFilePath)
	viper.Set("sort_order", config.SortOrder)
	viper.Set("strip_string", config.StripString)
	viper.Set("editor", config.Editor)
	viper.Set("docker_container", config.DockerContainer)

	configPath := getConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	viper.SetConfigFile(configPath)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getConfigPath returns the path to the configuration JSON file.
func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logging.ErrorLogger.Printf("Failed to get user home directory: %v\n", err)
	}
	return filepath.Join(homeDir, ".config", "gollama", "config.json")
}
