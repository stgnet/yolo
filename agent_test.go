package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── convertParamValue ──────────────────────────────────────────────

func TestConvertParamValue_Integer(t *testing.T) {
	assert.Equal(t, int64(42), convertParamValue("42"))
	assert.Equal(t, int64(0), convertParamValue("0"))
	assert.Equal(t, int64(-7), convertParamValue("-7"))
}

func TestConvertParamValue_Float(t *testing.T) {
	assert.Equal(t, 3.14, convertParamValue("3.14"))
	assert.Equal(t, -0.5, convertParamValue("-0.5"))
}

func TestConvertParamValue_Bool(t *testing.T) {
	assert.Equal(t, true, convertParamValue("true"))
	assert.Equal(t, false, convertParamValue("false"))
	assert.Equal(t, true, convertParamValue("True"))
}

func TestConvertParamValue_String(t *testing.T) {
	assert.Equal(t, "hello", convertParamValue("hello"))
	assert.Equal(t, "foo bar", convertParamValue("foo bar"))
	assert.Equal(t, "", convertParamValue(""))
}

// ─── parseParamString ───────────────────────────────────────────────

func TestParseParamString_Basic(t *testing.T) {
	result := parseParamString(`path="tools.go", offset=100`)
	assert.Equal(t, "tools.go", result["path"])
	assert.Equal(t, int64(100), result["offset"])
}

func TestParseParamString_SingleQuotes(t *testing.T) {
	result := parseParamString(`path='main.go'`)
	assert.Equal(t, "main.go", result["path"])
}

func TestParseParamString_BoolValues(t *testing.T) {
	result := parseParamString(`verbose=true, quiet=false`)
	assert.Equal(t, true, result["verbose"])
	assert.Equal(t, false, result["quiet"])
}

func TestParseParamString_Empty(t *testing.T) {
	result := parseParamString("")
	assert.Empty(t, result)
}

func TestParseParamString_NoPairs(t *testing.T) {
	result := parseParamString("just some text")
	assert.Empty(t, result)
}

// ─── parseFuncCallArgs ──────────────────────────────────────────────

func TestParseFuncCallArgs_QuotedWithCommas(t *testing.T) {
	result := parseFuncCallArgs(`command="cd /src && ls -la", limit=100`)
	assert.Equal(t, "cd /src && ls -la", result["command"])
	assert.Equal(t, int64(100), result["limit"])
}

func TestParseFuncCallArgs_EscapedQuotes(t *testing.T) {
	result := parseFuncCallArgs(`text="say \"hello\""`)
	assert.Equal(t, `say \"hello\"`, result["text"])
}

func TestParseFuncCallArgs_Unquoted(t *testing.T) {
	result := parseFuncCallArgs(`path=tools.go, offset=50`)
	assert.Equal(t, "tools.go", result["path"])
	assert.Equal(t, int64(50), result["offset"])
}

func TestParseFuncCallArgs_Empty(t *testing.T) {
	result := parseFuncCallArgs("")
	assert.Empty(t, result)
}

func TestParseFuncCallArgs_SingleQuoted(t *testing.T) {
	result := parseFuncCallArgs(`path='my file.go'`)
	assert.Equal(t, "my file.go", result["path"])
}

// ─── isFileMutationTool ─────────────────────────────────────────────

func TestIsFileMutationTool(t *testing.T) {
	assert.True(t, isFileMutationTool("write_file"))
	assert.True(t, isFileMutationTool("edit_file"))
	assert.True(t, isFileMutationTool("move_file"))
	assert.False(t, isFileMutationTool("read_file"))
	assert.False(t, isFileMutationTool("run_command"))
	assert.False(t, isFileMutationTool("copy_file"))
	assert.False(t, isFileMutationTool(""))
}

// ─── stripTextToolCalls ─────────────────────────────────────────────

func TestStripTextToolCalls_ActivityBlock(t *testing.T) {
	input := "Here is my plan.\n[tool activity] read_file(path=\"foo.go\")\n[/tool activity]\nDone."
	result := stripTextToolCalls(input)
	assert.Contains(t, result, "Here is my plan.")
	assert.Contains(t, result, "Done.")
	assert.NotContains(t, result, "[tool activity]")
}

func TestStripTextToolCalls_ToolCallXML(t *testing.T) {
	input := `Some text. <tool_call>{"name":"read_file","args":{"path":"foo"}}</tool_call> More text.`
	result := stripTextToolCalls(input)
	assert.Contains(t, result, "Some text.")
	assert.Contains(t, result, "More text.")
	assert.NotContains(t, result, "<tool_call>")
}

func TestStripTextToolCalls_BareFunction(t *testing.T) {
	input := "Hello\n<function=read_file><parameter=path>foo.go</parameter></function>\nWorld"
	result := stripTextToolCalls(input)
	assert.Contains(t, result, "Hello")
	assert.Contains(t, result, "World")
	assert.NotContains(t, result, "<function=")
}

func TestStripTextToolCalls_PreservesPlainText(t *testing.T) {
	input := "This is just a normal response with no tool calls."
	result := stripTextToolCalls(input)
	assert.Equal(t, input, result)
}

func TestStripTextToolCalls_CollapsesBlankLines(t *testing.T) {
	input := "Before\n\n\n\n\nAfter"
	result := stripTextToolCalls(input)
	assert.Equal(t, "Before\n\nAfter", result)
}

// ─── stripOrphanedCloseTags ─────────────────────────────────────────

func TestStripOrphanedCloseTags_RemovesOrphans(t *testing.T) {
	input := "some text</parameter> more text</function>"
	result := stripOrphanedCloseTags(input)
	assert.Equal(t, "some text more text", result)
}

func TestStripOrphanedCloseTags_PreservesMatched(t *testing.T) {
	input := "<parameter=path>foo</parameter>"
	result := stripOrphanedCloseTags(input)
	assert.Equal(t, input, result)
}

func TestStripOrphanedCloseTags_RemovesToolActivityOrphan(t *testing.T) {
	input := "text[/tool activity]more"
	result := stripOrphanedCloseTags(input)
	assert.Equal(t, "textmore", result)
}

func TestStripOrphanedCloseTags_PlainText(t *testing.T) {
	input := "nothing to strip here"
	result := stripOrphanedCloseTags(input)
	assert.Equal(t, input, result)
}

// ─── parseTextToolCalls ─────────────────────────────────────────────

// Minimal agent stub for testing parseTextToolCalls
func newTestAgent() *YoloAgent {
	return &YoloAgent{baseDir: "."}
}

func TestParseTextToolCalls_Format1_JSON(t *testing.T) {
	a := newTestAgent()
	input := `<tool_call>{"name": "read_file", "args": {"path": "main.go"}}</tool_call>`
	calls := a.parseTextToolCalls(input)
	require.Len(t, calls, 1)
	assert.Equal(t, "read_file", calls[0].Name)
	assert.Equal(t, "main.go", calls[0].Args["path"])
}

func TestParseTextToolCalls_Format2_XMLParams(t *testing.T) {
	a := newTestAgent()
	input := `<tool_call><function=read_file><parameter=path>main.go</parameter><parameter=offset>10</parameter></function></tool_call>`
	calls := a.parseTextToolCalls(input)
	require.Len(t, calls, 1)
	assert.Equal(t, "read_file", calls[0].Name)
	assert.Equal(t, "main.go", calls[0].Args["path"])
	assert.Equal(t, int64(10), calls[0].Args["offset"])
}

func TestParseTextToolCalls_Format2b_BareFunction(t *testing.T) {
	a := newTestAgent()
	input := `<function=read_file><parameter=path>main.go</parameter></function>`
	calls := a.parseTextToolCalls(input)
	require.Len(t, calls, 1)
	assert.Equal(t, "read_file", calls[0].Name)
	assert.Equal(t, "main.go", calls[0].Args["path"])
}

func TestParseTextToolCalls_Format6_InlineActivity(t *testing.T) {
	a := newTestAgent()
	input := `[tool activity] read_file(path="main.go", offset=1, limit=200)`
	calls := a.parseTextToolCalls(input)
	require.Len(t, calls, 1)
	assert.Equal(t, "read_file", calls[0].Name)
	assert.Equal(t, "main.go", calls[0].Args["path"])
}

func TestParseTextToolCalls_MultipleCalls(t *testing.T) {
	a := newTestAgent()
	input := `<tool_call>{"name": "read_file", "args": {"path": "a.go"}}</tool_call>
<tool_call>{"name": "read_file", "args": {"path": "b.go"}}</tool_call>`
	calls := a.parseTextToolCalls(input)
	require.Len(t, calls, 2)
	assert.Equal(t, "a.go", calls[0].Args["path"])
	assert.Equal(t, "b.go", calls[1].Args["path"])
}

func TestParseTextToolCalls_NoToolCalls(t *testing.T) {
	a := newTestAgent()
	input := "This is just a normal response with no tool calls at all."
	calls := a.parseTextToolCalls(input)
	assert.Empty(t, calls)
}

func TestParseTextToolCalls_InvalidJSON(t *testing.T) {
	a := newTestAgent()
	input := `<tool_call>{not valid json}</tool_call>`
	calls := a.parseTextToolCalls(input)
	assert.Empty(t, calls)
}

func TestParseTextToolCalls_EmptyArgs(t *testing.T) {
	a := newTestAgent()
	input := `<tool_call>{"name": "list_models", "args": {}}</tool_call>`
	calls := a.parseTextToolCalls(input)
	require.Len(t, calls, 1)
	assert.Equal(t, "list_models", calls[0].Name)
	assert.NotNil(t, calls[0].Args)
}

func TestParseTextToolCalls_NilArgs(t *testing.T) {
	a := newTestAgent()
	input := `<tool_call>{"name": "list_models"}</tool_call>`
	calls := a.parseTextToolCalls(input)
	require.Len(t, calls, 1)
	assert.Equal(t, "list_models", calls[0].Name)
	assert.NotNil(t, calls[0].Args) // should default to empty map
}

// ─── deduplicateToolCalls ───────────────────────────────────────────

func TestDeduplicateToolCalls_RemovesDupes(t *testing.T) {
	calls := []ParsedToolCall{
		{Name: "read_file", Args: map[string]any{"path": "a.go"}},
		{Name: "read_file", Args: map[string]any{"path": "a.go"}},
		{Name: "read_file", Args: map[string]any{"path": "b.go"}},
	}
	result := deduplicateToolCalls(calls)
	assert.Len(t, result, 2)
}

func TestDeduplicateToolCalls_NoDupes(t *testing.T) {
	calls := []ParsedToolCall{
		{Name: "read_file", Args: map[string]any{"path": "a.go"}},
		{Name: "write_file", Args: map[string]any{"path": "b.go"}},
	}
	result := deduplicateToolCalls(calls)
	assert.Len(t, result, 2)
}

func TestDeduplicateToolCalls_Empty(t *testing.T) {
	result := deduplicateToolCalls(nil)
	assert.Empty(t, result)
}
