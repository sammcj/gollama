# Design Document

## Overview

This design addresses the critical service binding issue in the Gollama Wails v3 GUI where JavaScript frontend cannot access Go service methods. The issue stems from incorrect service registration and method exposure patterns in Wails v3 alpha.9. This design provides multiple solution approaches to establish proper communication between the frontend and backend service layer.

## Architecture

### Current Problem Analysis

The current implementation has the following issues:
1. **Service Registration**: Go service methods are not properly exposed to the JavaScript runtime
2. **Method Binding**: Wails v3 alpha.9 uses different binding mechanisms than expected
3. **Context Handling**: Service context and lifecycle management may be incorrect
4. **API Surface**: The JavaScript API surface is not properly established

### Target Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    JavaScript Frontend                      │
│  ┌─────────────────┐    ┌─────────────────────────────────┐ │
│  │   HTMX Client   │    │     JavaScript API Layer       │ │
│  │                 │    │  - window.wails.GetModels()    │ │
│  │ - Model List    │◄──►│  - window.wails.DeleteModel()  │ │
│  │ - Actions       │    │  - window.wails.RunModel()     │ │
│  │ - Settings      │    │  - Error Handling              │ │
│  └─────────────────┘    └─────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                                    │
                                    │ Wails v3 Bridge
                                    ▼
┌─────────────────────────────────────────────────────────────┐
│                    Wails v3 Runtime                        │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              Service Registration                       │ │
│  │  - application.NewService(service)                     │ │
│  │  - Method Exposure Configuration                       │ │
│  │  - Context Management                                  │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                                    │
                                    │ Direct Method Calls
                                    ▼
┌─────────────────────────────────────────────────────────────┐
│                   Go Service Layer                         │
│  ┌─────────────────┐    ┌─────────────────────────────────┐ │
│  │ GollamaService  │    │        Core Operations          │ │
│  │                 │    │                                 │ │
│  │ - GetModels()   │◄──►│  - Ollama API Integration       │ │
│  │ - DeleteModel() │    │  - Model Management             │ │
│  │ - RunModel()    │    │  - Configuration Handling       │ │
│  │ - GetConfig()   │    │  - Event System                 │ │
│  └─────────────────┘    └─────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. Service Interface Definition

```go
// Service interface that will be exposed to JavaScript
type GUIService interface {
    // Model Management
    GetModels() ([]ModelDTO, error)
    GetModel(id string) (*ModelDTO, error)
    DeleteModel(id string) error
    RunModel(id string) error
    UnloadModel(id string) error
    CopyModel(source, dest string) error
    PushModel(id string) error
    PullModel(id string) error

    // Model Information
    GetModelInfo(id string) (*ModelInfoDTO, error)
    GetRunningModels() ([]RunningModelDTO, error)

    // Configuration
    GetConfig() (*ConfigDTO, error)
    UpdateConfig(config *ConfigDTO) error

    // vRAM Estimation
    EstimateVRAM(model string, constraints VRAMConstraintsDTO) (*VRAMEstimationDTO, error)

    // Health Check
    HealthCheck() error
}
```

### 2. Wails v3 Service Registration

```go
// Updated app.go structure
type App struct {
    ctx     context.Context
    service *core.GollamaService
}

func NewApp() *App {
    return &App{}
}

func (a *App) OnStartup(ctx context.Context) {
    a.ctx = ctx

    // Initialise service
    serviceConfig := core.ServiceConfig{
        OllamaAPIURL: "http://localhost:11434",
        Context:      ctx,
    }

    var err error
    a.service, err = core.NewGollamaService(serviceConfig)
    if err != nil {
        log.Printf("Failed to initialise service: %v", err)
        return
    }

    log.Println("Gollama GUI service initialized successfully")
}

// Exposed methods for JavaScript
func (a *App) GetModels() ([]ModelDTO, error) {
    models, err := a.service.ListModels()
    if err != nil {
        return nil, err
    }
    return convertToModelDTOs(models), nil
}

func (a *App) DeleteModel(id string) error {
    return a.service.DeleteModel(id)
}

func (a *App) RunModel(id string) error {
    return a.service.RunModel(id)
}

// ... additional method implementations
```

### 3. JavaScript API Layer

```javascript
// Enhanced JavaScript API with proper error handling
class GollamaAPI {
    constructor() {
        this.initialized = false;
        this.checkInitialization();
    }

    checkInitialization() {
        // Check if Wails runtime is available
        if (typeof window.wails === 'undefined') {
            console.error('Wails runtime not available');
            return false;
        }

        // Log available methods for debugging
        console.log('Available Wails methods:', Object.keys(window.wails));
        this.initialized = true;
        return true;
    }

    async getModels() {
        if (!this.initialized) {
            throw new Error('Wails runtime not initialized');
        }

        try {
            // Try multiple possible API patterns
            if (window.wails.GetModels) {
                return await window.wails.GetModels();
            } else if (window.GetModels) {
                return await window.GetModels();
            } else if (window.wails.App && window.wails.App.GetModels) {
                return await window.wails.App.GetModels();
            } else {
                throw new Error('GetModels method not found in Wails runtime');
            }
        } catch (error) {
            console.error('Error calling GetModels:', error);
            throw error;
        }
    }

    async deleteModel(id) {
        if (!this.initialized) {
            throw new Error('Wails runtime not initialized');
        }

        try {
            if (window.wails.DeleteModel) {
                return await window.wails.DeleteModel(id);
            } else if (window.DeleteModel) {
                return await window.DeleteModel(id);
            } else if (window.wails.App && window.wails.App.DeleteModel) {
                return await window.wails.App.DeleteModel(id);
            } else {
                throw new Error('DeleteModel method not found in Wails runtime');
            }
        } catch (error) {
            console.error('Error calling DeleteModel:', error);
            throw error;
        }
    }

    // ... additional API methods
}

// Global API instance
window.gollamaAPI = new GollamaAPI();
```

## Data Models

### 1. Data Transfer Objects (DTOs)

```go
// DTOs for JavaScript communication
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
}

type RunningModelDTO struct {
    Name      string    `json:"name"`
    Size      int64     `json:"size"`
    LoadedAt  time.Time `json:"loaded_at"`
    ExpiresAt time.Time `json:"expires_at"`
}

type ConfigDTO struct {
    OllamaAPIURL     string `json:"ollama_api_url"`
    LogLevel         string `json:"log_level"`
    AutoRefresh      bool   `json:"auto_refresh"`
    RefreshInterval  int    `json:"refresh_interval"`
    WindowWidth      int    `json:"window_width"`
    WindowHeight     int    `json:"window_height"`
    DefaultView      string `json:"default_view"`
    ShowSystemTray   bool   `json:"show_system_tray"`
}

type VRAMConstraintsDTO struct {
    AvailableVRAM int    `json:"available_vram"`
    Context       int    `json:"context"`
    Quantization  string `json:"quantization"`
}

type VRAMEstimationDTO struct {
    ModelName        string  `json:"model_name"`
    RequiredVRAM     float64 `json:"required_vram"`
    AvailableVRAM    float64 `json:"available_vram"`
    CanRun           bool    `json:"can_run"`
    RecommendedQuant string  `json:"recommended_quant"`
    Details          string  `json:"details"`
}
```

### 2. Conversion Functions

```go
// Convert core models to DTOs
func convertToModelDTOs(models []core.Model) []ModelDTO {
    dtos := make([]ModelDTO, len(models))
    for i, model := range models {
        dtos[i] = ModelDTO{
            ID:                model.ID,
            Name:              model.Name,
            Size:              model.Size,
            SizeFormatted:     formatSize(model.Size),
            Family:            model.Family,
            ParameterSize:     model.ParameterSize,
            QuantizationLevel: model.QuantizationLevel,
            Modified:          model.Modified,
            ModifiedFormatted: model.Modified.Format("2006-01-02 15:04:05"),
            IsRunning:         model.IsRunning,
            Digest:            model.Digest,
        }
    }
    return dtos
}

func convertToConfigDTO(config *config.Config) *ConfigDTO {
    return &ConfigDTO{
        OllamaAPIURL:    config.OllamaAPIURL,
        LogLevel:        config.LogLevel,
        AutoRefresh:     config.AutoRefresh,
        RefreshInterval: config.RefreshInterval,
        WindowWidth:     config.WindowWidth,
        WindowHeight:    config.WindowHeight,
        DefaultView:     config.DefaultView,
        ShowSystemTray:  config.ShowSystemTray,
    }
}
```

## Error Handling

### 1. Go Error Handling

```go
// Standardized error responses
type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

func (a *App) handleError(err error, operation string) error {
    log.Printf("Error in %s: %v", operation, err)

    // Return structured error for JavaScript
    return &APIError{
        Code:    "OPERATION_FAILED",
        Message: fmt.Sprintf("Failed to %s", operation),
        Details: err.Error(),
    }
}

func (a *App) GetModels() ([]ModelDTO, error) {
    models, err := a.service.ListModels()
    if err != nil {
        return nil, a.handleError(err, "get models")
    }
    return convertToModelDTOs(models), nil
}
```

### 2. JavaScript Error Handling

```javascript
// Enhanced error handling in JavaScript
class GollamaAPI {
    async handleAPICall(methodName, apiCall) {
        try {
            const result = await apiCall();
            console.log(`${methodName} successful:`, result);
            return result;
        } catch (error) {
            console.error(`${methodName} failed:`, error);

            // Show user-friendly error message
            this.showError(`Failed to ${methodName.toLowerCase()}: ${error.message}`);
            throw error;
        }
    }

    showError(message) {
        // Display toast notification or modal
        const toast = document.createElement('div');
        toast.className = 'fixed top-4 right-4 bg-red-500 text-white px-4 py-2 rounded-lg shadow-lg z-50';
        toast.textContent = message;
        document.body.appendChild(toast);

        setTimeout(() => {
            toast.remove();
        }, 5000);
    }

    async getModels() {
        return this.handleAPICall('GetModels', async () => {
            if (window.wails && window.wails.GetModels) {
                return await window.wails.GetModels();
            }
            throw new Error('GetModels method not available');
        });
    }
}
```

## Testing Strategy

### 1. Service Binding Tests

```go
// Test service registration and method exposure
func TestServiceBinding(t *testing.T) {
    app := NewApp()
    ctx := context.Background()

    // Test service initialization
    app.OnStartup(ctx)
    assert.NotNil(t, app.service)

    // Test method availability
    models, err := app.GetModels()
    assert.NoError(t, err)
    assert.NotNil(t, models)
}

func TestMethodExposure(t *testing.T) {
    app := NewApp()
    ctx := context.Background()
    app.OnStartup(ctx)

    // Test all exposed methods
    testCases := []struct {
        name string
        test func() error
    }{
        {"GetModels", func() error { _, err := app.GetModels(); return err }},
        {"GetConfig", func() error { _, err := app.GetConfig(); return err }},
        {"HealthCheck", func() error { return app.HealthCheck() }},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            err := tc.test()
            assert.NoError(t, err)
        })
    }
}
```

### 2. JavaScript Integration Tests

```javascript
// Test JavaScript API integration
describe('Gollama API Integration', () => {
    beforeEach(() => {
        // Mock Wails runtime for testing
        window.wails = {
            GetModels: jest.fn().mockResolvedValue([]),
            DeleteModel: jest.fn().mockResolvedValue(true),
            RunModel: jest.fn().mockResolvedValue(true),
        };
    });

    test('should initialize API correctly', () => {
        const api = new GollamaAPI();
        expect(api.initialized).toBe(true);
    });

    test('should call GetModels successfully', async () => {
        const api = new GollamaAPI();
        const models = await api.getModels();
        expect(window.wails.GetModels).toHaveBeenCalled();
        expect(models).toEqual([]);
    });

    test('should handle API errors gracefully', async () => {
        window.wails.GetModels = jest.fn().mockRejectedValue(new Error('API Error'));

        const api = new GollamaAPI();
        await expect(api.getModels()).rejects.toThrow('API Error');
    });
});
```

### 3. End-to-End Testing

```go
// E2E test for complete service binding flow
func TestE2EServiceBinding(t *testing.T) {
    // Start application
    app := NewApp()
    ctx := context.Background()
    app.OnStartup(ctx)

    // Test complete workflow
    t.Run("Complete Model Management Flow", func(t *testing.T) {
        // Get models
        models, err := app.GetModels()
        assert.NoError(t, err)

        if len(models) > 0 {
            modelID := models[0].ID

            // Get model details
            model, err := app.GetModel(modelID)
            assert.NoError(t, err)
            assert.Equal(t, modelID, model.ID)

            // Test model operations (if safe)
            if !model.IsRunning {
                err = app.RunModel(modelID)
                assert.NoError(t, err)
            }
        }
    })
}
```

## Implementation Approach

### Phase 1: Service Registration Fix
1. **Update Wails v3 Service Registration**: Modify `main.go` and `app.go` to properly register the service
2. **Method Exposure**: Ensure all required methods are exposed to JavaScript
3. **Context Management**: Fix service context and lifecycle management
4. **Logging Enhancement**: Add comprehensive logging for debugging

### Phase 2: JavaScript API Layer
1. **API Wrapper**: Create robust JavaScript API wrapper with multiple fallback patterns
2. **Error Handling**: Implement comprehensive error handling and user feedback
3. **Method Discovery**: Add runtime method discovery and logging
4. **Testing Framework**: Set up JavaScript testing for API integration

### Phase 3: Integration Testing
1. **Service Binding Tests**: Verify Go service methods are accessible
2. **JavaScript Integration**: Test JavaScript API calls work correctly
3. **E2E Validation**: Complete end-to-end testing of GUI functionality
4. **Performance Testing**: Ensure service calls perform adequately

### Phase 4: Documentation and Optimization
1. **API Documentation**: Document the service binding approach
2. **Performance Optimization**: Optimize service call performance
3. **Error Message Improvement**: Enhance error messages for better debugging
4. **Best Practices**: Document Wails v3 service binding best practices

This design provides multiple solution approaches to resolve the service binding issue, with comprehensive error handling, testing strategies, and a clear implementation path to restore GUI functionality.