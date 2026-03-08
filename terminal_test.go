package main

import (
	"os"
	"strings"
	"testing"
)

func TestTerminalUIWrapTextEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		cols          int
		expectedLines int
	}{
		{"empty_string", "", 40, 1},                    // Returns ""
		{"single_word_fits", "hello", 40, 1},           // No wrap needed
		{"words_with_spaces", "hello world test", 10, 2}, // Wraps to 2 lines at 10 chars max
		{"exact_fit", "exactly twenty chars", 20, 1},   // Fits exactly
		{"preserves_newlines", "line1\n\nline2", 40, 3}, // Preserves existing newlines including empty line
		{"empty_line_in_middle", "a\n\nb", 40, 3},       // Empty line in middle preserved
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui := &TerminalUI{cols: tt.cols}
			result := ui.wrapText(tt.text)

			lines := strings.Split(result, "\n")
			
			if len(lines) != tt.expectedLines {
				t.Errorf("wrapText produced %d lines, want %d. Result:\n%q", len(lines), tt.expectedLines, result)
			}
		})
	}
}

func TestHistoryManagerWriteRead(t *testing.T) {
	dir, err := os.MkdirTemp("", "yolo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	hm := NewHistoryManager(dir)

	if hm.Data.Messages == nil {
		t.Errorf("New history manager should start with empty Messages slice, got nil")
	}

	if len(hm.Data.Messages) != 0 {
		t.Errorf("New history manager should start with empty messages, got %d items", len(hm.Data.Messages))
	}

	hm.AddMessage("user", "test message", nil)
	
	if len(hm.Data.Messages) != 1 {
		t.Errorf("History should have 1 item after AddMessage, got %d", len(hm.Data.Messages))
	}

	entry := hm.Data.Messages[0]
	if entry.Role != "user" || entry.Content != "test message" {
		t.Errorf("History entry mismatch: %+v", entry)
	}
}

func TestInputManagerBasic(t *testing.T) {
	agent := &YoloAgent{}
	im := NewInputManager(agent)
	
	// buf can be nil or empty, both are valid for a new InputManager
	if im.buf != nil && len(im.buf) > 0 {
		t.Errorf("New InputManager should start with empty buffer, got %q", string(im.buf))
	}
	
	if im.agent != agent {
		t.Errorf("InputManager should store agent reference")
	}
	
	if im.Lines == nil {
		t.Errorf("InputManager Lines channel should be initialized")
	}
	
	if im.rawBytes == nil || cap(im.rawBytes) < 1 {
		t.Errorf("InputManager rawBytes channel should be initialized with capacity")
	}
}
