package main

import (
	"encoding/json"
	"testing"
)

// TestDeduplicateToolCalls tests the tool call deduplication logic.
func TestDeduplicateToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		input    []ParsedToolCall
		expected int
	}{
		{
			name:     "empty list",
			input:    []ParsedToolCall{},
			expected: 0,
		},
		{
			name: "single call",
			input: []ParsedToolCall{
				{Name: "test_func", Args: map[string]any{"arg": "value"}},
			},
			expected: 1,
		},
		{
			name: "no duplicates",
			input: []ParsedToolCall{
				{Name: "func_a", Args: map[string]any{"x": 1}},
				{Name: "func_b", Args: map[string]any{"y": 2}},
			},
			expected: 2,
		},
		{
			name: "exact duplicates removed",
			input: []ParsedToolCall{
				{Name: "same_func", Args: map[string]any{"arg": "value"}},
				{Name: "same_func", Args: map[string]any{"arg": "value"}},
				{Name: "same_func", Args: map[string]any{"arg": "value"}},
			},
			expected: 1,
		},
		{
			name: "different args not deduplicated",
			input: []ParsedToolCall{
				{Name: "file_edit", Args: map[string]any{"path": "a.txt", "content": "old"}},
				{Name: "file_edit", Args: map[string]any{"path": "a.txt", "content": "new"}},
			},
			expected: 2,
		},
		{
			name: "interleaved duplicates",
			input: []ParsedToolCall{
				{Name: "func_x", Args: map[string]any{"n": 1}},
				{Name: "func_y", Args: map[string]any{"m": 2}},
				{Name: "func_x", Args: map[string]any{"n": 1}},
				{Name: "func_z", Args: map[string]any{"p": 3}},
				{Name: "func_y", Args: map[string]any{"m": 2}},
			},
			expected: 3, // func_x, func_y, func_z (only first occurrence kept)
		},
		{
			name: "preserves order of first occurrences",
			input: []ParsedToolCall{
				{Name: "first", Args: map[string]any{}},
				{Name: "second", Args: map[string]any{}},
				{Name: "third", Args: map[string]any{}},
				{Name: "first", Args: map[string]any{}},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateToolCalls(tt.input)
			if len(result) != tt.expected {
				t.Errorf("deduplicateToolCalls() returned %d unique calls, want %d", len(result), tt.expected)
			}

			// Verify no duplicates in output
			seen := make(map[string]bool)
			for _, tc := range result {
				argsJSON, _ := json.Marshal(tc.Args)
				key := tc.Name + "|" + string(argsJSON)
				if seen[key] {
					t.Errorf("duplicate found in output: %s with args %v", tc.Name, tc.Args)
				}
				seen[key] = true
			}

			// For the "preserves order" test, verify first items are kept
			if tt.name == "preserves order of first occurrences" {
				if len(result) >= 3 && (result[0].Name != "first" || result[1].Name != "second" || result[2].Name != "third") {
					t.Errorf("order not preserved: got %v, want [first, second, third]",
						[]string{result[0].Name, result[1].Name, result[2].Name})
				}
			}
		})
	}
}

// TestToolDef tests the tool definition builder.
func TestToolDef(t *testing.T) {
	tests := []struct {
		name       string
		funcName   string
		desc       string
		props      map[string]ToolParam
		required   []string
		wantType   string
		wantFunc   string
		wantParams int
	}{
		{
			name:     "minimal tool def",
			funcName: "echo",
			desc:     "Echo a message",
			props:    map[string]ToolParam{},
			required: nil,
			wantType: "function",
			wantFunc: "echo",
		},
		{
			name:     "tool with required params",
			funcName: "read_file",
			desc:     "Read a file's contents",
			props: map[string]ToolParam{
				"path":  {Type: "string", Description: "Path to file"},
				"limit": {Type: "integer", Description: "Max lines"},
			},
			required:   []string{"path"},
			wantType:   "function",
			wantFunc:   "read_file",
			wantParams: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toolDef(tt.funcName, tt.desc, tt.props, tt.required)

			if result.Type != "function" {
				t.Errorf("toolDef().Type = %q, want \"function\"", result.Type)
			}
			if result.Function.Name != tt.funcName {
				t.Errorf("toolDef().Function.Name = %q, want %q", result.Function.Name, tt.funcName)
			}
			if result.Function.Description != tt.desc {
				t.Errorf("toolDef().Description = %q, want %q", result.Function.Description, tt.desc)
			}
			if len(result.Function.Parameters.Properties) != len(tt.props) {
				t.Errorf("toolDef().Parameters.Properties count = %d, want %d",
					len(result.Function.Parameters.Properties), len(tt.props))
			}
			if len(result.Function.Parameters.Required) != len(tt.required) {
				t.Errorf("toolDef().Required count = %d, want %d",
					len(result.Function.Parameters.Required), len(tt.required))
			}
		})
	}
}

// TestOllamaModelJSON tests model JSON marshaling/unmarshaling.
func TestOllamaModelJSON(t *testing.T) {
	jsonStr := `{"name":"llama3.2:1b"}`
	var model OllamaModel
	if err := json.Unmarshal([]byte(jsonStr), &model); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if model.Name != "llama3.2:1b" {
		t.Errorf("model.Name = %q, want %q", model.Name, "llama3.2:1b")
	}

	// Test marshaling back
	data, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	var roundtrip OllamaModel
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("roundtrip unmarshal failed: %v", err)
	}
	if roundtrip.Name != model.Name {
		t.Errorf("roundtrip.Name = %q, want %q", roundtrip.Name, model.Name)
	}
}

// TestOllamaTagsResponseJSON tests tags response parsing.
func TestOllamaTagsResponseJSON(t *testing.T) {
	jsonStr := `{
		"models": [
			{"name": "llama3.2:1b"},
			{"name": "mistral:latest"},
			{"name": "phi3:mini"}
		]
	}`
	var response OllamaTagsResponse
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if len(response.Models) != 3 {
		t.Errorf("len(Models) = %d, want 3", len(response.Models))
	}

	expectedNames := []string{"llama3.2:1b", "mistral:latest", "phi3:mini"}
	for i, expected := range expectedNames {
		if response.Models[i].Name != expected {
			t.Errorf("Models[%d].Name = %q, want %q", i, response.Models[i].Name, expected)
		}
	}
}

// TestChatMessageJSON tests chat message JSON handling.
func TestChatMessageJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		role    string
		content string
	}{
		{
			name:    "user message",
			input:   `{"role":"user","content":"hello"}`,
			role:    "user",
			content: "hello",
		},
		{
			name: "assistant message with tool call",
			input: `{
				"role": "assistant",
				"content": "",
				"tool_calls": [
					{"id": "call_123", "function": {"name": "test_tool", "arguments": "{}"}}
				]
			}`,
			role: "assistant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg ChatMessage
			if err := json.Unmarshal([]byte(tt.input), &msg); err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}
			if msg.Role != tt.role {
				t.Errorf("msg.Role = %q, want %q", msg.Role, tt.role)
			}
			if tt.content != "" && msg.Content != tt.content {
				t.Errorf("msg.Content = %q, want %q", msg.Content, tt.content)
			}
		})
	}
}

// TestToolCallJSON tests tool call JSON parsing.
func TestToolCallJSON(t *testing.T) {
	// Note: In the Ollama API response, arguments is a JSON-encoded string (escaped).
	// The RawMessage will contain the escaped representation as returned by Ollama.
	jsonStr := `{
		"id": "call_abc123",
		"function": {
			"name": "read_file",
			"arguments": "{\"path\": \"test.txt\", \"limit\": 10}"
		}
	}`
	var call ToolCall
	if err := json.Unmarshal([]byte(jsonStr), &call); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if call.ID != "call_abc123" {
		t.Errorf("call.ID = %q, want %q", call.ID, "call_abc123")
	}
	if call.Function.Name != "read_file" {
		t.Errorf("call.Function.Name = %q, want %q", call.Function.Name, "read_file")
	}

	// Verify arguments are present (it's a JSON string containing the args)
	if len(call.Function.Arguments) == 0 {
		t.Error("call.Function.Arguments should not be empty")
	}

	// The RawMessage contains a JSON-encoded string, so we need to unmarshal twice
	var argsJSONStr string
	if err := json.Unmarshal(call.Function.Arguments, &argsJSONStr); err != nil {
		t.Errorf("failed to unmarshal arguments as string: %v", err)
		return
	}

	// Now unmarshal the actual JSON object
	var argsMap map[string]any
	if err := json.Unmarshal([]byte(argsJSONStr), &argsMap); err != nil {
		t.Errorf("failed to parse tool call arguments JSON: %v", err)
	}

	if path, ok := argsMap["path"].(string); !ok || path != "test.txt" {
		t.Errorf("args.path = %v, want \"test.txt\"", argsMap["path"])
	}
	if limit, ok := argsMap["limit"].(float64); !ok || limit != 10 {
		t.Errorf("args.limit = %v, want 10", argsMap["limit"])
	}
}
