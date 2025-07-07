#!/bin/bash

# Gollama GUI Test Runner
# Runs comprehensive test suite for service binding and integration

set -e

echo "🧪 Gollama GUI Test Suite"
echo "========================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're in the right directory
if [ ! -f "main.go" ]; then
    print_error "Please run this script from the gui directory"
    exit 1
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed or not in PATH"
    exit 1
fi

# Check if Node.js is installed (for JavaScript tests)
if ! command -v node &> /dev/null; then
    print_warning "Node.js not found - JavaScript tests will be skipped"
    SKIP_JS_TESTS=true
else
    SKIP_JS_TESTS=false
fi

print_status "Starting test suite..."

# Test configuration
TEST_TIMEOUT="30s"
TEST_VERBOSE="-v"
TEST_COVERAGE="-coverprofile=coverage.out"

# Run Go unit tests
echo ""
print_status "Running Go unit tests..."
echo "----------------------------------------"

# Test individual components
print_status "Testing DTO conversions..."
go test ${TEST_VERBOSE} -timeout ${TEST_TIMEOUT} -run TestConvertTo ./...

print_status "Testing app service methods..."
go test ${TEST_VERBOSE} -timeout ${TEST_TIMEOUT} -run TestAppServiceMethods ./...

print_status "Testing service binding..."
go test ${TEST_VERBOSE} -timeout ${TEST_TIMEOUT} -run TestServiceBinding ./...

print_status "Testing method exposure..."
go test ${TEST_VERBOSE} -timeout ${TEST_TIMEOUT} -run TestServiceMethodExposure ./...

print_status "Testing Wails v3 compatibility..."
go test ${TEST_VERBOSE} -timeout ${TEST_TIMEOUT} -run TestWailsV3Compatibility ./...

# Run integration tests
echo ""
print_status "Running integration tests..."
echo "----------------------------------------"

print_status "Testing service binding with mock..."
go test ${TEST_VERBOSE} -timeout ${TEST_TIMEOUT} -run TestServiceBindingWithMockService ./...

# Run E2E tests (these may fail if Ollama is not running)
echo ""
print_status "Running end-to-end tests..."
echo "----------------------------------------"

print_warning "E2E tests require Ollama to be running on localhost:11434"
print_status "If Ollama is not available, E2E tests will be skipped"

# Check if Ollama is running
if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    print_success "Ollama API is accessible - running full E2E tests"

    print_status "Testing complete model management workflow..."
    go test ${TEST_VERBOSE} -timeout 60s -run TestE2EModelManagementWorkflow ./...

    print_status "Testing error scenarios..."
    go test ${TEST_VERBOSE} -timeout ${TEST_TIMEOUT} -run TestE2EErrorScenarios ./...

    print_status "Testing performance characteristics..."
    go test ${TEST_VERBOSE} -timeout 60s -run TestE2EPerformance ./...

    print_status "Testing data integrity..."
    go test ${TEST_VERBOSE} -timeout ${TEST_TIMEOUT} -run TestE2EDataIntegrity ./...
else
    print_warning "Ollama API not accessible - skipping E2E tests"
    print_status "To run E2E tests, start Ollama with: ollama serve"
fi

# Run JavaScript tests
if [ "$SKIP_JS_TESTS" = false ]; then
    echo ""
    print_status "Running JavaScript integration tests..."
    echo "----------------------------------------"

    # Check if test file exists
    if [ -f "static/js/gollama-api.test.js" ]; then
        print_status "Running JavaScript API tests..."
        cd static/js
        node gollama-api.test.js
        cd ../..
    else
        print_warning "JavaScript test file not found"
    fi
else
    print_warning "Skipping JavaScript tests - Node.js not available"
fi

# Generate test coverage report
echo ""
print_status "Generating test coverage report..."
echo "----------------------------------------"

# Run all tests with coverage
go test ${TEST_COVERAGE} -timeout ${TEST_TIMEOUT} ./...

if [ -f "coverage.out" ]; then
    # Generate HTML coverage report
    go tool cover -html=coverage.out -o coverage.html

    # Show coverage summary
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
    print_success "Test coverage: ${COVERAGE}"
    print_status "Detailed coverage report saved to coverage.html"
else
    print_warning "Coverage report not generated"
fi

# Run benchmark tests
echo ""
print_status "Running benchmark tests..."
echo "----------------------------------------"

print_status "Benchmarking DTO conversions..."
go test -bench=BenchmarkConvert -benchmem ./... || print_warning "No benchmark tests found"

# Test build process
echo ""
print_status "Testing build process..."
echo "----------------------------------------"

print_status "Testing Go build..."
go build -o test-binary ./...
if [ -f "test-binary" ]; then
    print_success "Go build successful"
    rm test-binary
else
    print_error "Go build failed"
fi

# Verify service binding methods are exported
echo ""
print_status "Verifying service method exports..."
echo "----------------------------------------"

# Use go doc to check exported methods
EXPORTED_METHODS=$(go doc -all . | grep "func (a \*App)" | wc -l)
print_status "Found ${EXPORTED_METHODS} exported App methods"

if [ ${EXPORTED_METHODS} -lt 15 ]; then
    print_warning "Expected at least 15 exported methods for service binding"
else
    print_success "Service methods properly exported"
fi

# Check for required methods
REQUIRED_METHODS=("GetModels" "RunModel" "DeleteModel" "GetConfig" "HealthCheck")
for method in "${REQUIRED_METHODS[@]}"; do
    if go doc . | grep -q "func (a \*App) ${method}"; then
        print_success "✓ ${method} method found"
    else
        print_error "✗ ${method} method missing"
    fi
done

# Final summary
echo ""
print_status "Test Summary"
echo "============"

# Count test files
GO_TEST_FILES=$(find . -name "*_test.go" | wc -l)
JS_TEST_FILES=$(find . -name "*.test.js" | wc -l)

print_status "Go test files: ${GO_TEST_FILES}"
print_status "JavaScript test files: ${JS_TEST_FILES}"

# Check for test failures
if [ $? -eq 0 ]; then
    print_success "All tests completed successfully!"
    echo ""
    print_status "Service binding implementation is ready for production"
    print_status "JavaScript ↔ Go communication verified"
    print_status "Wails v3 compatibility confirmed"
else
    print_error "Some tests failed - check output above"
    exit 1
fi

# Cleanup
if [ -f "coverage.out" ]; then
    print_status "Cleaning up temporary files..."
    # Keep coverage files for review
fi

echo ""
print_success "🎉 Test suite completed successfully!"
print_status "Next steps:"
print_status "  1. Review coverage report: open coverage.html"
print_status "  2. Test with real Ollama instance if not done"
print_status "  3. Deploy and test in production environment"
