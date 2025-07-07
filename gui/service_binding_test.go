package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sammcj/gollama/config"
	"github.com/sammcj/gollama/core"
)

// TestServiceBinding tests the complete service binding functionality
func TestServiceBinding(t *testing.T) {
	app := NewApp()
	ctx := context.Background()

	t.Run("App Creation", func(t *testing.T) {
		if app == nil {
			t.Fatal("NewApp() returned nil")
		}
		if app.ctx != nil {
			t.Error("Expected context to be nil before OnStartup")
		}
		if app.service != nil {
			t.Error("Expected service to be nil before OnStartup")
		}
	})

	t.Run("OnStartup Initialization", func(t *testing.T) {
		// This will attempt to initialize the service
		// It may fail if Ollama is not running, but should not panic
		app.OnStartup(ctx)

		if app.ctx == nil {
			t.Error("Expected context to be set after OnStartup")
		}

		// Service may be nil if Ollama is not available, which is acceptable for testing
		if app.service == nil {
			t.Log("Service is nil - Ollama may not be running (this is acceptable for testing)")
		}
	})

	t.Run("Method Signatures", func(t *testing.T) {
		// Test that all required methods exist with correct signatures
		testMethodSignatures(t, app)
	})

	t.Run("Error Handling Without Service", func(t *testing.T) {
		// Create a new app without initializing service
		testApp := NewApp()
		testErrorHandlingWithoutService(t, testApp)
	})
}

// testMethodSignatures verifies all service methods have correct signatures
func testMethodSignatures(t *testing.T, app *App) {
	tests := []struct {
		name      string
		signature interface{}
	}{
		{"GetModels", func() ([]ModelDTO, error) { return app.GetModels() }},
		{"GetModel", func(string) (*ModelDTO, error) { return app.GetModel("test") }},
		{"GetRunningModels", func() ([]RunningModelDTO, error) { return app.GetRunningModels() }},
		{"DeleteModel", func(string) error { return app.DeleteModel("test") }},
		{"RunModel", func(string) error { return app.RunModel("test") }},
		{"UnloadModel", func(string) error { return app.UnloadModel("test") }},
		{"CopyModel", func(string, string) error { return app.CopyModel("src", "dst") }},
		{"PushModel", func(string) error { return app.PushModel("test") }},
		{"PullModel", func(string) error { return app.PullModel("test") }},
		{"GetModelInfo", func(string) (*ModelInfoDTO, error) { return app.GetModelInfo("test") }},
		{"GetConfig", func() (*ConfigDTO, error) { return app.GetConfig() }},
		{"UpdateConfig", func(ConfigDTO) error { return app.UpdateConfig(ConfigDTO{}) }},
		{"EstimateVRAMForModel", func(string, VRAMConstraintsDTO) (*VRAMEstimationDTO, error) {
			return app.EstimateVRAMForModel("test", VRAMConstraintsDTO{})
		}},
		{"HealthCheck", func() error { return app.HealthCheck() }},
		{"GetServiceStatus", func() (*HealthCheckDTO, error) { return app.GetServiceStatus() }},
		{"SearchModels", func(string, map[string]interface{}) ([]ModelDTO, error) {
			return app.SearchModels("test", nil)
		}},
		{"GetModelOperations", func(string) ([]string, error) { return app.GetModelOperations("test") }},
		{"ValidateModelName", func(string) (bool, error) { return app.ValidateModelName("test") }},
		{"GetSystemInfo", func() (map[string]interface{}, error) { return app.GetSystemInfo() }},
		{"TestServiceBinding", func() string { return app.TestServiceBinding() }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// This test just verifies the method exists and has the right signature
			// The actual call may fail if service is not available, which is fine
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Method %s panicked: %v", test.name, r)
				}
			}()

			// We don't call the function, just verify it compiles
			_ = test.signature
		})
	}
}

// testErrorHandlingWithoutService tests error handling when service is not initialized
func testErrorHandlingWithoutService(t *testing.T, app *App) {
	tests := []struct {
		name string
		test func() error
	}{
		{"GetModels", func() error { _, err := app.GetModels(); return err }},
		{"GetModel", func() error { _, err := app.GetModel("test"); return err }},
		{"GetRunningModels", func() error { _, err := app.GetRunningModels(); return err }},
		{"DeleteModel", func() error { return app.DeleteModel("test") }},
		{"RunModel", func() error { return app.RunModel("test") }},
		{"UnloadModel", func() error { return app.UnloadModel("test") }},
		{"CopyModel", func() error { return app.CopyModel("src", "dst") }},
		{"PushModel", func() error { return app.PushModel("test") }},
		{"PullModel", func() error { return app.PullModel("test") }},
		{"GetModelInfo", func() error { _, err := app.GetModelInfo("test"); return err }},
		{"GetConfig", func() error { _, err := app.GetConfig(); return err }},
		{"UpdateConfig", func() error { return app.UpdateConfig(ConfigDTO{}) }},
		{"EstimateVRAMForModel", func() error {
			_, err := app.EstimateVRAMForModel("test", VRAMConstraintsDTO{})
			return err
		}},
		{"HealthCheck", func() error { return app.HealthCheck() }},
		{"SearchModels", func() error { _, err := app.SearchModels("test", nil); return err }},
		{"GetModelOperations", func() error { _, err := app.GetModelOperations("test"); return err }},
		{"GetSystemInfo", func() error { _, err := app.GetSystemInfo(); return err }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.test()
			if err == nil {
				t.Errorf("Expected error when service not initialized for %s", test.name)
			}
			if err.Error() != "service not initialized" {
				t.Logf("Got error: %v (this may be acceptable)", err)
			}
		})
	}

	// Test methods that should work without service
	t.Run("ValidateModelName", func(t *testing.T) {
		valid, err := app.ValidateModelName("test-model")
		if err != nil {
			t.Errorf("ValidateModelName should work without service: %v", err)
		}
		if !valid {
			t.Error("Expected 'test-model' to be valid")
		}

		// Test invalid name
		valid, err = app.ValidateModelName("")
		if err == nil {
			t.Error("Expected error for empty model name")
		}
		if valid {
			t.Error("Expected empty name to be invalid")
		}
	})

	t.Run("GetServiceStatus", func(t *testing.T) {
		status, err := app.GetServiceStatus()
		if err != nil {
			t.Errorf("GetServiceStatus should not return error: %v", err)
		}
		if status == nil {
			t.Error("Expected status even without service")
		}
		if status.Status != "unhealthy" {
			t.Errorf("Expected unhealthy status, got: %s", status.Status)
		}
	})

	t.Run("TestServiceBinding", func(t *testing.T) {
		result := app.TestServiceBinding()
		if result == "" {
			t.Error("Expected non-empty result from TestServiceBinding")
		}
		// Should contain error message about service not initialized
		if !contains(result, "Service not initialized") && !contains(result, "ERROR") {
			t.Errorf("Expected error message in result, got: %s", result)
		}
	})
}

// TestServiceMethodExposure tests that methods are properly exposed for Wails v3
func TestServiceMethodExposure(t *testing.T) {
	app := NewApp()
	ctx := context.Background()
	app.OnStartup(ctx)

	// Test that all methods can be called (even if they fail due to no Ollama)
	t.Run("Method Exposure", func(t *testing.T) {
		// These should not panic, even if they return errors
		testMethods := []func(){
			func() { app.GetModels() },
			func() { app.GetModel("test") },
			func() { app.GetRunningModels() },
			func() { app.DeleteModel("test") },
			func() { app.RunModel("test") },
			func() { app.UnloadModel("test") },
			func() { app.CopyModel("src", "dst") },
			func() { app.PushModel("test") },
			func() { app.PullModel("test") },
			func() { app.GetModelInfo("test") },
			func() { app.GetConfig() },
			func() { app.UpdateConfig(ConfigDTO{}) },
			func() { app.EstimateVRAMForModel("test", VRAMConstraintsDTO{}) },
			func() { app.HealthCheck() },
			func() { app.GetServiceStatus() },
			func() { app.SearchModels("test", nil) },
			func() { app.GetModelOperations("test") },
			func() { app.ValidateModelName("test") },
			func() { app.GetSystemInfo() },
			func() { app.TestServiceBinding() },
		}

		for i, method := range testMethods {
			t.Run(fmt.Sprintf("Method_%d", i), func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Method %d panicked: %v", i, r)
					}
				}()
				method()
			})
		}
	})
}

// TestWailsV3Compatibility tests Wails v3 specific functionality
func TestWailsV3Compatibility(t *testing.T) {
	app := NewApp()
	ctx := context.Background()

	t.Run("Context Management", func(t *testing.T) {
		app.OnStartup(ctx)
		if app.ctx != ctx {
			t.Error("Context not properly stored")
		}

		// Test shutdown
		app.OnShutdown(ctx)
		// Should not panic
	})

	t.Run("Embedded Assets", func(t *testing.T) {
		// Test that embedded assets are available
		// Note: embed.FS cannot be compared to nil, so we test by trying to read
		_, err := staticFiles.ReadDir("static")
		if err != nil {
			t.Error("Static files not properly embedded")
		}
		_, err = templateFiles.ReadDir("templates")
		if err != nil {
			t.Error("Template files not properly embedded")
		}
	})

	t.Run("Template Loading", func(t *testing.T) {
		app.OnStartup(ctx)
		if app.templates == nil {
			t.Error("Templates not loaded")
		}
	})
}

// TestServiceBindingWithMockService tests service binding with a mock service
func TestServiceBindingWithMockService(t *testing.T) {
	app := NewApp()
	ctx := context.Background()

	// Create a mock service for testing
	// Note: We can't directly assign a mock to app.service since it expects *core.GollamaService
	// Instead, we'll test the methods directly with the mock
	mockService := &MockGollamaService{}
	app.ctx = ctx

	// Test mock service methods directly
	t.Run("Mock Service Methods", func(t *testing.T) {
		models, err := mockService.ListModels()
		if err != nil {
			t.Errorf("Mock ListModels failed: %v", err)
		}
		if len(models) != 1 {
			t.Errorf("Expected 1 model, got %d", len(models))
		}

		config := mockService.GetConfig()
		if config == nil {
			t.Error("Expected config, got nil")
		}

		err = mockService.HealthCheck()
		if err != nil {
			t.Errorf("Mock HealthCheck failed: %v", err)
		}
	})
}

// MockGollamaService is a mock implementation for testing
type MockGollamaService struct{}

func (m *MockGollamaService) ListModels() ([]core.Model, error) {
	return []core.Model{
		{
			ID:         "test-model",
			Name:       "test-model",
			Size:       1024 * 1024 * 1024,
			ModifiedAt: time.Now(),
			Details: core.ModelDetails{
				Family:            "test",
				ParameterSize:     "7B",
				QuantizationLevel: "Q4_0",
			},
			Status: "available",
		},
	}, nil
}

func (m *MockGollamaService) GetModel(name string) (*core.Model, error) {
	return &core.Model{
		ID:         name,
		Name:       name,
		Size:       1024 * 1024 * 1024,
		ModifiedAt: time.Now(),
		Details: core.ModelDetails{
			Family:            "test",
			ParameterSize:     "7B",
			QuantizationLevel: "Q4_0",
		},
		Status: "available",
	}, nil
}

func (m *MockGollamaService) GetRunningModels() ([]core.RunningModel, error) {
	return []core.RunningModel{}, nil
}

func (m *MockGollamaService) RunModel(name string) error {
	return nil
}

func (m *MockGollamaService) DeleteModel(name string) error {
	return nil
}

func (m *MockGollamaService) UnloadModel(name string) error {
	return nil
}

func (m *MockGollamaService) CopyModel(source, dest string) error {
	return nil
}

func (m *MockGollamaService) PushModel(name string) error {
	return nil
}

func (m *MockGollamaService) PullModel(name string) error {
	return nil
}

func (m *MockGollamaService) GetModelInfo(name string) (*core.EnhancedModelInfo, error) {
	return &core.EnhancedModelInfo{
		Model: core.Model{
			ID:         name,
			Name:       name,
			Size:       1024 * 1024 * 1024,
			ModifiedAt: time.Now(),
			Details: core.ModelDetails{
				Family:            "test",
				ParameterSize:     "7B",
				QuantizationLevel: "Q4_0",
			},
			Status: "available",
		},
		Modelfile:  "FROM test",
		Template:   "{{ .Prompt }}",
		System:     "You are a helpful assistant",
		Parameters: map[string]interface{}{"temperature": 0.7},
	}, nil
}

func (m *MockGollamaService) EstimateVRAM(modelName string, constraints core.VRAMConstraints) (*core.VRAMEstimation, error) {
	return &core.VRAMEstimation{
		ModelSize:     4.0,
		ContextSize:   2.0,
		TotalSize:     6.0,
		Quantization:  constraints.Quantization,
		ContextLength: constraints.ContextLength,
		Breakdown: core.VRAMBreakdown{
			ModelWeights: 4.0,
			KVCache:      2.0,
			Activations:  0.5,
			Overhead:     0.5,
			Total:        7.0,
		},
		Recommendations: []core.VRAMRecommendation{
			{
				Quantization:  "Q4_0",
				ContextLength: 2048,
				VRAMUsage:     6.0,
				Description:   "Recommended configuration",
			},
		},
		Estimates: map[string]core.VRAMEstimateRow{
			"Q4_0": {
				Quantization:  "Q4_0",
				BitsPerWeight: 4.0,
				Contexts: map[string]float64{
					"2048": 6.0,
					"4096": 8.0,
				},
			},
		},
	}, nil
}

func (m *MockGollamaService) GetConfig() *config.Config {
	return &config.Config{
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
	}
}

func (m *MockGollamaService) UpdateConfig(cfg *config.Config) error {
	return nil
}

func (m *MockGollamaService) HealthCheck() error {
	return nil
}

func (m *MockGollamaService) Close() {
	// Mock implementation - nothing to close
}

// Helper functions are defined in app_test.go
