package main

import (
	"testing"
)

func TestParseTextToolCallsFormat1(t *testing.T) {
	agent := &YoloAgent{}

	// Format 1: <tool_call>{"name": ..., "args": ...}</tool_call>
	text := `<tool_call>{"name": "read_file", "args": {"path": "main.go"}}</tool_call>`
	calls := agent.parseTextToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "read_file" {
		t.Errorf("Expected name 'read_file', got %q", calls[0].Name)
	}
	if calls[0].Args["path"] != "main.go" {
		t.Errorf("Expected path 'main.go', got %v", calls[0].Args["path"])
	}
}

func TestParseTextToolCallsFormat1Multiple(t *testing.T) {
	agent := &YoloAgent{}

	text := `<tool_call>{"name": "read_file", "args": {"path": "a.go"}}</tool_call>
some text
<tool_call>{"name": "write_file", "args": {"path": "b.go", "content": "test"}}</tool_call>`

	calls := agent.parseTextToolCalls(text)
	if len(calls) != 2 {
		t.Fatalf("Expected 2 calls, got %d", len(calls))
	}
	if calls[0].Name != "read_file" {
		t.Errorf("Call 0: expected 'read_file', got %q", calls[0].Name)
	}
	if calls[1].Name != "write_file" {
		t.Errorf("Call 1: expected 'write_file', got %q", calls[1].Name)
	}
}

func TestParseTextToolCallsFormat1NoArgs(t *testing.T) {
	agent := &YoloAgent{}

	text := `<tool_call>{"name": "list_files"}</tool_call>`
	calls := agent.parseTextToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(calls))
	}
	if calls[0].Args == nil {
		t.Error("Args should be non-nil empty map")
	}
}

func TestParseTextToolCallsFormat2(t *testing.T) {
	agent := &YoloAgent{}

	// Format 2: <tool_call><function=name><parameter=key>value</parameter></function></tool_call>
	text := `<tool_call><function=read_file><parameter=path>main.go</parameter></function></tool_call>`
	calls := agent.parseTextToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "read_file" {
		t.Errorf("Expected 'read_file', got %q", calls[0].Name)
	}
	if calls[0].Args["path"] != "main.go" {
		t.Errorf("Expected path 'main.go', got %v", calls[0].Args["path"])
	}
}

func TestParseTextToolCallsFormat3(t *testing.T) {
	agent := &YoloAgent{}

	// Format 3: [tool_name] {"key": "value"}
	text := `[read_file] {"path": "test.go"}`
	calls := agent.parseTextToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "read_file" {
		t.Errorf("Expected 'read_file', got %q", calls[0].Name)
	}
}

func TestParseTextToolCallsFormat4(t *testing.T) {
	agent := &YoloAgent{}

	// Format 4: <tool_name>{"key": "value"}</tool_name>
	text := `<read_file>{"path": "hello.go"}</read_file>`
	calls := agent.parseTextToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "read_file" {
		t.Errorf("Expected 'read_file', got %q", calls[0].Name)
	}
}

func TestParseTextToolCallsFormat4XMLParams(t *testing.T) {
	agent := &YoloAgent{}

	// Format 4 with XML-style params
	text := `<read_file><path>hello.go</path></read_file>`
	calls := agent.parseTextToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(calls))
	}
	if calls[0].Args["path"] != "hello.go" {
		t.Errorf("Expected path 'hello.go', got %v", calls[0].Args["path"])
	}
}

func TestParseTextToolCallsNoMatch(t *testing.T) {
	agent := &YoloAgent{}

	text := "just some regular text with no tool calls"
	calls := agent.parseTextToolCalls(text)
	if len(calls) != 0 {
		t.Errorf("Expected 0 calls for plain text, got %d", len(calls))
	}
}

func TestParseTextToolCallsInvalidJSON(t *testing.T) {
	agent := &YoloAgent{}

	text := `<tool_call>{"name": broken json}</tool_call>`
	calls := agent.parseTextToolCalls(text)
	if len(calls) != 0 {
		t.Errorf("Expected 0 calls for invalid JSON, got %d", len(calls))
	}
}

func TestParseTextToolCallsFormat5(t *testing.T) {
	agent := &YoloAgent{}

	text := `[tool activity]
[read_file] => path=main.go
[/tool activity]`
	calls := agent.parseTextToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "read_file" {
		t.Errorf("Expected 'read_file', got %q", calls[0].Name)
	}
}

func TestParseParamStringExtended(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(map[string]any) bool
	}{
		{
			"integer value",
			"count=42",
			func(m map[string]any) bool {
				v, ok := m["count"].(int64)
				return ok && v == 42
			},
		},
		{
			"float value",
			"ratio=3.14",
			func(m map[string]any) bool {
				v, ok := m["ratio"].(float64)
				return ok && v == 3.14
			},
		},
		{
			"bool value",
			"flag=true",
			func(m map[string]any) bool {
				v, ok := m["flag"].(bool)
				return ok && v
			},
		},
		{
			"quoted string",
			`name="hello world"`,
			func(m map[string]any) bool {
				v, ok := m["name"].(string)
				return ok && v == "hello world"
			},
		},
		{
			"single quoted",
			"name='test'",
			func(m map[string]any) bool {
				v, ok := m["name"].(string)
				return ok && v == "test"
			},
		},
		{
			"no equals sign",
			"noequalssign",
			func(m map[string]any) bool {
				return len(m) == 0
			},
		},
		{
			"empty string",
			"",
			func(m map[string]any) bool {
				return len(m) == 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseParamString(tt.input)
			if !tt.check(result) {
				t.Errorf("parseParamString(%q) = %+v, check failed", tt.input, result)
			}
		})
	}
}

func TestToolDef(t *testing.T) {
	td := toolDef("test_tool", "A test tool",
		map[string]ToolParam{
			"path": {Type: "string", Description: "File path"},
		},
		[]string{"path"},
	)

	if td.Type != "function" {
		t.Errorf("Expected type 'function', got %q", td.Type)
	}
	if td.Function.Name != "test_tool" {
		t.Errorf("Expected name 'test_tool', got %q", td.Function.Name)
	}
	if td.Function.Description != "A test tool" {
		t.Errorf("Expected description 'A test tool', got %q", td.Function.Description)
	}
	if td.Function.Parameters.Type != "object" {
		t.Errorf("Expected params type 'object', got %q", td.Function.Parameters.Type)
	}
	if len(td.Function.Parameters.Properties) != 1 {
		t.Errorf("Expected 1 property, got %d", len(td.Function.Parameters.Properties))
	}
	if len(td.Function.Parameters.Required) != 1 || td.Function.Parameters.Required[0] != "path" {
		t.Errorf("Expected required=['path'], got %v", td.Function.Parameters.Required)
	}
}

func TestDeduplicateToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		calls    []ParsedToolCall
		wantLen  int
		wantName []string
	}{
		{
			name:    "empty",
			calls:   nil,
			wantLen: 0,
		},
		{
			name: "single call unchanged",
			calls: []ParsedToolCall{
				{Name: "write_file", Args: map[string]any{"path": "a.go", "content": "x"}},
			},
			wantLen:  1,
			wantName: []string{"write_file"},
		},
		{
			name: "duplicate write_file removed",
			calls: []ParsedToolCall{
				{Name: "write_file", Args: map[string]any{"path": "a.go", "content": "x"}},
				{Name: "write_file", Args: map[string]any{"path": "a.go", "content": "x"}},
			},
			wantLen:  1,
			wantName: []string{"write_file"},
		},
		{
			name: "different args kept",
			calls: []ParsedToolCall{
				{Name: "write_file", Args: map[string]any{"path": "a.go", "content": "x"}},
				{Name: "write_file", Args: map[string]any{"path": "b.go", "content": "y"}},
			},
			wantLen:  2,
			wantName: []string{"write_file", "write_file"},
		},
		{
			name: "different tools kept",
			calls: []ParsedToolCall{
				{Name: "read_file", Args: map[string]any{"path": "a.go"}},
				{Name: "write_file", Args: map[string]any{"path": "a.go", "content": "x"}},
			},
			wantLen:  2,
			wantName: []string{"read_file", "write_file"},
		},
		{
			name: "triple duplicate reduced to one",
			calls: []ParsedToolCall{
				{Name: "edit_file", Args: map[string]any{"path": "f.go", "old_text": "a", "new_text": "b"}},
				{Name: "edit_file", Args: map[string]any{"path": "f.go", "old_text": "a", "new_text": "b"}},
				{Name: "edit_file", Args: map[string]any{"path": "f.go", "old_text": "a", "new_text": "b"}},
			},
			wantLen:  1,
			wantName: []string{"edit_file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deduplicateToolCalls(tt.calls)
			if len(got) != tt.wantLen {
				t.Errorf("deduplicateToolCalls() returned %d calls, want %d", len(got), tt.wantLen)
			}
			for i, name := range tt.wantName {
				if i < len(got) && got[i].Name != name {
					t.Errorf("call[%d].Name = %q, want %q", i, got[i].Name, name)
				}
			}
		})
	}
}
