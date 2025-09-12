package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/compose-spec/compose-go/v2/dotenv"
)

const (
	DataDir     = ".oar"
	ProjectsDir = "projects"
	GitDir      = "git"
	TmpDir      = "tmp"
)

// EnvProvider abstracts environment variable access for testing
type EnvProvider interface {
	Getenv(key string) string
	UserHomeDir() (string, error)
}

// DefaultEnvProvider implements EnvProvider using real OS functions
type DefaultEnvProvider struct{}

func (p *DefaultEnvProvider) Getenv(key string) string {
	return os.Getenv(key)
}

func (p *DefaultEnvProvider) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

// GetDefaultDataDir returns the default Oar data directory following XDG Base Directory specification
func GetDefaultDataDir() string {
	return getDefaultDataDirWithEnv(&DefaultEnvProvider{})
}

// getDefaultDataDirWithEnv allows dependency injection for testing
func getDefaultDataDirWithEnv(env EnvProvider) string {
	// Use XDG_DATA_HOME if set, otherwise fallback to ~/.local/share
	xdgDataHome := env.Getenv("XDG_DATA_HOME")
	if xdgDataHome != "" {
		return filepath.Join(xdgDataHome, "oar")
	}

	homeDir, _ := env.UserHomeDir()
	return filepath.Join(homeDir, ".local", "share", "oar")
}

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

	// Watcher
	PollInterval time.Duration

	// Encryption
	EncryptionKey string

	// Runtime environment
	Containerized bool

	// Environment provider for testing
	env EnvProvider
}

// NewConfigForCLI creates a new configuration for CLI usage with optional data directory override
func NewConfigForCLI(cliDataDir string) (*Config, error) {
	return newConfigWithEnv(&DefaultEnvProvider{}, cliDataDir)
}

// NewConfigForCLIWithEnv creates a new configuration with custom environment provider (for testing)
func NewConfigForCLIWithEnv(env EnvProvider, cliDataDir string) (*Config, error) {
	return newConfigWithEnv(env, cliDataDir)
}

func newConfigWithEnv(env EnvProvider, cliDataDir string) (*Config, error) {
	c := &Config{env: env}

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

	// Try to read encryption key from .env file as fallback (after data dir is finalized)
	if c.EncryptionKey == "" {
		if key := c.readEncryptionKeyFromEnvFile(); key != "" {
			c.EncryptionKey = key
		}
	}

	// Validate
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return c, nil
}

// NewConfigForWebApp creates a new configuration for web application usage
// This version only uses environment variables and defaults, no CLI overrides
func NewConfigForWebApp() (*Config, error) {
	return NewConfigForWebAppWithEnv(&DefaultEnvProvider{})
}

// NewConfigForWebAppWithEnv creates a new configuration with custom environment provider (for testing)
func NewConfigForWebAppWithEnv(env EnvProvider) (*Config, error) {
	c := &Config{env: env}

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
	c.DataDir = getDefaultDataDirWithEnv(c.env)
	c.LogLevel = "info"
	c.ColorEnabled = true
	c.DockerHost = "unix:///var/run/docker.sock"
	c.DockerCommand = "docker"
	c.HTTPHost = "127.0.0.1"
	c.HTTPPort = 8080
	c.GitTimeout = 5 * time.Minute
	c.PollInterval = 5 * time.Minute
	// Don't set default encryption key - it must be provided explicitly
}

// loadFromEnv loads configuration from environment variables
func (c *Config) loadFromEnv() {
	if v := c.env.Getenv("OAR_DATA_DIR"); v != "" {
		c.DataDir = v
	}
	if v := c.env.Getenv("OAR_DATABASE_PATH"); v != "" {
		c.DatabasePath = v
	}
	if v := c.env.Getenv("OAR_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := c.env.Getenv("OAR_COLOR_ENABLED"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			c.ColorEnabled = enabled
		}
	}
	if v := c.env.Getenv("OAR_DOCKER_HOST"); v != "" {
		c.DockerHost = v
	}
	if v := c.env.Getenv("OAR_DOCKER_COMMAND"); v != "" {
		c.DockerCommand = v
	}
	if v := c.env.Getenv("OAR_HTTP_HOST"); v != "" {
		c.HTTPHost = v
	}
	if v := c.env.Getenv("OAR_HTTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.HTTPPort = port
		}
	}
	if v := c.env.Getenv("OAR_GIT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.GitTimeout = d
		}
	}
	if v := c.env.Getenv("OAR_POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.PollInterval = d
		}
	}
	if v := c.env.Getenv("OAR_ENCRYPTION_KEY"); v != "" {
		c.EncryptionKey = v
	}
	if v := c.env.Getenv("OAR_CONTAINERIZED"); v != "" {
		if containerized, err := strconv.ParseBool(v); err == nil {
			c.Containerized = containerized
		}
	}
}

// readEncryptionKeyFromEnvFile attempts to read OAR_ENCRYPTION_KEY from .env file in data directory
func (c *Config) readEncryptionKeyFromEnvFile() string {
	envFile := filepath.Join(c.DataDir, ".env")

	envVars, err := dotenv.Read(envFile)
	if err != nil {
		// .env file doesn't exist or can't be read, that's okay
		return ""
	}

	return envVars["OAR_ENCRYPTION_KEY"]
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

	// Validate poll interval
	if c.PollInterval <= 0 {
		return fmt.Errorf("poll interval must be positive, got: %v", c.PollInterval)
	}

	// Validate Docker command is not empty
	if c.DockerCommand == "" {
		return fmt.Errorf("docker command cannot be empty")
	}

	// Require encryption key to be provided via environment variable or .env file
	if c.EncryptionKey == "" {
		return fmt.Errorf(
			"encryption key is required - set OAR_ENCRYPTION_KEY environment variable or ensure .env file exists in data directory (%s)",
			c.DataDir,
		)
	}

	return nil
}

// GetLogLevel returns the configured log level
func (c *Config) GetLogLevel() string {
	return c.LogLevel
}
