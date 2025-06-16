package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/sammcj/gollama/core"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*
var templateFiles embed.FS

// App struct
type App struct {
	ctx       context.Context
	service   *core.GollamaService
	templates *template.Template
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// OnStartup is called when the app starts up
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx

	// Initialize service
	cfg := core.ServiceConfig{
		OllamaAPIURL: "http://localhost:11434",
		LogLevel:     "info",
		Context:      ctx,
	}

	service, err := core.NewGollamaService(cfg)
	if err != nil {
		fmt.Printf("Failed to initialize service: %v\n", err)
		return
	}

	a.service = service

	// Load templates with custom functions
	funcMap := template.FuncMap{
		"formatSize":     formatSize,
		"formatTime":     formatTime,
		"formatDuration": formatDuration,
		"add":            func(a, b int) int { return a + b },
		"sub":            func(a, b int) int { return a - b },
		"mul":            func(a, b int) int { return a * b },
		"div":            func(a, b int) int { return a / b },
		"mod":            func(a, b int) int { return a % b },
		"eq":             func(a, b interface{}) bool { return a == b },
		"ne":             func(a, b interface{}) bool { return a != b },
		"lt":             func(a, b int) bool { return a < b },
		"le":             func(a, b int) bool { return a <= b },
		"gt":             func(a, b int) bool { return a > b },
		"ge":             func(a, b int) bool { return a >= b },
		"contains":       strings.Contains,
		"hasPrefix":      strings.HasPrefix,
		"hasSuffix":      strings.HasSuffix,
		"toLower":        strings.ToLower,
		"toUpper":        strings.ToUpper,
		"trim":           strings.TrimSpace,
	}

	a.templates = template.Must(template.New("").Funcs(funcMap).ParseFS(templateFiles, "templates/*.html"))

	fmt.Println("Gollama GUI initialized successfully")
}

// OnShutdown is called when the app is shutting down
func (a *App) OnShutdown(ctx context.Context) {
	if a.service != nil {
		a.service.Close()
	}
}

// GetModels returns all models for the frontend
func (a *App) GetModels() ([]GuiModel, error) {
	models, err := a.service.ListModels()
	if err != nil {
		return nil, err
	}

	// Get running models to mark them as running
	runningModels, _ := a.service.GetRunningModels()
	runningMap := make(map[string]bool)
	for _, rm := range runningModels {
		runningMap[rm.Name] = true
	}

	var guiModels []GuiModel
	for _, model := range models {
		guiModels = append(guiModels, GuiModel{
			Name:              model.Name,
			ID:                model.Name, // Use name as ID for now
			Size:              float64(model.Size),
			SizeFormatted:     formatSize(model.Size),
			QuantizationLevel: model.Details.QuantizationLevel,
			Modified:          model.ModifiedAt,
			ModifiedFormatted: formatTime(model.ModifiedAt),
			Family:            model.Details.Family,
			ParameterSize:     model.Details.ParameterSize,
			IsRunning:         runningMap[model.Name],
			Status:            model.Status,
		})
	}

	return guiModels, nil
}

// GetRunningModels returns currently running models
func (a *App) GetRunningModels() ([]GuiRunningModel, error) {
	models, err := a.service.GetRunningModels()
	if err != nil {
		return nil, err
	}

	var guiModels []GuiRunningModel
	for _, model := range models {
		vramGB := float64(model.SizeVRAM) / (1024 * 1024 * 1024) // Convert bytes to GB
		guiModels = append(guiModels, GuiRunningModel{
			Name:               model.Name,
			VRAMUsage:          vramGB,
			VRAMFormatted:      formatSize(model.SizeVRAM),
			LoadedAt:           model.LoadedAt,
			LoadedAtFormatted:  formatTime(model.LoadedAt),
			ExpiresAt:          model.ExpiresAt,
			ExpiresAtFormatted: formatTime(model.ExpiresAt),
		})
	}

	return guiModels, nil
}

// RunModel starts a model
func (a *App) RunModel(name string) error {
	return a.service.RunModel(name)
}

// DeleteModel deletes a model
func (a *App) DeleteModel(name string) error {
	return a.service.DeleteModel(name)
}

// UnloadModel unloads a model
func (a *App) UnloadModel(name string) error {
	return a.service.UnloadModel(name)
}

// GetModelDetails returns detailed information about a model
func (a *App) GetModelDetails(name string) (*ModelDetailsData, error) {
	info, err := a.service.GetModelInfo(name)
	if err != nil {
		return nil, err
	}

	// Get running models to check status
	runningModels, _ := a.service.GetRunningModels()
	isRunning := false
	for _, rm := range runningModels {
		if rm.Name == name {
			isRunning = true
			break
		}
	}

	guiModel := GuiModel{
		Name:              info.Model.Name,
		ID:                info.Model.Name,
		Size:              float64(info.Model.Size),
		SizeFormatted:     formatSize(info.Model.Size),
		QuantizationLevel: info.Model.Details.QuantizationLevel,
		Modified:          info.Model.ModifiedAt,
		ModifiedFormatted: formatTime(info.Model.ModifiedAt),
		Family:            info.Model.Details.Family,
		ParameterSize:     info.Model.Details.ParameterSize,
		IsRunning:         isRunning,
		Status:            info.Model.Status,
	}

	return &ModelDetailsData{
		Model:   guiModel,
		Details: info,
	}, nil
}

// EstimateVRAM estimates vRAM usage for a model
func (a *App) EstimateVRAM(request VRAMEstimateRequest) (*VRAMEstimateResponse, error) {
	constraints := core.VRAMConstraints{
		AvailableVRAM: request.VRAMAvailable,
		ContextLength: request.ContextLength,
		Quantization:  request.Quantization,
	}

	estimation, err := a.service.EstimateVRAM(request.ModelName, constraints)
	if err != nil {
		return &VRAMEstimateResponse{
			ModelName: request.ModelName,
			Error:     err.Error(),
		}, nil
	}

	// Generate recommendations
	var recommendations []VRAMRecommendation
	for quantType, estimate := range estimation.Estimates {
		for contextStr, vramUsage := range estimate.Contexts {
			contextLength, _ := strconv.Atoi(strings.TrimSuffix(contextStr, "k"))
			contextLength *= 1024 // Convert from k to actual tokens

			recommendations = append(recommendations, VRAMRecommendation{
				Quantization:  quantType,
				VRAMRequired:  vramUsage,
				ContextLength: contextLength,
				Fits:          request.VRAMAvailable == 0 || vramUsage <= request.VRAMAvailable,
			})
		}
	}

	return &VRAMEstimateResponse{
		ModelName:       request.ModelName,
		Estimation:      estimation,
		Recommendations: recommendations,
	}, nil
}

// GetConfig returns the current configuration
func (a *App) GetConfig() *SettingsData {
	config := a.service.GetConfig()
	return &SettingsData{
		OllamaAPIURL:    config.OllamaAPIURL,
		Theme:           config.Theme,
		AutoRefresh:     config.AutoRefresh,
		RefreshInterval: config.RefreshInterval,
		WindowWidth:     config.WindowWidth,
		WindowHeight:    config.WindowHeight,
		DefaultView:     config.DefaultView,
		ShowSystemTray:  config.ShowSystemTray,
	}
}

// UpdateConfig updates the configuration
func (a *App) UpdateConfig(settings SettingsData) error {
	config := a.service.GetConfig()

	// Update fields
	config.OllamaAPIURL = settings.OllamaAPIURL
	config.Theme = settings.Theme
	config.AutoRefresh = settings.AutoRefresh
	config.RefreshInterval = settings.RefreshInterval
	config.WindowWidth = settings.WindowWidth
	config.WindowHeight = settings.WindowHeight
	config.DefaultView = settings.DefaultView
	config.ShowSystemTray = settings.ShowSystemTray

	return a.service.UpdateConfig(config)
}

// HealthCheck verifies the service is healthy
func (a *App) HealthCheck() error {
	return a.service.HealthCheck()
}

// Helper functions

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}
