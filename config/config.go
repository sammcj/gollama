package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
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
	modified          bool     // Internal flag to track if the config has been modified
}

var defaultConfig = Config{
	Columns:           []string{"Name", "Size", "Quant", "Family", "Modified", "ID"},
	OllamaAPIKey:      "",
	OllamaAPIURL:      getAPIUrl(),
	LMStudioFilePaths: "",
	LogLevel:          "info",
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

func CreateDefaultConfig() error {
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

	return SaveConfig(defaultConfig)
}

func LoadConfig() (Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(filepath.Join(os.Getenv("HOME"), ".config", "gollama"))

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; create it
			if err := CreateDefaultConfig(); err != nil {
				return Config{}, fmt.Errorf("failed to create default config: %w", err)
			}
		} else {
			// if the config file is borked, recreate it and let the user know
			if err := CreateDefaultConfig(); err != nil {
				backupPath := getConfigPath() + ".borked." + time.Now().Format("2006-01-02")
				if err := os.Rename(getConfigPath(), backupPath); err != nil {
					return Config{}, fmt.Errorf("failed to rename config file: %w", err)
				}
				if err := CreateDefaultConfig(); err != nil {
					return Config{}, fmt.Errorf("failed to recreate default config: %w", err)
				}
				fmt.Println("Your config file is borked!\nConfig recreated with default values, your old one has been backed up to", backupPath)
				fmt.Println("Press enter to continue...")
				fmt.Scanln()
			}
		}
	}

	var config Config
	config.OllamaAPIURL = viper.GetString("ollama_api_url")
	config.LogLevel = viper.GetString("log_level")

	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
	})
	viper.WatchConfig()

	return config, nil
}

func SaveConfig(config Config) error {
	if config.modified {
		viper.Set("sort_order", config.SortOrder)
	}

	configPath := getConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := viper.SafeWriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (c *Config) SaveIfModified() error {
	if c.modified {
		return SaveConfig(*c)
	}
	return nil
}

func (c *Config) SetModified() {
	c.modified = true
}

// getConfigPath returns the path to the configuration JSON file.
func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logging.ErrorLogger.Printf("Failed to get user home directory: %v\n", err)
	}
	return filepath.Join(homeDir, ".config", "gollama", "config.json")
}
