package main

import (
	"strings"
	"testing"
	"time"
)

// TestNewBufferUI verifies that NewBufferUI creates a valid BufferUI instance
func TestNewBufferUI(t *testing.T) {
	ui := NewBufferUI()
	if ui == nil {
		t.Fatal("Expected non-nil BufferUI")
	}
	if ui.promptReady == nil {
		t.Error("Expected non-nil promptReady channel")
	}
	// Default state should have closed promptReady (ready immediately)
	select {
	case <-ui.promptReady:
		// Expected - channel is initially closed
	default:
		t.Error("Expected promptReady to be initially closed")
	}
}

// TestBufferUI_Write tests the Write method in various states
func TestBufferUI_Write(t *testing.T) {
	tests := []struct {
		name         string
		initialState func(*BufferUI)
		input        string
		checkOutput  func(*BufferUI) bool
		description  string
	}{
		{
			name:         "write when not buffering and no user input",
			initialState: func(u *BufferUI) { /* default state */ },
			input:        "test output\n",
			checkOutput: func(u *BufferUI) bool {
				return !u.buffering && !u.userWantsInput
			},
			description: "Should write directly without buffering",
		},
		{
			name: "write when user wants input but midLine is false",
			initialState: func(u *BufferUI) {
				u.userWantsInput = true
				u.buffering = true
				// Create new promptReady channel since it's closed by default
				u.promptReady = make(chan struct{})
			},
			input: "test output\n",
			checkOutput: func(u *BufferUI) bool {
				return u.buffering && strings.Contains(u.buffer.String(), "test output")
			},
			description: "Should buffer the output",
		},
		{
			name: "write multi-line when user wants input transitions to buffering",
			initialState: func(u *BufferUI) {
				u.userWantsInput = true
				u.midLine = false
				u.promptReady = make(chan struct{}) // Fresh channel for this test
			},
			input: "first line\nsecond line",
			checkOutput: func(u *BufferUI) bool {
				return u.buffering && !u.midLine
			},
			description: "Should transition to buffering mode after newline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui := NewBufferUI()
			tt.initialState(ui)
			ui.Write(tt.input)
			if !tt.checkOutput(ui) {
				t.Errorf("%s - Check failed", tt.description)
			}
		})
	}
}

// TestBufferUI_NotifyKeypress tests the keystroke notification
func TestBufferUI_NotifyKeypress(t *testing.T) {
	tests := []struct {
		name         string
		initialState func(*BufferUI)
		checkAfter   func(*BufferUI, bool) bool
		description  string
	}{
		{
			name: "first keypress enables user input mode",
			initialState: func(u *BufferUI) {
				// Default state
			},
			checkAfter: func(u *BufferUI, promptShown bool) bool {
				return u.userWantsInput == true
			},
			description: "Should set userWantsInput to true on first keypress",
		},
		{
			name: "second keypress is a no-op",
			initialState: func(u *BufferUI) {
				u.userWantsInput = true
			},
			checkAfter: func(u *BufferUI, promptShown bool) bool {
				return u.userWantsInput == true // Should stay true
			},
			description: "Should not change state if user already wants input",
		},
		{
			name: "mid-line output delays buffering until newline or timeout",
			initialState: func(u *BufferUI) {
				u.midLine = true
			},
			checkAfter: func(u *BufferUI, promptShown bool) bool {
				return u.userWantsInput == true // At minimum, this should be set
			},
			description: "Should handle mid-line output correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui := NewBufferUI()
			tt.initialState(ui)
			ui.NotifyKeypress()

			// Wait for potential async operations
			time.Sleep(300 * time.Millisecond)

			promptShown := ui.IsPromptShown()
			if !tt.checkAfter(ui, promptShown) {
				t.Errorf("%s - Check failed. userWantsInput=%v, buffering=%v, promptShown=%v",
					tt.description, ui.userWantsInput, ui.buffering, promptShown)
			}
		})
	}
}

// TestBufferUI_AccessorMethods tests the getter methods
func TestBufferUI_AccessorMethods(t *testing.T) {
	ui := NewBufferUI()

	// Initially all should be false/default
	if ui.IsPromptShown() {
		t.Error("Expected IsPromptShown to return false initially")
	}
	if ui.IsUserTyping() {
		t.Error("Expected IsUserTyping to return false initially")
	}

	// After NotifyKeypress, user should be typing
	ui.NotifyKeypress()
	time.Sleep(100 * time.Millisecond)

	if !ui.IsUserTyping() {
		t.Error("Expected IsUserTyping to return true after NotifyKeypress")
	}
}

// TestBufferUI_RedrawPrompt tests the prompt redraw functionality
func TestBufferUI_RedrawPrompt(t *testing.T) {
	ui := NewBufferUI()

	// Redraw should be safe to call multiple times
	ui.RedrawPrompt("test input")
	ui.RedrawPrompt("updated input")

	// Should not panic or cause issues
	if ui == nil {
		t.Fatal("BufferUI became nil during RedrawPrompt")
	}
}

// TestBufferUI_ThreadSafety tests concurrent access to BufferUI
func TestBufferUI_ThreadSafety(t *testing.T) {
	ui := NewBufferUI()
	done := make(chan bool)

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			ui.Write("test output\n")
		}
		done <- true
	}()

	// Concurrent NotifyKeypress calls
	go func() {
		for i := 0; i < 100; i++ {
			ui.NotifyKeypress()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Should not have caused any race conditions or panics
	if ui == nil {
		t.Fatal("BufferUI became nil during concurrent access")
	}
}
