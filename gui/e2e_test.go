package main

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestE2EModelManagementWorkflow tests the complete end-to-end model management workflow
func TestE2EModelManagementWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	app := NewApp()
	ctx := context.Background()

	// Initialize the app
	app.OnStartup(ctx)

	// Skip if service is not available (Ollama not running)
	if app.service == nil {
		t.Skip("Skipping E2E tests - Ollama service not available")
	}

	t.Run("Complete Model Management Workflow", func(t *testing.T) {
		// Test 1: Health Check
		t.Run("Health Check", func(t *testing.T) {
			err := app.HealthCheck()
			if err != nil {
				t.Fatalf("Health check failed: %v", err)
			}
		})

		// Test 2: Get Service Status
		t.Run("Service Status", func(t *testing.T) {
			status, err := app.GetServiceStatus()
			if err != nil {
				t.Fatalf("Failed to get service status: %v", err)
			}
			if status == nil {
				t.Fatal("Service status is nil")
			}
			if status.Status != "healthy" && status.Status != "unhealthy" {
				t.Errorf("Invalid service status: %s", status.Status)
			}
		})

		// Test 3: Get Configuration
		t.Run("Configuration Management", func(t *testing.T) {
			config, err := app.GetConfig()
			if err != nil {
				t.Fatalf("Failed to get config: %v", err)
			}
			if config == nil {
				t.Fatal("Config is nil")
			}
			if config.OllamaAPIURL == "" {
				t.Error("Ollama API URL is empty")
			}

			// Test configuration update
			originalRefreshInterval := config.RefreshInterval
			config.RefreshInterval = 60

			err = app.UpdateConfig(*config)
			if err != nil {
				t.Errorf("Failed to update config: %v", err)
			}

			// Restore original config
			config.RefreshInterval = originalRefreshInterval
			err = app.UpdateConfig(*config)
			if err != nil {
				t.Errorf("Failed to restore config: %v", err)
			}
		})

		// Test 4: List Models
		t.Run("List Models", func(t *testing.T) {
			models, err := app.GetModels()
			if err != nil {
				t.Fatalf("Failed to get models: %v", err)
			}
			if models == nil {
				t.Fatal("Models list is nil")
			}

			t.Logf("Found %d models", len(models))

			// Validate model structure
			for i, model := range models {
				if model.Name == "" {
					t.Errorf("Model %d has empty name", i)
				}
				if model.Size <= 0 {
					t.Errorf("Model %d has invalid size: %d", i, model.Size)
				}
				if model.SizeFormatted == "" {
					t.Errorf("Model %d has empty formatted size", i)
				}
			}
		})

		// Test 5: Get Running Models
		t.Run("Running Models", func(t *testing.T) {
			runningModels, err := app.GetRunningModels()
			if err != nil {
				t.Fatalf("Failed to get running models: %v", err)
			}
			if runningModels == nil {
				t.Fatal("Running models list is nil")
			}

			t.Logf("Found %d running models", len(runningModels))

			// Validate running model structure
			for i, model := range runningModels {
				if model.Name == "" {
					t.Errorf("Running model %d has empty name", i)
				}
				if model.Size <= 0 {
					t.Errorf("Running model %d has invalid size: %d", i, model.Size)
				}
			}
		})

		// Test 6: Model Operations (if models are available)
		models, err := app.GetModels()
		if err == nil && len(models) > 0 {
			testModel := models[0]

			t.Run("Model Details", func(t *testing.T) {
				details, err := app.GetModelInfo(testModel.Name)
				if err != nil {
					t.Errorf("Failed to get model details for %s: %v", testModel.Name, err)
				} else {
					if details == nil {
						t.Error("Model details are nil")
					} else {
						if details.Model.Name != testModel.Name {
							t.Errorf("Model name mismatch: expected %s, got %s", testModel.Name, details.Model.Name)
						}
						if details.Modelfile == "" {
							t.Error("Modelfile is empty")
						}
					}
				}
			})

			t.Run("Model Operations", func(t *testing.T) {
				operations, err := app.GetModelOperations(testModel.Name)
				if err != nil {
					t.Errorf("Failed to get model operations for %s: %v", testModel.Name, err)
				} else {
					if len(operations) == 0 {
						t.Error("No operations available for model")
					}
					t.Logf("Available operations for %s: %v", testModel.Name, operations)
				}
			})

			t.Run("vRAM Estimation", func(t *testing.T) {
				constraints := VRAMConstraintsDTO{
					AvailableVRAM:  16.0,
					Context:        2048,
					Quantization:   "Q4_0",
					BatchSize:      1,
					SequenceLength: 1,
				}

				estimation, err := app.EstimateVRAMForModel(testModel.Name, constraints)
				if err != nil {
					t.Errorf("Failed to estimate vRAM for %s: %v", testModel.Name, err)
				} else {
					if estimation == nil {
						t.Error("vRAM estimation is nil")
					} else {
						if estimation.ModelName != testModel.Name {
							t.Errorf("Model name mismatch in estimation: expected %s, got %s", testModel.Name, estimation.ModelName)
						}
						if estimation.RequiredVRAM <= 0 {
							t.Error("Required vRAM should be positive")
						}
						t.Logf("vRAM estimation for %s: %.2f GB required, can run: %v",
							testModel.Name, estimation.RequiredVRAM, estimation.CanRun)
					}
				}
			})

			// Test model state management (only if model is not running)
			if !testModel.IsRunning {
				t.Run("Model State Management", func(t *testing.T) {
					// Test running a model (this may take time)
					t.Run("Run Model", func(t *testing.T) {
						err := app.RunModel(testModel.Name)
						if err != nil {
							t.Logf("Failed to run model %s (this may be expected): %v", testModel.Name, err)
						} else {
							t.Logf("Successfully started model: %s", testModel.Name)

							// Wait a bit for model to load
							time.Sleep(2 * time.Second)

							// Check if model is now running
							runningModels, err := app.GetRunningModels()
							if err == nil {
								found := false
								for _, rm := range runningModels {
									if rm.Name == testModel.Name {
										found = true
										break
									}
								}
								if found {
									t.Logf("Model %s is now running", testModel.Name)

									// Test unloading the model
									t.Run("Unload Model", func(t *testing.T) {
										err := app.UnloadModel(testModel.Name)
										if err != nil {
											t.Errorf("Failed to unload model %s: %v", testModel.Name, err)
										} else {
											t.Logf("Successfully unloaded model: %s", testModel.Name)
										}
									})
								}
							}
						}
					})
				})
			}
		} else {
			t.Log("No models available for testing model operations")
		}

		// Test 7: Search Models
		t.Run("Search Models", func(t *testing.T) {
			// Test empty search (should return all models)
			allModels, err := app.SearchModels("", nil)
			if err != nil {
				t.Errorf("Failed to search models with empty query: %v", err)
			} else {
				t.Logf("Empty search returned %d models", len(allModels))
			}

			// Test search with query
			searchResults, err := app.SearchModels("test", nil)
			if err != nil {
				t.Errorf("Failed to search models with query: %v", err)
			} else {
				t.Logf("Search for 'test' returned %d models", len(searchResults))
			}
		})

		// Test 8: System Information
		t.Run("System Information", func(t *testing.T) {
			sysInfo, err := app.GetSystemInfo()
			if err != nil {
				t.Errorf("Failed to get system info: %v", err)
			} else {
				if sysInfo == nil {
					t.Error("System info is nil")
				} else {
					if sysInfo["ollama_url"] == nil {
						t.Error("System info missing ollama_url")
					}
					if sysInfo["version"] == nil {
						t.Error("System info missing version")
					}
					t.Logf("System info: %+v", sysInfo)
				}
			}
		})

		// Test 9: Validation Functions
		t.Run("Validation Functions", func(t *testing.T) {
			// Test valid model names
			validNames := []string{"test-model", "model_name", "model123", "namespace/model"}
			for _, name := range validNames {
				valid, err := app.ValidateModelName(name)
				if err != nil {
					t.Errorf("Validation failed for valid name '%s': %v", name, err)
				}
				if !valid {
					t.Errorf("Valid name '%s' was marked as invalid", name)
				}
			}

			// Test invalid model names
			invalidNames := []string{"", "model with spaces", "model/with\\backslash", "model:with:colons"}
			for _, name := range invalidNames {
				valid, err := app.ValidateModelName(name)
				if name == "" && err == nil {
					t.Error("Empty name should return error")
				}
				if valid && name != "" {
					t.Errorf("Invalid name '%s' was marked as valid", name)
				}
			}
		})

		// Test 10: Service Binding Test
		t.Run("Service Binding Test", func(t *testing.T) {
			result := app.TestServiceBinding()
			if result == "" {
				t.Error("Service binding test returned empty result")
			}
			t.Logf("Service binding test result: %s", result)

			// Check if result indicates success or failure
			if contains(result, "SUCCESS") {
				t.Log("Service binding test indicates success")
			} else if contains(result, "ERROR") || contains(result, "WARNING") {
				t.Logf("Service binding test indicates issues: %s", result)
			}
		})
	})

	// Cleanup
	app.OnShutdown(ctx)
}

// TestE2EErrorScenarios tests error handling in various scenarios
func TestE2EErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E error tests in short mode")
	}

	app := NewApp()
	ctx := context.Background()
	app.OnStartup(ctx)

	t.Run("Error Scenarios", func(t *testing.T) {
		// Test operations with invalid model names
		t.Run("Invalid Model Operations", func(t *testing.T) {
			invalidNames := []string{"", "nonexistent-model", "invalid/model/name"}

			for _, name := range invalidNames {
				t.Run(fmt.Sprintf("GetModel_%s", name), func(t *testing.T) {
					_, err := app.GetModel(name)
					if err == nil && name != "" {
						t.Errorf("Expected error for invalid model name: %s", name)
					}
				})

				if name != "" { // Skip empty name for operations that validate input
					t.Run(fmt.Sprintf("RunModel_%s", name), func(t *testing.T) {
						err := app.RunModel(name)
						if err == nil {
							t.Errorf("Expected error for running nonexistent model: %s", name)
						}
					})

					t.Run(fmt.Sprintf("DeleteModel_%s", name), func(t *testing.T) {
						err := app.DeleteModel(name)
						if err == nil {
							t.Errorf("Expected error for deleting nonexistent model: %s", name)
						}
					})
				}
			}
		})

		// Test invalid configuration updates
		t.Run("Invalid Configuration", func(t *testing.T) {
			invalidConfigs := []ConfigDTO{
				{OllamaAPIURL: "", LogLevel: "info"},                         // Empty URL
				{OllamaAPIURL: "invalid-url", LogLevel: "invalid"},           // Invalid log level
				{OllamaAPIURL: "http://localhost:11434", RefreshInterval: 0}, // Invalid refresh interval
				{OllamaAPIURL: "http://localhost:11434", WindowWidth: 100},   // Too small window
			}

			for i, config := range invalidConfigs {
				t.Run(fmt.Sprintf("InvalidConfig_%d", i), func(t *testing.T) {
					err := app.UpdateConfig(config)
					if err == nil {
						t.Errorf("Expected error for invalid config %d", i)
					}
				})
			}
		})

		// Test invalid vRAM constraints
		t.Run("Invalid vRAM Constraints", func(t *testing.T) {
			invalidConstraints := []VRAMConstraintsDTO{
				{AvailableVRAM: 0, Context: 2048},                // Zero vRAM
				{AvailableVRAM: 16, Context: 100},                // Too small context
				{AvailableVRAM: 16, Context: 200000},             // Too large context
				{AvailableVRAM: 16, Context: 2048, BatchSize: 0}, // Zero batch size
			}

			for i, constraints := range invalidConstraints {
				t.Run(fmt.Sprintf("InvalidConstraints_%d", i), func(t *testing.T) {
					_, err := app.EstimateVRAMForModel("test-model", constraints)
					if err == nil {
						t.Errorf("Expected error for invalid constraints %d", i)
					}
				})
			}
		})
	})

	app.OnShutdown(ctx)
}

// TestE2EPerformance tests performance characteristics
func TestE2EPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E performance tests in short mode")
	}

	app := NewApp()
	ctx := context.Background()
	app.OnStartup(ctx)

	if app.service == nil {
		t.Skip("Skipping performance tests - Ollama service not available")
	}

	t.Run("Performance Tests", func(t *testing.T) {
		// Test response times for common operations
		operations := []struct {
			name string
			fn   func() error
		}{
			{"GetModels", func() error { _, err := app.GetModels(); return err }},
			{"GetRunningModels", func() error { _, err := app.GetRunningModels(); return err }},
			{"GetConfig", func() error { _, err := app.GetConfig(); return err }},
			{"HealthCheck", func() error { return app.HealthCheck() }},
			{"GetServiceStatus", func() error { _, err := app.GetServiceStatus(); return err }},
			{"GetSystemInfo", func() error { _, err := app.GetSystemInfo(); return err }},
		}

		for _, op := range operations {
			t.Run(op.name, func(t *testing.T) {
				start := time.Now()
				err := op.fn()
				duration := time.Since(start)

				if err != nil {
					t.Errorf("Operation %s failed: %v", op.name, err)
				}

				t.Logf("Operation %s took %v", op.name, duration)

				// Warn if operation takes too long (adjust thresholds as needed)
				if duration > 5*time.Second {
					t.Logf("WARNING: Operation %s took longer than expected: %v", op.name, duration)
				}
			})
		}

		// Test concurrent operations
		t.Run("Concurrent Operations", func(t *testing.T) {
			const numConcurrent = 5
			done := make(chan error, numConcurrent)

			start := time.Now()
			for i := 0; i < numConcurrent; i++ {
				go func() {
					_, err := app.GetModels()
					done <- err
				}()
			}

			// Wait for all operations to complete
			for i := 0; i < numConcurrent; i++ {
				err := <-done
				if err != nil {
					t.Errorf("Concurrent operation %d failed: %v", i, err)
				}
			}

			duration := time.Since(start)
			t.Logf("Concurrent operations completed in %v", duration)
		})
	})

	app.OnShutdown(ctx)
}

// TestE2EDataIntegrity tests data consistency and integrity
func TestE2EDataIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E data integrity tests in short mode")
	}

	app := NewApp()
	ctx := context.Background()
	app.OnStartup(ctx)

	if app.service == nil {
		t.Skip("Skipping data integrity tests - Ollama service not available")
	}

	t.Run("Data Integrity Tests", func(t *testing.T) {
		// Test data consistency between different endpoints
		t.Run("Model Data Consistency", func(t *testing.T) {
			// Get all models
			allModels, err := app.GetModels()
			if err != nil {
				t.Fatalf("Failed to get all models: %v", err)
			}

			// Get running models
			runningModels, err := app.GetRunningModels()
			if err != nil {
				t.Fatalf("Failed to get running models: %v", err)
			}

			// Check that running models are marked as running in all models
			runningMap := make(map[string]bool)
			for _, rm := range runningModels {
				runningMap[rm.Name] = true
			}

			for _, model := range allModels {
				expectedRunning := runningMap[model.Name]
				if model.IsRunning != expectedRunning {
					t.Errorf("Model %s running status inconsistent: allModels=%v, runningModels=%v",
						model.Name, model.IsRunning, expectedRunning)
				}
			}
		})

		// Test DTO conversion integrity
		t.Run("DTO Conversion Integrity", func(t *testing.T) {
			// Get config and verify DTO conversion
			config, err := app.GetConfig()
			if err != nil {
				t.Fatalf("Failed to get config: %v", err)
			}

			// Validate required fields are present
			if config.OllamaAPIURL == "" {
				t.Error("Config DTO missing OllamaAPIURL")
			}
			if config.LogLevel == "" {
				t.Error("Config DTO missing LogLevel")
			}
			if config.RefreshInterval <= 0 {
				t.Error("Config DTO has invalid RefreshInterval")
			}
		})

		// Test model detail consistency
		t.Run("Model Detail Consistency", func(t *testing.T) {
			models, err := app.GetModels()
			if err != nil || len(models) == 0 {
				t.Skip("No models available for detail consistency test")
			}

			testModel := models[0]

			// Get model details
			details, err := app.GetModelInfo(testModel.Name)
			if err != nil {
				t.Errorf("Failed to get model details: %v", err)
				return
			}

			// Verify consistency between model list and model details
			if details.Model.Name != testModel.Name {
				t.Errorf("Model name inconsistent: list=%s, details=%s", testModel.Name, details.Model.Name)
			}
			if details.Model.Size != testModel.Size {
				t.Errorf("Model size inconsistent: list=%d, details=%d", testModel.Size, details.Model.Size)
			}
		})
	})

	app.OnShutdown(ctx)
}

// Helper functions are defined in app_test.go
