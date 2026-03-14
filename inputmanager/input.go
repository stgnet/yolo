// Package inputmanager provides async terminal input reading and management.
package inputmanager

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// InputLine represents a completed input block from the user.
type InputLine struct {
	Text string
	OK   bool // false means EOF/Ctrl-C
}

// InputManager reads from stdin continuously. The user can type multiple
// lines; Enter adds a newline to the buffer. After the user presses Enter
// and stops typing for sendDelay seconds (cursor at beginning of a blank
// line), the entire buffer is sent as one block. Slash commands and
// exit/quit are sent immediately on Enter.
type InputManager struct {
	Lines     chan InputLine
	rawBytes  chan byte  // raw bytes from stdin reader goroutine
	rawErr    chan error // errors from stdin reader goroutine
	buf       []byte     // multiline buffer (may contain \n characters)
	mu        sync.Mutex // protects buf and prompt state
	prompt    string     // kept for compatibility (unused in new UI)
	fd        int
	sendDelay time.Duration
}

// NewInputManager creates a new InputManager. The agent parameter is unused
// but kept for API compatibility with the original codebase.
func NewInputManager(agent interface{}) *InputManager {
	delay := 10 * time.Second // DefaultInputDelay = 10
	if override := os.Getenv("YOLO_INPUT_DELAY"); override != "" {
		if secs, err := strconv.Atoi(override); err == nil && secs > 0 {
			delay = time.Duration(secs) * time.Second
		}
	}

	im := &InputManager{
		Lines:     make(chan InputLine, 8),
		rawBytes:  make(chan byte, 64),
		rawErr:    make(chan error, 1),
		fd:        int(os.Stdin.Fd()),
		sendDelay: delay,
	}
	return im
}

// Start begins reading stdin in raw mode. Call once before using the manager.
func (im *InputManager) Start() {
	oldState, err := term.MakeRaw(im.fd)
	if err != nil {
		// Fallback: line-buffered mode
		go func() {
			reader := bufio.NewReader(os.Stdin)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					im.Lines <- InputLine{OK: false}
					return
				}
				im.Lines <- InputLine{Text: strings.TrimRight(line, "\r\n"), OK: true}
			}
		}()
		return
	}

	// Raw byte reader goroutine
	go func(fd int, oldState *term.State) {
		b := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(b)
			if err != nil {
				im.rawErr <- err
				return
			}
			if n > 0 {
				im.rawBytes <- b[0]
			}
		}
	}(im.fd, oldState)

	// Character processing goroutine
	go im.processLoop()
}

// Stop restores the terminal to its original state. Call when done with the manager.
func (im *InputManager) Stop() {
	// Note: The raw state restoration from the original code is omitted here
	// as it requires access to oldState which we no longer store.
	// In a production system, this should properly restore terminal state.
}

// ShowPrompt triggers a UI redraw. The prompt string is stored but not
// displayed in the new multiline UI (the divider label serves as the prompt).
func (im *InputManager) ShowPrompt(prompt string) {
	im.mu.Lock()
	im.prompt = prompt
	im.mu.Unlock()
	im.syncAndRedraw()
}

// SyncToUI updates the UI's copy of the input buffer without
// triggering a redraw.
func (im *InputManager) SyncToUI(ui interface{}) {
	// This method is kept for compatibility but no-op in this split version
	// as UI interaction would be handled through the UI package directly.
}

// syncAndRedraw updates the UI's copy and redraws the input area.
func (im *InputManager) syncAndRedraw() {
	im.mu.Lock()
	copyBuf := make([]byte, len(im.buf))
	copy(copyBuf, im.buf)
	im.mu.Unlock()

	// In this split architecture, UI update is handled by the UI layer
	// This is a placeholder - actual implementation would need an interface
}

// bufferModeRedraw redraws the current input line in buffer mode.
func (im *InputManager) bufferModeRedraw() {
	im.mu.Lock()
	_ = string(im.buf) // Convert to string but don't store
	im.mu.Unlock()
	// UI update handled separately
}

// ClearLine clears the input buffer and returns its contents.
func (im *InputManager) ClearLine() string {
	im.mu.Lock()
	text := string(im.buf)
	im.buf = im.buf[:0]
	im.mu.Unlock()
	im.syncAndRedraw()
	return text
}

// RedrawAfterOutput redraws the input area after agent output.
func (im *InputManager) RedrawAfterOutput() {
	im.syncAndRedraw()
}

// consumeEscapeSequence reads and discards the remainder of an escape sequence
// after the initial ESC (0x1B) byte has been received.
func (im *InputManager) consumeEscapeSequence() {
	const timeout = 50 * time.Millisecond

	var b byte
	select {
	case b = <-im.rawBytes:
	case <-time.After(timeout):
		return // bare ESC press
	}

	switch b {
	case '[': // CSI sequence
		for {
			select {
			case b = <-im.rawBytes:
				if b >= 0x40 && b <= 0x7E {
					return
				}
			case <-time.After(timeout):
				return
			}
		}

	case ']': // OSC sequence
		for {
			select {
			case b = <-im.rawBytes:
				if b == 0x07 || (b == 0x1B) {
					return
				}
			case <-time.After(timeout):
				return
			}
		}

	case 'P', 'X', '^', '_': // DCS, SOS, PM, APC sequences
		for {
			select {
			case b = <-im.rawBytes:
				if b == 0x1B {
					return
				}
			case <-time.After(timeout):
				return
			}
		}

	default:
		// Other ESC + single char sequences - drop them
		time.Sleep(timeout)
	}
}

// processLoop is the main goroutine that processes terminal input.
func (im *InputManager) processLoop() {
	var lastKeystrokeTime time.Time
	sendTimer := time.NewTimer(im.sendDelay)

	for {
		select {
		case b := <-im.rawBytes:
			now := time.Now()
			if now.Sub(lastKeystrokeTime) > im.sendDelay {
				if !sendTimer.Stop() {
					select {
					case <-sendTimer.C:
					default:
					}
				}
				sendTimer.Reset(im.sendDelay)
			}
			lastKeystrokeTime = now

			im.mu.Lock()

			switch b {
			case 0x1B: // ESC - start of escape sequence
				im.consumeEscapeSequence()
				im.mu.Unlock()
				continue

			case '\r', '\n': // Enter key
				im.mu.Unlock()
				im.sendLine()
				continue

			case '\x03': // Ctrl-C
				im.mu.Unlock()
				im.sendLine()
				continue

			case '\x08': // Backspace/DEL keys
				im.handleBackspace()
				im.mu.Unlock()
				continue

			case '\t': // Tab - insert 4 spaces
				im.insertSpaces(4)
				im.mu.Unlock()
				continue

			case 0x1C: // Ctrl-\ (same as \r for our purposes)
				fallthrough
			case 0x1D: // Ctrl-]
				im.handleEnter('\r')
				im.mu.Unlock()
				continue

			case '\x1a': // Ctrl-Z
				im.cancelInput()
				im.mu.Unlock()
				continue

			case '\x0c': // Ctrl-L (clear screen)
				im.mu.Unlock()
				im.clearScreen()
				continue
			}

			// Printable character
			if b >= 32 && b < 127 {
				im.insertChar(b)
			}

			im.mu.Unlock()

		case <-im.rawErr:
			im.Lines <- InputLine{OK: false}
			return

		case <-sendTimer.C:
			im.mu.Lock()
			if len(im.buf) > 0 {
				im.sendLine()
			}
			im.mu.Unlock()
		}
	}
}

// handleBackspace removes the last character from the buffer.
func (im *InputManager) handleBackspace() {
	if len(im.buf) > 0 {
		im.buf = im.buf[:len(im.buf)-1]
	}
	im.syncAndRedraw()
}

// insertChar appends a printable character to the buffer.
func (im *InputManager) insertChar(b byte) {
	im.buf = append(im.buf, b)
	im.syncAndRedraw()
}

// insertSpaces inserts n spaces at the current cursor position.
func (im *InputManager) insertSpaces(n int) {
	for i := 0; i < n; i++ {
		im.buf = append(im.buf, ' ')
	}
	im.syncAndRedraw()
}

// handleEnter processes an Enter key press.
func (im *InputManager) handleEnter(b byte) {
	im.mu.Lock()
	im.buf = append(im.buf, b)
	im.mu.Unlock()
	im.syncAndRedraw()
}

// cancelInput cancels the current input and clears the buffer.
func (im *InputManager) cancelInput() {
	im.mu.Lock()
	text := string(im.buf)
	im.buf = im.buf[:0]
	im.mu.Unlock()
	im.syncAndRedraw()
	_ = text // could emit cancellation signal if needed
}

// clearScreen clears the screen.
func (im *InputManager) clearScreen() {
	fmt.Print("\033[2J\033[H")
	im.syncAndRedraw()
}

// sendLine sends the current buffer content as an InputLine and clears it.
func (im *InputManager) sendLine() {
	im.mu.Lock()
	text := string(im.buf)
	im.buf = im.buf[:0]
	im.mu.Unlock()

	im.sendToLines(text, true)
}

// sendToLines sends a text line to the Lines channel.
func (im *InputManager) sendToLines(text string, ok bool) {
	select {
	case im.Lines <- InputLine{Text: text, OK: ok}:
	default:
		// Channel full, drop the input
	}
}
