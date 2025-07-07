# Technology Stack & Build System

## Primary Languages
- **Go 1.24+** - Main application language
- **JavaScript/HTML/CSS** - GUI frontend (embedded in Go binary)

## Key Frameworks & Libraries

### TUI (Terminal User Interface)
- **Bubble Tea** (`github.com/charmbracelet/bubbletea`) - TUI framework
- **Lipgloss** (`github.com/charmbracelet/lipgloss`) - Styling and layout
- **Bubbles** (`github.com/charmbracelet/bubbles`) - TUI components (list, table, progress, textinput)

### GUI (Graphical User Interface)
- **Wails v3** (`github.com/wailsapp/wails/v3`) - Go + Web frontend framework
- **HTMX** - Frontend interactivity without heavy JavaScript
- **Tailwind CSS** - Utility-first CSS framework
- **Go HTML Templates** - Server-side rendering

### Core Dependencies
- **Ollama API Client** (`github.com/ollama/ollama/api`) - Ollama integration
- **Viper** (`github.com/spf13/viper`) - Configuration management
- **Zerolog** (`github.com/rs/zerolog`) - Structured logging
- **Spf13/pflag** - Command-line flag parsing
- **Fsnotify** - File system notifications

### Specialized Libraries
- **Spitter** (`github.com/sammcj/spitter`) - Model copying to remote hosts
- **Tablewriter** (`github.com/olekukonko/tablewriter`) - CLI table formatting
- **Gopsutil** (`github.com/shirou/gopsutil/v3`) - System information

## Build System

### Main Application
```bash
# Development build
make build

# Cross-platform build (CI)
make ci

# Run application
make run

# Clean build artifacts
make clean

# Run tests
make test
```

### GUI Application
```bash
# Build GUI with assets
cd gui && make all

# Development build
cd gui && make dev

# Build assets only
cd gui && make assets

# Run GUI application
cd gui && make run
```

### Asset Generation
- CSS: Tailwind CSS compilation via Go build script
- JS: HTMX minification and bundling
- Templates: Go embed for HTML templates
- Static files: Embedded filesystem using `//go:embed`

## Development Commands
- `go run .` - Run TUI application
- `cd gui && go run .` - Run GUI application
- `go test ./...` - Run all tests
- `go mod tidy` - Clean up dependencies
- `gofmt -w -s .` - Format code

## Platform Support
- **macOS** (primary development platform)
- **Linux** (full support)
- **Windows** (limited, mainly for LM Studio linking)

## Configuration
- **JSON-based config** in `~/.config/gollama/config.json`
- **Viper** for config management with defaults and validation
- **Theme system** with JSON theme files in `~/.config/gollama/themes/`
