package terminalui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// BufferUI manages output buffering for the linear (non-terminal) display mode.
// In buffer mode, output streams linearly to stdout without scroll regions or
// screen redraws. When the user presses a printable key, the current line of
// agent output finishes and subsequent output is buffered until the user
// finishes their input.
type BufferUI struct {
	mu             sync.Mutex
	userWantsInput bool            // set on first printable keypress
	buffering      bool            // true once transition to input mode is complete
	buffer         strings.Builder // agent output accumulated while user types
	promptShown    bool            // true once input prompt is ready
	promptReady    chan struct{}   // closed when prompt becomes ready
	midLine        bool            // true if last Write did not end with \n
	pastFirstLine  bool            // true after first Enter in current input session
	prevInputLines int             // number of display lines used by previous redraw
}

// NewBufferUI creates a new buffer-mode output manager.
func NewBufferUI() *BufferUI {
	ch := make(chan struct{})
	close(ch) // initially "ready" (no-op state)
	return &BufferUI{
		promptReady: ch,
	}
}

// Write outputs text, respecting the current buffering state.
// Agent/subagent output flows through this method.
func (b *BufferUI) Write(text string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.buffering {
		b.buffer.WriteString(text)
		return
	}

	if !b.userWantsInput {
		rawWrite(text)
		if len(text) > 0 {
			b.midLine = !strings.HasSuffix(text, "\n")
		}
		return
	}

	// User wants input — finish current line, then buffer the rest
	idx := strings.Index(text, "\n")
	if idx >= 0 {
		rawWrite(text[:idx+1])
		b.midLine = false
		b.buffering = true
		if idx+1 < len(text) {
			b.buffer.WriteString(text[idx+1:])
		}
		b.showPromptLocked()
	} else {
		// No newline yet — keep outputting the current line
		rawWrite(text)
		b.midLine = true
	}
}

// NotifyKeypress is called when the user presses a printable key.
// It triggers the transition from streaming output to user input mode.
func (b *BufferUI) NotifyKeypress() {
	b.mu.Lock()
	if b.userWantsInput {
		b.mu.Unlock()
		return
	}
	b.userWantsInput = true
	b.promptReady = make(chan struct{})

	if !b.midLine {
		// No partial output line — immediately enter input mode
		b.buffering = true
		b.showPromptLocked()
		b.mu.Unlock()
		return
	}
	b.mu.Unlock()

	// Output is mid-line — wait for the \n to arrive or timeout
	go func() {
		time.Sleep(200 * time.Millisecond)
		b.mu.Lock()
		if b.userWantsInput && !b.promptShown {
			b.buffering = true
			b.showPromptLocked()
		}
		b.mu.Unlock()
	}()
}

// PromptReady returns a channel that closes when the input prompt is ready.
func (b *BufferUI) PromptReady() <-chan struct{} {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.promptReady
}

// IsPromptShown returns whether the prompt is currently displayed.
func (b *BufferUI) IsPromptShown() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.promptShown
}

// IsUserTyping returns whether the user is in input mode.
func (b *BufferUI) IsUserTyping() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.userWantsInput
}

func (b *BufferUI) showPromptLocked() {
	if b.promptShown {
		return
	}
	b.promptShown = true
	if b.midLine {
		rawWrite("\r\n")
		b.midLine = false
	}
	close(b.promptReady)
}

// RedrawPrompt redraws the "you> " prompt with the current input line.
func (b *BufferUI) RedrawPrompt(inputBuf string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.promptShown {
		return
	}

	lastNL := strings.LastIndex(inputBuf, "\n")
	var lastLine string
	if lastNL >= 0 {
		lastLine = inputBuf[lastNL+1:]
	} else {
		lastLine = inputBuf
	}

	cols := 80
	if c, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && c > 0 {
		cols = c
	}

	var prefix string
	if !b.pastFirstLine {
		prefix = "you> "
	}
	fullLine := prefix + lastLine

	var displayLines []string
	for len(fullLine) > cols {
		displayLines = append(displayLines, fullLine[:cols])
		fullLine = fullLine[cols:]
	}
	displayLines = append(displayLines, fullLine)

	var buf strings.Builder
	if b.prevInputLines > 1 {
		fmt.Fprintf(&buf, "\033[%dA", b.prevInputLines-1)
	}

	for i, dl := range displayLines {
		if i > 0 {
			buf.WriteString("\r\n")
		}
		fmt.Fprintf(&buf, "\r\033[K%s%s%s", Green, dl, Reset)
	}

	for i := len(displayLines); i < b.prevInputLines; i++ {
		buf.WriteString("\r\n\033[K")
	}

	if extra := b.prevInputLines - len(displayLines); extra > 0 {
		fmt.Fprintf(&buf, "\033[%dA", extra)
	}

	b.prevInputLines = len(displayLines)
	rawWrite(buf.String())
}

// EnterKey outputs a carriage-return + line-feed for the Enter key in buffer mode.
func (b *BufferUI) EnterKey() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.promptShown {
		if b.prevInputLines > 1 {
			rawWrite(fmt.Sprintf("\033[%dB", b.prevInputLines-1))
		}
		rawWrite("\r\n")
		b.pastFirstLine = true
		b.prevInputLines = 0
	}
}

// FlushBuffer outputs all buffered agent output and resets to streaming mode.
func (b *BufferUI) FlushBuffer() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.buffer.Len() > 0 {
		rawWrite(Reset)
		rawWrite(b.buffer.String())
		b.buffer.Reset()
	}
	b.userWantsInput = false
	b.buffering = false
	b.promptShown = false
	b.pastFirstLine = false
}

// CancelInput resets buffer state and flushes pending output (e.g., on Ctrl-C).
func (b *BufferUI) CancelInput() {
	b.FlushBuffer()
}

// rawWrite writes text to stdout, converting lone \n to \r\n for raw terminal mode.
func rawWrite(s string) {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", "\r\n")
	fmt.Print(s)
}
