package mcpclient

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestClientCreation tests creating an MCP client with a simple server
func TestClientCreation(t *testing.T) {
	// This is a mock test - in real usage, you'd connect to an actual MCP server
	// For now, we test the basic structures and serialization
	
	t.Run("Tool serializes correctly", func(t *testing.T) {
		tool := Tool{
			Name:        "test_tool",
			Description: "A test tool",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`),
		}
		
		data, err := json.Marshal(tool)
		if err != nil {
			t.Fatalf("Failed to marshal tool: %v", err)
		}
		
		var unmarshaled Tool
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal tool: %v", err)
		}
		
		if unmarshaled.Name != tool.Name {
			t.Errorf("Expected name %s, got %s", tool.Name, unmarshaled.Name)
		}
	})

	t.Run("Request serializes correctly", func(t *testing.T) {
		req := Request{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "initialize",
			Params:  json.RawMessage(`{"protocolVersion":"2024-11-05"}`),
		}
		
		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}
		
		var unmarshaled Request
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal request: %v", err)
		}
		
		if unmarshaled.Method != req.Method {
			t.Errorf("Expected method %s, got %s", req.Method, unmarshaled.Method)
		}
	})

	t.Run("CallToolRequestParams serializes correctly", func(t *testing.T) {
		args := map[string]any{
			"query": "test query",
			"limit": 10,
		}
		
		params := CallToolRequestParams{
			Name:      "search",
			Arguments: args,
		}
		
		data, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal call tool params: %v", err)
		}
		
		var unmarshaled CallToolRequestParams
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal call tool params: %v", err)
		}
		
		if unmarshaled.Name != params.Name {
			t.Errorf("Expected name %s, got %s", params.Name, unmarshaled.Name)
		}
		
		if unmarshaled.Arguments["query"] != args["query"] {
			t.Errorf("Expected query %v, got %v", args["query"], unmarshaled.Arguments["query"])
		}
	})

	t.Run("CallToolResultParams with content", func(t *testing.T) {
		result := CallToolResultParams{
			Content: []Content{
				{
					Type: "text",
					Text: "Hello, world!",
				},
			},
			IsError: false,
		}
		
		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("Failed to marshal call tool result: %v", err)
		}
		
		var unmarshaled CallToolResultParams
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal call tool result: %v", err)
		}
		
		if len(unmarshaled.Content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(unmarshaled.Content))
		}
		
		if unmarshaled.Content[0].Text != "Hello, world!" {
			t.Errorf("Expected text 'Hello, world!', got %s", unmarshaled.Content[0].Text)
		}
	})
}

// TestProtocolVersion tests that the protocol version is set correctly
func TestProtocolVersion(t *testing.T) {
	if ProtocolVersion == "" {
		t.Error("ProtocolVersion should not be empty")
	}
	
	// Verify it's a valid date format or identifier
	if !strings.Contains(ProtocolVersion, "-") {
		t.Logf("Warning: ProtocolVersion %s doesn't look like a date-based version", ProtocolVersion)
	}
}

// TestClientCapabilities tests client capabilities structure
func TestClientCapabilities(t *testing.T) {
	caps := ClientCapabilities{
		Tools:     &struct{ ListChanged bool }{},
		Prompts:   &struct{ ListChanged bool }{},
		Resources: &struct{ ListChanged bool }{},
	}
	
	data, err := json.Marshal(caps)
	if err != nil {
		t.Fatalf("Failed to marshal capabilities: %v", err)
	}
	
	// Should contain all capability types
	if !strings.Contains(string(data), "tools") ||
	   !strings.Contains(string(data), "prompts") ||
	   !strings.Contains(string(data), "resources") {
		t.Error("Capabilities should contain tools, prompts, and resources")
	}
}

// TestServerInformation tests server info structure
func TestServerInformation(t *testing.T) {
	info := ServerInformation{
		Name:    "test-server",
		Version: "1.0.0",
	}
	
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal server info: %v", err)
	}
	
	var unmarshaled ServerInformation
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal server info: %v", err)
	}
	
	if unmarshaled.Name != "test-server" {
		t.Errorf("Expected name 'test-server', got %s", unmarshaled.Name)
	}
	
	if unmarshaled.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", unmarshaled.Version)
	}
}

// TestErrorStructure tests the error structure
func TestErrorStructure(t *testing.T) {
	err := &Error{
		Code:    -32601, // Method not found
		Message: "Method not found",
		Data:    map[string]string{"hint": "check method name"},
	}
	
	data, errMarshal := json.Marshal(err)
	if errMarshal != nil {
		t.Fatalf("Failed to marshal error: %v", errMarshal)
	}
	
	var unmarshaled Error
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal error: %v", err)
	}
	
	if unmarshaled.Code != -32601 {
		t.Errorf("Expected code -32601, got %d", unmarshaled.Code)
	}
	
	if unmarshaled.Message != "Method not found" {
		t.Errorf("Expected message 'Method not found', got %s", unmarshaled.Message)
	}
}

// TestContextTimeout tests that context timeouts work (mock test)
func TestContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	select {
	case <-ctx.Done():
		// Expected
	case <-time.After(200 * time.Millisecond):
		t.Error("Context should have timed out")
	}
}

// TestMustMarshal tests the helper function
func TestMustMarshal(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("mustMarshal panicked: %v", r)
		}
	}()
	
	data := map[string]any{
		"test": "value",
		"number": 42,
	}
	
	result := mustMarshal(data)
	if len(result) == 0 {
		t.Error("mustMarshal should return non-empty data")
	}
}

// TestResourceStructure tests resource structure
func TestResourceStructure(t *testing.T) {
	resource := Resource{
		URI:         "file:///path/to/file.txt",
		Name:        "My File",
		Description: "A test file",
		MimeType:    "text/plain",
	}
	
	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Failed to marshal resource: %v", err)
	}
	
	var unmarshaled Resource
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal resource: %v", err)
	}
	
	if unmarshaled.URI != resource.URI {
		t.Errorf("Expected URI %s, got %s", resource.URI, unmarshaled.URI)
	}
	
	if unmarshaled.MimeType != resource.MimeType {
		t.Errorf("Expected MIME type %s, got %s", resource.MimeType, unmarshaled.MimeType)
	}
}

// TestPromptStructure tests prompt structure
func TestPromptStructure(t *testing.T) {
	prompt := Prompt{
		Name:        "greeting",
		Description: "A greeting template",
		Arguments: []PromptArgument{
			{
				Name:        "name",
				Description: "Name of the person to greet",
				Required:    true,
			},
		},
	}
	
	data, err := json.Marshal(prompt)
	if err != nil {
		t.Fatalf("Failed to marshal prompt: %v", err)
	}
	
	var unmarshaled Prompt
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal prompt: %v", err)
	}
	
	if unmarshaled.Name != prompt.Name {
		t.Errorf("Expected name %s, got %s", prompt.Name, unmarshaled.Name)
	}
	
	if len(unmarshaled.Arguments) != 1 {
		t.Errorf("Expected 1 argument, got %d", len(unmarshaled.Arguments))
	}
}

// TestContentTypes tests different content types
func TestContentTypes(t *testing.T) {
	textContent := Content{
		Type: "text",
		Text: "Hello!",
	}
	
	data, err := json.Marshal(textContent)
	if err != nil {
		t.Fatalf("Failed to marshal text content: %v", err)
	}
	
	var unmarshaled Content
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal text content: %v", err)
	}
	
	if unmarshaled.Type != "text" {
		t.Errorf("Expected type 'text', got %s", unmarshaled.Type)
	}
	
	if unmarshaled.Text != "Hello!" {
		t.Errorf("Expected text 'Hello!', got %s", unmarshaled.Text)
	}
}

// TestInitializeRequestParams tests initialization parameters
func TestInitializeRequestParams(t *testing.T) {
	params := InitializeRequestParams{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ClientCapabilities{
			Tools: &struct{ ListChanged bool }{},
		},
		ClientInfo: ServerInformation{
			Name:    "yolo",
			Version: "1.0.0",
		},
	}
	
	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal init params: %v", err)
	}
	
	var unmarshaled InitializeRequestParams
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal init params: %v", err)
	}
	
	if unmarshaled.ProtocolVersion != ProtocolVersion {
		t.Errorf("Expected protocol version %s, got %s", ProtocolVersion, unmarshaled.ProtocolVersion)
	}
}
