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

// TestXMLParameterToolCallFormat tests Format 2 parsing with <tool_call><function=name><parameter=key>value</parameter></function></tool_call>
func TestXMLParameterToolCallFormat(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []ParsedToolCall
	}{
		{
			name: "multiline parameter values",
			content: `
<tool_call>
<function=read_file>
<parameter=path>
main_test.go
</parameter>
<parameter=offset>
800
</parameter>
<parameter=limit>
50
</parameter>
</function>
</tool_call>
`,
			expected: []ParsedToolCall{
				{Name: "read_file", Args: map[string]any{"path": "main_test.go", "offset": int64(800), "limit": int64(50)}},
			},
		},
		{
			name:    "inline parameter values",
			content: `<tool_call><function=read_file><parameter=path>main.go</parameter><parameter=limit>100</parameter></function></tool_call>`,
			expected: []ParsedToolCall{
				{Name: "read_file", Args: map[string]any{"path": "main.go", "limit": int64(100)}},
			},
		},
		{
			name:    "bare function without tool_call wrapper",
			content: `<function=write_file><parameter=path>out.txt</parameter><parameter=content>hello world</parameter></function>`,
			expected: []ParsedToolCall{
				{Name: "write_file", Args: map[string]any{"path": "out.txt", "content": "hello world"}},
			},
		},
		{
			name: "mixed whitespace in parameters",
			content: `
<tool_call>
<function=read_file>
<parameter=path>  main_test.go  </parameter>
<parameter=offset>800</parameter>
<parameter=limit>
50
</parameter>
</function>
</tool_call>`,
			expected: []ParsedToolCall{
				{Name: "read_file", Args: map[string]any{"path": "main_test.go", "offset": int64(800), "limit": int64(50)}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &YoloAgent{}
			calls := a.parseTextToolCalls(tt.content)

			if len(calls) != len(tt.expected) {
				t.Fatalf("Expected %d calls, got %d. Calls: %+v", len(tt.expected), len(calls), calls)
			}

			for i, call := range calls {
				if call.Name != tt.expected[i].Name {
					t.Errorf("Call %d: expected name '%s', got '%s'", i, tt.expected[i].Name, call.Name)
				}
				if !reflect.DeepEqual(call.Args, tt.expected[i].Args) {
					t.Errorf("Call %d: expected args %+v, got %+v", i, tt.expected[i].Args, call.Args)
				}
			}
		})
	}
}

// TestUnclosedTagToolCallFormat tests Format 2c parsing where closing tags are missing
func TestUnclosedTagToolCallFormat(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []ParsedToolCall
	}{
		{
			name: "unclosed parameter and function tags",
			content: `<tool_call>
<function=run_command>
<parameter=command>
head -n 149 /Users/user/src/yolo/tools.go > /tmp/tools_temp.go && cat >> /tmp/tools_temp.go << 'EOF'
	toolDef("interrupt", "Pause execution")
EOF`,
			expected: []ParsedToolCall{
				{Name: "run_command", Args: map[string]any{"command": "head -n 149 /Users/user/src/yolo/tools.go > /tmp/tools_temp.go && cat >> /tmp/tools_temp.go << 'EOF'\n\ttoolDef(\"interrupt\", \"Pause execution\")\nEOF"}},
			},
		},
		{
			name: "unclosed tags with single parameter inline",
			content: `<function=read_file>
<parameter=path>main.go`,
			expected: []ParsedToolCall{
				{Name: "read_file", Args: map[string]any{"path": "main.go"}},
			},
		},
		{
			name: "unclosed tags with multiple parameters",
			content: `<tool_call>
<function=read_file>
<parameter=path>main.go
<parameter=limit>100`,
			expected: []ParsedToolCall{
				{Name: "read_file", Args: map[string]any{"path": "main.go", "limit": int64(100)}},
			},
		},
		{
			name:    "properly closed tags still work via earlier formats",
			content: `<function=read_file><parameter=path>main.go</parameter></function>`,
			expected: []ParsedToolCall{
				{Name: "read_file", Args: map[string]any{"path": "main.go"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &YoloAgent{}
			calls := a.parseTextToolCalls(tt.content)

			if len(calls) != len(tt.expected) {
				t.Fatalf("Expected %d calls, got %d. Calls: %+v", len(tt.expected), len(calls), calls)
			}

			for i, call := range calls {
				if call.Name != tt.expected[i].Name {
					t.Errorf("Call %d: expected name '%s', got '%s'", i, tt.expected[i].Name, call.Name)
				}
				if !reflect.DeepEqual(call.Args, tt.expected[i].Args) {
					t.Errorf("Call %d: expected args %+v, got %+v", i, tt.expected[i].Args, call.Args)
				}
			}
		})
	}
}

// TestStripUnclosedToolCalls tests that stripTextToolCalls handles unclosed tags
func TestStripUnclosedToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "strip unclosed tool_call with function",
			input:    "Here is my plan:\n<tool_call>\n<function=run_command>\n<parameter=command>\nls -la",
			expected: "Here is my plan:",
		},
		{
			name:     "strip unclosed bare function",
			input:    "Let me read that file.\n<function=read_file>\n<parameter=path>main.go",
			expected: "Let me read that file.",
		},
		{
			name:     "properly closed tags still stripped",
			input:    "Done.\n<function=read_file><parameter=path>main.go</parameter></function>",
			expected: "Done.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripTextToolCalls(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
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

// TestHybridBracketXMLFormat tests Format 8: [tool_name]\n<parameter=key>value
func TestHybridBracketXMLFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []ParsedToolCall
	}{
		{
			name: "single parameter with closing tag",
			input: `[run_command]
<parameter=command>ls -la</parameter>`,
			expected: []ParsedToolCall{
				{Name: "run_command", Args: map[string]any{"command": "ls -la"}},
			},
		},
		{
			name: "multiple parameters",
			input: `[search_files]
<parameter=query>md5</parameter>
<parameter=pattern>**/*.go</parameter>`,
			expected: []ParsedToolCall{
				{Name: "search_files", Args: map[string]any{"query": "md5", "pattern": "**/*.go"}},
			},
		},
		{
			name: "parameter without closing tag",
			input: `[run_command]
<parameter=command>cd /src && grep -rn "crypto/md5" . 2>/dev/null | head -20`,
			expected: []ParsedToolCall{
				{Name: "run_command", Args: map[string]any{"command": `cd /src && grep -rn "crypto/md5" . 2>/dev/null | head -20`}},
			},
		},
		{
			name: "with thinking text before",
			input: `[thinking] Let me search for md5 usage.
[search_files]
<parameter=query>crypto/md5</parameter>
<parameter=pattern>**/*.go</parameter>`,
			expected: []ParsedToolCall{
				{Name: "search_files", Args: map[string]any{"query": "crypto/md5", "pattern": "**/*.go"}},
			},
		},
		{
			name: "parameter value on next line",
			input: "[run_command]\n<parameter=command>\nls -la\n</parameter>",
			expected: []ParsedToolCall{
				{Name: "run_command", Args: map[string]any{"command": "ls -la"}},
			},
		},
		{
			name: "multi-line parameter value",
			input: "[run_command]\n<parameter=command>\nfind . -name '*.go' | head -10\n</parameter>",
			expected: []ParsedToolCall{
				{Name: "run_command", Args: map[string]any{"command": "find . -name '*.go' | head -10"}},
			},
		},
	}

	a := &YoloAgent{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := a.parseTextToolCalls(tt.input)
			if len(calls) != len(tt.expected) {
				t.Errorf("Expected %d tool calls, got %d. Calls: %+v", len(tt.expected), len(calls), calls)
				return
			}
			for i, call := range calls {
				if call.Name != tt.expected[i].Name {
					t.Errorf("Call %d: expected name %q, got %q", i, tt.expected[i].Name, call.Name)
				}
				if !reflect.DeepEqual(call.Args, tt.expected[i].Args) {
					t.Errorf("Call %d: expected args %+v, got %+v", i, tt.expected[i].Args, call.Args)
				}
			}
		})
	}
}
