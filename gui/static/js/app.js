// Gollama GUI JavaScript

// Global state with performance optimizations
let currentModels = [];
let currentRunningModels = [];
let currentView = 'models';
let autoRefreshInterval = null;
let loadingStates = new Set();
let lastRefreshTime = 0;
let refreshDebounceTimer = null;

// Performance constants
const REFRESH_DEBOUNCE_DELAY = 500; // ms
const MIN_REFRESH_INTERVAL = 2000; // ms
const LOADING_ANIMATION_DELAY = 200; // ms

// Initialize the application
document.addEventListener('DOMContentLoaded', async function() {
    console.log('Gollama GUI initialized');

    try {
        // Wait for API to be ready
        await waitForAPI();

        // Initialize the application
        await initializeApp();

        // Load initial view
        await showModels();
    } catch (error) {
        console.error('Failed to initialize Gollama GUI:', error);
        showInitializationError(error);
    }
});

// Wait for the API to be initialized
async function waitForAPI() {
    const maxAttempts = 100; // 10 seconds max
    let attempts = 0;

    console.log('⏳ Waiting for Gollama API to initialize...');

    while (attempts < maxAttempts) {
        if (window.gollamaAPI && window.gollamaAPI.initialized) {
            console.log('✅ Gollama API is ready');
            return;
        }

        await new Promise(resolve => setTimeout(resolve, 100));
        attempts++;

        // Log progress every 2 seconds
        if (attempts % 20 === 0) {
            console.log(`⏳ Still waiting for API... (${attempts/10}s)`);
        }
    }

    throw new Error('Gollama API failed to initialize within timeout (10 seconds)');
}

// Initialize application
async function initializeApp() {
    try {
        console.log('Initializing Gollama GUI with enhanced API layer...');

        // Test the service binding first
        console.log('Testing service binding...');
        try {
            const bindingTest = await window.gollamaAPI.testServiceBinding();
            console.log('Service binding test result:', bindingTest);
        } catch (error) {
            console.warn('Service binding test failed, continuing with health check:', error.message);
        }

        // Test the service connection
        try {
            await window.gollamaAPI.healthCheck();
            console.log('✅ Service connection established successfully');
        } catch (error) {
            console.warn('Health check failed, service may not be fully ready:', error.message);
        }

        // Load settings and apply theme
        try {
            const settings = await window.gollamaAPI.getConfig();
            console.log('Settings loaded:', settings);

            applyTheme(settings.Theme || settings.theme || 'dark');

            // Set up auto-refresh if enabled
            if (settings.AutoRefresh || settings.autoRefresh) {
                setupAutoRefresh(settings.RefreshInterval || settings.refreshInterval || 30);
            }
        } catch (error) {
            console.warn('Failed to load settings, using defaults:', error.message);
            applyTheme('dark');
        }

        console.log('✅ Gollama GUI initialization complete');

    } catch (error) {
        console.error('Failed to initialize application:', error);
        window.gollamaAPI.showError('Initialization Error', 'Failed to initialize Gollama GUI: ' + error.message);
        throw error;
    }
}

// Show initialization error
function showInitializationError(error) {
    document.getElementById('main-content').innerHTML = `
        <div class="text-center py-12">
            <div class="mb-6">
                <svg class="mx-auto h-16 w-16 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"/>
                </svg>
            </div>
            <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-4">Initialization Failed</h2>
            <p class="text-gray-600 dark:text-gray-400 mb-6">${error.message || 'Failed to initialize Gollama GUI'}</p>
            <div class="space-y-4">
                <button class="btn-primary" onclick="location.reload()">Reload Application</button>
                <button class="px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200" onclick="showDiagnostics()">Show Diagnostics</button>
            </div>
        </div>
    `;
}

// Enhanced loading state management
function showLoadingState(message = 'Loading...') {
    const loadingId = `loading-${Date.now()}`;
    loadingStates.add(loadingId);

    // Delay showing loading indicator to prevent flashing for fast operations
    setTimeout(() => {
        if (loadingStates.has(loadingId)) {
            const loadingElement = document.getElementById('loading-indicator');
            if (loadingElement) {
                loadingElement.textContent = message;
                loadingElement.classList.remove('hidden');
            }
        }
    }, LOADING_ANIMATION_DELAY);

    return loadingId;
}

function hideLoadingState(loadingId = null) {
    if (loadingId) {
        loadingStates.delete(loadingId);
    } else {
        loadingStates.clear();
    }

    // Only hide if no other loading operations are active
    if (loadingStates.size === 0) {
        const loadingElement = document.getElementById('loading-indicator');
        if (loadingElement) {
            loadingElement.classList.add('hidden');
        }
    }
}

// Performance monitoring display
function updatePerformanceDisplay() {
    const perfElement = document.getElementById('performance-metrics');
    if (perfElement && window.gollamaAPI) {
        const metrics = window.gollamaAPI.getPerformanceMetrics();
        const successRate = metrics.totalRequests > 0 ?
            (metrics.successfulRequests / metrics.totalRequests * 100).toFixed(1) : '0';
        const cacheHitRate = (metrics.cacheHits + metrics.cacheMisses) > 0 ?
            (metrics.cacheHits / (metrics.cacheHits + metrics.cacheMisses) * 100).toFixed(1) : '0';

        perfElement.innerHTML = `
            <div class="text-xs text-gray-500 dark:text-gray-400">
                Requests: ${metrics.totalRequests} |
                Success: ${successRate}% |
                Cache: ${cacheHitRate}% |
                Avg: ${metrics.averageResponseTime.toFixed(0)}ms
            </div>
        `;
    }
}

// Debounced refresh function
function debouncedRefresh(refreshFunction, delay = REFRESH_DEBOUNCE_DELAY) {
    if (refreshDebounceTimer) {
        clearTimeout(refreshDebounceTimer);
    }

    refreshDebounceTimer = setTimeout(() => {
        refreshFunction();
        refreshDebounceTimer = null;
    }, delay);
}

// Navigation functions
async function showModels(forceRefresh = false) {
    currentView = 'models';
    updateNavigation();

    // Debounce rapid refresh requests
    if (!forceRefresh && Date.now() - lastRefreshTime < MIN_REFRESH_INTERVAL) {
        console.log('⏳ Skipping refresh due to rate limiting');
        return;
    }

    try {
        showLoadingState('Loading models...');

        const models = await window.gollamaAPI.handleAPICall('GetModels',
            () => window.gollamaAPI.getModels(!forceRefresh), // Use cache unless forced
            {
                showLoading: false,
                showSuccess: false,
                showError: true,
                errorTitle: 'Failed to Load Models',
                loadingMessage: 'Loading models...'
            }
        );

        currentModels = models || [];
        renderModels(currentModels);
        lastRefreshTime = Date.now();

        // Update performance metrics display if visible
        updatePerformanceDisplay();

    } catch (error) {
        console.error('Failed to load models:', error);
        renderErrorState('Error Loading Models', error.message || 'Unknown error occurred', 'showModels(true)');
    } finally {
        hideLoadingState();
    }
}

async function showRunning(forceRefresh = false) {
    currentView = 'running';
    updateNavigation();

    // Debounce rapid refresh requests
    if (!forceRefresh && Date.now() - lastRefreshTime < MIN_REFRESH_INTERVAL) {
        console.log('⏳ Skipping refresh due to rate limiting');
        return;
    }

    try {
        showLoadingState('Loading running models...');

        const runningModels = await window.gollamaAPI.handleAPICall('GetRunningModels',
            () => window.gollamaAPI.getRunningModels(!forceRefresh), // Use cache unless forced
            {
                showLoading: false,
                showSuccess: false,
                showError: true,
                errorTitle: 'Failed to Load Running Models',
                loadingMessage: 'Loading running models...'
            }
        );

        currentRunningModels = runningModels || [];
        renderRunningModels(currentRunningModels);
        lastRefreshTime = Date.now();

        // Update performance metrics display if visible
        updatePerformanceDisplay();

    } catch (error) {
        console.error('Failed to load running models:', error);
        renderErrorState('Error Loading Running Models', error.message || 'Unknown error occurred', 'showRunning(true)');
    } finally {
        hideLoadingState();
    }
}

async function showSettings() {
    try {
        const settings = await window.gollamaAPI.handleAPICall('GetConfig',
            () => window.gollamaAPI.getConfig(),
            {
                showLoading: false,
                showSuccess: false,
                showError: true,
                errorTitle: 'Failed to Load Settings',
                loadingMessage: 'Loading settings...'
            }
        );

        renderSettings(settings);
    } catch (error) {
        console.error('Failed to load settings:', error);
        window.gollamaAPI.showError('Settings Error', 'Unable to load settings. Please try again.');
    }
}

// Error state rendering
function renderErrorState(title, message, retryFunction) {
    document.getElementById('main-content').innerHTML = `
        <div class="text-center py-12">
            <div class="mb-6">
                <svg class="mx-auto h-16 w-16 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                </svg>
            </div>
            <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-4">${title}</h2>
            <p class="text-gray-600 dark:text-gray-400 mb-6">${message}</p>
            <div class="space-y-4">
                <button class="btn-primary" onclick="${retryFunction}">Retry</button>
                <button class="px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200" onclick="showDiagnostics()">Show Diagnostics</button>
            </div>
        </div>
    `;
}

// Render functions
function renderModels(models) {
    const content = `
        <div class="mb-6">
            <div class="sm:flex sm:items-center sm:justify-between">
                <h2 class="text-2xl font-bold text-gray-900 dark:text-white">Models</h2>
                <div class="mt-4 sm:mt-0 sm:flex sm:space-x-4">
                    <input type="text" id="search-input"
                           placeholder="Search models..."
                           class="rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm focus:border-gollama-primary focus:ring-gollama-primary"
                           onkeyup="filterModels(this.value)">
                    <select id="sort-select" class="rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                            onchange="sortModels(this.value)">
                        <option value="name">Name</option>
                        <option value="size">Size</option>
                        <option value="modified">Modified</option>
                        <option value="family">Family</option>
                    </select>
                </div>
            </div>
        </div>
        <div id="model-grid" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
            ${models.map(model => renderModelCard(model)).join('')}
        </div>
    `;

    document.getElementById('main-content').innerHTML = content;
}

function renderModelCard(model) {
    return `
        <div class="model-card p-6 ${model.IsRunning ? 'ring-2 ring-green-500' : ''}"
             data-model-name="${model.Name}"
             data-model-size="${model.Size}"
             data-model-family="${model.Family}"
             data-model-modified="${new Date(model.Modified).getTime()}">
            <div class="flex justify-between items-start mb-4">
                <h3 class="text-lg font-semibold text-gray-900 dark:text-white truncate" title="${model.Name}">${model.Name}</h3>
                <span class="text-sm text-gray-500 dark:text-gray-400">${model.SizeFormatted}</span>
            </div>
            <div class="space-y-2 mb-4">
                <div class="flex justify-between text-sm">
                    <span class="text-gray-600 dark:text-gray-400">Family:</span>
                    <span class="text-gray-900 dark:text-white">${model.Family || 'Unknown'}</span>
                </div>
                <div class="flex justify-between text-sm">
                    <span class="text-gray-600 dark:text-gray-400">Quantisation:</span>
                    <span class="text-gray-900 dark:text-white">${model.QuantizationLevel || 'Unknown'}</span>
                </div>
                <div class="flex justify-between text-sm">
                    <span class="text-gray-600 dark:text-gray-400">Modified:</span>
                    <span class="text-gray-900 dark:text-white">${model.ModifiedFormatted}</span>
                </div>
            </div>
            <div class="flex space-x-2">
                <button class="btn-primary flex-1 ${model.IsRunning ? 'opacity-50 cursor-not-allowed' : ''}"
                        ${!model.IsRunning ? `onclick="runModel('${model.Name}')"` : ''}>
                    ${model.IsRunning ? 'Running' : 'Run'}
                </button>
                <button class="px-3 py-2 text-sm bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600"
                        onclick="showModelDetails('${model.Name}')">
                    Details
                </button>
                <button class="btn-danger px-3 py-2 text-sm"
                        onclick="deleteModel('${model.Name}')">
                    Delete
                </button>
            </div>
        </div>
    `;
}

function renderRunningModels(runningModels) {
    let content = `
        <div class="mb-6">
            <h2 class="text-2xl font-bold text-gray-900 dark:text-white">Running Models</h2>
            <p class="text-gray-600 dark:text-gray-400">Currently loaded models in Ollama</p>
        </div>
        <div id="running-models-list" class="space-y-4">
    `;

    if (runningModels.length === 0) {
        content += `
            <div class="text-center py-12">
                <div class="text-gray-500 dark:text-gray-400 text-lg">No models currently running</div>
                <p class="text-gray-400 dark:text-gray-500 mt-2">Load a model to see it here</p>
                <button class="btn-primary mt-4" onclick="showModels()">View All Models</button>
            </div>
        `;
    } else {
        content += runningModels.map(model => `
            <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
                <div class="flex justify-between items-center">
                    <div class="flex-1">
                        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">${model.Name}</h3>
                        <div class="mt-2 grid grid-cols-3 gap-4 text-sm">
                            <div>
                                <span class="text-gray-600 dark:text-gray-400">vRAM Usage:</span>
                                <span class="text-gray-900 dark:text-white font-medium">${model.VRAMFormatted}</span>
                            </div>
                            <div>
                                <span class="text-gray-600 dark:text-gray-400">Loaded:</span>
                                <span class="text-gray-900 dark:text-white font-medium">${model.LoadedAtFormatted}</span>
                            </div>
                            <div>
                                <span class="text-gray-600 dark:text-gray-400">Expires:</span>
                                <span class="text-gray-900 dark:text-white font-medium">${model.ExpiresAtFormatted}</span>
                            </div>
                        </div>
                    </div>
                    <div class="ml-4">
                        <button class="btn-danger" onclick="unloadModel('${model.Name}')">Unload</button>
                    </div>
                </div>
            </div>
        `).join('');
    }

    content += '</div>';
    document.getElementById('main-content').innerHTML = content;
}

function renderSettings(settings) {
    const modalContent = `
        <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onclick="closeModal()">
            <div class="bg-white dark:bg-gray-800 rounded-lg max-w-md w-full mx-4" onclick="event.stopPropagation()">
                <div class="p-6">
                    <div class="flex justify-between items-center mb-6">
                        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">Settings</h2>
                        <button class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 text-2xl" onclick="closeModal()">×</button>
                    </div>
                    <form id="settings-form" onsubmit="saveSettings(event)">
                        <div class="space-y-6">
                            <div>
                                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Ollama API URL</label>
                                <input type="url" name="ollama_api_url" id="ollama_api_url" value="${settings.OllamaAPIURL}" class="w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm focus:border-gollama-primary focus:ring-gollama-primary">
                            </div>
                            <div>
                                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Theme</label>
                                <select name="theme" id="theme" class="w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm focus:border-gollama-primary focus:ring-gollama-primary">
                                    <option value="system" ${settings.Theme === 'system' ? 'selected' : ''}>System</option>
                                    <option value="light" ${settings.Theme === 'light' ? 'selected' : ''}>Light</option>
                                    <option value="dark" ${settings.Theme === 'dark' ? 'selected' : ''}>Dark</option>
                                </select>
                            </div>
                            <div class="flex items-center">
                                <input type="checkbox" name="auto_refresh" id="auto_refresh" value="true" ${settings.AutoRefresh ? 'checked' : ''} class="h-4 w-4 text-gollama-primary focus:ring-gollama-primary border-gray-300 dark:border-gray-600 rounded">
                                <label for="auto_refresh" class="ml-2 block text-sm text-gray-700 dark:text-gray-300">Auto-refresh model list</label>
                            </div>
                            <div>
                                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Refresh Interval (seconds)</label>
                                <input type="number" name="refresh_interval" id="refresh_interval" min="5" max="300" value="${settings.RefreshInterval}" class="w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm focus:border-gollama-primary focus:ring-gollama-primary">
                            </div>
                        </div>
                        <div class="flex space-x-4 pt-6 mt-6 border-t border-gray-200 dark:border-gray-700">
                            <button type="submit" class="btn-primary flex-1">Save Settings</button>
                            <button type="button" class="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700" onclick="closeModal()">Cancel</button>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    `;

    document.getElementById('modal-container').innerHTML = modalContent;
}

// Model operations
async function runModel(name) {
    if (!name) {
        window.gollamaAPI.showError('Invalid Request', 'Model name is required');
        return;
    }

    const loadingId = showLoadingState(`Starting ${name}...`);

    try {
        // Show immediate feedback
        const modelCard = document.querySelector(`[data-model-name="${name}"]`);
        if (modelCard) {
            const button = modelCard.querySelector('.btn-primary');
            if (button) {
                button.disabled = true;
                button.textContent = 'Starting...';
                button.classList.add('opacity-50', 'cursor-not-allowed');
            }
        }

        await window.gollamaAPI.handleAPICall('RunModel',
            () => window.gollamaAPI.runModel(name),
            {
                showLoading: false,
                successTitle: 'Model Started',
                successMessage: `${name} is now running`,
                errorTitle: 'Failed to Start Model',
                loadingMessage: `Starting ${name}...`
            }
        );

        // Use debounced refresh to prevent excessive API calls
        debouncedRefresh(() => {
            if (currentView === 'models') {
                showModels(true); // Force refresh to get updated state
            } else if (currentView === 'running') {
                showRunning(true);
            }
        });

    } catch (error) {
        console.error('Failed to run model:', error);

        // Reset button state on error
        const modelCard = document.querySelector(`[data-model-name="${name}"]`);
        if (modelCard) {
            const button = modelCard.querySelector('.btn-primary');
            if (button) {
                button.disabled = false;
                button.textContent = 'Run';
                button.classList.remove('opacity-50', 'cursor-not-allowed');
            }
        }
    } finally {
        hideLoadingState(loadingId);
    }
}

async function deleteModel(name) {
    if (!name) {
        window.gollamaAPI.showError('Invalid Request', 'Model name is required');
        return;
    }

    // Enhanced confirmation dialog
    const confirmed = await showConfirmDialog(
        'Delete Model',
        `Are you sure you want to delete "${name}"? This action cannot be undone.`,
        'Delete',
        'Cancel',
        'danger'
    );

    if (!confirmed) {
        return;
    }

    const loadingId = showLoadingState(`Deleting ${name}...`);

    try {
        // Show immediate feedback
        const modelCard = document.querySelector(`[data-model-name="${name}"]`);
        if (modelCard) {
            modelCard.style.opacity = '0.5';
            modelCard.style.pointerEvents = 'none';
        }

        await window.gollamaAPI.handleAPICall('DeleteModel',
            () => window.gollamaAPI.deleteModel(name),
            {
                showLoading: false,
                successTitle: 'Model Deleted',
                successMessage: `${name} has been deleted`,
                errorTitle: 'Failed to Delete Model',
                loadingMessage: `Deleting ${name}...`
            }
        );

        // Remove model card immediately for better UX
        if (modelCard) {
            modelCard.remove();
        }

        // Use debounced refresh
        debouncedRefresh(() => {
            if (currentView === 'models') {
                showModels(true);
            }
        });

    } catch (error) {
        console.error('Failed to delete model:', error);

        // Reset model card state on error
        if (modelCard) {
            modelCard.style.opacity = '1';
            modelCard.style.pointerEvents = 'auto';
        }
    } finally {
        hideLoadingState(loadingId);
    }
}

async function unloadModel(name) {
    if (!name) {
        window.gollamaAPI.showError('Invalid Request', 'Model name is required');
        return;
    }

    try {
        showLoadingState(`Unloading ${name}...`);

        await window.gollamaAPI.handleAPICall('UnloadModel',
            () => window.gollamaAPI.unloadModel(name),
            {
                showLoading: false,
                successTitle: 'Model Unloaded',
                successMessage: `${name} has been unloaded`,
                errorTitle: 'Failed to Unload Model',
                loadingMessage: `Unloading ${name}...`
            }
        );

        // Refresh current view
        if (currentView === 'running') {
            await showRunning();
        } else if (currentView === 'models') {
            await showModels();
        }
    } catch (error) {
        console.error('Failed to unload model:', error);
        // Error already handled by handleAPICall
    } finally {
        hideLoadingState();
    }
}

async function showModelDetails(name) {
    if (!name) {
        window.gollamaAPI.showError('Invalid Request', 'Model name is required');
        return;
    }

    try {
        showLoadingState(`Loading details for ${name}...`);

        const details = await window.gollamaAPI.handleAPICall('GetModelDetails',
            () => window.gollamaAPI.getModelDetails(name),
            {
                showLoading: false,
                showSuccess: false,
                showError: true,
                errorTitle: 'Failed to Load Model Details',
                loadingMessage: `Loading details for ${name}...`
            }
        );

        renderModelDetails(details);
    } catch (error) {
        console.error('Failed to load model details:', error);
        window.gollamaAPI.showError('Model Details Error', `Unable to load details for ${name}. Please try again.`);
    } finally {
        hideLoadingState();
    }
}

function renderModelDetails(details) {
    const modalContent = `
        <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onclick="closeModal()">
            <div class="bg-white dark:bg-gray-800 rounded-lg max-w-4xl w-full mx-4 max-h-[90vh] overflow-y-auto" onclick="event.stopPropagation()">
                <div class="p-6">
                    <div class="flex justify-between items-center mb-6">
                        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">${details.Model.Name}</h2>
                        <button class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 text-2xl" onclick="closeModal()">×</button>
                    </div>
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
                        <div class="space-y-4">
                            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">Basic Information</h3>
                            <div class="space-y-2">
                                <div class="flex justify-between">
                                    <span class="text-gray-600 dark:text-gray-400">Size:</span>
                                    <span class="text-gray-900 dark:text-white font-medium">${details.Model.SizeFormatted}</span>
                                </div>
                                <div class="flex justify-between">
                                    <span class="text-gray-600 dark:text-gray-400">Family:</span>
                                    <span class="text-gray-900 dark:text-white font-medium">${details.Model.Family || 'Unknown'}</span>
                                </div>
                                <div class="flex justify-between">
                                    <span class="text-gray-600 dark:text-gray-400">Parameters:</span>
                                    <span class="text-gray-900 dark:text-white font-medium">${details.Model.ParameterSize || 'Unknown'}</span>
                                </div>
                                <div class="flex justify-between">
                                    <span class="text-gray-600 dark:text-gray-400">Quantisation:</span>
                                    <span class="text-gray-900 dark:text-white font-medium">${details.Model.QuantizationLevel || 'Unknown'}</span>
                                </div>
                            </div>
                        </div>
                        <div class="space-y-4">
                            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">Model Status</h3>
                            <div class="space-y-2">
                                <div class="flex justify-between">
                                    <span class="text-gray-600 dark:text-gray-400">Status:</span>
                                    <span class="px-2 py-1 rounded-full text-xs font-medium ${details.Model.IsRunning ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'}">
                                        ${details.Model.IsRunning ? 'Running' : 'Stopped'}
                                    </span>
                                </div>
                                <div class="flex justify-between">
                                    <span class="text-gray-600 dark:text-gray-400">Modified:</span>
                                    <span class="text-gray-900 dark:text-white font-medium">${details.Model.ModifiedFormatted}</span>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="flex space-x-4 pt-6 border-t border-gray-200 dark:border-gray-700">
                        <button class="btn-primary" onclick="runModel('${details.Model.Name}'); closeModal();">Run Model</button>
                        <button class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg" onclick="showVRAMEstimator('${details.Model.Name}')">Estimate vRAM</button>
                        <button class="btn-danger" onclick="deleteModel('${details.Model.Name}'); closeModal();">Delete</button>
                    </div>
                </div>
            </div>
        </div>
    `;

    document.getElementById('modal-container').innerHTML = modalContent;
}

// Settings operations
async function saveSettings(event) {
    event.preventDefault();

    const formData = new FormData(event.target);
    const settings = {
        OllamaAPIURL: formData.get('ollama_api_url') || '',
        Theme: formData.get('theme') || 'dark',
        AutoRefresh: formData.has('auto_refresh'),
        RefreshInterval: parseInt(formData.get('refresh_interval') || '30'),
        WindowWidth: 1200,
        WindowHeight: 800,
        DefaultView: 'models',
        ShowSystemTray: false
    };

    try {
        showLoadingState('Saving settings...');

        await window.gollamaAPI.handleAPICall('UpdateConfig',
            () => window.gollamaAPI.updateConfig(settings),
            {
                showLoading: false,
                successTitle: 'Settings Saved',
                successMessage: 'Your settings have been saved successfully',
                errorTitle: 'Failed to Save Settings',
                loadingMessage: 'Saving settings...'
            }
        );

        // Apply theme change
        applyTheme(settings.Theme);

        // Update auto-refresh
        if (settings.AutoRefresh) {
            setupAutoRefresh(settings.RefreshInterval);
        } else {
            clearAutoRefresh();
        }

        closeModal();
    } catch (error) {
        console.error('Failed to save settings:', error);
        // Error already handled by handleAPICall
    } finally {
        hideLoadingState();
    }
}

// vRAM Estimator
function showVRAMEstimator(modelName = '') {
    const modalContent = `
        <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onclick="closeModal()">
            <div class="bg-white dark:bg-gray-800 rounded-lg max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto" onclick="event.stopPropagation()">
                <div class="p-6">
                    <div class="flex justify-between items-center mb-6">
                        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">vRAM Estimator</h2>
                        <button class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 text-2xl" onclick="closeModal()">×</button>
                    </div>
                    <form id="vram-form" onsubmit="calculateVRAM(event)">
                        <div class="space-y-4 mb-6">
                            <div>
                                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Model Name or HuggingFace ID</label>
                                <input type="text" name="model_name" id="vram_model_name" value="${modelName}" class="w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm focus:border-gollama-primary focus:ring-gollama-primary" placeholder="e.g., llama2:7b or microsoft/DialoGPT-medium">
                            </div>
                            <div>
                                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Available vRAM (GB)</label>
                                <input type="number" name="vram_gb" id="vram_gb" step="0.1" min="0" class="w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm focus:border-gollama-primary focus:ring-gollama-primary" placeholder="e.g., 8.0">
                            </div>
                            <div class="grid grid-cols-2 gap-4">
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Context Length</label>
                                    <select name="context" id="vram_context" class="w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm focus:border-gollama-primary focus:ring-gollama-primary">
                                        <option value="">Auto</option>
                                        <option value="2048">2K</option>
                                        <option value="4096">4K</option>
                                        <option value="8192">8K</option>
                                        <option value="16384">16K</option>
                                        <option value="32768">32K</option>
                                    </select>
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Quantisation</label>
                                    <select name="quantization" id="vram_quantization" class="w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm focus:border-gollama-primary focus:ring-gollama-primary">
                                        <option value="">Auto</option>
                                        <option value="Q4_0">Q4_0</option>
                                        <option value="Q4_1">Q4_1</option>
                                        <option value="Q5_0">Q5_0</option>
                                        <option value="Q5_1">Q5_1</option>
                                        <option value="Q8_0">Q8_0</option>
                                        <option value="F16">F16</option>
                                        <option value="F32">F32</option>
                                    </select>
                                </div>
                            </div>
                        </div>
                        <button type="submit" class="btn-primary w-full mb-6">Calculate vRAM Requirements</button>
                    </form>
                    <div id="vram-results"></div>
                </div>
            </div>
        </div>
    `;

    document.getElementById('modal-container').innerHTML = modalContent;
}

async function calculateVRAM(event) {
    event.preventDefault();

    const formData = new FormData(event.target);
    const request = {
        modelName: formData.get('model_name')?.toString() || '',
        vramAvailable: parseFloat(formData.get('vram_gb')?.toString() || '0'),
        contextLength: parseInt(formData.get('context')?.toString() || '0'),
        quantization: formData.get('quantization')?.toString() || ''
    };

    if (!request.modelName) {
        window.gollamaAPI.showError('Invalid Request', 'Model name is required for vRAM estimation');
        return;
    }

    try {
        showLoadingState('Calculating vRAM requirements...');

        const result = await window.gollamaAPI.handleAPICall('EstimateVRAM',
            () => window.gollamaAPI.estimateVRAM(request),
            {
                showLoading: false,
                showSuccess: false,
                showError: true,
                errorTitle: 'vRAM Calculation Failed',
                loadingMessage: 'Calculating vRAM requirements...'
            }
        );

        renderVRAMResults(result);
    } catch (error) {
        console.error('Failed to calculate vRAM:', error);
        window.gollamaAPI.showError('vRAM Estimation Error', 'Unable to calculate vRAM requirements. Please check your inputs and try again.');
    } finally {
        hideLoadingState();
    }
}

// Loading state management
function showLoadingState(message = 'Loading...') {
    const loadingOverlay = document.getElementById('loading-overlay');
    if (loadingOverlay) {
        loadingOverlay.querySelector('.loading-message').textContent = message;
        loadingOverlay.classList.remove('hidden');
    } else {
        // Create loading overlay if it doesn't exist
        const overlay = document.createElement('div');
        overlay.id = 'loading-overlay';
        overlay.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50';
        overlay.innerHTML = `
            <div class="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-sm mx-4">
                <div class="flex items-center space-x-4">
                    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-gollama-primary"></div>
                    <span class="loading-message text-gray-900 dark:text-white">${message}</span>
                </div>
            </div>
        `;
        document.body.appendChild(overlay);
    }
}

function hideLoadingState() {
    const loadingOverlay = document.getElementById('loading-overlay');
    if (loadingOverlay) {
        loadingOverlay.classList.add('hidden');
    }
}

// Diagnostic functions
async function showDiagnostics() {
    try {
        const diagnostics = await window.gollamaAPI.getDiagnostics();

        const modalContent = `
            <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onclick="closeModal()">
                <div class="bg-white dark:bg-gray-800 rounded-lg max-w-4xl w-full mx-4 max-h-[90vh] overflow-y-auto" onclick="event.stopPropagation()">
                    <div class="p-6">
                        <div class="flex justify-between items-center mb-6">
                            <h2 class="text-2xl font-bold text-gray-900 dark:text-white">System Diagnostics</h2>
                            <button class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 text-2xl" onclick="closeModal()">×</button>
                        </div>
                        <div class="space-y-6">
                            <div>
                                <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-3">API Status</h3>
                                <div class="bg-gray-50 dark:bg-gray-700 rounded-lg p-4">
                                    <div class="grid grid-cols-2 gap-4 text-sm">
                                        <div>
                                            <span class="text-gray-600 dark:text-gray-400">Initialized:</span>
                                            <span class="ml-2 px-2 py-1 rounded text-xs ${diagnostics.initialized ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'}">
                                                ${diagnostics.initialized ? 'Yes' : 'No'}
                                            </span>
                                        </div>
                                        <div>
                                            <span class="text-gray-600 dark:text-gray-400">Wails Runtime:</span>
                                            <span class="ml-2 px-2 py-1 rounded text-xs ${diagnostics.wailsRuntime ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'}">
                                                ${diagnostics.wailsRuntime ? 'Available' : 'Not Available'}
                                            </span>
                                        </div>
                                    </div>
                                </div>
                            </div>
                            <div>
                                <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-3">Available Methods (${diagnostics.availableMethods.length})</h3>
                                <div class="bg-gray-50 dark:bg-gray-700 rounded-lg p-4 max-h-40 overflow-y-auto">
                                    <div class="grid grid-cols-2 gap-2 text-sm">
                                        ${diagnostics.availableMethods.map(method => `
                                            <div class="text-gray-900 dark:text-white">✓ ${method}</div>
                                        `).join('')}
                                    </div>
                                </div>
                            </div>
                            ${diagnostics.serviceStatus ? `
                                <div>
                                    <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-3">Service Status</h3>
                                    <div class="bg-gray-50 dark:bg-gray-700 rounded-lg p-4">
                                        <pre class="text-sm text-gray-900 dark:text-white overflow-x-auto">${JSON.stringify(diagnostics.serviceStatus, null, 2)}</pre>
                                    </div>
                                </div>
                            ` : ''}
                        </div>
                        <div class="flex space-x-4 pt-6 mt-6 border-t border-gray-200 dark:border-gray-700">
                            <button class="btn-primary" onclick="window.gollamaAPI.logDiagnostics(); closeModal();">Log to Console</button>
                            <button class="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700" onclick="closeModal()">Close</button>
                        </div>
                    </div>
                </div>
            </div>
        `;

        document.getElementById('modal-container').innerHTML = modalContent;
    } catch (error) {
        console.error('Failed to get diagnostics:', error);
        window.gollamaAPI.showError('Diagnostics Error', 'Unable to retrieve diagnostic information');
    }
}

function renderVRAMResults(result) {
    const resultsContainer = document.getElementById('vram-results');
    if (!resultsContainer) return;

    if (result.Error) {
        resultsContainer.innerHTML = `
            <div class="bg-red-100 dark:bg-red-900 border border-red-400 dark:border-red-600 text-red-700 dark:text-red-200 px-4 py-3 rounded">
                <strong>Error:</strong> ${result.Error}
            </div>
        `;
        return;
    }

    let tableRows = '';
    if (result.Recommendations && result.Recommendations.length > 0) {
        tableRows = result.Recommendations.map(rec => `
            <tr class="${rec.Fits ? 'bg-green-50 dark:bg-green-900' : ''}">
                <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">${rec.Quantization}</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">${rec.ContextLength}</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">${rec.VRAMRequired.toFixed(1)} GB</td>
                <td class="px-6 py-4 whitespace-nowrap">
                    ${rec.Fits ?
                        '<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">✓ Yes</span>' :
                        '<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200">✗ No</span>'
                    }
                </td>
            </tr>
        `).join('');
    }

    resultsContainer.innerHTML = `
        <div class="bg-gray-50 dark:bg-gray-700 rounded-lg p-4">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">vRAM Estimation Results</h3>
            <div class="space-y-4">
                <div class="text-sm text-gray-600 dark:text-gray-400">
                    Model: <span class="font-medium text-gray-900 dark:text-white">${result.ModelName}</span>
                </div>
                ${tableRows ? `
                    <div class="overflow-x-auto">
                        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-600">
                            <thead class="bg-gray-50 dark:bg-gray-800">
                                <tr>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Quantisation</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Context</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">vRAM Required</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Fits</th>
                                </tr>
                            </thead>
                            <tbody class="bg-white dark:bg-gray-700 divide-y divide-gray-200 dark:divide-gray-600">
                                ${tableRows}
                            </tbody>
                        </table>
                    </div>
                ` : '<p class="text-gray-500 dark:text-gray-400">No recommendations available</p>'}
            </div>
        </div>
    `;
}

// Utility functions
function showLoading(message = 'Loading...') {
    const loading = document.getElementById('loading');
    if (loading) {
        // Update loading message if provided
        const messageElement = loading.querySelector('span');
        if (messageElement && message) {
            messageElement.textContent = message;
        }
        loading.classList.add('show');
    }
}

function hideLoading() {
    const loading = document.getElementById('loading');
    if (loading) {
        loading.classList.remove('show');
    }
}

// Make loading functions available globally for the API
window.showLoading = showLoading;
window.hideLoading = hideLoading;

function showToast(type, title, message) {
    const container = document.getElementById('toast-container');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = `toast toast-${type} transform transition-all duration-300 translate-x-full`;

    const iconMap = {
        success: '✓',
        error: '✗',
        warning: '⚠',
        info: 'ℹ'
    };

    toast.innerHTML = `
        <div class="flex items-start space-x-3">
            <div class="toast-icon">${iconMap[type] || 'ℹ'}</div>
            <div class="flex-1">
                <div class="toast-title">${title}</div>
                <div class="toast-message">${message}</div>
            </div>
            <button class="toast-close" onclick="closeToast(this)">×</button>
        </div>
    `;

    container.appendChild(toast);

    // Animate in
    setTimeout(() => {
        toast.classList.remove('translate-x-full');
    }, 10);

    // Auto-remove after 5 seconds
    setTimeout(() => {
        closeToast(toast.querySelector('.toast-close'));
    }, 5000);
}

function closeToast(button) {
    const toast = button.closest('.toast');
    if (toast) {
        toast.classList.add('translate-x-full');
        setTimeout(() => {
            toast.remove();
        }, 300);
    }
}

function closeModal() {
    const container = document.getElementById('modal-container');
    if (container) {
        container.innerHTML = '';
        document.body.style.overflow = '';
    }
}

function updateNavigation() {
    // Update navigation active states
    const navButtons = document.querySelectorAll('.nav-link');
    navButtons.forEach(btn => {
        btn.classList.remove('active');
    });

    // Add active class to current view button
    const activeButton = document.querySelector(`[onclick="show${currentView.charAt(0).toUpperCase() + currentView.slice(1)}()"]`);
    if (activeButton) {
        activeButton.classList.add('active');
    }
}

function applyTheme(theme) {
    const body = document.body;

    if (theme === 'system') {
        // Use system preference
        const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        body.setAttribute('data-theme', prefersDark ? 'dark' : 'light');
    } else {
        body.setAttribute('data-theme', theme);
    }

    localStorage.setItem('theme', theme);
}

function setupAutoRefresh(intervalSeconds) {
    clearAutoRefresh();

    autoRefreshInterval = setInterval(async () => {
        if (currentView === 'models') {
            try {
                const models = await window.gollamaAPI.getModels();
                currentModels = models || [];
                renderModels(currentModels);
            } catch (error) {
                console.error('Auto-refresh failed:', error);
            }
        } else if (currentView === 'running') {
            try {
                const runningModels = await window.gollamaAPI.getRunningModels();
                currentRunningModels = runningModels || [];
                renderRunningModels(currentRunningModels);
            } catch (error) {
                console.error('Auto-refresh failed:', error);
            }
        }
    }, intervalSeconds * 1000);
}

function clearAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
    }
}

// Search and filter functions
function filterModels(searchTerm) {
    const cards = document.querySelectorAll('.model-card');
    const term = searchTerm.toLowerCase();

    cards.forEach(card => {
        const name = card.getAttribute('data-model-name').toLowerCase();
        const family = card.getAttribute('data-model-family').toLowerCase();

        if (name.includes(term) || family.includes(term)) {
            card.style.display = 'block';
        } else {
            card.style.display = 'none';
        }
    });
}

function sortModels(sortBy) {
    const grid = document.getElementById('model-grid');
    const cards = Array.from(grid.querySelectorAll('.model-card'));

    cards.sort((a, b) => {
        switch (sortBy) {
            case 'size':
                return parseFloat(b.getAttribute('data-model-size')) - parseFloat(a.getAttribute('data-model-size'));
            case 'modified':
                return parseInt(b.getAttribute('data-model-modified')) - parseInt(a.getAttribute('data-model-modified'));
            case 'family':
                return a.getAttribute('data-model-family').localeCompare(b.getAttribute('data-model-family'));
            case 'name':
            default:
                return a.getAttribute('data-model-name').localeCompare(b.getAttribute('data-model-name'));
        }
    });

    // Clear and re-append sorted cards
    grid.innerHTML = '';
    cards.forEach(card => grid.appendChild(card));
}

// Refresh function
async function refreshModels() {
    if (currentView === 'models') {
        await showModels();
    } else if (currentView === 'running') {
        await showRunning();
    }
}

// Keyboard shortcuts
document.addEventListener('keydown', function(e) {
    // Escape key closes modals
    if (e.key === 'Escape') {
        closeModal();
    }

    // Ctrl/Cmd + R refreshes models
    if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
        e.preventDefault();
        refreshModels();
    }

    // Number keys for quick navigation
    if (e.key === '1') {
        showModels();
    } else if (e.key === '2') {
        showRunning();
    } else if (e.key === '3') {
        showSettings();
    }
});

// Export functions for global use
window.showModels = showModels;
window.showRunning = showRunning;
window.showSettings = showSettings;
window.refreshModels = refreshModels;
window.runModel = runModel;
window.deleteModel = deleteModel;
window.unloadModel = unloadModel;
window.showModelDetails = showModelDetails;
window.showVRAMEstimator = showVRAMEstimator;
window.calculateVRAM = calculateVRAM;
window.saveSettings = saveSettings;
window.filterModels = filterModels;
window.sortModels = sortModels;
window.closeModal = closeModal;
window.closeToast = closeToast;

// ===========================================
// Utility Functions
// ===========================================

// Close modal function
function closeModal() {
    const modalContainer = document.getElementById('modal-container');
    if (modalContainer) {
        modalContainer.innerHTML = '';
    }
}

// Navigation update function
function updateNavigation() {
    // Update navigation active states
    const navItems = document.querySelectorAll('[data-nav]');
    navItems.forEach(item => {
        const navType = item.getAttribute('data-nav');
        if (navType === currentView) {
            item.classList.add('active');
        } else {
            item.classList.remove('active');
        }
    });
}

// Refresh current view
async function refreshModels() {
    try {
        showLoadingState('Refreshing...');

        if (currentView === 'models') {
            await showModels();
        } else if (currentView === 'running') {
            await showRunning();
        } else {
            await showModels(); // Default to models view
        }

        window.gollamaAPI.showSuccess('Refreshed', 'Data has been refreshed successfully');
    } catch (error) {
        console.error('Failed to refresh:', error);
        window.gollamaAPI.showError('Refresh Failed', 'Unable to refresh data. Please try again.');
    } finally {
        hideLoadingState();
    }
}

// Theme application
function applyTheme(theme) {
    const html = document.documentElement;

    if (theme === 'dark' || (theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
        html.classList.add('dark');
    } else {
        html.classList.remove('dark');
    }

    console.log(`Applied theme: ${theme}`);
}

// Auto-refresh functionality
function setupAutoRefresh(intervalSeconds) {
    clearAutoRefresh();

    if (intervalSeconds > 0) {
        autoRefreshInterval = setInterval(async () => {
            console.log('Auto-refreshing data...');

            try {
                if (currentView === 'models') {
                    await showModels();
                } else if (currentView === 'running') {
                    await showRunning();
                }
            } catch (error) {
                console.warn('Auto-refresh failed:', error);
            }
        }, intervalSeconds * 1000);

        console.log(`Auto-refresh enabled: ${intervalSeconds}s interval`);
    }
}

function clearAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
        console.log('Auto-refresh disabled');
    }
}

// Model filtering and sorting
function filterModels(searchTerm) {
    const modelCards = document.querySelectorAll('.model-card');
    const term = searchTerm.toLowerCase();

    modelCards.forEach(card => {
        const modelName = card.getAttribute('data-model-name').toLowerCase();
        const modelFamily = card.getAttribute('data-model-family').toLowerCase();

        if (modelName.includes(term) || modelFamily.includes(term)) {
            card.style.display = 'block';
        } else {
            card.style.display = 'none';
        }
    });
}

function sortModels(sortBy) {
    const modelGrid = document.getElementById('model-grid');
    if (!modelGrid) return;

    const modelCards = Array.from(modelGrid.querySelectorAll('.model-card'));

    modelCards.sort((a, b) => {
        switch (sortBy) {
            case 'name':
                return a.getAttribute('data-model-name').localeCompare(b.getAttribute('data-model-name'));
            case 'size':
                return parseInt(b.getAttribute('data-model-size')) - parseInt(a.getAttribute('data-model-size'));
            case 'modified':
                return parseInt(b.getAttribute('data-model-modified')) - parseInt(a.getAttribute('data-model-modified'));
            case 'family':
                return a.getAttribute('data-model-family').localeCompare(b.getAttribute('data-model-family'));
            default:
                return 0;
        }
    });

    // Re-append sorted cards
    modelCards.forEach(card => modelGrid.appendChild(card));
}

// ===========================================
// Enhanced Error Handling
// ===========================================

// Global error handler for unhandled promise rejections
window.addEventListener('unhandledrejection', function(event) {
    console.error('Unhandled promise rejection:', event.reason);

    if (window.gollamaAPI) {
        window.gollamaAPI.showError(
            'Unexpected Error',
            'An unexpected error occurred. Please check the console for details.'
        );
    }

    // Prevent the default browser error handling
    event.preventDefault();
});

// Global error handler for JavaScript errors
window.addEventListener('error', function(event) {
    console.error('JavaScript error:', event.error);

    if (window.gollamaAPI) {
        window.gollamaAPI.showError(
            'Application Error',
            'A JavaScript error occurred. Please refresh the page and try again.'
        );
    }
});

// ===========================================
// Development and Debug Functions
// ===========================================

// Debug function to test API connectivity
window.testAPI = async function() {
    console.log('=== API CONNECTIVITY TEST ===');

    if (!window.gollamaAPI) {
        console.error('❌ Gollama API not available');
        return;
    }

    if (!window.gollamaAPI.initialized) {
        console.error('❌ Gollama API not initialized');
        return;
    }

    const tests = [
        { name: 'Health Check', fn: () => window.gollamaAPI.healthCheck() },
        { name: 'Get Models', fn: () => window.gollamaAPI.getModels() },
        { name: 'Get Config', fn: () => window.gollamaAPI.getConfig() },
        { name: 'Get Running Models', fn: () => window.gollamaAPI.getRunningModels() }
    ];

    for (const test of tests) {
        try {
            console.log(`🔄 Testing ${test.name}...`);
            const result = await test.fn();
            console.log(`✅ ${test.name} passed:`, result);
        } catch (error) {
            console.error(`❌ ${test.name} failed:`, error);
        }
    }

    console.log('=== TEST COMPLETE ===');
};

// Debug function to show API diagnostics
window.showAPIDiagnostics = function() {
    if (window.gollamaAPI) {
        window.gollamaAPI.logDiagnostics();
    } else {
        console.error('Gollama API not available');
    }
};

console.log('✅ Gollama GUI JavaScript enhanced with improved API integration');
// Diagnostics functions
async function showDiagnostics() {
    console.log('🔍 Showing diagnostics page...');
    currentView = 'diagnostics';

    try {
        // Create diagnostics page content
        const diagnosticsHTML = `
            <div class="max-w-6xl mx-auto">
                <!-- Header -->
                <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 mb-6">
                    <div class="flex items-center justify-between">
                        <div>
                            <h1 class="text-2xl font-bold text-gray-900">🔍 Gollama Diagnostics</h1>
                            <p class="text-gray-600 mt-1">Comprehensive service binding and health diagnostics</p>
                        </div>
                        <div class="flex space-x-3">
                            <button id="runQuickDiagBtn" class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors">
                                🧪 Quick Test
                            </button>
                            <button id="runFullDiagBtn" class="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors">
                                🔍 Full Diagnostics
                            </button>
                            <button id="exportDiagBtn" class="px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors">
                                📄 Export Report
                            </button>
                        </div>
                    </div>
                </div>

                <!-- Quick Status Cards -->
                <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
                    <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
                        <div class="flex items-center">
                            <div class="flex-shrink-0">
                                <div id="apiStatusIcon" class="w-8 h-8 rounded-full bg-gray-300 flex items-center justify-center">
                                    <span class="text-sm">⏳</span>
                                </div>
                            </div>
                            <div class="ml-3">
                                <p class="text-sm font-medium text-gray-900">API Status</p>
                                <p id="apiStatusText" class="text-sm text-gray-500">Checking...</p>
                            </div>
                        </div>
                    </div>

                    <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
                        <div class="flex items-center">
                            <div class="flex-shrink-0">
                                <div id="serviceStatusIcon" class="w-8 h-8 rounded-full bg-gray-300 flex items-center justify-center">
                                    <span class="text-sm">⏳</span>
                                </div>
                            </div>
                            <div class="ml-3">
                                <p class="text-sm font-medium text-gray-900">Service Status</p>
                                <p id="serviceStatusText" class="text-sm text-gray-500">Checking...</p>
                            </div>
                        </div>
                    </div>

                    <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
                        <div class="flex items-center">
                            <div class="flex-shrink-0">
                                <div id="methodsStatusIcon" class="w-8 h-8 rounded-full bg-gray-300 flex items-center justify-center">
                                    <span class="text-sm">⏳</span>
                                </div>
                            </div>
                            <div class="ml-3">
                                <p class="text-sm font-medium text-gray-900">Methods</p>
                                <p id="methodsStatusText" class="text-sm text-gray-500">Checking...</p>
                            </div>
                        </div>
                    </div>

                    <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
                        <div class="flex items-center">
                            <div class="flex-shrink-0">
                                <div id="testsStatusIcon" class="w-8 h-8 rounded-full bg-gray-300 flex items-center justify-center">
                                    <span class="text-sm">⏳</span>
                                </div>
                            </div>
                            <div class="ml-3">
                                <p class="text-sm font-medium text-gray-900">Tests</p>
                                <p id="testsStatusText" class="text-sm text-gray-500">Not Run</p>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Console Output -->
                <div class="bg-white rounded-lg shadow-sm border border-gray-200 mb-6">
                    <div class="px-6 py-4 border-b border-gray-200">
                        <h2 class="text-lg font-semibold text-gray-900">📟 Console Output</h2>
                        <p class="text-sm text-gray-600">Real-time diagnostic output (also check browser console)</p>
                    </div>
                    <div class="p-6">
                        <div id="consoleOutput" class="bg-gray-900 text-green-400 p-4 rounded-lg font-mono text-sm h-64 overflow-y-auto">
                            <div class="text-gray-500">Diagnostic output will appear here...</div>
                        </div>
                    </div>
                </div>

                <!-- Instructions -->
                <div class="bg-blue-50 border border-blue-200 rounded-lg p-6">
                    <h3 class="text-lg font-semibold text-blue-900 mb-3">🔍 How to Use Diagnostics</h3>
                    <div class="space-y-2 text-blue-800">
                        <p><strong>Quick Test:</strong> Runs basic connectivity and service binding tests</p>
                        <p><strong>Full Diagnostics:</strong> Comprehensive analysis of all system components</p>
                        <p><strong>Browser Console:</strong> Open Developer Tools (F12) for detailed logs and interactive testing</p>
                        <p><strong>Console Commands:</strong> Try <code class="bg-blue-100 px-1 rounded">await runQuickDiagnostics()</code> or <code class="bg-blue-100 px-1 rounded">diagnosticsHelp()</code></p>
                    </div>
                </div>
            </div>
        `;

        // Update main content
        document.getElementById('main-content').innerHTML = diagnosticsHTML;

        // Set up event listeners
        document.getElementById('runQuickDiagBtn').addEventListener('click', runQuickDiagnosticsUI);
        document.getElementById('runFullDiagBtn').addEventListener('click', runFullDiagnosticsUI);
        document.getElementById('exportDiagBtn').addEventListener('click', exportDiagnosticsUI);

        // Run initial status check
        await updateDiagnosticStatus();

    } catch (error) {
        console.error('❌ Failed to show diagnostics:', error);
        showError('Failed to load diagnostics', error.message);
    }
}

async function updateDiagnosticStatus() {
    try {
        // API Status
        if (window.gollamaAPI && window.gollamaAPI.initialized) {
            setStatusCard('api', 'healthy', 'Initialized');
        } else {
            setStatusCard('api', 'unhealthy', 'Not Initialized');
        }

        // Service Status
        try {
            const status = await window.gollamaAPI.getServiceStatus();
            if (status.status === 'healthy') {
                setStatusCard('service', 'healthy', 'Healthy');
            } else {
                setStatusCard('service', 'unhealthy', 'Unhealthy');
            }
        } catch (error) {
            setStatusCard('service', 'unhealthy', 'Error');
        }

        // Methods Status
        if (window.gollamaAPI && window.gollamaAPI.bindingMethods) {
            const methodCount = Object.keys(window.gollamaAPI.bindingMethods).length;
            setStatusCard('methods', 'healthy', `${methodCount} Available`);
        } else {
            setStatusCard('methods', 'unhealthy', 'Unknown');
        }

    } catch (error) {
        console.error('❌ Failed to update diagnostic status:', error);
    }
}

function setStatusCard(type, status, text) {
    const icon = document.getElementById(`${type}StatusIcon`);
    const textEl = document.getElementById(`${type}StatusText`);

    if (!icon || !textEl) return;

    // Update icon and color
    if (status === 'healthy') {
        icon.className = 'w-8 h-8 rounded-full bg-green-100 flex items-center justify-center';
        icon.innerHTML = '<span class="text-sm">✅</span>';
    } else if (status === 'unhealthy') {
        icon.className = 'w-8 h-8 rounded-full bg-red-100 flex items-center justify-center';
        icon.innerHTML = '<span class="text-sm">❌</span>';
    } else {
        icon.className = 'w-8 h-8 rounded-full bg-yellow-100 flex items-center justify-center';
        icon.innerHTML = '<span class="text-sm">⚠️</span>';
    }

    textEl.textContent = text;
}

async function runQuickDiagnosticsUI() {
    const btn = document.getElementById('runQuickDiagBtn');
    const originalText = btn.textContent;

    try {
        btn.textContent = '🔄 Running...';
        btn.disabled = true;

        logToConsole('🧪 Running quick diagnostics...');

        const result = await window.gollamaDiagnostics.runQuickDiagnostics();

        logToConsole('✅ Quick diagnostics completed successfully');
        logToConsole(JSON.stringify(result, null, 2));

        // Update test status
        setStatusCard('tests', 'healthy', 'Quick Test Passed');

        showSuccess('Quick Diagnostics', 'All basic tests passed successfully');

    } catch (error) {
        logToConsole(`❌ Quick diagnostics failed: ${error.message}`);
        setStatusCard('tests', 'unhealthy', 'Quick Test Failed');
        showError('Quick Diagnostics Failed', error.message);
    } finally {
        btn.textContent = originalText;
        btn.disabled = false;
    }
}

async function runFullDiagnosticsUI() {
    const btn = document.getElementById('runFullDiagBtn');
    const originalText = btn.textContent;

    try {
        btn.textContent = '🔄 Running...';
        btn.disabled = true;

        logToConsole('🔍 Running full diagnostics...');

        const result = await window.gollamaDiagnostics.runFullDiagnostics();

        logToConsole('✅ Full diagnostics completed successfully');
        logToConsole(JSON.stringify(result, null, 2));

        // Update test status based on results
        if (result.tests && result.tests.summary) {
            const { passed, total } = result.tests.summary;
            if (passed === total) {
                setStatusCard('tests', 'healthy', `${passed}/${total} Passed`);
            } else {
                setStatusCard('tests', 'unhealthy', `${passed}/${total} Passed`);
            }
        }

        showSuccess('Full Diagnostics', 'Comprehensive diagnostics completed successfully');

    } catch (error) {
        logToConsole(`❌ Full diagnostics failed: ${error.message}`);
        setStatusCard('tests', 'unhealthy', 'Full Test Failed');
        showError('Full Diagnostics Failed', error.message);
    } finally {
        btn.textContent = originalText;
        btn.disabled = false;
    }
}

async function exportDiagnosticsUI() {
    const btn = document.getElementById('exportDiagBtn');
    const originalText = btn.textContent;

    try {
        btn.textContent = '📄 Exporting...';
        btn.disabled = true;

        logToConsole('💾 Exporting diagnostic data...');

        await window.gollamaDiagnostics.exportDiagnostics();

        logToConsole('✅ Diagnostic data exported successfully');
        showSuccess('Export Complete', 'Diagnostic report has been downloaded');

    } catch (error) {
        logToConsole(`❌ Export failed: ${error.message}`);
        showError('Export Failed', error.message);
    } finally {
        btn.textContent = originalText;
        btn.disabled = false;
    }
}

function logToConsole(message) {
    const consoleOutput = document.getElementById('consoleOutput');
    if (!consoleOutput) return;

    const timestamp = new Date().toLocaleTimeString();
    const logLine = document.createElement('div');
    logLine.textContent = `[${timestamp}] ${message}`;

    // Add color based on message type
    if (message.includes('✅')) {
        logLine.className = 'text-green-400';
    } else if (message.includes('❌')) {
        logLine.className = 'text-red-400';
    } else if (message.includes('⚠️')) {
        logLine.className = 'text-yellow-400';
    } else if (message.includes('🔍') || message.includes('🧪')) {
        logLine.className = 'text-blue-400';
    }

    consoleOutput.appendChild(logLine);
    consoleOutput.scrollTop = consoleOutput.scrollHeight;

    // Keep only last 100 lines
    while (consoleOutput.children.length > 100) {
        consoleOutput.removeChild(consoleOutput.firstChild);
    }
}

// Enhanced confirmation dialog
function showConfirmDialog(title, message, confirmText = 'Confirm', cancelText = 'Cancel', type = 'primary') {
    return new Promise((resolve) => {
        const modal = document.createElement('div');
        modal.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50';
        modal.innerHTML = `
            <div class="bg-white dark:bg-gray-800 rounded-lg max-w-md w-full mx-4" onclick="event.stopPropagation()">
                <div class="p-6">
                    <div class="flex items-center mb-4">
                        <div class="flex-shrink-0">
                            ${type === 'danger' ? `
                                <svg class="h-6 w-6 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"/>
                                </svg>
                            ` : `
                                <svg class="h-6 w-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                                </svg>
                            `}
                        </div>
                        <div class="ml-3">
                            <h3 class="text-lg font-medium text-gray-900 dark:text-white">${title}</h3>
                        </div>
                    </div>
                    <div class="mb-6">
                        <p class="text-sm text-gray-600 dark:text-gray-400">${message}</p>
                    </div>
                    <div class="flex space-x-4">
                        <button id="confirm-btn" class="${type === 'danger' ? 'btn-danger' : 'btn-primary'} flex-1">
                            ${confirmText}
                        </button>
                        <button id="cancel-btn" class="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 flex-1">
                            ${cancelText}
                        </button>
                    </div>
                </div>
            </div>
        `;

        document.body.appendChild(modal);

        const confirmBtn = modal.querySelector('#confirm-btn');
        const cancelBtn = modal.querySelector('#cancel-btn');

        confirmBtn.onclick = () => {
            document.body.removeChild(modal);
            resolve(true);
        };

        cancelBtn.onclick = () => {
            document.body.removeChild(modal);
            resolve(false);
        };

        modal.onclick = () => {
            document.body.removeChild(modal);
            resolve(false);
        };
    });
}

// Auto-refresh with performance optimization
function setupAutoRefresh(intervalSeconds) {
    clearAutoRefresh();

    if (intervalSeconds > 0) {
        autoRefreshInterval = setInterval(() => {
            // Only refresh if user is active and not currently loading
            if (document.visibilityState === 'visible' && loadingStates.size === 0) {
                console.log('🔄 Auto-refreshing current view...');

                if (currentView === 'models') {
                    showModels(false); // Use cache for auto-refresh
                } else if (currentView === 'running') {
                    showRunning(false);
                }
            }
        }, intervalSeconds * 1000);

        console.log(`✅ Auto-refresh enabled (${intervalSeconds}s interval)`);
    }
}

function clearAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
        console.log('🛑 Auto-refresh disabled');
    }
}

// Performance monitoring functions
async function showPerformanceMetrics() {
    try {
        const metrics = await window.gollamaAPI.getBackendPerformanceMetrics();

        const modalContent = `
            <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onclick="closeModal()">
                <div class="bg-white dark:bg-gray-800 rounded-lg max-w-4xl w-full mx-4 max-h-[90vh] overflow-y-auto" onclick="event.stopPropagation()">
                    <div class="p-6">
                        <div class="flex justify-between items-center mb-6">
                            <h2 class="text-2xl font-bold text-gray-900 dark:text-white">Performance Metrics</h2>
                            <button class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 text-2xl" onclick="closeModal()">×</button>
                        </div>

                        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                            <div class="space-y-4">
                                <h3 class="text-lg font-semibold text-gray-900 dark:text-white">Frontend Metrics</h3>
                                <div class="bg-gray-50 dark:bg-gray-700 p-4 rounded-lg">
                                    <div class="space-y-2 text-sm">
                                        <div class="flex justify-between">
                                            <span>Total Requests:</span>
                                            <span class="font-medium">${metrics.frontend.totalRequests}</span>
                                        </div>
                                        <div class="flex justify-between">
                                            <span>Success Rate:</span>
                                            <span class="font-medium">${metrics.combined.success_rate}</span>
                                        </div>
                                        <div class="flex justify-between">
                                            <span>Cache Hit Rate:</span>
                                            <span class="font-medium">${metrics.combined.cache_hit_rate}</span>
                                        </div>
                                        <div class="flex justify-between">
                                            <span>Avg Response Time:</span>
                                            <span class="font-medium">${metrics.combined.average_response_time}</span>
                                        </div>
                                        <div class="flex justify-between">
                                            <span>Cache Size:</span>
                                            <span class="font-medium">${metrics.frontend.cacheSize} entries</span>
                                        </div>
                                    </div>
                                </div>
                            </div>

                            <div class="space-y-4">
                                <h3 class="text-lg font-semibold text-gray-900 dark:text-white">Backend Metrics</h3>
                                <div class="bg-gray-50 dark:bg-gray-700 p-4 rounded-lg">
                                    <div class="space-y-2 text-sm">
                                        ${Object.entries(metrics.backend.cache_stats || {}).map(([key, value]) => `
                                            <div class="flex justify-between">
                                                <span>${key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}:</span>
                                                <span class="font-medium">${typeof value === 'boolean' ? (value ? 'Yes' : 'No') : value}</span>
                                            </div>
                                        `).join('')}
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div class="flex space-x-4 pt-6 mt-6 border-t border-gray-200 dark:border-gray-700">
                            <button class="btn-primary" onclick="refreshPerformanceMetrics()">Refresh Metrics</button>
                            <button class="px-4 py-2 bg-yellow-600 hover:bg-yellow-700 text-white rounded-lg" onclick="clearAllCaches()">Clear Caches</button>
                            <button class="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700" onclick="closeModal()">Close</button>
                        </div>
                    </div>
                </div>
            </div>
        `;

        document.getElementById('modal-container').innerHTML = modalContent;
    } catch (error) {
        console.error('Failed to load performance metrics:', error);
        window.gollamaAPI.showError('Performance Metrics Error', 'Unable to load performance metrics. Please try again.');
    }
}

async function refreshPerformanceMetrics() {
    try {
        await window.gollamaAPI.refreshCache();
        window.gollamaAPI.showSuccess('Cache Refreshed', 'All caches have been refreshed successfully');
        closeModal();
    } catch (error) {
        console.error('Failed to refresh cache:', error);
        window.gollamaAPI.showError('Cache Refresh Error', 'Failed to refresh cache. Please try again.');
    }
}

async function clearAllCaches() {
    try {
        window.gollamaAPI.clearCache();
        window.gollamaAPI.resetPerformanceMetrics();
        window.gollamaAPI.showSuccess('Caches Cleared', 'All caches and metrics have been cleared');
        closeModal();
    } catch (error) {
        console.error('Failed to clear caches:', error);
        window.gollamaAPI.showError('Clear Cache Error', 'Failed to clear caches. Please try again.');
    }
}

// Enhanced error messages with better user feedback
function enhanceErrorMessages() {
    // Override the default error handler to provide more context
    window.addEventListener('error', (event) => {
        console.error('Global error caught:', event.error);

        // Show user-friendly error for common issues
        if (event.error && event.error.message) {
            const message = event.error.message.toLowerCase();

            if (message.includes('network') || message.includes('fetch')) {
                window.gollamaAPI?.showError(
                    'Connection Error',
                    'Unable to connect to the Ollama service. Please check if Ollama is running.'
                );
            } else if (message.includes('timeout')) {
                window.gollamaAPI?.showError(
                    'Request Timeout',
                    'The request took too long to complete. Please try again.'
                );
            } else if (message.includes('not found') || message.includes('404')) {
                window.gollamaAPI?.showError(
                    'Resource Not Found',
                    'The requested resource was not found. It may have been deleted or moved.'
                );
            }
        }
    });
}

// Global refresh function for the refresh button
async function refreshModels() {
    console.log('🔄 Manual refresh triggered');

    if (currentView === 'models') {
        await showModels(true); // Force refresh
    } else if (currentView === 'running') {
        await showRunning(true); // Force refresh
    }

    window.gollamaAPI.showSuccess('Refreshed', 'Data has been refreshed successfully');
}

// Enhanced navigation update with performance indicator
function updateNavigation() {
    // Update active nav button
    document.querySelectorAll('.nav-link').forEach(link => {
        link.classList.remove('bg-gray-100', 'dark:bg-gray-700', 'text-gray-900', 'dark:text-white');
        link.classList.add('text-gray-500', 'dark:text-gray-400');
    });

    // Highlight current view
    const navButtons = {
        'models': 0,
        'running': 1,
        'settings': 2,
        'diagnostics': 3,
        'performance': 4
    };

    const activeIndex = navButtons[currentView];
    if (activeIndex !== undefined) {
        const activeButton = document.querySelectorAll('.nav-link')[activeIndex];
        if (activeButton) {
            activeButton.classList.add('bg-gray-100', 'dark:bg-gray-700', 'text-gray-900', 'dark:text-white');
            activeButton.classList.remove('text-gray-500', 'dark:text-gray-400');
        }
    }

    // Show/hide performance metrics based on user preference
    const perfElement = document.getElementById('performance-metrics');
    if (perfElement && window.gollamaAPI) {
        const showPerf = localStorage.getItem('showPerformanceMetrics') === 'true';
        if (showPerf) {
            perfElement.classList.remove('hidden');
            updatePerformanceDisplay();
        } else {
            perfElement.classList.add('hidden');
        }
    }
}

// Toggle performance metrics display
function togglePerformanceDisplay() {
    const perfElement = document.getElementById('performance-metrics');
    if (perfElement) {
        const isHidden = perfElement.classList.contains('hidden');
        if (isHidden) {
            perfElement.classList.remove('hidden');
            localStorage.setItem('showPerformanceMetrics', 'true');
            updatePerformanceDisplay();
        } else {
            perfElement.classList.add('hidden');
            localStorage.setItem('showPerformanceMetrics', 'false');
        }
    }
}

// Initialize enhanced error handling and performance monitoring
document.addEventListener('DOMContentLoaded', () => {
    enhanceErrorMessages();

    // Set up performance metrics update interval
    setInterval(() => {
        if (!document.getElementById('performance-metrics').classList.contains('hidden')) {
            updatePerformanceDisplay();
        }
    }, 5000); // Update every 5 seconds

    // Add keyboard shortcuts
    document.addEventListener('keydown', (event) => {
        // Ctrl/Cmd + R for refresh
        if ((event.ctrlKey || event.metaKey) && event.key === 'r') {
            event.preventDefault();
            refreshModels();
        }

        // Ctrl/Cmd + P for performance metrics
        if ((event.ctrlKey || event.metaKey) && event.key === 'p') {
            event.preventDefault();
            togglePerformanceDisplay();
        }

        // Escape to close modals
        if (event.key === 'Escape') {
            closeModal();
        }
    });
});
