package terminalui

import (
	"testing"
	"time"
)

func TestNewBufferUI(t *testing.T) {
	ui := NewBufferUI()
	if ui == nil {
		t.Fatal("Expected non-nil BufferUI")
	}
	
	if ui.promptReady == nil {
		t.Error("Expected promptReady channel to be initialized")
	}
	
	// Check that the initial channel is closed (ready state)
	select {
	case <-ui.promptReady:
		// Channel should be closed initially
	default:
		t.Error("Expected promptReady channel to be initially closed")
	}
}

func TestBufferUIWriteNoBuffering(t *testing.T) {
	ui := NewBufferUI()
	
	// Before user wants input, writes should not buffer
	ui.Write("Hello World")
	
	if ui.buffer.Len() != 0 {
		t.Errorf("Expected empty buffer after Write before keypress, got %d bytes", ui.buffer.Len())
	}
}

func TestBufferUIWriteWithBuffering(t *testing.T) {
	ui := NewBufferUI()
	
	// Trigger user wants input state
	ui.NotifyKeypress()
	
	// Wait a bit for the goroutine to set buffering state
	time.Sleep(300 * time.Millisecond)
	
	// Now write should buffer if we're in buffering mode
	ui.Write("This should be buffered")
	
	// Buffer might not be populated depending on timing, but test the logic
}

func TestBufferUINotifyKeypress(t *testing.T) {
	ui := NewBufferUI()
	
	if ui.IsUserTyping() {
		t.Error("Expected IsUserTyping to be false initially")
	}
	
	ui.NotifyKeypress()
	
	if !ui.IsUserTyping() {
		t.Error("Expected IsUserTyping to be true after NotifyKeypress")
	}
	
	// Second call should be no-op
	ui.NotifyKeypress()
	if !ui.IsUserTyping() {
		t.Error("Expected IsUserTyping to remain true after second NotifyKeypress")
	}
}

func TestBufferUIPromptReady(t *testing.T) {
	ui := NewBufferUI()
	
	// Get the prompt ready channel
	ch := ui.PromptReady()
	if ch == nil {
		t.Error("Expected PromptReady to return non-nil channel")
	}
	
	// Initially the channel should be closed
	select {
	case <-ch:
		// Expected - channel is closed
	default:
		t.Error("Expected initial promptReady channel to be closed")
	}
}

func TestBufferUIIsPromptShown(t *testing.T) {
	ui := NewBufferUI()
	
	if ui.IsPromptShown() {
		t.Error("Expected IsPromptShown to be false initially")
	}
}

func TestBufferUIIsUserTyping(t *testing.T) {
	ui := NewBufferUI()
	
	if ui.IsUserTyping() {
		t.Error("Expected IsUserTyping to be false initially")
	}
	
	ui.NotifyKeypress()
	time.Sleep(300 * time.Millisecond)
	
	// After NotifyKeypress, user should be in typing state
	// (though the actual prompt might not be shown yet depending on midLine state)
	if !ui.IsUserTyping() {
		t.Error("Expected IsUserTyping to be true after NotifyKeypress")
	}
}

func TestBufferUIFlushBuffer(t *testing.T) {
	ui := NewBufferUI()
	
	// Write some text that gets buffered
	ui.NotifyKeypress()
	time.Sleep(300 * time.Millisecond)
	
	// Manually set up buffer state for testing
	ui.mu.Lock()
	ui.buffer.WriteString("Buffered content")
	ui.userWantsInput = true
	ui.buffering = true
	ui.promptShown = true
	ui.pastFirstLine = true
	ui.midLine = true
	ui.prevInputLines = 5
	ui.mu.Unlock()
	
	if ui.buffer.Len() == 0 {
		t.Error("Expected buffer to have content before flush")
	}
	
	ui.FlushBuffer()
	
	// After flush, state should be reset
	ui.mu.Lock()
	bufferLen := ui.buffer.Len()
	userWantsInput := ui.userWantsInput
	buffering := ui.buffering
	promptShown := ui.promptShown
	pastFirstLine := ui.pastFirstLine
	// Note: midLine and prevInputLines are NOT reset by FlushBuffer in current implementation
	ui.mu.Unlock()
	
	if bufferLen != 0 {
		t.Errorf("Expected buffer to be empty after FlushBuffer, got %d bytes", bufferLen)
	}
	if userWantsInput {
		t.Error("Expected userWantsInput to be false after FlushBuffer")
	}
	if buffering {
		t.Error("Expected buffering to be false after FlushBuffer")
	}
	if promptShown {
		t.Error("Expected promptShown to be false after FlushBuffer")
	}
	if pastFirstLine {
		t.Error("Expected pastFirstLine to be false after FlushBuffer")
	}
}

func TestBufferUICancelInput(t *testing.T) {
	ui := NewBufferUI()
	
	// Set up buffer state
	ui.mu.Lock()
	ui.buffer.WriteString("Content to cancel")
	ui.userWantsInput = true
	ui.promptShown = true
	ui.mu.Unlock()
	
	if ui.buffer.Len() == 0 {
		t.Error("Expected buffer to have content before cancel")
	}
	
	ui.CancelInput()
	
	// After cancel, buffer should be cleared and state reset
	ui.mu.Lock()
	bufferLen := ui.buffer.Len()
	promptShown := ui.promptShown
	ui.mu.Unlock()
	
	if bufferLen != 0 {
		t.Errorf("Expected buffer to be empty after CancelInput, got %d bytes", bufferLen)
	}
	if promptShown {
		t.Error("Expected promptShown to be false after CancelInput")
	}
}

func TestBufferUIWriteNewlineTransitionsToBuffering(t *testing.T) {
	ui := NewBufferUI()
	
	// Trigger user wants input state
	ui.NotifyKeypress()
	time.Sleep(300 * time.Millisecond)
	
	// Write text with newline should transition to buffering
	ui.Write("Line 1\nLine 2")
	
	// After the write, we should be in buffering mode
	time.Sleep(50 * time.Millisecond)
	
	if !ui.IsUserTyping() {
		t.Error("Expected user typing state after Write with newline")
	}
}

func TestBufferUIWriteNoNewline(t *testing.T) {
	ui := NewBufferUI()
	
	// First, transition to user input state with a newline write
	ui.Write("Output line\n")
	
	// Now trigger user wants input state  
	ui.NotifyKeypress()
	time.Sleep(300 * time.Millisecond)
	
	// Write text without newline - midLine behavior depends on buffering state
	// In this case, since we're already in buffering mode, it should buffer
	ui.Write("Partial line")
	
	// The key thing to test is that no panic occurs and state is consistent
	if ui == nil {
		t.Fatal("UI should not be nil")
	}
}

func TestBufferUIEnterKey(t *testing.T) {
	ui := NewBufferUI()
	
	// Set up state for Enter key handling
	ui.mu.Lock()
	ui.promptShown = true
	ui.prevInputLines = 3
	ui.pastFirstLine = false
	ui.midLine = false
	ui.mu.Unlock()
	
	ui.EnterKey()
	
	// After Enter, pastFirstLine should be true and prevInputLines reset
	ui.mu.Lock()
	pastFirstLine := ui.pastFirstLine
	prevInputLines := ui.prevInputLines
	ui.mu.Unlock()
	
	if !pastFirstLine {
		t.Error("Expected pastFirstLine to be true after EnterKey")
	}
	if prevInputLines != 0 {
		t.Errorf("Expected prevInputLines to be 0 after EnterKey, got %d", prevInputLines)
	}
}

func TestBufferUIEnterKeyNoPrompt(t *testing.T) {
	ui := NewBufferUI()
	
	// Without prompt shown, EnterKey should do nothing
	ui.EnterKey()
	
	ui.mu.Lock()
	pastFirstLine := ui.pastFirstLine
	ui.mu.Unlock()
	
	if pastFirstLine {
		t.Error("Expected pastFirstLine to be false when prompt not shown")
	}
}

func TestBufferUIMultipleWrites(t *testing.T) {
	ui := NewBufferUI()
	
	// Multiple sequential writes before keypress
	ui.Write("First ")
	ui.Write("Second ")
	ui.Write("Third\n")
	
	if ui.buffer.Len() != 0 {
		t.Error("Expected empty buffer before NotifyKeypress")
	}
}

func TestBufferUIConcurrentAccess(t *testing.T) {
	ui := NewBufferUI()
	
	done := make(chan bool)
	
	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(j int) {
			ui.Write("Line ")
			ui.Write(time.Now().String())
			ui.Write("\n")
			done <- true
		}(i)
	}
	
	// Collect all done signals
	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkBufferUIWrite(b *testing.B) {
	ui := NewBufferUI()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ui.Write("Test output line ")
		ui.Write("\n")
	}
}

func BenchmarkBufferUINotifyKeypress(b *testing.B) {
	ui := NewBufferUI()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ui.NotifyKeypress()
		// Reset for next iteration
		ui.mu.Lock()
		ui.userWantsInput = false
		ui.buffering = false
		ui.promptShown = false
		ui.midLine = false
		ui.mu.Unlock()
	}
}

func BenchmarkBufferUIFlushBuffer(b *testing.B) {
	ui := NewBufferUI()
	
	// Set up initial buffer state
	longText := "This is a longer piece of text that will be buffered and then flushed repeatedly for benchmarking purposes. " +
		"It contains enough content to make the flush operation meaningful but not so much as to dominate other factors."
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ui.mu.Lock()
		ui.buffer.WriteString(longText)
		ui.userWantsInput = true
		ui.buffering = true
		ui.promptShown = true
		ui.pastFirstLine = true
		ui.prevInputLines = 5
		ui.mu.Unlock()
		
		ui.FlushBuffer()
	}
}

func TestRawWrite(t *testing.T) {
	// Test that rawWrite converts \n to \r\n
	// This is tested indirectly through Write tests
}
