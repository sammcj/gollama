package main

import (
	"fmt"
	"time"

	"github.com/sammcj/gollama/config"
	"github.com/sammcj/gollama/core"
)

// ModelDTO represents a model with JavaScript-friendly formatting
type ModelDTO struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Size              int64     `json:"size"`
	SizeFormatted     string    `json:"size_formatted"`
	Family            string    `json:"family"`
	ParameterSize     string    `json:"parameter_size"`
	QuantizationLevel string    `json:"quantization_level"`
	Modified          time.Time `json:"modified"`
	ModifiedFormatted string    `json:"modified_formatted"`
	IsRunning         bool      `json:"is_running"`
	Digest            string    `json:"digest"`
	Status            string    `json:"status"`
}

// RunningModelDTO represents a running model with JavaScript-friendly formatting
type RunningModelDTO struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	LoadedAt  time.Time `json:"loaded_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ConfigDTO represents configuration with JavaScript-friendly formatting
type ConfigDTO struct {
	OllamaAPIURL    string `json:"ollama_api_url"`
	LogLevel        string `json:"log_level"`
	AutoRefresh     bool   `json:"auto_refresh"`
	RefreshInterval int    `json:"refresh_interval"`
	WindowWidth     int    `json:"window_width"`
	WindowHeight    int    `json:"window_height"`
	DefaultView     string `json:"default_view"`
	ShowSystemTray  bool   `json:"show_system_tray"`
	Theme           string `json:"theme"`
	Editor          string `json:"editor"`
	SortOrder       string `json:"sort_order"`
	StripString     string `json:"strip_string"`
	DockerContainer string `json:"docker_container"`
}

// VRAMConstraintsDTO represents vRAM estimation constraints
type VRAMConstraintsDTO struct {
	AvailableVRAM  float64 `json:"available_vram"`
	Context        int     `json:"context"`
	Quantization   string  `json:"quantization"`
	BatchSize      int     `json:"batch_size"`
	SequenceLength int     `json:"sequence_length"`
}

// VRAMEstimationDTO represents vRAM estimation results
type VRAMEstimationDTO struct {
	ModelName        string                     `json:"model_name"`
	RequiredVRAM     float64                    `json:"required_vram"`
	AvailableVRAM    float64                    `json:"available_vram"`
	CanRun           bool                       `json:"can_run"`
	RecommendedQuant string                     `json:"recommended_quant"`
	Details          string                     `json:"details"`
	ModelSize        float64                    `json:"model_size"`
	ContextSize      float64                    `json:"context_size"`
	TotalSize        float64                    `json:"total_size"`
	Quantization     string                     `json:"quantization"`
	ContextLength    int                        `json:"context_length"`
	Recommendations  []VRAMRecommendationDTO    `json:"recommendations"`
	Breakdown        VRAMBreakdownDTO           `json:"breakdown"`
	Estimates        map[string]VRAMEstimateDTO `json:"estimates"`
}

// VRAMRecommendationDTO represents a vRAM recommendation
type VRAMRecommendationDTO struct {
	Quantization  string  `json:"quantization"`
	ContextLength int     `json:"context_length"`
	VRAMUsage     float64 `json:"vram_usage"`
	Description   string  `json:"description"`
}

// VRAMBreakdownDTO represents detailed vRAM usage breakdown
type VRAMBreakdownDTO struct {
	ModelWeights float64 `json:"model_weights"`
	KVCache      float64 `json:"kv_cache"`
	Activations  float64 `json:"activations"`
	Overhead     float64 `json:"overhead"`
	Total        float64 `json:"total"`
}

// VRAMEstimateDTO represents a single vRAM estimation row
type VRAMEstimateDTO struct {
	Quantization  string             `json:"quantization"`
	BitsPerWeight float64            `json:"bits_per_weight"`
	Contexts      map[string]float64 `json:"contexts"` // context length -> vRAM usage
}

// ModelInfoDTO represents detailed model information
type ModelInfoDTO struct {
	Model      ModelDTO               `json:"model"`
	Modelfile  string                 `json:"modelfile"`
	Template   string                 `json:"template"`
	System     string                 `json:"system"`
	Parameters map[string]interface{} `json:"parameters"`
	VRAMUsage  *VRAMEstimationDTO     `json:"vram_usage,omitempty"`
}

// OperationProgressDTO represents the progress of a long-running operation
type OperationProgressDTO struct {
	Status     string  `json:"status"`
	Completed  int64   `json:"completed"`
	Total      int64   `json:"total"`
	Percentage float64 `json:"percentage"`
	Message    string  `json:"message"`
}

// ModelOperationDTO represents an operation that can be performed on a model
type ModelOperationDTO struct {
	Type        string                 `json:"type"`
	ModelName   string                 `json:"model_name"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Progress    *OperationProgressDTO  `json:"progress,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// APIErrorDTO represents a structured error response
type APIErrorDTO struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// HealthCheckDTO represents the health status of the service
type HealthCheckDTO struct {
	Status    string                      `json:"status"`
	Timestamp time.Time                   `json:"timestamp"`
	Version   string                      `json:"version"`
	Services  map[string]ServiceStatusDTO `json:"services"`
}

// ServiceStatusDTO represents the status of individual services
type ServiceStatusDTO struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// Conversion functions from core models to DTOs

// ConvertToModelDTO converts a core.Model to ModelDTO
func ConvertToModelDTO(model core.Model) ModelDTO {
	return ModelDTO{
		ID:                model.ID,
		Name:              model.Name,
		Size:              model.Size,
		SizeFormatted:     formatSize(model.Size),
		Family:            model.Details.Family,
		ParameterSize:     model.Details.ParameterSize,
		QuantizationLevel: model.Details.QuantizationLevel,
		Modified:          model.ModifiedAt,
		ModifiedFormatted: model.ModifiedAt.Format("2006-01-02 15:04:05"),
		IsRunning:         model.Status == "running",
		Digest:            model.Digest,
		Status:            model.Status,
	}
}

// ConvertToModelDTOs converts a slice of core.Model to []ModelDTO
func ConvertToModelDTOs(models []core.Model) []ModelDTO {
	dtos := make([]ModelDTO, len(models))
	for i, model := range models {
		dtos[i] = ConvertToModelDTO(model)
	}
	return dtos
}

// ConvertToRunningModelDTO converts a core.RunningModel to RunningModelDTO
func ConvertToRunningModelDTO(model core.RunningModel) RunningModelDTO {
	return RunningModelDTO{
		Name:      model.Name,
		Size:      model.Size,
		LoadedAt:  model.LoadedAt,
		ExpiresAt: model.ExpiresAt,
	}
}

// ConvertToRunningModelDTOs converts a slice of core.RunningModel to []RunningModelDTO
func ConvertToRunningModelDTOs(models []core.RunningModel) []RunningModelDTO {
	dtos := make([]RunningModelDTO, len(models))
	for i, model := range models {
		dtos[i] = ConvertToRunningModelDTO(model)
	}
	return dtos
}

// ConvertToConfigDTO converts a config.Config to ConfigDTO
func ConvertToConfigDTO(cfg *config.Config) *ConfigDTO {
	return &ConfigDTO{
		OllamaAPIURL:    cfg.OllamaAPIURL,
		LogLevel:        cfg.LogLevel,
		AutoRefresh:     cfg.AutoRefresh,
		RefreshInterval: cfg.RefreshInterval,
		WindowWidth:     cfg.WindowWidth,
		WindowHeight:    cfg.WindowHeight,
		DefaultView:     cfg.DefaultView,
		ShowSystemTray:  cfg.ShowSystemTray,
		Theme:           cfg.Theme,
		Editor:          cfg.Editor,
		SortOrder:       cfg.SortOrder,
		StripString:     cfg.StripString,
		DockerContainer: cfg.DockerContainer,
	}
}

// ConvertFromConfigDTO converts a ConfigDTO back to config.Config
func ConvertFromConfigDTO(dto *ConfigDTO) *config.Config {
	return &config.Config{
		OllamaAPIURL:    dto.OllamaAPIURL,
		LogLevel:        dto.LogLevel,
		AutoRefresh:     dto.AutoRefresh,
		RefreshInterval: dto.RefreshInterval,
		WindowWidth:     dto.WindowWidth,
		WindowHeight:    dto.WindowHeight,
		DefaultView:     dto.DefaultView,
		ShowSystemTray:  dto.ShowSystemTray,
		Theme:           dto.Theme,
		Editor:          dto.Editor,
		SortOrder:       dto.SortOrder,
		StripString:     dto.StripString,
		DockerContainer: dto.DockerContainer,
	}
}

// ConvertToVRAMEstimationDTO converts a core.VRAMEstimation to VRAMEstimationDTO
func ConvertToVRAMEstimationDTO(estimation *core.VRAMEstimation, modelName string, availableVRAM float64) *VRAMEstimationDTO {
	if estimation == nil {
		return nil
	}

	dto := &VRAMEstimationDTO{
		ModelName:       modelName,
		RequiredVRAM:    estimation.TotalSize,
		AvailableVRAM:   availableVRAM,
		CanRun:          estimation.TotalSize <= availableVRAM,
		ModelSize:       estimation.ModelSize,
		ContextSize:     estimation.ContextSize,
		TotalSize:       estimation.TotalSize,
		Quantization:    estimation.Quantization,
		ContextLength:   estimation.ContextLength,
		Recommendations: ConvertToVRAMRecommendationDTOs(estimation.Recommendations),
		Breakdown:       ConvertToVRAMBreakdownDTO(estimation.Breakdown),
		Estimates:       ConvertToVRAMEstimateDTOs(estimation.Estimates),
	}

	// Set recommended quantization if model doesn't fit
	if !dto.CanRun && len(dto.Recommendations) > 0 {
		dto.RecommendedQuant = dto.Recommendations[0].Quantization
		dto.Details = dto.Recommendations[0].Description
	} else if dto.CanRun {
		dto.Details = "Model fits within available vRAM"
	} else {
		dto.Details = "Model may not fit within available vRAM"
	}

	return dto
}

// ConvertToVRAMRecommendationDTOs converts []core.VRAMRecommendation to []VRAMRecommendationDTO
func ConvertToVRAMRecommendationDTOs(recommendations []core.VRAMRecommendation) []VRAMRecommendationDTO {
	dtos := make([]VRAMRecommendationDTO, len(recommendations))
	for i, rec := range recommendations {
		dtos[i] = VRAMRecommendationDTO{
			Quantization:  rec.Quantization,
			ContextLength: rec.ContextLength,
			VRAMUsage:     rec.VRAMUsage,
			Description:   rec.Description,
		}
	}
	return dtos
}

// ConvertToVRAMBreakdownDTO converts core.VRAMBreakdown to VRAMBreakdownDTO
func ConvertToVRAMBreakdownDTO(breakdown core.VRAMBreakdown) VRAMBreakdownDTO {
	return VRAMBreakdownDTO{
		ModelWeights: breakdown.ModelWeights,
		KVCache:      breakdown.KVCache,
		Activations:  breakdown.Activations,
		Overhead:     breakdown.Overhead,
		Total:        breakdown.Total,
	}
}

// ConvertToVRAMEstimateDTOs converts map[string]core.VRAMEstimateRow to map[string]VRAMEstimateDTO
func ConvertToVRAMEstimateDTOs(estimates map[string]core.VRAMEstimateRow) map[string]VRAMEstimateDTO {
	dtos := make(map[string]VRAMEstimateDTO)
	for key, estimate := range estimates {
		// Convert map[string]float64 to map[string]float64 (contexts)
		contexts := make(map[string]float64)
		for ctxKey, ctxValue := range estimate.Contexts {
			contexts[ctxKey] = ctxValue
		}

		dtos[key] = VRAMEstimateDTO{
			Quantization:  estimate.Quantization,
			BitsPerWeight: estimate.BitsPerWeight,
			Contexts:      contexts,
		}
	}
	return dtos
}

// ConvertToModelInfoDTO converts core.EnhancedModelInfo to ModelInfoDTO
func ConvertToModelInfoDTO(info *core.EnhancedModelInfo, availableVRAM float64) *ModelInfoDTO {
	if info == nil {
		return nil
	}

	return &ModelInfoDTO{
		Model:      ConvertToModelDTO(info.Model),
		Modelfile:  info.Modelfile,
		Template:   info.Template,
		System:     info.System,
		Parameters: info.Parameters,
		VRAMUsage:  ConvertToVRAMEstimationDTO(info.VRAMUsage, info.Model.Name, availableVRAM),
	}
}

// ConvertToOperationProgressDTO converts core.OperationProgress to OperationProgressDTO
func ConvertToOperationProgressDTO(progress *core.OperationProgress) *OperationProgressDTO {
	if progress == nil {
		return nil
	}

	return &OperationProgressDTO{
		Status:     progress.Status,
		Completed:  progress.Completed,
		Total:      progress.Total,
		Percentage: progress.Percentage,
		Message:    progress.Message,
	}
}

// ConvertToModelOperationDTO converts core.ModelOperation to ModelOperationDTO
func ConvertToModelOperationDTO(operation core.ModelOperation) ModelOperationDTO {
	return ModelOperationDTO{
		Type:        operation.Type,
		ModelName:   operation.ModelName,
		Parameters:  operation.Parameters,
		Progress:    ConvertToOperationProgressDTO(operation.Progress),
		Error:       operation.Error,
		StartedAt:   operation.StartedAt,
		CompletedAt: operation.CompletedAt,
	}
}

// ConvertToModelOperationDTOs converts []core.ModelOperation to []ModelOperationDTO
func ConvertToModelOperationDTOs(operations []core.ModelOperation) []ModelOperationDTO {
	dtos := make([]ModelOperationDTO, len(operations))
	for i, op := range operations {
		dtos[i] = ConvertToModelOperationDTO(op)
	}
	return dtos
}

// Validation functions for DTOs

// Validate validates the ConfigDTO and returns any validation errors
func (dto *ConfigDTO) Validate() []string {
	var errors []string

	if dto.OllamaAPIURL == "" {
		errors = append(errors, "Ollama API URL is required")
	}

	if dto.RefreshInterval < 1 {
		errors = append(errors, "Refresh interval must be at least 1 second")
	}

	if dto.WindowWidth < 400 {
		errors = append(errors, "Window width must be at least 400 pixels")
	}

	if dto.WindowHeight < 300 {
		errors = append(errors, "Window height must be at least 300 pixels")
	}

	validViews := []string{"models", "running", "settings", "vram"}
	isValidView := false
	for _, view := range validViews {
		if dto.DefaultView == view {
			isValidView = true
			break
		}
	}
	if !isValidView {
		errors = append(errors, "Default view must be one of: models, running, settings, vram")
	}

	validLogLevels := []string{"debug", "info", "warn", "error"}
	isValidLogLevel := false
	for _, level := range validLogLevels {
		if dto.LogLevel == level {
			isValidLogLevel = true
			break
		}
	}
	if !isValidLogLevel {
		errors = append(errors, "Log level must be one of: debug, info, warn, error")
	}

	return errors
}

// Validate validates the VRAMConstraintsDTO and returns any validation errors
func (dto *VRAMConstraintsDTO) Validate() []string {
	var errors []string

	if dto.AvailableVRAM <= 0 {
		errors = append(errors, "Available vRAM must be greater than 0")
	}

	if dto.Context < 512 {
		errors = append(errors, "Context length must be at least 512")
	}

	if dto.Context > 131072 {
		errors = append(errors, "Context length cannot exceed 131072")
	}

	if dto.BatchSize < 1 {
		errors = append(errors, "Batch size must be at least 1")
	}

	if dto.SequenceLength < 1 {
		errors = append(errors, "Sequence length must be at least 1")
	}

	return errors
}

// Error handling utilities

// NewAPIError creates a new APIErrorDTO
func NewAPIError(code, message, details string) *APIErrorDTO {
	return &APIErrorDTO{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// NewHealthCheckDTO creates a new HealthCheckDTO with current timestamp
func NewHealthCheckDTO(status, version string) *HealthCheckDTO {
	return &HealthCheckDTO{
		Status:    status,
		Timestamp: time.Now(),
		Version:   version,
		Services:  make(map[string]ServiceStatusDTO),
	}
}

// AddService adds a service status to the health check
func (dto *HealthCheckDTO) AddService(name, status, message, latency string) {
	dto.Services[name] = ServiceStatusDTO{
		Status:  status,
		Message: message,
		Latency: latency,
	}
}

// IsHealthy returns true if all services are healthy
func (dto *HealthCheckDTO) IsHealthy() bool {
	if dto.Status != "healthy" {
		return false
	}

	for _, service := range dto.Services {
		if service.Status != "healthy" {
			return false
		}
	}

	return true
}

// Utility functions for common operations

// GetModelByID finds a model DTO by ID in a slice
func GetModelByID(models []ModelDTO, id string) *ModelDTO {
	for i := range models {
		if models[i].ID == id {
			return &models[i]
		}
	}
	return nil
}

// GetModelByName finds a model DTO by name in a slice
func GetModelByName(models []ModelDTO, name string) *ModelDTO {
	for i := range models {
		if models[i].Name == name {
			return &models[i]
		}
	}
	return nil
}

// FilterRunningModels filters models to only return running ones
func FilterRunningModels(models []ModelDTO) []ModelDTO {
	var running []ModelDTO
	for _, model := range models {
		if model.IsRunning {
			running = append(running, model)
		}
	}
	return running
}

// SortModelsByName sorts models by name (ascending)
func SortModelsByName(models []ModelDTO) {
	for i := 0; i < len(models)-1; i++ {
		for j := i + 1; j < len(models); j++ {
			if models[i].Name > models[j].Name {
				models[i], models[j] = models[j], models[i]
			}
		}
	}
}

// SortModelsBySize sorts models by size (descending)
func SortModelsBySize(models []ModelDTO) {
	for i := 0; i < len(models)-1; i++ {
		for j := i + 1; j < len(models); j++ {
			if models[i].Size < models[j].Size {
				models[i], models[j] = models[j], models[i]
			}
		}
	}
}

// SortModelsByModified sorts models by modification date (most recent first)
func SortModelsByModified(models []ModelDTO) {
	for i := 0; i < len(models)-1; i++ {
		for j := i + 1; j < len(models); j++ {
			if models[i].Modified.Before(models[j].Modified) {
				models[i], models[j] = models[j], models[i]
			}
		}
	}
}

// Conversion functions for backward compatibility with existing GUI types

// ConvertToGuiModel converts a core.Model to GuiModel (existing type)
func ConvertToGuiModel(model core.Model) GuiModel {
	return GuiModel{
		ID:                model.ID,
		Name:              model.Name,
		Size:              model.Size,
		SizeFormatted:     formatSize(model.Size),
		Family:            model.Details.Family,
		ParameterSize:     model.Details.ParameterSize,
		QuantizationLevel: model.Details.QuantizationLevel,
		Modified:          model.ModifiedAt,
		ModifiedFormatted: model.ModifiedAt.Format("2006-01-02 15:04:05"),
		IsRunning:         model.Status == "running",
		Digest:            model.Digest,
		Status:            model.Status,
		Selected:          false, // Default to not selected
	}
}

// ConvertToGuiModels converts a slice of core.Model to []GuiModel
func ConvertToGuiModels(models []core.Model) []GuiModel {
	guiModels := make([]GuiModel, len(models))
	for i, model := range models {
		guiModels[i] = ConvertToGuiModel(model)
	}
	return guiModels
}

// ConvertToGuiRunningModel converts a core.RunningModel to GuiRunningModel (existing type)
func ConvertToGuiRunningModel(model core.RunningModel) GuiRunningModel {
	vramGB := float64(model.SizeVRAM) / (1024 * 1024 * 1024) // Convert bytes to GB
	return GuiRunningModel{
		Name:               model.Name,
		Size:               model.Size,
		LoadedAt:           model.LoadedAt,
		ExpiresAt:          model.ExpiresAt,
		VRAMUsage:          vramGB,
		VRAMFormatted:      fmt.Sprintf("%.1f GB", vramGB),
		LoadedAtFormatted:  model.LoadedAt.Format("15:04:05"),
		ExpiresAtFormatted: model.ExpiresAt.Format("15:04:05"),
	}
}

// ConvertToGuiRunningModels converts a slice of core.RunningModel to []GuiRunningModel
func ConvertToGuiRunningModels(models []core.RunningModel) []GuiRunningModel {
	guiModels := make([]GuiRunningModel, len(models))
	for i, model := range models {
		guiModels[i] = ConvertToGuiRunningModel(model)
	}
	return guiModels
}

// ConvertToSettingsData converts a config.Config to SettingsData (existing type)
func ConvertToSettingsData(cfg *config.Config) *SettingsData {
	return &SettingsData{
		OllamaAPIURL:    cfg.OllamaAPIURL,
		Theme:           cfg.Theme,
		AutoRefresh:     cfg.AutoRefresh,
		RefreshInterval: cfg.RefreshInterval,
		WindowWidth:     cfg.WindowWidth,
		WindowHeight:    cfg.WindowHeight,
		DefaultView:     cfg.DefaultView,
		ShowSystemTray:  cfg.ShowSystemTray,
	}
}

// ConvertFromSettingsData converts a SettingsData back to config.Config
func ConvertFromSettingsData(settings *SettingsData) *config.Config {
	return &config.Config{
		OllamaAPIURL:    settings.OllamaAPIURL,
		Theme:           settings.Theme,
		AutoRefresh:     settings.AutoRefresh,
		RefreshInterval: settings.RefreshInterval,
		WindowWidth:     settings.WindowWidth,
		WindowHeight:    settings.WindowHeight,
		DefaultView:     settings.DefaultView,
		ShowSystemTray:  settings.ShowSystemTray,
	}
}
