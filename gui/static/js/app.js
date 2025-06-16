// Gollama GUI JavaScript

// Global state
let currentModels = [];
let currentRunningModels = [];
let currentView = 'models';
let autoRefreshInterval = null;

// Initialize the application
document.addEventListener('DOMContentLoaded', function() {
    console.log('Gollama GUI initialized');

    // Initialize the application
    initializeApp();

    // Load initial view
    showModels();
});

// Initialize application
async function initializeApp() {
    try {
        // Check service health
        await window.go.main.App.HealthCheck();
        console.log('Service connection established');

        // Load settings and apply theme
        const settings = await window.go.main.App.GetConfig();
        applyTheme(settings.Theme);

        // Set up auto-refresh if enabled
        if (settings.AutoRefresh) {
            setupAutoRefresh(settings.RefreshInterval);
        }

    } catch (error) {
        console.error('Failed to initialize application:', error);
        showToast('error', 'Connection Error', 'Failed to connect to Ollama service');
    }
}

// Navigation functions
async function showModels() {
    currentView = 'models';
    updateNavigation();
    showLoading();

    try {
        const models = await window.go.main.App.GetModels();
        currentModels = models || [];
        renderModels(currentModels);
    } catch (error) {
        console.error('Failed to load models:', error);
        showToast('error', 'Error', 'Failed to load models');
        document.getElementById('main-content').innerHTML = `
            <div class="text-center py-12">
                <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-4">Error Loading Models</h2>
                <p class="text-gray-600 dark:text-gray-400 mb-4">${error.message || 'Unknown error occurred'}</p>
                <button class="btn-primary" onclick="showModels()">Retry</button>
            </div>
        `;
    } finally {
        hideLoading();
    }
}

async function showRunning() {
    currentView = 'running';
    updateNavigation();
    showLoading();

    try {
        const runningModels = await window.go.main.App.GetRunningModels();
        currentRunningModels = runningModels || [];
        renderRunningModels(currentRunningModels);
    } catch (error) {
        console.error('Failed to load running models:', error);
        showToast('error', 'Error', 'Failed to load running models');
        document.getElementById('main-content').innerHTML = `
            <div class="text-center py-12">
                <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-4">Error Loading Running Models</h2>
                <p class="text-gray-600 dark:text-gray-400 mb-4">${error.message || 'Unknown error occurred'}</p>
                <button class="btn-primary" onclick="showRunning()">Retry</button>
            </div>
        `;
    } finally {
        hideLoading();
    }
}

async function showSettings() {
    try {
        const settings = await window.go.main.App.GetConfig();
        renderSettings(settings);
    } catch (error) {
        console.error('Failed to load settings:', error);
        showToast('error', 'Error', 'Failed to load settings');
    }
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
    showLoading();
    try {
        await window.go.main.App.RunModel(name);
        showToast('success', 'Model Started', `${name} is now running`);

        // Refresh current view
        if (currentView === 'models') {
            await showModels();
        } else if (currentView === 'running') {
            await showRunning();
        }
    } catch (error) {
        console.error('Failed to run model:', error);
        showToast('error', 'Error', `Failed to run model: ${error.message}`);
    } finally {
        hideLoading();
    }
}

async function deleteModel(name) {
    if (!confirm(`Delete model ${name}? This cannot be undone.`)) {
        return;
    }

    showLoading();
    try {
        await window.go.main.App.DeleteModel(name);
        showToast('success', 'Model Deleted', `${name} has been deleted`);

        // Refresh models view
        if (currentView === 'models') {
            await showModels();
        }
    } catch (error) {
        console.error('Failed to delete model:', error);
        showToast('error', 'Error', `Failed to delete model: ${error.message}`);
    } finally {
        hideLoading();
    }
}

async function unloadModel(name) {
    showLoading();
    try {
        await window.go.main.App.UnloadModel(name);
        showToast('success', 'Model Unloaded', `${name} has been unloaded`);

        // Refresh current view
        if (currentView === 'running') {
            await showRunning();
        } else if (currentView === 'models') {
            await showModels();
        }
    } catch (error) {
        console.error('Failed to unload model:', error);
        showToast('error', 'Error', `Failed to unload model: ${error.message}`);
    } finally {
        hideLoading();
    }
}

async function showModelDetails(name) {
    showLoading();
    try {
        const details = await window.go.main.App.GetModelDetails(name);
        renderModelDetails(details);
    } catch (error) {
        console.error('Failed to load model details:', error);
        showToast('error', 'Error', `Failed to load model details: ${error.message}`);
    } finally {
        hideLoading();
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
    showLoading();

    const formData = new FormData(event.target);
    const settings = {
        OllamaAPIURL: formData.get('ollama_api_url'),
        Theme: formData.get('theme'),
        AutoRefresh: formData.has('auto_refresh'),
        RefreshInterval: parseInt(formData.get('refresh_interval')),
        WindowWidth: 1200,
        WindowHeight: 800,
        DefaultView: 'models',
        ShowSystemTray: false
    };

    try {
        await window.go.main.App.UpdateConfig(settings);
        showToast('success', 'Settings Saved', 'Your settings have been saved successfully');

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
        showToast('error', 'Error', `Failed to save settings: ${error.message}`);
    } finally {
        hideLoading();
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
    showLoading();

    const formData = new FormData(event.target);
    const request = {
        ModelName: formData.get('model_name') || '',
        VRAMAvailable: parseFloat(formData.get('vram_gb') || '0'),
        ContextLength: parseInt(formData.get('context') || '0'),
        Quantization: formData.get('quantization') || ''
    };

    try {
        const result = await window.go.main.App.EstimateVRAM(request);
        renderVRAMResults(result);
    } catch (error) {
        console.error('Failed to calculate vRAM:', error);
        showToast('error', 'Error', `Failed to calculate vRAM: ${error.message}`);
    } finally {
        hideLoading();
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
function showLoading() {
    const loading = document.getElementById('loading');
    if (loading) {
        loading.classList.add('show');
    }
}

function hideLoading() {
    const loading = document.getElementById('loading');
    if (loading) {
        loading.classList.remove('show');
    }
}

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
                const models = await window.go.main.App.GetModels();
                currentModels = models || [];
                renderModels(currentModels);
            } catch (error) {
                console.error('Auto-refresh failed:', error);
            }
        } else if (currentView === 'running') {
            try {
                const runningModels = await window.go.main.App.GetRunningModels();
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
