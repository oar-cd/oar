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
        getAvailableOperations,
        getOperationConfig,
        isValidOperation,
        STREAM_OPERATIONS
    };
}