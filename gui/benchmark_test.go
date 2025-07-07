package main

import (
	"context"
	"testing"
	"time"

	"github.com/sammcj/gollama/config"
	"github.com/sammcj/gollama/core"
)

// BenchmarkConvertToModelDTO benchmarks DTO conversion performance
func BenchmarkConvertToModelDTO(b *testing.B) {
	model := core.Model{
		ID:         "test-model",
		Name:       "test-model",
		Size:       1024 * 1024 * 1024,
		Digest:     "sha256:abc123",
		ModifiedAt: time.Now(),
		Details: core.ModelDetails{
			Family:            "llama",
			ParameterSize:     "7B",
			QuantizationLevel: "Q4_0",
		},
		Status: "available",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ConvertToModelDTO(model)
	}
}

// BenchmarkConvertToModelDTOs benchmarks batch DTO conversion
func BenchmarkConvertToModelDTOs(b *testing.B) {
	models := make([]core.Model, 100)
	for i := range models {
		models[i] = core.Model{
			ID:         "test-model",
			Name:       "test-model",
			Size:       1024 * 1024 * 1024,
			Digest:     "sha256:abc123",
			ModifiedAt: time.Now(),
			Details: core.ModelDetails{
				Family:            "llama",
				ParameterSize:     "7B",
				QuantizationLevel: "Q4_0",
			},
			Status: "available",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ConvertToModelDTOs(models)
	}
}

// BenchmarkConvertToConfigDTO benchmarks config DTO conversion
func BenchmarkConvertToConfigDTO(b *testing.B) {
	cfg := &config.Config{
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ConvertToConfigDTO(cfg)
	}
}

// BenchmarkValidateModelName benchmarks model name validation
func BenchmarkValidateModelName(b *testing.B) {
	app := NewApp()
	testNames := []string{
		"valid-model-name",
		"another_valid_name",
		"model123",
		"namespace/model",
		"",
		"invalid name with spaces",
		"model:with:colons",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := testNames[i%len(testNames)]
		_, _ = app.ValidateModelName(name)
	}
}

// BenchmarkGetServiceStatus benchmarks service status generation
func BenchmarkGetServiceStatus(b *testing.B) {
	app := NewApp()
	ctx := context.Background()
	app.ctx = ctx

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = app.GetServiceStatus()
	}
}

// BenchmarkTestServiceBinding benchmarks service binding test
func BenchmarkTestServiceBinding(b *testing.B) {
	app := NewApp()
	ctx := context.Background()
	app.ctx = ctx

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = app.TestServiceBinding()
	}
}

// BenchmarkHealthCheckDTO benchmarks health check DTO operations
func BenchmarkHealthCheckDTO(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		health := NewHealthCheckDTO("healthy", "1.0.0")
		health.AddService("service1", "healthy", "OK", "10ms")
		health.AddService("service2", "healthy", "OK", "5ms")
		health.AddService("service3", "unhealthy", "Error", "")
		_ = health.IsHealthy()
	}
}

// BenchmarkVRAMConstraintsValidation benchmarks vRAM constraints validation
func BenchmarkVRAMConstraintsValidation(b *testing.B) {
	constraints := VRAMConstraintsDTO{
		AvailableVRAM:  16.0,
		Context:        2048,
		Quantization:   "Q4_0",
		BatchSize:      1,
		SequenceLength: 1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = constraints.Validate()
	}
}

// BenchmarkConfigDTOValidation benchmarks config DTO validation
func BenchmarkConfigDTOValidation(b *testing.B) {
	config := ConfigDTO{
		OllamaAPIURL:    "http://localhost:11434",
		LogLevel:        "info",
		AutoRefresh:     true,
		RefreshInterval: 30,
		WindowWidth:     1200,
		WindowHeight:    800,
		DefaultView:     "models",
		ShowSystemTray:  false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Validate()
	}
}

// BenchmarkModelUtilityFunctions benchmarks model utility functions
func BenchmarkModelUtilityFunctions(b *testing.B) {
	models := []ModelDTO{
		{ID: "1", Name: "model-a", Size: 1000, IsRunning: true},
		{ID: "2", Name: "model-b", Size: 2000, IsRunning: false},
		{ID: "3", Name: "model-c", Size: 500, IsRunning: true},
		{ID: "4", Name: "model-d", Size: 1500, IsRunning: false},
		{ID: "5", Name: "model-e", Size: 800, IsRunning: true},
	}

	b.Run("GetModelByID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = GetModelByID(models, "3")
		}
	})

	b.Run("GetModelByName", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = GetModelByName(models, "model-c")
		}
	})

	b.Run("FilterRunningModels", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = FilterRunningModels(models)
		}
	})

	b.Run("SortModelsByName", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			testModels := make([]ModelDTO, len(models))
			copy(testModels, models)
			SortModelsByName(testModels)
		}
	})

	b.Run("SortModelsBySize", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			testModels := make([]ModelDTO, len(models))
			copy(testModels, models)
			SortModelsBySize(testModels)
		}
	})
}

// BenchmarkFormatSize benchmarks size formatting function
func BenchmarkFormatSize(b *testing.B) {
	sizes := []int64{
		1024,
		1024 * 1024,
		1024 * 1024 * 1024,
		1024 * 1024 * 1024 * 1024,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		size := sizes[i%len(sizes)]
		_ = formatSize(size)
	}
}

// BenchmarkAppMethodCalls benchmarks app method calls without service
func BenchmarkAppMethodCalls(b *testing.B) {
	app := NewApp()
	ctx := context.Background()
	app.ctx = ctx
	// Note: Cannot assign mock service directly, testing methods that work without service

	b.Run("GetServiceStatus", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = app.GetServiceStatus()
		}
	})

	b.Run("ValidateModelName", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = app.ValidateModelName("test-model")
		}
	})

	b.Run("TestServiceBinding", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = app.TestServiceBinding()
		}
	})
}

// BenchmarkConcurrentServiceCalls benchmarks concurrent service calls
func BenchmarkConcurrentServiceCalls(b *testing.B) {
	app := NewApp()
	ctx := context.Background()
	app.ctx = ctx
	// Note: Cannot assign mock service directly, testing concurrent calls to GetServiceStatus

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = app.GetServiceStatus()
		}
	})
}
