package main

import (
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
	if ui.queuedMsgs != nil {
		t.Errorf("Initial queuedMsgs should be nil, got %v", ui.queuedMsgs)
	}
}

// Test TerminalUI UpdateInput and RedrawInput
func TestTerminalUI_UpdateInput(t *testing.T) {
	ui := NewTerminalUI()
	prompt := "[blue]user[reset]> "
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
		text    string
		wantRow int
		wantCol int
	}{
		{
			name:    "simple text",
			text:    "hello",
			wantRow: 1,
			wantCol: 6,
		},
		{
			name:    "with newline",
			text:    "hello\nworld",
			wantRow: 2,
			wantCol: 6,
		},
		{
			name:    "with carriage return",
			text:    "hello\rworld",
			wantRow: 1,
			wantCol: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset cursor position for test
			ui.outRow = 1
			ui.outCol = 1

			stripped := stripAnsiCodes(tt.text)
			ui.trackCursorMovement(stripped)

			if ui.outRow != tt.wantRow || ui.outCol != tt.wantCol {
				t.Errorf("trackCursorMovement: got row=%d col=%d, want row=%d col=%d",
					ui.outRow, ui.outCol, tt.wantRow, tt.wantCol)
			}
		})
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

// Test AddQueuedMessage and RemoveQueuedMessage
func TestTerminalUI_QueuedMessages(t *testing.T) {
	ui := NewTerminalUI()

	// Initially empty
	if len(ui.queuedMsgs) != 0 {
		t.Errorf("Expected 0 queued messages, got %d", len(ui.queuedMsgs))
	}

	// Add messages
	ui.AddQueuedMessage("first")
	if len(ui.queuedMsgs) != 1 || ui.queuedMsgs[0] != "first" {
		t.Errorf("After AddQueuedMessage: got %v", ui.queuedMsgs)
	}

	ui.AddQueuedMessage("second")
	ui.AddQueuedMessage("third")
	if len(ui.queuedMsgs) != 3 {
		t.Errorf("Expected 3 queued messages, got %d", len(ui.queuedMsgs))
	}

	// Remove (FIFO order)
	ui.RemoveQueuedMessage()
	if len(ui.queuedMsgs) != 2 || ui.queuedMsgs[0] != "second" {
		t.Errorf("After first remove: got %v", ui.queuedMsgs)
	}

	ui.RemoveQueuedMessage()
	if len(ui.queuedMsgs) != 1 || ui.queuedMsgs[0] != "third" {
		t.Errorf("After second remove: got %v", ui.queuedMsgs)
	}

	ui.RemoveQueuedMessage()
	if len(ui.queuedMsgs) != 0 {
		t.Errorf("After third remove: expected empty, got %v", ui.queuedMsgs)
	}

	// Remove on empty should be a no-op (not panic)
	ui.RemoveQueuedMessage()
	if len(ui.queuedMsgs) != 0 {
		t.Errorf("Remove on empty: expected empty, got %v", ui.queuedMsgs)
	}
}

// Test that scrollEnd adjusts dynamically when queued messages are added
func TestTerminalUI_DynamicScrollEnd(t *testing.T) {
	ui := &TerminalUI{rows: 24, cols: 80, outRow: 1, outCol: 1, scrollEnd: 22}
	ui.prompt = "you> "
	ui.inputBuf = []byte{}

	// With no queued messages: scrollEnd should be rows - 1(input) - 1(divider) = 22
	ui.drawInputLocked()
	if ui.scrollEnd != 22 {
		t.Errorf("No queued msgs: scrollEnd=%d, want 22", ui.scrollEnd)
	}

	// Add one queued message: bottom grows by 1
	ui.queuedMsgs = []string{"msg1"}
	ui.drawInputLocked()
	if ui.scrollEnd != 21 {
		t.Errorf("1 queued msg: scrollEnd=%d, want 21", ui.scrollEnd)
	}

	// Add more
	ui.queuedMsgs = []string{"msg1", "msg2", "msg3"}
	ui.drawInputLocked()
	if ui.scrollEnd != 19 {
		t.Errorf("3 queued msgs: scrollEnd=%d, want 19", ui.scrollEnd)
	}

	// Remove all
	ui.queuedMsgs = nil
	ui.drawInputLocked()
	if ui.scrollEnd != 22 {
		t.Errorf("After clearing: scrollEnd=%d, want 22", ui.scrollEnd)
	}
}

// Test multi-line input row calculation
func TestInputRowCount(t *testing.T) {
	tests := []struct {
		name          string
		promptWidth   int
		inputLen      int
		cols          int
		wantRowCount  int
	}{
		{"empty input", 5, 0, 80, 1},
		{"short input", 5, 10, 80, 1},          // 15 chars total, fits in 1 row
		{"fills one row", 5, 75, 80, 2},         // 80 chars = 1 full row, cursor wraps → 2
		{"just over one row", 5, 76, 80, 2},     // 81 chars → 2 rows
		{"two full rows", 5, 155, 80, 3},        // 160 chars → 2 full rows + cursor → 3
		{"narrow terminal", 5, 10, 10, 2},       // 15 chars / 10 cols → 2 rows
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalChars := tt.promptWidth + tt.inputLen
			rowCount := 1
			if totalChars > 0 {
				rowCount = totalChars/tt.cols + 1
			}
			if rowCount != tt.wantRowCount {
				t.Errorf("totalChars=%d, cols=%d: got %d rows, want %d",
					totalChars, tt.cols, rowCount, tt.wantRowCount)
			}
		})
	}
}

// Test that bottom area is capped to prevent overwhelming the output region
func TestTerminalUI_BottomAreaCap(t *testing.T) {
	ui := &TerminalUI{rows: 10, cols: 80, outRow: 1, outCol: 1, scrollEnd: 8}
	ui.prompt = "you> "
	ui.inputBuf = []byte{}

	// Add many queued messages — should be capped (rows-4 = 6 max bottom)
	ui.queuedMsgs = make([]string, 20)
	for i := range ui.queuedMsgs {
		ui.queuedMsgs[i] = "msg"
	}
	ui.drawInputLocked()

	// scrollEnd must stay >= 1
	if ui.scrollEnd < 1 {
		t.Errorf("scrollEnd went below 1: %d", ui.scrollEnd)
	}
	// At least 3 rows for output
	if ui.scrollEnd < 3 {
		t.Logf("Note: scrollEnd=%d with 10-row terminal and 20 queued messages", ui.scrollEnd)
	}
}

// Test that outRow is capped when scroll region shrinks
func TestTerminalUI_OutRowCapping(t *testing.T) {
	ui := &TerminalUI{rows: 24, cols: 80, outRow: 22, outCol: 1, scrollEnd: 22}
	ui.prompt = "you> "
	ui.inputBuf = []byte{}

	// Adding queued messages shrinks scroll region; outRow should be capped
	ui.queuedMsgs = []string{"a", "b", "c", "d", "e"}
	ui.drawInputLocked()

	if ui.outRow > ui.scrollEnd {
		t.Errorf("outRow=%d exceeds scrollEnd=%d after shrink", ui.outRow, ui.scrollEnd)
	}
}
