/**
 * StreamingManager - Core EventSource streaming functionality
 */
class StreamingManager {
    /**
     * Create a generic streaming handler
     */
    static createStreamingHandler(config) {
        const {
            projectId,
            streamType,
            url,
            initialMessage,
            button,
            buttonDisabledText,
            onComplete,
            setupCleanup,
        } = config;

        // Validate required dependencies
        if (!window.ModalController || !window.UIUpdater || !window.projectStateManager) {
            console.error('Required dependencies not loaded: ModalController, UIUpdater, or projectStateManager');
            return null;
        }

        const outputContent = window.ModalController.getOutputContent(projectId, streamType);
        if (!outputContent) {
            console.error(`Output content element not found for ${streamType} streaming`);
            return null;
        }

        // Clear and show initial message
        outputContent.innerHTML = `<span class="oar-text">${initialMessage}</span>\n\n`;
        outputContent.lastWasDockerCompose = false;

        // Disable button during operation
        if (button && buttonDisabledText) {
            window.UIUpdater.setButtonBusyState(button, buttonDisabledText);
        }

        // Create EventSource connection
        const eventSource = new EventSource(url);

        // Setup cleanup if provided (for logs modal)
        if (setupCleanup) {
            setupCleanup(eventSource);
        }

        // Handle streaming messages
        eventSource.onmessage = function (event) {
            StreamingManager.handleStreamingMessage(event, {
                projectId,
                streamType,
                outputContent,
                button,
                eventSource,
                onComplete
            });
        };

        // Handle connection errors
        eventSource.onerror = function (event) {
            StreamingManager.handleStreamingError(event, {
                projectId,
                streamType,
                outputContent,
                button,
                eventSource,
                onComplete
            });
        };

        return eventSource;
    }

    /**
     * Handle incoming streaming messages
     */
    static handleStreamingMessage(event, context) {
        const { projectId, streamType, outputContent, button, eventSource, onComplete } = context;
        const data = event.data;

        try {
            // Parse JSON message
            const message = JSON.parse(data);

            // Handle project state updates
            if (message.projectState) {
                // Store project state for later use when modal closes
                window.projectStateManager.storeValidatedProjectState(projectId, message.projectState);
            }

            // Handle display messages
            if (message.message) {
                if (message.type === 'info' && message.source === 'docker') {
                    // Docker Compose output - display as plain text with separation
                    window.UIUpdater.addDockerComposeOutput(outputContent, message.message);
                } else {
                    // Oar messages (info/success/error) - display with italic styling
                    window.UIUpdater.addOarText(outputContent, message.message, message.type);
                }
            }

            // Handle completion based on message type
            // Only treat success/error messages with projectState as completion
            if ((message.type === 'success' || message.type === 'error') && message.projectState) {
                // This is a completion message, close the stream and setup button
                eventSource.close();

                if (button) {
                    window.UIUpdater.setupButtonCloseState(button, projectId, streamType);
                }

                if (onComplete) {
                    onComplete(message.type, message.message);
                }
            }
        } catch (e) {
            // Fallback for non-JSON messages (shouldn't happen with new unified format)
            console.warn('Received non-JSON message:', data);
            outputContent.innerHTML += data + '\n';
        }

        window.UIUpdater.autoScroll(outputContent);
    }

    /**
     * Handle streaming connection errors
     */
    static handleStreamingError(event, context) {
        const { projectId, streamType, outputContent, button, eventSource, onComplete } = context;

        const errorMessage = streamType === 'logs'
            ? 'Connection error or log stream ended'
            : 'Connection error or process completed';

        window.UIUpdater.addOarText(outputContent, errorMessage, 'error');
        eventSource.close();

        if (button) {
            window.UIUpdater.setupButtonCloseState(button, projectId, streamType);
        }

        if (onComplete) {
            onComplete('error', errorMessage);
        }
    }

    /**
     * Validate streaming configuration
     */
    static validateConfig(config) {
        const required = ['projectId', 'streamType', 'url', 'initialMessage'];

        for (const field of required) {
            if (!config[field]) {
                console.error(`Missing required field in streaming config: ${field}`);
                return false;
            }
        }

        if (!window.ModalController.validateModalConfig(config.streamType)) {
            console.error(`Invalid stream type: ${config.streamType}`);
            return false;
        }

        return true;
    }

    /**
     * Create streaming handler with validation
     */
    static createValidatedStreamingHandler(config) {
        if (!StreamingManager.validateConfig(config)) {
            return null;
        }

        return StreamingManager.createStreamingHandler(config);
    }
}

// Export for module systems or global access
if (typeof module !== 'undefined' && module.exports) {
    module.exports = StreamingManager;
} else {
    window.StreamingManager = StreamingManager;
}