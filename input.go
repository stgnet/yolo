package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/term"
)

// ─── Input Manager (async multiline input) ─────────────────────────────

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
	Lines    chan InputLine
	rawBytes chan byte  // raw bytes from stdin reader goroutine
	rawErr   chan error // errors from stdin reader goroutine
	buf      []byte     // multiline buffer (may contain \n characters)
	mu       sync.Mutex // protects buf and prompt state
	prompt   string     // kept for compatibility (unused in new UI)
	agent    *YoloAgent
	oldState *term.State
	fd       int
	sendDelay time.Duration
}

func NewInputManager(agent *YoloAgent) *InputManager {
	delay := time.Duration(DefaultInputDelay) * time.Second
	if override := os.Getenv("YOLO_INPUT_DELAY"); override != "" {
		if secs, err := strconv.Atoi(override); err == nil && secs > 0 {
			delay = time.Duration(secs) * time.Second
		}
	}

	im := &InputManager{
		Lines:     make(chan InputLine, 8),
		rawBytes:  make(chan byte, 64),
		rawErr:    make(chan error, 1),
		agent:     agent,
		fd:        int(os.Stdin.Fd()),
		sendDelay: delay,
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

// ShowPrompt triggers a UI redraw. The prompt string is stored but not
// displayed in the new multiline UI (the divider label serves as the prompt).
func (im *InputManager) ShowPrompt(prompt string) {
	im.mu.Lock()
	im.prompt = prompt
	im.mu.Unlock()
	im.syncAndRedraw()
}

// SyncToUI updates the TerminalUI's copy of the input buffer without
// triggering a redraw.
func (im *InputManager) SyncToUI() {
	im.mu.Lock()
	prompt := im.prompt
	buf := make([]byte, len(im.buf))
	copy(buf, im.buf)
	im.mu.Unlock()

	if globalUI != nil {
		globalUI.UpdateInput(prompt, buf)
	}
}

// syncAndRedraw updates the TerminalUI's copy and redraws the input area.
// In buffer mode it redraws the inline "you> " prompt instead.
func (im *InputManager) syncAndRedraw() {
	if bufferUI != nil && globalUI == nil {
		im.bufferModeRedraw()
		return
	}

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

// bufferModeRedraw redraws the current input line in buffer mode.
// It writes "you> <last-line>" using \r\033[K to overwrite the current line.
func (im *InputManager) bufferModeRedraw() {
	if bufferUI == nil {
		return
	}
	im.mu.Lock()
	buf := string(im.buf)
	im.mu.Unlock()

	bufferUI.RedrawPrompt(buf)
}

// ClearLine clears the input buffer.
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
	case 'O': // SS3 sequence
		select {
		case <-im.rawBytes:
		case <-time.After(timeout):
		}
	default:
		// Simple ESC + single char
	}
}

// sendBuffer grabs the current buffer contents, clears the buffer, redraws,
// and sends the text to the Lines channel. Called when the send timer fires
// or for immediate-send commands.
func (im *InputManager) sendBuffer() {
	im.mu.Lock()
	text := string(im.buf)
	im.buf = im.buf[:0]
	im.mu.Unlock()

	// Trim trailing newlines but preserve internal ones
	text = strings.TrimRight(text, "\n")

	// In buffer mode, flush buffered agent output before sending
	if bufferUI != nil && globalUI == nil {
		bufferUI.FlushBuffer()
	}

	if strings.TrimSpace(text) != "" {
		im.syncAndRedraw()
		im.Lines <- InputLine{Text: text, OK: true}
	} else {
		im.syncAndRedraw()
	}
}

func (im *InputManager) processLoop() {
	sendTimer := time.NewTimer(0)
	if !sendTimer.Stop() {
		<-sendTimer.C
	}

	// inBufferMode returns true when the linear buffer UI is active.
	inBufferMode := func() bool { return bufferUI != nil && globalUI == nil }

	for {
		select {
		case ch := <-im.rawBytes:
			im.mu.Lock()
			switch {
			case ch == '\r' || ch == '\n': // Enter
				// Check if this is a single-line command that should send immediately
				trimmed := strings.TrimSpace(string(im.buf))
				hasNewline := strings.Contains(string(im.buf), "\n")
				isImmediateCmd := !hasNewline && trimmed != "" &&
					(strings.HasPrefix(trimmed, "/") ||
						strings.ToLower(trimmed) == "exit" ||
						strings.ToLower(trimmed) == "quit")

				if isImmediateCmd {
					// Immediate send for slash commands and exit/quit
					text := string(im.buf)
					im.buf = im.buf[:0]
					im.mu.Unlock()
					sendTimer.Stop()
					if inBufferMode() {
						bufferUI.EnterKey()
						bufferUI.FlushBuffer()
					}
					im.syncAndRedraw()
					im.Lines <- InputLine{Text: text, OK: true}
				} else {
					// Add newline to buffer and start/reset the send timer
					im.buf = append(im.buf, '\n')
					im.mu.Unlock()
					sendTimer.Reset(im.sendDelay)
					if inBufferMode() {
						bufferUI.EnterKey()
					}
					im.syncAndRedraw()
				}

			case ch == 127 || ch == 8: // Backspace
				if len(im.buf) > 0 {
					// Remove last rune (handles UTF-8 properly)
					_, size := utf8.DecodeLastRune(im.buf)
					if size > 0 {
						im.buf = im.buf[:len(im.buf)-size]
					} else {
						im.buf = im.buf[:len(im.buf)-1]
					}
				}
				// Cancel send timer — user is still editing
				sendTimer.Stop()
				im.mu.Unlock()
				im.syncAndRedraw()

			case ch == 3: // Ctrl-C
				im.buf = im.buf[:0]
				im.mu.Unlock()
				sendTimer.Stop()
				if inBufferMode() {
					bufferUI.CancelInput()
				}
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

			case ch == 21: // Ctrl-U (kill entire buffer)
				im.buf = im.buf[:0]
				im.mu.Unlock()
				sendTimer.Stop()
				im.syncAndRedraw()

			case ch == 23: // Ctrl-W (kill word, don't cross newlines)
				for len(im.buf) > 0 && im.buf[len(im.buf)-1] == ' ' {
					im.buf = im.buf[:len(im.buf)-1]
				}
				for len(im.buf) > 0 && im.buf[len(im.buf)-1] != ' ' && im.buf[len(im.buf)-1] != '\n' {
					im.buf = im.buf[:len(im.buf)-1]
				}
				im.mu.Unlock()
				sendTimer.Stop()
				im.syncAndRedraw()

			case ch == 27: // Escape sequence
				im.mu.Unlock()
				im.consumeEscapeSequence()

			default:
				// Cancel send timer — user is typing
				sendTimer.Stop()

				// Buffer mode: on first printable key, notify bufferUI
				// to transition from streaming output to user input.
				needsNotify := inBufferMode() && !bufferUI.IsUserTyping()

				if ch >= 0xC0 && ch < 0xFE {
					// UTF-8 leading byte: collect continuation bytes
					size := utf8ByteLen(ch)
					utfBuf := []byte{ch}
					im.mu.Unlock()
					for i := 1; i < size; i++ {
						select {
						case cb := <-im.rawBytes:
							if cb&0xC0 == 0x80 {
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
					if needsNotify {
						bufferUI.NotifyKeypress()
						go func() {
							<-bufferUI.PromptReady()
							im.bufferModeRedraw()
						}()
					}
					im.syncAndRedraw()
				} else if ch >= 32 && ch < 0x80 && unicode.IsPrint(rune(ch)) {
					// Slash at column 0 with preceding text: separate
					// the earlier text as user input so the slash command
					// is processed independently.
					if ch == '/' && len(im.buf) > 0 && im.buf[len(im.buf)-1] == '\n' {
						precedingText := strings.TrimRight(string(im.buf), "\n")
						im.buf = im.buf[:0]
						im.buf = append(im.buf, '/')
						im.mu.Unlock()
						if strings.TrimSpace(precedingText) != "" {
							im.Lines <- InputLine{Text: precedingText, OK: true}
						}
						if needsNotify {
							bufferUI.NotifyKeypress()
							go func() {
								<-bufferUI.PromptReady()
								im.bufferModeRedraw()
							}()
						}
						im.syncAndRedraw()
					} else {
						im.buf = append(im.buf, ch)
						im.mu.Unlock()
						if needsNotify {
							bufferUI.NotifyKeypress()
							go func() {
								<-bufferUI.PromptReady()
								im.bufferModeRedraw()
							}()
						}
						im.syncAndRedraw()
					}
				} else {
					im.mu.Unlock()
					im.syncAndRedraw()
				}
			}

		case <-sendTimer.C:
			// Send timer expired — deliver the entire buffer as one block
			im.sendBuffer()

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
