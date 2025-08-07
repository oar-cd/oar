/**
 * StreamingOperations - Specific streaming operation functions
 */

// Stream operation configurations
const STREAM_OPERATIONS = {
    deploy: {
        endpoint: '/deploy/stream',
        initialMessage: 'Deploying project...',
        buttonDisabledText: 'Deploying...',
        buttonSelector: (projectId) => `deploy-button-${projectId}`,
        getUrl: (projectId) => {
            // Check for pull changes checkbox
            const pullCheckbox = document.getElementById(`pull-changes-${projectId}`);
            const pullChanges = pullCheckbox ? pullCheckbox.checked : true;
            return `/projects/${projectId}/deploy/stream${pullChanges ? '?pull=true' : ''}`;
        }
    },
    stop: {
        endpoint: '/stop/stream',
        initialMessage: 'Stopping project...',
        buttonDisabledText: 'Stopping...',
        buttonSelector: (projectId) => `stop-button-${projectId}`,
        getUrl: (projectId) => `/projects/${projectId}/stop/stream`
    },
    logs: {
        endpoint: '/logs/stream',
        initialMessage: 'Starting log stream...',
        buttonDisabledText: null,
        buttonSelector: () => null,
        getUrl: (projectId) => `/projects/${projectId}/logs/stream`
    }
};

/**
 * Deploy project with streaming output
 */
function deployProjectWithStreaming(projectId) {
    const config = STREAM_OPERATIONS.deploy;
    const deployButton = document.getElementById(config.buttonSelector(projectId));

    if (!window.StreamingManager) {
        console.error('StreamingManager not available');
        return;
    }

    window.StreamingManager.createValidatedStreamingHandler({
        projectId,
        streamType: 'deploy',
        url: config.getUrl(projectId),
        initialMessage: config.initialMessage,
        button: deployButton,
        buttonDisabledText: config.buttonDisabledText,
    });
}

/**
 * Stop project with streaming output
 */
function stopProjectWithStreaming(projectId) {
    const config = STREAM_OPERATIONS.stop;
    const stopButton = document.getElementById(config.buttonSelector(projectId));

    if (!window.StreamingManager) {
        console.error('StreamingManager not available');
        return;
    }

    window.StreamingManager.createValidatedStreamingHandler({
        projectId,
        streamType: 'stop',
        url: config.getUrl(projectId),
        initialMessage: config.initialMessage,
        button: stopButton,
        buttonDisabledText: config.buttonDisabledText,
    });
}

/**
 * View logs with streaming output
 */
function viewLogsWithStreaming(projectId) {
    const config = STREAM_OPERATIONS.logs;
    const modal = document.getElementById(`logs-modal-${projectId}`);

    if (!modal) {
        console.error('Modal element not found for logs streaming');
        return;
    }

    if (!window.StreamingManager || !window.ModalController) {
        console.error('Required streaming components not available');
        return;
    }

    window.StreamingManager.createValidatedStreamingHandler({
        projectId,
        streamType: 'logs',
        url: config.getUrl(projectId),
        initialMessage: config.initialMessage,
        setupCleanup: function (eventSource) {
            return window.ModalController.setupModalCleanup(modal, eventSource);
        },
    });
}

/**
 * Get available stream operations
 */
function getAvailableOperations() {
    return Object.keys(STREAM_OPERATIONS);
}

/**
 * Get operation configuration
 */
function getOperationConfig(operationType) {
    return STREAM_OPERATIONS[operationType] || null;
}

/**
 * Load project configuration (non-streaming)
 */
function loadProjectConfig(projectId) {
    const outputContent = document.getElementById(`config-output-content-${projectId}`);

    if (!outputContent) {
        console.error('Config output element not found');
        return;
    }

    // Fetch configuration
    fetch(`/projects/${projectId}/config`)
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            return response.text();
        })
        .then(configText => {
            // Display the configuration with consistent text color (gray-400)
            outputContent.innerHTML = `<pre class="whitespace-pre-wrap text-gray-400">${escapeHtml(configText)}</pre>`;
        })
        .catch(error => {
            console.error('Error loading config:', error);
            outputContent.innerHTML = '<span class="oar-text text-red-400">Failed to load configuration. Please try again.</span>';
        });
}

/**
 * Escape HTML to prevent XSS
 */
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

/**
 * Validate operation type
 */
function isValidOperation(operationType) {
    return operationType in STREAM_OPERATIONS;
}

// Export functions for global access
if (typeof window !== 'undefined') {
    window.deployProjectWithStreaming = deployProjectWithStreaming;
    window.stopProjectWithStreaming = stopProjectWithStreaming;
    window.viewLogsWithStreaming = viewLogsWithStreaming;
    window.loadProjectConfig = loadProjectConfig;
    window.getAvailableOperations = getAvailableOperations;
    window.getOperationConfig = getOperationConfig;
    window.isValidOperation = isValidOperation;
}

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        deployProjectWithStreaming,
        stopProjectWithStreaming,
        viewLogsWithStreaming,
        loadProjectConfig,
        getAvailableOperations,
        getOperationConfig,
        isValidOperation,
        STREAM_OPERATIONS
    };
}
