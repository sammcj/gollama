package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sammcj/gollama/utils"
	"github.com/spf13/viper"
)

type Config struct {
	Columns           []string `mapstructure:"columns"`
	OllamaAPIKey      string   `mapstructure:"ollama_api_key"`
	OllamaAPIURL      string   `mapstructure:"ollama_api_url"`
	OllamaModelsDir   string   `mapstructure:"ollama_models_dir"`
	LMStudioFilePaths string   `mapstructure:"lm_studio_file_paths"`
	LogLevel          string   `mapstructure:"log_level"`
	LogFilePath       string   `mapstructure:"log_file_path"`
	SortOrder         string   `mapstructure:"sort_order"`   // Current sort order
	StripString       string   `mapstructure:"strip_string"` // Optional string to strip from model names in the TUI (e.g. a private registry URL)
	Editor            string   `mapstructure:"editor"`
	Theme             string   `mapstructure:"theme"`            // Name of the theme to use (without .json extension)
	DockerContainer   string   `mapstructure:"docker_container"` // Optionally specify a docker container to run the ollama commands in

	// GUI-specific configuration fields
	GuiEnabled      bool   `mapstructure:"gui_enabled"`       // Enable GUI interface
	WindowWidth     int    `mapstructure:"window_width"`      // GUI window width
	WindowHeight    int    `mapstructure:"window_height"`     // GUI window height
	WindowMinWidth  int    `mapstructure:"window_min_width"`  // GUI window minimum width
	WindowMinHeight int    `mapstructure:"window_min_height"` // GUI window minimum height
	DefaultView     string `mapstructure:"default_view"`      // Default view (models, running, settings)
	AutoRefresh     bool   `mapstructure:"auto_refresh"`      // Auto-refresh model list
	RefreshInterval int    `mapstructure:"refresh_interval"`  // Refresh interval in seconds
	ShowSystemTray  bool   `mapstructure:"show_system_tray"`  // Show system tray icon

	modified        bool   // Internal flag to track if the config has been modified
}

var defaultConfig = Config{
	Columns:           []string{"Name", "Size", "Quant", "Family", "Modified", "ID"},
	OllamaAPIKey:      "",
	OllamaAPIURL:      getAPIUrl(),
	OllamaModelsDir:   GetOllamaModelDir(),
	LMStudioFilePaths: GetLMStudioModelDir(),
	LogLevel:          "info",
	SortOrder:         "modified",
	StripString:       "",
	Editor:            "/usr/bin/vim",
	Theme:             "dark-neon",
	DockerContainer:   "",

	// GUI defaults
	GuiEnabled:      false,
	WindowWidth:     1200,
	WindowHeight:    800,
	WindowMinWidth:  800,
	WindowMinHeight: 600,
	DefaultView:     "models",
	AutoRefresh:     true,
	RefreshInterval: 30,
	ShowSystemTray:  false,
}

// GetOllamaModelDir returns the default Ollama models directory for the current OS
func GetOllamaModelDir() string {
	homeDir := utils.GetHomeDir()
	if runtime.GOOS == "darwin" {
		return filepath.Join(homeDir, ".ollama", "models")
	} else if runtime.GOOS == "linux" {
		return "/usr/share/ollama/models"
	}
	// Add Windows path if needed
	return filepath.Join(homeDir, ".ollama", "models")
}

// GetLMStudioModelDir returns the default LM Studio models directory for the current OS
func GetLMStudioModelDir() string {
	homeDir := utils.GetHomeDir()
	if runtime.GOOS == "darwin" {
		return filepath.Join(homeDir, ".lmstudio", "models")
	} else if runtime.GOOS == "linux" {
		return filepath.Join(homeDir, ".lmstudio", "models")
	}
	// Add Windows path if needed
	return filepath.Join(homeDir, ".lmstudio", "models")
}

// getAPIUrl determines the API URL based on environment variables.
func getAPIUrl() string {
	if apiUrl := os.Getenv("OLLAMA_API_URL"); apiUrl != "" {
		return apiUrl
	}
	if host := os.Getenv("OLLAMA_HOST"); host != "" {
		// Check if the host already starts with http:// or https://
		if len(host) >= 7 && host[:7] == "http://" || len(host) >= 8 && host[:8] == "https://" {
			return host
		}
		// If not, prepend http://
		return "http://" + host
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
	viper.SetDefault("theme", defaultConfig.Theme)
	viper.SetDefault("docker_container", defaultConfig.DockerContainer)

	// GUI defaults
	viper.SetDefault("gui_enabled", defaultConfig.GuiEnabled)
	viper.SetDefault("window_width", defaultConfig.WindowWidth)
	viper.SetDefault("window_height", defaultConfig.WindowHeight)
	viper.SetDefault("window_min_width", defaultConfig.WindowMinWidth)
	viper.SetDefault("window_min_height", defaultConfig.WindowMinHeight)
	viper.SetDefault("default_view", defaultConfig.DefaultView)
	viper.SetDefault("auto_refresh", defaultConfig.AutoRefresh)
	viper.SetDefault("refresh_interval", defaultConfig.RefreshInterval)
	viper.SetDefault("show_system_tray", defaultConfig.ShowSystemTray)

	return SaveConfig(defaultConfig)
}

func LoadConfig() (Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	// Dir of config file
	viper.AddConfigPath(utils.GetConfigDir())

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; create it
			if err := CreateDefaultConfig(); err != nil {
				// if the file already exists - return it, otherwise throw an error
				if _, err := os.Stat(utils.GetConfigPath()); err == nil {
					return LoadConfig()
				}
				return Config{}, fmt.Errorf("failed to create default config: %w", err)
			}
		} else {
			// if the config file is borked, recreate it and let the user know
			if err := CreateDefaultConfig(); err != nil {
				backupPath := utils.GetConfigPath() + ".borked." + time.Now().Format("2006-01-02")
				if err := os.Rename(utils.GetConfigPath(), backupPath); err != nil {
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

	viper.SetDefault("columns", defaultConfig.Columns)
	viper.SetDefault("ollama_api_key", defaultConfig.OllamaAPIKey)
	viper.SetDefault("ollama_api_url", defaultConfig.OllamaAPIURL)
	viper.SetDefault("ollama_models_dir", defaultConfig.OllamaModelsDir)
	viper.SetDefault("lm_studio_file_paths", defaultConfig.LMStudioFilePaths)
	viper.SetDefault("log_level", defaultConfig.LogLevel)
	viper.SetDefault("log_file_path", defaultConfig.LogFilePath)
	viper.SetDefault("sort_order", defaultConfig.SortOrder)
	viper.SetDefault("strip_string", defaultConfig.StripString)
	viper.SetDefault("editor", defaultConfig.Editor)
	viper.SetDefault("theme", defaultConfig.Theme)
	viper.SetDefault("docker_container", defaultConfig.DockerContainer)

	// GUI defaults
	viper.SetDefault("gui_enabled", defaultConfig.GuiEnabled)
	viper.SetDefault("window_width", defaultConfig.WindowWidth)
	viper.SetDefault("window_height", defaultConfig.WindowHeight)
	viper.SetDefault("window_min_width", defaultConfig.WindowMinWidth)
	viper.SetDefault("window_min_height", defaultConfig.WindowMinHeight)
	viper.SetDefault("default_view", defaultConfig.DefaultView)
	viper.SetDefault("auto_refresh", defaultConfig.AutoRefresh)
	viper.SetDefault("refresh_interval", defaultConfig.RefreshInterval)
	viper.SetDefault("show_system_tray", defaultConfig.ShowSystemTray)

	config.Columns = viper.GetStringSlice("columns")
	config.OllamaAPIKey = viper.GetString("ollama_api_key")
	config.OllamaAPIURL = viper.GetString("ollama_api_url")
	config.OllamaModelsDir = viper.GetString("ollama_models_dir")
	config.LMStudioFilePaths = viper.GetString("lm_studio_file_paths")
	config.LogLevel = viper.GetString("log_level")
	config.LogFilePath = viper.GetString("log_file_path")
	config.SortOrder = viper.GetString("sort_order")
	config.StripString = viper.GetString("strip_string")
	config.Editor = viper.GetString("editor")
	config.Theme = viper.GetString("theme")
	config.DockerContainer = viper.GetString("docker_container")

	// GUI fields
	config.GuiEnabled = viper.GetBool("gui_enabled")
	config.WindowWidth = viper.GetInt("window_width")
	config.WindowHeight = viper.GetInt("window_height")
	config.WindowMinWidth = viper.GetInt("window_min_width")
	config.WindowMinHeight = viper.GetInt("window_min_height")
	config.DefaultView = viper.GetString("default_view")
	config.AutoRefresh = viper.GetBool("auto_refresh")
	config.RefreshInterval = viper.GetInt("refresh_interval")
	config.ShowSystemTray = viper.GetBool("show_system_tray")

	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
	})
	viper.WatchConfig()

	return config, nil
}

func SaveConfig(config Config) error {
	// Set all config values in viper
	viper.Set("columns", config.Columns)
	viper.Set("ollama_api_key", config.OllamaAPIKey)
	viper.Set("ollama_api_url", config.OllamaAPIURL)
	viper.Set("ollama_models_dir", config.OllamaModelsDir)
	viper.Set("lm_studio_file_paths", config.LMStudioFilePaths)
	viper.Set("log_level", config.LogLevel)
	viper.Set("log_file_path", config.LogFilePath)
	viper.Set("sort_order", config.SortOrder)
	viper.Set("strip_string", config.StripString)
	viper.Set("editor", config.Editor)
	viper.Set("theme", config.Theme)
	viper.Set("docker_container", config.DockerContainer)

	// GUI fields
	viper.Set("gui_enabled", config.GuiEnabled)
	viper.Set("window_width", config.WindowWidth)
	viper.Set("window_height", config.WindowHeight)
	viper.Set("window_min_width", config.WindowMinWidth)
	viper.Set("window_min_height", config.WindowMinHeight)
	viper.Set("default_view", config.DefaultView)
	viper.Set("auto_refresh", config.AutoRefresh)
	viper.Set("refresh_interval", config.RefreshInterval)
	viper.Set("show_system_tray", config.ShowSystemTray)

	configPath := utils.GetConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := viper.WriteConfig(); err != nil {
		// If the config file doesn't exist, create it
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := viper.SafeWriteConfigAs(configPath); err != nil {
				return fmt.Errorf("failed to write config file: %w", err)
			}
		} else {
			return fmt.Errorf("failed to write config file: %w", err)
		}
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
