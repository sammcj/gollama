package main

import (
	"testing"
	"time"

	"github.com/sammcj/gollama/config"
	"github.com/sammcj/gollama/core"
)

func TestConvertToModelDTO(t *testing.T) {
	// Create a test core.Model
	testModel := core.Model{
		ID:         "test-id",
		Name:       "test-model",
		Size:       1024 * 1024 * 1024, // 1GB
		Digest:     "test-digest",
		ModifiedAt: time.Now(),
		Details: core.ModelDetails{
			Family:            "llama",
			ParameterSize:     "7B",
			QuantizationLevel: "Q4_0",
		},
		Status: "running",
	}

	// Convert to DTO
	dto := ConvertToModelDTO(testModel)

	// Verify conversion
	if dto.ID != testModel.ID {
		t.Errorf("Expected ID %s, got %s", testModel.ID, dto.ID)
	}
	if dto.Name != testModel.Name {
		t.Errorf("Expected Name %s, got %s", testModel.Name, dto.Name)
	}
	if dto.Size != testModel.Size {
		t.Errorf("Expected Size %d, got %d", testModel.Size, dto.Size)
	}
	if dto.Family != testModel.Details.Family {
		t.Errorf("Expected Family %s, got %s", testModel.Details.Family, dto.Family)
	}
	if dto.ParameterSize != testModel.Details.ParameterSize {
		t.Errorf("Expected ParameterSize %s, got %s", testModel.Details.ParameterSize, dto.ParameterSize)
	}
	if dto.QuantizationLevel != testModel.Details.QuantizationLevel {
		t.Errorf("Expected QuantizationLevel %s, got %s", testModel.Details.QuantizationLevel, dto.QuantizationLevel)
	}
	if !dto.IsRunning {
		t.Error("Expected IsRunning to be true")
	}
	if dto.SizeFormatted == "" {
		t.Error("Expected SizeFormatted to be populated")
	}
	if dto.ModifiedFormatted == "" {
		t.Error("Expected ModifiedFormatted to be populated")
	}
}

func TestConvertToConfigDTO(t *testing.T) {
	// Create a test config.Config
	testConfig := &config.Config{
		OllamaAPIURL:    "http://localhost:11434",
		LogLevel:        "info",
		AutoRefresh:     true,
		RefreshInterval: 30,
		WindowWidth:     1200,
		WindowHeight:    800,
		DefaultView:     "models",
		ShowSystemTray:  false,
		Theme:           "dark",
		Editor:          "vim",
		SortOrder:       "name",
		StripString:     "",
		DockerContainer: "",
	}

	// Convert to DTO
	dto := ConvertToConfigDTO(testConfig)

	// Verify conversion
	if dto.OllamaAPIURL != testConfig.OllamaAPIURL {
		t.Errorf("Expected OllamaAPIURL %s, got %s", testConfig.OllamaAPIURL, dto.OllamaAPIURL)
	}
	if dto.LogLevel != testConfig.LogLevel {
		t.Errorf("Expected LogLevel %s, got %s", testConfig.LogLevel, dto.LogLevel)
	}
	if dto.AutoRefresh != testConfig.AutoRefresh {
		t.Errorf("Expected AutoRefresh %t, got %t", testConfig.AutoRefresh, dto.AutoRefresh)
	}
	if dto.RefreshInterval != testConfig.RefreshInterval {
		t.Errorf("Expected RefreshInterval %d, got %d", testConfig.RefreshInterval, dto.RefreshInterval)
	}
	if dto.WindowWidth != testConfig.WindowWidth {
		t.Errorf("Expected WindowWidth %d, got %d", testConfig.WindowWidth, dto.WindowWidth)
	}
	if dto.WindowHeight != testConfig.WindowHeight {
		t.Errorf("Expected WindowHeight %d, got %d", testConfig.WindowHeight, dto.WindowHeight)
	}
}

func TestConfigDTOValidation(t *testing.T) {
	// Test valid config
	validConfig := &ConfigDTO{
		OllamaAPIURL:    "http://localhost:11434",
		LogLevel:        "info",
		RefreshInterval: 30,
		WindowWidth:     800,
		WindowHeight:    600,
		DefaultView:     "models",
	}

	errors := validConfig.Validate()
	if len(errors) != 0 {
		t.Errorf("Expected no validation errors for valid config, got: %v", errors)
	}

	// Test invalid config
	invalidConfig := &ConfigDTO{
		OllamaAPIURL:    "",        // Empty URL
		LogLevel:        "invalid", // Invalid log level
		RefreshInterval: 0,         // Invalid refresh interval
		WindowWidth:     100,       // Too small
		WindowHeight:    100,       // Too small
		DefaultView:     "invalid", // Invalid view
	}

	errors = invalidConfig.Validate()
	if len(errors) == 0 {
		t.Error("Expected validation errors for invalid config")
	}

	// Check specific error messages
	expectedErrors := []string{
		"Ollama API URL is required",
		"Refresh interval must be at least 1 second",
		"Window width must be at least 400 pixels",
		"Window height must be at least 300 pixels",
		"Default view must be one of: models, running, settings, vram",
		"Log level must be one of: debug, info, warn, error",
	}

	if len(errors) != len(expectedErrors) {
		t.Errorf("Expected %d validation errors, got %d: %v", len(expectedErrors), len(errors), errors)
	}
}

func TestVRAMConstraintsDTOValidation(t *testing.T) {
	// Test valid constraints
	validConstraints := &VRAMConstraintsDTO{
		AvailableVRAM:  16.0,
		Context:        2048,
		Quantization:   "Q4_0",
		BatchSize:      1,
		SequenceLength: 1,
	}

	errors := validConstraints.Validate()
	if len(errors) != 0 {
		t.Errorf("Expected no validation errors for valid constraints, got: %v", errors)
	}

	// Test invalid constraints
	invalidConstraints := &VRAMConstraintsDTO{
		AvailableVRAM:  0,   // Invalid vRAM
		Context:        100, // Too small
		BatchSize:      0,   // Invalid batch size
		SequenceLength: 0,   // Invalid sequence length
	}

	errors = invalidConstraints.Validate()
	if len(errors) == 0 {
		t.Error("Expected validation errors for invalid constraints")
	}
}

func TestUtilityFunctions(t *testing.T) {
	// Create test models
	models := []ModelDTO{
		{ID: "1", Name: "model-a", Size: 1000, IsRunning: true},
		{ID: "2", Name: "model-b", Size: 2000, IsRunning: false},
		{ID: "3", Name: "model-c", Size: 500, IsRunning: true},
	}

	// Test GetModelByID
	model := GetModelByID(models, "2")
	if model == nil || model.Name != "model-b" {
		t.Error("GetModelByID failed to find correct model")
	}

	// Test GetModelByName
	model = GetModelByName(models, "model-c")
	if model == nil || model.ID != "3" {
		t.Error("GetModelByName failed to find correct model")
	}

	// Test FilterRunningModels
	running := FilterRunningModels(models)
	if len(running) != 2 {
		t.Errorf("Expected 2 running models, got %d", len(running))
	}

	// Test sorting functions
	testModels := make([]ModelDTO, len(models))
	copy(testModels, models)

	SortModelsByName(testModels)
	if testModels[0].Name != "model-a" {
		t.Error("SortModelsByName failed")
	}

	copy(testModels, models)
	SortModelsBySize(testModels)
	if testModels[0].Size != 2000 {
		t.Error("SortModelsBySize failed")
	}
}

func TestHealthCheckDTO(t *testing.T) {
	health := NewHealthCheckDTO("healthy", "1.0.0")

	if health.Status != "healthy" {
		t.Error("Expected status to be healthy")
	}

	if health.Version != "1.0.0" {
		t.Error("Expected version to be 1.0.0")
	}

	// Add services
	health.AddService("ollama", "healthy", "Connected", "10ms")
	health.AddService("database", "unhealthy", "Connection failed", "")

	if len(health.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(health.Services))
	}

	// Test IsHealthy
	if health.IsHealthy() {
		t.Error("Expected health check to be unhealthy due to database service")
	}

	// Fix database service
	health.AddService("database", "healthy", "Connected", "5ms")
	if !health.IsHealthy() {
		t.Error("Expected health check to be healthy")
	}
}
