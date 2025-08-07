/**
 * ModalController - Manages modal operations and configurations
 */
class ModalController {
    // Modal configuration for different types
    static MODAL_CONFIGS = {
        deploy: {
            outputMessage: 'Output will show up here when you deploy the project...',
            buttonText: 'Deploy Project',
            buttonHandler: 'deployProjectWithStreaming'
        },
        stop: {
            outputMessage: 'Output will show up here when you stop the project...',
            buttonText: 'Stop Project',
            buttonHandler: 'stopProjectWithStreaming'
        },
        logs: {
            outputMessage: 'Starting log stream...',
            buttonText: null,
            buttonHandler: null
        },
        config: {
            outputMessage: '',
            buttonText: null,
            buttonHandler: null
        }
    };

    // Modal trigger patterns for parsing
    static MODAL_PATTERNS = [
        { pattern: 'deploy-project-modal-', type: 'deploy' },
        { pattern: 'stop-project-modal-', type: 'stop' },
        { pattern: 'logs-modal-', type: 'logs' },
        { pattern: 'config-modal-', type: 'config' }
    ];

    /**
     * Extract project ID and modal type from modal ID
     */
    static parseModalId(modalId) {
        for (const { pattern, type } of ModalController.MODAL_PATTERNS) {
            if (modalId.includes(pattern)) {
                return {
                    projectId: modalId.replace(pattern, ''),
                    modalType: type
                };
            }
        }
        return null;
    }

    /**
     * Reset modal content to fresh state
     */
    static resetModalContent(projectId, modalType) {
        const config = ModalController.MODAL_CONFIGS[modalType];
        if (!config) {
            console.warn('Unknown modal type:', modalType);
            return;
        }

        const outputContent = document.getElementById(`${modalType}-output-content-${projectId}`);
        const button = document.getElementById(`${modalType}-button-${projectId}`);

        if (outputContent) {
            outputContent.innerHTML = `<span class="oar-text">${config.outputMessage}</span>`;
            // Reset output tracking flags
            outputContent.lastWasDockerCompose = false;
        }

        if (button && config.buttonText) {
            button.disabled = false;
            button.textContent = config.buttonText;

            // Set up button handler if specified
            if (config.buttonHandler && window[config.buttonHandler]) {
                button.onclick = function () {
                    window[config.buttonHandler](projectId);
                };
            }
        }
    }

    /**
     * Setup event source cleanup for modal close
     */
    static setupModalCleanup(modal, eventSource) {
        if (!modal || !eventSource) return;

        // Store eventSource reference on the modal for cleanup
        modal.logsEventSource = eventSource;

        // Add event listener for modal close to stop streaming
        const handleModalClose = function () {
            if (eventSource && eventSource.readyState !== EventSource.CLOSED) {
                eventSource.close();
            }
        };

        // Listen for modal close events - try multiple possible event names
        modal.addEventListener('modal:closed', handleModalClose, { once: true });
        modal.addEventListener('close', handleModalClose, { once: true });
        modal.addEventListener('hidden', handleModalClose, { once: true });

        // Also listen for close button clicks directly
        const closeButtons = modal.querySelectorAll('[data-modal-close]');
        closeButtons.forEach((button) => {
            button.addEventListener('click', handleModalClose, { once: true });
        });

        return handleModalClose;
    }

    /**
     * Initialize modal event listeners
     */
    static initializeModalEventListeners() {
        // Listen for clicks on modal triggers
        document.addEventListener('click', function (event) {
            const trigger = event.target.closest('[data-modal-trigger]');
            if (!trigger) return;

            const modalId = trigger.getAttribute('data-modal-trigger');
            const parsed = ModalController.parseModalId(modalId);

            if (parsed) {
                const { projectId, modalType } = parsed;

                // Reset modal content to fresh state
                setTimeout(function () {
                    ModalController.resetModalContent(projectId, modalType);

                    // Auto-start logs streaming for logs modal
                    if (modalType === 'logs' && window.viewLogsWithStreaming) {
                        window.viewLogsWithStreaming(projectId);
                    }

                    // Auto-load config for config modal
                    if (modalType === 'config' && window.loadProjectConfig) {
                        window.loadProjectConfig(projectId);
                    }
                }, 100); // Give modal time to open
            }
        });
    }

    /**
     * Get modal element by project ID and type
     */
    static getModal(projectId, modalType) {
        const modalId = `${modalType}-project-modal-${projectId}`;
        return document.getElementById(modalId);
    }

    /**
     * Get output content element for modal
     */
    static getOutputContent(projectId, modalType) {
        const elementId = `${modalType}-output-content-${projectId}`;
        return document.getElementById(elementId);
    }

    /**
     * Get button element for modal
     */
    static getButton(projectId, modalType) {
        const buttonId = `${modalType}-button-${projectId}`;
        return document.getElementById(buttonId);
    }

    /**
     * Validate modal configuration
     */
    static validateModalConfig(modalType) {
        return modalType in ModalController.MODAL_CONFIGS;
    }

    /**
     * Close modal by ID
     */
    static closeModal(modalId) {
        if (window.UIUpdater) {
            window.UIUpdater.closeModal(modalId);
        }
    }
}

// Export for module systems or global access
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ModalController;
} else {
    window.ModalController = ModalController;
}
