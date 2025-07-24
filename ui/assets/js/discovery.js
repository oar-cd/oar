// Project discovery functionality for the home page

function validateAndDiscover() {
    const gitUrlInput = document.getElementById('new-project-git-url');
    if (!gitUrlInput.checkValidity()) {
        gitUrlInput.reportValidity();
        return; // Stop execution, don't trigger HTMX
    }

    // Manually trigger HTMX request
    htmx.ajax('POST', '/discover', {
        source: '#discover-button',
        target: '#compose-files-section',
        swap: 'innerHTML',
        values: {
            git_url: gitUrlInput.value,
        },
    });
}

