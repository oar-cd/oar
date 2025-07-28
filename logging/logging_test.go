package logging

import (
	"log/slog"
	"testing"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		wantLevel slog.Level
	}{
		{
			name:      "debug level",
			level:     "debug",
			wantLevel: slog.LevelDebug,
		},
		{
			name:      "info level",
			level:     "info",
			wantLevel: slog.LevelInfo,
		},
		{
			name:      "warning level",
			level:     "warning",
			wantLevel: slog.LevelWarn,
		},
		{
			name:      "error level",
			level:     "error",
			wantLevel: slog.LevelError,
		},
		{
			name:      "invalid level defaults to info",
			level:     "invalid",
			wantLevel: slog.LevelInfo,
		},
		{
			name:      "empty string defaults to info",
			level:     "",
			wantLevel: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseLogLevel(tt.level)
			if got != tt.wantLevel {
				t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.level, got, tt.wantLevel)
			}
		})
	}
}

func TestLogLevelFlag_Set(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantError bool
		wantValue string
		wantSet   bool
	}{
		{
			name:      "valid debug level",
			value:     "debug",
			wantError: false,
			wantValue: "debug",
			wantSet:   true,
		},
		{
			name:      "valid info level",
			value:     "info",
			wantError: false,
			wantValue: "info",
			wantSet:   true,
		},
		{
			name:      "valid warning level",
			value:     "warning",
			wantError: false,
			wantValue: "warning",
			wantSet:   true,
		},
		{
			name:      "valid error level",
			value:     "error",
			wantError: false,
			wantValue: "error",
			wantSet:   true,
		},
		{
			name:      "invalid level",
			value:     "invalid",
			wantError: true,
			wantValue: "info", // Should remain at default
			wantSet:   false,  // Should not be marked as set
		},
		{
			name:      "empty string",
			value:     "",
			wantError: true,
			wantValue: "info", // Should remain at default
			wantSet:   false,  // Should not be marked as set
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := &logLevelFlag{value: "info", set: false}

			err := flag.Set(tt.value)

			if tt.wantError && err == nil {
				t.Errorf("logLevelFlag.Set() expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("logLevelFlag.Set() unexpected error: %v", err)
			}

			if flag.String() != tt.wantValue {
				t.Errorf("logLevelFlag.Set() value = %v, want %v", flag.String(), tt.wantValue)
			}

			if flag.IsSet() != tt.wantSet {
				t.Errorf("logLevelFlag.IsSet() = %v, want %v", flag.IsSet(), tt.wantSet)
			}
		})
	}
}

func TestLogLevelFlag_Type(t *testing.T) {
	flag := &logLevelFlag{value: "info", set: false}

	got := flag.Type()
	want := "one of [debug|info|warning|error]"

	if got != want {
		t.Errorf("logLevelFlag.Type() = %v, want %v", got, want)
	}
}
