/**
 * Gollama Diagnostics Console Helper
 *
 * This script provides easy-to-use functions for testing and debugging
 * the Gollama service binding from the browser console.
 *
 * Usage:
 * - Open browser console (F12)
 * - Type: await runQuickDiagnostics()
 * - Type: await runFullDiagnostics()
 * - Type: await testServiceBinding()
 */

// Quick diagnostic functions for console use
window.gollamaDiagnostics = {

    /**
     * Run quick diagnostics and log results
     */
    async runQuickDiagnostics() {
        console.log('🔍 Running quick diagnostics...');

        try {
            if (!window.gollamaAPI) {
                console.error('❌ Gollama API not available');
                return;
            }

            // Test basic connectivity
            console.log('🧪 Testing basic connectivity...');
            const testResult = await window.gollamaAPI.testServiceBinding();
            console.log('✅ Service binding test:', testResult);

            // Get service status
            console.log('⚙️ Getting service status...');
            const status = await window.gollamaAPI.getServiceStatus();
            console.log('✅ Service status:', status);

            // Get available methods
            console.log('📋 Available methods:', Object.keys(window.gollamaAPI.bindingMethods));

            console.log('✅ Quick diagnostics completed successfully');
            return {
                testResult,
                status,
                availableMethods: Object.keys(window.gollamaAPI.bindingMethods)
            };

        } catch (error) {
            console.error('❌ Quick diagnostics failed:', error);
            throw error;
        }
    },

    /**
     * Run comprehensive diagnostics
     */
    async runFullDiagnostics() {
        console.log('🔍 Running full diagnostics...');

        try {
            if (!window.gollamaAPI) {
                console.error('❌ Gollama API not available');
                return;
            }

            // Get comprehensive diagnostics
            const diagnostics = await window.gollamaAPI.getDiagnostics();
            console.log('📊 Full diagnostics:', diagnostics);

            // Run diagnostic tests
            const tests = await window.gollamaAPI.runDiagnosticTests();
            console.log('🧪 Diagnostic tests:', tests);

            // Verify service binding
            const verification = await window.gollamaAPI.verifyServiceBinding();
            console.log('🔗 Service binding verification:', verification);

            // Get diagnostic info from Go service
            const info = await window.gollamaAPI.getDiagnosticInfo();
            console.log('📋 Go service diagnostic info:', info);

            console.log('✅ Full diagnostics completed successfully');
            return {
                diagnostics,
                tests,
                verification,
                info
            };

        } catch (error) {
            console.error('❌ Full diagnostics failed:', error);
            throw error;
        }
    },

    /**
     * Test specific service methods
     */
    async testServiceMethods() {
        console.log('🧪 Testing service methods...');

        const methods = [
            { name: 'HealthCheck', args: [] },
            { name: 'GetConfig', args: [] },
            { name: 'GetModels', args: [] },
            { name: 'GetRunningModels', args: [] },
            { name: 'GetSystemInfo', args: [] }
        ];

        const results = {};

        for (const method of methods) {
            try {
                console.log(`🔄 Testing ${method.name}...`);
                const result = await window.gollamaAPI.callMethod(method.name, ...method.args);
                results[method.name] = { status: 'success', result };
                console.log(`✅ ${method.name} passed`);
            } catch (error) {
                results[method.name] = { status: 'failed', error: error.message };
                console.log(`❌ ${method.name} failed:`, error.message);
            }
        }

        console.log('🧪 Service method testing completed');
        return results;
    },

    /**
     * Test service binding specifically
     */
    async testServiceBinding() {
        console.log('🔗 Testing service binding...');

        try {
            // Check if API is initialized
            if (!window.gollamaAPI || !window.gollamaAPI.initialized) {
                console.error('❌ Gollama API not initialized');
                return { status: 'failed', error: 'API not initialized' };
            }

            // Test service binding method
            const bindingTest = await window.gollamaAPI.testServiceBinding();
            console.log('✅ Service binding test result:', bindingTest);

            // Verify service binding
            const verification = await window.gollamaAPI.verifyServiceBinding();
            console.log('✅ Service binding verification:', verification);

            return {
                status: 'success',
                bindingTest,
                verification
            };

        } catch (error) {
            console.error('❌ Service binding test failed:', error);
            return { status: 'failed', error: error.message };
        }
    },

    /**
     * Monitor service health continuously
     */
    startHealthMonitoring(interval = 10000) {
        console.log(`💓 Starting health monitoring (${interval}ms interval)...`);

        if (this.healthMonitorInterval) {
            clearInterval(this.healthMonitorInterval);
        }

        this.healthMonitorInterval = setInterval(async () => {
            try {
                const status = await window.gollamaAPI.getServiceStatus();
                const timestamp = new Date().toLocaleTimeString();
                console.log(`💓 [${timestamp}] Health check:`, status.status);

                if (status.status !== 'healthy') {
                    console.warn(`⚠️ [${timestamp}] Service unhealthy:`, status);
                }
            } catch (error) {
                const timestamp = new Date().toLocaleTimeString();
                console.error(`💔 [${timestamp}] Health check failed:`, error.message);
            }
        }, interval);

        return this.healthMonitorInterval;
    },

    /**
     * Stop health monitoring
     */
    stopHealthMonitoring() {
        if (this.healthMonitorInterval) {
            clearInterval(this.healthMonitorInterval);
            this.healthMonitorInterval = null;
            console.log('⏹️ Health monitoring stopped');
        }
    },

    /**
     * Generate diagnostic report
     */
    async generateReport() {
        console.log('📄 Generating diagnostic report...');

        try {
            const report = await window.gollamaAPI.createDiagnosticReport();
            console.log('📄 Diagnostic report generated:', report);

            // Also log the readable summary
            console.log('\n' + report.readableSummary);

            return report;
        } catch (error) {
            console.error('❌ Failed to generate diagnostic report:', error);
            throw error;
        }
    },

    /**
     * Show diagnostic modal
     */
    async showDiagnosticModal() {
        console.log('🔍 Showing diagnostic modal...');

        try {
            await window.gollamaAPI.displayDiagnosticReport(true);
        } catch (error) {
            console.error('❌ Failed to show diagnostic modal:', error);
            throw error;
        }
    },

    /**
     * Export diagnostic data
     */
    async exportDiagnostics() {
        console.log('💾 Exporting diagnostic data...');

        try {
            const diagnostics = await this.runFullDiagnostics();

            const blob = new Blob([JSON.stringify(diagnostics, null, 2)], {
                type: 'application/json'
            });

            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `gollama-diagnostics-${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.json`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);

            console.log('💾 Diagnostic data exported successfully');
            return diagnostics;

        } catch (error) {
            console.error('❌ Failed to export diagnostics:', error);
            throw error;
        }
    },

    /**
     * Help function to show available commands
     */
    help() {
        console.log(`
🔍 Gollama Diagnostics Console Helper

Available commands:
• await gollamaDiagnostics.runQuickDiagnostics()     - Run basic diagnostics
• await gollamaDiagnostics.runFullDiagnostics()      - Run comprehensive diagnostics
• await gollamaDiagnostics.testServiceMethods()      - Test individual service methods
• await gollamaDiagnostics.testServiceBinding()      - Test service binding specifically
• gollamaDiagnostics.startHealthMonitoring(10000)    - Start health monitoring (10s interval)
• gollamaDiagnostics.stopHealthMonitoring()          - Stop health monitoring
• await gollamaDiagnostics.generateReport()          - Generate diagnostic report
• await gollamaDiagnostics.showDiagnosticModal()     - Show diagnostic modal
• await gollamaDiagnostics.exportDiagnostics()       - Export diagnostic data
• gollamaDiagnostics.help()                          - Show this help

Examples:
  await gollamaDiagnostics.runQuickDiagnostics()
  await gollamaDiagnostics.testServiceBinding()
  gollamaDiagnostics.startHealthMonitoring(5000)
        `);
    }
};

// Convenience aliases for console use
window.runQuickDiagnostics = window.gollamaDiagnostics.runQuickDiagnostics.bind(window.gollamaDiagnostics);
window.runFullDiagnostics = window.gollamaDiagnostics.runFullDiagnostics.bind(window.gollamaDiagnostics);
window.testServiceBinding = window.gollamaDiagnostics.testServiceBinding.bind(window.gollamaDiagnostics);
window.diagnosticsHelp = window.gollamaDiagnostics.help.bind(window.gollamaDiagnostics);

// Auto-show help on load
console.log('🔍 Gollama Diagnostics Console Helper loaded!');
console.log('Type "diagnosticsHelp()" for available commands or "await runQuickDiagnostics()" to start.');
