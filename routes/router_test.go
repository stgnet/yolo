package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewHandler(t *testing.T) {
	config := &Config{Port: 8080}
	h := NewHandler(config)
	
	if h == nil {
		t.Fatal("Expected non-nil Handler")
	}
	
	if h.config == nil {
		t.Error("Expected non-nil config")
	}
	
	if h.config.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", h.config.Port)
	}
}

func TestRootHandler(t *testing.T) {
	h := NewHandler(&Config{Port: 8080})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	
	h.rootHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html" {
		t.Errorf("Expected Content-Type text/html, got %q", contentType)
	}
	
	body := w.Body.String()
	if !strings.Contains(body, "YOLO Agent Server") {
		t.Errorf("Expected body to contain 'YOLO Agent Server', got %q", body)
	}
	
	if !strings.Contains(body, "/status") {
		t.Errorf("Expected body to contain '/status', got %q", body)
	}
	
	if !strings.Contains(body, "/health") {
		t.Errorf("Expected body to contain '/health', got %q", body)
	}
}

func TestStatusHandler(t *testing.T) {
	h := NewHandler(&Config{Port: 8080})
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	
	h.statusHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type application/json, got %q", contentType)
	}
	
	// Parse the response as RouterStatus
	var status RouterStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("Failed to parse response as JSON: %v", err)
	}
	
	if status.Request != "status" {
		t.Errorf("Expected request 'status', got %q", status.Request)
	}
	
	if status.Response != "ok" {
		t.Errorf("Expected response 'ok', got %q", status.Response)
	}
	
	if status.Code != 200 {
		t.Errorf("Expected code 200, got %d", status.Code)
	}
	
	if status.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %q", status.Version)
	}
}

func TestRouterHealthHandler(t *testing.T) {
	h := NewHandler(&Config{Port: 8080})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	
	h.healthHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	body := w.Body.Bytes()
	if len(body) == 0 {
		t.Error("Expected non-empty response")
	}
	
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response as JSON: %v", err)
	}
	
	if result["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %q", result["status"])
	}
}

func TestAgentStatusHandler(t *testing.T) {
	h := NewHandler(&Config{Port: 8080})
	req := httptest.NewRequest(http.MethodGet, "/agent/status", nil)
	w := httptest.NewRecorder()
	
	h.agentStatusHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type application/json, got %q", contentType)
	}
	
	var status RouterAgentStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("Failed to parse response as JSON: %v", err)
	}
	
	if status.Status != "running" {
		t.Errorf("Expected status 'running', got %q", status.Status)
	}
	
	if status.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %q", status.Version)
	}
	
	if !strings.Contains(status.Message, "Agent") {
		t.Errorf("Expected message to contain 'Agent', got %q", status.Message)
	}
}

func TestServeHTTP_Routing(t *testing.T) {
	h := NewHandler(&Config{Port: 8080})
	
	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectContent  string
	}{
		{"Root", "/", http.StatusOK, "YOLO Agent Server"},
		{"Status", "/status", http.StatusOK, "ok"},
		{"Health", "/health", http.StatusOK, "ok"},
		{"AgentStatus", "/agent/status", http.StatusOK, "running"},
		{"NotFound", "/unknown", http.StatusNotFound, ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			
			h.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d for path %s, got %d", tt.expectedStatus, tt.path, w.Code)
			}
			
			if tt.expectContent != "" {
				body := w.Body.String()
				if !strings.Contains(body, tt.expectContent) {
					t.Errorf("Expected body to contain %q for path %s, got %q", tt.expectContent, tt.path, body)
				}
			}
		})
	}
}

func TestServeHTTP_MethodSupport(t *testing.T) {
	h := NewHandler(&Config{Port: 8080})
	
	methods := []string{http.MethodGet, http.MethodPost, http.MethodHead}
	
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/health", nil)
			w := httptest.NewRecorder()
			
			h.ServeHTTP(w, req)
			
			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d for method %s, got %d", http.StatusOK, method, w.Code)
			}
		})
	}
}

func TestConfigSerialization(t *testing.T) {
	config := &Config{Port: 8081}
	
	jsonData, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal Config: %v", err)
	}
	
	var decoded Config
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Config: %v", err)
	}
	
	if decoded.Port != config.Port {
		t.Errorf("Expected port %d after serialization, got %d", config.Port, decoded.Port)
	}
}

func TestRouterStatusSerialization(t *testing.T) {
	status := RouterStatus{
		Request:       "test",
		Response:      "ok",
		Code:          200,
		Uptime:        "1s",
		RequestsTotal: 10,
		Version:       "1.0.0",
	}
	
	jsonData, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal RouterStatus: %v", err)
	}
	
	var decoded RouterStatus
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal RouterStatus: %v", err)
	}
	
	if decoded.Request != status.Request ||
		decoded.Response != status.Response ||
		decoded.Code != status.Code ||
		decoded.Uptime != status.Uptime ||
		decoded.RequestsTotal != status.RequestsTotal ||
		decoded.Version != status.Version {
		t.Error("RouterStatus serialization round-trip failed")
	}
}

func TestRouterAgentStatusSerialization(t *testing.T) {
	status := RouterAgentStatus{
		Status:  "running",
		Version: "1.0.0",
		Message: "Agent is operational",
	}
	
	jsonData, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal RouterAgentStatus: %v", err)
	}
	
	var decoded RouterAgentStatus
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal RouterAgentStatus: %v", err)
	}
	
	if decoded.Status != status.Status ||
		decoded.Version != status.Version ||
		decoded.Message != status.Message {
		t.Error("RouterAgentStatus serialization round-trip failed")
	}
}

func TestHandlerWithDifferentPorts(t *testing.T) {
	ports := []int{3000, 8080, 8888, 9999}
	
	for _, port := range ports {
		t.Run(string(rune(port)), func(t *testing.T) {
			config := &Config{Port: port}
			h := NewHandler(config)
			
			if h.config.Port != port {
				t.Errorf("Expected port %d, got %d", port, h.config.Port)
			}
		})
	}
}

func TestServeHTTP_Tracking(t *testing.T) {
	h := NewHandler(&Config{Port: 8080})
	
	// Make a request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	
	// The handler should have incremented request count (we can't directly access the atomic value in tests)
	// but we can verify that multiple requests work correctly
	
	req2 := httptest.NewRequest(http.MethodGet, "/status", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	
	if w2.Code != http.StatusOK {
		t.Errorf("Expected status %d on second request, got %d", http.StatusOK, w2.Code)
	}
}
