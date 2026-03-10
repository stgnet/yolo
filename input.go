package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/term"
)

// ─── Input Manager (async input) ─────────────────────────────────────

// InputLine represents a completed line from the user.
type InputLine struct {
	Text string
	OK   bool // false means EOF/Ctrl-C
}

// InputManager reads from stdin continuously, allowing the user to type
// even while the agent is processing. Completed lines are sent to Lines.
type InputManager struct {
	Lines    chan InputLine
	rawBytes chan byte  // raw bytes from stdin reader goroutine
	rawErr   chan error // errors from stdin reader goroutine
	buf      []byte     // current line being edited
	mu       sync.Mutex // protects buf and prompt state
	prompt   string     // current prompt prefix being displayed
	agent    *YoloAgent
	oldState *term.State
	fd       int
}

func NewInputManager(agent *YoloAgent) *InputManager {
	im := &InputManager{
		Lines:    make(chan InputLine, 8),
		rawBytes: make(chan byte, 64),
		rawErr:   make(chan error, 1),
		agent:    agent,
		fd:       int(os.Stdin.Fd()),
	}
	return im
}

// Start begins reading stdin in raw mode. Call once.
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
	im.oldState = oldState

	// Raw byte reader goroutine
	go func() {
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
	}()

	// Character processing goroutine
	go im.processLoop()
}

// Stop restores the terminal.
func (im *InputManager) Stop() {
	if im.oldState != nil {
		term.Restore(im.fd, im.oldState)
	}
}

// ShowPrompt displays a prompt and enables editing. Call from the main goroutine
// or when ready for input. The prompt is redisplayed after agent output.
func (im *InputManager) ShowPrompt(prompt string) {
	im.mu.Lock()
	im.prompt = prompt
	im.mu.Unlock()
	im.syncAndRedraw()
}

// syncAndRedraw updates the TerminalUI's copy and redraws the input line.
func (im *InputManager) syncAndRedraw() {
	im.mu.Lock()
	prompt := im.prompt
	buf := make([]byte, len(im.buf))
	copy(buf, im.buf)
	im.mu.Unlock()

	if globalUI != nil {
		globalUI.UpdateInput(prompt, buf)
		globalUI.RedrawInput()
	} else {
		fmt.Printf("\r\033[K%s%s", prompt, string(buf))
	}
}

// ClearLine clears the input buffer but keeps the prompt visible.
func (im *InputManager) ClearLine() string {
	im.mu.Lock()
	text := string(im.buf)
	im.buf = im.buf[:0]
	im.mu.Unlock()
	// Redraw with empty buffer so the prompt stays visible
	im.syncAndRedraw()
	return text
}

// RedrawAfterOutput redraws the prompt and current buffer after agent output.
func (im *InputManager) RedrawAfterOutput() {
	im.syncAndRedraw()
}

// consumeEscapeSequence reads and discards the remainder of an escape sequence
// after the initial ESC (0x1B) byte has been received. It handles:
//   - CSI sequences: ESC [ <params> <letter>  (arrow keys, function keys, etc.)
//   - SS3 sequences: ESC O <letter>           (some function keys)
//   - Simple sequences: ESC <letter>           (Alt+key combos)
//
// The previous implementation consumed a fixed 2 bytes, which was wrong for
// variable-length CSI sequences (e.g., ESC[1;5C for Ctrl+Right is 6 bytes).
// Leftover bytes would leak into the input buffer as garbage, and real user
// keystrokes could be consumed as part of the sequence, causing input loss.
func (im *InputManager) consumeEscapeSequence() {
	const timeout = 50 * time.Millisecond

	// Read the first byte after ESC
	var b byte
	select {
	case b = <-im.rawBytes:
	case <-time.After(timeout):
		return // bare ESC press, nothing to consume
	}

	switch b {
	case '[': // CSI sequence: ESC [ <params...> <final byte>
		// Parameters are bytes in range 0x20-0x3F (digits, semicolons, etc.)
		// Final byte is in range 0x40-0x7E (letters, @, ~, etc.)
		for {
			select {
			case b = <-im.rawBytes:
				if b >= 0x40 && b <= 0x7E {
					return // final byte reached, sequence complete
				}
				// intermediate/parameter byte, keep consuming
			case <-time.After(timeout):
				return // incomplete sequence, give up
			}
		}
	case 'O': // SS3 sequence: ESC O <letter>
		select {
		case <-im.rawBytes: // consume the final byte
		case <-time.After(timeout):
		}
	default:
		// Simple ESC + single char (e.g., Alt+key), already consumed
	}
}

func (im *InputManager) processLoop() {
	for {
		select {
		case ch := <-im.rawBytes:
			im.mu.Lock()
			switch {
			case ch == '\r' || ch == '\n': // Enter
				line := string(im.buf)
				im.buf = im.buf[:0]
				im.mu.Unlock()
				// Show a queued indicator if the agent is busy
				im.agent.mu.Lock()
				busy := im.agent.busy
				im.agent.mu.Unlock()
				trimmed := strings.TrimSpace(line)
				if busy && trimmed != "" && globalUI != nil {
					globalUI.AddQueuedMessage(trimmed)
				}
				// Clear input line after submit
				im.syncAndRedraw()
				im.Lines <- InputLine{Text: line, OK: true}
			case ch == 127 || ch == 8: // Backspace
				if len(im.buf) > 0 {
					im.buf = im.buf[:len(im.buf)-1]
				}
				im.mu.Unlock()
				im.syncAndRedraw()
			case ch == 3: // Ctrl-C
				im.buf = im.buf[:0]
				im.mu.Unlock()
				im.syncAndRedraw()
				// If agent is busy, cancel the current chat
				im.agent.mu.Lock()
				cancel := im.agent.cancelChat
				im.agent.mu.Unlock()
				if cancel != nil {
					cancel()
				} else {
					im.Lines <- InputLine{OK: false}
				}
			case ch == 4: // Ctrl-D
				if len(im.buf) == 0 {
					im.mu.Unlock()
					im.Lines <- InputLine{OK: false}
				} else {
					im.mu.Unlock()
				}
			case ch == 21: // Ctrl-U (kill line)
				im.buf = im.buf[:0]
				im.mu.Unlock()
				im.syncAndRedraw()
			case ch == 23: // Ctrl-W (kill word)
				for len(im.buf) > 0 && im.buf[len(im.buf)-1] == ' ' {
					im.buf = im.buf[:len(im.buf)-1]
				}
				for len(im.buf) > 0 && im.buf[len(im.buf)-1] != ' ' {
					im.buf = im.buf[:len(im.buf)-1]
				}
				im.mu.Unlock()
				im.syncAndRedraw()
			case ch == 27: // Escape sequence
				im.mu.Unlock()
				im.consumeEscapeSequence()
			default:
				if ch >= 0xC0 && ch < 0xFE {
					// UTF-8 leading byte: collect continuation bytes
					size := utf8ByteLen(ch)
					utfBuf := []byte{ch}
					im.mu.Unlock()
					for i := 1; i < size; i++ {
						select {
						case cb := <-im.rawBytes:
							if cb&0xC0 == 0x80 { // valid continuation byte
								utfBuf = append(utfBuf, cb)
							} else {
								utfBuf = nil
								break
							}
						case <-time.After(50 * time.Millisecond):
							utfBuf = nil
							break
						}
						if utfBuf == nil {
							break
						}
					}
					if utfBuf != nil {
						if r, _ := utf8.DecodeRune(utfBuf); r != utf8.RuneError && unicode.IsPrint(r) {
							im.mu.Lock()
							im.buf = append(im.buf, utfBuf...)
							im.mu.Unlock()
						}
					}
					im.syncAndRedraw()
				} else {
					if ch >= 32 && ch < 0x80 && unicode.IsPrint(rune(ch)) {
						im.buf = append(im.buf, ch)
					}
					im.mu.Unlock()
					im.syncAndRedraw()
				}
			}

		case <-im.rawErr:
			im.Lines <- InputLine{OK: false}
			return
		}
	}
}

// utf8ByteLen returns the expected byte length of a UTF-8 sequence from its leading byte.
func utf8ByteLen(lead byte) int {
	switch {
	case lead < 0xE0:
		return 2
	case lead < 0xF0:
		return 3
	default:
		return 4
	}
}
