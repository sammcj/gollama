package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ollama/ollama/api"
	"golang.org/x/term"

	"gollama/config"
	"gollama/logging"
)

type AppModel struct {
	client              *api.Client
	list                list.Model
	keys                *KeyMap
	models              []Model
	width               int
	height              int
	confirmDeletion     bool
	selectedForDeletion []Model
	ollamaModelsDir     string
	lmStudioModelsDir   string
	noCleanup           bool
	cfg                 *config.Config
	message             string
}

func main() {
	var version = "1.0.5"

	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	// Initialize logging
	err = logging.Init(cfg.LogLevel, cfg.LogFilePath)
	if err != nil {
		fmt.Println("Error initializing logging:", err)
		os.Exit(1)
	}

	logging.InfoLogger.Printf("Starting gollama version %s\n", version)

	// Parse command-line arguments
	listFlag := flag.Bool("l", false, "List all available Ollama models and exit")
	ollamaDirFlag := flag.String("ollama-dir", cfg.OllamaAPIKey, "Custom Ollama models directory")
	lmStudioDirFlag := flag.String("lm-dir", cfg.LMStudioFilePaths, "Custom LM Studio models directory")
	noCleanupFlag := flag.Bool("no-cleanup", false, "Don't cleanup broken symlinks")
	cleanupFlag := flag.Bool("cleanup", false, "Remove all symlinked models and empty directories and exit")
	flag.Parse()

	client, err := api.ClientFromEnvironment()
	if err != nil {
		logging.ErrorLogger.Println("Error creating API client:", err)
		return
	}

	ctx := context.Background()
	resp, err := client.List(ctx)
	if err != nil {
		logging.ErrorLogger.Println("Error fetching models:", err)
		return
	}

	logging.InfoLogger.Println("Fetched models from API")
	models := parseAPIResponse(resp)

	// Group models by ID and normalize sizes to GB
	modelMap := make(map[string][]Model)
	for _, model := range models {
		model.Size = normalizeSize(model.Size)
		modelMap[model.ID] = append(modelMap[model.ID], model)
	}

	// Flatten the map into a slice
	groupedModels := make([]Model, 0)
	for _, group := range modelMap {
		groupedModels = append(groupedModels, group...)
	}

	// Apply sorting order from config
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
		width, height = 80, 24 // default size if terminal size can't be determined
	}
	app := AppModel{
		client:            client,
		keys:              keys,
		models:            groupedModels,
		width:             width,
		height:            height,
		ollamaModelsDir:   *ollamaDirFlag,
		lmStudioModelsDir: *lmStudioDirFlag,
		noCleanup:         *noCleanupFlag,
		cfg:               &cfg,
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

	l := list.New(items, NewItemDelegate(&app), width, height-5)

	l.Title = "Ollama Models"
	l.InfiniteScrolling = true
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)

	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			keys.Space,
			keys.Delete,
			keys.SortByName,
			keys.SortBySize,
			keys.SortByModified,
			keys.RunModel,
			keys.ConfirmYes,
			keys.ConfirmNo,
			keys.LinkModel,
			keys.LinkAllModels,
		}
	}

	app.list = l

	p := tea.NewProgram(&app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		logging.ErrorLogger.Printf("Error: %v", err)
	} else {
		// Clear the terminal screen again to refresh the application view
		fmt.Print("\033[H\033[2J")
	}

	// Save the updated configuration
	cfg.SortOrder = keys.GetSortOrder()
	if err := config.SaveConfig(cfg); err != nil {
		panic(err)
	}
}
