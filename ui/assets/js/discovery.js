// Project discovery functionality for the home page

function validateAndDiscover() {
    const gitUrlInput = document.getElementById('new-project-git-url');
    if (!gitUrlInput.checkValidity()) {
        gitUrlInput.reportValidity();
        return; // Stop execution, don't trigger HTMX
    }

    // Show discovery loading state
    const indicator = document.getElementById('discover-indicator');
    const icon = document.getElementById('discover-icon');
    if (indicator && icon) {
        indicator.classList.remove('hidden');
        icon.classList.add('hidden');
    }

    // Collect authentication fields
    const authTypeInput = document.querySelector('input[name="auth_type"]');
    const authType = authTypeInput ? authTypeInput.value : 'none';

    const values = {
        git_url: gitUrlInput.value,
        auth_type: authType,
    };

    // Add auth-specific fields based on type
    if (authType === 'http') {
        const usernameInput = document.getElementById('new-project-username');
        const passwordInput = document.getElementById('new-project-password');
        if (usernameInput && passwordInput) {
            values.username = usernameInput.value;
            values.password = passwordInput.value;
        }
    } else if (authType === 'ssh') {
        const sshUserInput = document.getElementById('new-project-ssh-user');
        const sshKeyInput = document.getElementById('new-project-ssh-key');
        if (sshUserInput && sshKeyInput) {
            values.ssh_user = sshUserInput.value;
            values.ssh_private_key = sshKeyInput.value;
        }
    }

    // Manually trigger HTMX request
    htmx.ajax('POST', '/discover', {
        source: '#discover-button',
        target: '#compose-files-section',
        swap: 'innerHTML',
        values: values,
    })
        .then(function () {
            // Hide discovery loading state when done
            if (indicator && icon) {
                indicator.classList.add('hidden');
                icon.classList.remove('hidden');
            }
        })
        .catch(function () {
            // Hide discovery loading state on error too
            if (indicator && icon) {
                indicator.classList.add('hidden');
                icon.classList.remove('hidden');
            }
        });
}

function checkDiscoveryButtonState() {
    const gitUrlInput = document.getElementById('new-project-git-url');
    const discoverButton = document.getElementById('discover-button');

    if (gitUrlInput && discoverButton) {
        const isValid = gitUrlInput.value.trim() !== '';
        discoverButton.disabled = !isValid;
        if (isValid) {
            discoverButton.classList.remove('opacity-50', 'cursor-not-allowed');
        } else {
            discoverButton.classList.add('opacity-50', 'cursor-not-allowed');
        }
    }
}

function checkTestGitAuthButtonState() {
    const gitUrlInput = document.getElementById('new-project-git-url');
    const authTypeInput = document.querySelector('input[name="auth_type"]');
    const testButton = document.getElementById('test-git-auth-button');

    if (!gitUrlInput || !authTypeInput || !testButton) return;

    const gitUrl = gitUrlInput.value.trim();
    const authType = authTypeInput.value;
    let authFieldsValid = false;

    if (authType === 'http') {
        const usernameInput = document.getElementById('new-project-username');
        const passwordInput = document.getElementById('new-project-password');
        authFieldsValid =
            usernameInput &&
            passwordInput &&
            usernameInput.value.trim() !== '' &&
            passwordInput.value.trim() !== '';
    } else if (authType === 'ssh') {
        const sshKeyInput = document.getElementById('new-project-ssh-key');
        authFieldsValid = sshKeyInput && sshKeyInput.value.trim() !== '';
    }

    const isValid = gitUrl !== '' && authType !== 'none' && authFieldsValid;
    testButton.disabled = !isValid;
    if (isValid) {
        testButton.classList.remove('opacity-50', 'cursor-not-allowed');
    } else {
        testButton.classList.add('opacity-50', 'cursor-not-allowed');
    }
}

// Initialize button states on page load
document.addEventListener('DOMContentLoaded', function () {
    checkDiscoveryButtonState();
    checkTestGitAuthButtonState();

    // Add event listeners for form field changes
    const gitUrlInput = document.getElementById('new-project-git-url');
    if (gitUrlInput) {
        gitUrlInput.addEventListener('input', function () {
            checkDiscoveryButtonState();
            checkTestGitAuthButtonState();
        });
    }

    // Listen for auth type changes
    const authTypeInput = document.querySelector('input[name="auth_type"]');
    if (authTypeInput) {
        // Use MutationObserver to watch for value changes since it's managed by selectbox
        const observer = new MutationObserver(function () {
            checkTestGitAuthButtonState();
        });
        observer.observe(authTypeInput, {
            attributes: true,
            attributeFilter: ['value'],
        });

        // Also listen to the parent selectbox for changes
        const selectboxTrigger = document.getElementById(
            'new-project-auth-type',
        );
        if (selectboxTrigger) {
            selectboxTrigger.addEventListener(
                'change',
                checkTestGitAuthButtonState,
            );
        }
    }

    const usernameInput = document.getElementById('new-project-username');
    const passwordInput = document.getElementById('new-project-password');
    const sshKeyInput = document.getElementById('new-project-ssh-key');

    if (usernameInput) {
        usernameInput.addEventListener('input', checkTestGitAuthButtonState);
    }
    if (passwordInput) {
        passwordInput.addEventListener('input', checkTestGitAuthButtonState);
    }
    if (sshKeyInput) {
        sshKeyInput.addEventListener('input', checkTestGitAuthButtonState);
    }
});

function validateAndTestGitAuth() {
    // Show test git auth loading state
    const indicator = document.getElementById('test-git-auth-indicator');
    const icon = document.getElementById('test-git-auth-icon');
    if (indicator && icon) {
        indicator.classList.remove('hidden');
        icon.classList.add('hidden');
    }

    // Collect authentication fields
    const gitUrlInput = document.getElementById('new-project-git-url');
    const authTypeInput = document.querySelector('input[name="auth_type"]');
    const authType = authTypeInput ? authTypeInput.value : 'none';

    const values = {
        git_url: gitUrlInput.value,
        auth_type: authType,
    };

    // Add auth-specific fields based on type
    if (authType === 'http') {
        const usernameInput = document.getElementById('new-project-username');
        const passwordInput = document.getElementById('new-project-password');
        if (usernameInput && passwordInput) {
            values.username = usernameInput.value;
            values.password = passwordInput.value;
        }
    } else if (authType === 'ssh') {
        const sshUserInput = document.getElementById('new-project-ssh-user');
        const sshKeyInput = document.getElementById('new-project-ssh-key');
        if (sshUserInput && sshKeyInput) {
            values.ssh_user = sshUserInput.value;
            values.ssh_private_key = sshKeyInput.value;
        }
    }

    // Create a temporary target div for the test result
    let testResultDiv = document.getElementById('test-git-auth-result');
    if (!testResultDiv) {
        testResultDiv = document.createElement('div');
        testResultDiv.id = 'test-git-auth-result';
        testResultDiv.style.display = 'none';
        document.body.appendChild(testResultDiv);
    }

    // Manually trigger HTMX request to test git auth
    htmx.ajax('POST', '/test-git-auth', {
        source: '#test-git-auth-button',
        target: '#test-git-auth-result',
        swap: 'innerHTML',
        values: values,
    })
        .then(function () {
            // Hide test git auth loading state when done
            if (indicator && icon) {
                indicator.classList.add('hidden');
                icon.classList.remove('hidden');
            }
            // Show the result in an alert or toast
            const result = testResultDiv.textContent || testResultDiv.innerText;
            if (result.includes('success') || result.includes('Success')) {
                alert('🟢 Authentication successful');
            } else {
                alert('🔴 Authentication failed: ' + result);
            }
        })
        .catch(function (error) {
            // Hide test git auth loading state on error too
            if (indicator && icon) {
                indicator.classList.add('hidden');
                icon.classList.remove('hidden');
            }
            alert(
                '🔴 Authentication failed: ' +
                    (error.message || 'Unknown error'),
            );
        });
}

function resetAddProjectForm() {
    // Reset the form
    const form = document.getElementById('add-project-form');
    if (form) {
        form.reset();
    }

    // Clear any discovered files from the compose files section, but keep the default fields
    // Only clear if it contains discovered content (check for specific discovered elements)
    const composeFilesSection = document.getElementById(
        'compose-files-section',
    );
    if (
        composeFilesSection &&
        composeFilesSection.querySelector('.discovered-files')
    ) {
        // Only clear discovered content, the default form fields should remain
        const discoveredElements = composeFilesSection.querySelectorAll(
            '.discovered-files, .discovery-error',
        );
        discoveredElements.forEach((el) => el.remove());
    }

    // Reset auth type to default
    const authTypeInput = document.querySelector('input[name="auth_type"]');
    if (authTypeInput) {
        authTypeInput.value = 'none';
    }

    // Hide auth fields
    const httpAuthFields = document.getElementById('http-auth-fields');
    const sshAuthFields = document.getElementById('ssh-auth-fields');
    if (httpAuthFields) {
        httpAuthFields.style.display = 'none';
    }
    if (sshAuthFields) {
        sshAuthFields.style.display = 'none';
    }

    // Update button states
    checkDiscoveryButtonState();
    checkTestGitAuthButtonState();
}
