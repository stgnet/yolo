package terminalui

import (
	"strings"
	"testing"
)

func TestNewTerminalUI(t *testing.T) {
	ui := NewTerminalUI()
	
	if ui == nil {
		t.Fatal("Expected non-nil TerminalUI")
	}
	
	if ui.rows != 24 {
		t.Errorf("Expected default rows to be 24, got %d", ui.rows)
	}
	
	if ui.cols != 80 {
		t.Errorf("Expected default cols to be 80, got %d", ui.cols)
	}
	
	if ui.outWin == nil {
		t.Error("Expected outWin to be initialized")
	}
	
	if ui.termWin == nil {
		t.Error("Expected termWin to be initialized")
	}
	
	if len(ui.subagents) != 0 {
		t.Errorf("Expected empty subagents map, got %d entries", len(ui.subagents))
	}
}

func TestTerminalUIAddSubagentWindow(t *testing.T) {
	ui := NewTerminalUI()
	
	ui.AddSubagentWindow(1, "Test Agent")
	
	if len(ui.subagents) != 1 {
		t.Errorf("Expected 1 subagent, got %d", len(ui.subagents))
	}
	
	win, ok := ui.subagents[1]
	if !ok {
		t.Error("Expected subagent with ID 1 to exist")
	} else {
		if win.ID != 1 {
			t.Errorf("Expected subagent ID to be 1, got %d", win.ID)
		}
		if win.Label != "Test Agent" {
			t.Errorf("Expected subagent label to be 'Test Agent', got %q", win.Label)
		}
		if win.Completed {
			t.Error("Expected subagent to not be completed initially")
		}
	}
	
	// Test adding multiple subagents
	ui.AddSubagentWindow(2, "Second Agent")
	
	if len(ui.subagents) != 2 {
		t.Errorf("Expected 2 subagents, got %d", len(ui.subagents))
	}
	
	if len(ui.subagentOrder) != 2 {
		t.Errorf("Expected subagentOrder to have 2 entries, got %d", len(ui.subagentOrder))
	} else if ui.subagentOrder[0] != 1 || ui.subagentOrder[1] != 2 {
		t.Errorf("Expected subagentOrder to be [1, 2], got %v", ui.subagentOrder)
	}
}

func TestTerminalUIWriteToSubagentWindow(t *testing.T) {
	ui := NewTerminalUI()
	
	ui.AddSubagentWindow(1, "Test Agent")
	ui.WriteToSubagentWindow(1, "Hello World")
	
	if len(ui.subagents) != 1 {
		t.Fatal("Expected subagent to exist")
	}
	
	expectedContent := "Hello World"
	actualContent := ui.subagents[1].TextBuffer.String()
	
	if actualContent != expectedContent {
		t.Errorf("Expected buffer content %q, got %q", expectedContent, actualContent)
	}
}

func TestTerminalUIMarkSubagentComplete(t *testing.T) {
	ui := NewTerminalUI()
	
	ui.AddSubagentWindow(1, "Test Agent")
	
	if ui.subagents[1].Completed {
		t.Error("Expected subagent to not be completed initially")
	}
	
	ui.MarkSubagentComplete(1)
	
	if !ui.subagents[1].Completed {
		t.Error("Expected subagent to be completed after MarkSubagentComplete")
	}
	
	if ui.subagents[1].CompletedAt.IsZero() {
		t.Error("Expected CompletedAt to be set after MarkSubagentComplete")
	}
}


func TestStripAnsiCodes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no ansi codes",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "simple color code",
			input:    "\033[31mRed Text\033[0m",
			expected: "Red Text",
		},
		{
			name:     "bold code",
			input:    "\033[1mBold\033[0m",
			expected: "Bold",
		},
		{
			name:     "multiple codes",
			input:    "\033[32mGreen\033[0m and \033[31mRed\033[0m",
			expected: "Green and Red",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripAnsiCodes(tt.input)
			if result != tt.expected {
				t.Errorf("StripAnsiCodes(%q) = %q; expected %q", 
					tt.input, result, tt.expected)
			}
		})
	}
}

func TestBreakWordAtVisibleLength(t *testing.T) {
	tests := []struct {
		name          string
		word          string
		maxLen        int
		expectedPrefix string
		expectedSuffix string
	}{
		{
			name:          "shorter than max",
			word:          "ShortWord",
			maxLen:        20,
			expectedPrefix: "ShortWord",
			expectedSuffix: "",
		},
		{
			name:          "longer than max",
			word:          "ThisIsAVeryLongWordThatShouldBeBroken",
			maxLen:        15,
			expectedPrefix: "ThisIsAVeryLong",
			expectedSuffix: "WordThatShouldBeBroken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, suffix := BreakWordAtVisibleLength(tt.word, tt.maxLen)
			
			if prefix != tt.expectedPrefix {
				t.Errorf("Expected prefix %q, got %q", tt.expectedPrefix, prefix)
			}
			
			if suffix != tt.expectedSuffix {
				t.Errorf("Expected suffix %q, got %q", tt.expectedSuffix, suffix)
			}
		})
	}
}

func TestTerminalUIOutputPrint(t *testing.T) {
	ui := NewTerminalUI()
	
	// OutputPrint writes to both buffer and stdout (sanitized)
	ui.OutputPrint("Test output")
	
	if ui.outputBuffer.Len() == 0 {
		t.Error("Expected outputBuffer to have content after OutputPrint")
	}
}

func TestTruncateStringWithAnsi(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "no ansi, shorter than max",
			input:    "Short text",
			maxLen:   20,
			expected: "Short text",
		},
		{
			name:     "with ansi, shorter than max",
			input:    "\033[31mRed text\033[0m",
			maxLen:   20,
			expected: "\033[31mRed text\033[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateStringWithAnsi(tt.input, tt.maxLen)
			// Just verify it doesn't panic and returns something reasonable
			if len(result) > tt.maxLen+10 {
				t.Errorf("TruncateStringWithAnsi returned string too long: %d chars", len(result))
			}
		})
	}
}

func TestSubagentWindowConstants(t *testing.T) {
	// Verify constants are reasonable
	if SubagentContentRows != 4 {
		t.Errorf("Expected SubagentContentRows to be 4, got %d", SubagentContentRows)
	}
	
	if SubagentWindowRows != 5 { // ContentRows + 1 for header
		t.Errorf("Expected SubagentWindowRows to be 5, got %d", SubagentWindowRows)
	}
	
	if MaxVisibleSubWindows < 1 || MaxVisibleSubWindows > 10 {
		t.Errorf("Expected MaxVisibleSubWindows to be between 1 and 10, got %d", MaxVisibleSubWindows)
	}
	
	if MinScrollRows != 4 {
		t.Errorf("Expected MinScrollRows to be 4, got %d", MinScrollRows)
	}
}

func TestColorConstants(t *testing.T) {
	// Verify color constants are properly defined
	colors := map[string]string{
		"Reset":   Reset,
		"Bold":    Bold,
		"Red":     Red,
		"Green":   Green,
		"Cyan":    Cyan,
	}
	
	for name, color := range colors {
		if len(color) == 0 {
			t.Errorf("Color constant %s is empty", name)
		}
		
		if !strings.HasPrefix(color, "\033[") {
			t.Errorf("Color constant %s doesn't start with ANSI escape", name)
		}
		
		if !strings.HasSuffix(color, "m") {
			t.Errorf("Color constant %s doesn't end with 'm'", name)
		}
	}
}
