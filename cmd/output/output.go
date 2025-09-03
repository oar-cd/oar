// Package output provides functions to print messages with optional color formatting
package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/ch00k/oar/services"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

const (
	Plain   = color.FgWhite
	Success = color.FgGreen
	Warning = color.FgYellow
	Error   = color.FgRed
)

var maybeColorize func(kind color.Attribute, tmpl string, a ...any) string

// InitColors sets up color functions based on environment
func InitColors(isColorDisabled bool) {
	// Check if colors should be enabled
	if color.NoColor || isColorDisabled {
		// Fallback to plain formatting if colors are not supported
		maybeColorize = func(kind color.Attribute, tmpl string, a ...any) string {
			return fmt.Sprintf(tmpl, a...)
		}
	} else {
		// Enable colors
		maybeColorize = func(kind color.Attribute, tmpl string, a ...any) string {
			return color.New(kind).SprintfFunc()(tmpl, a...)
		}
	}
}

// PrintMessage formats a message with color (if enabled) and prints it
func PrintMessage(kind color.Attribute, tmpl string, a ...any) string {
	if maybeColorize == nil || kind == Plain {
		return fmt.Sprintf(tmpl+"\n", a...)
	} else {
		// TODO: Print warnings and errors to stderr?
		return fmt.Sprintln(maybeColorize(kind, tmpl, a...))
	}
}

func PrintTable(header []string, data [][]string) (string, error) {
	buf := strings.Builder{}

	table := tablewriter.NewTable(
		&buf,
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Borders: tw.BorderNone,
			Settings: tw.Settings{
				Lines: tw.Lines{
					ShowHeaderLine: tw.Off,
				},
				Separators: tw.Separators{
					BetweenColumns: tw.Off,
				},
			},
		})),
		tablewriter.WithConfig(tablewriter.Config{
			Row: tw.CellConfig{
				Alignment: tw.CellAlignment{PerColumn: []tw.Align{tw.AlignRight, tw.AlignLeft}},
			},
		}))

	if len(header) > 0 {
		table.Header(header)
	}

	if err := table.Bulk(data); err != nil {
		return "", fmt.Errorf("bulk adding data to table: %w", err)
	}

	if err := table.Render(); err != nil {
		return "", fmt.Errorf("rendering table: %w", err)
	}

	return buf.String(), nil
}

func PrintProjectDetails(project *services.Project, short bool) (string, error) {
	gitDir, err := project.GitDir()
	if err != nil {
		return "", fmt.Errorf("failed to get git directory: %w", err)
	}

	// Basic project information
	data := [][]string{
		{"Name", project.Name},
		{"Status", formatProjectStatus(project.Status.String())},
		{"Git URL", project.GitURL},
	}

	if !short {
		// Git authentication section
		authMethod, authUser := getAuthenticationInfo(project)
		data = append(data,
			[]string{"Authentication", authMethod},
		)
		if authUser != "" {
			data = append(data, []string{"Auth User", authUser})
		}

		// Show credential in masked form
		authCredential := getAuthenticationCredential(project)
		if authCredential != "" {
			if project.GitAuth != nil && project.GitAuth.HTTPAuth != nil {
				data = append(data, []string{"Auth Password", authCredential})
			} else if project.GitAuth != nil && project.GitAuth.SSHAuth != nil {
				data = append(data, []string{"SSH Key", authCredential})
			}
		}

		// Repository information
		data = append(data,
			[][]string{
				{"Working Directory", project.WorkingDir},
				{"Git Directory", gitDir},
				{"Last Commit", formatCommitDetails(project.LastCommitStr())},
			}...,
		)

		// Compose configuration
		data = append(data,
			[]string{"Compose Files", formatStringList(project.ComposeFiles)},
		)

		// Environment variables
		if len(project.Variables) > 0 {
			data = append(data,
				[]string{"Environment Variables", formatStringList(project.Variables)},
			)
		} else {
			data = append(data,
				[]string{"Environment Variables", "(none)"},
			)
		}

		// Timestamps
		data = append(data,
			[][]string{
				{"Created At", project.CreatedAt.Format("2006-01-02 15:04:05")},
				{"Updated At", project.UpdatedAt.Format("2006-01-02 15:04:05")},
			}...,
		)

		// System information
		data = append(data,
			[]string{"Project ID", project.ID.String()},
		)
	}

	table, err := PrintTable([]string{}, data)
	if err != nil {
		return "", fmt.Errorf("printing project details table: %w", err)
	}
	return table, nil
}

// getAuthenticationInfo returns the authentication method and user from a project
func getAuthenticationInfo(project *services.Project) (method, user string) {
	if project.GitAuth == nil {
		return "None", ""
	}

	if project.GitAuth.HTTPAuth != nil {
		return "HTTP", project.GitAuth.HTTPAuth.Username
	}

	if project.GitAuth.SSHAuth != nil {
		return "SSH", project.GitAuth.SSHAuth.User
	}

	return "None", ""
}

// getAuthenticationCredential returns the masked credential for display
func getAuthenticationCredential(project *services.Project) string {
	if project.GitAuth == nil {
		return ""
	}

	if project.GitAuth.HTTPAuth != nil {
		password := project.GitAuth.HTTPAuth.Password
		if password == "" {
			return "(not set)"
		}
		return maskSensitiveValue(password)
	}

	if project.GitAuth.SSHAuth != nil {
		key := project.GitAuth.SSHAuth.PrivateKey
		if key == "" {
			return "(not set)"
		}
		return "SSH Private Key (***masked***)"
	}

	return ""
}

// maskSensitiveValue masks a sensitive value for display
func maskSensitiveValue(value string) string {
	if len(value) == 0 {
		return "(not set)"
	}
	if len(value) <= 8 {
		// For very short values, show first and last char
		if len(value) <= 2 {
			return strings.Repeat("*", len(value))
		}
		return string(value[0]) + strings.Repeat("*", len(value)-2) + string(value[len(value)-1])
	}
	// For longer values, show first 3 and last 3 chars
	return value[:3] + strings.Repeat("*", len(value)-6) + value[len(value)-3:]
}

// formatCommitDetails formats commit hash with full and short versions
func formatCommitDetails(commit string) string {
	if commit == "" {
		return "(no commits)"
	}
	if len(commit) > 8 {
		return fmt.Sprintf("%s (%s)", commit[:8], commit)
	}
	return commit
}

// formatStringList formats a list of strings with proper line breaks and numbering
func formatStringList(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	if len(items) == 1 {
		return items[0]
	}

	var result strings.Builder
	for i, item := range items {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(fmt.Sprintf("%d. %s", i+1, item))
	}
	return result.String()
}

func PrintProjectList(projects []*services.Project) (string, error) {
	if len(projects) == 0 {
		return PrintMessage(Plain, "No projects found."), nil
	}

	header := []string{
		"ID",
		"Name",
		"Status",
		"Git URL",
		"Commit",
		"Created At",
		"Updated At",
	}
	var data [][]string
	for _, project := range projects {
		// Format status with color coding
		statusStr := formatProjectStatus(project.Status.String())

		// Truncate Git URL if too long (similar to web UI)
		gitURL := truncateString(project.GitURL, 50)

		// Format commit as short hash (8 chars like web UI)
		commit := formatCommitHash(project.LastCommitStr())

		data = append(data, []string{
			project.ID.String(),
			project.Name,
			statusStr,
			gitURL,
			commit,
			project.CreatedAt.Format("2006-01-02 15:04:05"),
			project.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	table, err := PrintTable(header, data)
	if err != nil {
		return "", fmt.Errorf("printing project list table: %w", err)
	}

	return table, nil
}

// formatProjectStatus applies color coding to project status
func formatProjectStatus(status string) string {
	// If colors are not initialized, return plain status
	if maybeColorize == nil {
		return status
	}

	switch strings.ToLower(status) {
	case "running":
		return maybeColorize(Success, status)
	case "stopped":
		return maybeColorize(Warning, status)
	case "error":
		return maybeColorize(Error, status)
	default:
		return maybeColorize(Plain, status)
	}
}

// truncateString truncates a string to maxLength with "..." if needed
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}

// formatCommitHash formats commit hash as short version (8 chars)
func formatCommitHash(commit string) string {
	if commit == "" {
		return "-"
	}
	if len(commit) > 8 {
		return commit[:8]
	}
	return commit
}

// CLI flag for disabling color output

// NoColor is a flag that can be used to disable colored output in the CLI.
var NoColor = &noColorFlag{set: false}

type noColorFlag struct {
	set bool
}

func (f *noColorFlag) Set(value string) error {
	// This is a boolean flag, so we ignore the value and just mark it as set
	f.set = true
	return nil
}

func (f *noColorFlag) String() string {
	if f.set {
		return "true"
	}
	return "false"
}

func (f *noColorFlag) Type() string {
	return "bool"
}

// IsSet returns true if the --no-color flag was explicitly set
func (f *noColorFlag) IsSet() bool {
	return f.set
}

// IsBoolFlag tells pflag this is a boolean flag (no argument required)
func (f *noColorFlag) IsBoolFlag() bool {
	return true
}

// CLI output helpers

// Fprint writes a formatted message to the writer and returns any error
func Fprint(w io.Writer, kind color.Attribute, format string, args ...any) error {
	message := PrintMessage(kind, format, args...)
	_, err := fmt.Fprint(w, message)
	return err
}

// FprintCmd writes a formatted message to the command's output and returns any error
func FprintCmd(
	cmd interface{ OutOrStdout() io.Writer },
	kind color.Attribute,
	format string,
	args ...any,
) error {
	return Fprint(cmd.OutOrStdout(), kind, format, args...)
}

// Convenience functions for specific color types

func FprintPlain(cmd interface{ OutOrStdout() io.Writer }, format string, args ...any) error {
	return FprintCmd(cmd, Plain, format, args...)
}

func FprintSuccess(cmd interface{ OutOrStdout() io.Writer }, format string, args ...any) error {
	return FprintCmd(cmd, Success, format, args...)
}

func FprintWarning(cmd interface{ OutOrStdout() io.Writer }, format string, args ...any) error {
	return FprintCmd(cmd, Warning, format, args...)
}

func FprintError(cmd interface{ OutOrStdout() io.Writer }, format string, args ...any) error {
	return FprintCmd(cmd, Error, format, args...)
}
