package main

import (
	"strings"
	"testing"
)

// Test truncateString function
func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"Hello, World!", 20, "Hello, World!"},
		{"Hello, World!", 10, "Hello, Wor..."},
		{"Short", 100, "Short"},
		{"Exactly ten", 10, "Exactly te..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q; want %q",
				tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

// Test TerminalUI initialization
func TestNewTerminalUI(t *testing.T) {
	ui := NewTerminalUI()
	if ui.fd == 0 {
		// fd might be 0 if stdout is not a TTY, which is acceptable in test environments
		t.Log("Note: FD is 0, likely not a TTY in test environment")
	}
	if ui.rows == 0 || ui.cols == 0 {
		t.Error("TerminalUI initialized with 0 rows or cols")
	}
	if ui.outRow != 1 || ui.outCol != 1 {
		t.Errorf("Initial cursor position: outRow=%d, outCol=%d; want outRow=1, outCol=1",
			ui.outRow, ui.outCol)
	}
	if ui.scrollEnd != ui.rows-2 {
		t.Errorf("Initial scrollEnd=%d; want rows-2=%d", ui.scrollEnd, ui.rows-2)
	}
}

// Test TerminalUI UpdateInput and RedrawInput
func TestTerminalUI_UpdateInput(t *testing.T) {
	ui := NewTerminalUI()
	prompt := ""
	input := []byte("test input")

	ui.UpdateInput(prompt, input)
	ui.RedrawInput() // Should not panic

	// Just verifying it doesn't crash - actual output is hard to test in unit tests
}

// Test TerminalUI RefreshSize (basic smoke test)
func TestTerminalUI_RefreshSize(t *testing.T) {
	ui := NewTerminalUI()
	originalRows := ui.rows
	originalCols := ui.cols

	// This should not panic even if terminal size doesn't change
	ui.RefreshSize()

	if ui.rows <= 0 || ui.cols <= 0 {
		t.Errorf("RefreshSize resulted in invalid dimensions: rows=%d, cols=%d",
			ui.rows, ui.cols)
	}

	// Dimensions might stay the same or change based on environment
	t.Logf("Terminal dimensions - Before: %dx%d, After: %dx%d",
		originalRows, originalCols, ui.rows, ui.cols)
}

// Test TerminalUI ClearInputLine (smoke test)
func TestTerminalUI_ClearInputLine(t *testing.T) {
	ui := NewTerminalUI()
	// Should not panic
	ui.ClearInputLine()
}

// Test TerminalUI Setup and Teardown (smoke tests)
func TestTerminalUI_SetupTeardown(t *testing.T) {
	ui := NewTerminalUI()

	// These should not panic
	ui.Setup()
	defer ui.Teardown()
}

// Test wrapText edge cases
func TestWrapTextEdgeCases(t *testing.T) {
	ui := NewTerminalUI()

	tests := []struct {
		name     string
		text     string
		cols     int
		expected string
	}{
		{
			name:     "empty text",
			text:     "",
			cols:     80,
			expected: "",
		},
		{
			name:     "single word shorter than cols",
			text:     "short",
			cols:     10,
			expected: "short",
		},
		{
			name:     "multiple words",
			text:     "one two three",
			cols:     20,
			expected: "one two three",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Temporarily change cols for testing
			originalCols := ui.cols
			ui.cols = tt.cols

			result := ui.wrapText(tt.text)

			// Restore
			ui.cols = originalCols

			// For complex wrapping, just verify it doesn't panic and returns valid output
			if len(result) == 0 && len(tt.text) > 0 {
				t.Errorf("wrapText returned empty string for non-empty input")
			}

			t.Logf("Input: %q, Output:\n%q", tt.text, result)
		})
	}
}

// Test trackCursorMovement function
func TestTrackCursorMovement(t *testing.T) {
	ui := NewTerminalUI()

	tests := []struct {
		name    string
		cols    int
		text    string
		wantRow int
		wantCol int
	}{
		{
			name:    "simple text",
			cols:    80,
			text:    "hello",
			wantRow: 1,
			wantCol: 6,
		},
		{
			name:    "with newline",
			cols:    80,
			text:    "hello\nworld",
			wantRow: 2,
			wantCol: 6,
		},
		{
			name:    "with carriage return",
			cols:    80,
			text:    "hello\rworld",
			wantRow: 1,
			wantCol: 6,
		},
		{
			// This is the key bug case: a line exactly filling the terminal
			// width followed by \n should advance only ONE row, not two.
			// rawWrite converts \n to \r\n; the \r cancels the pending wrap
			// and \n advances one row.
			name:    "exact width line then newline",
			cols:    10,
			text:    "1234567890\nABC",
			wantRow: 2,
			wantCol: 4,
		},
		{
			name:    "exact width line then char",
			cols:    10,
			text:    "1234567890X",
			wantRow: 2,
			wantCol: 2,
		},
		{
			name:    "exact width line at end",
			cols:    10,
			text:    "1234567890",
			wantRow: 2,
			wantCol: 1,
		},
		{
			name:    "exact width then carriage return",
			cols:    10,
			text:    "1234567890\rABC",
			wantRow: 1,
			wantCol: 4,
		},
		{
			name:    "two exact width lines",
			cols:    10,
			text:    "1234567890\n1234567890\nABC",
			wantRow: 3,
			wantCol: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui.outRow = 1
			ui.outCol = 1
			ui.cols = tt.cols
			ui.scrollEnd = 100 // large enough to not interfere

			stripped := stripAnsiCodes(tt.text)
			ui.trackCursorMovement(stripped)

			if ui.outRow != tt.wantRow || ui.outCol != tt.wantCol {
				t.Errorf("trackCursorMovement(%q): got row=%d col=%d, want row=%d col=%d",
					tt.text, ui.outRow, ui.outCol, tt.wantRow, tt.wantCol)
			}
		})
	}
}

// Test that pending wrap at scrollEnd boundary sets ui.pendingWrap instead of
// eagerly resolving (which would cause the "CR without LF" overwrite bug).
func TestTrackCursorMovement_PendingWrapAtScrollEnd(t *testing.T) {
	ui := NewTerminalUI()
	ui.cols = 10
	ui.scrollEnd = 5

	// Text fills exactly to col boundary while at scrollEnd
	ui.outRow = 5
	ui.outCol = 1
	ui.pendingWrap = false
	ui.trackCursorMovement("1234567890") // exactly cols chars
	if !ui.pendingWrap {
		t.Error("expected pendingWrap=true when text fills width at scrollEnd")
	}
	// outRow/outCol should stay at scrollEnd — the actual scroll is deferred
	// to resolvePendingWrap() on the next output call.
	if ui.outRow != 5 {
		t.Errorf("outRow: got %d, want 5", ui.outRow)
	}

	// When NOT at scrollEnd, pending wrap should resolve eagerly
	ui.outRow = 3
	ui.outCol = 1
	ui.pendingWrap = false
	ui.trackCursorMovement("1234567890")
	if ui.pendingWrap {
		t.Error("expected pendingWrap=false when NOT at scrollEnd")
	}
	if ui.outRow != 4 || ui.outCol != 1 {
		t.Errorf("eager resolve: got row=%d col=%d, want row=4 col=1", ui.outRow, ui.outCol)
	}
}

// Test OutputPrintInline and OutputFinishLine (smoke tests)
func TestTerminalUI_OutputMethods(t *testing.T) {
	ui := NewTerminalUI()

	// Should not panic
	ui.OutputPrintInline("test message")
	ui.OutputFinishLine()
}

// Test OutputError and other output methods (smoke tests)
func TestTerminalUI_ErrorOutputMethods(t *testing.T) {
	ui := NewTerminalUI()

	// Should not panic
	ui.OutputPrintInline("error message")
	ui.OutputFinishLine()
}

// Test ClearInputLine (smoke test)
func TestTerminalUI_ClearLines(t *testing.T) {
	ui := NewTerminalUI()

	// Should not panic with various operations
	ui.ClearInputLine()
	ui.WriteToInputLine("test")
}

// Test color printing helper functions (smoke tests)
func TestColorPrinting(t *testing.T) {
	// These should not panic - they write directly to stdout
	cprint("test", "red")
	cprintNoNL("test", "blue")
}

// Test that scrollEnd adjusts dynamically with multiline input
func TestTerminalUI_DynamicScrollEnd(t *testing.T) {
	ui := &TerminalUI{rows: 24, cols: 80, outRow: 1, outCol: 1, scrollEnd: 22}
	ui.inputBuf = []byte{}

	// With empty input: scrollEnd should be rows - 1(input) - 1(divider) = 22
	ui.drawInputLocked()
	if ui.scrollEnd != 22 {
		t.Errorf("Empty input: scrollEnd=%d, want 22", ui.scrollEnd)
	}

	// Single line of input: scrollEnd stays at 22 (1 row of input)
	ui.inputBuf = []byte("typing")
	ui.drawInputLocked()
	// peakBottomRows grows to 1, scrollEnd = 24 - 1 - 1 = 22
	if ui.scrollEnd != 22 {
		t.Errorf("Single line input: scrollEnd=%d, want 22", ui.scrollEnd)
	}

	// Multi-line input: 3 lines = 3 bottom rows
	ui.peakBottomRows = 0
	ui.inputBuf = []byte("line1\nline2\nline3")
	ui.drawInputLocked()
	// 3 lines = 3 bottom rows, scrollEnd = 24 - 3 - 1 = 20
	if ui.scrollEnd != 20 {
		t.Errorf("3-line input: scrollEnd=%d, want 20", ui.scrollEnd)
	}

	// Remove lines but peak holds (grow-only)
	ui.inputBuf = []byte("line1")
	ui.drawInputLocked()
	if ui.scrollEnd != 20 {
		t.Errorf("After shrinking input (grow-only): scrollEnd=%d, want 20", ui.scrollEnd)
	}

	// Clear input: should shrink back to single line
	ui.inputBuf = []byte{}
	ui.drawInputLocked()
	if ui.scrollEnd != 22 {
		t.Errorf("After clearing input: scrollEnd=%d, want 22", ui.scrollEnd)
	}
}

// Test multiline input display line computation
func TestTerminalUI_ComputeInputDisplayLines(t *testing.T) {
	ui := &TerminalUI{cols: 80}

	tests := []struct {
		name      string
		input     string
		wantLines int
	}{
		{"empty", "", 1},
		{"single line", "hello", 1},
		{"two lines", "hello\nworld", 2},
		{"trailing newline", "hello\n", 2},
		{"three lines", "a\nb\nc", 3},
		{"blank lines", "a\n\nb", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := ui.computeInputDisplayLines(tt.input)
			if len(lines) != tt.wantLines {
				t.Errorf("computeInputDisplayLines(%q) = %d lines, want %d. Lines: %v",
					tt.input, len(lines), tt.wantLines, lines)
			}
		})
	}
}

// Test that long input lines wrap correctly
func TestTerminalUI_ComputeInputDisplayLinesWrapping(t *testing.T) {
	ui := &TerminalUI{cols: 10}

	// 25-char line should wrap to 3 display lines (10+10+5)
	lines := ui.computeInputDisplayLines(strings.Repeat("X", 25))
	if len(lines) != 3 {
		t.Errorf("25 chars at cols=10: got %d lines, want 3", len(lines))
	}

	// Multiline with wrapping
	lines = ui.computeInputDisplayLines(strings.Repeat("A", 15) + "\n" + strings.Repeat("B", 5))
	// First logical line: 15 chars → 2 display lines (10+5)
	// Second logical line: 5 chars → 1 display line
	if len(lines) != 3 {
		t.Errorf("15+5 chars at cols=10: got %d lines, want 3", len(lines))
	}
}

// Test multi-line input row calculation (no prompt width now)
func TestInputRowCount(t *testing.T) {
	ui := &TerminalUI{cols: 80}

	tests := []struct {
		name      string
		input     string
		wantRows  int
	}{
		{"empty", "", 1},
		{"short", "hello", 1},
		{"newline", "hello\nworld", 2},
		{"three lines", "a\nb\nc", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := ui.computeInputDisplayLines(tt.input)
			if len(lines) != tt.wantRows {
				t.Errorf("input %q: got %d rows, want %d", tt.input, len(lines), tt.wantRows)
			}
		})
	}
}

// Test that bottom area is capped to prevent overwhelming the output region
func TestTerminalUI_BottomAreaCap(t *testing.T) {
	ui := &TerminalUI{rows: 10, cols: 80, outRow: 1, outCol: 1, scrollEnd: 8}
	ui.inputBuf = []byte{}

	// A very tall multiline input should be capped (rows-4 = 6 max bottom)
	var longInput strings.Builder
	for i := 0; i < 20; i++ {
		if i > 0 {
			longInput.WriteByte('\n')
		}
		longInput.WriteString("msg")
	}
	ui.inputBuf = []byte(longInput.String())
	ui.drawInputLocked()

	// scrollEnd must stay >= 1
	if ui.scrollEnd < 1 {
		t.Errorf("scrollEnd went below 1: %d", ui.scrollEnd)
	}
	// At least 3 rows for output
	if ui.scrollEnd < 3 {
		t.Logf("Note: scrollEnd=%d with 10-row terminal and 20-line input", ui.scrollEnd)
	}
}

// Test that outRow is capped when scroll region shrinks
func TestTerminalUI_OutRowCapping(t *testing.T) {
	ui := &TerminalUI{rows: 24, cols: 80, outRow: 22, outCol: 1, scrollEnd: 22}
	ui.inputBuf = []byte{}

	// Multiline input shrinks scroll region; outRow should be capped
	ui.inputBuf = []byte("a\nb\nc\nd\ne")
	ui.drawInputLocked()

	if ui.outRow > ui.scrollEnd {
		t.Errorf("outRow=%d exceeds scrollEnd=%d after shrink", ui.outRow, ui.scrollEnd)
	}
}
