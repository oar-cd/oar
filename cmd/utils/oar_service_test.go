package utils

import (
	"testing"

	"github.com/ch00k/oar/services"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCreateOarServiceComposeProject_Structure(t *testing.T) {
	// Create a test command
	cmd := &cobra.Command{}

	// Test the function - this may fail if compose.yaml doesn't exist, but that's expected
	projectInterface, err := CreateOarServiceComposeProject(cmd)

	if err != nil {
		// If there's an error, it should be about missing compose.yaml
		assert.Contains(t, err.Error(), "oar compose.yaml not found")
		assert.Nil(t, projectInterface)
	} else {
		// If successful, verify the project structure by type asserting to the concrete type
		project, ok := projectInterface.(*services.ComposeProject)
		assert.True(t, ok, "Expected *services.ComposeProject")
		assert.Equal(t, "oar", project.Name)
		assert.Equal(t, []string{"compose.yaml"}, project.ComposeFiles)
		assert.Empty(t, project.Variables)
		assert.NotNil(t, project.Config)
		assert.Equal(t, "docker", project.Config.DockerCommand)
		assert.Equal(t, "unix:///var/run/docker.sock", project.Config.DockerHost)
	}
}
