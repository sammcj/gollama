package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/sammcj/gollama/utils"
)

// Theme represents a colour scheme for the TUI
type Theme struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Colours     ThemeColours      `json:"colours"`
	Family      map[string]string `json:"family"` // Model family-specific colours
}

// ThemeColours contains all the colour definitions for the TUI
type ThemeColours struct {
	// General UI elements
	HeaderForeground string `json:"header_foreground"` // Header text
	HeaderBorder     string `json:"header_border"`     // Header borders
	Selected         string `json:"selected"`          // Selected item text
	SelectedBg       string `json:"selected_bg"`       // Background colour for selected items

	// Text input elements
	PromptText      string `json:"prompt_text"`      // Prompt text (>)
	InputText       string `json:"input_text"`       // User input text
	PlaceholderText string `json:"placeholder_text"` // Placeholder text
	CursorBg        string `json:"cursor_bg"`        // Background colour for text cursor

	// Status message colours
	Error   string `json:"error"`   // Error messages
	Success string `json:"success"` // Success messages
	Info    string `json:"info"`    // Info messages
	Warning string `json:"warning"` // Warning messages

	// List item elements
	ItemName      []string `json:"item_name"`      // Alternating colours for item names
	ItemId        string   `json:"item_id"`        // Legacy field for metadata
	ItemDate      string   `json:"item_date"`      // Date metadata colour
	ItemSha       string   `json:"item_sha"`       // SHA metadata colour
	ItemBorder    string   `json:"item_border"`    // Item borders
	ItemHighlight string   `json:"item_highlight"` // Background colour for highlighted items

	// Help view colours
	HelpText string `json:"help_text"` // Help text
	HelpBg   string `json:"help_bg"`   // Background colour for help view

	// Compare view colours
	CompareHeader    string `json:"compare_header"`    // Comparison view headers
	CompareCommand   string `json:"compare_command"`   // Command text in comparisons
	CompareLocal     string `json:"compare_local"`     // Local version text
	CompareRemote    string `json:"compare_remote"`    // Remote version text
	CompareModified  string `json:"compare_modified"`  // Modified items
	CompareAdded     string `json:"compare_added"`     // Added items
	CompareRemoved   string `json:"compare_removed"`   // Removed items
	CompareSeparator string `json:"compare_separator"` // Comparison separators

	// Search view colour
	SearchHighlight string `json:"search_highlight"` // Highlighted search matches
	SearchText      string `json:"search_text"`      // Search text
	SearchHeader    string `json:"search_header"`    // Search headers

	// VRAM estimation indicators
	VRAMExceeds string `json:"vram_exceeds"` // For VRAM usage exceeding available memory
	VRAMWithin  string `json:"vram_within"`  // For VRAM usage within available memory
	VRAMUnknown string `json:"vram_unknown"` // For VRAM usage when available memory is unknown
}

// DarkNeonTheme returns the dark neon theme with current colour
var DarkNeonTheme = Theme{
	Name:        "dark-neon",
	Description: "Default dark theme with neon accents",
	Colours: ThemeColours{
		// General UI
		HeaderForeground: "#8B00FF",
		HeaderBorder:     "#8B21AA",
		Selected:         "#FFFFFF",
		SelectedBg:       "#6600CC",

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
		ItemName:      []string{"#FFFFFF", "#CBCBCB"},
		ItemId:        "#222222",
		ItemDate:      "#CBCCBC",
		ItemSha:       "#444444",
		ItemBorder:    "#8B008B",
		ItemHighlight: "#8B115B",

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
		VRAMExceeds: "#BB0000",
		VRAMWithin:  "#006400",
		VRAMUnknown: "#8B4513",
	},
	Family: map[string]string{
		"llama":       "#C71585",
		"mllama":      "#C71CC5",
		"alpaca":      "#8B008C",
		"command-r":   "#BB4144",
		"starcoder2":  "#800080",
		"starcoder":   "#4B0082",
		"gemma":       "#AA3D8B",
		"qwen2":       "#1444B0",
		"qwen3":       "#4D11DD",
		"phi":         "#000CCA",
		"phi2":        "#004CCA",
		"phi3":        "#008CCA",
		"granite":     "#2A4F4F",
		"deepseek":    "#008B8B",
		"deepseek2":   "#0CCB8B",
		"vicuna":      "#4169E1",
		"bert":        "#8B4253",
		"nomic-bert":  "#A0522D",
		"nomic":       "#8B6914",
		"qwen":        "#006400",
		"placeholder": "#483D8B",
	},
}

// LightTheme represents the light theme with neon accents
var LightTheme = Theme{
	Name:        "light-neon",
	Description: "Light theme with neon accents",
	Colours: ThemeColours{
		// General UI
		HeaderForeground: "238",
		HeaderBorder:     "237",
		Selected:         "#FFE5FF",
		SelectedBg:       "#4F0082",

		// Text input
		PromptText:      "#8B00FF",
		InputText:       "#4B0082",
		PlaceholderText: "#6600CC",
		CursorBg:        "#4B0082",

		// Status messages
		Error:   "#FF0000",
		Success: "#8B008B",
		Info:    "92",
		Warning: "#FF1493",

		// List items
		ItemName:      []string{"#000000", "#400044"},
		ItemId:        "#222222",
		ItemDate:      "#433444",
		ItemSha:       "#444444",
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
		"llama":       "#C71585",
		"mllama":      "#C71CC5",
		"alpaca":      "#8B008B",
		"command-r":   "#BB4144",
		"starcoder2":  "#800080",
		"starcoder":   "#4B0082",
		"gemma":       "#483D8B",
		"qwen2":       "#000080",
		"qwen3":       "#0000FF",
		"phi":         "#0000CD",
		"phi2":        "#004CCA",
		"phi3":        "#008CCA",
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

// BuiltinThemes contains all the built-in themes
var BuiltinThemes = map[string]Theme{
	"dark-neon":  DarkNeonTheme,
	"light-neon": LightTheme,
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

// SaveThemes ensures all built-in themes exist and are up to date with all fields
func SaveThemes() error {
	if err := EnsureThemesDir(); err != nil {
		return err
	}

	for themeName, builtinTheme := range BuiltinThemes {
		themePath := filepath.Join(GetThemesDir(), themeName+".json")
		var theme Theme
		updated := false

		// Check if theme file exists
		if data, err := os.ReadFile(themePath); err == nil {
			// Theme exists, try to parse it
			if err := json.Unmarshal(data, &theme); err != nil {
				// Move invalid JSON file to .borked extension
				borkedPath := themePath + ".borked"
				if err := os.Rename(themePath, borkedPath); err != nil {
					return fmt.Errorf("failed to move invalid theme file %s to .borked: %w", themeName, err)
				}
				// Create new theme file with default content
				theme = builtinTheme
				updated = true
			} else {
				// Update missing fields if JSON was valid
				if theme.Colours.HeaderForeground == "" {
					theme.Colours.HeaderForeground = builtinTheme.Colours.HeaderForeground
					updated = true
				}
				if theme.Colours.HeaderBorder == "" {
					theme.Colours.HeaderBorder = builtinTheme.Colours.HeaderBorder
					updated = true
				}
				if theme.Colours.Selected == "" {
					theme.Colours.Selected = builtinTheme.Colours.Selected
					updated = true
				}
				if theme.Colours.SelectedBg == "" {
					theme.Colours.SelectedBg = builtinTheme.Colours.SelectedBg
					updated = true
				}
				if theme.Colours.PromptText == "" {
					theme.Colours.PromptText = builtinTheme.Colours.PromptText
					updated = true
				}
				if theme.Colours.InputText == "" {
					theme.Colours.InputText = builtinTheme.Colours.InputText
					updated = true
				}
				if theme.Colours.PlaceholderText == "" {
					theme.Colours.PlaceholderText = builtinTheme.Colours.PlaceholderText
					updated = true
				}
				if theme.Colours.CursorBg == "" {
					theme.Colours.CursorBg = builtinTheme.Colours.CursorBg
					updated = true
				}
				if theme.Colours.Error == "" {
					theme.Colours.Error = builtinTheme.Colours.Error
					updated = true
				}
				if theme.Colours.Success == "" {
					theme.Colours.Success = builtinTheme.Colours.Success
					updated = true
				}
				if theme.Colours.Info == "" {
					theme.Colours.Info = builtinTheme.Colours.Info
					updated = true
				}
				if theme.Colours.Warning == "" {
					theme.Colours.Warning = builtinTheme.Colours.Warning
					updated = true
				}
				if len(theme.Colours.ItemName) == 0 {
					theme.Colours.ItemName = builtinTheme.Colours.ItemName
					updated = true
				}
				if theme.Colours.ItemId == "" {
					theme.Colours.ItemId = builtinTheme.Colours.ItemId
					updated = true
				}
				if theme.Colours.ItemDate == "" {
					theme.Colours.ItemDate = builtinTheme.Colours.ItemDate
					updated = true
				}
				if theme.Colours.ItemSha == "" {
					theme.Colours.ItemSha = builtinTheme.Colours.ItemSha
					updated = true
				}
				if theme.Colours.ItemBorder == "" {
					theme.Colours.ItemBorder = builtinTheme.Colours.ItemBorder
					updated = true
				}
				if theme.Colours.ItemHighlight == "" {
					theme.Colours.ItemHighlight = builtinTheme.Colours.ItemHighlight
					updated = true
				}
				if theme.Colours.HelpText == "" {
					theme.Colours.HelpText = builtinTheme.Colours.HelpText
					updated = true
				}
				if theme.Colours.HelpBg == "" {
					theme.Colours.HelpBg = builtinTheme.Colours.HelpBg
					updated = true
				}
				if theme.Colours.CompareHeader == "" {
					theme.Colours.CompareHeader = builtinTheme.Colours.CompareHeader
					updated = true
				}
				if theme.Colours.CompareCommand == "" {
					theme.Colours.CompareCommand = builtinTheme.Colours.CompareCommand
					updated = true
				}
				if theme.Colours.CompareLocal == "" {
					theme.Colours.CompareLocal = builtinTheme.Colours.CompareLocal
					updated = true
				}
				if theme.Colours.CompareRemote == "" {
					theme.Colours.CompareRemote = builtinTheme.Colours.CompareRemote
					updated = true
				}
				if theme.Colours.CompareModified == "" {
					theme.Colours.CompareModified = builtinTheme.Colours.CompareModified
					updated = true
				}
				if theme.Colours.CompareAdded == "" {
					theme.Colours.CompareAdded = builtinTheme.Colours.CompareAdded
					updated = true
				}
				if theme.Colours.CompareRemoved == "" {
					theme.Colours.CompareRemoved = builtinTheme.Colours.CompareRemoved
					updated = true
				}
				if theme.Colours.CompareSeparator == "" {
					theme.Colours.CompareSeparator = builtinTheme.Colours.CompareSeparator
					updated = true
				}
				if theme.Colours.SearchHighlight == "" {
					theme.Colours.SearchHighlight = builtinTheme.Colours.SearchHighlight
					updated = true
				}
				if theme.Colours.SearchText == "" {
					theme.Colours.SearchText = builtinTheme.Colours.SearchText
					updated = true
				}
				if theme.Colours.SearchHeader == "" {
					theme.Colours.SearchHeader = builtinTheme.Colours.SearchHeader
					updated = true
				}
				if theme.Colours.VRAMExceeds == "" {
					theme.Colours.VRAMExceeds = builtinTheme.Colours.VRAMExceeds
					updated = true
				}
				if theme.Colours.VRAMWithin == "" {
					theme.Colours.VRAMWithin = builtinTheme.Colours.VRAMWithin
					updated = true
				}
				if theme.Colours.VRAMUnknown == "" {
					theme.Colours.VRAMUnknown = builtinTheme.Colours.VRAMUnknown
					updated = true
				}

				// Check and update missing family colours
				if theme.Family == nil {
					theme.Family = make(map[string]string)
				}
				for family, colour := range builtinTheme.Family {
					if _, exists := theme.Family[family]; !exists {
						theme.Family[family] = colour
						updated = true
					}
				}
			}

			if updated {
				themeJSON, err := json.MarshalIndent(theme, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal updated theme %s: %w", themeName, err)
				}

				if err := os.WriteFile(themePath, themeJSON, 0644); err != nil {
					return fmt.Errorf("failed to write updated theme %s: %w", themeName, err)
				}
			}
		} else if os.IsNotExist(err) {
			// Theme doesn't exist, create it
			themeJSON, err := json.MarshalIndent(builtinTheme, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal new theme %s: %w", themeName, err)
			}

			if err := os.WriteFile(themePath, themeJSON, 0644); err != nil {
				return fmt.Errorf("failed to write new theme %s: %w", themeName, err)
			}
		} else {
			return fmt.Errorf("error checking theme file %s: %w", themeName, err)
		}
	}
	return nil
}

// LoadTheme loads a theme from the themes directory
func LoadTheme(name string) (*Theme, error) {
	// If it's a builtin theme, ensure it exists
	if _, isBuiltin := BuiltinThemes[name]; isBuiltin {
		if err := SaveThemes(); err != nil {
			return nil, fmt.Errorf("failed to ensure themes exist: %w", err)
		}
	}

	themePath := filepath.Join(GetThemesDir(), name+".json")
	themeData, err := os.ReadFile(themePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read theme file: %w", err)
	}

	var theme Theme
	if err := json.Unmarshal(themeData, &theme); err != nil {
		// Move invalid JSON file to .borked extension
		borkedPath := themePath + ".borked"
		if renameErr := os.Rename(themePath, borkedPath); renameErr != nil {
			return nil, fmt.Errorf("failed to move invalid theme file to .borked: %w", renameErr)
		}
		// Return built-in theme as default
		builtinTheme := BuiltinThemes[name]
		return &builtinTheme, nil
	}

	return &theme, nil
}

// GetColour returns a lipgloss.Colour from a theme colour string
func (t *Theme) GetColour(colour string) lipgloss.Color {
	return lipgloss.Color(colour)
}

// GetFamilyColour returns the colour for a model family
func (t *Theme) GetFamilyColour(family string) lipgloss.Color {
	if colour, ok := t.Family[family]; ok {
		return t.GetColour(colour)
	}
	return t.GetColour(t.Family["placeholder"])
}
