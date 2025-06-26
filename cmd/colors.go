package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	// Color functions - will be set based on environment detection
	colorSuccess func(format string, a ...any) string
	colorError   func(format string, a ...any) string
)

func init() {
	initColors()
}

// initColors sets up color functions based on environment
func initColors() {
	// Check if colors should be disabled
	shouldDisableColors := !isTerminal() || hasNoColorEnv()

	if shouldDisableColors {
		// Disable colors - use plain formatting
		color.NoColor = true
		colorSuccess = func(format string, a ...any) string {
			return fmt.Sprintf(format, a...)
		}
		colorError = func(format string, a ...any) string {
			return fmt.Sprintf(format, a...)
		}
	} else {
		// Enable colors
		color.NoColor = false
		colorSuccess = color.New(color.FgGreen).SprintfFunc()
		colorError = color.New(color.FgRed).SprintfFunc()
	}
}

// isTerminal checks if output is going to a terminal
func isTerminal() bool {
	// Check if stdout is a terminal
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	// Check if it's a character device (terminal)
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// hasNoColorEnv checks for NO_COLOR environment variable
func hasNoColorEnv() bool {
	// NO_COLOR environment variable disables colors
	// See: https://no-color.org/
	return os.Getenv("NO_COLOR") != ""
}

// printSuccess prints a success message in green (if colors enabled)
func printSuccess(format string, a ...any) {
	colored := colorSuccess(format, a...)
	os.Stdout.WriteString(colored + "\n") // nolint:errcheck
}

// printError prints an error message in red (if colors enabled) to stderr
func printError(format string, a ...any) {
	colored := colorError(format, a...)
	os.Stderr.WriteString(colored + "\n") // nolint:errcheck
}
