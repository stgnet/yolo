package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/term"
)

// ─── Terminal Output ──────────────────────────────────────────────────

// globalUI is set once the split UI is active. Before that, output goes to stdout directly.
var (
	globalUI   *TerminalUI
	ansiCodeRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
)

// rawWrite writes text to stdout, converting lone \n to \r\n for raw terminal mode.
// In raw mode, OPOST is disabled so \n only moves the cursor down without returning
// to column 1. This function ensures proper carriage return + line feed behavior.
func rawWrite(s string) {
	// Normalize \r\n to \n first, then convert all \n to \r\n for raw terminal mode.
	// Leave standalone \r alone — it's a valid cursor-positioning operation
	// (return to column 1) used by overwrite-in-place output.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", "\r\n")
	fmt.Print(s)
}

// rawWriteTo appends text to a builder, converting lone \n to \r\n for raw terminal mode.
func rawWriteTo(buf *strings.Builder, s string) {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", "\r\n")
	buf.WriteString(s)
}

func cprint(color, text string) {
	if globalUI != nil {
		globalUI.OutputPrint(fmt.Sprintf("%s%s%s\n", color, text, Reset))
	} else {
		rawWrite(fmt.Sprintf("%s%s%s\n", color, text, Reset))
	}
}

func cprintNoNL(color, text string) {
	if globalUI != nil {
		globalUI.OutputPrint(fmt.Sprintf("%s%s%s", color, text, Reset))
	} else {
		rawWrite(fmt.Sprintf("%s%s%s", color, text, Reset))
	}
}

// ─── Text Utilities ───────────────────────────────────────────────────

// expandTabs replaces tab characters with spaces aligned to 8-column tab stops.
// startCol is the 1-based column where the text begins (typically ui.outCol).
// This ensures the terminal and the cursor tracker agree on column positions:
// without expansion, the terminal advances to the next tab stop (up to 8 cols)
// while trackCursorMovement would count each tab as 1 column.
func expandTabs(s string, startCol int) string {
	if !strings.Contains(s, "\t") {
		return s
	}
	var buf strings.Builder
	col := startCol
	for _, ch := range s {
		if ch == '\t' {
			spaces := 8 - (col-1)%8
			for i := 0; i < spaces; i++ {
				buf.WriteByte(' ')
			}
			col += spaces
		} else {
			buf.WriteRune(ch)
			if ch == '\n' || ch == '\r' {
				col = 1
			} else {
				col++
			}
		}
	}
	return buf.String()
}

// stripAnsiCodes removes ANSI escape sequences from text for cursor tracking purposes.
// This ensures color codes don't mess up column/row calculations.
func stripAnsiCodes(s string) string {
	return ansiCodeRe.ReplaceAllString(s, "")
}

// breakWordAtVisibleLength breaks a word containing ANSI codes at the specified visible character count.
// It returns (truncatedPart, remainder) ensuring neither part has broken ANSI escape sequences.
func breakWordAtVisibleLength(word string, maxVisibleLen int) (string, string) {
	if len(stripAnsiCodes(word)) <= maxVisibleLen {
		return word, ""
	}

	var truncated strings.Builder
	remaining := word
	var visibleCount int

	for visibleCount < maxVisibleLen && remaining != "" {
		// Check if next character starts an ANSI escape sequence
		if strings.HasPrefix(remaining, "\x1b[") {
			// Find end of the full ANSI sequence (ends with a letter)
			endIdx := -1
			for i := 2; i < len(remaining); i++ {
				b := remaining[i]
				if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') {
					endIdx = i + 1
					break
				}
			}
			if endIdx > 0 {
				truncated.WriteString(remaining[:endIdx])
				remaining = remaining[endIdx:]
				continue
			}
		}

		// Not an ANSI sequence, take one character
		truncated.WriteString(remaining[:1])
		remaining = remaining[1:]
		visibleCount++
	}

	return truncated.String(), remaining
}

// truncateString truncates a string to maxLen runes, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// filterToolActivityMarkers removes [tool activity]...[/tool activity] markers from text
// to avoid confusing the model with its own previous tool call indicators
func filterToolActivityMarkers(text string) string {
	lines := strings.Split(text, "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "[tool activity]") &&
			!strings.HasPrefix(trimmed, "[/tool activity]") {
			filtered = append(filtered, line)
		}
	}
	return strings.Join(filtered, "\n")
}

// ─── Terminal UI (split output/input regions) ─────────────────────────

// TerminalUI manages a split terminal: a scrolling output region on top and
// a fixed input line at the bottom, separated by a divider.
type TerminalUI struct {
	mu             sync.Mutex
	fd             int
	rows           int
	cols           int
	inputBuf       []byte // mirrors InputManager's buffer for redraw
	prompt         string
	outRow         int      // tracked row of cursor in output region
	outCol         int      // tracked col of cursor in output region
	queuedMsgs     []string // messages queued while agent is busy
	scrollEnd      int      // last row of the scroll region (dynamic)
	peakBottomRows int      // high-water mark for bottom area (grow-only)
	streaming      bool     // true while inline output is in progress (prevents scroll region changes)
	pendingWrap    bool     // true when last output char filled the last column (terminal pending wrap state)

	// Rate-limited redraw support
	inputDirty atomic.Bool // set when input changes; cleared by redraw tick
	redrawStop chan struct{}
	redrawDone chan struct{}
}

// wrapText wraps text to the given width, inserting newlines at word boundaries.
// It preserves existing newlines exactly (no adding or doubling) and handles
// words longer than the terminal width by breaking them.
func (ui *TerminalUI) wrapText(text string) string {
	if ui.cols <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var wrappedLines []string

	for _, line := range lines {
		if len(line) == 0 {
			wrappedLines = append(wrappedLines, "")
			continue
		}

		words := strings.Fields(line)
		if len(words) == 0 {
			wrappedLines = append(wrappedLines, "")
			continue
		}

		var current strings.Builder
		currentLen := 0 // visible length (excluding ANSI codes)

		for _, word := range words {
			wordLen := len(stripAnsiCodes(word)) // visible length only

			if currentLen > 0 && currentLen+1+wordLen > ui.cols {
				// Current line is full, start a new one
				wrappedLines = append(wrappedLines, current.String())
				current.Reset()
				currentLen = 0
			}

			if currentLen > 0 {
				current.WriteString(" ")
				currentLen++
			}

			// Break words longer than terminal width (using visible length)
			for len(stripAnsiCodes(word)) > ui.cols && currentLen == 0 {
				// For oversized words, break at visible character boundary while preserving ANSI codes
				truncated, remainder := breakWordAtVisibleLength(word, ui.cols)
				wrappedLines = append(wrappedLines, truncated)
				word = remainder
			}

			current.WriteString(word)
			currentLen += wordLen
		}

		if current.Len() > 0 {
			wrappedLines = append(wrappedLines, current.String())
		}
	}

	return strings.Join(wrappedLines, "\n")
}

func NewTerminalUI() *TerminalUI {
	fd := int(os.Stdout.Fd())
	cols, rows, err := term.GetSize(fd)
	if err != nil {
		rows = 24
		cols = 80
	}
	return &TerminalUI{
		fd:         fd,
		rows:       rows,
		cols:       cols,
		outRow:     1,
		outCol:     1,
		scrollEnd:  rows - 2,
		redrawStop: make(chan struct{}),
		redrawDone: make(chan struct{}),
	}
}

// Setup initializes the scroll region, draws the divider, and starts the
// rate-limited redraw goroutine.
func (ui *TerminalUI) Setup() {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	// Refresh terminal size
	if cols, rows, err := term.GetSize(ui.fd); err == nil {
		ui.rows = rows
		ui.cols = cols
	}

	ui.scrollEnd = ui.rows - 2

	// Build the entire setup sequence in a buffer, flush once
	var buf strings.Builder
	buf.WriteString("\033[2J")                    // clear screen
	fmt.Fprintf(&buf, "\033[1;%dr", ui.scrollEnd) // set scroll region
	ui.writeDividerTo(&buf)
	ui.outRow = 1
	ui.outCol = 1
	fmt.Fprintf(&buf, "\033[%d;%dH", ui.outRow, ui.outCol)
	os.Stdout.WriteString(buf.String())

	// Start rate-limited redraw goroutine
	go ui.redrawLoop()
}

// redrawLoop runs in its own goroutine. It checks the dirty flag at ~60fps
// and redraws the input area only when something changed. This coalesces
// multiple rapid input events (typing, backspace) into a single redraw.
func (ui *TerminalUI) redrawLoop() {
	ticker := time.NewTicker(16 * time.Millisecond) // ~60fps
	defer ticker.Stop()
	defer close(ui.redrawDone)

	for {
		select {
		case <-ui.redrawStop:
			return
		case <-ticker.C:
			if ui.inputDirty.CompareAndSwap(true, false) {
				ui.mu.Lock()
				if ui.streaming {
					ui.renderInputTextOnlyTo()
				} else {
					ui.renderInputFull()
				}
				ui.mu.Unlock()
			}
		}
	}
}

func (ui *TerminalUI) writeDividerTo(buf *strings.Builder) {
	dividerRow := ui.scrollEnd + 1
	divider := strings.Repeat("─", ui.cols)
	fmt.Fprintf(buf, "\033[%d;1H%s%s%s", dividerRow, Gray, divider, Reset)
}

// Teardown restores the full scroll region, resets terminal attributes, and
// stops the redraw goroutine.
func (ui *TerminalUI) Teardown() {
	// Stop the redraw goroutine first (outside the lock)
	close(ui.redrawStop)
	<-ui.redrawDone

	ui.mu.Lock()
	defer ui.mu.Unlock()
	var buf strings.Builder
	buf.WriteString("\033[r")                       // reset scroll region to full terminal
	buf.WriteString("\033[0m")                      // reset all text attributes
	buf.WriteString("\033[?25h")                    // ensure cursor is visible
	fmt.Fprintf(&buf, "\033[%d;1H\n", ui.rows)     // position cursor at bottom
	os.Stdout.WriteString(buf.String())
}

func (ui *TerminalUI) trackCursorMovement(stripped string) {
	// Use deferred (pending) wrapping to match real terminal behavior.
	// When a character is written at the last column, real terminals enter a
	// "pending wrap" state — the cursor stays at the last column and only
	// wraps to the next line when the next printable character arrives.
	// Crucially, rawWrite converts \n to \r\n: if there's a pending wrap,
	// the \r cancels it (cursor stays on the same row) and then \n advances
	// one row — so the total is ONE row advance, not two.  The old code
	// wrapped immediately and then counted \n as a second advance, causing
	// the tracker to drift ahead of the real cursor position.
	pendingWrap := false
	for _, ch := range stripped {
		switch ch {
		case '\n':
			// rawWrite sends \r\n for each \n.  If there's a pending wrap,
			// the \r cancels it (no extra row advance) and \n advances one
			// row.  Either way: one row advance total.
			pendingWrap = false
			ui.outRow++
			ui.outCol = 1
			if ui.outRow > ui.scrollEnd {
				ui.outRow = ui.scrollEnd // scroll region keeps cursor at bottom
			}
		case '\r':
			// Carriage return cancels any pending wrap without advancing.
			pendingWrap = false
			ui.outCol = 1
		default:
			if pendingWrap {
				// Previous character filled the last column; now a printable
				// character forces the deferred wrap to resolve.
				pendingWrap = false
				ui.outCol = 1
				ui.outRow++
				if ui.outRow > ui.scrollEnd {
					ui.outRow = ui.scrollEnd
				}
			}
			ui.outCol++
			if ui.outCol > ui.cols {
				// Don't wrap yet — defer until we see what comes next.
				pendingWrap = true
			}
		}
	}
	// Resolve trailing pending wrap: advance to the next row.
	// When the next row would be beyond scrollEnd, we must NOT eagerly
	// resolve here because the terminal hasn't actually scrolled yet —
	// it's still in "pending wrap" state. An explicit \033[row;colH on
	// the next output call would cancel the pending wrap WITHOUT scrolling,
	// causing new text to overwrite the current line (the "CR without LF" bug).
	// Instead, store the pending wrap so the next output call can force the
	// scroll before repositioning the cursor.
	if pendingWrap {
		if ui.outRow < ui.scrollEnd {
			// Safe to resolve: the next row is within the scroll region,
			// and \033[row+1;1H will correctly position there.
			ui.outCol = 1
			ui.outRow++
		} else {
			// At the bottom of the scroll region. Defer to resolvePendingWrap().
			ui.pendingWrap = true
		}
	}
}

// resolvePendingWrapTo appends the escape sequence that forces the terminal
// to scroll when the last output character filled the final column at the
// bottom of the scroll region.
func (ui *TerminalUI) resolvePendingWrapTo(buf *strings.Builder) {
	if !ui.pendingWrap {
		return
	}
	fmt.Fprintf(buf, "\033[%d;%dH\033D\r", ui.scrollEnd, ui.cols)
	ui.pendingWrap = false
	ui.outRow = ui.scrollEnd
	ui.outCol = 1
}

// OutputPrint writes text to the output (scrolling) region.
// All ANSI escape sequences are buffered and flushed in a single write.
func (ui *TerminalUI) OutputPrint(text string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	text = expandTabs(text, ui.outCol)
	text = ui.wrapText(text)

	var buf strings.Builder
	ui.resolvePendingWrapTo(&buf)
	fmt.Fprintf(&buf, "\033[%d;%dH", ui.outRow, ui.outCol)
	rawWriteTo(&buf, text)

	stripped := stripAnsiCodes(text)
	ui.trackCursorMovement(stripped)

	if !ui.streaming {
		ui.renderInputFullTo(&buf)
	}

	os.Stdout.WriteString(buf.String())
}

// OutputPrintInline writes text without moving cursor back to input line.
// Used for streaming tokens within a single Chat response.
// Call OutputFinishLine when done with a block of inline output.
// The lock is held only for the output write itself; input redraws happen
// asynchronously via the rate-limited redraw goroutine.
func (ui *TerminalUI) OutputPrintInline(text string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	ui.streaming = true

	text = expandTabs(text, ui.outCol)

	var buf strings.Builder
	ui.resolvePendingWrapTo(&buf)
	fmt.Fprintf(&buf, "\033[%d;%dH", ui.outRow, ui.outCol)
	rawWriteTo(&buf, text)

	os.Stdout.WriteString(buf.String())

	stripped := stripAnsiCodes(text)
	ui.trackCursorMovement(stripped)
}

// OutputFinishLine redraws the input line after inline output is done.
func (ui *TerminalUI) OutputFinishLine() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.streaming = false
	ui.renderInputFull()
}

// renderInputFull renders the complete input area (scroll region adjustment,
// divider, queued messages, input line) to a buffer and flushes it atomically.
func (ui *TerminalUI) renderInputFull() {
	var buf strings.Builder
	ui.renderInputFullTo(&buf)
	os.Stdout.WriteString(buf.String())
}

// renderInputFullTo appends the complete input area rendering to buf.
// Caller must hold ui.mu.
func (ui *TerminalUI) renderInputFullTo(buf *strings.Builder) {
	promptStr := ui.prompt
	inputStr := string(ui.inputBuf)
	promptWidth := len(stripAnsiCodes(promptStr))

	totalChars := promptWidth + len(inputStr)
	inputRowCount := 1
	if totalChars > 0 {
		inputRowCount = totalChars/ui.cols + 1
	}

	var queuedDisplayLines []string
	for _, msg := range ui.queuedMsgs {
		msgLines := strings.Split(msg, "\n")
		for j, line := range msgLines {
			prefix := "  [queued] "
			if j > 0 {
				prefix = "           "
			}
			maxLen := ui.cols - len(prefix)
			if maxLen <= 0 {
				maxLen = 1
			}
			for len(line) > maxLen {
				queuedDisplayLines = append(queuedDisplayLines, prefix+line[:maxLen])
				line = line[maxLen:]
				prefix = "           "
			}
			queuedDisplayLines = append(queuedDisplayLines, prefix+line)
		}
	}
	totalQueuedLines := len(queuedDisplayLines)
	neededBottomRows := totalQueuedLines + inputRowCount

	canShrink := len(ui.queuedMsgs) == 0 && len(ui.inputBuf) == 0
	var bottomRows int
	if canShrink {
		bottomRows = 1
		ui.peakBottomRows = 0
	} else {
		if neededBottomRows > ui.peakBottomRows {
			ui.peakBottomRows = neededBottomRows
		}
		bottomRows = ui.peakBottomRows
	}

	maxBottom := ui.rows - 4
	if maxBottom < 1 {
		maxBottom = 1
	}
	if bottomRows > maxBottom {
		bottomRows = maxBottom
		if ui.peakBottomRows > maxBottom {
			ui.peakBottomRows = maxBottom
		}
	}

	displayQueuedLines := totalQueuedLines
	displayInputRows := inputRowCount
	if displayQueuedLines+displayInputRows > bottomRows {
		displayInputRows = bottomRows - displayQueuedLines
		if displayInputRows < 1 {
			displayInputRows = 1
			displayQueuedLines = bottomRows - 1
			if displayQueuedLines < 0 {
				displayQueuedLines = 0
			}
		}
	}

	newScrollEnd := ui.rows - bottomRows - 1
	if newScrollEnd < 1 {
		newScrollEnd = 1
	}

	oldScrollEnd := ui.scrollEnd
	if newScrollEnd > oldScrollEnd {
		for r := oldScrollEnd + 1; r <= newScrollEnd; r++ {
			fmt.Fprintf(buf, "\033[%d;1H\033[2K", r)
		}
	}

	if newScrollEnd != ui.scrollEnd {
		ui.pendingWrap = false
	}
	ui.scrollEnd = newScrollEnd
	fmt.Fprintf(buf, "\033[1;%dr", ui.scrollEnd)
	if ui.outRow > ui.scrollEnd {
		ui.outRow = ui.scrollEnd
	}

	for r := ui.scrollEnd + 1; r <= ui.rows; r++ {
		fmt.Fprintf(buf, "\033[%d;1H\033[2K", r)
	}

	divider := strings.Repeat("─", ui.cols)
	fmt.Fprintf(buf, "\033[%d;1H%s%s%s", ui.scrollEnd+1, Gray, divider, Reset)

	row := ui.scrollEnd + 2
	startIdx := len(queuedDisplayLines) - displayQueuedLines
	if startIdx < 0 {
		startIdx = 0
	}
	for i := startIdx; i < len(queuedDisplayLines); i++ {
		fmt.Fprintf(buf, "\033[%d;1H%s%s%s", row, Gray, queuedDisplayLines[i], Reset)
		row++
	}

	inputStartRow := row
	displayInput := inputStr
	maxChars := displayInputRows*ui.cols - promptWidth
	if maxChars < 0 {
		maxChars = ui.cols
	}
	if len(displayInput) > maxChars {
		displayInput = displayInput[len(displayInput)-maxChars:]
	}
	fmt.Fprintf(buf, "\033[%d;1H%s%s", inputStartRow, promptStr, displayInput)

	dispTotal := promptWidth + len(displayInput)
	cursorRow := inputStartRow + dispTotal/ui.cols
	cursorCol := dispTotal%ui.cols + 1
	fmt.Fprintf(buf, "\033[%d;%dH", cursorRow, cursorCol)
}

// drawInputLocked renders the input area. Used by legacy callers and tests.
// Caller must hold ui.mu.
func (ui *TerminalUI) drawInputLocked() {
	ui.renderInputFull()
}

// UpdateInput updates the UI's copy of the input state and marks input as dirty
// for the rate-limited redraw goroutine.
func (ui *TerminalUI) UpdateInput(prompt string, buf []byte) {
	ui.mu.Lock()
	ui.prompt = prompt
	ui.inputBuf = make([]byte, len(buf))
	copy(ui.inputBuf, buf)
	ui.mu.Unlock()
	ui.inputDirty.Store(true)
}

// RedrawInput marks the input as dirty so the redraw goroutine picks it up.
// For non-streaming callers that need an immediate redraw, it does the redraw
// synchronously.
func (ui *TerminalUI) RedrawInput() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	if ui.streaming {
		// During streaming, just mark dirty — the redraw goroutine will
		// handle it within 16ms without blocking the streaming path.
		ui.inputDirty.Store(true)
		return
	}
	ui.renderInputFull()
}

// renderInputTextOnlyTo renders the input area during streaming without
// changing the scroll region. After drawing, it restores the cursor to the
// output position so streaming can continue. Flushes atomically.
func (ui *TerminalUI) renderInputTextOnlyTo() {
	promptStr := ui.prompt
	inputStr := string(ui.inputBuf)
	promptWidth := len(stripAnsiCodes(promptStr))

	var queuedDisplayLines []string
	for _, msg := range ui.queuedMsgs {
		msgLines := strings.Split(msg, "\n")
		for j, line := range msgLines {
			prefix := "  [queued] "
			if j > 0 {
				prefix = "           "
			}
			maxLen := ui.cols - len(prefix)
			if maxLen <= 0 {
				maxLen = 1
			}
			for len(line) > maxLen {
				queuedDisplayLines = append(queuedDisplayLines, prefix+line[:maxLen])
				line = line[maxLen:]
				prefix = "           "
			}
			queuedDisplayLines = append(queuedDisplayLines, prefix+line)
		}
	}

	bottomRows := ui.rows - ui.scrollEnd - 1
	if bottomRows < 1 {
		bottomRows = 1
	}

	totalChars := promptWidth + len(inputStr)
	inputRowCount := 1
	if totalChars > 0 {
		inputRowCount = totalChars/ui.cols + 1
	}

	displayQueuedLines := len(queuedDisplayLines)
	displayInputRows := inputRowCount
	if displayQueuedLines+displayInputRows > bottomRows {
		displayInputRows = bottomRows - displayQueuedLines
		if displayInputRows < 1 {
			displayInputRows = 1
			displayQueuedLines = bottomRows - 1
			if displayQueuedLines < 0 {
				displayQueuedLines = 0
			}
		}
	}

	var buf strings.Builder

	// Clear everything below divider
	for r := ui.scrollEnd + 2; r <= ui.rows; r++ {
		fmt.Fprintf(&buf, "\033[%d;1H\033[2K", r)
	}

	// Draw queued messages
	row := ui.scrollEnd + 2
	startIdx := len(queuedDisplayLines) - displayQueuedLines
	if startIdx < 0 {
		startIdx = 0
	}
	for i := startIdx; i < len(queuedDisplayLines); i++ {
		fmt.Fprintf(&buf, "\033[%d;1H%s%s%s", row, Gray, queuedDisplayLines[i], Reset)
		row++
	}

	// Draw input
	displayInput := inputStr
	maxChars := displayInputRows*ui.cols - promptWidth
	if maxChars < 0 {
		maxChars = ui.cols
	}
	if len(displayInput) > maxChars {
		displayInput = displayInput[len(displayInput)-maxChars:]
	}
	fmt.Fprintf(&buf, "\033[%d;1H%s%s", row, promptStr, displayInput)

	// Restore cursor to the output streaming position
	fmt.Fprintf(&buf, "\033[%d;%dH", ui.outRow, ui.outCol)

	os.Stdout.WriteString(buf.String())
}

// WriteToInputLine writes directly to the input line area (for character echo).
func (ui *TerminalUI) WriteToInputLine(s string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Print(s)
}

// ClearInputLine clears the entire bottom area (queued messages + input).
func (ui *TerminalUI) ClearInputLine() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	var buf strings.Builder
	for r := ui.scrollEnd + 2; r <= ui.rows; r++ {
		fmt.Fprintf(&buf, "\033[%d;1H\033[2K", r)
	}
	os.Stdout.WriteString(buf.String())
}

// RefreshSize updates terminal size and redraws layout.
func (ui *TerminalUI) RefreshSize() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	if cols, rows, err := term.GetSize(ui.fd); err == nil && (rows != ui.rows || cols != ui.cols) {
		ui.rows = rows
		ui.cols = cols
		if !ui.streaming {
			ui.renderInputFull()
		}
		// If streaming, OutputFinishLine will redraw with updated dimensions
	}
}

// AddQueuedMessage adds a message to the queued display below the divider.
func (ui *TerminalUI) AddQueuedMessage(text string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.queuedMsgs = append(ui.queuedMsgs, text)
	if !ui.streaming {
		ui.renderInputFull()
	}
	// If streaming, OutputFinishLine will redraw with the queued messages
}

// RemoveQueuedMessage removes the oldest queued message from the display.
func (ui *TerminalUI) RemoveQueuedMessage() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	if len(ui.queuedMsgs) > 0 {
		ui.queuedMsgs = ui.queuedMsgs[1:]
		ui.renderInputFull()
	}
}
