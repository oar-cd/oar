// Package output provides functions to print messages with optional color formatting
package output

import (
	"fmt"

	"github.com/fatih/color"
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
