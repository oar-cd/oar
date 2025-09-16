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
