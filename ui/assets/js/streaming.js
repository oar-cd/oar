/**
 * Oar Streaming JavaScript - Main entry point
 *
 * This file coordinates all streaming functionality by loading the modular components
 * and initializing the streaming system.
 *
 * Module Architecture:
 * - ui-utils.js: DOM manipulation and UI updates
 * - state-manager.js: Project state storage and management
 * - modal-manager.js: Modal lifecycle and configuration
 * - core.js: Core EventSource streaming functionality
 * - operations.js: Specific streaming operations (deploy, stop, logs)
 */

// Module loading helper
function loadStreamingModules() {
    const modules = [
        '/ui/assets/js/streaming/ui-utils.js',
        '/ui/assets/js/streaming/state-manager.js',
        '/ui/assets/js/streaming/modal-manager.js',
        '/ui/assets/js/streaming/core.js',
        '/ui/assets/js/streaming/operations.js'
    ];

    // Since we can't use ES6 modules in the browser without bundling,
    // we'll rely on the modules being loaded via script tags in the HTML
    // and just verify they're available

    const requiredClasses = [
        'UIUpdater',
        'ProjectStateManager',
        'ModalController',
        'StreamingManager'
    ];

    const requiredFunctions = [
        'deployProjectWithStreaming',
        'stopProjectWithStreaming',
        'viewLogsWithStreaming'
    ];

    // Check if all required components are loaded
    const missingClasses = requiredClasses.filter(className => !window[className]);
    const missingFunctions = requiredFunctions.filter(funcName => !window[funcName]);

    if (missingClasses.length > 0 || missingFunctions.length > 0) {
        console.error('Missing streaming components:', {
            classes: missingClasses,
            functions: missingFunctions
        });
        return false;
    }

    return true;
}

// Initialize streaming system
function initializeStreaming() {
    if (!loadStreamingModules()) {
        console.error('Failed to load streaming modules');
        return;
    }

    // Initialize modal event listeners
    if (window.ModalController) {
        window.ModalController.initializeModalEventListeners();
    }

    console.log('Oar streaming system initialized successfully');
}

// Backwards compatibility functions (for any existing references)
// These are deprecated and should be replaced with the new modular functions

function updateProjectCardWithState(projectId) {
    console.warn('updateProjectCardWithState is deprecated. Use projectStateManager.updateProjectCardWithState instead.');
    if (window.projectStateManager) {
        window.projectStateManager.updateProjectCardWithState(projectId);
    }
}

function closeModal(modalId) {
    console.warn('closeModal is deprecated. Use UIUpdater.closeModal instead.');
    if (window.UIUpdater) {
        window.UIUpdater.closeModal(modalId);
    }
}

// Export backwards compatibility functions
window.updateProjectCardWithState = updateProjectCardWithState;
window.closeModal = closeModal;

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', function() {
    initializeStreaming();
});