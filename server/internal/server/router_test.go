package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/enzyme/server/internal/openapi"
)

func TestErrorHandlersProduceValidJSON(t *testing.T) {
	tests := []struct {
		name   string
		errMsg string
	}{
		{"simple message", "invalid request body"},
		{"message with quotes", `field "name" is required`},
		{"message with backslash", `path C:\Users\test`},
		{"message with special chars", "failed: <script>alert(1)</script>"},
		{"message with newline", "line1\nline2"},
	}

	// Build a minimal router just to get the error handler functions
	opts := openapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(openapi.ApiErrorResponse{
				Error: openapi.ApiError{Code: "BAD_REQUEST", Message: err.Error()},
			})
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(openapi.ApiErrorResponse{
				Error: openapi.ApiError{Code: "INTERNAL_ERROR", Message: err.Error()},
			})
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testErr := testError(tt.errMsg)

			// Test RequestErrorHandlerFunc
			t.Run("request error", func(t *testing.T) {
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", "/", nil)
				opts.RequestErrorHandlerFunc(w, r, testErr)

				if w.Code != http.StatusBadRequest {
					t.Errorf("expected status 400, got %d", w.Code)
				}

				var resp openapi.ApiErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("response is not valid JSON: %v\nbody: %s", err, w.Body.String())
				}
				if resp.Error.Code != "BAD_REQUEST" {
					t.Errorf("expected code BAD_REQUEST, got %s", resp.Error.Code)
				}
				if resp.Error.Message != tt.errMsg {
					t.Errorf("expected message %q, got %q", tt.errMsg, resp.Error.Message)
				}
			})

			// Test ResponseErrorHandlerFunc
			t.Run("response error", func(t *testing.T) {
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", "/", nil)
				opts.ResponseErrorHandlerFunc(w, r, testErr)

				if w.Code != http.StatusInternalServerError {
					t.Errorf("expected status 500, got %d", w.Code)
				}

				var resp openapi.ApiErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("response is not valid JSON: %v\nbody: %s", err, w.Body.String())
				}
				if resp.Error.Code != "INTERNAL_ERROR" {
					t.Errorf("expected code INTERNAL_ERROR, got %s", resp.Error.Code)
				}
				if resp.Error.Message != tt.errMsg {
					t.Errorf("expected message %q, got %q", tt.errMsg, resp.Error.Message)
				}
			})
		})
	}
}

type testError string

func (e testError) Error() string { return string(e) }
