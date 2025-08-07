package handlers

import (
	"reflect"
	"testing"
)

func TestParseVariablesFromRaw(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single variable",
			input:    "KEY1=value1",
			expected: []string{"KEY1=value1"},
		},
		{
			name:     "multiple variables",
			input:    "KEY1=value1\nKEY2=value2\nKEY3=value3",
			expected: []string{"KEY1=value1", "KEY2=value2", "KEY3=value3"},
		},
		{
			name:     "variables with spaces in values",
			input:    "KEY1=value with spaces\nKEY2=another value",
			expected: []string{"KEY1=value with spaces", "KEY2=another value"},
		},
		{
			name:     "variables with equals in values",
			input:    "KEY1=value=with=equals\nKEY2=url=https://example.com",
			expected: []string{"KEY1=value=with=equals", "KEY2=url=https://example.com"},
		},
		{
			name:     "skip comments",
			input:    "# This is a comment\nKEY1=value1\n# Another comment\nKEY2=value2",
			expected: []string{"KEY1=value1", "KEY2=value2"},
		},
		{
			name:     "skip empty lines",
			input:    "KEY1=value1\n\n\nKEY2=value2\n\n",
			expected: []string{"KEY1=value1", "KEY2=value2"},
		},
		{
			name:     "trim whitespace around keys and values",
			input:    "   KEY1   =   value1   \n\t\tKEY2\t=\tvalue2\t\n",
			expected: []string{"KEY1=value1", "KEY2=value2"},
		},
		{
			name:     "skip lines without equals",
			input:    "KEY1=value1\ninvalid line without equals\nKEY2=value2",
			expected: []string{"KEY1=value1", "KEY2=value2"},
		},
		{
			name:     "skip empty keys",
			input:    "KEY1=value1\n=value_without_key\nKEY2=value2",
			expected: []string{"KEY1=value1", "KEY2=value2"},
		},
		{
			name:     "handle empty values",
			input:    "KEY1=value1\nEMPTY_KEY=\nKEY2=value2",
			expected: []string{"KEY1=value1", "EMPTY_KEY=", "KEY2=value2"},
		},
		{
			name: "mixed content with all edge cases",
			input: `# Environment variables for the application
# Database configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=myapp

# API configuration  
API_KEY=secret=key=with=equals
API_URL=https://api.example.com

# Empty values
EMPTY_VAR=
   TRIMMED_KEY   =   trimmed value   

# Invalid lines (should be ignored)
invalid line without equals
=value_without_key
# More comments

# Final variable
DEBUG=true`,
			expected: []string{
				"DB_HOST=localhost",
				"DB_PORT=5432",
				"DB_NAME=myapp",
				"API_KEY=secret=key=with=equals",
				"API_URL=https://api.example.com",
				"EMPTY_VAR=",
				"TRIMMED_KEY=trimmed value",
				"DEBUG=true",
			},
		},
		{
			name:     "only comments and empty lines",
			input:    "# Comment 1\n\n# Comment 2\n\n\n",
			expected: []string{},
		},
		{
			name:     "only invalid lines",
			input:    "invalid line 1\ninvalid line 2\nno equals here",
			expected: []string{},
		},
		{
			name:     "windows line endings",
			input:    "KEY1=value1\r\nKEY2=value2\r\n",
			expected: []string{"KEY1=value1", "KEY2=value2"},
		},
		{
			name:     "special characters in keys and values",
			input:    "KEY_WITH_UNDERSCORES=value\nKEY-WITH-DASHES=value@#$%^&*()\nNUMBERS123=456789",
			expected: []string{"KEY_WITH_UNDERSCORES=value", "KEY-WITH-DASHES=value@#$%^&*()", "NUMBERS123=456789"},
		},
		{
			name:     "multiple consecutive equals",
			input:    "KEY1===value\nKEY2=value===more\nKEY3====",
			expected: []string{"KEY1===value", "KEY2=value===more", "KEY3===="},
		},
		{
			name:     "mixed line endings",
			input:    "KEY1=value1\nKEY2=value2\r\nKEY3=value3",
			expected: []string{"KEY1=value1", "KEY2=value2", "KEY3=value3"},
		},
		{
			name:     "keys with numbers and special chars",
			input:    "API_V2_KEY=secret\nDB_HOST_1=localhost\n_PRIVATE_KEY=private",
			expected: []string{"API_V2_KEY=secret", "DB_HOST_1=localhost", "_PRIVATE_KEY=private"},
		},
		{
			name:  "very long key and value",
			input: "VERY_LONG_KEY_NAME_WITH_MANY_CHARACTERS=very_long_value_that_contains_many_characters_and_should_be_handled_correctly",
			expected: []string{
				"VERY_LONG_KEY_NAME_WITH_MANY_CHARACTERS=very_long_value_that_contains_many_characters_and_should_be_handled_correctly",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVariablesFromRaw(tt.input)

			// Handle nil vs empty slice comparison
			if len(result) == 0 && len(tt.expected) == 0 {
				return // Both are effectively empty
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseVariablesFromRaw() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
