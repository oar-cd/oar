/**
 * ProjectStateManager - Manages project state storage and UI updates
 */
class ProjectStateManager {
    constructor() {
        this.states = {};
    }

    /**
     * Store project state for later use
     */
    storeProjectState(projectId, projectState) {
        if (!projectState) return;

        this.states[projectId] = projectState;
    }

    /**
     * Get stored project state
     */
    getProjectState(projectId) {
        return this.states[projectId] || null;
    }

    /**
     * Update project card using stored state information
     */
    updateProjectCardWithState(projectId) {
        const projectState = this.getProjectState(projectId);
        if (!projectState) {
            console.warn('No stored project state found for project:', projectId);
            return;
        }

        // Update the status badge
        if (window.UIUpdater) {
            window.UIUpdater.updateProjectCardStatus(projectId, projectState.status);

            // Update last commit if provided
            if (projectState.lastCommit) {
                window.UIUpdater.updateProjectCardCommit(projectId, projectState.lastCommit);
            }

            // Update action button visibility based on new status
            window.UIUpdater.updateProjectActionButtons(projectId, projectState.status);
        }

        // Clean up stored state
        this.clearProjectState(projectId);
    }

    /**
     * Clear stored state for a project
     */
    clearProjectState(projectId) {
        delete this.states[projectId];
    }

    /**
     * Clear all stored states
     */
    clearAllStates() {
        this.states = {};
    }

    /**
     * Get all stored project IDs
     */
    getStoredProjectIds() {
        return Object.keys(this.states);
    }

    /**
     * Check if project has stored state
     */
    hasProjectState(projectId) {
        return projectId in this.states;
    }

    /**
     * Validate project state structure
     */
    validateProjectState(projectState) {
        if (!projectState || typeof projectState !== 'object') {
            return false;
        }

        // Required fields
        if (!projectState.status) {
            return false;
        }

        // Valid status values
        const validStatuses = ['running', 'stopped', 'error', 'unknown'];
        if (!validStatuses.includes(projectState.status)) {
            return false;
        }

        return true;
    }

    /**
     * Store project state with validation
     */
    storeValidatedProjectState(projectId, projectState) {
        if (!this.validateProjectState(projectState)) {
            console.warn('Invalid project state provided for project:', projectId, projectState);
            return false;
        }

        this.storeProjectState(projectId, projectState);
        return true;
    }
}

// Export for module systems or global access
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ProjectStateManager;
} else {
    window.ProjectStateManager = ProjectStateManager;
    // Create global instance for easy access
    window.projectStateManager = new ProjectStateManager();
}