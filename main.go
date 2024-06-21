// main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ollama/ollama/api"
	"golang.org/x/term"

	"github.com/sammcj/gollama/config"
	"github.com/sammcj/gollama/logging"
)

type AppModel struct {
	width             int
	height            int
	ollamaModelsDir   string
	cfg               *config.Config
	inspectedModel    Model
	list              list.Model
	models            []Model
	selectedModels    []Model
	confirmDeletion   bool
	inspecting        bool
	editing           bool
	message           string
	keys              KeyMap
	client            *api.Client
	lmStudioModelsDir string
	noCleanup         bool
	table             table.Model
	filterInput       tea.Model
	showTop           bool
	progress          progress.Model
	altscreenActive   bool
	view              View
	showProgress      bool
	needsRefresh      bool
}

type progressMsg struct {
	modelName string
}

type runFinishedMessage struct{ err error }

type pushSuccessMsg struct {
	modelName string
}

type pushErrorMsg struct {
	err error
}

type genericMsg struct {
	message string
}

var (
	Version string // Version will be set during the build process
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	err = logging.Init(cfg.LogLevel, cfg.LogFilePath)
	if err != nil {
		fmt.Println("Error initializing logging:", err)
		os.Exit(1)
	}

	listFlag := flag.Bool("l", false, "List all available Ollama models and exit")
	ollamaDirFlag := flag.String("ollama-dir", cfg.OllamaAPIKey, "Custom Ollama models directory")
	lmStudioDirFlag := flag.String("lm-dir", cfg.LMStudioFilePaths, "Custom LM Studio models directory")
	noCleanupFlag := flag.Bool("no-cleanup", false, "Don't cleanup broken symlinks")
	cleanupFlag := flag.Bool("cleanup", false, "Remove all symlinked models and empty directories and exit")
	unloadModelsFlag := flag.Bool("u", false, "Unload all models and exit")
	versionFlag := flag.Bool("v", false, "Print the version and exit")

	flag.Parse()

	if *versionFlag {
		fmt.Println(Version)
		os.Exit(0)
	}

	os.Setenv("EDITOR", cfg.Editor)

	// Initialize the API client
	ctx := context.Background()
	httpClient := &http.Client{}
	url, err := url.Parse(cfg.OllamaAPIURL)

	if err != nil {
		message := fmt.Sprintf("Error parsing API URL: %v", err)
		logging.ErrorLogger.Println(message)
		fmt.Println(message)
		os.Exit(1)
	}

	client := api.NewClient(url, httpClient)

	resp, err := client.List(ctx)
	if err != nil {
		message := fmt.Sprintf("Error fetching models:\n- Error: %v\n- Configured API URL: %v", err, cfg.OllamaAPIURL)
		logging.ErrorLogger.Println(message)
		fmt.Println(message)
		os.Exit(1)
	}

	models := parseAPIResponse(resp)

	modelMap := make(map[string][]Model)
	for _, model := range models {
		model.Size = normalizeSize(model.Size)
		modelMap[model.ID] = append(modelMap[model.ID], model)
	}

	groupedModels := make([]Model, 0)
	for _, group := range modelMap {
		groupedModels = append(groupedModels, group...)
	}

	switch cfg.SortOrder {
	case "name":
		sort.Slice(groupedModels, func(i, j int) bool {
			return groupedModels[i].Name < groupedModels[j].Name
		})
	case "size":
		sort.Slice(groupedModels, func(i, j int) bool {
			return groupedModels[i].Size > groupedModels[j].Size
		})
	case "modified":
		sort.Slice(groupedModels, func(i, j int) bool {
			return groupedModels[i].Modified.After(groupedModels[j].Modified)
		})
	case "family":
		sort.Slice(groupedModels, func(i, j int) bool {
			return groupedModels[i].Family < groupedModels[j].Family
		})
	}

	items := make([]list.Item, len(groupedModels))
	for i, model := range groupedModels {
		items[i] = model
	}

	keys := NewKeyMap()

	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 80, 24
	}

	app := AppModel{
		client:            client,
		keys:              *keys,
		models:            groupedModels,
		width:             width,
		height:            height,
		ollamaModelsDir:   *ollamaDirFlag,
		lmStudioModelsDir: *lmStudioDirFlag,
		noCleanup:         *noCleanupFlag,
		cfg:               &cfg,
		progress:          progress.New(progress.WithDefaultGradient()),
	}

	if *ollamaDirFlag == "" {
		app.ollamaModelsDir = filepath.Join(os.Getenv("HOME"), ".ollama", "models")
	}
	if *lmStudioDirFlag == "" {
		app.lmStudioModelsDir = filepath.Join(os.Getenv("HOME"), ".cache", "lm-studio", "models")
	}

	if *listFlag {
		listModels(models)
		os.Exit(0)
	}

	if *cleanupFlag {
		cleanupSymlinkedModels(app.lmStudioModelsDir)
		os.Exit(0)
	}

	if *unloadModelsFlag {
		// get any loaded models
		client := app.client

		ctx := context.Background()
		loadedModels, err := client.ListRunning(ctx)
		if err != nil {
			logging.ErrorLogger.Printf("Error fetching running models: %v", err)
			os.Exit(1)
		}

		// unload the models
		var unloadedModels []string
		for _, model := range loadedModels.Models {
			_, err := unloadModel(client, model.Name)
			if err != nil {
				logging.ErrorLogger.Printf("Error unloading model %s: %v\n", model.Name, err)
			} else {
				unloadedModels = append(unloadedModels, lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB6C1")).Render(model.Name))
				logging.InfoLogger.Printf("Model %s unloaded\n", model.Name)
			}
		}
		if len(unloadedModels) == 0 {
			fmt.Println("No models to unload")
		} else {
			logging.InfoLogger.Printf("Unloaded models: %v\n", unloadedModels)
			fmt.Printf("Unloaded models: %v\n", unloadedModels)
		}
		os.Exit(0)
	}

	l := list.New(items, NewItemDelegate(&app), width, height-5)
	l.Title = "Ollama Models"
	l.Help.Styles.ShortDesc.Bold(true)
	l.Help.Styles.ShortDesc.UnsetFaint()
	l.Help.Styles.ShortDesc.Foreground(lipgloss.Color("#FF00FF"))
	l.Help.Styles.ShortDesc.Background(lipgloss.Color("#000000"))
	l.Help.Styles.ShortDesc.Width(20)
	l.Help.Styles.ShortDesc.Padding(0, 1)
	l.Help.Styles.ShortDesc.Margin(0, 1)
	l.Help.Styles.ShortDesc.Border(lipgloss.Border{Left: " ", Right: " "})

	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			keys.Space,
			keys.Delete,
			keys.SortByName,
			keys.SortBySize,
			keys.SortByModified,
			keys.SortByQuant,
			keys.SortByFamily,
			keys.RunModel,
			keys.ConfirmYes,
			keys.ConfirmNo,
			keys.LinkModel,
			keys.LinkAllModels,
			keys.CopyModel,
			keys.PushModel,
			keys.Top,
			keys.UpdateModel,
			keys.Help,
		}
	}

	app.list = l

	p := tea.NewProgram(&app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		logging.ErrorLogger.Printf("Error: %v", err)
	} else {
		fmt.Print("\033[H\033[2J")
	}

	// Throw a warning if the users terminal cannot display colours
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Println("Warning: Your terminal does not support colours. Please consider using a terminal that does.")
	}

	cfg.SortOrder = keys.GetSortOrder()
	if err := config.SaveConfig(cfg); err != nil {
		panic(err)
	}

	p.ReleaseTerminal()
}
