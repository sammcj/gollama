/**
 * Gollama API - Robust JavaScript API layer with error handling
 * Provides a comprehensive wrapper around Wails v3 service bindings
 */

class GollamaAPI {
    constructor() {
        this.initialized = false;
        this.bindingMethods = {};
        this.fallbackPatterns = [
            'window.wails',
            'window',
            'window.wails.App'
        ];

        // Performance optimization features
        this.cache = new Map();
        this.debounceTimers = new Map();
        this.requestQueue = new Map();
        this.performanceMetrics = {
            totalRequests: 0,
            successfulRequests: 0,
            failedRequests: 0,
            averageResponseTime: 0,
            cacheHits: 0,
            cacheMisses: 0
        };

        // Initialise the API
        this.initialize();
    }

    /**
     * Initialise the API and discover available methods
     */
    async initialize() {
        console.log('🔧 Initializing Gollama API...');

        try {
            // Check if Wails runtime is available
            if (typeof window === 'undefined') {
                throw new Error('Window object not available');
            }

            // Try to import Wails v3 bindings first
            await this.loadWailsBindings();

            // Discover available methods
            this.discoverMethods();

            // Test basic connectivity
            await this.testConnectivity();

            this.initialized = true;
            console.log('✅ Gollama API initialized successfully');
            console.log('📋 Available methods:', Object.keys(this.bindingMethods));

        } catch (error) {
            console.error('❌ Failed to initialize Gollama API:', error);
            this.initialized = false;
            throw error;
        }
    }

    /**
     * Load Wails v3 bindings dynamically
     */
    async loadWailsBindings() {
        try {
            // Try to import the generated bindings
            const bindings = await import('/bindings/gollama-gui/app.js');
            console.log('✅ Wails v3 bindings loaded successfully');

            // Store binding methods
            this.bindingMethods = {
                GetModels: bindings.GetModels,
                GetModel: bindings.GetModel,
                GetRunningModels: bindings.GetRunningModels,
                RunModel: bindings.RunModel,
                DeleteModel: bindings.DeleteModel,
                UnloadModel: bindings.UnloadModel,
                CopyModel: bindings.CopyModel,
                PushModel: bindings.PushModel,
                PullModel: bindings.PullModel,
                GetModelDetails: bindings.GetModelDetails,
                EstimateVRAM: bindings.EstimateVRAM,
                GetConfig: bindings.GetConfig,
                UpdateConfig: bindings.UpdateConfig,
                HealthCheck: bindings.HealthCheck,
                TestServiceBinding: bindings.TestServiceBinding,
                GetServiceStatus: bindings.GetServiceStatus,
                GetDiagnosticInfo: bindings.GetDiagnosticInfo,
                RunDiagnosticTests: bindings.RunDiagnosticTests,
                VerifyServiceBinding: bindings.VerifyServiceBinding
            };

        } catch (error) {
            console.warn('⚠️  Failed to load Wails v3 bindings, trying fallback patterns:', error);
            this.discoverFallbackMethods();
        }
    }

    /**
     * Discover methods using fallback patterns
     */
    discoverFallbackMethods() {
        const expectedMethods = [
            'GetModels', 'GetModel', 'GetRunningModels', 'RunModel', 'DeleteModel',
            'UnloadModel', 'CopyModel', 'PushModel', 'PullModel', 'GetModelDetails',
            'EstimateVRAM', 'GetConfig', 'UpdateConfig', 'HealthCheck',
            'TestServiceBinding', 'GetServiceStatus', 'GetDiagnosticInfo',
            'RunDiagnosticTests', 'VerifyServiceBinding'
        ];

        for (const pattern of this.fallbackPatterns) {
            const obj = this.getObjectByPath(pattern);
            if (obj) {
                console.log(`🔍 Checking pattern: ${pattern}`);

                for (const method of expectedMethods) {
                    if (typeof obj[method] === 'function') {
                        this.bindingMethods[method] = obj[method].bind(obj);
                        console.log(`✓ Found method: ${method} in ${pattern}`);
                    }
                }
            }
        }
    }

    /**
     * Discover all available methods for debugging
     */
    discoverMethods() {
        console.log('🔍 Discovering available Wails methods...');

        // Log Wails runtime information
        if (typeof window.wails !== 'undefined') {
            console.log('✓ window.wails available:', Object.keys(window.wails));
        }

        // Log available binding methods
        const availableMethods = Object.keys(this.bindingMethods).filter(
            key => typeof this.bindingMethods[key] === 'function'
        );

        console.log(`📊 Found ${availableMethods.length} available methods:`, availableMethods);

        // Log missing methods
        const expectedMethods = [
            'GetModels', 'GetModel', 'GetRunningModels', 'RunModel', 'DeleteModel',
            'UnloadModel', 'CopyModel', 'PushModel', 'PullModel', 'GetModelDetails',
            'EstimateVRAM', 'GetConfig', 'UpdateConfig', 'HealthCheck'
        ];

        const missingMethods = expectedMethods.filter(method => !availableMethods.includes(method));
        if (missingMethods.length > 0) {
            console.warn('⚠️  Missing methods:', missingMethods);
        }
    }

    /**
     * Test basic connectivity with the service
     */
    async testConnectivity() {
        console.log('🔍 Testing service connectivity...');

        try {
            // Try TestServiceBinding first if available
            if (this.bindingMethods.TestServiceBinding) {
                const result = await this.bindingMethods.TestServiceBinding();
                console.log('✅ Service binding test result:', result);
                return;
            }

            // Fallback to HealthCheck
            if (this.bindingMethods.HealthCheck) {
                await this.bindingMethods.HealthCheck();
                console.log('✅ Health check passed');
                return;
            }

            // Last resort - try GetModels
            if (this.bindingMethods.GetModels) {
                await this.bindingMethods.GetModels();
                console.log('✅ GetModels test passed');
                return;
            }

            throw new Error('No connectivity test methods available');

        } catch (error) {
            console.error('❌ Connectivity test failed:', error);
            throw new Error(`Service connectivity test failed: ${error.message}`);
        }
    }

    /**
     * Get object by dot-notation path
     */
    getObjectByPath(path) {
        try {
            return path.split('.').reduce((obj, key) => obj && obj[key], window);
        } catch (error) {
            return null;
        }
    }

    /**
     * Generic method caller with comprehensive error handling and performance optimization
     */
    async callMethod(methodName, ...args) {
        if (!this.initialized) {
            throw new Error('Gollama API not initialized. Call initialize() first.');
        }

        const method = this.bindingMethods[methodName];
        if (!method || typeof method !== 'function') {
            throw new Error(`Method '${methodName}' not available. Available methods: ${Object.keys(this.bindingMethods).join(', ')}`);
        }

        // Performance tracking
        const startTime = performance.now();
        this.performanceMetrics.totalRequests++;

        try {
            console.log(`🔄 Calling ${methodName}${args.length > 0 ? ' with args:' : ''}`, args);
            const result = await method(...args);

            // Update performance metrics
            const duration = performance.now() - startTime;
            this.performanceMetrics.successfulRequests++;
            this.updateAverageResponseTime(duration);

            console.log(`✅ ${methodName} completed successfully (${duration.toFixed(2)}ms)`);
            return result;
        } catch (error) {
            // Update performance metrics
            this.performanceMetrics.failedRequests++;

            console.error(`❌ ${methodName} failed:`, error);

            // Enhanced error information
            const enhancedError = new Error(`${methodName} failed: ${error.message}`);
            enhancedError.originalError = error;
            enhancedError.methodName = methodName;
            enhancedError.args = args;
            enhancedError.duration = performance.now() - startTime;

            throw enhancedError;
        }
    }

    /**
     * Update average response time metric
     */
    updateAverageResponseTime(duration) {
        const totalRequests = this.performanceMetrics.successfulRequests;
        const currentAverage = this.performanceMetrics.averageResponseTime;
        this.performanceMetrics.averageResponseTime =
            ((currentAverage * (totalRequests - 1)) + duration) / totalRequests;
    }

    /**
     * Debounced method caller to prevent excessive API calls
     */
    async callMethodDebounced(methodName, delay = 300, ...args) {
        const key = `${methodName}_${JSON.stringify(args)}`;

        // Clear existing timer
        if (this.debounceTimers.has(key)) {
            clearTimeout(this.debounceTimers.get(key));
        }

        return new Promise((resolve, reject) => {
            const timer = setTimeout(async () => {
                try {
                    const result = await this.callMethod(methodName, ...args);
                    this.debounceTimers.delete(key);
                    resolve(result);
                } catch (error) {
                    this.debounceTimers.delete(key);
                    reject(error);
                }
            }, delay);

            this.debounceTimers.set(key, timer);
        });
    }

    /**
     * Cached method caller with TTL support
     */
    async callMethodCached(methodName, ttlSeconds = 30, ...args) {
        const cacheKey = `${methodName}_${JSON.stringify(args)}`;
        const cached = this.cache.get(cacheKey);

        // Check if cached result is still valid
        if (cached && Date.now() - cached.timestamp < ttlSeconds * 1000) {
            console.log(`📋 Cache hit for ${methodName}`);
            this.performanceMetrics.cacheHits++;
            return cached.data;
        }

        // Cache miss - make the actual call
        console.log(`📥 Cache miss for ${methodName}`);
        this.performanceMetrics.cacheMisses++;

        try {
            const result = await this.callMethod(methodName, ...args);

            // Cache the result
            this.cache.set(cacheKey, {
                data: result,
                timestamp: Date.now()
            });

            return result;
        } catch (error) {
            // Don't cache errors
            throw error;
        }
    }

    /**
     * Clear cache for specific method or all methods
     */
    clearCache(methodName = null) {
        if (methodName) {
            // Clear cache entries for specific method
            for (const [key] of this.cache) {
                if (key.startsWith(`${methodName}_`)) {
                    this.cache.delete(key);
                }
            }
            console.log(`🗑️ Cleared cache for ${methodName}`);
        } else {
            // Clear all cache
            this.cache.clear();
            console.log('🗑️ Cleared all cache');
        }
    }

    /**
     * Get performance metrics
     */
    getPerformanceMetrics() {
        return {
            ...this.performanceMetrics,
            cacheSize: this.cache.size,
            activeDebouncers: this.debounceTimers.size,
            timestamp: new Date().toISOString()
        };
    }

    /**
     * Reset performance metrics
     */
    resetPerformanceMetrics() {
        this.performanceMetrics = {
            totalRequests: 0,
            successfulRequests: 0,
            failedRequests: 0,
            averageResponseTime: 0,
            cacheHits: 0,
            cacheMisses: 0
        };
        console.log('📊 Performance metrics reset');
    }

    /**
     * Refresh cache by calling the backend refresh method
     */
    async refreshCache() {
        try {
            // Clear local cache first
            this.clearCache();

            // Call backend cache refresh
            const result = await this.callMethod('RefreshCache');
            console.log('🔄 Cache refreshed successfully');
            return result;
        } catch (error) {
            console.error('❌ Failed to refresh cache:', error);
            throw error;
        }
    }

    /**
     * Get comprehensive performance metrics from backend
     */
    async getBackendPerformanceMetrics() {
        try {
            const backendMetrics = await this.callMethod('GetPerformanceMetrics');
            const frontendMetrics = this.getPerformanceMetrics();

            return {
                frontend: frontendMetrics,
                backend: backendMetrics,
                combined: {
                    timestamp: new Date().toISOString(),
                    total_requests: frontendMetrics.totalRequests,
                    success_rate: frontendMetrics.totalRequests > 0 ?
                        (frontendMetrics.successfulRequests / frontendMetrics.totalRequests * 100).toFixed(2) + '%' : '0%',
                    cache_hit_rate: (frontendMetrics.cacheHits + frontendMetrics.cacheMisses) > 0 ?
                        (frontendMetrics.cacheHits / (frontendMetrics.cacheHits + frontendMetrics.cacheMisses) * 100).toFixed(2) + '%' : '0%',
                    average_response_time: frontendMetrics.averageResponseTime.toFixed(2) + 'ms'
                }
            };
        } catch (error) {
            console.error('❌ Failed to get backend performance metrics:', error);
            throw error;
        }
    }

    // ===========================================
    // Model Management Methods
    // ===========================================

    /**
     * Get all available models with caching
     */
    async getModels(useCache = true) {
        if (useCache) {
            return this.callMethodCached('GetModels', 30); // 30 second cache
        }
        return this.callMethod('GetModels');
    }

    /**
     * Get a specific model by ID with caching
     */
    async getModel(id, useCache = true) {
        if (!id) {
            throw new Error('Model ID is required');
        }
        if (useCache) {
            return this.callMethodCached('GetModel', 60, id); // 60 second cache for individual models
        }
        return this.callMethod('GetModel', id);
    }

    /**
     * Get currently running models with shorter cache TTL
     */
    async getRunningModels(useCache = true) {
        if (useCache) {
            return this.callMethodCached('GetRunningModels', 10); // 10 second cache for running models
        }
        return this.callMethod('GetRunningModels');
    }

    /**
     * Run a model and invalidate relevant caches
     */
    async runModel(name) {
        if (!name) {
            throw new Error('Model name is required');
        }

        try {
            const result = await this.callMethod('RunModel', name);

            // Invalidate caches since model state changed
            this.clearCache('GetModels');
            this.clearCache('GetRunningModels');

            return result;
        } catch (error) {
            throw error;
        }
    }

    /**
     * Delete a model and invalidate relevant caches
     */
    async deleteModel(name) {
        if (!name) {
            throw new Error('Model name is required');
        }

        try {
            const result = await this.callMethod('DeleteModel', name);

            // Invalidate caches since model was deleted
            this.clearCache('GetModels');
            this.clearCache('GetRunningModels');
            this.clearCache('GetModel');

            return result;
        } catch (error) {
            throw error;
        }
    }

    /**
     * Unload a running model and invalidate relevant caches
     */
    async unloadModel(name) {
        if (!name) {
            throw new Error('Model name is required');
        }

        try {
            const result = await this.callMethod('UnloadModel', name);

            // Invalidate caches since model state changed
            this.clearCache('GetModels');
            this.clearCache('GetRunningModels');

            return result;
        } catch (error) {
            throw error;
        }
    }

    /**
     * Copy a model
     */
    async copyModel(source, dest) {
        if (!source || !dest) {
            throw new Error('Source and destination model names are required');
        }
        return this.callMethod('CopyModel', source, dest);
    }

    /**
     * Push a model to registry
     */
    async pushModel(name) {
        if (!name) {
            throw new Error('Model name is required');
        }
        return this.callMethod('PushModel', name);
    }

    /**
     * Pull a model from registry
     */
    async pullModel(name) {
        if (!name) {
            throw new Error('Model name is required');
        }
        return this.callMethod('PullModel', name);
    }

    /**
     * Get detailed model information
     */
    async getModelDetails(name) {
        if (!name) {
            throw new Error('Model name is required');
        }
        return this.callMethod('GetModelDetails', name);
    }

    // ===========================================
    // Configuration Methods
    // ===========================================

    /**
     * Get current configuration
     */
    async getConfig() {
        return this.callMethod('GetConfig');
    }

    /**
     * Update configuration
     */
    async updateConfig(settings) {
        if (!settings || typeof settings !== 'object') {
            throw new Error('Settings object is required');
        }
        return this.callMethod('UpdateConfig', settings);
    }

    // ===========================================
    // Utility Methods
    // ===========================================

    /**
     * Estimate vRAM usage for a model
     */
    async estimateVRAM(request) {
        if (!request || !request.modelName) {
            throw new Error('vRAM estimation request with modelName is required');
        }
        return this.callMethod('EstimateVRAM', request);
    }

    /**
     * Health check
     */
    async healthCheck() {
        return this.callMethod('HealthCheck');
    }

    /**
     * Test service binding
     */
    async testServiceBinding() {
        return this.callMethod('TestServiceBinding');
    }

    /**
     * Get detailed service status for debugging
     */
    async getServiceStatus() {
        return this.callMethod('GetServiceStatus');
    }

    /**
     * Get comprehensive diagnostic information
     */
    async getDiagnosticInfo() {
        return this.callMethod('GetDiagnosticInfo');
    }

    /**
     * Run comprehensive diagnostic tests
     */
    async runDiagnosticTests() {
        return this.callMethod('RunDiagnosticTests');
    }

    /**
     * Verify service binding is working correctly
     */
    async verifyServiceBinding() {
        return this.callMethod('VerifyServiceBinding');
    }

    // ===========================================
    // Error Handling and User Feedback
    // ===========================================

    /**
     * Show error toast notification
     */
    showError(title, message, duration = 5000) {
        console.error(`${title}: ${message}`);

        // Create toast element
        const toast = document.createElement('div');
        toast.className = 'fixed top-4 right-4 bg-red-500 text-white px-6 py-4 rounded-lg shadow-lg z-50 max-w-md';
        toast.innerHTML = `
            <div class="flex items-start">
                <div class="flex-shrink-0">
                    <svg class="h-5 w-5" fill="currentColor" viewBox="0 0 20 20">
                        <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/>
                    </svg>
                </div>
                <div class="ml-3">
                    <h3 class="text-sm font-medium">${title}</h3>
                    <p class="mt-1 text-sm opacity-90">${message}</p>
                </div>
                <button class="ml-4 flex-shrink-0 text-white hover:text-gray-200" onclick="this.parentElement.parentElement.remove()">
                    <svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
                        <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd"/>
                    </svg>
                </button>
            </div>
        `;

        document.body.appendChild(toast);

        // Auto-remove after duration
        setTimeout(() => {
            if (toast.parentElement) {
                toast.remove();
            }
        }, duration);
    }

    /**
     * Show success toast notification
     */
    showSuccess(title, message, duration = 3000) {
        console.log(`${title}: ${message}`);

        const toast = document.createElement('div');
        toast.className = 'fixed top-4 right-4 bg-green-500 text-white px-6 py-4 rounded-lg shadow-lg z-50 max-w-md';
        toast.innerHTML = `
            <div class="flex items-start">
                <div class="flex-shrink-0">
                    <svg class="h-5 w-5" fill="currentColor" viewBox="0 0 20 20">
                        <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
                    </svg>
                </div>
                <div class="ml-3">
                    <h3 class="text-sm font-medium">${title}</h3>
                    <p class="mt-1 text-sm opacity-90">${message}</p>
                </div>
                <button class="ml-4 flex-shrink-0 text-white hover:text-gray-200" onclick="this.parentElement.parentElement.remove()">
                    <svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
                        <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd"/>
                    </svg>
                </button>
            </div>
        `;

        document.body.appendChild(toast);

        setTimeout(() => {
            if (toast.parentElement) {
                toast.remove();
            }
        }, duration);
    }

    /**
     * Handle API call with automatic error handling and user feedback
     */
    async handleAPICall(methodName, apiCall, options = {}) {
        const {
            showLoading = true,
            showSuccess = true,
            showError = true,
            successTitle = 'Success',
            successMessage = `${methodName} completed successfully`,
            errorTitle = 'Error',
            loadingMessage = `Processing ${methodName}...`
        } = options;

        try {
            if (showLoading && typeof window.showLoading === 'function') {
                window.showLoading(loadingMessage);
            }

            const result = await apiCall();

            if (showSuccess) {
                this.showSuccess(successTitle, successMessage);
            }

            return result;

        } catch (error) {
            console.error(`${methodName} failed:`, error);

            if (showError) {
                this.showError(
                    errorTitle,
                    error.message || `Failed to ${methodName.toLowerCase()}`
                );
            }

            throw error;

        } finally {
            if (showLoading && typeof window.hideLoading === 'function') {
                window.hideLoading();
            }
        }
    }

    // ===========================================
    // Diagnostic Methods
    // ===========================================

    /**
     * Get comprehensive diagnostic information
     */
    async getDiagnostics() {
        const diagnostics = {
            timestamp: new Date().toISOString(),
            initialized: this.initialized,
            availableMethods: Object.keys(this.bindingMethods),
            wailsRuntime: typeof window.wails !== 'undefined',
            bindingPatterns: this.fallbackPatterns,
            runtimeInfo: this.getRuntimeInfo(),
            methodDiscovery: this.performMethodDiscovery(),
            connectivityTests: await this.runConnectivityTests()
        };

        try {
            // Try to get service status if available
            if (this.bindingMethods.GetServiceStatus) {
                diagnostics.serviceStatus = await this.bindingMethods.GetServiceStatus();
            }
        } catch (error) {
            diagnostics.serviceStatusError = error.message;
        }

        return diagnostics;
    }

    /**
     * Get detailed runtime information
     */
    getRuntimeInfo() {
        const info = {
            userAgent: navigator.userAgent,
            platform: navigator.platform,
            language: navigator.language,
            cookieEnabled: navigator.cookieEnabled,
            onLine: navigator.onLine,
            windowLocation: window.location.href,
            windowSize: {
                width: window.innerWidth,
                height: window.innerHeight
            },
            screen: {
                width: screen.width,
                height: screen.height,
                colorDepth: screen.colorDepth
            },
            wailsObjects: {}
        };

        // Inspect Wails objects
        if (typeof window.wails !== 'undefined') {
            info.wailsObjects.wails = {
                type: typeof window.wails,
                keys: Object.keys(window.wails),
                methods: Object.keys(window.wails).filter(key => typeof window.wails[key] === 'function')
            };
        }

        // Check for other potential Wails objects
        const potentialWailsObjects = ['wails', 'go', 'backend', 'app'];
        for (const obj of potentialWailsObjects) {
            if (typeof window[obj] !== 'undefined') {
                info.wailsObjects[obj] = {
                    type: typeof window[obj],
                    keys: typeof window[obj] === 'object' ? Object.keys(window[obj]) : [],
                    methods: typeof window[obj] === 'object' ?
                        Object.keys(window[obj]).filter(key => typeof window[obj][key] === 'function') : []
                };
            }
        }

        return info;
    }

    /**
     * Perform comprehensive method discovery
     */
    performMethodDiscovery() {
        const discovery = {
            expectedMethods: [
                'GetModels', 'GetModel', 'GetRunningModels', 'RunModel', 'DeleteModel',
                'UnloadModel', 'CopyModel', 'PushModel', 'PullModel', 'GetModelInfo',
                'EstimateVRAMForModel', 'GetConfig', 'UpdateConfig', 'HealthCheck',
                'TestServiceBinding', 'GetServiceStatus', 'SearchModels', 'GetModelOperations',
                'ValidateModelName', 'GetSystemInfo', 'GetDiagnosticInfo', 'RunDiagnosticTests',
                'VerifyServiceBinding'
            ],
            foundMethods: {},
            missingMethods: [],
            unexpectedMethods: [],
            bindingPatternResults: {}
        };

        // Check each binding pattern
        for (const pattern of this.fallbackPatterns) {
            const obj = this.getObjectByPath(pattern);
            const patternResult = {
                available: obj !== null && obj !== undefined,
                type: typeof obj,
                methods: []
            };

            if (obj) {
                const allKeys = Object.keys(obj);
                const methods = allKeys.filter(key => typeof obj[key] === 'function');
                patternResult.methods = methods;
                patternResult.allKeys = allKeys;
            }

            discovery.bindingPatternResults[pattern] = patternResult;
        }

        // Check for expected methods
        for (const method of discovery.expectedMethods) {
            let found = false;
            let foundIn = [];

            for (const pattern of this.fallbackPatterns) {
                const obj = this.getObjectByPath(pattern);
                if (obj && typeof obj[method] === 'function') {
                    found = true;
                    foundIn.push(pattern);
                }
            }

            if (found) {
                discovery.foundMethods[method] = foundIn;
            } else {
                discovery.missingMethods.push(method);
            }
        }

        // Find unexpected methods in window.wails
        if (typeof window.wails !== 'undefined') {
            const wailsMethods = Object.keys(window.wails).filter(key => typeof window.wails[key] === 'function');
            discovery.unexpectedMethods = wailsMethods.filter(method => !discovery.expectedMethods.includes(method));
        }

        return discovery;
    }

    /**
     * Run comprehensive connectivity tests
     */
    async runConnectivityTests() {
        const tests = {
            timestamp: new Date().toISOString(),
            results: {},
            summary: {
                total: 0,
                passed: 0,
                failed: 0,
                skipped: 0
            }
        };

        const testCases = [
            {
                name: 'TestServiceBinding',
                description: 'Test basic service binding functionality',
                method: 'TestServiceBinding',
                args: []
            },
            {
                name: 'HealthCheck',
                description: 'Test service health check',
                method: 'HealthCheck',
                args: []
            },
            {
                name: 'GetServiceStatus',
                description: 'Get detailed service status',
                method: 'GetServiceStatus',
                args: []
            },
            {
                name: 'GetSystemInfo',
                description: 'Get system information',
                method: 'GetSystemInfo',
                args: []
            },
            {
                name: 'GetDiagnosticInfo',
                description: 'Get comprehensive diagnostic information',
                method: 'GetDiagnosticInfo',
                args: []
            },
            {
                name: 'RunDiagnosticTests',
                description: 'Run comprehensive diagnostic tests',
                method: 'RunDiagnosticTests',
                args: []
            },
            {
                name: 'VerifyServiceBinding',
                description: 'Verify service binding is working correctly',
                method: 'VerifyServiceBinding',
                args: []
            },
            {
                name: 'GetConfig',
                description: 'Get configuration',
                method: 'GetConfig',
                args: []
            },
            {
                name: 'GetModels',
                description: 'Get model list',
                method: 'GetModels',
                args: []
            }
        ];

        for (const testCase of testCases) {
            tests.summary.total++;
            const testResult = {
                name: testCase.name,
                description: testCase.description,
                status: 'unknown',
                startTime: new Date().toISOString(),
                duration: 0,
                error: null,
                result: null
            };

            const startTime = Date.now();

            try {
                if (!this.bindingMethods[testCase.method]) {
                    testResult.status = 'skipped';
                    testResult.error = 'Method not available';
                    tests.summary.skipped++;
                } else {
                    testResult.result = await this.bindingMethods[testCase.method](...testCase.args);
                    testResult.status = 'passed';
                    tests.summary.passed++;
                }
            } catch (error) {
                testResult.status = 'failed';
                testResult.error = error.message;
                tests.summary.failed++;
            }

            testResult.duration = Date.now() - startTime;
            testResult.endTime = new Date().toISOString();
            tests.results[testCase.name] = testResult;
        }

        return tests;
    }

    /**
     * Log comprehensive diagnostic information
     */
    async logDiagnostics() {
        console.log('=== 🔍 GOLLAMA API DIAGNOSTICS ===');

        const diagnostics = await this.getDiagnostics();

        console.log('📊 API Status:', {
            initialized: diagnostics.initialized,
            availableMethods: diagnostics.availableMethods.length,
            wailsRuntime: diagnostics.wailsRuntime
        });

        console.log('🌐 Runtime Info:', diagnostics.runtimeInfo);
        console.log('🔍 Method Discovery:', diagnostics.methodDiscovery);
        console.log('🧪 Connectivity Tests:', diagnostics.connectivityTests);

        console.log('📋 Available Methods:', diagnostics.availableMethods);

        if (diagnostics.serviceStatus) {
            console.log('⚙️  Service Status:', diagnostics.serviceStatus);
        }

        if (diagnostics.serviceStatusError) {
            console.warn('⚠️  Service Status Error:', diagnostics.serviceStatusError);
        }

        console.log('=== END DIAGNOSTICS ===');

        return diagnostics;
    }

    /**
     * Create diagnostic report for debugging
     */
    async createDiagnosticReport() {
        const diagnostics = await this.getDiagnostics();

        const report = {
            title: 'Gollama API Diagnostic Report',
            timestamp: diagnostics.timestamp,
            summary: {
                apiInitialized: diagnostics.initialized,
                wailsRuntimeAvailable: diagnostics.wailsRuntime,
                methodsFound: diagnostics.availableMethods.length,
                connectivityTestsPassed: diagnostics.connectivityTests?.summary?.passed || 0,
                connectivityTestsFailed: diagnostics.connectivityTests?.summary?.failed || 0
            },
            details: diagnostics
        };

        // Generate human-readable summary
        report.readableSummary = this.generateReadableSummary(report);

        return report;
    }

    /**
     * Generate human-readable diagnostic summary
     */
    generateReadableSummary(report) {
        const lines = [];

        lines.push(`🔍 Gollama API Diagnostic Report - ${new Date(report.timestamp).toLocaleString()}`);
        lines.push('');

        // API Status
        lines.push('📊 API Status:');
        lines.push(`  • Initialized: ${report.summary.apiInitialized ? '✅' : '❌'}`);
        lines.push(`  • Wails Runtime: ${report.summary.wailsRuntimeAvailable ? '✅' : '❌'}`);
        lines.push(`  • Methods Found: ${report.summary.methodsFound}`);
        lines.push('');

        // Connectivity Tests
        if (report.details.connectivityTests) {
            const tests = report.details.connectivityTests;
            lines.push('🧪 Connectivity Tests:');
            lines.push(`  • Total: ${tests.summary.total}`);
            lines.push(`  • Passed: ${tests.summary.passed} ✅`);
            lines.push(`  • Failed: ${tests.summary.failed} ${tests.summary.failed > 0 ? '❌' : ''}`);
            lines.push(`  • Skipped: ${tests.summary.skipped} ${tests.summary.skipped > 0 ? '⏭️' : ''}`);
            lines.push('');
        }

        // Method Discovery
        if (report.details.methodDiscovery) {
            const discovery = report.details.methodDiscovery;
            lines.push('🔍 Method Discovery:');
            lines.push(`  • Expected Methods: ${discovery.expectedMethods.length}`);
            lines.push(`  • Found Methods: ${Object.keys(discovery.foundMethods).length}`);
            lines.push(`  • Missing Methods: ${discovery.missingMethods.length}`);

            if (discovery.missingMethods.length > 0) {
                lines.push(`    Missing: ${discovery.missingMethods.join(', ')}`);
            }
            lines.push('');
        }

        // Service Status
        if (report.details.serviceStatus) {
            const status = report.details.serviceStatus;
            lines.push('⚙️  Service Status:');
            lines.push(`  • Overall Status: ${status.status} ${status.status === 'healthy' ? '✅' : '❌'}`);
            lines.push(`  • Version: ${status.version}`);

            if (status.services) {
                lines.push('  • Services:');
                for (const [name, service] of Object.entries(status.services)) {
                    lines.push(`    - ${name}: ${service.status} ${service.status === 'healthy' ? '✅' : '❌'}`);
                }
            }
        }

        return lines.join('\n');
    }

    /**
     * Display diagnostic report in console and optionally in UI
     */
    async displayDiagnosticReport(showInUI = false) {
        const report = await this.createDiagnosticReport();

        console.log(report.readableSummary);

        if (showInUI) {
            this.showDiagnosticModal(report);
        }

        return report;
    }

    /**
     * Show diagnostic modal in UI
     */
    showDiagnosticModal(report) {
        // Create modal backdrop
        const backdrop = document.createElement('div');
        backdrop.className = 'fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center p-4';

        // Create modal content
        const modal = document.createElement('div');
        modal.className = 'bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-hidden';

        modal.innerHTML = `
            <div class="flex items-center justify-between p-6 border-b">
                <h2 class="text-xl font-semibold text-gray-900">🔍 Diagnostic Report</h2>
                <button class="text-gray-400 hover:text-gray-600" onclick="this.closest('.fixed').remove()">
                    <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                    </svg>
                </button>
            </div>
            <div class="p-6 overflow-y-auto max-h-[70vh]">
                <div class="space-y-6">
                    <div class="bg-gray-50 p-4 rounded-lg">
                        <h3 class="font-medium text-gray-900 mb-2">📊 Summary</h3>
                        <div class="grid grid-cols-2 gap-4 text-sm">
                            <div>API Initialized: ${report.summary.apiInitialized ? '✅ Yes' : '❌ No'}</div>
                            <div>Wails Runtime: ${report.summary.wailsRuntimeAvailable ? '✅ Available' : '❌ Not Available'}</div>
                            <div>Methods Found: ${report.summary.methodsFound}</div>
                            <div>Tests Passed: ${report.summary.connectivityTestsPassed}/${report.summary.connectivityTestsPassed + report.summary.connectivityTestsFailed}</div>
                        </div>
                    </div>

                    <div class="bg-gray-50 p-4 rounded-lg">
                        <h3 class="font-medium text-gray-900 mb-2">📋 Available Methods</h3>
                        <div class="text-sm text-gray-600">
                            ${report.details.availableMethods.join(', ')}
                        </div>
                    </div>

                    <div class="bg-gray-50 p-4 rounded-lg">
                        <h3 class="font-medium text-gray-900 mb-2">🧪 Test Results</h3>
                        <div class="space-y-2 text-sm">
                            ${Object.entries(report.details.connectivityTests?.results || {}).map(([name, result]) => `
                                <div class="flex items-center justify-between">
                                    <span>${result.description}</span>
                                    <span class="${result.status === 'passed' ? 'text-green-600' : result.status === 'failed' ? 'text-red-600' : 'text-yellow-600'}">
                                        ${result.status === 'passed' ? '✅' : result.status === 'failed' ? '❌' : '⏭️'} ${result.status}
                                    </span>
                                </div>
                            `).join('')}
                        </div>
                    </div>

                    <div class="bg-gray-50 p-4 rounded-lg">
                        <h3 class="font-medium text-gray-900 mb-2">📄 Full Report</h3>
                        <pre class="text-xs text-gray-600 overflow-x-auto whitespace-pre-wrap">${JSON.stringify(report.details, null, 2)}</pre>
                    </div>
                </div>
            </div>
            <div class="flex justify-end space-x-3 p-6 border-t">
                <button class="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200"
                        onclick="navigator.clipboard.writeText(JSON.stringify(${JSON.stringify(report)}, null, 2)); this.textContent='Copied!'">
                    Copy Report
                </button>
                <button class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
                        onclick="this.closest('.fixed').remove()">
                    Close
                </button>
            </div>
        `;

        backdrop.appendChild(modal);
        document.body.appendChild(backdrop);

        // Close on backdrop click
        backdrop.addEventListener('click', (e) => {
            if (e.target === backdrop) {
                backdrop.remove();
            }
        });
    }

    /**
     * Run continuous health monitoring
     */
    startHealthMonitoring(interval = 30000) {
        if (this.healthMonitorInterval) {
            clearInterval(this.healthMonitorInterval);
        }

        console.log(`🔄 Starting health monitoring (interval: ${interval}ms)`);

        this.healthMonitorInterval = setInterval(async () => {
            try {
                if (this.bindingMethods.GetServiceStatus) {
                    const status = await this.bindingMethods.GetServiceStatus();
                    console.log('💓 Health check:', status.status, new Date().toISOString());

                    // Emit custom event for health status
                    window.dispatchEvent(new CustomEvent('gollamaHealthUpdate', {
                        detail: status
                    }));
                }
            } catch (error) {
                console.warn('💔 Health check failed:', error.message);

                window.dispatchEvent(new CustomEvent('gollamaHealthUpdate', {
                    detail: { status: 'unhealthy', error: error.message }
                }));
            }
        }, interval);

        return this.healthMonitorInterval;
    }

    /**
     * Stop health monitoring
     */
    stopHealthMonitoring() {
        if (this.healthMonitorInterval) {
            clearInterval(this.healthMonitorInterval);
            this.healthMonitorInterval = null;
            console.log('⏹️  Health monitoring stopped');
        }
    }
}

// Create global API instance
window.gollamaAPI = new GollamaAPI();

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
    module.exports = GollamaAPI;
}
