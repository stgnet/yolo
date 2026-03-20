package inputmanager

import (
	"testing"
	"time"
)

func TestNewInputManager(t *testing.T) {
	im := NewInputManager(nil)
	
	if im == nil {
		t.Fatal("Expected non-nil InputManager")
	}
	
	// Check that channels are initialized
	if im.Lines == nil {
		t.Error("Expected Lines channel to be initialized")
	}
	if cap(im.Lines) != 8 {
		t.Errorf("Expected Lines buffer capacity of 8, got %d", cap(im.Lines))
	}
	
	if im.rawBytes == nil {
		t.Error("Expected rawBytes channel to be initialized")
	}
	if cap(im.rawBytes) != 64 {
		t.Errorf("Expected rawBytes buffer capacity of 64, got %d", cap(im.rawBytes))
	}
	
	if im.rawErr == nil {
		t.Error("Expected rawErr channel to be initialized")
	}
	
	// Check default send delay
	if im.sendDelay != 10*time.Second {
		t.Errorf("Expected default send delay of 10s, got %v", im.sendDelay)
	}
}

func TestInputManagerClearLine(t *testing.T) {
	tests := []struct {
		name       string
		initialBuf []byte
		expected   string
	}{
		{
			name:       "empty buffer",
			initialBuf: []byte{},
			expected:   "",
		},
		{
			name:       "single character",
			initialBuf: []byte{'a'},
			expected:   "a",
		},
		{
			name:       "multiple characters",
			initialBuf: []byte("hello"),
			expected:   "hello",
		},
		{
			name:       "with newline",
			initialBuf: []byte("line1\nline2"),
			expected:   "line1\nline2",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewInputManager(nil)
			
			// Set initial buffer content
			im.mu.Lock()
			im.buf = make([]byte, len(tt.initialBuf))
			copy(im.buf, tt.initialBuf)
			im.mu.Unlock()
			
			result := im.ClearLine()
			
			if result != tt.expected {
				t.Errorf("Expected ClearLine to return %q, got %q", tt.expected, result)
			}
			
			// Verify buffer is cleared
			im.mu.Lock()
			bufLen := len(im.buf)
			im.mu.Unlock()
			if bufLen != 0 {
				t.Errorf("Expected buffer to be empty after ClearLine, got length %d", bufLen)
			}
		})
	}
}

func TestInputManagerInsertChar(t *testing.T) {
	tests := []struct {
		name       string
		initialBuf []byte
		inputChar  byte
		expectedBuf []byte
	}{
		{
			name:       "append to empty buffer",
			initialBuf: []byte{},
			inputChar:  'a',
			expectedBuf: []byte{'a'},
		},
		{
			name:       "append to non-empty buffer",
			initialBuf: []byte("hello"),
			inputChar:  '!',
			expectedBuf: []byte("hello!"),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewInputManager(nil)
			
			// Set initial buffer
			im.mu.Lock()
			im.buf = make([]byte, len(tt.initialBuf))
			copy(im.buf, tt.initialBuf)
			im.mu.Unlock()
			
			im.insertChar(tt.inputChar)
			
			// Verify buffer content
			im.mu.Lock()
			if !equalBytes(im.buf, tt.expectedBuf) {
				t.Errorf("Expected buffer %v, got %v", tt.expectedBuf, im.buf)
			}
			im.mu.Unlock()
		})
	}
}

func TestInputManagerHandleBackspace(t *testing.T) {
	tests := []struct {
		name       string
		initialBuf []byte
		expectedBuf []byte
	}{
		{
			name:       "backspace on empty buffer",
			initialBuf: []byte{},
			expectedBuf: []byte{},
		},
		{
			name:       "backspace removes last character",
			initialBuf: []byte("hello"),
			expectedBuf: []byte("hell"),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewInputManager(nil)
			
			// Set initial buffer
			im.mu.Lock()
			im.buf = make([]byte, len(tt.initialBuf))
			copy(im.buf, tt.initialBuf)
			im.mu.Unlock()
			
			im.handleBackspace()
			
			// Verify buffer content
			im.mu.Lock()
			if !equalBytes(im.buf, tt.expectedBuf) {
				t.Errorf("Expected buffer %v, got %v", tt.expectedBuf, im.buf)
			}
			im.mu.Unlock()
		})
	}
}

func TestInputManagerInsertSpaces(t *testing.T) {
	tests := []struct {
		name       string
		initialBuf []byte
		spaces     int
		expectedBuf []byte
	}{
		{
			name:       "insert spaces to empty buffer",
			initialBuf: []byte{},
			spaces:     3,
			expectedBuf: []byte("   "),
		},
		{
			name:       "insert spaces to non-empty buffer",
			initialBuf: []byte("hello"),
			spaces:     2,
			expectedBuf: []byte("hello  "),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewInputManager(nil)
			
			// Set initial buffer
			im.mu.Lock()
			im.buf = make([]byte, len(tt.initialBuf))
			copy(im.buf, tt.initialBuf)
			im.mu.Unlock()
			
			im.insertSpaces(tt.spaces)
			
			// Verify buffer content
			im.mu.Lock()
			if !equalBytes(im.buf, tt.expectedBuf) {
				t.Errorf("Expected buffer %v, got %v", tt.expectedBuf, im.buf)
			}
			im.mu.Unlock()
		})
	}
}

func TestInputManagerHandleEnter(t *testing.T) {
	tests := []struct {
		name       string
		initialBuf []byte
		inputChar  byte
		expectedBuf []byte
	}{
		{
			name:       "enter adds carriage return",
			initialBuf: []byte("hello"),
			inputChar:  '\r',
			expectedBuf: []byte("hello\r"),
		},
		{
			name:       "enter on empty buffer",
			initialBuf: []byte{},
			inputChar:  '\r',
			expectedBuf: []byte("\r"),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewInputManager(nil)
			
			// Set initial buffer
			im.mu.Lock()
			im.buf = make([]byte, len(tt.initialBuf))
			copy(im.buf, tt.initialBuf)
			im.mu.Unlock()
			
			im.handleEnter(tt.inputChar)
			
			// Verify buffer content
			im.mu.Lock()
			if !equalBytes(im.buf, tt.expectedBuf) {
				t.Errorf("Expected buffer %v, got %v", tt.expectedBuf, im.buf)
			}
			im.mu.Unlock()
		})
	}
}

func TestInputManagerCancelInput(t *testing.T) {
	tests := []struct {
		name       string
		initialBuf []byte
	}{
		{
			name:       "cancel clears non-empty buffer",
			initialBuf: []byte("some text"),
		},
		{
			name:       "cancel on empty buffer",
			initialBuf: []byte{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewInputManager(nil)
			
			// Set initial buffer
			im.mu.Lock()
			im.buf = make([]byte, len(tt.initialBuf))
			copy(im.buf, tt.initialBuf)
			im.mu.Unlock()
			
			im.cancelInput()
			
			// Verify buffer is cleared
			im.mu.Lock()
			bufLen := len(im.buf)
			im.mu.Unlock()
			if bufLen != 0 {
				t.Errorf("Expected buffer to be empty after cancelInput, got length %d", bufLen)
			}
		})
	}
}

func TestInputManagerSendLine(t *testing.T) {
	tests := []struct {
		name       string
		initialBuf []byte
		expectSend bool
		expectedText string
	}{
		{
			name:       "send empty buffer",
			initialBuf: []byte{},
			expectSend: true,
			expectedText: "",
		},
		{
			name:       "send non-empty buffer",
			initialBuf: []byte("test line"),
			expectSend: true,
			expectedText: "test line",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewInputManager(nil)
			
			// Set initial buffer
			im.mu.Lock()
			im.buf = make([]byte, len(tt.initialBuf))
			copy(im.buf, tt.initialBuf)
			im.mu.Unlock()
			
			// Set up receiver for Lines channel
			received := make(chan InputLine, 1)
			go func() {
				select {
				case line := <-im.Lines:
					received <- line
				case <-time.After(time.Second):
					close(received)
				}
			}()
			
			im.sendLine()
			time.Sleep(50 * time.Millisecond) // Give goroutine time to receive
			
			if tt.expectSend {
				select {
				case line := <-received:
					if !line.OK {
						t.Error("Expected OK to be true")
					}
					if line.Text != tt.expectedText {
						t.Errorf("Expected text %q, got %q", tt.expectedText, line.Text)
					}
				default:
					t.Error("Expected line to be sent to Lines channel")
				}
			}
			
			// Verify buffer is cleared
			im.mu.Lock()
			bufLen := len(im.buf)
			im.mu.Unlock()
			if bufLen != 0 {
				t.Errorf("Expected buffer to be empty after sendLine, got length %d", bufLen)
			}
		})
	}
}

func TestInputManagerSendToLines(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		ok       bool
		expectedOK bool
	}{
		{
			name:     "send with OK=true",
			text:     "hello world",
			ok:       true,
			expectedOK: true,
		},
		{
			name:     "send with OK=false",
			text:     "",
			ok:       false,
			expectedOK: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewInputManager(nil)
			
			received := make(chan InputLine, 1)
			go func() {
				select {
				case line := <-im.Lines:
					received <- line
				case <-time.After(time.Second):
					close(received)
				}
			}()
			
			im.sendToLines(tt.text, tt.ok)
			time.Sleep(50 * time.Millisecond)
			
			select {
			case line := <-received:
				if line.OK != tt.expectedOK {
					t.Errorf("Expected OK to be %v, got %v", tt.expectedOK, line.OK)
				}
				if line.Text != tt.text {
					t.Errorf("Expected text %q, got %q", tt.text, line.Text)
				}
			default:
				t.Error("Expected line to be sent to Lines channel")
			}
		})
	}
}

func TestInputManagerSendToLinesChannelFull(t *testing.T) {
	im := NewInputManager(nil)
	
	// Fill the channel to capacity
	for i := 0; i < cap(im.Lines); i++ {
		select {
		case im.Lines <- InputLine{Text: "fill", OK: true}:
		default:
			t.Fatal("Should not reach here during fill")
		}
	}
	
	// Channel is now full, sendToLines should drop the input without blocking
	done := make(chan struct{})
	go func() {
		im.sendToLines("should be dropped", true)
		close(done)
	}()
	
	select {
	case <-done:
		// Good - didn't block on full channel
	case <-time.After(time.Second):
		t.Error("sendToLines should not block when channel is full")
	}
}

func TestInputManagerShowPrompt(t *testing.T) {
	im := NewInputManager(nil)
	
	im.ShowPrompt("Test prompt>")
	
	// Verify prompt was stored
	im.mu.Lock()
	prompt := im.prompt
	im.mu.Unlock()
	
	if prompt != "Test prompt>" {
		t.Errorf("Expected prompt to be 'Test prompt>', got %q", prompt)
	}
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
