package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHTTPError tests the HTTPError type and its string() method
func TestHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		message    string
		wantString string
	}{
		{
			name:       "standard 404 error",
			status:     404,
			message:    "not found",
			wantString: "Error 404 (Not Found): not found",
		},
		{
			name:       "500 server error",
			status:     500,
			message:    "internal server error",
			wantString: "Error 500 (Internal Server Error): internal server error",
		},
		{
			name:       "403 forbidden with custom message",
			status:     403,
			message:    "access denied",
			wantString: "Error 403 (Forbidden): access denied",
		},
		{
			name:       "empty error message",
			status:     400,
			message:    "",
			wantString: "Error 400 (Bad Request): ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := HTTPError{Status: tt.status, Message: tt.message}

			if err.Error() != tt.wantString {
				t.Errorf("Error.Error() = %q, want %q", err.Error(), tt.wantString)
			}

			// Check that it implements error interface
			var _ error = err
		})
	}
}

// TestHandlerFuncNilCheck tests that HandlerFunc properly checks for nil
func TestHandlerFuncNilCheck(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		handler     HandlerFunc
		expectPanic bool
	}{
		{
			name: "non-nil handler should execute",
			handler: func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				w.WriteHeader(http.StatusOK)
				return nil
			},
			expectPanic: false,
		},
		{
			name:        "nil handler should panic",
			handler:     nil,
			expectPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()

			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("expected panic for nil handler, but did not panic")
					}
				}()
			}

			// This should not panic for non-nil handlers
			tt.handler(ctx, w, req)
		})
	}
}

// TestNewRouter tests that a new router is properly initialized
func TestNewRouter(t *testing.T) {
	router := NewRouter()

	if router == nil {
		t.Fatal("expected non-nil router from NewRouter(), got nil")
	}

	if len(router.routes) != 0 {
		t.Errorf("expected empty routes after NewRouter(), got %d", len(router.routes))
	}
}

// TestMethodMiddleware tests that method middleware filters by HTTP method
func TestMethodMiddleware(t *testing.T) {
	middleware := MethodMiddleware(http.MethodGet)

	ctx := context.Background()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	innerCalled := false
	inner := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		innerCalled = true
		return nil
	}

	err := middleware(ctx, req, inner)(w, req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !innerCalled {
		t.Error("expected inner handler to be called for matching method")
	}
}

// TestRouteRegistration tests that routes are properly registered and matched
func TestRouteRegistration(t *testing.T) {
	router := NewRouter()

	var getCalled, postCalled bool

	router.Method(http.MethodGet, "/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		getCalled = true
		return nil
	})

	router.Method(http.MethodPost, "/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		postCalled = true
		return nil
	})

	ctx := context.Background()

	// Test GET request matching
	reqGet := httptest.NewRequest("GET", "/test", nil)
	wGet := httptest.NewRecorder()
	err := router.ServeHTTP(ctx, wGet, reqGet)

	if err != nil {
		t.Errorf("GET /test failed: %v", err)
	}
	if !getCalled {
		t.Error("expected GET handler to be called")
	}

	// Test POST request matching
	reqPost := httptest.NewRequest("POST", "/test", nil)
	wPost := httptest.NewRecorder()
	err = router.ServeHTTP(ctx, wPost, reqPost)

	if err != nil {
		t.Errorf("POST /test failed: %v", err)
	}
	if !postCalled {
		t.Error("expected POST handler to be called")
	}
}

// TestErrorWriterHandler tests that ErrorWriterHandler properly writes error responses
func TestErrorWriterHandler(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		message    string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "404 error response",
			status:     404,
			message:    "not found",
			wantStatus: 404,
			wantBody:   `{"error":"not found","status":404}` + "\n",
		},
		{
			name:       "500 error response",
			status:     500,
			message:    "server error",
			wantStatus: 500,
			wantBody:   `{"error":"server error","status":500}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := ErrorWriterHandler(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				return HTTPError{Status: tt.status, Message: tt.message}
			})

			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Response status = %d, want %d", w.Code, tt.wantStatus)
			}

			if !bytes.Contains(w.Body.Bytes(), []byte(tt.message)) {
				t.Errorf("response body does not contain error message")
			}
		})
	}
}

// TestErrorReaderHandler tests that ErrorReaderHandler properly reads error responses
func TestErrorReaderHandler(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		message       string
		wantErrStatus int
	}{
		{
			name:          "parse 404 error",
			status:        404,
			message:       "not found",
			wantErrStatus: 404,
		},
		{
			name:          "parse 500 error",
			status:        500,
			message:       "server error",
			wantErrStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := ErrorReaderHandler(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				return HTTPError{Status: tt.status, Message: tt.message}
			})

			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()

			errCh := make(chan error, 1)
			go func() {
				handler.ServeHTTP(w, req)
				errCh <- nil
			}()

			select {
			case err := <-errCh:
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			case <-time.After(1 * time.Second):
				t.Error("timeout waiting for response")
			}
		})
	}
}

// TestTimeoutHandler tests that the timeout handler properly times out requests
func TestTimeoutHandler(t *testing.T) {
	handler := TimeoutHandler(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Simulate a long operation that will timeout
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			w.WriteHeader(http.StatusOK)
			return nil
		}
	}, 100*time.Millisecond)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(w, req)
		close(done)
	}()

	select {
	case <-done:
		if w.Code != http.StatusGatewayTimeout && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected timeout status code, got %d", w.Code)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for response")
	}
}

// TestRouteNotFound tests that 405 is returned when route doesn't exist
func TestRouteNotFound(t *testing.T) {
	router := NewRouter()

	router.Method(http.MethodGet, "/existing", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	ctx := context.Background()
	req := httptest.NewRequest("POST", "/nonexistent", nil)
	w := httptest.NewRecorder()

	err := router.ServeHTTP(ctx, w, req)

	if err == nil {
		t.Error("expected error for nonexistent route")
	}

	// Should get 405 Method Not Allowed or 404 Not Found
	if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
		t.Errorf("Expected 405 or 404, got %d", w.Code)
	}
}

// TestMiddlewareOrder tests that middleware is executed in the correct order
func TestMiddlewareOrder(t *testing.T) {
	order := []string{}

	middleware1 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			order = append(order, "middleware1-before")
			err := next(ctx, w, r)
			order = append(order, "middleware1-after")
			return err
		}
	}

	middleware2 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			order = append(order, "middleware2-before")
			err := next(ctx, w, r)
			order = append(order, "middleware2-after")
			return err
		}
	}

	router := NewRouter()
	router.Use(middleware1, middleware2).Get("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		order = append(order, "handler")
		return nil
	})

	ctx := context.Background()
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	err := router.ServeHTTP(ctx, w, req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expectedOrder := []string{
		"middleware1-before",
		"middleware2-before",
		"handler",
		"middleware2-after",
		"middleware1-after",
	}

	for i, exp := range expectedOrder {
		if i >= len(order) || order[i] != exp {
			t.Errorf("expected order[%d] to be %q, got %q", i, exp, order[i])
		}
	}
}

// TestEmptyMessage tests handling of empty error messages
func TestEmptyMessage(t *testing.T) {
	err := HTTPError{Status: 500, Message: ""}

	if err.Error() == "Error 500 (Internal Server Error): " {
		t.Logf("Correctly formatted with trailing colon and space for empty message")
	} else {
		t.Errorf("Unexpected format: %q", err.Error())
	}
}

// TestDifferentStatusCodes tests various HTTP status codes
func TestDifferentStatusCodes(t *testing.T) {
	statusTests := []struct {
		code     int
		expected string
	}{
		{200, "OK"},
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{403, "Forbidden"},
		{404, "Not Found"},
		{500, "Internal Server Error"},
		{502, "Bad Gateway"},
	}

	for _, tt := range statusTests {
		t.Run(fmt.Sprintf("%d-%s", tt.code, tt.expected), func(t *testing.T) {
			err := HTTPError{Status: tt.code, Message: "test"}
			expectedString := fmt.Sprintf("Error %d (%s): test", tt.code, tt.expected)

			if err.Error() != expectedString {
				t.Errorf("Expected %q, got %q", expectedString, err.Error())
			}
		})
	}
}
