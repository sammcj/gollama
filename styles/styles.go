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

// InitTheme initializes the current theme
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
		Foreground(theme.GetColor(theme.Colors.HeaderForeground)).
		MarginBottom(1)
}

func HeaderBorderStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.GetColor(theme.Colors.HeaderBorder))
}

// List item styles
func ItemNameStyle(index int) lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.ItemName[index%len(theme.Colors.ItemName)]))
}

func ItemIDStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.ItemId)).
		Faint(true)
}

func ItemBorderStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.InnerHalfBlockBorder()).
		BorderForeground(theme.GetColor(theme.Colors.ItemBorder)).
		PaddingLeft(1)
}

func SelectedItemStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Background(theme.GetColor(theme.Colors.ItemHighlight)).
		Bold(true).
		Italic(true)
}

// Size styles
func SizeStyle(size float64) lipgloss.Style {
	theme := GetTheme()
	switch {
	case size > 50:
		return lipgloss.NewStyle().Foreground(theme.GetColor(theme.Colors.VRAMExceeds))
	case size > 20:
		return lipgloss.NewStyle().Foreground(theme.GetColor(theme.Colors.Warning))
	case size > 10:
		return lipgloss.NewStyle().Foreground(theme.GetColor(theme.Colors.Info))
	default:
		return lipgloss.NewStyle().Foreground(theme.GetColor(theme.Colors.Success))
	}
}

// Quantization styles
func QuantStyle(level string) lipgloss.Style {
	theme := GetTheme()
	switch {
	case strings.Contains(level, "Q2"):
		return lipgloss.NewStyle().Foreground(theme.GetColor(theme.Colors.VRAMExceeds))
	case strings.Contains(level, "Q3"):
		return lipgloss.NewStyle().Foreground(theme.GetColor(theme.Colors.Warning))
	case strings.Contains(level, "Q4"):
		return lipgloss.NewStyle().Foreground(theme.GetColor(theme.Colors.Info))
	case strings.Contains(level, "Q5"):
		return lipgloss.NewStyle().Foreground(theme.GetColor(theme.Colors.Success))
	default:
		return lipgloss.NewStyle().Foreground(theme.GetColor(theme.Colors.ItemId))
	}
}

// Text input styles
func PromptStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.PromptText))
}

func InputTextStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.InputText))
}

func PlaceholderStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.PlaceholderText))
}

func CursorStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Background(theme.GetColor(theme.Colors.CursorBg))
}

// Message styles
func ErrorStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.Error))
}

func SuccessStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.Success))
}

func InfoStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.Info))
}

func WarningStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.Warning))
}

// Help styles
func HelpTextStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.HelpText)).
		Background(theme.GetColor(theme.Colors.HelpBg))
}

// Compare view styles
func CompareHeaderStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.CompareHeader)).
		MarginBottom(1).
		Bold(true)
}

func CompareCommandStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.CompareCommand)).
		Padding(0, 1)
}

func CompareLocalStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.CompareLocal)).
		Padding(0, 1)
}

func CompareRemoteStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.CompareRemote)).
		Padding(0, 1)
}

func CompareModifiedStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.CompareModified)).
		Padding(0, 1)
}

func CompareAddedStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.CompareAdded)).
		Padding(0, 1)
}

func CompareRemovedStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.CompareRemoved)).
		Padding(0, 1)
}

func CompareSeparatorStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.CompareSeparator))
}

// Model family color
func FamilyStyle(family string) lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetFamilyColor(family))
}

// VRAM estimation styles
func VRAMExceedsStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.VRAMExceeds))
}

func VRAMWithinStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.VRAMWithin))
}

func VRAMUnknownStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.VRAMUnknown))
}

// Search view styles
func SearchHighlightStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.SearchHighlight))
}

func SearchTextStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.SearchText))

}

func SearchHeaderStyle() lipgloss.Style {
	theme := GetTheme()
	return lipgloss.NewStyle().
		Foreground(theme.GetColor(theme.Colors.SearchHeader))
}
