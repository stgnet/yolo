package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestServerNew(t *testing.T) {
	server := NewServer("test-server", "1.0.0")
	if server == nil {
		t.Fatal("Expected non-nil server")
	}
	
	if server.name != "test-server" {
		t.Errorf("Expected name 'test-server', got '%s'", server.name)
	}
	
	if server.version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", server.version)
	}
}

func TestServerAddTool(t *testing.T) {
	server := NewServer("test", "1.0")
	
	handler := func(ctx *Server, args map[string]interface{}) (*CallToolResult, error) {
		return &CallToolResult{}, nil
	}
	
	server.AddTool(Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{},
	}, handler)
	
	if len(server.tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(server.tools))
	}
	
	if server.tools[0].Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", server.tools[0].Name)
	}
	
	if _, exists := server.toolHandlers["test_tool"]; !exists {
		t.Error("Expected tool handler to be registered")
	}
}

func TestServerHandleListTools(t *testing.T) {
	server := NewServer("test", "1.0")
	
	server.AddTool(Tool{
		Name:        "tool1",
		Description: "First tool",
		InputSchema: map[string]interface{}{},
	}, nil)
	
	server.AddTool(Tool{
		Name:        "tool2",
		Description: "Second tool",
		InputSchema: map[string]interface{}{},
	}, nil)
	
	req := &Request{
		Method: "tools/list",
		ID:     json.RawMessage(`1`),
		Params: json.RawMessage(`{}`),
	}
	
	resp := server.handleRequest(req)
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatal("Expected 'tools' field in result")
	}
	
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}
}

func TestServerHandleInitialize(t *testing.T) {
	server := NewServer("test-server", "1.0.0")
	
	initParams := map[string]interface{}{
		"protocolVersion": ProtocolVersion,
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "test-client",
			"version": "1.0",
		},
	}
	
	paramsJSON, _ := json.Marshal(initParams)
	
	req := &Request{
		Method: "initialize",
		ID:     json.RawMessage(`1`),
		Params: paramsJSON,
	}
	
	resp := server.handleRequest(req)
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	if result["protocolVersion"] != ProtocolVersion {
		t.Errorf("Expected protocol version '%s', got '%v'", ProtocolVersion, result["protocolVersion"])
	}
	
	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected serverInfo in response")
	}
	
	if serverInfo["name"] != "test-server" {
		t.Errorf("Expected server name 'test-server', got '%v'", serverInfo["name"])
	}
	
	if serverInfo["version"] != "1.0.0" {
		t.Errorf("Expected server version '1.0.0', got '%v'", serverInfo["version"])
	}
}

func TestServerHandlePing(t *testing.T) {
	server := NewServer("test", "1.0")
	
	req := &Request{
		Method: "ping",
		ID:     json.RawMessage(`1`),
		Params: json.RawMessage(`{}`),
	}
	
	resp := server.handleRequest(req)
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	
	if resp.Result != nil {
		t.Error("Expected nil result for ping")
	}
}

func TestServerHandleSetLevel(t *testing.T) {
	server := NewServer("test", "1.0")
	
	setLevelParams := map[string]interface{}{
		"level": LevelDebug,
	}
	
	paramsJSON, _ := json.Marshal(setLevelParams)
	
	req := &Request{
		Method: "logging/setLevel",
		ID:     json.RawMessage(`1`),
		Params: paramsJSON,
	}
	
	resp := server.handleRequest(req)
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	
	if resp.Result != nil {
		t.Error("Expected nil result for setLevel")
	}
}

func TestServerHandleUnknownMethod(t *testing.T) {
	server := NewServer("test", "1.0")
	
	req := &Request{
		Method: "unknown/method",
		ID:     json.RawMessage(`1`),
		Params: json.RawMessage(`{}`),
	}
	
	resp := server.handleRequest(req)
	if resp == nil {
		t.Fatal("Expected non-nil response (error)")
	}
	
	if resp.Error == nil {
		t.Error("Expected error for unknown method")
	}
	
	if resp.Error.Code != MethodNotFound {
		t.Errorf("Expected error code %d, got %d", MethodNotFound, resp.Error.Code)
	}
	
	if !strings.Contains(resp.Error.Message, "Method not found") {
		t.Errorf("Expected 'Method not found' message, got '%s'", resp.Error.Message)
	}
}

func TestServerHandleNotifications(t *testing.T) {
	server := NewServer("test", "1.0")
	
	req := &Request{
		Method: "initialized",
		ID:     nil,
		Params: json.RawMessage(`{}`),
	}
	
	resp := server.handleRequest(req)
	if resp != nil {
		t.Error("Expected nil response for notification")
	}
}

func TestServerAddResource(t *testing.T) {
	server := NewServer("test", "1.0")
	
	reader := func(ctx *Server) (*ReadResourceResult, error) {
		return &ReadResourceResult{}, nil
	}
	
	server.AddResource(Resource{
		URI: "resource://test",
	}, reader)
	
	if len(server.resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(server.resources))
	}
	
	if _, exists := server.resourceReaders["resource://test"]; !exists {
		t.Error("Expected resource reader to be registered")
	}
}

func TestServerAddPrompt(t *testing.T) {
	server := NewServer("test", "1.0")
	
	handler := func(ctx *Server, args map[string]interface{}) (*GetPromptResult, error) {
		return &GetPromptResult{}, nil
	}
	
	server.AddPrompt(Prompt{
		Name:        "test_prompt",
		Description: "A test prompt",
	}, handler)
	
	if len(server.prompts) != 1 {
		t.Errorf("Expected 1 prompt, got %d", len(server.prompts))
	}
	
	if _, exists := server.promptHandlers["test_prompt"]; !exists {
		t.Error("Expected prompt handler to be registered")
	}
}

func TestServerConcurrentToolAdd(t *testing.T) {
	server := NewServer("test", "1.0")
	
	done := make(chan bool)
	
	for i := 0; i < 10; i++ {
		go func(idx int) {
			name := "tool_" + string(rune('a'+idx))
			handler := func(ctx *Server, args map[string]interface{}) (*CallToolResult, error) {
				return &CallToolResult{}, nil
			}
			server.AddTool(Tool{
				Name:        name,
				Description: "Tool " + name,
				InputSchema: map[string]interface{}{},
			}, handler)
			done <- true
		}(i)
	}
	
	for i := 0; i < 10; i++ {
		<-done
	}
	
	if len(server.tools) != 10 {
		t.Errorf("Expected 10 tools, got %d", len(server.tools))
	}
}
