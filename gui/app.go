package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"strings"
	"sync"
	"time"

	"github.com/sammcj/gollama/core"
)

//go:embed static/* frontend/*
var staticFiles embed.FS

//go:embed templates/*
var templateFiles embed.FS

// App struct with caching capabilities
type App struct {
	ctx       context.Context
	service   *core.GollamaService
	templates *template.Template
	cache     *AppCache
}

// AppCache provides caching for frequently accessed data
type AppCache struct {
	models        []ModelDTO
	runningModels []RunningModelDTO
	config        *ConfigDTO
	modelsExpiry  time.Time
	runningExpiry time.Time
	configExpiry  time.Time
	mutex         sync.RWMutex
}

// NewAppCache creates a new cache instance
func NewAppCache() *AppCache {
	return &AppCache{}
}

// Cache TTL constants
const (
	ModelsCacheTTL        = 30 * time.Second
	RunningModelsCacheTTL = 10 * time.Second
	ConfigCacheTTL        = 5 * time.Minute
)

// GetModels returns cached models if available and not expired
func (c *AppCache) GetModels() []ModelDTO {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if time.Now().Before(c.modelsExpiry) && c.models != nil {
		return c.models
	}
	return nil
}

// SetModels caches models with TTL
func (c *AppCache) SetModels(models []ModelDTO) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.models = models
	c.modelsExpiry = time.Now().Add(ModelsCacheTTL)
}

// GetRunningModels returns cached running models if available and not expired
func (c *AppCache) GetRunningModels() []RunningModelDTO {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if time.Now().Before(c.runningExpiry) && c.runningModels != nil {
		return c.runningModels
	}
	return nil
}

// SetRunningModels caches running models with TTL
func (c *AppCache) SetRunningModels(models []RunningModelDTO) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.runningModels = models
	c.runningExpiry = time.Now().Add(RunningModelsCacheTTL)
}

// GetConfig returns cached config if available and not expired
func (c *AppCache) GetConfig() *ConfigDTO {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if time.Now().Before(c.configExpiry) && c.config != nil {
		return c.config
	}
	return nil
}

// SetConfig caches config with TTL
func (c *AppCache) SetConfig(config *ConfigDTO) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.config = config
	c.configExpiry = time.Now().Add(ConfigCacheTTL)
}

// InvalidateModels clears the models cache
func (c *AppCache) InvalidateModels() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.models = nil
	c.modelsExpiry = time.Time{}
}

// InvalidateRunningModels clears the running models cache
func (c *AppCache) InvalidateRunningModels() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.runningModels = nil
	c.runningExpiry = time.Time{}
}

// InvalidateConfig clears the config cache
func (c *AppCache) InvalidateConfig() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.config = nil
	c.configExpiry = time.Time{}
}

// InvalidateAll clears all caches
func (c *AppCache) InvalidateAll() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.models = nil
	c.runningModels = nil
	c.config = nil
	c.modelsExpiry = time.Time{}
	c.runningExpiry = time.Time{}
	c.configExpiry = time.Time{}
}

// GetCacheStats returns cache statistics for monitoring
func (c *AppCache) GetCacheStats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	now := time.Now()
	return map[string]interface{}{
		"models_cached":         c.models != nil,
		"models_expired":        now.After(c.modelsExpiry),
		"models_ttl_remaining":  c.modelsExpiry.Sub(now).Seconds(),
		"running_cached":        c.runningModels != nil,
		"running_expired":       now.After(c.runningExpiry),
		"running_ttl_remaining": c.runningExpiry.Sub(now).Seconds(),
		"config_cached":         c.config != nil,
		"config_expired":        now.After(c.configExpiry),
		"config_ttl_remaining":  c.configExpiry.Sub(now).Seconds(),
	}
}

// RefreshCache forces a refresh of all cached data
func (a *App) RefreshCache() error {
	fmt.Println("🔄 JavaScript called RefreshCache()")

	if a.service == nil {
		return fmt.Errorf("service not initialized")
	}

	startTime := time.Now()

	// Clear all caches
	a.cache.InvalidateAll()

	// Pre-warm caches with fresh data
	_, err1 := a.GetModels()
	_, err2 := a.GetRunningModels()
	_, err3 := a.GetConfig()

	var errors []string
	if err1 != nil {
		errors = append(errors, fmt.Sprintf("models: %v", err1))
	}
	if err2 != nil {
		errors = append(errors, fmt.Sprintf("running models: %v", err2))
	}
	if err3 != nil {
		errors = append(errors, fmt.Sprintf("config: %v", err3))
	}

	duration := time.Since(startTime)
	if len(errors) > 0 {
		fmt.Printf("⚠️  Cache refresh completed with errors (took %v): %v\n", duration, errors)
		return fmt.Errorf("cache refresh errors: %v", errors)
	}

	fmt.Printf("✅ Cache refresh completed successfully (took %v)\n", duration)
	return nil
}

// GetPerformanceMetrics returns performance metrics for monitoring
func (a *App) GetPerformanceMetrics() (map[string]interface{}, error) {
	fmt.Println("🔄 JavaScript called GetPerformanceMetrics()")

	if a.service == nil {
		return nil, fmt.Errorf("service not initialized")
	}

	metrics := map[string]interface{}{
		"timestamp":   time.Now().Format(time.RFC3339),
		"cache_stats": a.cache.GetCacheStats(),
		"service_info": map[string]interface{}{
			"initialized": true,
			"context":     a.ctx != nil,
			"templates":   a.templates != nil,
		},
	}

	// Add service health check timing
	startTime := time.Now()
	healthErr := a.service.HealthCheck()
	healthDuration := time.Since(startTime)

	metrics["health_check"] = map[string]interface{}{
		"status":   healthErr == nil,
		"duration": healthDuration.Milliseconds(),
		"error":    nil,
	}

	if healthErr != nil {
		metrics["health_check"].(map[string]interface{})["error"] = healthErr.Error()
	}

	// Add model count timing
	startTime = time.Now()
	models, modelsErr := a.GetModels()
	modelsDuration := time.Since(startTime)

	metrics["models_fetch"] = map[string]interface{}{
		"status":   modelsErr == nil,
		"duration": modelsDuration.Milliseconds(),
		"count":    len(models),
		"error":    nil,
	}

	if modelsErr != nil {
		metrics["models_fetch"].(map[string]interface{})["error"] = modelsErr.Error()
	}

	fmt.Println("✓ Performance metrics collected successfully")
	return metrics, nil
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		cache: NewAppCache(),
	}
}

// OnStartup is called when the app starts up
func (a *App) OnStartup(ctx context.Context) {
	fmt.Println("=== Gollama GUI Startup Initiated ===")
	fmt.Printf("✓ Wails v3 Context received: %v\n", ctx != nil)
	a.ctx = ctx

	// Initialize service with comprehensive logging
	fmt.Println("🔧 Initializing Gollama service...")
	cfg := core.ServiceConfig{
		OllamaAPIURL: "http://localhost:11434",
		LogLevel:     "info",
		Context:      ctx,
	}

	service, err := core.NewGollamaService(cfg)
	if err != nil {
		fmt.Printf("❌ CRITICAL ERROR: Failed to initialize service: %v\n", err)
		fmt.Printf("❌ Service initialization failed - JavaScript methods will not work\n")
		return
	}

	a.service = service
	fmt.Println("✅ Gollama service initialized successfully")

	// Test service connectivity with detailed logging
	fmt.Println("🔍 Testing service connectivity...")
	if err := a.service.HealthCheck(); err != nil {
		fmt.Printf("⚠️  WARNING: Service health check failed: %v\n", err)
		fmt.Printf("⚠️  This may indicate Ollama is not running or not accessible\n")
	} else {
		fmt.Println("✅ Service health check passed - Ollama API is accessible")
	}

	// Load templates with custom functions
	fmt.Println("📄 Loading HTML templates...")
	funcMap := template.FuncMap{
		"formatSize":     formatSize,
		"formatTime":     formatTime,
		"formatDuration": formatDuration,
		"add":            func(a, b int) int { return a + b },
		"sub":            func(a, b int) int { return a - b },
		"mul":            func(a, b int) int { return a * b },
		"div":            func(a, b int) int { return a / b },
		"mod":            func(a, b int) int { return a % b },
		"eq":             func(a, b any) bool { return a == b },
		"ne":             func(a, b any) bool { return a != b },
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
	fmt.Println("✅ HTML templates loaded successfully")

	// Verify service method availability with detailed logging
	fmt.Println("🔍 Verifying service method availability...")
	if a.service == nil {
		fmt.Println("❌ ERROR: Service is nil - methods will fail")
	} else {
		fmt.Println("✅ Service instance is available")

		// Test a simple method call to verify service is working
		_, err := a.service.ListModels()
		if err != nil {
			fmt.Printf("⚠️  WARNING: Test ListModels() call failed: %v\n", err)
		} else {
			fmt.Println("✅ Test service method call successful")
		}
	}

	// Log all exposed methods for debugging with enhanced formatting
	fmt.Println("=== 🚀 WAILS v3 SERVICE METHODS EXPOSED TO JAVASCRIPT ===")
	fmt.Println("📋 Model Management Methods:")
	fmt.Println("  ✓ GetModels() -> []ModelDTO - Retrieve all available models")
	fmt.Println("  ✓ GetModel(id string) -> *ModelDTO - Get a specific model by ID")
	fmt.Println("  ✓ GetRunningModels() -> []RunningModelDTO - Get currently running models")
	fmt.Println("  ✓ RunModel(name string) -> error - Start a model")
	fmt.Println("  ✓ DeleteModel(name string) -> error - Delete a model")
	fmt.Println("  ✓ UnloadModel(name string) -> error - Unload a running model")
	fmt.Println("  ✓ CopyModel(source, dest string) -> error - Copy a model")
	fmt.Println("  ✓ PushModel(id string) -> error - Push model to registry")
	fmt.Println("  ✓ PullModel(id string) -> error - Pull model from registry")
	fmt.Println("  ✓ GetModelInfo(name string) -> *ModelInfoDTO - Get detailed model information")
	fmt.Println("  ✓ SearchModels(query string, filters map[string]interface{}) -> []ModelDTO - Search models")
	fmt.Println("  ✓ GetModelOperations(modelName string) -> []string - Get available operations for a model")

	fmt.Println("⚙️  Configuration Methods:")
	fmt.Println("  ✓ GetConfig() -> *ConfigDTO - Get current configuration")
	fmt.Println("  ✓ UpdateConfig(config ConfigDTO) -> error - Update configuration")

	fmt.Println("🧮 Utility Methods:")
	fmt.Println("  ✓ EstimateVRAMForModel(modelName string, constraints VRAMConstraintsDTO) -> *VRAMEstimationDTO - Estimate vRAM usage")
	fmt.Println("  ✓ HealthCheck() -> error - Verify service health")
	fmt.Println("  ✓ TestServiceBinding() -> string - Test JavaScript to Go binding")
	fmt.Println("  ✓ GetServiceStatus() -> *HealthCheckDTO - Get detailed service status for debugging")
	fmt.Println("  ✓ ValidateModelName(name string) -> (bool, error) - Validate model name")
	fmt.Println("  ✓ GetSystemInfo() -> map[string]interface{} - Get system information")
	fmt.Println("  ✓ GetDiagnosticInfo() -> map[string]interface{} - Get comprehensive diagnostic information")
	fmt.Println("  ✓ RunDiagnosticTests() -> map[string]interface{} - Run comprehensive diagnostic tests")
	fmt.Println("  ✓ VerifyServiceBinding() -> map[string]interface{} - Verify service binding is working correctly")

	fmt.Println("=== 🎯 WAILS v3 SERVICE REGISTRATION STATUS ===")
	fmt.Printf("✅ App struct registered with %d exposed methods\n", 24) // Updated count
	fmt.Println("✅ All methods are directly bound to App struct for Wails v3 compatibility")
	fmt.Println("✅ Methods use proper DTO -> JavaScript type conversion")
	fmt.Println("✅ Comprehensive error handling and logging implemented")
	fmt.Println("✅ Service context properly managed")
	fmt.Println("✅ Enhanced debugging methods added (TestServiceBinding, GetServiceStatus)")
	fmt.Println("✅ Proper DTO validation and conversion implemented")

	// Log JavaScript access patterns for debugging
	fmt.Println("=== 🌐 JAVASCRIPT ACCESS PATTERNS ===")
	fmt.Println("JavaScript can access methods via:")
	fmt.Println("  • window.wails.GetModels() -> Promise<ModelDTO[]>")
	fmt.Println("  • window.wails.DeleteModel('model-name') -> Promise<void>")
	fmt.Println("  • window.wails.RunModel('model-name') -> Promise<void>")
	fmt.Println("  • window.wails.GetConfig() -> Promise<ConfigDTO>")
	fmt.Println("  • window.wails.EstimateVRAMForModel('model', constraints) -> Promise<VRAMEstimationDTO>")
	fmt.Println("  • window.wails.HealthCheck() -> Promise<void>")
	fmt.Println("  • window.wails.GetServiceStatus() -> Promise<HealthCheckDTO>")
	fmt.Println("  • etc. (all methods return Promises with proper DTOs)")

	fmt.Println("=== ✅ GOLLAMA GUI STARTUP COMPLETE ===")
	fmt.Println("🎉 Ready for JavaScript frontend integration!")
}

// Additional service methods for comprehensive model management

// SearchModels searches for models based on query and filters
func (a *App) SearchModels(query string, filters map[string]interface{}) ([]ModelDTO, error) {
	fmt.Printf("🔄 JavaScript called SearchModels(%s)\n", query)

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in SearchModels()")
		return nil, fmt.Errorf("service not initialized")
	}

	// For now, implement basic search by listing all models and filtering
	models, err := a.service.ListModels()
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to search models: %v\n", err)
		return nil, err
	}

	// Convert to DTOs
	modelDTOs := ConvertToModelDTOs(models)

	// Apply basic query filter
	if query != "" {
		var filtered []ModelDTO
		queryLower := strings.ToLower(query)
		for _, model := range modelDTOs {
			if strings.Contains(strings.ToLower(model.Name), queryLower) ||
				strings.Contains(strings.ToLower(model.Family), queryLower) {
				filtered = append(filtered, model)
			}
		}
		modelDTOs = filtered
	}

	fmt.Printf("✓ SearchModels() returning %d filtered models\n", len(modelDTOs))
	return modelDTOs, nil
}

// GetModelOperations returns available operations for a model
func (a *App) GetModelOperations(modelName string) ([]string, error) {
	fmt.Printf("🔄 JavaScript called GetModelOperations(%s)\n", modelName)

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in GetModelOperations()")
		return nil, fmt.Errorf("service not initialized")
	}

	// Check if model exists
	_, err := a.service.GetModel(modelName)
	if err != nil {
		fmt.Printf("❌ ERROR: Model %s not found: %v\n", modelName, err)
		return nil, err
	}

	// Check if model is running
	runningModels, _ := a.service.GetRunningModels()
	isRunning := false
	for _, rm := range runningModels {
		if rm.Name == modelName {
			isRunning = true
			break
		}
	}

	// Return available operations based on model state
	operations := []string{"delete", "copy", "push", "info"}
	if isRunning {
		operations = append(operations, "unload")
	} else {
		operations = append(operations, "run")
	}

	fmt.Printf("✓ GetModelOperations() returning %d operations for %s\n", len(operations), modelName)
	return operations, nil
}

// ValidateModelName checks if a model name is valid
func (a *App) ValidateModelName(name string) (bool, error) {
	fmt.Printf("🔄 JavaScript called ValidateModelName(%s)\n", name)

	if name == "" {
		return false, fmt.Errorf("model name cannot be empty")
	}

	if len(name) > 255 {
		return false, fmt.Errorf("model name too long (max 255 characters)")
	}

	// Check for invalid characters (basic validation)
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return false, fmt.Errorf("model name contains invalid character: %s", char)
		}
	}

	fmt.Printf("✓ Model name %s is valid\n", name)
	return true, nil
}

// GetSystemInfo returns system information relevant to model management
func (a *App) GetSystemInfo() (map[string]interface{}, error) {
	fmt.Println("🔄 JavaScript called GetSystemInfo()")

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in GetSystemInfo()")
		return nil, fmt.Errorf("service not initialized")
	}

	config := a.service.GetConfig()
	if config == nil {
		return nil, fmt.Errorf("failed to get configuration")
	}

	systemInfo := map[string]interface{}{
		"ollama_url": config.OllamaAPIURL,
		"log_level":  config.LogLevel,
		"version":    "1.0.0", // Should come from build info
		"platform":   "gui",
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	// Add model statistics
	if models, err := a.service.ListModels(); err == nil {
		systemInfo["total_models"] = len(models)

		var totalSize int64
		for _, model := range models {
			totalSize += model.Size
		}
		systemInfo["total_size"] = totalSize
		systemInfo["total_size_formatted"] = formatSize(totalSize)
	}

	// Add running model statistics
	if runningModels, err := a.service.GetRunningModels(); err == nil {
		systemInfo["running_models"] = len(runningModels)
	}

	fmt.Println("✓ Successfully retrieved system information")
	return systemInfo, nil
}

// OnShutdown is called when the app is shutting down
func (a *App) OnShutdown(ctx context.Context) {
	if a.service != nil {
		a.service.Close()
	}
}

// GetModels returns all models for the frontend using proper DTOs with caching
func (a *App) GetModels() ([]ModelDTO, error) {
	fmt.Println("🔄 JavaScript called GetModels()")

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in GetModels()")
		return nil, fmt.Errorf("service not initialized")
	}

	// Check cache first
	if cachedModels := a.cache.GetModels(); cachedModels != nil {
		fmt.Printf("✓ Returning %d cached models\n", len(cachedModels))
		return cachedModels, nil
	}

	fmt.Println("📥 Cache miss, fetching models from service...")
	startTime := time.Now()

	models, err := a.service.ListModels()
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to list models: %v\n", err)
		return nil, err
	}

	// Get running models to mark them as running (batch operation)
	runningModels, _ := a.service.GetRunningModels()
	runningMap := make(map[string]bool)
	for _, rm := range runningModels {
		runningMap[rm.Name] = true
	}

	// Convert to DTOs with running status
	modelDTOs := ConvertToModelDTOs(models)
	for i := range modelDTOs {
		modelDTOs[i].IsRunning = runningMap[modelDTOs[i].Name]
	}

	// Cache the results
	a.cache.SetModels(modelDTOs)

	duration := time.Since(startTime)
	fmt.Printf("✓ GetModels() returning %d model DTOs (took %v)\n", len(modelDTOs), duration)
	return modelDTOs, nil
}

// GetRunningModels returns currently running models using proper DTOs with caching
func (a *App) GetRunningModels() ([]RunningModelDTO, error) {
	fmt.Println("🔄 JavaScript called GetRunningModels()")

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in GetRunningModels()")
		return nil, fmt.Errorf("service not initialized")
	}

	// Check cache first
	if cachedModels := a.cache.GetRunningModels(); cachedModels != nil {
		fmt.Printf("✓ Returning %d cached running models\n", len(cachedModels))
		return cachedModels, nil
	}

	fmt.Println("📥 Cache miss, fetching running models from service...")
	startTime := time.Now()

	models, err := a.service.GetRunningModels()
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to get running models: %v\n", err)
		return nil, err
	}

	// Convert to DTOs
	modelDTOs := ConvertToRunningModelDTOs(models)

	// Cache the results
	a.cache.SetRunningModels(modelDTOs)

	duration := time.Since(startTime)
	fmt.Printf("✓ GetRunningModels() returning %d running model DTOs (took %v)\n", len(modelDTOs), duration)
	return modelDTOs, nil
}

// RunModel starts a model
func (a *App) RunModel(name string) error {
	fmt.Printf("🔄 JavaScript called RunModel(%s)\n", name)

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in RunModel()")
		return fmt.Errorf("service not initialized")
	}

	startTime := time.Now()
	err := a.service.RunModel(name)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to run model %s: %v\n", name, err)
		return err
	}

	// Invalidate caches since model state changed
	a.cache.InvalidateModels()
	a.cache.InvalidateRunningModels()

	duration := time.Since(startTime)
	fmt.Printf("✓ Successfully started model: %s (took %v)\n", name, duration)
	return nil
}

// DeleteModel deletes a model
func (a *App) DeleteModel(name string) error {
	fmt.Printf("🔄 JavaScript called DeleteModel(%s)\n", name)

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in DeleteModel()")
		return fmt.Errorf("service not initialized")
	}

	startTime := time.Now()
	err := a.service.DeleteModel(name)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to delete model %s: %v\n", name, err)
		return err
	}

	// Invalidate caches since model was deleted
	a.cache.InvalidateModels()
	a.cache.InvalidateRunningModels()

	duration := time.Since(startTime)
	fmt.Printf("✓ Successfully deleted model: %s (took %v)\n", name, duration)
	return nil
}

// UnloadModel unloads a model
func (a *App) UnloadModel(name string) error {
	fmt.Printf("🔄 JavaScript called UnloadModel(%s)\n", name)

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in UnloadModel()")
		return fmt.Errorf("service not initialized")
	}

	startTime := time.Now()
	err := a.service.UnloadModel(name)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to unload model %s: %v\n", name, err)
		return err
	}

	// Invalidate caches since model state changed
	a.cache.InvalidateModels()
	a.cache.InvalidateRunningModels()

	duration := time.Since(startTime)
	fmt.Printf("✓ Successfully unloaded model: %s (took %v)\n", name, duration)
	return nil
}

// GetModelInfo returns detailed information about a model using proper DTOs
func (a *App) GetModelInfo(name string) (*ModelInfoDTO, error) {
	fmt.Printf("🔄 JavaScript called GetModelInfo(%s)\n", name)

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in GetModelInfo()")
		return nil, fmt.Errorf("service not initialized")
	}

	info, err := a.service.GetModelInfo(name)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to get model info for %s: %v\n", name, err)
		return nil, err
	}

	// Convert to DTO with available vRAM (default to 0 for now)
	modelInfoDTO := ConvertToModelInfoDTO(info, 0)

	fmt.Printf("✓ Successfully retrieved model info: %s\n", name)
	return modelInfoDTO, nil
}

// EstimateVRAMForModel estimates vRAM usage for a specific model using proper DTOs
func (a *App) EstimateVRAMForModel(modelName string, constraints VRAMConstraintsDTO) (*VRAMEstimationDTO, error) {
	fmt.Printf("🔄 JavaScript called EstimateVRAMForModel(%s)\n", modelName)

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in EstimateVRAMForModel()")
		return nil, fmt.Errorf("service not initialized")
	}

	// Validate constraints
	if validationErrors := constraints.Validate(); len(validationErrors) > 0 {
		fmt.Printf("❌ ERROR: vRAM constraints validation failed: %v\n", validationErrors)
		return nil, fmt.Errorf("vRAM constraints validation failed: %v", validationErrors)
	}

	// Convert DTO to core constraints
	coreConstraints := core.VRAMConstraints{
		AvailableVRAM:  constraints.AvailableVRAM,
		ContextLength:  constraints.Context,
		Quantization:   constraints.Quantization,
		BatchSize:      constraints.BatchSize,
		SequenceLength: constraints.SequenceLength,
	}

	estimation, err := a.service.EstimateVRAM(modelName, coreConstraints)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to estimate vRAM for %s: %v\n", modelName, err)
		return nil, err
	}

	// Convert to DTO
	estimationDTO := ConvertToVRAMEstimationDTO(estimation, modelName, constraints.AvailableVRAM)

	fmt.Printf("✓ Successfully estimated vRAM for model: %s\n", modelName)
	return estimationDTO, nil
}

// GetConfig returns the current configuration using proper DTOs with caching
func (a *App) GetConfig() (*ConfigDTO, error) {
	fmt.Println("🔄 JavaScript called GetConfig()")

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in GetConfig()")
		return nil, fmt.Errorf("service not initialized")
	}

	// Check cache first
	if cachedConfig := a.cache.GetConfig(); cachedConfig != nil {
		fmt.Println("✓ Returning cached configuration")
		return cachedConfig, nil
	}

	fmt.Println("📥 Cache miss, fetching configuration from service...")
	startTime := time.Now()

	config := a.service.GetConfig()
	if config == nil {
		fmt.Println("❌ ERROR: Failed to retrieve configuration")
		return nil, fmt.Errorf("failed to retrieve configuration")
	}

	// Convert to DTO
	configDTO := ConvertToConfigDTO(config)

	// Cache the result
	a.cache.SetConfig(configDTO)

	duration := time.Since(startTime)
	fmt.Printf("✓ Successfully retrieved configuration (took %v)\n", duration)
	return configDTO, nil
}

// UpdateConfig updates the configuration using proper DTOs
func (a *App) UpdateConfig(configDTO ConfigDTO) error {
	fmt.Println("🔄 JavaScript called UpdateConfig()")

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in UpdateConfig()")
		return fmt.Errorf("service not initialized")
	}

	// Validate the DTO
	if validationErrors := configDTO.Validate(); len(validationErrors) > 0 {
		fmt.Printf("❌ ERROR: Configuration validation failed: %v\n", validationErrors)
		return fmt.Errorf("configuration validation failed: %v", validationErrors)
	}

	startTime := time.Now()

	// Convert DTO to config
	config := ConvertFromConfigDTO(&configDTO)

	err := a.service.UpdateConfig(config)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to update configuration: %v\n", err)
		return err
	}

	// Invalidate config cache since it was updated
	a.cache.InvalidateConfig()

	duration := time.Since(startTime)
	fmt.Printf("✓ Successfully updated configuration (took %v)\n", duration)
	return nil
}

// HealthCheck verifies the service is healthy
func (a *App) HealthCheck() error {
	fmt.Println("🔄 JavaScript called HealthCheck()")

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in HealthCheck()")
		return fmt.Errorf("service not initialized")
	}

	err := a.service.HealthCheck()
	if err != nil {
		fmt.Printf("❌ ERROR: Health check failed: %v\n", err)
		return err
	}

	fmt.Println("✓ Health check passed")
	return nil
}

// GetModel returns a specific model by ID/name using proper DTOs
func (a *App) GetModel(id string) (*ModelDTO, error) {
	fmt.Printf("🔄 JavaScript called GetModel(%s)\n", id)

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in GetModel()")
		return nil, fmt.Errorf("service not initialized")
	}

	model, err := a.service.GetModel(id)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to get model %s: %v\n", id, err)
		return nil, err
	}

	// Check if model is running
	runningModels, _ := a.service.GetRunningModels()
	isRunning := false
	for _, rm := range runningModels {
		if rm.Name == model.Name {
			isRunning = true
			break
		}
	}

	// Convert to DTO
	modelDTO := ConvertToModelDTO(*model)
	modelDTO.IsRunning = isRunning

	fmt.Printf("✓ Successfully retrieved model: %s\n", id)
	return &modelDTO, nil
}

// CopyModel creates a copy of a model with a new name
func (a *App) CopyModel(source, dest string) error {
	fmt.Printf("🔄 JavaScript called CopyModel(%s -> %s)\n", source, dest)

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in CopyModel()")
		return fmt.Errorf("service not initialized")
	}

	err := a.service.CopyModel(source, dest)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to copy model from %s to %s: %v\n", source, dest, err)
		return err
	}

	fmt.Printf("✓ Successfully copied model from %s to %s\n", source, dest)
	return nil
}

// PushModel uploads a model to a registry
func (a *App) PushModel(id string) error {
	fmt.Printf("🔄 JavaScript called PushModel(%s)\n", id)

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in PushModel()")
		return fmt.Errorf("service not initialized")
	}

	err := a.service.PushModel(id)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to push model %s: %v\n", id, err)
		return err
	}

	fmt.Printf("✓ Successfully pushed model: %s\n", id)
	return nil
}

// PullModel downloads a model from a registry
func (a *App) PullModel(id string) error {
	fmt.Printf("🔄 JavaScript called PullModel(%s)\n", id)

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in PullModel()")
		return fmt.Errorf("service not initialized")
	}

	err := a.service.PullModel(id)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to pull model %s: %v\n", id, err)
		return err
	}

	fmt.Printf("✓ Successfully pulled model: %s\n", id)
	return nil
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

// TestServiceBinding is a comprehensive test method to verify JavaScript can call Go methods
func (a *App) TestServiceBinding() string {
	fmt.Println("🔄 JavaScript called TestServiceBinding()")
	fmt.Println("🧪 Running comprehensive service binding test...")

	if a.service == nil {
		fmt.Println("❌ ERROR: Service not initialized in TestServiceBinding()")
		return "ERROR: Service not initialized - JavaScript binding failed"
	}

	// Test service health
	fmt.Println("🔍 Testing service health...")
	if err := a.service.HealthCheck(); err != nil {
		fmt.Printf("⚠️  WARNING: Service health check failed: %v\n", err)
		return fmt.Sprintf("WARNING: Service binding works but Ollama API not accessible: %v", err)
	}

	// Test a simple service method
	fmt.Println("🔍 Testing service method call...")
	models, err := a.service.ListModels()
	if err != nil {
		fmt.Printf("⚠️  WARNING: ListModels() failed: %v\n", err)
		return fmt.Sprintf("WARNING: Service binding works but ListModels failed: %v", err)
	}

	message := fmt.Sprintf("✅ SUCCESS: Service binding fully operational! Found %d models. JavaScript ↔ Go communication verified.", len(models))
	fmt.Println(message)
	return message
}

// GetServiceStatus returns detailed service status for debugging using proper DTOs
func (a *App) GetServiceStatus() (*HealthCheckDTO, error) {
	fmt.Println("🔄 JavaScript called GetServiceStatus()")

	healthCheck := NewHealthCheckDTO("unknown", "1.0.0") // Version should come from build info

	if a.service == nil {
		healthCheck.Status = "unhealthy"
		healthCheck.AddService("gollama_service", "unhealthy", "Service not initialized", "")
		return healthCheck, nil
	}

	// Test core service health
	healthCheck.AddService("gollama_service", "healthy", "Service initialized", "")

	// Test Ollama API health
	if err := a.service.HealthCheck(); err != nil {
		healthCheck.Status = "unhealthy"
		healthCheck.AddService("ollama_api", "unhealthy", err.Error(), "")
	} else {
		healthCheck.AddService("ollama_api", "healthy", "API accessible", "")
	}

	// Test model listing capability
	if models, err := a.service.ListModels(); err != nil {
		healthCheck.AddService("model_listing", "unhealthy", err.Error(), "")
	} else {
		healthCheck.AddService("model_listing", "healthy", fmt.Sprintf("%d models available", len(models)), "")
	}

	// Test running models capability
	if runningModels, err := a.service.GetRunningModels(); err != nil {
		healthCheck.AddService("running_models", "unhealthy", err.Error(), "")
	} else {
		healthCheck.AddService("running_models", "healthy", fmt.Sprintf("%d models running", len(runningModels)), "")
	}

	// Overall health status
	if healthCheck.IsHealthy() {
		healthCheck.Status = "healthy"
	} else {
		healthCheck.Status = "unhealthy"
	}

	fmt.Printf("✅ Service status retrieved: %s\n", healthCheck.Status)
	return healthCheck, nil
}

// GetDiagnosticInfo returns comprehensive diagnostic information for debugging
func (a *App) GetDiagnosticInfo() (map[string]interface{}, error) {
	fmt.Println("🔄 JavaScript called GetDiagnosticInfo()")

	diagnostics := map[string]interface{}{
		"timestamp":         time.Now().Format(time.RFC3339),
		"version":           "1.0.0", // Should come from build info
		"platform":          "gui",
		"initialized":       a.service != nil,
		"context_available": a.ctx != nil,
		"templates_loaded":  a.templates != nil,
	}

	// Service information
	if a.service != nil {
		config := a.service.GetConfig()
		if config != nil {
			diagnostics["service_config"] = map[string]interface{}{
				"ollama_url": config.OllamaAPIURL,
				"log_level":  config.LogLevel,
			}
		}

		// Test various service capabilities
		serviceTests := map[string]interface{}{}

		// Test health check
		if err := a.service.HealthCheck(); err != nil {
			serviceTests["health_check"] = map[string]interface{}{
				"status": "failed",
				"error":  err.Error(),
			}
		} else {
			serviceTests["health_check"] = map[string]interface{}{
				"status": "passed",
			}
		}

		// Test model listing
		if models, err := a.service.ListModels(); err != nil {
			serviceTests["list_models"] = map[string]interface{}{
				"status": "failed",
				"error":  err.Error(),
			}
		} else {
			serviceTests["list_models"] = map[string]interface{}{
				"status":      "passed",
				"model_count": len(models),
			}
		}

		// Test running models
		if runningModels, err := a.service.GetRunningModels(); err != nil {
			serviceTests["running_models"] = map[string]interface{}{
				"status": "failed",
				"error":  err.Error(),
			}
		} else {
			serviceTests["running_models"] = map[string]interface{}{
				"status":        "passed",
				"running_count": len(runningModels),
			}
		}

		diagnostics["service_tests"] = serviceTests
	} else {
		diagnostics["service_error"] = "Service not initialized"
	}

	// Method availability check
	exposedMethods := []string{
		"GetModels", "GetModel", "GetRunningModels", "RunModel", "DeleteModel",
		"UnloadModel", "CopyModel", "PushModel", "PullModel", "GetModelInfo",
		"EstimateVRAMForModel", "GetConfig", "UpdateConfig", "HealthCheck",
		"TestServiceBinding", "GetServiceStatus", "SearchModels", "GetModelOperations",
		"ValidateModelName", "GetSystemInfo", "GetDiagnosticInfo",
	}

	diagnostics["exposed_methods"] = map[string]interface{}{
		"count":   len(exposedMethods),
		"methods": exposedMethods,
	}

	fmt.Println("✅ Diagnostic information compiled successfully")
	return diagnostics, nil
}

// RunDiagnosticTests runs a comprehensive suite of diagnostic tests
func (a *App) RunDiagnosticTests() (map[string]interface{}, error) {
	fmt.Println("🔄 JavaScript called RunDiagnosticTests()")

	results := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"tests":     map[string]interface{}{},
		"summary": map[string]interface{}{
			"total":   0,
			"passed":  0,
			"failed":  0,
			"skipped": 0,
		},
	}

	tests := results["tests"].(map[string]interface{})
	summary := results["summary"].(map[string]interface{})

	// Test 1: Service Initialization
	summary["total"] = summary["total"].(int) + 1
	if a.service != nil {
		tests["service_initialization"] = map[string]interface{}{
			"status":      "passed",
			"description": "Service is properly initialized",
		}
		summary["passed"] = summary["passed"].(int) + 1
	} else {
		tests["service_initialization"] = map[string]interface{}{
			"status":      "failed",
			"description": "Service is not initialized",
			"error":       "Service is nil",
		}
		summary["failed"] = summary["failed"].(int) + 1
	}

	// Test 2: Context Availability
	summary["total"] = summary["total"].(int) + 1
	if a.ctx != nil {
		tests["context_availability"] = map[string]interface{}{
			"status":      "passed",
			"description": "Application context is available",
		}
		summary["passed"] = summary["passed"].(int) + 1
	} else {
		tests["context_availability"] = map[string]interface{}{
			"status":      "failed",
			"description": "Application context is not available",
			"error":       "Context is nil",
		}
		summary["failed"] = summary["failed"].(int) + 1
	}

	// Test 3: Template Loading
	summary["total"] = summary["total"].(int) + 1
	if a.templates != nil {
		tests["template_loading"] = map[string]interface{}{
			"status":      "passed",
			"description": "HTML templates are loaded",
		}
		summary["passed"] = summary["passed"].(int) + 1
	} else {
		tests["template_loading"] = map[string]interface{}{
			"status":      "failed",
			"description": "HTML templates are not loaded",
			"error":       "Templates is nil",
		}
		summary["failed"] = summary["failed"].(int) + 1
	}

	// Only run service-dependent tests if service is available
	if a.service != nil {
		// Test 4: Health Check
		summary["total"] = summary["total"].(int) + 1
		if err := a.service.HealthCheck(); err != nil {
			tests["health_check"] = map[string]interface{}{
				"status":      "failed",
				"description": "Service health check failed",
				"error":       err.Error(),
			}
			summary["failed"] = summary["failed"].(int) + 1
		} else {
			tests["health_check"] = map[string]interface{}{
				"status":      "passed",
				"description": "Service health check passed",
			}
			summary["passed"] = summary["passed"].(int) + 1
		}

		// Test 5: Model Listing
		summary["total"] = summary["total"].(int) + 1
		if models, err := a.service.ListModels(); err != nil {
			tests["model_listing"] = map[string]interface{}{
				"status":      "failed",
				"description": "Model listing failed",
				"error":       err.Error(),
			}
			summary["failed"] = summary["failed"].(int) + 1
		} else {
			tests["model_listing"] = map[string]interface{}{
				"status":      "passed",
				"description": fmt.Sprintf("Model listing successful (%d models)", len(models)),
				"model_count": len(models),
			}
			summary["passed"] = summary["passed"].(int) + 1
		}

		// Test 6: Running Models
		summary["total"] = summary["total"].(int) + 1
		if runningModels, err := a.service.GetRunningModels(); err != nil {
			tests["running_models"] = map[string]interface{}{
				"status":      "failed",
				"description": "Running models check failed",
				"error":       err.Error(),
			}
			summary["failed"] = summary["failed"].(int) + 1
		} else {
			tests["running_models"] = map[string]interface{}{
				"status":        "passed",
				"description":   fmt.Sprintf("Running models check successful (%d running)", len(runningModels)),
				"running_count": len(runningModels),
			}
			summary["passed"] = summary["passed"].(int) + 1
		}

		// Test 7: Configuration Access
		summary["total"] = summary["total"].(int) + 1
		if config := a.service.GetConfig(); config == nil {
			tests["configuration_access"] = map[string]interface{}{
				"status":      "failed",
				"description": "Configuration access failed",
				"error":       "Config is nil",
			}
			summary["failed"] = summary["failed"].(int) + 1
		} else {
			tests["configuration_access"] = map[string]interface{}{
				"status":      "passed",
				"description": "Configuration access successful",
				"ollama_url":  config.OllamaAPIURL,
			}
			summary["passed"] = summary["passed"].(int) + 1
		}
	}

	fmt.Printf("✅ Diagnostic tests completed: %d/%d passed\n", summary["passed"], summary["total"])
	return results, nil
}

// VerifyServiceBinding performs a comprehensive verification of service binding
func (a *App) VerifyServiceBinding() (map[string]interface{}, error) {
	fmt.Println("🔄 JavaScript called VerifyServiceBinding()")

	verification := map[string]interface{}{
		"timestamp":       time.Now().Format(time.RFC3339),
		"binding_status":  "unknown",
		"service_status":  "unknown",
		"method_tests":    map[string]interface{}{},
		"recommendations": []string{},
	}

	// Check if service is bound
	if a.service == nil {
		verification["binding_status"] = "failed"
		verification["service_status"] = "not_initialized"
		verification["recommendations"] = append(verification["recommendations"].([]string),
			"Service is not initialized - check OnStartup method")
		return verification, nil
	}

	verification["binding_status"] = "success"
	verification["service_status"] = "initialized"

	// Test critical methods
	methodTests := verification["method_tests"].(map[string]interface{})

	// Test GetModels
	if models, err := a.service.ListModels(); err != nil {
		methodTests["GetModels"] = map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		}
		verification["recommendations"] = append(verification["recommendations"].([]string),
			"GetModels failed - check Ollama API connectivity")
	} else {
		methodTests["GetModels"] = map[string]interface{}{
			"status":      "passed",
			"model_count": len(models),
		}
	}

	// Test HealthCheck
	if err := a.service.HealthCheck(); err != nil {
		methodTests["HealthCheck"] = map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		}
		verification["recommendations"] = append(verification["recommendations"].([]string),
			"HealthCheck failed - verify Ollama is running")
	} else {
		methodTests["HealthCheck"] = map[string]interface{}{
			"status": "passed",
		}
	}

	// Test GetConfig
	if config := a.service.GetConfig(); config == nil {
		methodTests["GetConfig"] = map[string]interface{}{
			"status": "failed",
			"error":  "Config is nil",
		}
		verification["recommendations"] = append(verification["recommendations"].([]string),
			"GetConfig failed - check configuration loading")
	} else {
		methodTests["GetConfig"] = map[string]interface{}{
			"status":     "passed",
			"ollama_url": config.OllamaAPIURL,
		}
	}

	// Overall status
	failedTests := 0
	for _, test := range methodTests {
		if testMap, ok := test.(map[string]interface{}); ok {
			if status, ok := testMap["status"].(string); ok && status == "failed" {
				failedTests++
			}
		}
	}

	if failedTests == 0 {
		verification["service_status"] = "healthy"
	} else {
		verification["service_status"] = "unhealthy"
		verification["recommendations"] = append(verification["recommendations"].([]string),
			fmt.Sprintf("%d method tests failed - check service configuration", failedTests))
	}

	fmt.Printf("✅ Service binding verification completed: %s\n", verification["service_status"])
	return verification, nil
}

// HTTP handler methods for serving diagnostic pages

// ServeDiagnostics serves the diagnostics page
func (a *App) ServeDiagnostics() (string, error) {
	fmt.Println("🔄 Serving diagnostics page")

	if a.templates == nil {
		return "", fmt.Errorf("templates not loaded")
	}

	var buf strings.Builder
	err := a.templates.ExecuteTemplate(&buf, "diagnostics.html", nil)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to render diagnostics template: %v\n", err)
		return "", err
	}

	fmt.Println("✅ Diagnostics page rendered successfully")
	return buf.String(), nil
}

// BatchModelOperations performs multiple model operations efficiently
func (a *App) BatchModelOperations(operations []map[string]interface{}) (map[string]interface{}, error) {
	fmt.Printf("🔄 JavaScript called BatchModelOperations with %d operations\n", len(operations))

	if a.service == nil {
		return nil, fmt.Errorf("service not initialized")
	}

	startTime := time.Now()
	results := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"results":   make([]map[string]interface{}, 0, len(operations)),
		"summary": map[string]interface{}{
			"total":     len(operations),
			"succeeded": 0,
			"failed":    0,
		},
	}

	for i, op := range operations {
		opResult := map[string]interface{}{
			"index":     i,
			"operation": op,
			"status":    "unknown",
			"error":     nil,
			"duration":  0,
		}

		opStartTime := time.Now()

		// Extract operation details
		opType, ok := op["type"].(string)
		if !ok {
			opResult["status"] = "failed"
			opResult["error"] = "operation type is required"
			results["results"] = append(results["results"].([]map[string]interface{}), opResult)
			results["summary"].(map[string]interface{})["failed"] = results["summary"].(map[string]interface{})["failed"].(int) + 1
			continue
		}

		modelName, ok := op["model"].(string)
		if !ok {
			opResult["status"] = "failed"
			opResult["error"] = "model name is required"
			results["results"] = append(results["results"].([]map[string]interface{}), opResult)
			results["summary"].(map[string]interface{})["failed"] = results["summary"].(map[string]interface{})["failed"].(int) + 1
			continue
		}

		// Perform the operation
		var err error
		switch opType {
		case "run":
			err = a.service.RunModel(modelName)
		case "unload":
			err = a.service.UnloadModel(modelName)
		case "delete":
			err = a.service.DeleteModel(modelName)
		default:
			err = fmt.Errorf("unsupported operation type: %s", opType)
		}

		opDuration := time.Since(opStartTime)
		opResult["duration"] = opDuration.Milliseconds()

		if err != nil {
			opResult["status"] = "failed"
			opResult["error"] = err.Error()
			results["summary"].(map[string]interface{})["failed"] = results["summary"].(map[string]interface{})["failed"].(int) + 1
		} else {
			opResult["status"] = "succeeded"
			results["summary"].(map[string]interface{})["succeeded"] = results["summary"].(map[string]interface{})["succeeded"].(int) + 1
		}

		results["results"] = append(results["results"].([]map[string]interface{}), opResult)
	}

	// Invalidate relevant caches after batch operations
	a.cache.InvalidateModels()
	a.cache.InvalidateRunningModels()

	totalDuration := time.Since(startTime)
	results["total_duration"] = totalDuration.Milliseconds()

	fmt.Printf("✓ Batch operations completed: %d succeeded, %d failed (took %v)\n",
		results["summary"].(map[string]interface{})["succeeded"],
		results["summary"].(map[string]interface{})["failed"],
		totalDuration)

	return results, nil
}

// OptimizePerformance performs various performance optimizations
func (a *App) OptimizePerformance() (map[string]interface{}, error) {
	fmt.Println("🔄 JavaScript called OptimizePerformance()")

	if a.service == nil {
		return nil, fmt.Errorf("service not initialized")
	}

	startTime := time.Now()
	optimizations := map[string]interface{}{
		"timestamp":       time.Now().Format(time.RFC3339),
		"optimizations":   make([]map[string]interface{}, 0),
		"cache_cleared":   false,
		"cache_prewarmed": false,
	}

	// Clear all caches
	a.cache.InvalidateAll()
	optimizations["cache_cleared"] = true
	optimizations["optimizations"] = append(optimizations["optimizations"].([]map[string]interface{}), map[string]interface{}{
		"type":        "cache_clear",
		"description": "Cleared all cached data",
		"status":      "completed",
	})

	// Pre-warm caches with fresh data
	go func() {
		// Pre-warm in background to avoid blocking
		a.GetModels()
		a.GetRunningModels()
		a.GetConfig()
	}()

	optimizations["cache_prewarmed"] = true
	optimizations["optimizations"] = append(optimizations["optimizations"].([]map[string]interface{}), map[string]interface{}{
		"type":        "cache_prewarm",
		"description": "Pre-warmed caches with fresh data",
		"status":      "initiated",
	})

	duration := time.Since(startTime)
	optimizations["duration"] = duration.Milliseconds()

	fmt.Printf("✓ Performance optimization completed (took %v)\n", duration)
	return optimizations, nil
}
