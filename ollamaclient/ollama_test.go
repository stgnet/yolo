package ollamaclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewOllamaClient(t *testing.T) {
	client := NewOllamaClient("http://test-server")
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.baseURL != "http://test-server" {
		t.Errorf("baseURL is %q, want http://test-server", client.baseURL)
	}
	if client.client == nil {
		t.Error("Expected client.HTTP to be initialized")
	}
	if len(client.ctxCache) != 0 {
		t.Error("Expected empty ctxCache")
	}
}

func TestNewOllamaClientTrimsTrailingSlash(t *testing.T) {
	client := NewOllamaClient("http://test-server/")
	if client.baseURL != "http://test-server" {
		t.Errorf("baseURL is %q, want http://test-server (trailing slash should be trimmed)", client.baseURL)
	}
}

func TestOllamaClientListModels(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		expectedLen int
	}{
		{
			name: "successful list",
			response: `{
				"models": [
					{"name": "llama2:latest"},
					{"name": "mistral:7b"}
				]
			}`,
			expectedLen: 2,
		},
		{
			name:        "empty list",
			response:    `{"models": []}`,
			expectedLen: 0,
		},
		{
			name:        "no models field",
			response:    `{}`,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/tags" {
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewOllamaClient(server.URL)
			models := client.ListModels()

			if len(models) != tt.expectedLen {
				t.Errorf("expected %d models, got %d", tt.expectedLen, len(models))
			}

			if tt.expectedLen > 0 {
				expectedNames := []string{"llama2:latest", "mistral:7b"}
				for i, m := range expectedNames {
					if models[i] != m {
						t.Errorf("models[%d] = %q, want %q", i, models[i], m)
					}
				}
			}
		})
	}
}

func TestOllamaClientListModelsServerDown(t *testing.T) {
	client := NewOllamaClient("http://localhost:59999")
	models := client.ListModels()
	if models != nil {
		t.Errorf("Expected nil for unreachable server, got %v", models)
	}
}

func TestOllamaClientListModelsMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL)
	models := client.ListModels()
	if models != nil {
		t.Errorf("Expected nil for malformed JSON, got %v", models)
	}
}

func TestOllamaClientGetModelContextLength(t *testing.T) {
	tests := []struct {
		name           string
		model          string
		response       string
		expectedLength int
	}{
		{
			name:  "context length found",
			model: "llama2:latest",
			response: `{
				"model_info": {
					"general.parameter_count": 13607925504,
					"general.architecture.context_length": 4096
				}
			}`,
			expectedLength: 4096,
		},
		{
			name:  "context length not found",
			model: "test-model",
			response: `{
				"model_info": {
					"general.parameter_count": 7238014361
				}
			}`,
			expectedLength: 0,
		},
		{
			name:           "empty response",
			model:          "test-model",
			response:       `{}`,
			expectedLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/show" {
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewOllamaClient(server.URL)
			length := client.GetModelContextLength(tt.model)

			if length != tt.expectedLength {
				t.Errorf("GetModelContextLength(%q) = %d, want %d", tt.model, length, tt.expectedLength)
			}
		})
	}
}

func TestOllamaClientGetModelContextLengthCaching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"model_info": {"general.architecture.context_length": 8192}}`))
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL)
	
	// First call should hit the server
	length1 := client.GetModelContextLength("test-model")
	if length1 != 8192 {
		t.Errorf("First call: expected 8192, got %d", length1)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 server call after first GetModelContextLength, got %d", callCount)
	}

	// Second call with same model should still hit server (no caching implemented yet)
	length2 := client.GetModelContextLength("test-model")
	if length2 != 8192 {
		t.Errorf("Second call: expected 8192, got %d", length2)
	}
}

func TestOllamaClientContextCacheMethods(t *testing.T) {
	client := NewOllamaClient("http://test")
	
	model := "test-model"
	length := 4096
	
	// Set cache
	client.setContextCache(model, length)
	
	// Get cache
	retrieved, ok := client.getContextCache(model)
	if !ok {
		t.Error("Expected model to be in cache")
	}
	if retrieved != length {
		t.Errorf("Cached value = %d, want %d", retrieved, length)
	}
	
	// Get and delete cache
	retrieved2, ok := client.getAndDeleteContextCache(model)
	if !ok {
		t.Error("Expected model to be in cache before deletion")
	}
	if retrieved2 != length {
		t.Errorf("Cached value = %d, want %d", retrieved2, length)
	}
	
	// Verify deleted
	_, ok = client.getContextCache(model)
	if ok {
		t.Error("Expected model to be deleted from cache")
	}
	
	// Delete non-existent model (should not panic)
	client.deleteContextCache("non-existent-model")
	
	// Get non-existent model
	_, ok = client.getContextCache("non-existent-model")
	if ok {
		t.Error("Expected non-existent model to return false")
	}
}

func TestToolDefWith(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]ToolParam
		required    []string
		wantName    string
		wantDesc    string
		wantType    string
	}{
		{
			name: "basic tool def",
			params: map[string]ToolParam{
				"query": {Type: "string", Description: "Search query"},
			},
			required: []string{"query"},
			wantName: "search_web",
			wantDesc: "Search the web",
			wantType: "function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToolDefWith(tt.wantName, tt.wantDesc, tt.params, tt.required)
			
			if result.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", result.Type, tt.wantType)
			}
			if result.Function.Name != tt.wantName {
				t.Errorf("Function.Name = %q, want %q", result.Function.Name, tt.wantName)
			}
			if result.Function.Description != tt.wantDesc {
				t.Errorf("Function.Description = %q, want %q", result.Function.Description, tt.wantDesc)
			}
			if len(result.Function.Parameters.Properties) != len(tt.params) {
				t.Errorf("Parameters.Properties length = %d, want %d", 
					len(result.Function.Parameters.Properties), len(tt.params))
			}
		})
	}
}

func TestChatRequestWithTools(t *testing.T) {
	req := ChatRequest{
		Model:  "llama2:latest",
		Messages: []ChatMessage{
			{Role: "user", Content: "What's the weather?"},
		},
		Stream: true,
		Tools: []ToolDef{
			ToolDefWith("get_weather", "Get weather", nil, nil),
		},
	}

	if req.Model != "llama2:latest" {
		t.Errorf("Model = %q, want llama2:latest", req.Model)
	}
	if len(req.Messages) != 1 {
		t.Errorf("Messages length = %d, want 1", len(req.Messages))
	}
	if !req.Stream {
		t.Error("Stream should be true")
	}
	if len(req.Tools) != 1 {
		t.Errorf("Tools length = %d, want 1", len(req.Tools))
	}
}

func TestChatMessageWithToolCalls(t *testing.T) {
	msg := ChatMessage{
		Role:    "assistant",
		Content: "I'll check the weather for you.",
		ToolCalls: []ToolCall{
			{
				ID: "call_123",
				Function: ToolCallFunc{
					Name:      "get_weather",
					Arguments: json.RawMessage(`{"city": "New York"}`),
				},
			},
		},
	}

	if len(msg.ToolCalls) != 1 {
		t.Errorf("ToolCalls length = %d, want 1", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].ID != "call_123" {
		t.Errorf("ToolCall ID = %q, want call_123", msg.ToolCalls[0].ID)
	}
	if msg.ToolCalls[0].Function.Name != "get_weather" {
		t.Errorf("ToolCall Function.Name = %q, want get_weather", msg.ToolCalls[0].Function.Name)
	}
}

func BenchmarkListModels(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"models": [{"name": "model1"}, {"name": "model2"}]}`))
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.ListModels()
	}
}

func BenchmarkGetModelContextLength(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"model_info": {"general.architecture.context_length": 4096}}`))
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.GetModelContextLength("test-model")
	}
}
