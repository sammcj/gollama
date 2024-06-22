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
	OllamaAPIKeys     []string `mapstructure:"ollama_api_keys"`
	OllamaAPIURLs     []string `mapstructure:"ollama_api_urls"`
	OllamaAPIURL      string   `mapstructure:"ollama_api_url"` // Deprecated
	OllamaAPIKey      string   `mapstructure:"ollama_api_key"` // Deprecated
	CurrentServer     int      `mapstructure:"current_server"`
	LMStudioFilePaths string   `mapstructure:"lm_studio_file_paths"`
	LogLevel          string   `mapstructure:"log_level"`
	LogFilePath       string   `mapstructure:"log_file_path"`
	SortOrder         string   `mapstructure:"sort_order"`
	StripString       string   `mapstructure:"strip_string"`
	Editor            string   `mapstructure:"editor"`
	DockerContainer   string   `mapstructure:"docker_container"`
	Modified          bool     // Internal flag to track if the config has been modified
	ConfigVersion     int      `mapstructure:"config_version"`
}

var defaultConfig = Config{
	Columns:           []string{"Name", "Size", "Quant", "Family", "Modified", "ID"},
	OllamaAPIKeys:     []string{""},
	OllamaAPIURLs:     []string{getAPIUrl()},
	CurrentServer:     0,
	LMStudioFilePaths: "",
	LogLevel:          "info",
	SortOrder:         "modified",
	StripString:       "",
	Editor:            "/usr/bin/vim",
	DockerContainer:   "",
	ConfigVersion:     1,
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
	viper.SetDefault("ollama_api_keys", defaultConfig.OllamaAPIKeys)
	viper.SetDefault("ollama_api_url", defaultConfig.OllamaAPIURLs)
	viper.SetDefault("lm_studio_file_paths", defaultConfig.LMStudioFilePaths)
	viper.SetDefault("log_level", defaultConfig.LogLevel)
	viper.SetDefault("log_file_path", defaultConfig.LogFilePath)
	viper.SetDefault("sort_order", defaultConfig.SortOrder)
	viper.SetDefault("strip_string", defaultConfig.StripString)
	viper.SetDefault("editor", defaultConfig.Editor)
	viper.SetDefault("docker_container", defaultConfig.DockerContainer)
	viper.SetDefault("current_server", defaultConfig.CurrentServer)
	viper.SetDefault("config_version", defaultConfig.ConfigVersion)

	return SaveConfig(defaultConfig)
}

func LoadConfig() (Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(filepath.Join(os.Getenv("HOME"), ".config", "gollama"))

	/// V0 to V1 Config Migration ///
	// if the config file is an older version than the current default, check if it needs to be updated
	fmt.Println("Loading config")
	viper.ReadInConfig()
	// if viper.Get("config_version") != nil {
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	// if the config file has a single server (config.OllamaAPIURL), migrate it to the new format
	if viper.Get("ollama_api_url") != nil {
		oldServer := viper.GetString("ollama_api_url")
		oldKey := viper.GetString("ollama_api_key")
		viper.Set("ollama_api_urls", []string{oldServer})
		viper.Set("ollama_api_keys", []string{oldKey})
		viper.Set("current_server", 0)

		viper.Set("ollama_api_url", nil)
		viper.Set("ollama_api_key", nil)

		if err := viper.WriteConfig(); err != nil {
			return Config{}, fmt.Errorf("failed to write config file: %w", err)
		}
	}

	// print a message to let the user know the config is being updated
	fmt.Println("Updating config file to version", defaultConfig.ConfigVersion)

	// update the config version
	config.ConfigVersion = defaultConfig.ConfigVersion
	if err := SaveConfig(config); err != nil {
		return Config{}, fmt.Errorf("failed to save updated config: %w", err)
	}
	// }
	/// End V0 to V1 Config Migration ///

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

	// var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Check if the config is in the old format and migrate if necessary
	if len(config.OllamaAPIURLs) == 0 {
		oldURL := viper.GetString("ollama_api_url")
		oldKey := viper.GetString("ollama_api_key")
		if oldURL != "" {
			config.OllamaAPIURLs = []string{oldURL}
			config.OllamaAPIKeys = []string{oldKey}
			config.CurrentServer = 0
			if err := SaveConfig(config); err != nil {
				return Config{}, fmt.Errorf("failed to save migrated config: %w", err)
			}
		}
	}

	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
	})
	viper.WatchConfig()

	return config, nil
}

func SaveConfig(config Config) error {
	viper.Set("sort_order", config.SortOrder)
	viper.Set("current_server", config.CurrentServer)

	configPath := getConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := viper.SafeWriteConfigAs(configPath); err != nil {
		// update the config file in place
		if err := viper.WriteConfig(); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
	}

	return nil
}

func (c *Config) SaveIfModified() error {
	if c.Modified {
		return SaveConfig(*c)
	}
	return nil
}

func (c *Config) SetModified() {
	c.Modified = true
}

// getConfigPath returns the path to the configuration JSON file.
func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logging.ErrorLogger.Printf("Failed to get user home directory: %v\n", err)
	}
	return filepath.Join(homeDir, ".config", "gollama", "config.json")
}
