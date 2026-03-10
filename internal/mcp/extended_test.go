package mcp

import (
	"encoding/json"
	"testing"
)

func TestHandleListResources(t *testing.T) {
	server := NewServer("test", "1.0")

	reader := func(ctx *Server) (*ReadResourceResult, error) {
		return &ReadResourceResult{
			Contents: []Content{{URI: "resource://test", Text: "hello"}},
		}, nil
	}

	server.AddResource(Resource{URI: "resource://test", Name: "Test"}, reader)

	req := &Request{
		Method: "resources/list",
		ID:     json.RawMessage(`1`),
		Params: json.RawMessage(`{}`),
	}

	resp := server.handleRequest(req)
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %+v", resp.Error)
	}

	var result ListResourcesResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if len(result.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(result.Resources))
	}
}

func TestHandleReadResource(t *testing.T) {
	server := NewServer("test", "1.0")

	reader := func(ctx *Server) (*ReadResourceResult, error) {
		return &ReadResourceResult{
			Contents: []Content{{URI: "resource://test", Text: "content here"}},
		}, nil
	}

	server.AddResource(Resource{URI: "resource://test", Name: "Test"}, reader)

	params, _ := json.Marshal(ReadResourceRequest{URI: "resource://test"})
	req := &Request{
		Method: "resources/read",
		ID:     json.RawMessage(`1`),
		Params: params,
	}

	resp := server.handleRequest(req)
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %+v", resp.Error)
	}

	var result ReadResourceResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if len(result.Contents) != 1 || result.Contents[0].Text != "content here" {
		t.Errorf("Unexpected result: %+v", result)
	}
}

func TestHandleReadResourceNotFound(t *testing.T) {
	server := NewServer("test", "1.0")

	params, _ := json.Marshal(ReadResourceRequest{URI: "resource://missing"})
	req := &Request{
		Method: "resources/read",
		ID:     json.RawMessage(`1`),
		Params: params,
	}

	resp := server.handleRequest(req)
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	if resp.Error == nil {
		t.Fatal("Expected error for missing resource")
	}
	if resp.Error.Code != -32001 {
		t.Errorf("Expected error code -32001, got %d", resp.Error.Code)
	}
}

func TestHandleGetPrompt(t *testing.T) {
	server := NewServer("test", "1.0")

	handler := func(ctx *Server, args map[string]interface{}) (*GetPromptResult, error) {
		return &GetPromptResult{
			Messages: []Message{{Role: "user", Content: TextContent{Type: "text", Text: "prompt content"}}},
		}, nil
	}

	server.AddPrompt(Prompt{Name: "test_prompt", Description: "A test"}, handler)

	params, _ := json.Marshal(GetPromptRequest{Name: "test_prompt"})
	req := &Request{
		Method: "prompts/get",
		ID:     json.RawMessage(`1`),
		Params: params,
	}

	resp := server.handleRequest(req)
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %+v", resp.Error)
	}

	var result GetPromptResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if len(result.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result.Messages))
	}
}

func TestHandleGetPromptNotFound(t *testing.T) {
	server := NewServer("test", "1.0")

	params, _ := json.Marshal(GetPromptRequest{Name: "missing"})
	req := &Request{
		Method: "prompts/get",
		ID:     json.RawMessage(`1`),
		Params: params,
	}

	resp := server.handleRequest(req)
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	if resp.Error == nil {
		t.Fatal("Expected error for missing prompt")
	}
	if resp.Error.Code != MethodNotFound {
		t.Errorf("Expected MethodNotFound, got %d", resp.Error.Code)
	}
}

func TestHandleListRoots(t *testing.T) {
	server := NewServer("test", "1.0")

	req := &Request{
		Method: "roots/list",
		ID:     json.RawMessage(`1`),
		Params: json.RawMessage(`{}`),
	}

	resp := server.handleRequest(req)
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %+v", resp.Error)
	}

	var result ListRootsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if len(result.Roots) != 0 {
		t.Errorf("Expected empty roots, got %d", len(result.Roots))
	}
}

func TestLogLevelFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", DebugLevel},
		{"info", InfoLevel},
		{"notice", NoticeLevel},
		{"warning", WarningLevel},
		{"error", ErrorLevel},
		{"unknown", DebugLevel}, // defaults to debug
		{"", DebugLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := LogLevelFromString(tt.input)
			if got != tt.expected {
				t.Errorf("LogLevelFromString(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "debug"},
		{LevelInfo, "info"},
		{LevelNotice, "notice"},
		{LevelWarn, "warn"},
		{LevelError, "error"},
		{Level(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.expected {
				t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.expected)
			}
		})
	}
}

func TestLevelMarshalJSON(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, `"debug"`},
		{LevelInfo, `"info"`},
		{LevelError, `"error"`},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			data, err := json.Marshal(tt.level)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("Marshal = %s, want %s", data, tt.expected)
			}
		})
	}
}

func TestLevelUnmarshalJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{`"debug"`, LevelDebug},
		{`"info"`, LevelInfo},
		{`"notice"`, LevelNotice},
		{`"warn"`, LevelWarn},
		{`"warning"`, LevelWarn},
		{`"error"`, LevelError},
		{`"unknown"`, LevelInfo}, // defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var level Level
			if err := json.Unmarshal([]byte(tt.input), &level); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			if level != tt.expected {
				t.Errorf("Unmarshal(%s) = %d, want %d", tt.input, level, tt.expected)
			}
		})
	}
}

func TestLevelUnmarshalJSONInvalid(t *testing.T) {
	var level Level
	err := json.Unmarshal([]byte(`123`), &level)
	if err == nil {
		t.Error("Expected error for non-string JSON")
	}
}

func TestGetSetLogLevel(t *testing.T) {
	server := NewServer("test", "1.0")

	server.SetLogLevel(ErrorLevel)
	if got := server.GetLogLevel(); got != ErrorLevel {
		t.Errorf("GetLogLevel() = %d, want %d", got, ErrorLevel)
	}

	server.SetLogLevel(DebugLevel)
	if got := server.GetLogLevel(); got != DebugLevel {
		t.Errorf("GetLogLevel() = %d, want %d", got, DebugLevel)
	}
}

func TestNewTextContent(t *testing.T) {
	tc := newTextContent("hello")
	if tc.Type != "text" {
		t.Errorf("Expected type 'text', got %q", tc.Type)
	}
	if tc.Text != "hello" {
		t.Errorf("Expected text 'hello', got %q", tc.Text)
	}
}

func TestCreateErrorResponse(t *testing.T) {
	resp := createErrorResponse(json.RawMessage(`42`), -32600, "bad request", nil)
	if resp.ID == nil {
		t.Error("Expected ID in error response")
	}
	if resp.Error == nil {
		t.Fatal("Expected error in response")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("Expected code -32600, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "bad request" {
		t.Errorf("Expected message 'bad request', got %q", resp.Error.Message)
	}
}

func TestCreateSuccessResponse(t *testing.T) {
	result := json.RawMessage(`{"key":"value"}`)
	resp := createSuccessResponse(json.RawMessage(`1`), result)
	if resp.Error != nil {
		t.Error("Expected no error in success response")
	}
	if string(resp.Result) != `{"key":"value"}` {
		t.Errorf("Unexpected result: %s", resp.Result)
	}
}

func TestHandleReadResourceInvalidParams(t *testing.T) {
	server := NewServer("test", "1.0")

	req := &Request{
		Method: "resources/read",
		ID:     json.RawMessage(`1`),
		Params: json.RawMessage(`not valid json`),
	}

	resp := server.handleRequest(req)
	if resp == nil {
		t.Fatal("Expected response")
	}
	if resp.Error == nil {
		t.Fatal("Expected error for invalid params")
	}
	if resp.Error.Code != InvalidParams {
		t.Errorf("Expected InvalidParams, got %d", resp.Error.Code)
	}
}

func TestHandleGetPromptInvalidParams(t *testing.T) {
	server := NewServer("test", "1.0")

	req := &Request{
		Method: "prompts/get",
		ID:     json.RawMessage(`1`),
		Params: json.RawMessage(`bad json`),
	}

	resp := server.handleRequest(req)
	if resp == nil {
		t.Fatal("Expected response")
	}
	if resp.Error == nil {
		t.Fatal("Expected error for invalid params")
	}
	if resp.Error.Code != InvalidParams {
		t.Errorf("Expected InvalidParams, got %d", resp.Error.Code)
	}
}
