package services

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatErrorForUser(t *testing.T) {
	tests := []struct {
		name        string
		inputError  error
		expectedMsg string
	}{
		{
			name:        "nil error",
			inputError:  nil,
			expectedMsg: "",
		},
		{
			name:        "git authentication failed",
			inputError:  errors.New("authentication failed"),
			expectedMsg: "git authentication failed - please check your credentials",
		},
		{
			name:        "git terminal prompts disabled",
			inputError:  errors.New("terminal prompts disabled"),
			expectedMsg: "git authentication required - repository needs credentials to access",
		},
		{
			name:        "git could not read username",
			inputError:  errors.New("could not read Username for 'https://github.com'"),
			expectedMsg: "git authentication required - please provide valid credentials",
		},
		{
			name:        "ssh key authentication failed",
			inputError:  errors.New("permission denied (publickey)"),
			expectedMsg: "ssh key authentication failed - please check your private key",
		},
		{
			name:        "repository not found",
			inputError:  errors.New("repository not found"),
			expectedMsg: "git repository not found - please check the URL and your access permissions",
		},
		{
			name:        "unique constraint on name",
			inputError:  errors.New("UNIQUE constraint failed: projects.name"),
			expectedMsg: "a project with this name already exists",
		},
		{
			name:        "unknown error",
			inputError:  errors.New("some random error"),
			expectedMsg: "an unexpected error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatErrorForUser(tt.inputError)
			assert.Equal(t, tt.expectedMsg, result)
		})
	}
}
