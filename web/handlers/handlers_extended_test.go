package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/oar-cd/oar/services"
	"github.com/stretchr/testify/assert"
)

func TestSetupSSE(t *testing.T) {
	w := httptest.NewRecorder()

	SetupSSE(w)

	// Verify all required SSE headers are set
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestLogOperationError(t *testing.T) {
	// This test mainly verifies that logOperationError doesn't panic
	// and can be called with different parameter combinations
	tests := []struct {
		name      string
		operation string
		layer     string
		err       error
		fields    []any
	}{
		{
			name:      "basic error logging",
			operation: "test_operation",
			layer:     "test_layer",
			err:       fmt.Errorf("test error"),
			fields:    nil,
		},
		{
			name:      "error logging with fields",
			operation: "create_project",
			layer:     "handlers",
			err:       fmt.Errorf("validation failed"),
			fields:    []any{"project_id", "123", "validation_type", "required_field"},
		},
		{
			name:      "error logging with empty fields",
			operation: "delete_project",
			layer:     "services",
			err:       fmt.Errorf("project not found"),
			fields:    []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic
			assert.NotPanics(t, func() {
				LogOperationError(tt.operation, tt.layer, tt.err, tt.fields...)
			})
		})
	}
}

func TestStreamOutput(t *testing.T) {
	tests := []struct {
		name           string
		outputMessages []services.StreamMessage
		streamType     string
		expectError    bool
	}{
		{
			name:           "empty stream",
			outputMessages: []services.StreamMessage{},
			streamType:     "deployment",
			expectError:    false,
		},
		{
			name:           "single message stream",
			outputMessages: []services.StreamMessage{{Type: "stdout", Content: "Starting deployment"}},
			streamType:     "deployment",
			expectError:    false,
		},
		{
			name: "multiple messages stream",
			outputMessages: []services.StreamMessage{
				{Type: "stdout", Content: "Step 1: Pulling images"},
				{Type: "stdout", Content: "Step 2: Starting containers"},
				{Type: "stdout", Content: "Step 3: Running health checks"},
			},
			streamType:  "deployment",
			expectError: false,
		},
		{
			name:           "logs stream",
			outputMessages: []services.StreamMessage{{Type: "stdout", Content: "Application started on port 8080"}},
			streamType:     "logs",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a ResponseWriter that supports flushing
			w := httptest.NewRecorder()

			// Create a channel and populate it with test messages
			outputChan := make(chan services.StreamMessage, len(tt.outputMessages)+1)
			for _, msg := range tt.outputMessages {
				outputChan <- msg
			}
			close(outputChan)

			// Call streamOutput
			err := StreamOutput(w, outputChan, tt.streamType)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify the output contains expected SSE format
				output := w.Body.String()

				// Should contain connection message
				assert.Contains(t, output, fmt.Sprintf("Connected to %s stream", tt.streamType))

				// Should contain completion message
				expectedCompletion := strings.ToUpper(tt.streamType[:1]) + tt.streamType[1:] + " finished"
				assert.Contains(t, output, expectedCompletion)

				// Should contain all the messages
				for _, msg := range tt.outputMessages {
					assert.Contains(t, output, msg.Content)
				}

				// Verify SSE format (data: prefix and double newlines)
				lines := strings.Split(output, "\n")
				for _, line := range lines {
					if strings.TrimSpace(line) != "" {
						assert.True(t, strings.HasPrefix(line, "data: "), "Line should start with 'data: ': %s", line)
					}
				}
			}
		})
	}
}

func TestRenderComponent(t *testing.T) {
	// Create a simple test component
	testComponent := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := w.Write([]byte("<div>Test Component</div>"))
		return err
	})

	errorComponent := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return fmt.Errorf("render error")
	})

	tests := []struct {
		name         string
		component    templ.Component
		operation    string
		expectError  bool
		expectStatus int
	}{
		{
			name:         "successful render",
			component:    testComponent,
			operation:    "test_render",
			expectError:  false,
			expectStatus: http.StatusOK,
		},
		{
			name:         "render error",
			component:    errorComponent,
			operation:    "test_render_error",
			expectError:  true,
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			err := RenderComponent(w, r, tt.component, tt.operation)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectStatus, w.Code)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, w.Body.String(), "Test Component")
			}
		})
	}
}

func TestWithFormParsing(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		contentType  string
		body         string
		expectError  bool
		expectStatus int
	}{
		{
			name:         "valid form data",
			method:       http.MethodPost,
			contentType:  "application/x-www-form-urlencoded",
			body:         "name=test&value=123",
			expectError:  false,
			expectStatus: http.StatusOK,
		},
		{
			name:         "GET request (no form data)",
			method:       http.MethodGet,
			contentType:  "",
			body:         "",
			expectError:  false,
			expectStatus: http.StatusOK,
		},
		{
			name:         "invalid form data",
			method:       http.MethodPost,
			contentType:  "application/x-www-form-urlencoded",
			body:         "invalid%form%data%",
			expectError:  true, // ParseForm fails on invalid URL escapes
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that checks if form was parsed
			testHandler := func(w http.ResponseWriter, r *http.Request) {
				// Try to access form values - this should work if parsing succeeded
				_ = r.FormValue("name")
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte("OK")); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}

			// Wrap the test handler with withFormParsing middleware
			wrappedHandler := WithFormParsing(testHandler)

			// Create request
			r := httptest.NewRequest(tt.method, "/", strings.NewReader(tt.body))
			if tt.contentType != "" {
				r.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()

			// Call the wrapped handler
			wrappedHandler(w, r)

			assert.Equal(t, tt.expectStatus, w.Code)
			if !tt.expectError {
				assert.Equal(t, "OK", w.Body.String())
			} else {
				assert.Contains(t, w.Body.String(), "Bad Request")
			}
		})
	}
}
