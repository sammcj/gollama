# Gollama GUI Testing Documentation

This document describes the comprehensive testing suite for the Gollama GUI service binding implementation.

## Overview

The testing suite validates the Wails v3 service binding fix through multiple layers:

1. **Unit Tests** - Test individual components and methods
2. **Integration Tests** - Test JavaScript ↔ Go communication
3. **End-to-End Tests** - Test complete workflows
4. **Performance Tests** - Validate response times and concurrency

## Test Structure

```
gui/
├── *_test.go              # Go unit tests
├── service_binding_test.go # Service binding specific tests
├── e2e_test.go            # End-to-end workflow tests
├── static/js/
│   └── gollama-api.test.js # JavaScript integration tests
├── run_tests.sh           # Test runner script
└── TESTING.md             # This documentation
```

## Running Tests

### Quick Test Run

```bash
cd gui
./run_tests.sh
```

### Individual Test Categories

#### Unit Tests
```bash
# Test DTO conversions
go test -v -run TestConvertTo

# Test app service methods
go test -v -run TestAppServiceMethods

# Test service binding
go test -v -run TestServiceBinding
```

#### Integration Tests
```bash
# Test service binding with mock
go test -v -run TestServiceBindingWithMockService

# Test JavaScript API
cd static/js && node gollama-api.test.js
```

#### End-to-End Tests
```bash
# Requires Ollama running on localhost:11434
go test -v -run TestE2E
```

## Test Categories

### 1. Service Binding Tests (`service_binding_test.go`)

**Purpose**: Verify that Go service methods are properly exposed to JavaScript

**Key Tests**:
- `TestServiceBinding` - Complete service binding functionality
- `TestServiceMethodExposure` - Method exposure for Wails v3
- `TestWailsV3Compatibility` - Wails v3 specific features
- `TestServiceBindingWithMockService` - Mock service integration

**What it validates**:
- All required methods exist with correct signatures
- Methods handle errors gracefully when service is unavailable
- Wails v3 context management works correctly
- Embedded assets are properly loaded

### 2. JavaScript Integration Tests (`gollama-api.test.js`)

**Purpose**: Test the JavaScript API layer with mocked Wails runtime

**Key Features**:
- Mock Wails runtime implementation
- Comprehensive API method testing
- Error handling validation
- User feedback mechanisms

**Test Coverage**:
- API initialization and method discovery
- Service connectivity testing
- Model management operations
- Configuration management
- Error scenarios and validation
- Diagnostic capabilities

### 3. End-to-End Tests (`e2e_test.go`)

**Purpose**: Test complete workflows with real Ollama integration

**Test Scenarios**:
- Complete model management workflow
- Error handling scenarios
- Performance characteristics
- Data integrity validation

**Requirements**:
- Ollama running on `localhost:11434`
- At least one model available for testing

### 4. Unit Tests (`app_test.go`, `dto_test.go`)

**Purpose**: Test individual components and data transformations

**Coverage**:
- DTO conversion functions
- Validation logic
- Utility functions
- Error handling

## Test Requirements

### Prerequisites

1. **Go 1.21+** - For running Go tests
2. **Node.js** - For JavaScript tests (optional)
3. **Ollama** - For E2E tests (optional)

### Environment Setup

```bash
# Install dependencies
go mod tidy

# Start Ollama (for E2E tests)
ollama serve

# Pull a test model (optional)
ollama pull llama2:7b
```

## Test Scenarios

### Service Binding Validation

The tests verify that:

1. **Method Exposure**: All required methods are exposed to JavaScript
2. **Error Handling**: Proper error responses when service unavailable
3. **Data Conversion**: DTOs are correctly converted between Go and JavaScript
4. **Context Management**: Wails v3 context is properly managed

### JavaScript API Testing

The JavaScript tests validate:

1. **API Initialization**: Proper discovery of Wails methods
2. **Method Calling**: All service methods can be called from JavaScript
3. **Error Handling**: Graceful handling of service errors
4. **User Feedback**: Toast notifications and error messages

### End-to-End Workflows

E2E tests cover:

1. **Model Listing**: Get and display all available models
2. **Model Operations**: Run, delete, unload models
3. **Configuration**: Get and update application settings
4. **vRAM Estimation**: Calculate memory requirements
5. **Health Monitoring**: Service status and diagnostics

## Expected Test Results

### Successful Test Run

```
🧪 Gollama GUI Test Suite
=========================
[INFO] Running Go unit tests...
✅ TestServiceBinding PASSED
✅ TestServiceMethodExposure PASSED
✅ TestWailsV3Compatibility PASSED

[INFO] Running integration tests...
✅ TestServiceBindingWithMockService PASSED

[INFO] Running JavaScript integration tests...
✅ API Initialization PASSED
✅ Method Discovery PASSED
✅ Service Connectivity PASSED
... (all JavaScript tests)

[INFO] Running end-to-end tests...
✅ Complete Model Management Workflow PASSED
✅ Error Scenarios PASSED
✅ Performance Tests PASSED

[SUCCESS] All tests completed successfully!
Test coverage: 85.2%
```

### Test Failure Scenarios

Common failure scenarios and solutions:

1. **Service Not Available**
   ```
   [WARNING] Ollama API not accessible - skipping E2E tests
   ```
   **Solution**: Start Ollama with `ollama serve`

2. **Method Not Found**
   ```
   [ERROR] Method 'GetModels' not available
   ```
   **Solution**: Check service binding implementation

3. **JavaScript Runtime Error**
   ```
   [ERROR] Wails runtime not available
   ```
   **Solution**: Verify Wails v3 bindings are generated

## Coverage Goals

- **Unit Tests**: >90% coverage of service methods
- **Integration Tests**: All JavaScript API methods tested
- **E2E Tests**: Complete user workflows validated
- **Error Handling**: All error scenarios covered

## Continuous Integration

### GitHub Actions Integration

```yaml
name: GUI Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      - name: Run tests
        run: cd gui && ./run_tests.sh
```

### Pre-commit Hooks

```bash
# Install pre-commit hook
echo "cd gui && go test ./..." > .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

## Debugging Test Failures

### Common Issues

1. **Service Initialization Failures**
   - Check Ollama is running
   - Verify API URL configuration
   - Check network connectivity

2. **Method Binding Issues**
   - Verify method signatures match
   - Check Wails v3 binding generation
   - Validate context management

3. **JavaScript Integration Problems**
   - Check browser console for errors
   - Verify Wails runtime is loaded
   - Test method availability

### Debug Commands

```bash
# Run specific test with verbose output
go test -v -run TestServiceBinding

# Run with race detection
go test -race ./...

# Generate detailed coverage
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -html=coverage.out

# Debug JavaScript tests
cd static/js && node --inspect gollama-api.test.js
```

## Performance Benchmarks

### Expected Performance

- **GetModels**: < 1s for 100 models
- **GetRunningModels**: < 500ms
- **HealthCheck**: < 100ms
- **Configuration**: < 50ms

### Benchmark Tests

```bash
# Run performance benchmarks
go test -bench=. -benchmem ./...
```

## Security Considerations

### Test Data

- Tests use mock data where possible
- No sensitive information in test files
- Temporary files cleaned up after tests

### Network Security

- E2E tests only connect to localhost
- No external API calls in tests
- Mock services for external dependencies

## Contributing

### Adding New Tests

1. **Unit Tests**: Add to appropriate `*_test.go` file
2. **Integration Tests**: Extend JavaScript test suite
3. **E2E Tests**: Add to `e2e_test.go`
4. **Update Documentation**: Update this file

### Test Guidelines

- Use descriptive test names
- Include both positive and negative test cases
- Mock external dependencies
- Clean up resources after tests
- Document complex test scenarios

## Troubleshooting

### Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| `service not initialized` | Ollama not running | Start Ollama service |
| `Method not found` | Binding issue | Check method exposure |
| `Context deadline exceeded` | Slow response | Increase timeout |
| `Connection refused` | Network issue | Check API URL |

### Getting Help

1. Check test output for specific error messages
2. Review this documentation
3. Check Ollama service status
4. Verify Wails v3 configuration
5. Create issue with test output

---

This testing suite ensures the Gollama GUI service binding is robust, reliable, and ready for production use.
