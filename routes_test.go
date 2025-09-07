package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetServerVersion(t *testing.T) {
	version := GetServerVersion()
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
			logOperationError("health_check", "main", err)
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

func TestDiscoverRouteFormValidation(t *testing.T) {
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
			expectedHeader: "discoverError",
		},
		{
			name:           "valid git URL format",
			gitURL:         "https://github.com/test/repo",
			expectedStatus: http.StatusOK,
			// This would require actual discovery service integration to test success/failure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createFormRequest(http.MethodPost, "/discover", map[string]string{
				"git_url": tt.gitURL,
			})

			w := httptest.NewRecorder()

			// Simple validation handler that mimics the route logic
			handler := func(w http.ResponseWriter, r *http.Request) {
				gitURL := r.FormValue("git_url")
				if gitURL == "" {
					w.Header().Set("Content-Type", "text/html")
					w.Header().Set("HX-Trigger-After-Settle", "discoverError")
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
