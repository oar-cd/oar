/**
 * UIUpdater - Handles DOM manipulation and UI updates for streaming operations
 */
class UIUpdater {
    // Status styling configuration
    static STATUS_STYLES = {
        running: { classes: ['bg-green-200', 'text-green-800'] },
        stopped: { classes: ['bg-gray-200', 'text-gray-800'] },
        error: { classes: ['bg-red-200', 'text-red-800'] },
        unknown: { classes: ['bg-yellow-200', 'text-yellow-800'] }
    };

    static ALL_STATUS_CLASSES = [
        'bg-green-200', 'text-green-800',    // running
        'bg-gray-200', 'text-gray-800',      // stopped
        'bg-red-200', 'text-red-800',        // error
        'bg-yellow-200', 'text-yellow-800'   // unknown
    ];

    /**
     * Auto-scroll output content to bottom
     */
    static autoScroll(outputContent) {
        const codeElement = outputContent.closest('code');
        if (codeElement) {
            codeElement.scrollTop = codeElement.scrollHeight;
        }
    }

    /**
     * Add Oar-specific text with styling
     */
    static addOarText(outputContent, text, type = '') {
        // Add separation after Docker Compose output if the last message was Docker Compose
        const needsSeparationBefore = outputContent.lastWasDockerCompose;
        const prefix = needsSeparationBefore ? '\n' : '';

        const className = type ? `oar-text ${type}` : 'oar-text';
        outputContent.innerHTML += `${prefix}<span class="${className}" style="font-style: italic;">${text}</span>\n`;
        outputContent.lastWasDockerCompose = false;
    }

    /**
     * Add Docker Compose output with proper formatting
     */
    static addDockerComposeOutput(outputContent, text) {
        // Check if there's existing content and if the last message was not Docker Compose output
        const hasExistingContent = outputContent.innerHTML.trim().length > 0;
        const needsSeparationBefore = hasExistingContent && !outputContent.lastWasDockerCompose;

        // Add separation before Docker Compose output if needed
        const prefix = needsSeparationBefore ? '\n' : '';

        outputContent.innerHTML += prefix + text + '\n';
        outputContent.lastWasDockerCompose = true;
    }

    /**
     * Update project card status badge
     */
    static updateProjectCardStatus(projectId, newStatus) {
        const statusBadge = document.querySelector(`#project-card-${projectId} .status-badge`);
        if (!statusBadge) {
            console.warn('Status badge not found for project:', projectId);
            return;
        }

        // Remove all status-related classes
        statusBadge.classList.remove(...UIUpdater.ALL_STATUS_CLASSES);

        // Add appropriate classes based on new status
        const statusConfig = UIUpdater.STATUS_STYLES[newStatus] || UIUpdater.STATUS_STYLES.unknown;
        statusBadge.classList.add(...statusConfig.classes);

        // Update text content and data attribute
        const displayStatus = UIUpdater.STATUS_STYLES[newStatus] ? newStatus : 'unknown';
        statusBadge.textContent = displayStatus;
        statusBadge.setAttribute('data-status', displayStatus);
    }

    /**
     * Update project card commit badge
     */
    static updateProjectCardCommit(projectId, newCommit) {
        const commitBadge = document.querySelector(`#project-card-${projectId} .commit-badge`);
        if (!commitBadge) {
            console.warn('Commit badge not found for project:', projectId);
            return;
        }

        // Update classes for commit display
        commitBadge.classList.remove('bg-gray-100', 'text-gray-500'); // no commits style
        commitBadge.classList.add('bg-gray-200', 'text-gray-800');    // has commits style

        // Update text content (first 8 characters) and data attribute
        commitBadge.textContent = newCommit.substring(0, 8);
        commitBadge.setAttribute('data-commit', newCommit);
    }

    /**
     * Update project action buttons based on status
     */
    static updateProjectActionButtons(projectId, newStatus) {
        const projectCard = document.getElementById(`project-card-${projectId}`);
        if (!projectCard) {
            console.warn('Project card not found:', projectId);
            return;
        }

        const allButtons = projectCard.querySelectorAll('button');

        // Update stop button visibility
        const stopButton = allButtons[1]; // Stop button is typically at index 1 (red button)

        if (stopButton && stopButton.classList.contains('text-red-600')) {
            // Find the ActionButtonWithVisibility wrapper
            const actionButtonContainer = stopButton.closest('.relative.group');
            const visibilityContainer = actionButtonContainer ? actionButtonContainer.parentElement : null;

            if (visibilityContainer && visibilityContainer !== actionButtonContainer) {
                if (newStatus === 'running') {
                    // Show stop button by removing hidden class
                    visibilityContainer.classList.remove('hidden');
                } else {
                    // Hide stop button by adding hidden class
                    visibilityContainer.classList.add('hidden');
                }
            }
        }

        // Update edit button state
        const editButton = allButtons[2]; // Edit button is typically at index 2

        if (editButton) {
            const editButtonContainer = editButton.closest('.relative.group');

            if (editButtonContainer) {
                // Update edit button state based on running status
                UIUpdater.updateEditButtonState(editButtonContainer, projectId, newStatus === 'running');
            }
        }
    }

    /**
     * Update edit button state (enabled/disabled) based on project status
     */
    static updateEditButtonState(container, projectId, isRunning) {
        const button = container.querySelector('button');
        const tooltip = container.querySelector('div');

        const tooltipText = isRunning ? 'Cannot edit running project - stop it first' : 'Edit Project';
        const baseClasses = 'flex items-center justify-center p-2 rounded-md transition-colors relative';

        if (button) {
            if (isRunning) {
                // Disabled state for running projects
                button.disabled = true;
                button.removeAttribute('data-modal-trigger');
                button.className = `${baseClasses} text-gray-400 cursor-not-allowed opacity-50`;
            } else {
                // Enabled state for non-running projects
                button.disabled = false;
                button.setAttribute('data-modal-trigger', `edit-project-modal-${projectId}`);
                button.className = `${baseClasses} text-gray-600 hover:text-gray-800 hover:bg-gray-200`;
            }

            // Set tooltip text for both title and aria-label
            button.title = tooltipText;
            button.setAttribute('aria-label', tooltipText);
        }

        if (tooltip) {
            tooltip.textContent = tooltipText;
        }
    }

    /**
     * Setup button in "close" state after operation completion
     */
    static setupButtonCloseState(button, projectId, modalType) {
        if (!button) return;

        button.textContent = 'Close';
        button.disabled = false;
        button.onclick = function () {
            // Close the modal instead of reloading the page
            const modalId = `${modalType}-project-modal-${projectId}`;
            UIUpdater.closeModal(modalId);

            // Update the project card with the stored state information after modal closes
            setTimeout(() => {
                // This will be handled by ProjectStateManager
                if (window.projectStateManager) {
                    window.projectStateManager.updateProjectCardWithState(projectId);
                }
            }, 100); // Small delay to ensure modal close completes
        };
    }

    /**
     * Close modal by triggering its close mechanism
     */
    static closeModal(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            const closeButton = modal.querySelector('[data-modal-close]');
            if (closeButton) {
                closeButton.click();
            } else {
                // Fallback: manually trigger close
                modal.setAttribute('data-open', 'false');
                modal.style.display = 'none';
                modal.classList.add('hidden');
            }
        }
    }

    /**
     * Set button to disabled state during operation
     */
    static setButtonBusyState(button, disabledText) {
        if (!button) return;

        button.disabled = true;
        button.textContent = disabledText;
    }
}

// Export for module systems or global access
if (typeof module !== 'undefined' && module.exports) {
    module.exports = UIUpdater;
} else {
    window.UIUpdater = UIUpdater;
}
