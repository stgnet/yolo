package main

import (
	"reflect"
	"testing"
)

// TestToolActivityFormatParsing tests the actual parsing of [tool activity] blocks
func TestToolActivityFormatParsing(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []ParsedToolCall
	}{
		{
			name: "simple tool call",
			content: `[tool activity]
[read_file]`,
			expected: []ParsedToolCall{
				{Name: "read_file", Args: map[string]any{}},
			},
		},
		{
			name: "tool call with parameter description",
			content: `[tool activity]
[read_file] => Read a file's contents`,
			expected: []ParsedToolCall{
				{Name: "read_file", Args: map[string]any{}},
			},
		},
		{
			name: "multiple tool calls",
			content: `[tool activity]
[read_file]
[write_file] => Create or overwrite a file`,
			expected: []ParsedToolCall{
				{Name: "read_file", Args: map[string]any{}},
				{Name: "write_file", Args: map[string]any{}},
			},
		},
		{
			name: "tool call with parentheses",
			content: `[tool activity]
[spawn_subagent()] => Spawn a background sub-agent`,
			expected: []ParsedToolCall{
				{Name: "spawn_subagent", Args: map[string]any{}},
			},
		},
		{
			name: "tool call with actual parameters",
			content: `[tool activity]
[read_file] => path="main.go", limit=50`,
			expected: []ParsedToolCall{
				{Name: "read_file", Args: map[string]any{"path": "main.go", "limit": int64(50)}},
			},
		},
		{
			name: "tool call with string parameters",
			content: `[tool activity]
[write_file] => path="test.txt", content="hello world"`,
			expected: []ParsedToolCall{
				{Name: "write_file", Args: map[string]any{"path": "test.txt", "content": "hello world"}},
			},
		},
		{
			name: "invalid tool ignored",
			content: `[tool activity]
[invalid_tool] => This should be ignored`,
			expected: []ParsedToolCall{},
		},
		{
			name: "mixed valid and invalid tools",
			content: `[tool activity]
[read_file] => path="test.txt"
[invalid_tool] => Ignore this
[write_file] => path="out.txt", content="done"`,
			expected: []ParsedToolCall{
				{Name: "read_file", Args: map[string]any{"path": "test.txt"}},
				{Name: "write_file", Args: map[string]any{"path": "out.txt", "content": "done"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &YoloAgent{}
			calls := a.parseTextToolCalls(tt.content)
			
			if len(calls) != len(tt.expected) {
				t.Errorf("Expected %d calls, got %d. Expected: %+v, Got: %+v", 
					len(tt.expected), len(calls), tt.expected, calls)
				return
			}
			
			for i, call := range calls {
				if call.Name != tt.expected[i].Name {
					t.Errorf("Call %d: expected name '%s', got '%s'", i, tt.expected[i].Name, call.Name)
				}
				
				// Compare args maps
				if !reflect.DeepEqual(call.Args, tt.expected[i].Args) {
					t.Errorf("Call %d: expected args %+v, got %+v", i, tt.expected[i].Args, call.Args)
				}
			}
		})
	}
}

// TestToolActivityFormatWithModelResponse tests parsing within a realistic model response
func TestToolActivityFormatWithModelResponse(t *testing.T) {
	content := `Here's my plan for this task:

[tool activity]
I'll start by reading the main.go file to understand its structure.
[read_file] => path="main.go", limit=100

Then I'll search for any existing test files.
[search_files] => pattern="*_test.go"

Finally, I'll create a new test file based on what I find.
[write_file] => path="integration_test.go", content="package main"`

	a := &YoloAgent{}
	calls := a.parseTextToolCalls(content)
	
	expectedCount := 3
	if len(calls) != expectedCount {
		t.Errorf("Expected %d tool calls, got %d. Calls: %+v", expectedCount, len(calls), calls)
		return
	}

	// Verify each call
	expectedCalls := map[string]map[string]any{
		"read_file":    {"path": "main.go", "limit": int64(100)},
		"search_files": {"pattern": "*_test.go"},
		"write_file":   {"path": "integration_test.go", "content": "package main"},
	}

	for _, call := range calls {
		expected, exists := expectedCalls[call.Name]
		if !exists {
			t.Errorf("Unexpected tool call: %s", call.Name)
			continue
		}
		if !reflect.DeepEqual(call.Args, expected) {
			t.Errorf("Tool %s args mismatch. Expected %+v, got %+v", call.Name, expected, call.Args)
		}
	}
}
