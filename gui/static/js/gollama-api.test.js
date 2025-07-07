/**
 * JavaScript Integration Tests for Gollama API
 * Tests the JavaScript API layer with mocked Wails runtime
 */

// Mock Wails runtime for testing
class MockWailsRuntime {
    constructor() {
        this.methods = {};
        this.callLog = [];
        this.shouldFail = {};
        this.responses = {};
    }

    // Register a method
    registerMethod(name, implementation) {
        this.methods[name] = implementation;
    }

    // Set method to fail
    setMethodFailure(name, error) {
        this.shouldFail[name] = error;
    }

    // Set method response
    setMethodResponse(name, response) {
        this.responses[name] = response;
    }

    // Call a method
    async callMethod(name, ...args) {
        this.callLog.push({ method: name, args, timestamp: Date.now() });

        if (this.shouldFail[name]) {
            throw new Error(this.shouldFail[name]);
        }

        if (this.responses[name] !== undefined) {
            return this.responses[name];
        }

        if (this.methods[name]) {
            return await this.methods[name](...args);
        }

        throw new Error(`Method ${name} not found`);
    }

    // Get call log
    getCallLog() {
        return this.callLog;
    }

    // Clear call log
    clearCallLog() {
        this.callLog = [];
    }
}

// Test Suite
class GollamaAPITestSuite {
    constructor() {
        this.mockRuntime = new MockWailsRuntime();
        this.setupMockRuntime();
        this.testResults = [];
    }

    setupMockRuntime() {
        // Mock successful responses
        this.mockRuntime.setMethodResponse('GetModels', [
            {
                id: 'test-model-1',
                name: 'test-model-1',
                size: 1073741824,
                size_formatted: '1.0 GB',
                family: 'llama',
                parameter_size: '7B',
                quantization_level: 'Q4_0',
                modified: '2024-01-01T00:00:00Z',
                modified_formatted: '2024-01-01 00:00:00',
                is_running: false,
                digest: 'sha256:abc123',
                status: 'available'
            },
            {
                id: 'test-model-2',
                name: 'test-model-2',
                size: 2147483648,
                size_formatted: '2.0 GB',
                family: 'llama',
                parameter_size: '13B',
                quantization_level: 'Q4_0',
                modified: '2024-01-02T00:00:00Z',
                modified_formatted: '2024-01-02 00:00:00',
                is_running: true,
                digest: 'sha256:def456',
                status: 'running'
            }
        ]);

        this.mockRuntime.setMethodResponse('GetRunningModels', [
            {
                name: 'test-model-2',
                size: 2147483648,
                loaded_at: '2024-01-02T10:00:00Z',
                expires_at: '2024-01-02T11:00:00Z'
            }
        ]);

        this.mockRuntime.setMethodResponse('GetConfig', {
            ollama_api_url: 'http://localhost:11434',
            log_level: 'info',
            auto_refresh: true,
            refresh_interval: 30,
            window_width: 1200,
            window_height: 800,
            default_view: 'models',
            show_system_tray: false,
            theme: 'dark'
        });

        this.mockRuntime.setMethodResponse('HealthCheck', null);
        this.mockRuntime.setMethodResponse('TestServiceBinding', 'SUCCESS: Service binding fully operational!');

        this.mockRuntime.setMethodResponse('GetServiceStatus', {
            status: 'healthy',
            timestamp: '2024-01-01T00:00:00Z',
            version: '1.0.0',
            services: {
                gollama_service: { status: 'healthy', message: 'Service initialized' },
                ollama_api: { status: 'healthy', message: 'API accessible' },
                model_listing: { status: 'healthy', message: '2 models available' }
            }
        });

        // Mock method implementations
        this.mockRuntime.registerMethod('RunModel', async (name) => {
            if (!name) throw new Error('Model name is required');
            return null;
        });

        this.mockRuntime.registerMethod('DeleteModel', async (name) => {
            if (!name) throw new Error('Model name is required');
            return null;
        });

        this.mockRuntime.registerMethod('UnloadModel', async (name) => {
            if (!name) throw new Error('Model name is required');
            return null;
        });
    }

    // Mock window.wails for testing
    setupMockWindow() {
        global.window = {
            wails: {}
        };

        // Add all methods to window.wails
        const methods = [
            'GetModels', 'GetModel', 'GetRunningModels', 'RunModel', 'DeleteModel',
            'UnloadModel', 'CopyModel', 'PushModel', 'PullModel', 'GetModelDetails',
            'EstimateVRAM', 'GetConfig', 'UpdateConfig', 'HealthCheck',
            'TestServiceBinding', 'GetServiceStatus'
        ];

        methods.forEach(method => {
            global.window.wails[method] = (...args) => this.mockRuntime.callMethod(method, ...args);
        });

        // Mock document for toast notifications
        global.document = {
            body: {
                appendChild: () => {},
                removeChild: () => {}
            },
            createElement: () => ({
                className: '',
                innerHTML: '',
                remove: () => {}
            })
        };

        // Mock setTimeout
        global.setTimeout = (fn, delay) => {
            // Execute immediately for testing
            fn();
        };
    }

    async runTest(name, testFn) {
        try {
            console.log(`Running test: ${name}`);
            await testFn();
            this.testResults.push({ name, status: 'PASS', error: null });
            console.log(`✅ ${name} PASSED`);
        } catch (error) {
            this.testResults.push({ name, status: 'FAIL', error: error.message });
            console.error(`❌ ${name} FAILED:`, error.message);
        }
    }

    async runAllTests() {
        console.log('🧪 Starting Gollama API JavaScript Integration Tests');

        this.setupMockWindow();

        // Import and create API instance
        const GollamaAPI = require('./gollama-api.js');
        const api = new GollamaAPI();

        // Wait for initialization
        await new Promise(resolve => setTimeout(resolve, 100));

        // Test API Initialization
        await this.runTest('API Initialization', async () => {
            if (!api.initialized) {
                throw new Error('API should be initialized');
            }
        });

        // Test Method Discovery
        await this.runTest('Method Discovery', async () => {
            const expectedMethods = [
                'GetModels', 'GetRunningModels', 'RunModel', 'DeleteModel',
                'UnloadModel', 'GetConfig', 'HealthCheck', 'TestServiceBinding'
            ];

            for (const method of expectedMethods) {
                if (!api.bindingMethods[method]) {
                    throw new Error(`Method ${method} not discovered`);
                }
            }
        });

        // Test Service Connectivity
        await this.runTest('Service Connectivity', async () => {
            const result = await api.testServiceBinding();
            if (!result.includes('SUCCESS')) {
                throw new Error('Service binding test failed');
            }
        });

        // Test Health Check
        await this.runTest('Health Check', async () => {
            await api.healthCheck();
            // Should not throw
        });

        // Test Get Models
        await this.runTest('Get Models', async () => {
            const models = await api.getModels();
            if (!Array.isArray(models)) {
                throw new Error('Models should be an array');
            }
            if (models.length !== 2) {
                throw new Error(`Expected 2 models, got ${models.length}`);
            }
            if (models[0].name !== 'test-model-1') {
                throw new Error('First model name incorrect');
            }
        });

        // Test Get Running Models
        await this.runTest('Get Running Models', async () => {
            const runningModels = await api.getRunningModels();
            if (!Array.isArray(runningModels)) {
                throw new Error('Running models should be an array');
            }
            if (runningModels.length !== 1) {
                throw new Error(`Expected 1 running model, got ${runningModels.length}`);
            }
        });

        // Test Get Configuration
        await this.runTest('Get Configuration', async () => {
            const config = await api.getConfig();
            if (!config || typeof config !== 'object') {
                throw new Error('Config should be an object');
            }
            if (config.ollama_api_url !== 'http://localhost:11434') {
                throw new Error('Config API URL incorrect');
            }
        });

        // Test Model Operations
        await this.runTest('Run Model', async () => {
            await api.runModel('test-model-1');
            // Should not throw
        });

        await this.runTest('Delete Model', async () => {
            await api.deleteModel('test-model-1');
            // Should not throw
        });

        await this.runTest('Unload Model', async () => {
            await api.unloadModel('test-model-2');
            // Should not throw
        });

        // Test Error Handling
        await this.runTest('Error Handling - Empty Model Name', async () => {
            try {
                await api.runModel('');
                throw new Error('Should have thrown error for empty model name');
            } catch (error) {
                if (!error.message.includes('Model name is required')) {
                    throw new Error('Wrong error message');
                }
            }
        });

        // Test API Call Handler
        await this.runTest('API Call Handler', async () => {
            let successCalled = false;
            let errorCalled = false;

            // Mock showSuccess and showError
            api.showSuccess = () => { successCalled = true; };
            api.showError = () => { errorCalled = true; };

            // Test successful call
            await api.handleAPICall('TestCall', async () => 'success', {
                showSuccess: true,
                successTitle: 'Test Success'
            });

            if (!successCalled) {
                throw new Error('Success callback not called');
            }

            // Test failed call
            try {
                await api.handleAPICall('TestCall', async () => {
                    throw new Error('Test error');
                }, {
                    showError: true,
                    errorTitle: 'Test Error'
                });
            } catch (error) {
                // Expected
            }

            if (!errorCalled) {
                throw new Error('Error callback not called');
            }
        });

        // Test Service Status
        await this.runTest('Get Service Status', async () => {
            const status = await api.getServiceStatus();
            if (!status || typeof status !== 'object') {
                throw new Error('Status should be an object');
            }
            if (status.status !== 'healthy') {
                throw new Error('Status should be healthy');
            }
        });

        // Test Method Call Logging
        await this.runTest('Method Call Logging', async () => {
            this.mockRuntime.clearCallLog();
            await api.getModels();

            const callLog = this.mockRuntime.getCallLog();
            if (callLog.length === 0) {
                throw new Error('No method calls logged');
            }
            if (callLog[0].method !== 'GetModels') {
                throw new Error('Wrong method logged');
            }
        });

        // Test Diagnostics
        await this.runTest('Diagnostics', async () => {
            const diagnostics = await api.getDiagnostics();
            if (!diagnostics || typeof diagnostics !== 'object') {
                throw new Error('Diagnostics should be an object');
            }
            if (!diagnostics.initialized) {
                throw new Error('Diagnostics should show initialized');
            }
            if (!Array.isArray(diagnostics.availableMethods)) {
                throw new Error('Available methods should be an array');
            }
        });

        // Test Error Scenarios
        await this.runTest('Network Error Handling', async () => {
            // Set GetModels to fail
            this.mockRuntime.setMethodFailure('GetModels', 'Network error');

            try {
                await api.getModels();
                throw new Error('Should have thrown network error');
            } catch (error) {
                if (!error.message.includes('Network error')) {
                    throw new Error('Wrong error message');
                }
            }

            // Reset
            this.mockRuntime.setMethodFailure('GetModels', null);
        });

        // Test Validation
        await this.runTest('Input Validation', async () => {
            const validationTests = [
                { method: 'getModel', args: [''], expectedError: 'Model ID is required' },
                { method: 'runModel', args: [''], expectedError: 'Model name is required' },
                { method: 'deleteModel', args: [''], expectedError: 'Model name is required' },
                { method: 'copyModel', args: ['', 'dest'], expectedError: 'Source and destination model names are required' },
                { method: 'copyModel', args: ['src', ''], expectedError: 'Source and destination model names are required' }
            ];

            for (const test of validationTests) {
                try {
                    await api[test.method](...test.args);
                    throw new Error(`${test.method} should have thrown validation error`);
                } catch (error) {
                    if (!error.message.includes(test.expectedError)) {
                        throw new Error(`Wrong validation error for ${test.method}: ${error.message}`);
                    }
                }
            }
        });

        this.printResults();
    }

    printResults() {
        console.log('\n🧪 Test Results Summary');
        console.log('========================');

        const passed = this.testResults.filter(r => r.status === 'PASS').length;
        const failed = this.testResults.filter(r => r.status === 'FAIL').length;
        const total = this.testResults.length;

        console.log(`Total Tests: ${total}`);
        console.log(`Passed: ${passed}`);
        console.log(`Failed: ${failed}`);
        console.log(`Success Rate: ${((passed / total) * 100).toFixed(1)}%`);

        if (failed > 0) {
            console.log('\n❌ Failed Tests:');
            this.testResults
                .filter(r => r.status === 'FAIL')
                .forEach(r => console.log(`  - ${r.name}: ${r.error}`));
        }

        console.log('\n✅ All tests completed!');
    }
}

// Export for Node.js testing
if (typeof module !== 'undefined' && module.exports) {
    module.exports = GollamaAPITestSuite;
}

// Auto-run tests if this file is executed directly
if (typeof window === 'undefined' && typeof require !== 'undefined') {
    const testSuite = new GollamaAPITestSuite();
    testSuite.runAllTests().catch(console.error);
}
