package router

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHelloHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := httptest.NewRecorder()

	HelloHandler(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type application/json, got %q", contentType)
	}

	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Hello from YOLO") {
		t.Errorf("Expected body to contain 'Hello from YOLO', got %q", bodyStr)
	}

	if !strings.Contains(bodyStr, "success") {
		t.Errorf("Expected body to contain 'success', got %q", bodyStr)
	}

	if !strings.Contains(bodyStr, "timestamp") {
		t.Errorf("Expected body to contain 'timestamp', got %q", bodyStr)
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	HealthHandler(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type application/json, got %q", contentType)
	}

	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Service is healthy") {
		t.Errorf("Expected body to contain 'Service is healthy', got %q", bodyStr)
	}

	if !strings.Contains(bodyStr, "ok") {
		t.Errorf("Expected body to contain 'ok', got %q", bodyStr)
	}

	if !strings.Contains(bodyStr, "timestamp") {
		t.Errorf("Expected body to contain 'timestamp', got %q", bodyStr)
	}
}

func TestSetupRoutes(t *testing.T) {
	mux := SetupRoutes()
	if mux == nil {
		t.Fatal("Expected non-nil http.Handler")
	}

	// Test /hello endpoint
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected /hello to return %d, got %d", http.StatusOK, w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	if !strings.Contains(string(body), "Hello from YOLO") {
		t.Errorf("Expected /hello body to contain 'Hello from YOLO', got %q", string(body))
	}

	// Test /health endpoint
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected /health to return %d, got %d", http.StatusOK, w.Code)
	}

	body, _ = io.ReadAll(w.Body)
	if !strings.Contains(string(body), "Service is healthy") {
		t.Errorf("Expected /health body to contain 'Service is healthy', got %q", string(body))
	}
}

func TestResponseJSON(t *testing.T) {
	now := time.Now()
	resp := Response{
		Status:    "test",
		Message:   "Test message",
		Timestamp: now,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal Response: %v", err)
	}
	
	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, `"status":"test"`) {
		t.Errorf("Expected JSON to contain status:test, got: %s", jsonStr)
	}

	if !strings.Contains(jsonStr, `"message":"Test message"`) {
		t.Errorf("Expected JSON to contain message, got: %s", jsonStr)
	}
	
	// Verify the struct has the right fields set
	if resp.Status != "test" {
		t.Errorf("Expected Status to be 'test', got %q", resp.Status)
	}

	if resp.Message != "Test message" {
		t.Errorf("Expected Message to be 'Test message', got %q", resp.Message)
	}

	if !resp.Timestamp.Equal(now) {
		t.Errorf("Expected Timestamp to be equal, got %v", resp.Timestamp)
	}
}

func TestHelloHandlerPOST(t *testing.T) {
	// Test that handler works with different methods (even though it's meant for GET)
	req := httptest.NewRequest(http.MethodPost, "/hello", nil)
	w := httptest.NewRecorder()

	HelloHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestHealthHandlerPOST(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	HealthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSetupRoutesUnknownPath(t *testing.T) {
	mux := SetupRoutes()
	
	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected unknown path to return %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestSetupRoutes_HEADMethod(t *testing.T) {
	mux := SetupRoutes()
	
	req := httptest.NewRequest(http.MethodHead, "/hello", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected /hello HEAD to return %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type application/json, got %q", contentType)
	}
}
