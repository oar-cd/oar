// HTMX Configuration and Toast System for Oar

document.addEventListener('DOMContentLoaded', function() {
    // Configure HTMX global settings
    htmx.config.globalViewTransitions = true;
    htmx.config.useTemplateFragments = true;
    htmx.config.defaultSwapStyle = 'outerHTML';

    // Toast notification system
    const toastContainer = document.createElement('div');
    toastContainer.id = 'toast-container';
    toastContainer.className = 'fixed top-4 right-4 z-50 space-y-2';
    document.body.appendChild(toastContainer);

    // Icon loading function to dynamically load SVG icons
    window.loadIcon = async function(iconName, className = 'icon-sm') {
        try {
            const response = await fetch(`/assets/icons/${iconName}.svg`);
            if (!response.ok) {
                throw new Error(`Icon ${iconName} not found`);
            }
            const svgContent = await response.text();
            // Create a container div with the proper class like the Templ component does
            const container = document.createElement('div');
            container.className = `inline-block ${className}`;
            container.innerHTML = svgContent;
            return container.outerHTML;
        } catch (error) {
            console.error('Error loading icon:', error);
            // Fallback to help-circle icon
            try {
                const fallbackResponse = await fetch('/assets/icons/help-circle.svg');
                if (fallbackResponse.ok) {
                    const fallbackSvg = await fallbackResponse.text();
                    const container = document.createElement('div');
                    container.className = `inline-block ${className}`;
                    container.innerHTML = fallbackSvg;
                    return container.outerHTML;
                }
            } catch (fallbackError) {
                console.error('Error loading fallback icon:', fallbackError);
            }
            // Ultimate fallback if even help-circle fails
            return `<div class="inline-block ${className}"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><path d="M12 17h.01"/></svg></div>`;
        }
    };

    // Function to update project status pill
    window.updateProjectStatus = function(projectId) {
        const statusPill = document.getElementById(`status-pill-${projectId}`);
        if (statusPill) {
            // Use HTMX to fetch updated status pill
            htmx.ajax('GET', `/projects/${projectId}/status`, {
                target: `#status-pill-${projectId}`,
                swap: 'outerHTML'
            });
        }
    };

    window.showToast = async function(message, type = 'info') {
        const toast = document.createElement('div');

        // Map types to colors and icons
        const typeConfig = {
            'success': {
                borderColor: 'border-l-green-500',
                backgroundColor: 'bg-green-50',
                borderAllColor: 'border-green-500',
                textColor: 'text-green-700',
                icon: 'check'
            },
            'error': {
                borderColor: 'border-l-red-500',
                backgroundColor: 'bg-red-50',
                borderAllColor: 'border-red-500',
                textColor: 'text-red-700',
                icon: 'x'
            },
            'warning': {
                borderColor: 'border-l-yellow-500',
                backgroundColor: 'bg-yellow-50',
                borderAllColor: 'border-yellow-500',
                textColor: 'text-yellow-700',
                icon: 'triangle-alert'
            },
            'info': {
                borderColor: 'border-l-blue-500',
                backgroundColor: 'bg-blue-50',
                borderAllColor: 'border-blue-500',
                textColor: 'text-blue-700',
                icon: 'info'
            }
        };

        const config = typeConfig[type];
        toast.className = `${config.backgroundColor} ${config.borderAllColor} border ${config.borderColor} border-l-4 px-4 py-3 rounded-md shadow-lg max-w-sm transform transition-all duration-300 translate-x-full opacity-0`;

        // Create toast content with icon and message
        const iconHtml = await loadIcon(config.icon, 'icon-sm');
        toast.innerHTML = `
            <div class="flex items-center">
                <div class="${config.textColor} flex items-center">${iconHtml}</div>
                <div class="${config.textColor} ml-3 text-sm font-medium">${message}</div>
            </div>
        `;

        toastContainer.appendChild(toast);

        // Trigger enter animation
        setTimeout(() => {
            toast.classList.remove('translate-x-full', 'opacity-0');
        }, 10);

        // Auto-dismiss after 4 seconds
        setTimeout(() => {
            toast.classList.add('translate-x-full', 'opacity-0');
            setTimeout(() => {
                if (toast.parentNode) {
                    toast.parentNode.removeChild(toast);
                }
            }, 300);
        }, 6000);
    };

    // Modal management functions
    window.openModal = function(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.classList.remove('hidden');
            modal.classList.add('flex');
            // Focus management
            const firstInput = modal.querySelector('input, textarea, select, button');
            if (firstInput) {
                firstInput.focus();
            }
        }
    };

    window.closeModal = function(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.classList.add('hidden');
            modal.classList.remove('flex');
        }
    };

    // HTMX event listeners for better UX
    document.body.addEventListener('htmx:beforeRequest', function(event) {
        // Disable buttons during requests, but exclude action buttons (which open modals instantly)
        const target = event.detail.elt;
        if (target.tagName === 'BUTTON' && !target.classList.contains('action-button') && !target.classList.contains('action-button-inline')) {
            target.disabled = true;
        }
    });

    document.body.addEventListener('htmx:afterRequest', function(event) {
        // Restore button state
        const target = event.detail.elt;
        if (target.tagName === 'BUTTON' && !target.classList.contains('action-button') && !target.classList.contains('action-button-inline')) {
            target.disabled = false;
        }

        // Handle successful operations
        if (event.detail.successful) {
            const response = event.detail.xhr.responseText;
            const successMessage = event.detail.xhr.getResponseHeader('HX-Trigger-After-Settle');
            if (successMessage) {
                try {
                    const trigger = JSON.parse(successMessage);
                    if (trigger.showToast) {
                        showToast(trigger.showToast.message, trigger.showToast.type);
                    }
                } catch (e) {
                    // Handle simple string triggers
                    if (successMessage === 'testAuthSuccess') {
                        updateResultIndicator('test-auth-result', 'check', 'success', 'Git authentication successful');
                    } else if (successMessage === 'testAuthError') {
                        updateResultIndicator('test-auth-result', 'x', 'error', 'Git authentication failed');
                    } else if (successMessage === 'discoverSuccess') {
                        updateResultIndicator('discover-result', 'check', 'success', 'Compose files discovered successfully');
                    } else if (successMessage === 'discoverError') {
                        updateResultIndicator('discover-result', 'x', 'error', 'Failed to discover compose files');
                    } else if (successMessage === 'projectCreated') {
                        showToast('Project created successfully', 'success');
                        // Close the modal after successful project creation
                        closeModal('modal-container');
                    } else if (successMessage === 'projectUpdated') {
                        showToast('Project updated successfully', 'success');
                        // Close the modal after successful project update
                        closeModal('modal-container');
                    } else if (successMessage === 'projectDeleted') {
                        showToast('Project deleted successfully', 'success');
                        // Close the modal after successful project deletion
                        closeModal('modal-container');
                    }
                }
            }
        }
    });

    document.body.addEventListener('htmx:responseError', function(event) {
        // Handle error result indicators
        const url = event.detail.requestConfig.path;

        // Only show generic error toast for operations that don't have specific error handling
        if (url !== '/test-git-auth' && url !== '/discover') {
            showToast('Operation failed. Please try again.', 'error');
        }
        if (url === '/discover') {
            updateResultIndicator('discover-result', 'x', 'error', 'Discover failed', false);
        }

        // Restore button state on error
        const target = event.detail.elt;
        if (target.tagName === 'BUTTON' && !target.classList.contains('action-button') && !target.classList.contains('action-button-inline')) {
            target.disabled = false;
        }
    });


    // Form validation for action buttons
    function updateButtonStates() {
        const gitUrlInput = document.getElementById('git_url');
        const authMethodInputs = document.querySelectorAll('input[name="auth_method"]');
        const testAuthBtn = document.getElementById('test-auth-btn');
        const discoverBtn = document.getElementById('discover-btn');

        if (!gitUrlInput || !testAuthBtn || !discoverBtn) return;

        const gitUrl = gitUrlInput.value.trim();
        const selectedAuthMethod = Array.from(authMethodInputs).find(input => input.checked)?.value;

        // Test Git Auth: enabled only if auth method is not "none"
        if (selectedAuthMethod && selectedAuthMethod !== 'none') {
            testAuthBtn.disabled = false;
            testAuthBtn.classList.remove('opacity-50', 'cursor-not-allowed');
        } else {
            testAuthBtn.disabled = true;
            testAuthBtn.classList.add('opacity-50', 'cursor-not-allowed');
        }

        // Discover: enabled only if Git URL is filled
        if (gitUrl) {
            discoverBtn.disabled = false;
            discoverBtn.classList.remove('opacity-50', 'cursor-not-allowed');
        } else {
            discoverBtn.disabled = true;
            discoverBtn.classList.add('opacity-50', 'cursor-not-allowed');
        }
    }

    // Update authentication field requirements based on selected method
    function updateAuthFieldRequirements() {
        const authMethod = document.querySelector('input[name="auth_method"]:checked')?.value;

        // HTTP auth fields
        const usernameField = document.getElementById('username');
        const passwordField = document.getElementById('password');

        // SSH auth fields
        const sshUsernameField = document.getElementById('ssh_username');
        const privateKeyField = document.getElementById('private_key');

        // Remove required attribute from all auth fields
        [usernameField, passwordField, sshUsernameField, privateKeyField].forEach(field => {
            if (field) {
                field.removeAttribute('required');
            }
        });

        // Add required attribute based on selected auth method
        if (authMethod === 'http') {
            if (usernameField) usernameField.setAttribute('required', '');
            if (passwordField) passwordField.setAttribute('required', '');
        } else if (authMethod === 'ssh') {
            if (sshUsernameField) sshUsernameField.setAttribute('required', '');
            if (privateKeyField) privateKeyField.setAttribute('required', '');
        }
    }

    // Set up event listeners for form validation
    document.addEventListener('input', function(event) {
        if (event.target.id === 'git_url') {
            updateButtonStates();
        }
    });

    document.addEventListener('change', function(event) {
        if (event.target.name === 'auth_method') {
            updateButtonStates();
            updateAuthFieldRequirements();
        }
    });

    // Initialize button states when modal opens
    document.addEventListener('htmx:afterSwap', function(event) {
        // Check if modal content was swapped (either modal-panel or modal-container)
        if (event.target.closest('.modal-panel') || event.target.id === 'modal-container' || event.target.querySelector('.modal-panel')) {
            updateButtonStates();
            updateAuthFieldRequirements();
        }
    });

    // Helper function to validate required DOM elements
    function validateElements(elementIds) {
        const elements = {};
        for (const [key, id] of Object.entries(elementIds)) {
            elements[key] = document.getElementById(id);
            if (!elements[key]) {
                console.error(`${key} element not found: ${id}`);
                return null;
            }
        }
        return elements;
    }

    // Helper function to update result indicators
    async function updateResultIndicator(resultId, iconName, className, message, shouldShowToast = true) {
        const resultElement = document.getElementById(resultId);
        if (resultElement) {
            const iconHtml = await loadIcon(iconName, 'icon-sm');
            resultElement.innerHTML = iconHtml;
            resultElement.className = `result-indicator ${className}`;
            resultElement.classList.remove('hidden');
        }
        if (shouldShowToast) {
            showToast(message, className === 'success' ? 'success' : 'error');
        }
    }

    // Helper function to process server-sent events
    function processServerSentEvents(reader, decoder, contentElement, outputElement, onComplete) {
        let buffer = '';
        let hasError = false; // Track if we've seen any error messages

        function readChunk() {
            return reader.read().then(({ done, value }) => {
                if (done) {
                    onComplete(hasError); // Pass error state to completion handler
                    return;
                }

                buffer += decoder.decode(value, { stream: true });

                // Process complete lines
                let lines = buffer.split('\n');
                buffer = lines.pop(); // Keep incomplete line in buffer

                lines.forEach(line => {
                    if (line.startsWith('data: ')) {
                        try {
                            const data = JSON.parse(line.substring(6));

                            switch(data.type) {
                                case 'status':
                                    // Status updates handled by onComplete
                                    break;

                                case 'docker':
                                case 'info':
                                case 'success':
                                case 'error':
                                    // All messages now come with proper type field
                                    const messageType = data.type;
                                    const displayMessage = data.message;

                                    // Track error state
                                    if (messageType === 'error') {
                                        hasError = true;
                                    }

                                    // Style based on message type
                                    let cssClass;
                                    switch(messageType) {
                                        case 'info':
                                            cssClass = 'deploy-text-info';
                                            break;
                                        case 'success':
                                            cssClass = 'deploy-text-success';
                                            break;
                                        case 'error':
                                            cssClass = 'deploy-text-error';
                                            break;
                                        case 'docker':
                                        default:
                                            cssClass = 'deploy-text-backend';
                                            break;
                                    }

                                    const escapedMessage = displayMessage.replace(/</g, '&lt;').replace(/>/g, '&gt;');
                                    contentElement.innerHTML += `<span class="${cssClass}">${escapedMessage}</span>\n`;
                                    // Auto-scroll to bottom
                                    if (outputElement) {
                                        outputElement.scrollTop = outputElement.scrollHeight;
                                    }
                                    break;

                                case 'output':
                                    // Legacy fallback for any remaining 'output' type messages
                                    const legacyMessage = data.message;
                                    const escapedLegacyMessage = legacyMessage.replace(/</g, '&lt;').replace(/>/g, '&gt;');
                                    contentElement.innerHTML += `<span class="deploy-text-backend">${escapedLegacyMessage}</span>\n`;
                                    if (outputElement) {
                                        outputElement.scrollTop = outputElement.scrollHeight;
                                    }
                                    break;

                                case 'complete':
                                    // Complete message handled by onComplete
                                    if (outputElement) {
                                        outputElement.scrollTop = outputElement.scrollHeight;
                                    }
                                    break;
                            }
                        } catch (error) {
                            console.error('Error parsing streaming data:', error);
                        }
                    }
                });

                return readChunk();
            });
        }

        return readChunk();
    }

    // Generic streaming function
    function createStreamingHandler(config) {
        return function(projectId) {
            const elements = validateElements({
                button: config.btnId,
                content: config.contentId,
                output: config.outputId
            });

            if (!elements) return;

            // Update button state
            elements.button.disabled = true;

            // Append connecting message (preserve existing content)
            elements.content.innerHTML += `\n<span class="deploy-text-frontend-generic">${config.connectingMsg}</span>\n`;
            elements.content.className = 'streaming-output';

            // Create AbortController if needed (for logs)
            let controller = null;
            if (config.useAbortController) {
                // Cancel existing controller if exists
                if (currentLogsController) {
                    currentLogsController.abort();
                }
                controller = new AbortController();
                currentLogsController = controller;
            }

            // Start streaming with POST fetch
            const fetchOptions = {
                method: 'POST',
                headers: {
                    'Accept': 'text/event-stream',
                    'Cache-Control': 'no-cache'
                }
            };

            if (controller) {
                fetchOptions.signal = controller.signal;
            }

            fetch(config.endpoint(projectId), fetchOptions)
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                elements.content.innerHTML += `<span class="deploy-text-frontend-generic">${config.startingMsg}</span>\n`;
                elements.content.className = 'streaming-output';

                const reader = response.body.getReader();
                const decoder = new TextDecoder();

                return processServerSentEvents(reader, decoder, elements.content, elements.output, (hasError) => {
                    elements.button.disabled = false;

                    // Check if operation was successful based on tracked error state
                    if (hasError) {
                        elements.content.innerHTML += `\n<span class="deploy-text-frontend-error">${config.errorMsg}</span>\n`;
                        showToast(config.errorMsg, 'error');
                    } else {
                        elements.content.innerHTML += `\n<span class="deploy-text-frontend-success">${config.successMsg}</span>\n`;
                        showToast(config.successMsg, 'success');
                    }

                    // Update status pill for deploy/stop operations (regardless of success/failure)
                    if (config.updateStatus) {
                        updateProjectStatus(projectId);
                    }

                    if (elements.output) {
                        elements.output.scrollTop = elements.output.scrollHeight;
                    }
                });
            })
            .catch(error => {
                // Don't show error if stream was intentionally cancelled
                if (error.name === 'AbortError') {
                    console.log(`${config.name} streaming was cancelled`);
                    return;
                }

                console.error(`${config.name} streaming error:`, error);
                elements.button.disabled = false;
                elements.content.innerHTML += `\n<span class="deploy-text-frontend-error">ERROR: Connection to ${config.name.toLowerCase()} stream failed</span>\n`;
                showToast(`${config.name} connection failed`, 'error');
            });
        };
    }

    // Event delegation for deploy and stop buttons
    document.addEventListener('click', function(event) {
        if (event.target.id === 'deploy-btn' && event.target.dataset.projectId) {
            event.preventDefault();
            startDeployment(event.target.dataset.projectId);
        } else if (event.target.id === 'stop-btn' && event.target.dataset.projectId) {
            event.preventDefault();
            startStop(event.target.dataset.projectId);
        }
    });

    // Create streaming function configurations
    const deployConfig = {
        name: 'Deployment',
        btnId: 'deploy-btn',
        contentId: 'deploy-content',
        outputId: 'deploy-output',
        endpoint: (projectId) => `/projects/${projectId}/deploy/stream`,
        connectingMsg: 'Connecting to deployment stream...',
        startingMsg: 'Starting deployment...',
        successMsg: 'Deployment completed successfully',
        errorMsg: 'Deployment failed',
        updateStatus: true,
        useAbortController: false
    };

    const stopConfig = {
        name: 'Stop',
        btnId: 'stop-btn',
        contentId: 'stop-content',
        outputId: 'stop-output',
        endpoint: (projectId) => `/projects/${projectId}/stop/stream`,
        connectingMsg: 'Connecting to stop stream...',
        startingMsg: 'Starting stop process...',
        successMsg: 'Stop completed successfully',
        errorMsg: 'Stop failed',
        updateStatus: true,
        useAbortController: false
    };


    // Deployment streaming functionality
    window.startDeployment = createStreamingHandler(deployConfig);

    // Stop streaming functionality
    window.startStop = createStreamingHandler(stopConfig);


    // Event delegation for deployment output buttons
    document.addEventListener('click', function(e) {
        if (e.target.closest('.deployment-output-btn')) {
            const button = e.target.closest('.deployment-output-btn');
            const deploymentId = button.getAttribute('data-deployment-id');
            const output = button.getAttribute('data-deployment-output');
            showDeploymentOutput(deploymentId, output);
        }
    });

    // Show deployment output in a popup
    window.showDeploymentOutput = function(deploymentId, output) {
        // Create modal overlay without backdrop (since another modal is already active)
        const overlay = document.createElement('div');
        overlay.className = 'fixed inset-0 z-60 overflow-y-auto';
        overlay.innerHTML = `
            <div class="flex min-h-full items-center justify-center p-4 text-center">
                <div class="relative transform overflow-hidden rounded-lg bg-white px-6 pb-6 pt-6 text-left shadow-xl transition-all sm:my-4 sm:w-full sm:max-w-2xl sm:p-8">
                    <div class="flex justify-between items-center mb-4">
                        <h3 class="text-lg font-semibold text-gray-900">Deployment Output</h3>
                        <button type="button" class="text-gray-400 hover:text-gray-600 focus:outline-none" onclick="this.closest('.fixed').remove()">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                            </svg>
                        </button>
                    </div>
                    <div class="deployment-output-container">
                        <div class="deploy-code-block" style="max-height: 400px;">
                            <pre class="streaming-output">${output ? output.replace(/</g, '&lt;').replace(/>/g, '&gt;') : 'No output available'}</pre>
                        </div>
                    </div>
                </div>
            </div>
        `;

        document.body.appendChild(overlay);

        // Close on background click (clicking outside the modal content)
        overlay.addEventListener('click', function(e) {
            if (e.target === overlay) {
                overlay.remove();
            }
        });
    };

    // Fullscreen modal toggle functionality
    window.toggleModalFullscreen = async function() {
        const modal = document.getElementById('modal-container');
        const toggleButton = document.querySelector('.fullscreen-toggle');

        if (!modal || !toggleButton) {
            return;
        }

        const isFullscreen = modal.classList.contains('fullscreen');

        if (isFullscreen) {
            // Exit fullscreen
            modal.classList.remove('fullscreen');
            toggleButton.innerHTML = await loadIcon('maximize', 'w-5 h-5');
        } else {
            // Enter fullscreen
            modal.classList.add('fullscreen');
            toggleButton.innerHTML = await loadIcon('minimize', 'w-5 h-5');
        }
    };

    // Global keyboard shortcuts
    document.addEventListener('keydown', function(event) {
        // Modals can only be closed with X button, not Escape key
        // (Escape key modal closing is disabled per requirements)

        // Ctrl+Enter submits forms
        if (event.ctrlKey && event.key === 'Enter') {
            const activeForm = document.activeElement.closest('form');
            if (activeForm) {
                const submitButton = activeForm.querySelector('button[type="submit"]');
                if (submitButton) {
                    submitButton.click();
                }
            }
        }
    });
});
