# Gollama GUI Diagnostics

This document describes the comprehensive diagnostic and debugging capabilities implemented for the Gollama Wails v3 GUI application.

## Overview

The diagnostic system provides multiple layers of debugging and verification tools to help identify and resolve service binding issues, connectivity problems, and other runtime issues.

## Features

### 1. Runtime Method Discovery and Logging

The JavaScript API automatically discovers and logs available Wails methods at startup:

- **Method Discovery**: Scans multiple binding patterns (`window.wails`, `window`, `window.wails.App`)
- **Runtime Logging**: Comprehensive logging of method availability and binding status
- **Fallback Patterns**: Multiple fallback mechanisms for method access

### 2. Service Health Check Endpoints

Multiple health check endpoints provide detailed status information:

#### Go Service Methods:
- `GetServiceStatus()` - Detailed service health with component-level status
- `GetDiagnosticInfo()` - Comprehensive diagnostic information
- `RunDiagnosticTests()` - Automated test suite with pass/fail results
- `VerifyServiceBinding()` - Service binding verification with recommendations
- `TestServiceBinding()` - Simple connectivity test

#### JavaScript API Methods:
- `getDiagnostics()` - Client-side diagnostic information
- `runConnectivityTests()` - Automated connectivity test suite
- `performMethodDiscovery()` - Runtime method discovery
- `getRuntimeInfo()` - Browser and runtime environment information

### 3. Diagnostic Tools and Verification

#### Web-Based Diagnostics Page
Access via the GUI navigation: **🔍 Diagnostics**

Features:
- **Quick Status Cards**: Real-time status of API, Service, Methods, and Tests
- **Interactive Testing**: Run quick or comprehensive diagnostics
- **Console Output**: Real-time diagnostic output display
- **Export Functionality**: Download diagnostic reports

#### Browser Console Tools
Open Developer Tools (F12) and use these commands:

```javascript
// Quick diagnostics
await runQuickDiagnostics()

// Comprehensive diagnostics
await runFullDiagnostics()

// Test service binding specifically
await testServiceBinding()

// Show help
diagnosticsHelp()

// Start health monitoring (10 second intervals)
gollamaDiagnostics.startHealthMonitoring(10000)

// Export diagnostic data
await gollamaDiagnostics.exportDiagnostics()
```

## Diagnostic Methods

### Go Service Methods

#### `GetServiceStatus() -> *HealthCheckDTO`
Returns detailed health status for all service components:
- Gollama service initialization
- Ollama API connectivity
- Model listing capability
- Running models capability
- Configuration access
- Template loading
- Context availability

#### `GetDiagnosticInfo() -> map[string]interface{}`
Comprehensive diagnostic information including:
- Service configuration
- Method availability
- Service capability tests
- Exposed method list

#### `RunDiagnosticTests() -> map[string]interface{}`
Automated test suite covering:
- Service initialization
- Context availability
- Template loading
- Health check
- Model listing
- Running models
- Configuration access

#### `VerifyServiceBinding() -> map[string]interface{}`
Service binding verification with:
- Binding status verification
- Critical method testing
- Failure recommendations
- Overall health assessment

### JavaScript API Methods

#### `getDiagnostics()`
Client-side diagnostic information including:
- Runtime information (browser, platform, window)
- Method discovery results
- Connectivity test results
- Service status information

#### `runConnectivityTests()`
Automated connectivity tests for:
- TestServiceBinding
- HealthCheck
- GetServiceStatus
- GetSystemInfo
- GetConfig
- GetModels

#### `performMethodDiscovery()`
Runtime method discovery providing:
- Expected vs found methods
- Missing method identification
- Unexpected method detection
- Binding pattern analysis

## Usage Examples

### Quick Health Check
```javascript
// Check if everything is working
const status = await window.gollamaAPI.getServiceStatus();
console.log('Service Status:', status.status);
```

### Comprehensive Diagnostics
```javascript
// Run full diagnostic suite
const diagnostics = await window.gollamaAPI.getDiagnostics();
const tests = await window.gollamaAPI.runDiagnosticTests();
const verification = await window.gollamaAPI.verifyServiceBinding();

console.log('Full Diagnostics:', { diagnostics, tests, verification });
```

### Continuous Health Monitoring
```javascript
// Monitor service health every 30 seconds
window.gollamaAPI.startHealthMonitoring(30000);

// Listen for health updates
window.addEventListener('gollamaHealthUpdate', (event) => {
    console.log('Health Update:', event.detail);
});
```

## Troubleshooting

### Common Issues and Solutions

#### Service Not Initialized
**Symptoms**: `GetServiceStatus()` returns "service not initialized"
**Solutions**:
- Check OnStartup method execution
- Verify service configuration
- Check for initialization errors in logs

#### Method Not Available
**Symptoms**: "Method not found in Wails runtime"
**Solutions**:
- Run method discovery: `window.gollamaAPI.performMethodDiscovery()`
- Check binding patterns
- Verify Wails v3 service registration

#### Connectivity Issues
**Symptoms**: Health check failures, API timeouts
**Solutions**:
- Verify Ollama is running on configured URL
- Check network connectivity
- Review service configuration

### Diagnostic Report Export

Export comprehensive diagnostic data for troubleshooting:

```javascript
// Export via UI
await gollamaDiagnostics.exportDiagnostics()

// Or programmatically
const report = await window.gollamaAPI.createDiagnosticReport();
console.log(report.readableSummary);
```

## Implementation Details

### Architecture
- **Go Service Layer**: Comprehensive health checks and diagnostic methods
- **JavaScript API Layer**: Client-side diagnostics and method discovery
- **Web UI**: Interactive diagnostic interface
- **Console Tools**: Developer-friendly debugging commands

### Error Handling
- Graceful degradation when methods are unavailable
- Detailed error messages with context
- Automatic retry mechanisms for transient failures
- User-friendly error notifications

### Performance
- Asynchronous diagnostic operations
- Configurable health monitoring intervals
- Efficient method discovery caching
- Minimal impact on application performance

## Requirements Satisfied

This implementation satisfies the following requirements from the specification:

- **3.1**: Clear error messages and logging for debugging service binding issues
- **3.2**: JavaScript error handling with detailed problem identification
- **3.3**: Initialization status logging and request/response logging for troubleshooting
- **3.4**: Comprehensive diagnostic tools to verify service binding functionality

The diagnostic system provides developers with all necessary tools to identify, debug, and resolve service binding and connectivity issues in the Gollama Wails v3 GUI application.
