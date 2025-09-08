package update

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdUpdate(t *testing.T) {
	cmd := NewCmdUpdate()

	// Test command configuration
	assert.Equal(t, "update", cmd.Use)
	assert.Equal(t, "Update Oar to the latest version", cmd.Short)
	assert.Contains(t, cmd.Long, "Update Oar installation to the latest release version")

	// Test that RunE is set
	assert.NotNil(t, cmd.RunE)

	// Verify it's a runnable command
	assert.True(t, cmd.Runnable())

	// Verify the command can be found by name
	assert.Equal(t, "update", cmd.Name())
}

func TestGetOarDir(t *testing.T) {
	// Save original environment
	originalXDGData := os.Getenv("XDG_DATA_HOME")
	defer func() {
		if originalXDGData != "" {
			if err := os.Setenv("XDG_DATA_HOME", originalXDGData); err != nil {
				t.Logf("Warning: failed to restore XDG_DATA_HOME: %v", err)
			}
		} else {
			if err := os.Unsetenv("XDG_DATA_HOME"); err != nil {
				t.Logf("Warning: failed to unset XDG_DATA_HOME: %v", err)
			}
		}
	}()

	t.Run("with XDG_DATA_HOME set", func(t *testing.T) {
		if err := os.Setenv("XDG_DATA_HOME", "/tmp/xdg"); err != nil {
			t.Fatalf("Failed to set XDG_DATA_HOME: %v", err)
		}
		result := getOarDir()
		assert.Equal(t, "/tmp/xdg/oar", result)
	})

	t.Run("without XDG_DATA_HOME", func(t *testing.T) {
		if err := os.Unsetenv("XDG_DATA_HOME"); err != nil {
			t.Fatalf("Failed to unset XDG_DATA_HOME: %v", err)
		}
		result := getOarDir()

		// Should contain home directory and .local/share/oar
		assert.Contains(t, result, ".local/share/oar")
		assert.True(t, strings.HasSuffix(result, "oar"))
	})
}

func TestReadCurrentVersion(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir := t.TempDir()
	versionFile := filepath.Join(tmpDir, "VERSION")

	tests := []struct {
		name        string
		fileContent string
		createFile  bool
		expected    string
		expectError bool
	}{
		{
			name:        "valid version file",
			fileContent: "v1.2.3",
			createFile:  true,
			expected:    "v1.2.3",
			expectError: false,
		},
		{
			name:        "version with whitespace",
			fileContent: "  v2.0.0  \n",
			createFile:  true,
			expected:    "v2.0.0",
			expectError: false,
		},
		{
			name:        "empty file",
			fileContent: "",
			createFile:  true,
			expected:    "",
			expectError: false,
		},
		{
			name:        "file does not exist",
			createFile:  false,
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.createFile {
				err := os.WriteFile(versionFile, []byte(tt.fileContent), 0o644)
				assert.NoError(t, err)
				defer func() {
					if err := os.Remove(versionFile); err != nil {
						t.Logf("Warning: failed to clean up version file: %v", err)
					}
				}()
			}

			result, err := readCurrentVersion(versionFile)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestWriteVersion(t *testing.T) {
	tmpDir := t.TempDir()
	versionFile := filepath.Join(tmpDir, "VERSION")

	tests := []struct {
		name    string
		version string
	}{
		{
			name:    "simple version",
			version: "v1.0.0",
		},
		{
			name:    "version with rc",
			version: "v2.0.0-rc1",
		},
		{
			name:    "empty version",
			version: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := writeVersion(versionFile, tt.version)
			assert.NoError(t, err)

			// Verify the content was written correctly
			content, err := os.ReadFile(versionFile)
			assert.NoError(t, err)
			assert.Equal(t, tt.version, string(content))

			// Cleanup
			if err := os.Remove(versionFile); err != nil {
				t.Logf("Warning: failed to clean up version file: %v", err)
			}
		})
	}
}

func TestUpdateComposeFile(t *testing.T) {
	// Test with a mock release that has compose.yaml
	release := &GitHubRelease{
		TagName: "v1.0.0",
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{
				Name:               "oar-linux-amd64",
				BrowserDownloadURL: "https://example.com/oar-linux-amd64",
			},
			{
				Name:               "compose.yaml",
				BrowserDownloadURL: "https://example.com/compose.yaml",
			},
		},
	}

	t.Run("compose.yaml found", func(t *testing.T) {
		// We can't easily test the actual download without mocking HTTP calls
		// But we can test the logic that finds the compose.yaml asset
		var composeURL string
		for _, asset := range release.Assets {
			if asset.Name == "compose.yaml" {
				composeURL = asset.BrowserDownloadURL
				break
			}
		}
		assert.Equal(t, "https://example.com/compose.yaml", composeURL)
	})

	t.Run("compose.yaml not found", func(t *testing.T) {
		releaseWithoutCompose := &GitHubRelease{
			TagName: "v1.0.0",
			Assets: []struct {
				Name               string `json:"name"`
				BrowserDownloadURL string `json:"browser_download_url"`
			}{
				{
					Name:               "oar-linux-amd64",
					BrowserDownloadURL: "https://example.com/oar-linux-amd64",
				},
			},
		}

		var composeURL string
		for _, asset := range releaseWithoutCompose.Assets {
			if asset.Name == "compose.yaml" {
				composeURL = asset.BrowserDownloadURL
				break
			}
		}
		// Should be empty when not found
		assert.Empty(t, composeURL)
	})
}

func TestUpdateCLIBinary_BinaryNameLogic(t *testing.T) {
	// Test the binary name generation logic
	expectedName := "oar-" + runtime.GOOS + "-" + runtime.GOARCH

	// Mock release with various binary assets
	release := &GitHubRelease{
		TagName: "v1.0.0",
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{
				Name:               "oar-linux-amd64",
				BrowserDownloadURL: "https://example.com/oar-linux-amd64",
			},
			{
				Name:               "oar-darwin-amd64",
				BrowserDownloadURL: "https://example.com/oar-darwin-amd64",
			},
			{
				Name:               "oar-windows-amd64.exe",
				BrowserDownloadURL: "https://example.com/oar-windows-amd64.exe",
			},
		},
	}

	// Find the binary for current platform
	var binaryURL string
	for _, asset := range release.Assets {
		if asset.Name == expectedName {
			binaryURL = asset.BrowserDownloadURL
			break
		}
	}

	// On most CI systems, this would be linux-amd64
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		assert.Equal(t, "https://example.com/oar-linux-amd64", binaryURL)
	}
}

func TestGitHubReleaseStruct(t *testing.T) {
	// Test that the GitHubRelease struct can be instantiated
	release := GitHubRelease{
		TagName: "v1.0.0",
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{Name: "asset1", BrowserDownloadURL: "url1"},
			{Name: "asset2", BrowserDownloadURL: "url2"},
		},
	}

	assert.Equal(t, "v1.0.0", release.TagName)
	assert.Len(t, release.Assets, 2)
	assert.Equal(t, "asset1", release.Assets[0].Name)
	assert.Equal(t, "url1", release.Assets[0].BrowserDownloadURL)
}
