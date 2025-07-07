# Project Structure & Organization

## Root Directory Layout
```
gollama/
├── main.go                 # TUI application entry point
├── go.mod/go.sum          # Go module dependencies
├── Makefile               # Build automation for TUI
├── README.md              # Project documentation
├── LICENSE                # MIT license
├── gui/                   # GUI application (Wails v3)
├── core/                  # Shared business logic
├── config/                # Configuration management
├── styles/                # TUI styling (Lipgloss)
├── utils/                 # Utility functions
├── logging/               # Logging utilities
├── vramestimator/         # vRAM calculation logic
├── lmstudio/              # LM Studio integration
├── screenshots/           # Documentation images
└── scripts/               # Installation and utility scripts
```

## Core Architecture Patterns

### Separation of Concerns
- **`main.go`** - TUI application entry point and CLI argument handling
- **`gui/`** - Complete GUI application with Wails v3
- **`core/`** - Shared business logic between TUI and GUI
- **`config/`** - Configuration management with Viper

### GUI Application Structure (`gui/`)
```
gui/
├── main.go                # GUI entry point
├── app.go                 # Wails v3 app struct with exposed methods
├── types.go               # GUI-specific type definitions
├── go.mod                 # Separate module for GUI
├── Makefile               # GUI build automation
├── static/                # Web assets (CSS, JS, images)
├── templates/             # Go HTML templates
├── frontend/              # Generated Wails bindings
└── build/                 # Build scripts and tools
```

### Core Business Logic (`core/`)
```
core/
├── service.go             # Main service orchestration
├── client.go              # Ollama API client wrapper
├── models.go              # Data models and types
├── operations.go          # Model operations (CRUD)
├── events.go              # Event system for GUI/TUI communication
└── logger.go              # Centralized logging
```

## Code Organization Principles

### Package Naming
- Use lowercase, single-word package names
- Package names should be descriptive of their purpose
- Avoid generic names like `utils` unless truly utility-focused

### File Naming Conventions
- Use snake_case for file names
- Group related functionality in single files
- Separate test files with `_test.go` suffix
- Use descriptive names that indicate file purpose

### Type Definitions
- Define types close to where they're used
- Use `types.go` for shared types within a package
- GUI-specific types in `gui/types.go`
- Core business types in `core/models.go`

### Configuration Management
- Single source of truth in `config/config.go`
- JSON configuration files in user's config directory
- Viper for configuration loading with defaults
- Environment variable support for API URLs

### Error Handling
- Use structured logging with zerolog
- Wrap errors with context using `fmt.Errorf`
- Return errors from functions, don't panic
- Log errors at appropriate levels (Error, Warn, Info, Debug)

### Testing Structure
- Unit tests alongside source files (`*_test.go`)
- Integration tests in separate directories if needed
- Use table-driven tests for multiple test cases
- Mock external dependencies (Ollama API)

## Import Organization
```go
import (
    // Standard library first
    "context"
    "fmt"
    "os"

    // Third-party packages
    "github.com/charmbracelet/bubbletea"
    "github.com/ollama/ollama/api"

    // Local packages last
    "github.com/sammcj/gollama/config"
    "github.com/sammcj/gollama/core"
)
```

## Build and Asset Management
- **TUI**: Single binary with embedded assets
- **GUI**: Wails v3 with embedded web assets using `//go:embed`
- **CSS**: Tailwind compilation via build scripts
- **Templates**: Go HTML templates embedded in binary
- **Static files**: Embedded filesystem for web assets

## Development Workflow
1. **TUI Development**: Work in root directory, use `make build` and `make run`
2. **GUI Development**: Work in `gui/` directory, use `make dev` for development builds
3. **Shared Logic**: Modify `core/` package, test with both TUI and GUI
4. **Configuration**: Update `config/` package, ensure backward compatibility
5. **Testing**: Run `go test ./...` from root for comprehensive testing
