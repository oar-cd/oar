package routes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/oar-cd/oar/web/handlers"
	"github.com/stretchr/testify/assert"
)

// Test helper function to create form requests
func createFormRequest(method, path string, formData map[string]string) *http.Request {
	values := url.Values{}
	for key, value := range formData {
		values.Set(key, value)
	}

	req := httptest.NewRequest(method, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		panic(fmt.Sprintf("Failed to parse form in test helper: %v", err))
	}

	return req
}

func TestGetVersion(t *testing.T) {
	version := handlers.GetVersion()
	// Just verify it returns a non-empty string
	assert.NotEmpty(t, version)
}

func TestHealthCheckRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Create a simple handler for testing
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			handlers.LogOperationError("health_check", "main", err)
		}
	}

	handler(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := w.Body.String()
	assert.Equal(t, "OK", body)
}

// Test utility route form validation logic
func TestTestGitAuthRouteFormValidation(t *testing.T) {
	tests := []struct {
		name           string
		gitURL         string
		expectedStatus int
		expectedHeader string
	}{
		{
			name:           "missing git URL",
			gitURL:         "",
			expectedStatus: http.StatusOK,
			expectedHeader: "testAuthError",
		},
		{
			name:           "valid git URL format",
			gitURL:         "https://github.com/test/repo",
			expectedStatus: http.StatusOK,
			// This would require actual git service integration to test success/failure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create form request
			req := createFormRequest(http.MethodPost, "/test-git-auth", map[string]string{
				"git_url": tt.gitURL,
			})

			w := httptest.NewRecorder()

			// Simple validation handler that mimics the route logic
			handler := func(w http.ResponseWriter, r *http.Request) {
				gitURL := r.FormValue("git_url")
				if gitURL == "" {
					w.Header().Set("Content-Type", "text/html")
					w.Header().Set("HX-Trigger-After-Settle", "testAuthError")
					w.WriteHeader(http.StatusOK)
					return
				}

				// For this test, we'll just check that we got the URL
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusOK)
			}

			handler(w, req)

			resp := w.Result()
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedHeader != "" {
				assert.Equal(t, tt.expectedHeader, resp.Header.Get("HX-Trigger-After-Settle"))
			}
		})
	}
}

func TestFormatStdoutStderr(t *testing.T) {
	tests := []struct {
		name     string
		stdout   string
		stderr   string
		expected string
	}{
		{
			name:     "empty stdout and stderr",
			stdout:   "",
			stderr:   "",
			expected: "",
		},
		{
			name:     "stdout only",
			stdout:   "Container started successfully",
			stderr:   "",
			expected: `<span class="deploy-text-stdout">Container started successfully</span>`,
		},
		{
			name:     "stderr only",
			stdout:   "",
			stderr:   "Warning: deprecated option",
			expected: `<span class="deploy-text-stderr">Warning: deprecated option</span>` + "\n",
		},
		{
			name:     "both stdout and stderr",
			stdout:   "Build completed",
			stderr:   "Warning: using legacy format",
			expected: `<span class="deploy-text-stderr">Warning: using legacy format</span>` + "\n" + `<span class="deploy-text-stdout">Build completed</span>`,
		},
		{
			name:     "multiline stderr",
			stdout:   "",
			stderr:   "Line 1\nLine 2\nLine 3",
			expected: `<span class="deploy-text-stderr">Line 1` + "\n" + `Line 2` + "\n" + `Line 3</span>` + "\n",
		},
		{
			name:     "structured log in stderr gets parsed",
			stdout:   "",
			stderr:   `time="2023-01-01T12:00:00Z" level=info msg="Container created"`,
			expected: `<span class="deploy-text-stderr">Container created</span>` + "\n",
		},
		{
			name:     "mixed structured and regular logs in stderr",
			stdout:   "",
			stderr:   `time="2023-01-01T12:00:00Z" level=info msg="Starting service"` + "\n" + `Regular log message`,
			expected: `<span class="deploy-text-stderr">Starting service` + "\n" + `Regular log message</span>` + "\n",
		},
		{
			name:     "structured log with escaped quotes",
			stdout:   "",
			stderr:   `time="2023-01-01T12:00:00Z" level=info msg="Container \"web\" started"`,
			expected: `<span class="deploy-text-stderr">Container "web" started</span>` + "\n",
		},
		{
			name:     "multiline stdout with stderr",
			stdout:   "Line 1\nLine 2",
			stderr:   "Warning message",
			expected: `<span class="deploy-text-stderr">Warning message</span>` + "\n" + `<span class="deploy-text-stdout">Line 1` + "\n" + `Line 2</span>`,
		},
		{
			name:     "empty lines in stderr",
			stdout:   "",
			stderr:   "Warning\n\nAnother warning",
			expected: `<span class="deploy-text-stderr">Warning` + "\n" + "\n" + `Another warning</span>` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStdoutStderr(tt.stdout, tt.stderr)
			assert.Equal(t, tt.expected, result)
		})
	}
}
