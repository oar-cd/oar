// Package update provides the update command for Oar.
package update

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ch00k/oar/cmd/output"
	"github.com/spf13/cobra"
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func NewCmdUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update Oar to the latest version",
		Long:  `Update Oar installation to the latest release version.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate()
		},
	}

	return cmd
}

func runUpdate() error {
	// Get installation directory from XDG spec
	oarDir := getOarDir()
	versionFile := filepath.Join(oarDir, "VERSION")

	// Read current version
	currentVersion, err := readCurrentVersion(versionFile)
	if err != nil {
		fmt.Print(output.PrintMessage(output.Warning, "Could not determine current version: %v", err))
		currentVersion = "unknown"
	}

	// Get latest release info
	fmt.Print(output.PrintMessage(output.Plain, "Checking for updates..."))
	release, err := getLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	// Check if update is needed
	if currentVersion == release.TagName {
		fmt.Print(output.PrintMessage(output.Success, "Already on latest version (%s)", currentVersion))
		return nil
	}

	fmt.Print(output.PrintMessage(output.Plain, "Update available: %s → %s", currentVersion, release.TagName))

	// Update compose.yaml
	if err := updateComposeFile(oarDir, release); err != nil {
		return fmt.Errorf("failed to update compose file: %w", err)
	}

	// Update CLI binary
	if err := updateCLIBinary(release); err != nil {
		return fmt.Errorf("failed to update CLI binary: %w", err)
	}

	// Restart Docker Compose
	if err := restartDockerCompose(oarDir); err != nil {
		return fmt.Errorf("failed to restart Oar: %w", err)
	}

	// Update version file
	if err := writeVersion(versionFile, release.TagName); err != nil {
		fmt.Print(output.PrintMessage(output.Warning, "Failed to update version file: %v", err))
	}

	fmt.Print(output.PrintMessage(output.Success, "Successfully updated to %s!", release.TagName))
	return nil
}

func getOarDir() string {
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return filepath.Join(xdgData, "oar")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "oar")
}

func readCurrentVersion(versionFile string) (string, error) {
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func writeVersion(versionFile, version string) error {
	return os.WriteFile(versionFile, []byte(version), 0o644)
}

func getLatestRelease() (*GitHubRelease, error) {
	resp, err := http.Get("https://api.github.com/repos/ch00k/oar/releases/latest")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("Failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func updateComposeFile(oarDir string, release *GitHubRelease) error {
	fmt.Print(output.PrintMessage(output.Plain, "Updating compose.yaml..."))

	var composeURL string
	for _, asset := range release.Assets {
		if asset.Name == "compose.yaml" {
			composeURL = asset.BrowserDownloadURL
			break
		}
	}

	if composeURL == "" {
		return fmt.Errorf("compose.yaml not found in release assets")
	}

	return downloadFile(composeURL, filepath.Join(oarDir, "compose.yaml"))
}

func updateCLIBinary(release *GitHubRelease) error {
	fmt.Print(output.PrintMessage(output.Plain, "Updating CLI binary..."))

	// Determine binary name based on platform
	binaryName := fmt.Sprintf("oar-%s-%s", runtime.GOOS, runtime.GOARCH)

	var binaryURL string
	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			binaryURL = asset.BrowserDownloadURL
			break
		}
	}

	if binaryURL == "" {
		return fmt.Errorf("binary %s not found in release assets", binaryName)
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}

	// Download to temporary file first
	tempFile := execPath + ".tmp"
	if err := downloadFile(binaryURL, tempFile); err != nil {
		return err
	}

	// Make executable
	if err := os.Chmod(tempFile, 0o755); err != nil {
		if err := os.Remove(tempFile); err != nil {
			return fmt.Errorf("failed to remove temporary file: %w", err)
		}
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Replace current binary
	if err := os.Rename(tempFile, execPath); err != nil {
		if err := os.Remove(tempFile); err != nil {
			return fmt.Errorf("failed to remove temporary file after rename failure: %w", err)
		}
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	return nil
}

func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("Failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil {
			slog.Warn("Failed to close file", "file_path", filepath, "error", cerr)
		}
	}()

	_, err = io.Copy(out, resp.Body)
	return err
}

func restartDockerCompose(oarDir string) error {
	fmt.Print(output.PrintMessage(output.Plain, "Restarting Oar..."))

	// Change to oar directory
	oldDir, err := os.Getwd()
	if err != nil {
		return err
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			slog.Warn("Failed to restore working directory", "target_dir", oldDir, "error", err)
		}
	}()

	if err := os.Chdir(oarDir); err != nil {
		return fmt.Errorf("failed to change to oar directory: %w", err)
	}

	// Stop current containers
	stopCmd := exec.Command("docker", "compose", "--project-name", "oar", "down")
	if err := stopCmd.Run(); err != nil {
		fmt.Print(output.PrintMessage(output.Warning, "Failed to stop containers: %v", err))
	}

	// Start with new version
	startCmd := exec.Command("docker", "compose", "--project-name", "oar", "up", "-d")
	if err := startCmd.Run(); err != nil {
		return fmt.Errorf("failed to start containers: %w", err)
	}

	return nil
}
