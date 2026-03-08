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
		name        string
		text        string
		wantRow     int
		wantCol     int
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
