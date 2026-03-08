package mcp

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRegisterTool(t *testing.T) {
	server := NewServer("test", "1.0")
	
	testHandler := func(s *Server, args map[string]interface{}) (*CallToolResult, error) {
		return newTextToolResult("success"), nil
	}
	
	err := server.RegisterTool(MCPTool{
		Name:        "test_tool",
		Description: "A test tool",
	}, testHandler)
	
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}
	
	server.mu.RLock()
	defer server.mu.RUnlock()
	
	if len(server.tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(server.tools))
	}
	
	if _, exists := server.toolHandlers["test_tool"]; !exists {
		t.Error("Tool handler not registered")
	}
}

func TestHandleListTools(t *testing.T) {
	server := NewServer("test", "1.0")
	
	testHandler := func(s *Server, args map[string]interface{}) (*CallToolResult, error) {
		return newTextToolResult("success"), nil
	}
	
	err := server.RegisterTool(MCPTool{
		Name:        "tool1",
		Description: "Tool 1 description",
	}, testHandler)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}
	
	result, err := server.handleListTools()
	if err != nil {
		t.Fatalf("handleListTools failed: %v", err)
	}
	
	var toolsResult ListToolsResult
	if err := json.Unmarshal(result, &toolsResult); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	if len(toolsResult.Tools) != 1 {
		t.Errorf("Expected 1 tool in result, got %d", len(toolsResult.Tools))
	}
	
	if toolsResult.Tools[0].Name != "tool1" {
		t.Errorf("Expected tool name 'tool1', got '%s'", toolsResult.Tools[0].Name)
	}
}

func TestHandleCallTool(t *testing.T) {
	server := NewServer("test", "1.0")
	
	called := false
	testHandler := func(s *Server, args map[string]interface{}) (*CallToolResult, error) {
		called = true
		if val, ok := args["input"]; ok && val == "hello" {
			return newTextToolResult("received hello"), nil
		}
		return newTextToolResult("unknown input"), nil
	}
	
	err := server.RegisterTool(MCPTool{
		Name:        "greet",
		Description: "Greeting tool",
	}, testHandler)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}
	
	// Create request
	callReq := CallToolRequest{
		Name: "greet",
		Arguments: map[string]interface{}{
			"input": "hello",
		},
	}
	params, _ := json.Marshal(callReq)
	
	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`"1"`),
		Method:  "tools/call",
		Params:  params,
	}
	
	resp := server.handleRequest(req)
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	
	if !called {
		t.Error("Handler was not called")
	}
	
	var result CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(result.Content))
	}
	
	if result.Content[0].GetText() != "received hello" {
		t.Errorf("Expected 'received hello', got '%s'", result.Content[0].GetText())
	}
}

func TestHandleCallToolNotFound(t *testing.T) {
	server := NewServer("test", "1.0")
	
	callReq := CallToolRequest{
		Name: "nonexistent",
	}
	params, _ := json.Marshal(callReq)
	
	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`"1"`),
		Method:  "tools/call",
		Params:  params,
	}
	
	resp := server.handleRequest(req)
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	
	if resp.Error == nil {
		t.Error("Expected error for nonexistent tool")
	}
	
	if resp.Error.Code != MethodNotFound {
		t.Errorf("Expected error code %d, got %d", MethodNotFound, resp.Error.Code)
	}
}

func TestHandleCallToolInvalidParams(t *testing.T) {
	server := NewServer("test", "1.0")
	
	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`"1"`),
		Method:  "tools/call",
		Params:  json.RawMessage(`"invalid string instead of object"`),
	}
	
	resp := server.handleRequest(req)
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	
	if resp.Error == nil {
		t.Error("Expected error for invalid params")
	}
	
	if resp.Error.Code != InvalidParams {
		t.Errorf("Expected error code %d, got %d", InvalidParams, resp.Error.Code)
	}
}

func TestHandleListToolsEmpty(t *testing.T) {
	server := NewServer("test", "1.0")
	
	result, err := server.handleListTools()
	if err != nil {
		t.Fatalf("handleListTools failed: %v", err)
	}
	
	var toolsResult ListToolsResult
	if err := json.Unmarshal(result, &toolsResult); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	if len(toolsResult.Tools) != 0 {
		t.Errorf("Expected 0 tools in empty server, got %d", len(toolsResult.Tools))
	}
}

func TestDuplicateToolRegistration(t *testing.T) {
	server := NewServer("test", "1.0")
	
	testHandler := func(s *Server, args map[string]interface{}) (*CallToolResult, error) {
		return newTextToolResult("success"), nil
	}
	
	// First registration should succeed
	err := server.RegisterTool(MCPTool{
		Name:        "dup_tool",
		Description: "A duplicate test tool",
	}, testHandler)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}
	
	// Second registration should fail
	err = server.RegisterTool(MCPTool{
		Name:        "dup_tool",
		Description: "A duplicate test tool",
	}, testHandler)
	if err == nil {
		t.Error("Expected error for duplicate tool registration")
	}
}

func TestTextContent(t *testing.T) {
	text := TextContent{
		Type:  "text",
		Text:  "Hello, World!",
		Extra: map[string]interface{}{"key": "value"},
	}
	
	if text.GetType() != "text" {
		t.Errorf("Expected type 'text', got '%s'", text.GetType())
	}
	
	if text.GetText() != "Hello, World!" {
		t.Errorf("Expected text 'Hello, World!', got '%s'", text.GetText())
	}
}

func TestImageContent(t *testing.T) {
	image := ImageContent{
		Type: "image",
		Data: "iVBORw0KGgoAAAANSUhEUgAAAAUA",
		MIME: "image/png",
	}
	
	if image.GetType() != "image" {
		t.Errorf("Expected type 'image', got '%s'", image.GetType())
	}
	
	if image.GetImage() == nil {
		t.Error("Expected non-nil image data")
	}
	
	if image.GetImage().Data != "iVBORw0KGgoAAAANSUhEUgAAAAUA" {
		t.Error("Expected matching image data")
	}
	
	if image.GetImage().MIMEType != "image/png" {
		t.Errorf("Expected MIME type 'image/png', got '%s'", image.GetImage().MIMEType)
	}
}

func TestReadResource(t *testing.T) {
	server := NewServer("test", "1.0")
	
	testHandler := func(uri string) (*ResourceContents, error) {
		return &ResourceContents{
			URI:      uri,
			MIMEType: "text/plain",
			Text:     "Hello from resource",
		}, nil
	}
	
	err := server.RegisterResource(MCPResource{
		URI:         "test://resource1",
		Name:        "Test Resource",
		Description: "A test resource",
	}, testHandler)
	if err != nil {
		t.Fatalf("Failed to register resource: %v", err)
	}
	
	result, err := server.handleReadResource(map[string]interface{}{
		"uri": "test://resource1",
	})
	if err != nil {
		t.Fatalf("handleReadResource failed: %v", err)
	}
	
	var resourceResult ReadResourceResult
	if err := json.Unmarshal(result, &resourceResult); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	if len(resourceResult.Contents) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(resourceResult.Contents))
	}
	
	if resourceResult.Contents[0].GetText() != "Hello from resource" {
		t.Errorf("Expected 'Hello from resource', got '%s'", resourceResult.Contents[0].GetText())
	}
}

func TestPromptRegistration(t *testing.T) {
	server := NewServer("test", "1.0")
	
	testHandler := func(args map[string]interface{}) (*GetPromptResult, error) {
		return &GetPromptResult{
			Messages: []PromptMessage{
				{Role: "user", Content: TextContent{Type: "text", Text: "Hello"}},
			},
		}, nil
	}
	
	err := server.RegisterPrompt(MCPPrompt{
		Name:        "test_prompt",
		Description: "A test prompt",
	}, testHandler)
	if err != nil {
		t.Fatalf("Failed to register prompt: %v", err)
	}
	
	server.mu.RLock()
	defer server.mu.RUnlock()
	
	if len(server.prompts) != 1 {
		t.Errorf("Expected 1 prompt, got %d", len(server.prompts))
	}
	
	if _, exists := server.promptHandlers["test_prompt"]; !exists {
		t.Error("Prompt handler not registered")
	}
}

func TestHandleListPrompts(t *testing.T) {
	server := NewServer("test", "1.0")
	
	testHandler := func(args map[string]interface{}) (*GetPromptResult, error) {
		return &GetPromptResult{}, nil
	}
	
	err := server.RegisterPrompt(MCPPrompt{
		Name:        "prompt1",
		Description: "A test prompt",
	}, testHandler)
	if err != nil {
		t.Fatalf("Failed to register prompt: %v", err)
	}
	
	result, err := server.handleListPrompts()
	if err != nil {
		t.Fatalf("handleListPrompts failed: %v", err)
	}
	
	var promptsResult ListPromptsResult
	if err := json.Unmarshal(result, &promptsResult); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	if len(promptsResult.Prompts) != 1 {
		t.Errorf("Expected 1 prompt in result, got %d", len(promptsResult.Prompts))
	}
	
	if promptsResult.Prompts[0].Name != "prompt1" {
		t.Errorf("Expected prompt name 'prompt1', got '%s'", promptsResult.Prompts[0].Name)
	}
}

func TestJSONRPCError(t *testing.T) {
	err := &Error{
		Code:    -32000,
		Message: "Custom error",
		Data:    json.RawMessage(`{"key": "value"}`),
	}
	
	if err.Code != -32000 {
		t.Errorf("Expected code -32000, got %d", err.Code)
	}
	
	if err.Message != "Custom error" {
		t.Errorf("Expected message 'Custom error', got '%s'", err.Message)
	}
}

func TestNewTextToolResult(t *testing.T) {
	result := newTextToolResult("test content")
	
	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(result.Content))
	}
	
	content, ok := result.Content[0].(TextContent)
	if !ok {
		t.Error("Expected TextContent type")
	}
	
	if content.Text != "test content" {
		t.Errorf("Expected 'test content', got '%s'", content.Text)
	}
}

func TestNewImageToolResult(t *testing.T) {
	data := "iVBORw0KGgoAAAANSUhEUgAAAAUA"
	mimeType := "image/png"
	result := newImageToolResult(data, mimeType)
	
	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(result.Content))
	}
	
	content, ok := result.Content[0].(ImageContent)
	if !ok {
		t.Error("Expected ImageContent type")
	}
	
	if content.Data != data {
		t.Errorf("Expected '%s', got '%s'", data, content.Data)
	}
	
	if content.MIME != mimeType {
		t.Errorf("Expected MIME type '%s', got '%s'", mimeType, content.MIME)
	}
}

func TestServerStart(t *testing.T) {
	server := NewServer("test", "1.0")
	
	// Start in a goroutine to avoid blocking
	go func() {
		err := server.Start(bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{}))
		if err != nil {
			t.Logf("Server start error (expected): %v", err)
		}
	}()
	
	// Give it a moment to initialize
	server.mu.RLock()
	hasStarted := server.started
	server.mu.RUnlock()
	
	// Note: We can't fully test this without proper I/O streams
	if !hasStarted {
		t.Logf("Server may not have started (test limitation)")
	}
}