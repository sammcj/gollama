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
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ollama/ollama/api"
	"golang.org/x/term"

	"github.com/sammcj/gollama/config"
	"github.com/sammcj/gollama/logging"
	"github.com/sammcj/gollama/vramestimator"
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
	altScreenActive   bool
	view              View
	showProgress      bool
	pullInput         textinput.Model
	pulling           bool
	pullProgress      float64
	newModelPull      bool
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
		Version = "1.21.2"
	}

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
	linkFlag := flag.Bool("L", false, "Link a model to a specific name")
	ollamaDirFlag := flag.String("ollama-dir", cfg.OllamaAPIKey, "Custom Ollama models directory")
	lmStudioDirFlag := flag.String("lm-dir", cfg.LMStudioFilePaths, "Custom LM Studio models directory")
	noCleanupFlag := flag.Bool("no-cleanup", false, "Don't cleanup broken symlinks")
	cleanupFlag := flag.Bool("cleanup", false, "Remove all symlinked models and empty directories and exit")
	searchFlag := flag.String("s", "", "Search - return a list of models that contain the search term in their name")
	unloadModelsFlag := flag.Bool("u", false, "Unload all models and exit")
	versionFlag := flag.Bool("v", false, "Print the version and exit")
	hostFlag := flag.String("h", "", "Override the config file to set the Ollama API host (e.g. http://localhost:11434)")
	editFlag := flag.Bool("e", false, "Edit a model's modelfile")

	// vRAM estimation flags
	vramFlag := flag.Bool("vram", false, "Estimate vRAM usage")
	modelIDFlag := flag.String("model", "", "Model ID for vRAM estimation")
	quantFlag := flag.String("quant", "", "Quantisation type (e.g., q4_k_m) or bits per weight (e.g., 5.0)")
	contextFlag := flag.Int("context", 0, "Context length for vRAM estimation")
	kvCacheFlag := flag.String("kvcache", "fp16", "KV cache quantisation: fp16, q8_0, or q4_0")
	memoryFlag := flag.Float64("memory", 0, "Available memory in GB for context calculation")
	quantTypeFlag := flag.String("quanttype", "gguf", "Quantisation type: gguf or exl2")

	flag.Parse()

	if *versionFlag {
		fmt.Println(Version)
		os.Exit(0)
	}

	os.Setenv("EDITOR", cfg.Editor)

	if *hostFlag != "" {
		cfg.OllamaAPIURL = *hostFlag
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
		pullInput:         textinput.New(),
		pulling:           false,
		pullProgress:      0,
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

	if *searchFlag != "" {
		searchTerms := flag.Args()
		// If no additional arguments are provided, use the searchFlag value
		if len(searchTerms) == 0 {
			searchTerms = []string{*searchFlag}
		}
		searchModels(models, searchTerms...)
		os.Exit(0)
	}

	if *linkFlag {
		// Make sure we're not running on a remote host by checking the API URL to ensure it contains localhost or 127.0.0.1
		if !isLocalhost(cfg.OllamaAPIURL) {
			fmt.Println("Error: Linking models is only supported on localhost")
			os.Exit(1)
		}
		// link all models
		for _, model := range models {
			message, err := linkModel(model.Name, cfg.LMStudioFilePaths, false, client)
			logging.InfoLogger.Println(message)
			fmt.Printf("Linking model %s\n", model.Name)
			if err != nil {
				logging.ErrorLogger.Printf("Error linking model %s: %v\n", model.Name, err)
			} else {
				logging.InfoLogger.Printf("Model %s linked\n", model.Name)
			}
		}
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

	if *editFlag {
		if flag.NArg() == 0 {
			fmt.Println("Usage: gollama -e <model_name>")
			os.Exit(1)
		}
		modelName := flag.Args()[0]
		editModelfile(client, modelName)
		os.Exit(0)
	}

	// Handle vRAM estimation flags
	if *vramFlag {
		if *modelIDFlag == "" {
			fmt.Println("Error: Model ID is required for vRAM estimation")
			os.Exit(1)
		}

		var kvCacheQuant vramestimator.KVCacheQuantisation
		switch *kvCacheFlag {
		case "fp16":
			kvCacheQuant = vramestimator.KVCacheFP16
		case "q8_0":
			kvCacheQuant = vramestimator.KVCacheQ8_0
		case "q4_0":
			kvCacheQuant = vramestimator.KVCacheQ4_0
		default:
			fmt.Printf("Invalid KV cache quantisation: %s. Using default fp16.\n", *kvCacheFlag)
			kvCacheQuant = vramestimator.KVCacheFP16
		}

		if *memoryFlag > 0 && *contextFlag == 0 && *quantFlag == "" {
			// Calculate best BPW
			bestBPW, err := vramestimator.CalculateBPW(*modelIDFlag, *memoryFlag, 0, kvCacheQuant, *quantTypeFlag, "")
			if err != nil {
				fmt.Printf("Error calculating BPW: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Best BPW for %.2f GB of memory: %v\n", *memoryFlag, bestBPW)
		} else {
			// Parse the quant flag for other operations
			bpw, err := vramestimator.ParseBPWOrQuant(*quantFlag)
			if err != nil {
				fmt.Printf("Error parsing quantisation: %v\n", err)
				os.Exit(1)
			}

			if *memoryFlag > 0 && *contextFlag == 0 {
				// Calculate maximum context
				maxContext, err := vramestimator.CalculateContext(*modelIDFlag, *memoryFlag, bpw, kvCacheQuant, "")
				if err != nil {
					fmt.Printf("Error calculating context: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("Maximum context for %.2f GB of memory: %d\n", *memoryFlag, maxContext)
			} else if *contextFlag > 0 {
				// Calculate VRAM usage
				vram, err := vramestimator.CalculateVRAM(*modelIDFlag, bpw, *contextFlag, kvCacheQuant, "")
				if err != nil {
					fmt.Printf("Error calculating VRAM: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("Estimated VRAM usage: %.2f GB\n", vram)
			} else {
				fmt.Println("Error: Invalid combination of flags. Please specify either --memory, --context, or both.")
				os.Exit(1)
			}
		}

		os.Exit(0)
	}

	// TUI App
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
