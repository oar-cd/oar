package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	DataDir     = ".oar"
	ProjectsDir = "projects"
	GitDir      = "git"
	TmpDir      = "tmp"
)

// Config holds configuration for all services
type Config struct {
	// Core paths
	DataDir      string
	DatabasePath string
	TmpDir       string
	WorkspaceDir string

	// Logging
	LogLevel     string
	ColorEnabled bool

	// Docker
	DockerHost    string
	DockerCommand string

	// HTTP server
	HTTPHost string
	HTTPPort int

	// Git
	GitTimeout time.Duration
}

// NewConfigForCLI creates a new configuration for CLI usage with optional data directory override
func NewConfigForCLI(cliDataDir string) (*Config, error) {
	c := &Config{}

	// Set defaults first
	c.setDefaults()

	// Override with environment variables
	c.loadFromEnv()

	// Override with CLI flags (if provided)
	if cliDataDir != "" {
		c.DataDir = cliDataDir
	}

	// Derive dependent paths
	c.derivePaths()

	// Validate
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return c, nil
}

// NewConfigForWebApp creates a new configuration for web application usage
// This version only uses environment variables and defaults, no CLI overrides
func NewConfigForWebApp() (*Config, error) {
	c := &Config{}

	// Set defaults first
	c.setDefaults()

	// Override with environment variables
	c.loadFromEnv()

	// Derive dependent paths
	c.derivePaths()

	// Validate
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return c, nil
}

// setDefaults sets sensible default values
func (c *Config) setDefaults() {
	homeDir, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(homeDir, DataDir)

	c.DataDir = defaultDataDir
	c.LogLevel = "info"
	c.ColorEnabled = true
	c.DockerHost = "unix:///var/run/docker.sock"
	c.DockerCommand = "docker"
	c.HTTPHost = "127.0.0.1"
	c.HTTPPort = 8080
	c.GitTimeout = 5 * time.Minute
}

// loadFromEnv loads configuration from environment variables
func (c *Config) loadFromEnv() {
	if v := os.Getenv("OAR_DATA_DIR"); v != "" {
		c.DataDir = v
	}
	if v := os.Getenv("OAR_DATABASE_PATH"); v != "" {
		c.DatabasePath = v
	}
	if v := os.Getenv("OAR_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("OAR_COLOR_ENABLED"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			c.ColorEnabled = enabled
		}
	}
	if v := os.Getenv("OAR_DOCKER_HOST"); v != "" {
		c.DockerHost = v
	}
	if v := os.Getenv("OAR_DOCKER_COMMAND"); v != "" {
		c.DockerCommand = v
	}
	if v := os.Getenv("OAR_HTTP_HOST"); v != "" {
		c.HTTPHost = v
	}
	if v := os.Getenv("OAR_HTTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.HTTPPort = port
		}
	}
	if v := os.Getenv("OAR_GIT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.GitTimeout = d
		}
	}
}

// derivePaths calculates dependent paths from the base DataDir
func (c *Config) derivePaths() {
	c.TmpDir = filepath.Join(c.DataDir, TmpDir)
	c.WorkspaceDir = filepath.Join(c.DataDir, ProjectsDir)

	// Set default database path if not explicitly configured
	if c.DatabasePath == "" {
		c.DatabasePath = filepath.Join(c.DataDir, "oar.db")
	}
}

// validate ensures configuration values are valid
func (c *Config) validate() error {
	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warning": true, "error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warning, or error)", c.LogLevel)
	}

	// Validate HTTP port
	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d (must be 1-65535)", c.HTTPPort)
	}

	// Validate timeout
	if c.GitTimeout <= 0 {
		return fmt.Errorf("git timeout must be positive, got: %v", c.GitTimeout)
	}

	// Validate Docker command is not empty
	if c.DockerCommand == "" {
		return fmt.Errorf("docker command cannot be empty")
	}

	return nil
}

// GetLogLevel returns the configured log level
func (c *Config) GetLogLevel() string {
	return c.LogLevel
}
