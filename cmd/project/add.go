package project

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/ch00k/oar/cmd/output"
	"github.com/ch00k/oar/cmd/utils"
	"github.com/ch00k/oar/internal/app"
	"github.com/ch00k/oar/services"
	"github.com/spf13/cobra"
)

func NewCmdProjectAdd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a Git repository as a managed project",
		Long: `Add a new Docker Compose project from a Git repository.
Oar will clone the repository and manage it with Docker Compose.

Authentication examples:
  # HTTP authentication (GitHub token, etc.)
  oar project add --git-url https://github.com/user/repo.git \
                  --git-auth http --git-username token --git-password ghp_xxxxx

  # SSH authentication
  oar project add --git-url git@github.com:user/repo.git \
                  --git-auth ssh --git-username git --git-ssh-key-file ~/.ssh/id_rsa

Environment variables:
  # Individual variables
  oar project add --git-url https://github.com/user/repo.git \
                  --env DATABASE_URL=postgres://... --env PORT=3000

  # From file
  oar project add --git-url https://github.com/user/repo.git \
                  --env-file .env.production`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runProjectAdd(cmd); err != nil {
				utils.HandleCommandError("creating project", err)
				os.Exit(1)
			}
		},
	}

	// Basic flags
	cmd.Flags().StringP("git-url", "u", "", "Git repository URL")
	cmd.Flags().StringP("name", "n", "", "Custom project name (auto-detected if not specified)")
	cmd.Flags().
		StringArrayP("compose-file", "f", nil, `Docker Compose file path, relative to repository root. Can be used multiple times: --compose-file compose.yml --compose-file docker-compose.override.yml`)

	// Git authentication flags
	cmd.Flags().String("git-auth", "", "Git authentication method: http, ssh")
	cmd.Flags().String("git-username", "", "Git username (for HTTP) or SSH user (for SSH)")
	cmd.Flags().String("git-password", "", "Git password or token (for HTTP authentication)")
	cmd.Flags().String("git-ssh-key-file", "", "Path to SSH private key file (for SSH authentication)")

	// Environment variable flags
	cmd.Flags().
		StringArray("env", nil, `Environment variable in KEY=value format. Can be used multiple times: --env KEY1=val1 --env KEY2=val2`)
	cmd.Flags().String("env-file", "", "Path to environment file (.env format)")

	if err := cmd.MarkFlagRequired("git-url"); err != nil {
		slog.Error("Failed to mark git-url flag as required", "error", err)
		panic(fmt.Sprintf("CLI setup error: %v", err)) // This is a setup error, should panic
	}
	if err := cmd.MarkFlagRequired("compose-file"); err != nil {
		slog.Error("Failed to mark compose-file flag as required", "error", err)
		panic(fmt.Sprintf("CLI setup error: %v", err)) // This is a setup error, should panic
	}
	return cmd
}

// runProjectAdd handles the main logic for project creation
func runProjectAdd(cmd *cobra.Command) error {
	// Get basic flag values
	gitURL, _ := cmd.Flags().GetString("git-url")
	name, _ := cmd.Flags().GetString("name")
	composeFiles, _ := cmd.Flags().GetStringArray("compose-file")

	// Build Git authentication config
	gitAuth, err := buildGitAuthFromFlags(cmd)
	if err != nil {
		return fmt.Errorf("invalid authentication configuration: %w", err)
	}

	// Build environment variables
	variables, err := buildVariablesFromFlags(cmd)
	if err != nil {
		return fmt.Errorf("invalid environment variables: %w", err)
	}

	// Create project struct from CLI input
	project := services.NewProject(name, gitURL, composeFiles, variables)
	project.GitAuth = gitAuth

	// Call service
	createdProject, err := app.GetProjectService().Create(&project)
	if err != nil {
		return fmt.Errorf("failed to create project from %s: %w", gitURL, err)
	}

	// Print success output
	out, err := output.PrintProjectDetails(createdProject, true)
	if err != nil {
		return fmt.Errorf("failed to format project details: %w", err)
	}

	if err := output.FprintPlain(cmd, "%s", out); err != nil {
		return fmt.Errorf("failed to print output: %w", err)
	}

	return nil
}

// buildGitAuthFromFlags constructs GitAuthConfig from command flags
func buildGitAuthFromFlags(cmd *cobra.Command) (*services.GitAuthConfig, error) {
	authMethod, _ := cmd.Flags().GetString("git-auth")

	// No authentication specified
	if authMethod == "" {
		return nil, nil
	}

	switch authMethod {
	case "http":
		username, _ := cmd.Flags().GetString("git-username")
		password, _ := cmd.Flags().GetString("git-password")

		if username == "" && password == "" {
			return nil, fmt.Errorf("HTTP authentication requires --git-username and --git-password")
		}

		return &services.GitAuthConfig{
			HTTPAuth: &services.GitHTTPAuthConfig{
				Username: username,
				Password: password,
			},
		}, nil

	case "ssh":
		sshUser, _ := cmd.Flags().GetString("git-username")
		keyFile, _ := cmd.Flags().GetString("git-ssh-key-file")

		if keyFile == "" {
			return nil, fmt.Errorf("SSH authentication requires --git-ssh-key-file")
		}

		// Read private key from file
		privateKeyBytes, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read SSH key file %s: %w", keyFile, err)
		}

		privateKey := string(privateKeyBytes)

		// Default SSH user to "git" if not specified
		if sshUser == "" {
			sshUser = "git"
		}

		return &services.GitAuthConfig{
			SSHAuth: &services.GitSSHAuthConfig{
				PrivateKey: privateKey,
				User:       sshUser,
			},
		}, nil

	default:
		return nil, fmt.Errorf("invalid authentication method %q, must be 'http' or 'ssh'", authMethod)
	}
}

// buildVariablesFromFlags constructs environment variables from command flags
func buildVariablesFromFlags(cmd *cobra.Command) ([]string, error) {
	var variables []string

	// Get individual env vars
	envVars, _ := cmd.Flags().GetStringArray("env")
	for _, envVar := range envVars {
		if !strings.Contains(envVar, "=") {
			return nil, fmt.Errorf("invalid environment variable format %q, expected KEY=value", envVar)
		}
		variables = append(variables, envVar)
	}

	// Get env file
	envFile, _ := cmd.Flags().GetString("env-file")
	if envFile != "" {
		fileVars, err := readEnvFile(envFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read environment file %s: %w", envFile, err)
		}
		variables = append(variables, fileVars...)
	}

	return variables, nil
}

// readEnvFile reads environment variables from a .env file
func readEnvFile(filename string) ([]string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var variables []string
	lines := strings.Split(string(content), "\n")

	for i, line := range lines {
		// Skip empty lines and comments
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Validate format
		if !strings.Contains(line, "=") {
			return nil, fmt.Errorf("line %d: invalid format %q, expected KEY=value", i+1, line)
		}

		variables = append(variables, line)
	}

	return variables, nil
}
