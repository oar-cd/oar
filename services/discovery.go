package services

import (
	"context"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/v2/cli"
)

// ComposeFile represents a discovered Docker Compose file
type ComposeFile struct {
	Path string // Relative path from git root
}

// DiscoveryService handles discovery of project files
type DiscoveryService struct{}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService() *DiscoveryService {
	return &DiscoveryService{}
}

// DiscoverComposeFiles finds and validates Docker Compose files in the given directory
func (s *DiscoveryService) DiscoverComposeFiles(rootDir string) ([]ComposeFile, error) {
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
func (s *DiscoveryService) shouldSkipDir(dirName string) bool {
	// Skip hidden directories (including .git, .github, .vscode, etc.)
	return strings.HasPrefix(dirName, ".")
}

// isYAMLFile returns true if the file has a YAML extension
func (s *DiscoveryService) isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yml" || ext == ".yaml"
}

// isValidComposeFile validates if a YAML file is a valid Docker Compose file using compose-go v2
func (s *DiscoveryService) isValidComposeFile(path string) bool {
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
