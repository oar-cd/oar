package services

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
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

// GetDefaultDataDir returns the default Oar data directory
func GetDefaultDataDir() string {
	return "/opt/oar/data"
}

// YamlConfig represents the YAML configuration file structure
type YamlConfig struct {
	DataDir       string        `yaml:"data_dir"`
	DatabasePath  string        `yaml:"database_path,omitempty"`
	LogLevel      string        `yaml:"log_level,omitempty"`
	HTTP          HTTPConfig    `yaml:"http,omitempty"`
	Git           GitConfig     `yaml:"git,omitempty"`
	Watcher       WatcherConfig `yaml:"watcher,omitempty"`
	EncryptionKey string        `yaml:"encryption_key"`
}

type HTTPConfig struct {
	Host string `yaml:"host,omitempty"`
	Port int    `yaml:"port,omitempty"`
}

type GitConfig struct {
	Timeout string `yaml:"timeout,omitempty"`
}

type WatcherConfig struct {
	PollInterval string `yaml:"poll_interval,omitempty"`
}

// Config holds configuration for all services
type Config struct {
	// Core paths
	DataDir      string // Data directory (database, projects, tmp)
	DatabasePath string
	TmpDir       string
	WorkspaceDir string

	// Logging
	LogLevel string

	// HTTP server
	HTTPHost string
	HTTPPort int

	// Git
	GitTimeout time.Duration

	// Watcher
	PollInterval time.Duration

	// Encryption
	EncryptionKey string

	// Environment provider for testing
	env EnvProvider
}

// NewConfigForCLI creates a new configuration for CLI usage
func NewConfigForCLI() (*Config, error) {
	return newConfigForCLIWithEnv(&DefaultEnvProvider{})
}

// NewConfigForCLIWithEnv creates a new configuration with custom environment provider (for testing)
func NewConfigForCLIWithEnv(env EnvProvider) (*Config, error) {
	return newConfigForCLIWithEnv(env)
}

func newConfigForCLIWithEnv(env EnvProvider) (*Config, error) {
	slog.Debug("Loading configuration for CLI")
	c := &Config{env: env}

	// Set defaults first
	c.setDefaults()
	slog.Debug("Set default configuration values")

	// Override CLI-specific defaults
	c.LogLevel = "silent" // CLI should be quiet by default
	slog.Debug("Set CLI-specific defaults", "log_level", c.LogLevel)

	// Override with environment variables
	c.loadFromEnv()
	slog.Debug("Loaded configuration from environment variables")

	// Derive dependent paths
	c.derivePaths()
	slog.Debug("Derived configuration paths", "data_dir", c.DataDir, "database_path", c.DatabasePath)

	// Validate
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	slog.Debug("Configuration validation passed")

	// Log final configuration
	slog.Debug("Final configuration loaded for CLI",
		"data_dir", c.DataDir,
		"database_path", c.DatabasePath,
		"tmp_dir", c.TmpDir,
		"workspace_dir", c.WorkspaceDir,
		"log_level", c.LogLevel,
		"http_host", c.HTTPHost,
		"http_port", c.HTTPPort,
		"git_timeout", c.GitTimeout,
		"poll_interval", c.PollInterval,
		"has_encryption_key", c.EncryptionKey != "")

	return c, nil
}

// NewConfigForWebApp creates a new configuration for web application usage
// This version only uses environment variables and defaults, no CLI overrides
func NewConfigForWebApp() (*Config, error) {
	return NewConfigForWebAppWithEnv(&DefaultEnvProvider{})
}

// NewConfigForWebAppWithEnv creates a new configuration with custom environment provider (for testing)
func NewConfigForWebAppWithEnv(env EnvProvider) (*Config, error) {
	slog.Debug("Loading configuration for web application")
	c := &Config{env: env}

	// Set defaults first
	c.setDefaults()
	slog.Debug("Set default configuration values")

	// Override with environment variables
	c.loadFromEnv()
	slog.Debug("Loaded configuration from environment variables")

	// Derive dependent paths
	c.derivePaths()
	slog.Debug("Derived configuration paths", "data_dir", c.DataDir, "database_path", c.DatabasePath)

	// Validate
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	slog.Debug("Configuration validation passed")

	// Log final configuration
	slog.Debug("Final configuration loaded for web application",
		"data_dir", c.DataDir,
		"database_path", c.DatabasePath,
		"tmp_dir", c.TmpDir,
		"workspace_dir", c.WorkspaceDir,
		"log_level", c.LogLevel,
		"http_host", c.HTTPHost,
		"http_port", c.HTTPPort,
		"git_timeout", c.GitTimeout,
		"poll_interval", c.PollInterval,
		"has_encryption_key", c.EncryptionKey != "")

	return c, nil
}

// NewConfigFromFile creates a new configuration from a YAML config file
func NewConfigFromFile(configPath string) (*Config, error) {
	return NewConfigFromFileWithEnv(configPath, &DefaultEnvProvider{})
}

// NewConfigFromFileWithEnv creates a new configuration from a YAML config file with custom environment provider
func NewConfigFromFileWithEnv(configPath string, env EnvProvider) (*Config, error) {
	slog.Debug("Loading configuration from file", "config_path", configPath)
	c := &Config{env: env}

	// Set defaults first
	c.setDefaults()
	slog.Debug("Set default configuration values")

	// Load from YAML config file (if path is specified)
	if configPath != "" {
		if err := c.loadFromYamlFile(configPath); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	// Override with environment variables (env vars have higher priority than config file)
	c.loadFromEnv()
	slog.Debug("Loaded configuration from environment variables")

	// Derive dependent paths
	c.derivePaths()
	slog.Debug("Derived configuration paths", "data_dir", c.DataDir, "database_path", c.DatabasePath)

	// Validate
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	slog.Debug("Configuration validation passed")

	// Log final configuration
	slog.Debug("Final configuration loaded from file",
		"config_path", configPath,
		"data_dir", c.DataDir,
		"database_path", c.DatabasePath,
		"tmp_dir", c.TmpDir,
		"workspace_dir", c.WorkspaceDir,
		"log_level", c.LogLevel,
		"http_host", c.HTTPHost,
		"http_port", c.HTTPPort,
		"git_timeout", c.GitTimeout,
		"poll_interval", c.PollInterval,
		"has_encryption_key", c.EncryptionKey != "")

	return c, nil
}

// setDefaults sets sensible default values
func (c *Config) setDefaults() {
	c.DataDir = "/opt/oar/data"
	c.LogLevel = "info"
	c.HTTPHost = "127.0.0.1"
	c.HTTPPort = 3333
	c.GitTimeout = 5 * time.Minute
	c.PollInterval = 5 * time.Minute
	// Don't set default encryption key - it must be provided explicitly
}

// loadFromEnv loads configuration from environment variables
func (c *Config) loadFromEnv() {
	var envVarsFound []string

	if v := c.env.Getenv("OAR_DATA_DIR"); v != "" {
		c.DataDir = v
		envVarsFound = append(envVarsFound, "OAR_DATA_DIR")
	}
	if v := c.env.Getenv("OAR_DATABASE_PATH"); v != "" {
		c.DatabasePath = v
		envVarsFound = append(envVarsFound, "OAR_DATABASE_PATH")
	}
	if v := c.env.Getenv("OAR_LOG_LEVEL"); v != "" {
		c.LogLevel = v
		envVarsFound = append(envVarsFound, "OAR_LOG_LEVEL")
	}
	if v := c.env.Getenv("OAR_HTTP_HOST"); v != "" {
		c.HTTPHost = v
		envVarsFound = append(envVarsFound, "OAR_HTTP_HOST")
	}
	if v := c.env.Getenv("OAR_HTTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.HTTPPort = port
			envVarsFound = append(envVarsFound, "OAR_HTTP_PORT")
		}
	}
	if v := c.env.Getenv("OAR_GIT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.GitTimeout = d
			envVarsFound = append(envVarsFound, "OAR_GIT_TIMEOUT")
		}
	}
	if v := c.env.Getenv("OAR_POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.PollInterval = d
			envVarsFound = append(envVarsFound, "OAR_POLL_INTERVAL")
		}
	}
	if v := c.env.Getenv("OAR_ENCRYPTION_KEY"); v != "" {
		c.EncryptionKey = v
		envVarsFound = append(envVarsFound, "OAR_ENCRYPTION_KEY")
	}

	if len(envVarsFound) > 0 {
		slog.Debug("Found environment variables", "variables", envVarsFound)
	}
}

// loadFromYamlFile loads configuration from a YAML file
func (c *Config) loadFromYamlFile(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	slog.Info("Loading configuration from YAML file", "config_path", configPath)

	var yamlConfig YamlConfig
	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Apply YAML config to internal config
	if yamlConfig.DataDir != "" {
		c.DataDir = yamlConfig.DataDir
	}
	if yamlConfig.DatabasePath != "" {
		c.DatabasePath = yamlConfig.DatabasePath
	}
	if yamlConfig.LogLevel != "" {
		c.LogLevel = yamlConfig.LogLevel
	}
	if yamlConfig.HTTP.Host != "" {
		c.HTTPHost = yamlConfig.HTTP.Host
	}
	if yamlConfig.HTTP.Port != 0 {
		c.HTTPPort = yamlConfig.HTTP.Port
	}
	if yamlConfig.Git.Timeout != "" {
		if d, err := time.ParseDuration(yamlConfig.Git.Timeout); err == nil {
			c.GitTimeout = d
		}
	}
	if yamlConfig.Watcher.PollInterval != "" {
		if d, err := time.ParseDuration(yamlConfig.Watcher.PollInterval); err == nil {
			c.PollInterval = d
		}
	}
	if yamlConfig.EncryptionKey != "" {
		c.EncryptionKey = yamlConfig.EncryptionKey
	}

	return nil
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
		"debug": true, "info": true, "warning": true, "error": true, "silent": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warning, error, or silent)", c.LogLevel)
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

	// Require encryption key to be provided via environment variable or config file
	if c.EncryptionKey == "" {
		return fmt.Errorf("encryption key is required - set in config file or OAR_ENCRYPTION_KEY environment variable")
	}

	return nil
}

// GetLogLevel returns the configured log level
func (c *Config) GetLogLevel() string {
	return c.LogLevel
}
