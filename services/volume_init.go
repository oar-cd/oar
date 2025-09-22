package services

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/mount"
	"gopkg.in/yaml.v3"
)

// ComposeConfig represents the structure of a Docker Compose file
type ComposeConfig struct {
	Services map[string]Service `yaml:"services"`
}

// Service represents a service in the Docker Compose file
type Service struct {
	Image   string      `yaml:"image,omitempty"`
	User    string      `yaml:"user,omitempty"`
	Volumes []Volume    `yaml:"volumes,omitempty"`
	Build   interface{} `yaml:"build,omitempty"`
}

// Volume represents a volume mount - we only care about the source path (host path)
type Volume struct {
	Type   string `yaml:"type"`
	Source string `yaml:"source"`
}

// ComposeService represents a service with volume permission requirements
type ComposeService struct {
	Name    string
	User    *UserInfo
	Volumes []Volume // Volumes that need permission fixing
}

// UserInfo represents parsed user information
type UserInfo struct {
	UID string
	GID string
}

// parseUser parses user string in format "user:group"
func parseUser(userStr string) (*UserInfo, error) {
	if userStr == "" {
		return nil, nil
	}

	parts := strings.Split(userStr, ":")
	if len(parts) == 1 {
		return &UserInfo{UID: parts[0], GID: parts[0]}, nil
	}
	if len(parts) == 2 {
		return &UserInfo{UID: parts[0], GID: parts[1]}, nil
	}
	return nil, fmt.Errorf("invalid user format: %s", userStr)
}

// isNumeric checks if a string is a valid numeric ID using strconv.Atoi
func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// ServicesWithVolumes returns services that need volume permission fixing
func (config *ComposeConfig) ServicesWithVolumes(p *ComposeProject) ([]ComposeService, error) {
	var services []ComposeService

	for serviceName, service := range config.Services {
		slog.Debug("Processing service for volume permission check",
			"project_name", p.Name,
			"service", serviceName)

		// Skip services without volumes
		if len(service.Volumes) == 0 {
			slog.Debug("Service has no volumes, skipping",
				"project_name", p.Name,
				"service", serviceName)
			continue
		}

		// Get user info for this service
		userInfo, err := service.GetUser(serviceName, p)
		if err != nil {
			slog.Warn("Failed to get service user, assuming root",
				"project_name", p.Name,
				"service", serviceName,
				"error", err)
			return nil, fmt.Errorf("failed to get user for service %s: %w", serviceName, err)
		}

		slog.Debug("Service user info",
			"project_name", p.Name,
			"service", serviceName,
			"user", fmt.Sprintf("%s:%s", userInfo.UID, userInfo.GID))

		// Skip if user is root (no permission fix needed)
		if userInfo.UID == "0" || userInfo.UID == "root" {
			slog.Debug("Service runs as root, skipping permission fix",
				"project_name", p.Name,
				"service", serviceName)
			continue
		}

		services = append(services, ComposeService{
			Name:    serviceName,
			User:    userInfo,
			Volumes: service.Volumes,
		})
	}

	serviceNames := make([]string, len(services))
	for i, service := range services {
		serviceNames[i] = service.Name
	}

	slog.Debug("Found services needing volume permission fixing",
		"project_name", p.Name,
		"service_count", len(services),
		"services", serviceNames)

	return services, nil
}

// GetUser determines the user (UID:GID) that a service should run as
func (s *Service) GetUser(serviceName string, p *ComposeProject) (*UserInfo, error) {
	// Determine image name first (needed for both paths)
	imageName := s.Image
	if imageName == "" && s.Build != nil {
		// For build services without explicit image, use default naming: <project-name>-<service-name>
		imageName = fmt.Sprintf("%s-%s", p.Name, serviceName)
	}

	var userInfo *UserInfo
	var err error

	// Get user info based on whether service has explicit user field
	if s.User != "" {
		userInfo, err = parseUser(s.User)
		if err != nil {
			return nil, err
		}
	} else {
		if imageName == "" {
			return nil, fmt.Errorf("service %s has no image or build context", serviceName)
		}

		// Inspect the image to get default user
		userInfo, err = p.inspectImageUser(imageName)
		if err != nil {
			return nil, err
		}
	}

	// If we have non-numeric user IDs, resolve them to numeric using the service image
	if !isNumeric(userInfo.UID) || !isNumeric(userInfo.GID) {
		return p.resolveUserToNumeric(userInfo, imageName)
	}

	return userInfo, nil
}

// inspectImageUser inspects a Docker image to determine its default user
func (p *ComposeProject) inspectImageUser(imageName string) (*UserInfo, error) {
	dockerClient, err := NewDockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer func() {
		if closeErr := dockerClient.Close(); closeErr != nil {
			slog.Debug("Failed to close Docker client", "error", closeErr)
		}
	}()

	userStr, err := dockerClient.GetImageUser(imageName)
	if err != nil {
		slog.Warn("Failed to inspect image, assuming root user",
			"project_name", p.Name,
			"image", imageName,
			"error", err)
		return &UserInfo{UID: "0", GID: "0"}, nil
	}

	if userStr == "" {
		// No user specified in image, defaults to root
		return &UserInfo{UID: "0", GID: "0"}, nil
	}

	return parseUser(userStr)
}

// resolveUserToNumeric resolves usernames to numeric UIDs/GIDs using the actual image
func (p *ComposeProject) resolveUserToNumeric(userInfo *UserInfo, imageName string) (*UserInfo, error) {
	dockerClient, err := NewDockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client for user resolution: %w", err)
	}
	defer func() {
		if closeErr := dockerClient.Close(); closeErr != nil {
			slog.Debug("Failed to close Docker client", "error", closeErr)
		}
	}()

	// Build command to resolve user/group to numeric IDs using the service's own image
	// Use a simple command that outputs "UID:GID" format
	resolveCmd := fmt.Sprintf("echo $(id -u %s):$(id -g %s)", userInfo.UID, userInfo.GID)

	containerName := fmt.Sprintf("%s-user-resolve", p.Name)

	slog.Debug("Resolving username to numeric IDs",
		"project_name", p.Name,
		"image", imageName,
		"user_string", fmt.Sprintf("%s:%s", userInfo.UID, userInfo.GID),
		"command", resolveCmd)

	// Run container and get output
	output, err := dockerClient.RunContainerWithOutput(containerName, imageName, resolveCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user %s:%s to numeric IDs: %w", userInfo.UID, userInfo.GID, err)
	}

	// Parse the output (should be in format "UID:GID")
	parts := strings.Split(strings.TrimSpace(output), ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("unexpected output format from user resolution: %s", output)
	}

	// Verify the parts are numeric
	if !isNumeric(parts[0]) || !isNumeric(parts[1]) {
		return nil, fmt.Errorf("resolved user IDs are not numeric: %s", output)
	}

	resolvedUser := &UserInfo{UID: parts[0], GID: parts[1]}

	slog.Debug("Successfully resolved username to numeric IDs",
		"project_name", p.Name,
		"original", fmt.Sprintf("%s:%s", userInfo.UID, userInfo.GID),
		"resolved", fmt.Sprintf("%s:%s", resolvedUser.UID, resolvedUser.GID))

	return resolvedUser, nil
}

// HasBuildServices checks if any services in the config have build context
func (config *ComposeConfig) HasBuildServices() bool {
	for _, service := range config.Services {
		if service.Build != nil {
			return true
		}
	}
	return false
}

// InitializeVolumeMounts fixes volume mount point ownership using helper containers
func (p *ComposeProject) InitializeVolumeMounts() error {
	slog.Debug("Starting volume mounts initialization",
		"project_name", p.Name)

	// Get compose config first to check what we need to do
	configYAML, stderr, err := p.GetConfig()
	if err != nil {
		slog.Error("Failed to get compose config",
			"project_name", p.Name,
			"error", err)
		return fmt.Errorf("failed to get compose config: %w", err)
	}
	// Log stderr warnings but don't include them in the YAML processing
	if stderr != "" {
		slog.Debug("Docker compose config warnings", "project_name", p.Name, "warnings", stderr)
	}

	var config ComposeConfig
	err = yaml.Unmarshal([]byte(configYAML), &config)
	if err != nil {
		slog.Error("Failed to parse compose config YAML",
			"project_name", p.Name,
			"error", err)
		return fmt.Errorf("failed to parse compose config: %w", err)
	}

	// Note: Images and containers are now created by the calling code
	// before volume initialization using docker compose up --no-start

	// Get all services that need volume permission fixing
	services, err := config.ServicesWithVolumes(p)
	if err != nil {
		return fmt.Errorf("failed to get services with volumes: %w", err)
	}

	if len(services) == 0 {
		slog.Debug("No services need volume permission fixing, skipping initialization",
			"project_name", p.Name)
		return nil
	}

	// Fix permissions for each service
	for _, service := range services {
		err := p.fixServiceVolumePermissions(service)
		if err != nil {
			return fmt.Errorf("failed to fix permissions for service %s: %w", service.Name, err)
		}
	}

	slog.Debug("Completed volume permission initialization",
		"project_name", p.Name,
		"services_processed", len(services))

	return nil
}

// fixServiceVolumePermissions creates a helper container to fix permissions for a service's volumes
func (p *ComposeProject) fixServiceVolumePermissions(service ComposeService) error {
	dockerClient, err := NewDockerClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer func() {
		if closeErr := dockerClient.Close(); closeErr != nil {
			slog.Debug("Failed to close Docker client", "error", closeErr)
		}
	}()

	// Pre-create directory structure for bind mounts to ensure directories exist
	for _, volume := range service.Volumes {
		if volume.Type == "bind" {
			slog.Debug("Creating directory structure for bind mount",
				"project_name", p.Name,
				"service", service.Name,
				"source_path", volume.Source)

			// Check if the source path already exists and what type it is
			if stat, err := os.Stat(volume.Source); err == nil {
				// Path exists - check if it's a file
				if !stat.IsDir() {
					// It's a file, so create only the parent directory
					parentDir := filepath.Dir(volume.Source)
					slog.Debug("Source path is a file, creating parent directory",
						"project_name", p.Name,
						"service", service.Name,
						"source_path", volume.Source,
						"parent_dir", parentDir)

					err = os.MkdirAll(parentDir, 0o755)
					if err != nil {
						return fmt.Errorf(
							"failed to create parent directory %s for bind mount %s: %w",
							parentDir,
							volume.Source,
							err,
						)
					}
				}
				// If it's already a directory, nothing to do
			} else if os.IsNotExist(err) {
				// Path doesn't exist, try to create it as a directory
				err = os.MkdirAll(volume.Source, 0o755)
				if err != nil {
					return fmt.Errorf(
						"failed to create directory structure for bind mount %s: %w",
						volume.Source,
						err,
					)
				}
			} else {
				// Some other error occurred while checking
				return fmt.Errorf(
					"failed to check bind mount path %s: %w",
					volume.Source,
					err,
				)
			}
		}
	}

	// Prepare mounts for helper container
	var containerMounts []mount.Mount
	var chownCommands []string

	for i, volume := range service.Volumes {
		// Mount the volume to an arbitrary target in the helper container
		helperTarget := fmt.Sprintf("/volume-%d", i+1)

		// Use the correct mount type based on the volume type
		var mountType mount.Type
		switch volume.Type {
		case "bind":
			mountType = mount.TypeBind
		case "volume":
			mountType = mount.TypeVolume
		default:
			// Default to bind for backward compatibility
			mountType = mount.TypeBind
		}

		// For named volumes, we need to use the Docker Compose prefixed name
		var sourceForMount string
		if mountType == mount.TypeVolume {
			// Docker Compose prefixes named volumes with project name
			sourceForMount = fmt.Sprintf("%s_%s", p.Name, volume.Source)
			slog.Debug("Using prefixed volume name for helper container",
				"project_name", p.Name,
				"service", service.Name,
				"original_volume", volume.Source,
				"prefixed_volume", sourceForMount)
		} else {
			// For bind mounts, use the source path as-is
			sourceForMount = volume.Source
		}

		containerMounts = append(containerMounts, mount.Mount{
			Type:   mountType,
			Source: sourceForMount,
			Target: helperTarget,
		})

		// Add chown command for the helper target
		// Note: service.User should already contain numeric IDs at this point
		chownCmd := fmt.Sprintf("chown -R %s:%s %s", service.User.UID, service.User.GID, helperTarget)
		chownCommands = append(chownCommands, chownCmd)
	}

	// Create helper container
	containerName := fmt.Sprintf("%s-volume-init-%s", p.Name, service.Name)

	// Build command string
	cmdStr := strings.Join(chownCommands, " && ")

	slog.Debug("Running helper container to fix volume permissions",
		"project_name", p.Name,
		"service", service.Name,
		"container", containerName,
		"user", fmt.Sprintf("%s:%s", service.User.UID, service.User.GID),
		"chown_commands", cmdStr)

	// Run helper container
	err = dockerClient.RunVolumeChowningContainer(containerName, cmdStr, containerMounts)
	if err != nil {
		return fmt.Errorf("failed to run helper container: %w", err)
	}

	slog.Debug("Helper container completed successfully",
		"project_name", p.Name,
		"service", service.Name,
		"volume_count", len(service.Volumes))

	return nil
}
