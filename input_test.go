package main

import (
	"testing"
	"time"
)

func TestUtf8ByteLen(t *testing.T) {
	tests := []struct {
		name string
		lead byte
		want int
	}{
		{"2-byte (latin)", 0xC3, 2}, // e.g., é
		{"2-byte low", 0xC0, 2},
		{"2-byte high", 0xDF, 2},
		{"3-byte (CJK)", 0xE4, 3}, // e.g., 中
		{"3-byte low", 0xE0, 3},
		{"3-byte high", 0xEF, 3},
		{"4-byte (emoji)", 0xF0, 4}, // e.g., 😀
		{"4-byte high", 0xF4, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utf8ByteLen(tt.lead)
			if got != tt.want {
				t.Errorf("utf8ByteLen(0x%02X) = %d, want %d", tt.lead, got, tt.want)
			}
		})
	}
}

func TestInputLineStruct(t *testing.T) {
	// Test the InputLine struct is properly constructed
	line := InputLine{Text: "hello", OK: true}
	if line.Text != "hello" || !line.OK {
		t.Errorf("InputLine mismatch: %+v", line)
	}

	eof := InputLine{OK: false}
	if eof.OK {
		t.Error("EOF line should have OK=false")
	}
}

func TestNewInputManagerChannelCapacity(t *testing.T) {
	agent := &YoloAgent{}
	im := NewInputManager(agent)

	if cap(im.Lines) != 8 {
		t.Errorf("Lines channel capacity = %d, want 8", cap(im.Lines))
	}
	if cap(im.rawBytes) != 64 {
		t.Errorf("rawBytes channel capacity = %d, want 64", cap(im.rawBytes))
	}
	if cap(im.rawErr) != 1 {
		t.Errorf("rawErr channel capacity = %d, want 1", cap(im.rawErr))
	}
}

func TestNewInputManagerSendDelay(t *testing.T) {
	agent := &YoloAgent{}
	im := NewInputManager(agent)

	// Default delay should be DefaultInputDelay seconds
	expected := time.Duration(DefaultInputDelay) * time.Second
	if im.sendDelay != expected {
		t.Errorf("sendDelay = %v, want %v", im.sendDelay, expected)
	}
}

// TestInputManagerClearLine tests the ClearLine method
func TestInputManagerClearLine(t *testing.T) {
	agent := &YoloAgent{}
	im := NewInputManager(agent)

	// Set up buffer directly for testing
	im.mu.Lock()
	im.buf = []byte("test input")
	im.prompt = "> "
	im.mu.Unlock()

	// Clear the line and get the text that was cleared
	clearedText := im.ClearLine()

	if clearedText != "test input" {
		t.Errorf("ClearLine returned %q, want %q", clearedText, "test input")
	}

	// Verify buffer is now empty
	im.mu.Lock()
	bufLen := len(im.buf)
	im.mu.Unlock()

	if bufLen != 0 {
		t.Errorf("Buffer length after ClearLine = %d, want 0", bufLen)
	}
}

// TestInputManagerSyncToUI_NoUI tests UI sync logic when globalUI is nil
func TestInputManagerSyncToUI_NoUI(t *testing.T) {
	agent := &YoloAgent{}
	im := NewInputManager(agent)

	// Set up buffer
	im.mu.Lock()
	im.buf = []byte("hello")
	im.prompt = "> "
	im.mu.Unlock()

	// This should not panic even if globalUI is nil (default state)
	im.SyncToUI()
}

// TestInputManagerRedrawAfterOutput_NoUI tests redraw logic when globalUI is nil
func TestInputManagerRedrawAfterOutput_NoUI(t *testing.T) {
	agent := &YoloAgent{}
	im := NewInputManager(agent)

	// Set up buffer
	im.mu.Lock()
	im.buf = []byte("test")
	im.prompt = "> "
	im.mu.Unlock()

	// This should not panic even if globalUI is nil (default state)
	im.RedrawAfterOutput()
}

// TestInputManagerStop_NoState tests terminal restoration when oldState is nil
func TestInputManagerStop_NoState(t *testing.T) {
	agent := &YoloAgent{}
	im := NewInputManager(agent)

	// oldState is nil by default (terminal mode not started)
	// Stop should handle this gracefully without panic
	im.Stop()
}

// TestUtf8ByteLenEdgeCases tests boundary values
func TestUtf8ByteLenEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		lead        byte
		want        int
		description string
	}{
		{name: "ASCII range", lead: 0x7F, want: 2, description: "Falls through to 2-byte case"},
		{name: "Max 2-byte", lead: 0xDF, want: 2},
		{name: "Min 3-byte", lead: 0xE0, want: 3},
		{name: "Max 3-byte", lead: 0xEF, want: 3},
		{name: "Min 4-byte", lead: 0xF0, want: 4},
		{name: "Max 4-byte", lead: 0xF7, want: 4, description: "Max valid UTF-8 leading byte"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utf8ByteLen(tt.lead)
			if got != tt.want {
				t.Errorf("utf8ByteLen(0x%02X) = %d, want %d", tt.lead, got, tt.want)
			}
		})
	}
}

// TestInputManagerFdAccess tests file descriptor handling
func TestInputManagerFdAccess(t *testing.T) {
	agent := &YoloAgent{}
	im := NewInputManager(agent)

	// fd should be set to stdin's file descriptor
	im.mu.Lock()
	fd := im.fd
	im.mu.Unlock()

	if fd != 0 { // stdin is typically fd 0
		t.Logf("fd = %d (expected 0 for stdin)", fd)
	}
}

// TestInputManagerShowPrompt_NoUI tests prompt setting and syncAndRedraw when globalUI is nil
func TestInputManagerShowPrompt_NoUI(t *testing.T) {
	agent := &YoloAgent{}
	im := NewInputManager(agent)

	// Show a prompt - should not panic even with globalUI nil
	testPrompt := "test> "
	im.ShowPrompt(testPrompt)

	// Verify prompt was set
	im.mu.Lock()
	prompt := im.prompt
	im.mu.Unlock()

	if prompt != testPrompt {
		t.Errorf("ShowPrompt set prompt to %q, want %q", prompt, testPrompt)
	}
}
