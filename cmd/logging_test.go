package cmd

import (
	"log/slog"
	"testing"
)

func TestLogLevelValue_Set(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantError bool
		wantValue string
	}{
		{
			name:      "valid debug level",
			value:     "debug",
			wantError: false,
			wantValue: "debug",
		},
		{
			name:      "valid info level",
			value:     "info",
			wantError: false,
			wantValue: "info",
		},
		{
			name:      "valid warning level",
			value:     "warning",
			wantError: false,
			wantValue: "warning",
		},
		{
			name:      "valid error level",
			value:     "error",
			wantError: false,
			wantValue: "error",
		},
		{
			name:      "invalid level",
			value:     "invalid",
			wantError: true,
			wantValue: "info", // Should remain at default
		},
		{
			name:      "empty string",
			value:     "",
			wantError: true,
			wantValue: "info", // Should remain at default
		},
		{
			name:      "case sensitive",
			value:     "DEBUG",
			wantError: true,
			wantValue: "info", // Should remain at default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lv := newLogLevelValue("info", []string{"debug", "info", "warning", "error"})

			err := lv.Set(tt.value)

			if tt.wantError && err == nil {
				t.Errorf("logLevelValue.Set() expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("logLevelValue.Set() unexpected error: %v", err)
			}

			if lv.String() != tt.wantValue {
				t.Errorf("logLevelValue.Set() value = %v, want %v", lv.String(), tt.wantValue)
			}
		})
	}
}

func TestLogLevelValue_SlogValue(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lv := &logLevelValue{
				value:   tt.level,
				allowed: []string{"debug", "info", "warning", "error"},
			}

			got := lv.slogValue()
			if got != tt.wantLevel {
				t.Errorf("logLevelValue.slogValue() = %v, want %v", got, tt.wantLevel)
			}
		})
	}
}

func TestLogLevelValue_Type(t *testing.T) {
	lv := newLogLevelValue("info", []string{"debug", "info", "warning", "error"})

	got := lv.Type()
	want := "one of [debug|info|warning|error]"

	if got != want {
		t.Errorf("logLevelValue.Type() = %v, want %v", got, want)
	}
}

func TestLogLevelValue_String(t *testing.T) {
	lv := newLogLevelValue("warning", []string{"debug", "info", "warning", "error"})

	got := lv.String()
	want := "warning"

	if got != want {
		t.Errorf("logLevelValue.String() = %v, want %v", got, want)
	}
}
