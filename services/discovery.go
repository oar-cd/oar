package services

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/google/uuid"
)

// ComposeFile represents a discovered Docker Compose file
type ComposeFile struct {
	Path string // Relative path from git root
}

// EnvFile represents a discovered env file
type EnvFile struct {
	Path string // Relative path from git root
}

// DiscoveryResponse represents the result of file discovery
type DiscoveryResponse struct {
	ComposeFiles  []ComposeFile // Discovered compose files
	EnvFiles      []EnvFile     // Optional env files
	TempClonePath string        // Full filesystem path to temp clone
}

// ProjectDiscoveryService handles on-demand file discovery for project creation
type ProjectDiscoveryService struct {
	gitService GitExecutor
	config     *Config
}

// NewProjectDiscoveryService creates a new project discovery service
func NewProjectDiscoveryService(
	gitService GitExecutor,
	config *Config,
) *ProjectDiscoveryService {
	return &ProjectDiscoveryService{
		gitService: gitService,
		config:     config,
	}
}

// DiscoverFiles clones repository to temp location and discovers compose files
func (s *ProjectDiscoveryService) DiscoverFiles(gitURL string) (*DiscoveryResponse, error) {
	if gitURL == "" {
		return nil, fmt.Errorf("git URL is required")
	}

	slog.Info("Starting file discovery",
		"git_url", gitURL)

	// Create temporary directory for discovery
	tempID := uuid.New().String()
	tempDir := filepath.Join(s.config.TmpDir, "discovery-"+tempID)

	// Clone repository to temp location (using no auth for discovery)
	// TODO: Support authentication during discovery phase for private repositories
	if err := s.gitService.Clone(gitURL, nil, tempDir); err != nil {
		slog.Error("Service operation failed",
			"layer", "service",
			"operation", "discover_files",
			"git_url", gitURL,
			"error", err)
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Discover compose files in temp location
	composeFiles, err := s.DiscoverComposeFiles(tempDir)
	if err != nil {
		// Clean up temp directory on error
		if err := os.RemoveAll(tempDir); err != nil {
			slog.Error("Failed to clean up temp directory",
				"layer", "service",
				"operation", "discover_files_cleanup",
				"git_url", gitURL,
				"temp_dir", tempDir,
				"error", err)
		}

		slog.Error("Service operation failed",
			"layer", "service",
			"operation", "discover_files",
			"git_url", gitURL,
			"temp_dir", tempDir,
			"error", err)
		return nil, fmt.Errorf("failed to discover files: %w", err)
	}

	slog.Info("File discovery completed",
		"git_url", gitURL,
		"temp_id", tempID,
		"files_found", len(composeFiles))

	return &DiscoveryResponse{
		ComposeFiles:  composeFiles,
		EnvFiles:      []EnvFile{}, // TODO: Add env file discovery
		TempClonePath: tempDir,
	}, nil
}

// DiscoverComposeFiles finds and validates Docker Compose files in the given directory
func (s *ProjectDiscoveryService) DiscoverComposeFiles(rootDir string) ([]ComposeFile, error) {
	var discoveries []ComposeFile

	slog.Debug("Starting compose file discovery", "root_dir", rootDir)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			slog.Error("Discovery operation failed",
				"layer", "service",
				"operation", "discover_compose_files",
				"path", path,
				"error", err)
			return err
		}

		// Skip directories we don't want to traverse
		if d.IsDir() {
			if s.shouldSkipDir(d.Name()) {
				slog.Debug("Skipping directory", "dir", d.Name())
				return filepath.SkipDir
			}
			// Allow processing of non-skipped directories
			return nil
		}

		// Process YAML files (we know it's not a directory at this point)
		if s.isYAMLFile(path) {
			relPath, err := filepath.Rel(rootDir, path)
			if err != nil {
				slog.Error("Discovery operation failed",
					"layer", "service",
					"operation", "discover_compose_files",
					"path", path,
					"error", err)
				return nil // Continue processing other files
			}

			slog.Debug("Found YAML file", "path", relPath)

			// Validate if it's a compose file using compose-go
			if s.isValidComposeFile(path) {
				discovery := ComposeFile{
					Path: relPath,
				}
				discoveries = append(discoveries, discovery)
				slog.Debug("Valid compose file found", "path", relPath)
			} else {
				slog.Debug("YAML file is not a valid compose file", "path", relPath)
			}
		}

		return nil
	})
	if err != nil {
		slog.Error("Discovery operation failed",
			"layer", "service",
			"operation", "discover_compose_files",
			"root_dir", rootDir,
			"error", err)
		return nil, err
	}

	slog.Info("Compose file discovery completed",
		"root_dir", rootDir,
		"files_found", len(discoveries))

	return discoveries, nil
}

// shouldSkipDir returns true if the directory should be skipped during traversal
func (s *ProjectDiscoveryService) shouldSkipDir(dirName string) bool {
	// Skip hidden directories (including .git, .github, .vscode, etc.)
	return strings.HasPrefix(dirName, ".")
}

// isYAMLFile returns true if the file has a YAML extension
func (s *ProjectDiscoveryService) isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yml" || ext == ".yaml"
}

// isValidComposeFile validates if a YAML file is a valid Docker Compose file using compose-go v2
func (s *ProjectDiscoveryService) isValidComposeFile(path string) bool {
	slog.Debug("Validating compose file", "path", path)

	ctx := context.Background()

	// Create project options without specifying a name
	options, err := cli.NewProjectOptions([]string{path})
	if err != nil {
		slog.Debug("Failed to create project options", "path", path, "error", err)
		return false
	}

	// Try to load and validate the project
	_, err = options.LoadProject(ctx)
	if err != nil {
		slog.Debug("File is not a valid compose file", "path", path, "error", err)
		return false
	}

	slog.Debug("File is a valid compose file", "path", path)
	return true
}

// Future: Add DiscoverEnvFiles method here
// func (s *ProjectDiscoveryService) DiscoverEnvFiles(rootDir string) ([]EnvFile, error) {
//     // Implementation for env file discovery
// }
