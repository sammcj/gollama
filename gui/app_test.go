package main

import (
	"context"
	"testing"
	"time"

	"github.com/sammcj/gollama/core"
)

// TestAppServiceMethods tests that all service methods are properly implemented
func TestAppServiceMethods(t *testing.T) {
	// Create a mock app for testing
	app := NewApp()

	// Test that app is created properly
	if app == nil {
		t.Fatal("NewApp() returned nil")
	}

	// Test method signatures exist (compilation test)
	t.Run("GetModels signature", func(t *testing.T) {
		// This will fail to compile if signature is wrong
		var _ func() ([]ModelDTO, error) = app.GetModels
	})

	t.Run("GetModel signature", func(t *testing.T) {
		var _ func(string) (*ModelDTO, error) = app.GetModel
	})

	t.Run("GetRunningModels signature", func(t *testing.T) {
		var _ func() ([]RunningModelDTO, error) = app.GetRunningModels
	})

	t.Run("DeleteModel signature", func(t *testing.T) {
		var _ func(string) error = app.DeleteModel
	})

	t.Run("RunModel signature", func(t *testing.T) {
		var _ func(string) error = app.RunModel
	})

	t.Run("UnloadModel signature", func(t *testing.T) {
		var _ func(string) error = app.UnloadModel
	})

	t.Run("CopyModel signature", func(t *testing.T) {
		var _ func(string, string) error = app.CopyModel
	})

	t.Run("PushModel signature", func(t *testing.T) {
		var _ func(string) error = app.PushModel
	})

	t.Run("PullModel signature", func(t *testing.T) {
		var _ func(string) error = app.PullModel
	})

	t.Run("GetModelInfo signature", func(t *testing.T) {
		var _ func(string) (*ModelInfoDTO, error) = app.GetModelInfo
	})

	t.Run("GetConfig signature", func(t *testing.T) {
		var _ func() (*ConfigDTO, error) = app.GetConfig
	})

	t.Run("UpdateConfig signature", func(t *testing.T) {
		var _ func(ConfigDTO) error = app.UpdateConfig
	})

	t.Run("EstimateVRAMForModel signature", func(t *testing.T) {
		var _ func(string, VRAMConstraintsDTO) (*VRAMEstimationDTO, error) = app.EstimateVRAMForModel
	})

	t.Run("HealthCheck signature", func(t *testing.T) {
		var _ func() error = app.HealthCheck
	})

	t.Run("GetServiceStatus signature", func(t *testing.T) {
		var _ func() (*HealthCheckDTO, error) = app.GetServiceStatus
	})

	t.Run("SearchModels signature", func(t *testing.T) {
		var _ func(string, map[string]interface{}) ([]ModelDTO, error) = app.SearchModels
	})

	t.Run("GetModelOperations signature", func(t *testing.T) {
		var _ func(string) ([]string, error) = app.GetModelOperations
	})

	t.Run("ValidateModelName signature", func(t *testing.T) {
		var _ func(string) (bool, error) = app.ValidateModelName
	})

	t.Run("GetSystemInfo signature", func(t *testing.T) {
		var _ func() (map[string]interface{}, error) = app.GetSystemInfo
	})

	t.Run("TestServiceBinding signature", func(t *testing.T) {
		var _ func() string = app.TestServiceBinding
	})
}

// TestAppWithoutService tests app behavior when service is not initialized
func TestAppWithoutService(t *testing.T) {
	app := NewApp()

	t.Run("GetModels without service", func(t *testing.T) {
		models, err := app.GetModels()
		if err == nil {
			t.Error("Expected error when service not initialized")
		}
		if models != nil {
			t.Error("Expected nil models when service not initialized")
		}
	})

	t.Run("GetConfig without service", func(t *testing.T) {
		config, err := app.GetConfig()
		if err == nil {
			t.Error("Expected error when service not initialized")
		}
		if config != nil {
			t.Error("Expected nil config when service not initialized")
		}
	})

	t.Run("HealthCheck without service", func(t *testing.T) {
		err := app.HealthCheck()
		if err == nil {
			t.Error("Expected error when service not initialized")
		}
	})

	t.Run("GetServiceStatus without service", func(t *testing.T) {
		status, err := app.GetServiceStatus()
		if err != nil {
			t.Error("GetServiceStatus should not return error even without service")
		}
		if status == nil {
			t.Error("Expected status even without service")
		}
		if status.Status != "unhealthy" {
			t.Error("Expected unhealthy status when service not initialized")
		}
	})

	t.Run("ValidateModelName", func(t *testing.T) {
		// This method doesn't require service
		valid, err := app.ValidateModelName("test-model")
		if err != nil {
			t.Errorf("ValidateModelName failed: %v", err)
		}
		if !valid {
			t.Error("Expected valid model name")
		}

		// Test invalid name
		valid, err = app.ValidateModelName("")
		if err == nil {
			t.Error("Expected error for empty model name")
		}
		if valid {
			t.Error("Expected invalid for empty model name")
		}
	})

	t.Run("TestServiceBinding without service", func(t *testing.T) {
		result := app.TestServiceBinding()
		if result == "" {
			t.Error("Expected non-empty result from TestServiceBinding")
		}
		// Should contain error message about service not initialized
		if !contains(result, "Service not initialized") && !contains(result, "ERROR") {
			t.Error("Expected error message in TestServiceBinding result")
		}
	})
}

// TestAppStartup tests the app startup process
func TestAppStartup(t *testing.T) {
	app := NewApp()
	ctx := context.Background()

	// Test startup - this will fail because we don't have Ollama running
	// but it should not panic and should handle the error gracefully
	app.OnStartup(ctx)

	// Verify context is set
	if app.ctx == nil {
		t.Error("Expected context to be set after OnStartup")
	}

	// Test shutdown
	app.OnShutdown(ctx)
}

// TestDTOConversions tests that DTO conversions work properly
func TestDTOConversions(t *testing.T) {
	// Test model conversion
	coreModel := core.Model{
		Name:       "test-model",
		ID:         "test-id",
		Size:       1024 * 1024 * 1024, // 1GB
		Digest:     "sha256:abc123",
		ModifiedAt: time.Now(),
		Details: core.ModelDetails{
			Family:            "llama",
			ParameterSize:     "7B",
			QuantizationLevel: "Q4_0",
		},
		Status: "available",
	}

	dto := ConvertToModelDTO(coreModel)

	if dto.Name != coreModel.Name {
		t.Errorf("Expected name %s, got %s", coreModel.Name, dto.Name)
	}
	if dto.Size != coreModel.Size {
		t.Errorf("Expected size %d, got %d", coreModel.Size, dto.Size)
	}
	if dto.SizeFormatted == "" {
		t.Error("Expected formatted size to be set")
	}
	if dto.Family != coreModel.Details.Family {
		t.Errorf("Expected family %s, got %s", coreModel.Details.Family, dto.Family)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsAt(s, substr))))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
