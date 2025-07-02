// Package output provides functions to print messages with optional color formatting
package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/ch00k/oar/services"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
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
func PrintMessage(kind color.Attribute, tmpl string, a ...any) {
	if maybeColorize == nil || kind == Plain {
		fmt.Printf(tmpl+"\n", a...)
	} else {
		// TODO: Print warnings and errors to stderr?
		fmt.Println(maybeColorize(kind, tmpl, a...))
	}
}

func PrintTable(header []string, data [][]string) error {
	table := tablewriter.NewWriter(os.Stdout)

	if len(header) > 0 {
		table.Header(header)
	}

	if err := table.Bulk(data); err != nil {
		return fmt.Errorf("bulk adding data to table: %w", err)
	}

	if err := table.Render(); err != nil {
		return fmt.Errorf("rendering table: %w", err)
	}

	return nil
}

func PrintProjectDetails(project *services.Project) error {
	gitDir, err := project.GitDir()
	if err != nil {
		return fmt.Errorf("failed to get git directory: %w", err)
	}

	data := [][]string{
		{"ID", project.ID.String()},
		{"Name", project.Name},
		{"Working Directory", project.WorkingDir},
		{"Git Directory", gitDir},
		{"Git URL", project.GitURL},
		{"Compose Files", strings.Join(project.ComposeFiles, ", ")},
		{"Environment Files", strings.Join(project.EnvironmentFiles, ", ")},
		{"Status", project.Status.String()},
		{"Last Commit", project.LastCommitStr()},
		{"Created At", project.CreatedAt.Format("2006-01-02 15:04:05")},
		{"Updated At", project.UpdatedAt.Format("2006-01-02 15:04:05")},
	}

	return PrintTable([]string{}, data)
}

func PrintProjectList(projects []*services.Project) error {
	if len(projects) == 0 {
		PrintMessage(Plain, "No projects found.")
		return nil
	}

	header := []string{
		"ID",
		"Name",
		//"Working Directory",
		//"Git Directory",
		//"Git URL",
		//"Compose Files",
		//"Environment Files",
		"Status",
		//"Last Commit",
		"Created At",
		"Updated At",
	}
	var data [][]string
	for _, project := range projects {
		//gitDir, err := project.GitDir()
		//if err != nil {
		//    return fmt.Errorf("failed to get git directory: %w", err)
		//}

		data = append(data, []string{
			project.ID.String(),
			project.Name,
			//project.WorkingDir,
			// gitDir,
			// project.GitURL,
			// strings.Join(project.ComposeFiles, ", "),
			// strings.Join(project.EnvironmentFiles, ", "),
			// project.LastCommitStr(),
			project.Status.String(),
			project.CreatedAt.Format("2006-01-02 15:04:05"),
			project.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return PrintTable(header, data)
}
