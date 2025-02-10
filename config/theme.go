package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/sammcj/gollama/utils"
)

// Theme represents a color scheme for the TUI
type Theme struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Colors      ThemeColors       `json:"colors"`
	Family      map[string]string `json:"family"` // Model family-specific colors
}

// ThemeColors contains all the color definitions for the TUI
type ThemeColors struct {
	// General UI
	HeaderForeground string `json:"header_foreground"`
	HeaderBorder     string `json:"header_border"`
	Selected         string `json:"selected"`
	SelectedBg       string `json:"selected_bg"`

	// Text input
	PromptText      string `json:"prompt_text"`
	InputText       string `json:"input_text"`
	PlaceholderText string `json:"placeholder_text"`
	CursorBg        string `json:"cursor_bg"`

	// Status messages
	Error   string `json:"error"`
	Success string `json:"success"`
	Info    string `json:"info"`
	Warning string `json:"warning"`

	// List items
	ItemName      []string `json:"item_name"` // Alternating colors
	ItemId        string   `json:"item_id"`
	ItemBorder    string   `json:"item_border"`
	ItemHighlight string   `json:"item_highlight"`

	// Help view
	HelpText string `json:"help_text"`
	HelpBg   string `json:"help_bg"`

	// Compare view
	CompareHeader    string `json:"compare_header"`
	CompareCommand   string `json:"compare_command"`
	CompareLocal     string `json:"compare_local"`
	CompareRemote    string `json:"compare_remote"`
	CompareModified  string `json:"compare_modified"`
	CompareAdded     string `json:"compare_added"`
	CompareRemoved   string `json:"compare_removed"`
	CompareSeparator string `json:"compare_separator"`

	// Search view
	SearchHighlight string `json:"search_highlight"`
	SearchText      string `json:"search_text"`
	SearchHeader    string `json:"search_header"`

	// VRAM estimation
	VRAMExceeds string `json:"vram_exceeds"` // For VRAM usage exceeding available memory
	VRAMWithin  string `json:"vram_within"`  // For VRAM usage within available memory
	VRAMUnknown string `json:"vram_unknown"` // For VRAM usage when available memory is unknown
}

// DefaultTheme returns the default dark theme with current colors
var DefaultTheme = Theme{
	Name:        "default",
	Description: "Default dark theme with neon accents",
	Colors: ThemeColors{
		// General UI
		HeaderForeground: "241",
		HeaderBorder:     "240",
		Selected:         "229",
		SelectedBg:       "57",

		// Text input
		PromptText:      "#FF00FF",
		InputText:       "#FF00FF",
		PlaceholderText: "#AD00FF",
		CursorBg:        "#4E00FF",

		// Status messages
		Error:   "#8B0000",
		Success: "#EE82EE",
		Info:    "129",
		Warning: "#FFB6C1",

		// List items
		ItemName:      []string{"#FFFFFF", "#818FA1"},
		ItemId:        "254",
		ItemBorder:    "125",
		ItemHighlight: "92",

		// Help view
		HelpText: "#626262",
		HelpBg:   "#000000",

		// Compare view
		CompareHeader:    "#FF00FF",
		CompareCommand:   "#9932CC",
		CompareLocal:     "#60BFFF",
		CompareRemote:    "#00CED1",
		CompareModified:  "#FFFF00",
		CompareAdded:     "#00FF00",
		CompareRemoved:   "#FF0000",
		CompareSeparator: "#333333",

		// Search view
		SearchHighlight: "#FF60FF", // Dark mode color (light: #5000D3)
		SearchText:      "#FFFFFF", // Dark mode color (light: #000000)
		SearchHeader:    "#AAEE9A", // Dark mode color (light: #000000)

		// VRAM estimation
		VRAMExceeds: "#FF0000", // Red for exceeding memory
		VRAMWithin:  "#00FF00", // Green for within memory
		VRAMUnknown: "#FFB6C1", // Light pink for unknown
	},
	Family: map[string]string{
		"llama":       "#FF1493",
		"alpaca":      "#FF00FF",
		"command-r":   "#FB79B4",
		"starcoder2":  "#EE82EE",
		"starcoder":   "#DD40DD",
		"gemma":       "#A224AA",
		"qwen2":       "#AAE",
		"phi":         "#554FFF",
		"granite":     "#BFBBBB",
		"deepseek":    "#06AFFF",
		"deepseek2":   "#60BFFF",
		"vicuna":      "#00CED1",
		"bert":        "#FF7A00",
		"nomic-bert":  "#FF8C00",
		"nomic":       "#FFD700",
		"qwen":        "#7FFF00",
		"placeholder": "#554AAF",
	},
}

// LightTheme represents the light theme with neon accents
var LightTheme = Theme{
	Name:        "light-neon",
	Description: "Light theme with neon accents",
	Colors: ThemeColors{
		// General UI
		HeaderForeground: "238",
		HeaderBorder:     "237",
		Selected:         "232",
		SelectedBg:       "93",

		// Text input
		PromptText:      "#8B00FF",
		InputText:       "#8B00FF",
		PlaceholderText: "#6600CC",
		CursorBg:        "#4B0082",

		// Status messages
		Error:   "#FF0000",
		Success: "#8B008B",
		Info:    "92",
		Warning: "#FF1493",

		// List items
		ItemName:      []string{"#000000", "#444444"},
		ItemId:        "235",
		ItemBorder:    "92",
		ItemHighlight: "93",

		// Help view
		HelpText: "#444444",
		HelpBg:   "#FFFFFF",

		// Compare view
		CompareHeader:    "#8B008B",
		CompareCommand:   "#6A0DAD",
		CompareLocal:     "#0000CD",
		CompareRemote:    "#008B8B",
		CompareModified:  "#8B4513",
		CompareAdded:     "#006400",
		CompareRemoved:   "#8B0000",
		CompareSeparator: "#CCCCCC",

		// Search view
		SearchHighlight: "#5000D3",
		SearchText:      "#000000",
		SearchHeader:    "#000000",

		// VRAM estimation
		VRAMExceeds: "#8B0000",
		VRAMWithin:  "#006400",
		VRAMUnknown: "#8B4513",
	},
	Family: map[string]string{
		"llama":       "#8B0000",
		"alpaca":      "#8B008B",
		"command-r":   "#C71585",
		"starcoder2":  "#800080",
		"starcoder":   "#4B0082",
		"gemma":       "#483D8B",
		"qwen2":       "#000080",
		"phi":         "#0000CD",
		"granite":     "#2F4F4F",
		"deepseek":    "#008B8B",
		"deepseek2":   "#0000FF",
		"vicuna":      "#4169E1",
		"bert":        "#8B4513",
		"nomic-bert":  "#A0522D",
		"nomic":       "#8B6914",
		"qwen":        "#006400",
		"placeholder": "#483D8B",
	},
}

// GetThemesDir returns the path to the themes directory
func GetThemesDir() string {
	return filepath.Join(utils.GetConfigDir(), "themes")
}

// EnsureThemesDir creates the themes directory if it doesn't exist
func EnsureThemesDir() error {
	themesDir := GetThemesDir()
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		return fmt.Errorf("failed to create themes directory: %w", err)
	}
	return nil
}

// SaveLightTheme saves the light theme to the themes directory
func SaveLightTheme() error {
	if err := EnsureThemesDir(); err != nil {
		return err
	}

	themePath := filepath.Join(GetThemesDir(), "light-neon.json")

	// Only create if it doesn't exist
	if _, err := os.Stat(themePath); os.IsNotExist(err) {
		themeJSON, err := json.MarshalIndent(LightTheme, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal light theme: %w", err)
		}

		if err := os.WriteFile(themePath, themeJSON, 0644); err != nil {
			return fmt.Errorf("failed to write light theme: %w", err)
		}
	}
	return nil
}
// SaveDefaultTheme saves the default theme to the themes directory
func SaveDefaultTheme() error {
	if err := EnsureThemesDir(); err != nil {
		return err
	}

	themePath := filepath.Join(GetThemesDir(), "default.json")

	// Only create if it doesn't exist
	if _, err := os.Stat(themePath); os.IsNotExist(err) {
		themeJSON, err := json.MarshalIndent(DefaultTheme, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal default theme: %w", err)
		}

		if err := os.WriteFile(themePath, themeJSON, 0644); err != nil {
			return fmt.Errorf("failed to write default theme: %w", err)
		}
	}

	return nil
}

// LoadTheme loads a theme from the themes directory
func LoadTheme(name string) (*Theme, error) {
	themePath := filepath.Join(GetThemesDir(), name+".json")

	themeData, err := os.ReadFile(themePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read theme file: %w", err)
	}

	var theme Theme
	if err := json.Unmarshal(themeData, &theme); err != nil {
		return nil, fmt.Errorf("failed to parse theme file: %w", err)
	}

	return &theme, nil
}

// GetColor returns a lipgloss.Color from a theme color string
func (t *Theme) GetColor(color string) lipgloss.Color {
	return lipgloss.Color(color)
}

// GetFamilyColor returns the color for a model family
func (t *Theme) GetFamilyColor(family string) lipgloss.Color {
	if color, ok := t.Family[family]; ok {
		return t.GetColor(color)
	}
	return t.GetColor(t.Family["placeholder"])
}
