// main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ollama/ollama/api"
	"golang.org/x/term"

	"github.com/sammcj/gollama/config"
	"github.com/sammcj/gollama/logging"
	"github.com/sammcj/gollama/styles"
	"github.com/sammcj/gollama/vramestimator"
)

type AppModel struct {
	width               int
	height              int
	ollamaModelsDir     string
	cfg                 *config.Config
	inspectedModel      Model
	list                list.Model
	models              []Model
	selectedModels      []Model
	confirmDeletion     bool
	inspecting          bool
	editing             bool
	message             string
	keys                KeyMap
	client              *api.Client
	noCleanup           bool
	table               table.Model
	filterInput         tea.Model
	showTop             bool
	progress            progress.Model
	altScreenActive     bool
	view                View
	showProgress        bool
	pullInput           textinput.Model
	pulling             bool
	pullProgress        float64
	newModelPull        bool
	comparingModelfile  bool
	modelfileDiffs      []ModelfileDiff
	externalEditing     bool
	externalEditorFile  string
	externalEditorModel string
}

// TODO: Refactor: we don't need unique message types for every single action
type progressMsg struct {
	modelName string
	progress  float64
}

type runFinishedMessage struct{ err error }

type pushSuccessMsg struct {
	modelName string
}

type pushErrorMsg struct {
	err error
}

type pullSuccessMsg struct {
	modelName string
}

type pullErrorMsg struct {
	err error
}

type genericMsg struct {
	message string
}

type View int

var Version string // Version is set by the build system

func main() {
	if Version == "" {
		Version = "1.37.4"
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	// Initialise themes
	err = config.SaveThemes()
	if err != nil {
		fmt.Println("Error saving themes:", err)
		os.Exit(1)
	}

	// Load and initialise the theme
	theme, err := config.LoadTheme(cfg.Theme)
	if err != nil {
		fmt.Println("Error loading theme:", err)
		os.Exit(1)
	}
	styles.InitTheme(theme)

	err = logging.Init(cfg.LogLevel, cfg.LogFilePath)
	if err != nil {
		fmt.Println("Error initialising logging:", err)
		os.Exit(1)
	}

	listFlag := flag.Bool("l", false, "List all available Ollama models and exit")
	ollamaDirFlag := flag.String("ollama-dir", cfg.OllamaModelsDir, "Custom Ollama models directory")
	noCleanupFlag := flag.Bool("no-cleanup", false, "Don't cleanup broken symlinks")
	searchFlag := flag.String("s", "", "Search - return a list of models that contain the search term in their name")
	unloadModelsFlag := flag.Bool("u", false, "Unload all models and exit")
	versionFlag := flag.Bool("v", false, "Print the version and exit")
	hostFlag := flag.String("h", "", "Override the config file to set the Ollama API host (e.g. http://localhost:11434)")
	localHostFlag := flag.Bool("H", false, "Shortcut to connect to http://localhost:11434")
	editFlag := flag.Bool("e", false, "Edit a model's modelfile")
	logLevelFlag := flag.String("log-level", "", "Override log level (debug, info, warn, error)")
	flag.StringVar(logLevelFlag, "log", "", "Override log level (debug, info, warn, error)")
	// vRAM estimation flags
	// flag.Float64Var(&fitsVRAM, "fits", 0, "Highlight quant sizes and context sizes that fit in this amount of vRAM (in GB)")
	vramFlag := flag.String("vram", "", "Model to estimate VRAM usage for (e.g., 'qwen2:q4_0' or 'meta-llama/Llama-2-7b')")
	fitsVRAMFlag := flag.Float64("fits", 0, "Target VRAM constraint in GB (default: auto-detect)")
	contextFlag := flag.String("context", "", "Maximum context length (e.g., '32k' or '128k')")
	quantFlag := flag.String("quant", "", "Specific quantisation level (e.g., 'Q4_0', 'Q5_K_M')")
	vramToNthFlag := flag.String("vram-to-nth", "65536", "Top context length to search for (e.g., 65536, 32k, 2m)")

	flag.Parse()

	if *versionFlag {
		fmt.Println(Version)
		os.Exit(0)
	}

	if *localHostFlag {
		*hostFlag = "http://localhost:11434"
	}

	if *hostFlag != "" {
		cfg.OllamaAPIURL = *hostFlag
	}

	if *logLevelFlag != "" {
		cfg.LogLevel = *logLevelFlag
		// Reinitialise logging with the new level
		err = logging.Init(cfg.LogLevel, cfg.LogFilePath)
		if err != nil {
			fmt.Println("Error reinitialising logging with new level:", err)
			os.Exit(1)
		}
	}

	// Initialise the API client
	ctx := context.Background()
	httpClient := &http.Client{}
	url, err := url.Parse(cfg.OllamaAPIURL)

	if err != nil {
		message := fmt.Sprintf("Error parsing API URL: %v", err)
		logging.ErrorLogger.Println(message)
		fmt.Println(message)
		os.Exit(1)
	}

	// Handle --vram flag
	if *vramFlag != "" {
		modelName := *vramFlag
		logging.DebugLogger.Printf("Processing vRAM estimation for model: %s", modelName)

		// Parse the model identifier and quantisation level
		baseModel, quantLevel, err := vramestimator.ParseModelIdentifier(modelName)
		if err != nil {
			fmt.Printf("Error parsing model identifier: %v\n", err)
			os.Exit(1)
		}

		logging.DebugLogger.Printf("Parsed model identifier: base=%s, quant=%s", baseModel, quantLevel)

		// Override quantisation level if specified via flag
		if *quantFlag != "" {
			logging.DebugLogger.Printf("Overriding quantisation level from flag: %s", *quantFlag)
			quantLevel = *quantFlag
		}

		var isHuggingFaceModel = strings.Contains(baseModel, "/")
		var isOllamaModel = !isHuggingFaceModel

		// Parse the context size
		var topContext int
		var contextSource string
		if *contextFlag != "" && *contextFlag != "65536" {
			topContext, err = parseContextSize(*contextFlag)
			contextSource = "context"
		} else if *vramToNthFlag != "" {
			topContext, err = parseContextSize(*vramToNthFlag)
			contextSource = "vram-to-nth"
		} else {
			topContext = 65536
			contextSource = "default"
		}

		if err != nil {
			fmt.Printf("Error parsing context size from --%s flag: %v\n", contextSource, err)
			os.Exit(1)
		}

		logging.DebugLogger.Printf("Using context size %d from --%s", topContext, contextSource)

		// If a specific quantisation level is provided, verify it exists
		if quantLevel != "" {
			if _, exists := vramestimator.GGUFMapping[strings.ToUpper(quantLevel)]; !exists {
				fmt.Printf("Warning: Unknown quantisation level '%s'. Available levels:\n", quantLevel)
				var levels []string
				for level := range vramestimator.GGUFMapping {
					levels = append(levels, level)
				}
				sort.Strings(levels)
				for _, level := range levels {
					fmt.Printf("  - %s\n", level)
				}
				os.Exit(1)
			}
		}

		// Fetch model information from appropriate source
		var ollamaModelInfo *vramestimator.OllamaModelInfo
		if isOllamaModel {
			logging.DebugLogger.Printf("Fetching model info from Ollama API for %s", baseModel)
			ollamaModelInfo, err = vramestimator.FetchOllamaModelInfo(cfg.OllamaAPIURL, modelName)
			if err != nil {
				fmt.Printf("Error: Could not fetch Ollama model info: %v\n", err)
				os.Exit(1)
			}
		} else {
			logging.DebugLogger.Printf("Using HuggingFace model ID: %s", baseModel)
		}

		// Generate and display the table
		table, err := vramestimator.GenerateQuantTable(baseModel, *fitsVRAMFlag, ollamaModelInfo, topContext)
		if err != nil {
			fmt.Printf("Error generating VRAM estimation table: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(vramestimator.PrintFormattedTable(table))
		os.Exit(0)
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
		client:          client,
		keys:            *keys,
		models:          groupedModels,
		width:           width,
		height:          height,
		ollamaModelsDir: *ollamaDirFlag,
		noCleanup:       *noCleanupFlag,
		cfg:             &cfg,
		progress:        progress.New(progress.WithDefaultGradient()),
		pullInput:       textinput.New(),
		pulling:         false,
		pullProgress:    0,
	}

	if *ollamaDirFlag == "" {
		app.ollamaModelsDir = cfg.OllamaModelsDir
	}

	if *listFlag {
		listModels(models)
		os.Exit(0)
	}

	if *searchFlag != "" {
		searchTerms := flag.Args()
		// If no additional arguments are provided, use the searchFlag value
		if len(searchTerms) == 0 {
			searchTerms = []string{*searchFlag}
		}
		searchModels(models, searchTerms...)
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
				unloadedModels = append(unloadedModels, styles.WarningStyle().Render(model.Name))
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

	if *editFlag {
		if flag.NArg() == 0 {
			fmt.Println("Usage: gollama -e <model_name>")
			os.Exit(1)
		}
		modelName := flag.Args()[0]
		editModelfile(client, modelName)
		os.Exit(0)
	}

	// TUI App
	l := list.New(items, NewItemDelegate(&app), width, height-5)
	l.Title = fmt.Sprintf("Ollama Models - Connected to %s", cfg.OllamaAPIURL)
	l.Help.Styles.ShortDesc.Bold(true)
	l.Help.Styles.ShortDesc.UnsetFaint()
	l.Help.Styles.ShortDesc = styles.PromptStyle()
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
			keys.CopyModel,
			keys.PushModel,
			keys.Top,
			keys.EditModel,
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

	p.ReleaseTerminal()
}
