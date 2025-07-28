package output

import (
	"testing"
)

func TestNoColorFlag_Set(t *testing.T) {
	flag := &noColorFlag{set: false}

	// Test setting the flag
	err := flag.Set("anything") // Value is ignored for boolean flags
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !flag.IsSet() {
		t.Error("Flag should be marked as set after Set() is called")
	}

	if flag.String() != "true" {
		t.Errorf("Expected String() to return 'true', got %q", flag.String())
	}
}

func TestNoColorFlag_Default(t *testing.T) {
	flag := &noColorFlag{set: false}

	if flag.IsSet() {
		t.Error("Flag should not be marked as set initially")
	}

	if flag.String() != "false" {
		t.Errorf("Expected String() to return 'false', got %q", flag.String())
	}

	if flag.Type() != "bool" {
		t.Errorf("Expected Type() to return 'bool', got %q", flag.Type())
	}

	if !flag.IsBoolFlag() {
		t.Error("Expected IsBoolFlag() to return true")
	}
}
