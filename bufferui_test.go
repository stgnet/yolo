package main

import (
	"strings"
	"sync"
	"testing"
)

// TestBufferUI_New tests NewBufferUI creates valid state
func TestBufferUI_New(t *testing.T) {
	b := NewBufferUI()
	if b == nil {
		t.Fatal("NewBufferUI returned nil")
	}
	if !b.promptShown {
		// Should start as closed (ready)
		select {
		case <-b.promptReady:
			// OK - channel is closed
		default:
			t.Error("promptReady should be initially closed")
		}
	}
}

// TestBufferUI_Write_Buffering tests that Write buffers when buffering=true
func TestBufferUI_Write_Buffering(t *testing.T) {
	b := NewBufferUI()
	b.buffering = true
	
	testText := "test output line\n"
	b.Write(testText)
	
	expected := testText
	if b.buffer.String() != expected {
		t.Errorf("buffer mismatch: got %q, want %q", b.buffer.String(), expected)
	}
}

// TestBufferUI_Write_NotBuffering tests Write with buffering=false would output to stdout
func TestBufferUI_Write_NotBuffering(t *testing.T) {
	b := NewBufferUI()
	b.buffering = false
	
	// This would write to stdout, we can't easily capture it in test
	// But we can verify no panic and buffer is empty
	b.Write("test text\n")
	
	if b.buffer.Len() != 0 {
		t.Error("buffer should be empty when not buffering")
	}
}

// TestBufferUI_Write_NoEscaping tests Write does not escape markdown (passes through raw)
func TestBufferUI_Write_NoEscaping(t *testing.T) {
	b := NewBufferUI()
	b.buffering = true
	
	testText := "test **bold** and `code`"
	b.Write(testText)
	
	// BufferUI writes text as-is without escaping
	expected := testText
	if b.buffer.String() != expected {
		t.Errorf("text mismatch: got %q, want %q", b.buffer.String(), expected)
	}
}

// TestBufferUI_Write_MidLineTracking tests midLine state tracking when not buffering
func TestBufferUI_Write_MidLineTracking(t *testing.T) {
	b := NewBufferUI()
	// When buffering=true, Write() doesn't update midLine
	b.buffering = false
	
	// midLine starts as false
	if b.midLine {
		t.Error("midLine should be false initially")
	}
	
	// After writing without newline, midLine should be true
	b.Write("line2")
	if !b.midLine {
		t.Error("midLine should be true after partial line when not buffering")
	}
}

// TestBufferUI_PromptReady tests PromptReady returns correct channel state
func TestBufferUI_PromptReady(t *testing.T) {
	b := NewBufferUI()
	
	// Initial promptReady is closed, so reading from it should succeed immediately
	ch := b.PromptReady()
	
	select {
	case <-ch:
		// OK - channel was closed (initial state is "ready")
	default:
		t.Error("promptReady should be initially closable/readable")
	}
}

// TestBufferUI_IsUserTyping tests IsUserTyping returns correct state
func TestBufferUI_IsUserTyping(t *testing.T) {
	b := NewBufferUI()
	
	if b.IsUserTyping() {
		t.Error("IsUserTyping should return false initially")
	}
	
	b.mu.Lock()
	b.userWantsInput = true
	b.mu.Unlock()
	
	if !b.IsUserTyping() {
		t.Error("IsUserTyping should return true after user types")
	}
}

// TestBufferUI_IsPromptShown tests IsPromptShown returns correct state
func TestBufferUI_IsPromptShown(t *testing.T) {
	b := NewBufferUI()
	
	if b.IsPromptShown() {
		t.Error("IsPromptShown should return false initially")
	}
	
	b.mu.Lock()
	b.promptShown = true
	b.mu.Unlock()
	
	if !b.IsPromptShown() {
		t.Error("IsPromptShown should return true after prompt shown")
	}
}

// TestBufferUI_ShowInitialPrompt tests ShowInitialPrompt sets correct state
func TestBufferUI_ShowInitialPrompt(t *testing.T) {
	b := NewBufferUI()
	
	// NewBufferUI creates a closed channel, so we don't close it again
	b.ShowInitialPrompt()
	
	if !b.promptShown {
		t.Error("promptShown should be true after ShowInitialPrompt")
	}
}

// TestBufferUI_ResetPrompt tests ResetPrompt resets state correctly
func TestBufferUI_ResetPrompt(t *testing.T) {
	b := NewBufferUI()
	b.mu.Lock()
	b.promptShown = true
	b.mu.Unlock()
	
	b.ResetPrompt()
	
	// ResetPrompt only clears promptShown, not other states
	if b.IsUserTyping() {
		t.Error("userWantsInput should remain false")
	}
	// buffering and userWantsInput are not reset by ResetPrompt
	if b.promptShown {
		t.Error("promptShown should be false after ResetPrompt")
	}
}

// TestBufferUI_FlushBuffer tests FlushBuffer outputs buffered content and resets buffer
func TestBufferUI_FlushBuffer(t *testing.T) {
	b := NewBufferUI()
	b.buffering = true
	testText := "flushed content\n"
	b.Write(testText)
	
	// Capture buffer state before flush
	bufStr := b.buffer.String()
	if bufStr != testText {
		t.Errorf("buffer mismatch before flush: got %q, want %q", bufStr, testText)
	}
	
	// After flush, buffer should be empty
	b.FlushBuffer()
	if b.buffer.Len() != 0 {
		t.Error("buffer should be empty after FlushBuffer")
	}
	if b.buffering {
		t.Error("buffering should be false after FlushBuffer")
	}
}

// TestBufferUI_CancelInput tests CancelInput resets state correctly
func TestBufferUI_CancelInput(t *testing.T) {
	b := NewBufferUI()
	b.mu.Lock()
	b.userWantsInput = true
	b.buffering = true
	b.buffer.WriteString("cancelled\n")
	b.mu.Unlock()
	
	b.CancelInput()
	
	if b.IsUserTyping() {
		t.Error("userWantsInput should be false after CancelInput")
	}
	if b.buffering {
		t.Error("buffering should be false after CancelInput")
	}
	if b.buffer.Len() != 0 {
		t.Error("buffer should be empty after CancelInput")
	}
}

// TestBufferUI_EnterKey tests EnterKey sets pastFirstLine correctly
func TestBufferUI_EnterKey(t *testing.T) {
	b := NewBufferUI()
	
	// EnterKey requires promptShown=true to actually do anything
	b.mu.Lock()
	b.promptShown = true
	b.pastFirstLine = false
	b.mu.Unlock()
	
	b.EnterKey()
	
	if !b.pastFirstLine {
		t.Error("pastFirstLine should be true after EnterKey")
	}
}

// TestBufferUI_Write_Concurrent tests Write is safe for concurrent calls
func TestBufferUI_Write_Concurrent(t *testing.T) {
	b := NewBufferUI()
	b.buffering = true
	
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(val int) {
			b.Write("line")
			done <- true
		}(i)
	}
	
	for i := 0; i < 10; i++ {
		<-done
	}
	
	if !strings.Contains(b.buffer.String(), "line") {
		t.Error("concurrent writes should all be captured in buffer")
	}
}

// TestBufferUI_BasicState_MutexConsistency tests basic mutex is present
func TestBufferUI_BasicState_MutexConsistency(t *testing.T) {
	b := NewBufferUI()
	
	// Verify mutex exists and can be locked/unlocked
	var wg sync.WaitGroup
	wg.Add(10)
	
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			b.mu.Lock()
			_ = b.buffering // Read state
			b.mu.Unlock()
		}()
	}
	
	wg.Wait()
	// If we get here without deadlock, mutex is working
}
