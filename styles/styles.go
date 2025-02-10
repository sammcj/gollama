package styles

import (
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/sammcj/gollama/config"
)

var (
	currentTheme *config.Theme
	themeMutex   sync.RWMutex
)

// InitTheme initialises the current theme
func InitTheme(theme *config.Theme) {
	themeMutex.Lock()
	defer themeMutex.Unlock()
	currentTheme = theme
}

// GetTheme returns the current theme
func GetTheme() *config.Theme {
	themeMutex.RLock()
	defer themeMutex.RUnlock()
	return currentTheme
}

// Header styles
func HeaderStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.HeaderForeground)).
		MarginBottom(1)
}

func HeaderBorderStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.GetColour(theme.Colours.HeaderBorder))
}

// List item styles
func ItemNameStyle(index int) lipgloss.Style {
	theme := GetTheme()

	style := lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.ItemName[index%len(theme.Colours.ItemName)]))

	return style
}

func ItemDateStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.ItemDate)).
		Bold(true)
}

func ItemShaStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.ItemSha)).
		Bold(true)
}

// ItemIDStyle is kept for backwards compatibility
func ItemIDStyle() lipgloss.Style {
	return ItemDateStyle() // Default to date style for backwards compatibility
	// This ensures existing code continues to work while we transition to the new specific styles
}

func ItemBorderStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.InnerHalfBlockBorder()).
		BorderForeground(theme.GetColour(theme.Colours.ItemBorder)).
		PaddingLeft(1)
}

func SelectedItemStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Background(theme.GetColour(theme.Colours.SelectedBg)).
		Foreground(theme.GetColour(theme.Colours.Selected)).
		Bold(true).
		Italic(true)
}

// Size styles
func SizeStyle(size float64) lipgloss.Style {
	theme := GetTheme()
	switch {
	case size > 50:
		return lipgloss.NewStyle().Foreground(theme.GetColour(theme.Colours.VRAMExceeds))
	case size > 20:
		return lipgloss.NewStyle().Foreground(theme.GetColour(theme.Colours.Warning))
	case size > 10:
		return lipgloss.NewStyle().Foreground(theme.GetColour(theme.Colours.Info))
	default:
		return lipgloss.NewStyle().Foreground(theme.GetColour(theme.Colours.Success))
	}
}

// Quantization styles
func QuantStyle(level string) lipgloss.Style {
	theme := GetTheme()
	switch {
	case strings.Contains(level, "IQ"):
		return lipgloss.NewStyle().Foreground(theme.GetColour(theme.Colours.VRAMExceeds))
	case strings.Contains(level, "Q2"):
		return lipgloss.NewStyle().Foreground(theme.GetColour(theme.Colours.VRAMExceeds))
	case strings.Contains(level, "Q3"):
		return lipgloss.NewStyle().Foreground(theme.GetColour(theme.Colours.Warning))
	case strings.Contains(level, "Q4"):
		return lipgloss.NewStyle().Foreground(theme.GetColour(theme.Colours.Info))
	case strings.Contains(level, "Q5"):
		return lipgloss.NewStyle().Foreground(theme.GetColour(theme.Colours.Success))
	case strings.Contains(level, "Q6"):
		return lipgloss.NewStyle().Foreground(theme.GetColour(theme.Colours.Warning))
	case strings.Contains(level, "Q8"):
		return lipgloss.NewStyle().Foreground(theme.GetColour(theme.Colours.Info))
	case strings.Contains(level, "F16"):
		return lipgloss.NewStyle().Foreground(theme.GetColour(theme.Colours.Warning))
	default:
		return lipgloss.NewStyle().Foreground(theme.GetColour(theme.Colours.ItemId))
	}
}

// Text input styles
func PromptStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.PromptText))
}

func InputTextStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.InputText))
}

func PlaceholderStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.PlaceholderText))
}

func CursorStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Background(theme.GetColour(theme.Colours.CursorBg))
}

// Message styles
func ErrorStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.Error))
}

func SuccessStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.Success))
}

func InfoStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.Info))
}

func WarningStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.Warning))
}

// Help styles
func HelpTextStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.HelpText)).
		Background(theme.GetColour(theme.Colours.HelpBg))
}

// Compare view styles
func CompareHeaderStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.CompareHeader)).
		MarginBottom(1).
		Bold(true)
}

func CompareCommandStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.CompareCommand)).
		Padding(0, 1)
}

func CompareLocalStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.CompareLocal)).
		Padding(0, 1)
}

func CompareRemoteStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.CompareRemote)).
		Padding(0, 1)
}

func CompareModifiedStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.CompareModified)).
		Padding(0, 1)
}

func CompareAddedStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.CompareAdded)).
		Padding(0, 1)
}

func CompareRemovedStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.CompareRemoved)).
		Padding(0, 1)
}

func CompareSeparatorStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.CompareSeparator))
}

// Model family colour
func FamilyStyle(family string) lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().Bold(true).
		Foreground(theme.GetFamilyColour(family))
}

// VRAM estimation styles
func VRAMExceedsStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.VRAMExceeds))
}

func VRAMWithinStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.VRAMWithin))
}

func VRAMUnknownStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.VRAMUnknown))
}

// Search view styles
func SearchHighlightStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.SearchHighlight))
}

func SearchTextStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.SearchText))

}

func SearchHeaderStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColour(theme.Colours.SearchHeader))
}
