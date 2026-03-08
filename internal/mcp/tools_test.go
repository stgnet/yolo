package mcp

import (
	"encoding/json"
	"testing"
)

func TestRegisterTool(t *testing.T) {
	server := NewServer("test", "1.0")

	testHandler := func(s *Server, args map[string]interface{}) (*CallToolResult, error) {
		result := newTextToolResult("success")
		return &result, nil
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
		result := newTextToolResult("success")
		return &result, nil
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
			result := newTextToolResult("received hello")
			return &result, nil
		}
		result := newTextToolResult("unknown input")
		return &result, nil
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

	// Unmarshal the result and check the content
	var resultMap map[string]interface{}
	if err := json.Unmarshal(resp.Result, &resultMap); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	content, ok := resultMap["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatal("Expected content array in result")
	}

	firstContent, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatal("Expected content item to be a map")
	}

	if firstContent["text"] != "received hello" {
		t.Errorf("Expected 'received hello', got '%v'", firstContent["text"])
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
		result := newTextToolResult("success")
		return &result, nil
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
		Type: "text",
		Text: "Hello, World!",
	}

	if text.Type != "text" {
		t.Errorf("Expected type 'text', got '%s'", text.Type)
	}

	if text.Text != "Hello, World!" {
		t.Errorf("Expected text 'Hello, World!', got '%s'", text.Text)
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

func TestNewErrorToolResult(t *testing.T) {
	result := newErrorToolResult("something went wrong")

	if !result.IsError {
		t.Error("Expected IsError to be true")
	}

	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(result.Content))
	}

	content, ok := result.Content[0].(TextContent)
	if !ok {
		t.Error("Expected TextContent type")
	}

	if content.Text != "Error: something went wrong" {
		t.Errorf("Expected 'Error: something went wrong', got '%s'", content.Text)
	}
}

func TestHandleListPrompts(t *testing.T) {
	server := NewServer("test", "1.0")

	testHandler := func(s *Server, args map[string]interface{}) (*GetPromptResult, error) {
		return &GetPromptResult{}, nil
	}

	err := server.RegisterPrompt(Prompt{
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

func TestFormatArguments(t *testing.T) {
	args := map[string]interface{}{
		"name": "test",
	}

	result := formatArguments(args)
	if result == "" {
		t.Error("Expected non-empty formatted arguments")
	}
}
