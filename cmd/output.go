package cmd

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

func init() {
	initColors()
}

// initColors sets up color functions based on environment
func initColors() {
	// Check if colors should be enabled
	if color.NoColor || colorDisabled {
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

// printMessage formats a message with color (if enabled) and prints it
func printMessage(kind color.Attribute, tmpl string, a ...any) {
	if maybeColorize == nil || kind == Plain {
		fmt.Printf(tmpl+"\n", a...)
	} else {
		// TODO: Print warnings and errors to stderr?
		fmt.Println(maybeColorize(kind, tmpl, a...))
	}
}
