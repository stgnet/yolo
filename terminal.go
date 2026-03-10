package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/term"
)

// ─── Terminal Output ──────────────────────────────────────────────────

// globalUI is set once the split UI is active. Before that, output goes to stdout directly.
var (
	globalUI    *TerminalUI
	ansiCodeRe  = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
)

// rawWrite writes text to stdout, converting lone \n to \r\n for raw terminal mode.
// In raw mode, OPOST is disabled so \n only moves the cursor down without returning
// to column 1. This function ensures proper carriage return + line feed behavior.
func rawWrite(s string) {
	// Normalize \r\n to \n first, then convert all \n to \r\n for raw terminal mode.
	// Leave standalone \r alone — it's a valid cursor-positioning operation
	// (return to column 1) used by the spinner and other overwrite-in-place output.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", "\r\n")
	fmt.Print(s)
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
	outRow         int // tracked row of cursor in output region
	outCol         int // tracked col of cursor in output region
	queuedMsgs     []string // messages queued while agent is busy
	scrollEnd      int      // last row of the scroll region (dynamic)
	peakBottomRows int      // high-water mark for bottom area (grow-only)
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
		fd:        fd,
		rows:      rows,
		cols:      cols,
		outRow:    1,
		outCol:    1,
		scrollEnd: rows - 2,
	}
}

// Setup initializes the scroll region and draws the divider.
func (ui *TerminalUI) Setup() {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	// Refresh terminal size
	if cols, rows, err := term.GetSize(ui.fd); err == nil {
		ui.rows = rows
		ui.cols = cols
	}

	ui.scrollEnd = ui.rows - 2
	// Clear screen, set scroll region, draw divider
	fmt.Print("\033[2J")
	fmt.Printf("\033[1;%dr", ui.scrollEnd)
	ui.drawDividerLocked()
	ui.outRow = 1
	ui.outCol = 1
	fmt.Printf("\033[%d;%dH", ui.outRow, ui.outCol)
}

func (ui *TerminalUI) drawDividerLocked() {
	dividerRow := ui.scrollEnd + 1
	divider := strings.Repeat("─", ui.cols)
	fmt.Printf("\033[%d;1H%s%s%s", dividerRow, Gray, divider, Reset)
}

// Teardown restores the full scroll region.
func (ui *TerminalUI) Teardown() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("\033[1;%dr", ui.rows)
	fmt.Printf("\033[%d;1H\n", ui.rows)
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
	// Resolve any trailing pending wrap so that (outRow, outCol) is a valid
	// position for the explicit ANSI cursor-positioning on the next call.
	// The next OutputPrint/OutputPrintInline will \033[row;colH to this
	// position, which cancels the terminal's pending wrap and moves there.
	if pendingWrap {
		ui.outCol = 1
		ui.outRow++
		if ui.outRow > ui.scrollEnd {
			ui.outRow = ui.scrollEnd
		}
	}
}

// OutputPrint writes text to the output (scrolling) region.
func (ui *TerminalUI) OutputPrint(text string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	// Wrap text to terminal width before outputting
	text = ui.wrapText(text)

	// Move cursor to tracked output position within the scroll region
	fmt.Printf("\033[%d;%dH", ui.outRow, ui.outCol)
	// Write the text (rawWrite converts \n to \r\n for raw terminal mode)
	rawWrite(text)
	// Track where the cursor ended up.
	// Strip ANSI codes before counting to avoid off-by-several errors from color codes.
	stripped := stripAnsiCodes(text)
	ui.trackCursorMovement(stripped)
	// Redraw input line (output may have scrolled and clobbered it)
	ui.drawInputLocked()
}

// OutputPrintInline writes text without moving cursor back to input line.
// Used for streaming tokens within a single Chat response.
// Call OutputFinishLine when done with a block of inline output.
func (ui *TerminalUI) OutputPrintInline(text string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	// Do NOT wrap streaming tokens — they are fragments, not complete lines.
	// The terminal handles character-level wrapping, and trackCursorMovement
	// already accounts for it.

	fmt.Printf("\033[%d;%dH", ui.outRow, ui.outCol)
	// rawWrite converts \n to \r\n for raw terminal mode
	rawWrite(text)
	// Strip ANSI codes before counting to avoid off-by-several errors from color codes.
	stripped := stripAnsiCodes(text)
	ui.trackCursorMovement(stripped)
}

// OutputFinishLine redraws the input line after inline output is done.
func (ui *TerminalUI) OutputFinishLine() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.drawInputLocked()
}

func (ui *TerminalUI) drawInputLocked() {
	promptStr := ui.prompt
	inputStr := string(ui.inputBuf)
	promptWidth := len(stripAnsiCodes(promptStr))

	// Calculate how many rows the wrapped input occupies.
	// When the cursor is at the end and totalChars is a multiple of cols,
	// it wraps to the next row, so we need floor(totalChars/cols)+1.
	totalChars := promptWidth + len(inputStr)
	inputRowCount := 1
	if totalChars > 0 {
		inputRowCount = totalChars/ui.cols + 1
	}

	// Flatten queued messages into individual display lines (handles multi-line messages).
	// First line of each message gets "  [queued] " prefix, continuation lines get aligned spaces.
	// Long lines are wrapped (not truncated) so the full message is visible.
	var queuedDisplayLines []string
	for _, msg := range ui.queuedMsgs {
		msgLines := strings.Split(msg, "\n")
		for j, line := range msgLines {
			prefix := "  [queued] "
			if j > 0 {
				prefix = "           " // 11 chars, aligned with prefix above
			}
			maxLen := ui.cols - len(prefix)
			if maxLen <= 0 {
				maxLen = 1
			}
			// Wrap long lines instead of truncating
			for len(line) > maxLen {
				queuedDisplayLines = append(queuedDisplayLines, prefix+line[:maxLen])
				line = line[maxLen:]
				prefix = "           " // continuation lines use aligned spaces
			}
			queuedDisplayLines = append(queuedDisplayLines, prefix+line)
		}
	}
	totalQueuedLines := len(queuedDisplayLines)
	neededBottomRows := totalQueuedLines + inputRowCount

	// Grow-only behavior: the input area never shrinks UNLESS the queue is
	// empty AND the input buffer is empty, at which point it collapses to 1 row.
	canShrink := len(ui.queuedMsgs) == 0 && len(ui.inputBuf) == 0
	var bottomRows int
	if canShrink {
		bottomRows = 1 // single empty input line
		ui.peakBottomRows = 0
	} else {
		if neededBottomRows > ui.peakBottomRows {
			ui.peakBottomRows = neededBottomRows
		}
		bottomRows = ui.peakBottomRows
	}

	// Cap: leave at least 3 rows for output + 1 for divider
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

	// Determine how many queued lines and input rows to actually display
	// (may need to cap if bottom area is too small for all content)
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

	newScrollEnd := ui.rows - bottomRows - 1 // -1 for divider
	if newScrollEnd < 1 {
		newScrollEnd = 1
	}

	oldScrollEnd := ui.scrollEnd
	// Clean transition: when the scroll region is growing (scrollEnd increasing,
	// bottom area shrinking), clear the rows transitioning into the scroll region
	// so stale divider/queued content doesn't linger inside the output area.
	if newScrollEnd > oldScrollEnd {
		for r := oldScrollEnd + 1; r <= newScrollEnd; r++ {
			fmt.Printf("\033[%d;1H\033[2K", r)
		}
	}

	// Update scroll region
	ui.scrollEnd = newScrollEnd
	fmt.Printf("\033[1;%dr", ui.scrollEnd)
	if ui.outRow > ui.scrollEnd {
		ui.outRow = ui.scrollEnd
	}

	// Clear everything below the scroll region
	for r := ui.scrollEnd + 1; r <= ui.rows; r++ {
		fmt.Printf("\033[%d;1H\033[2K", r)
	}

	// Draw divider
	divider := strings.Repeat("─", ui.cols)
	fmt.Printf("\033[%d;1H%s%s%s", ui.scrollEnd+1, Gray, divider, Reset)

	// Draw queued message lines (show most recent if capped)
	row := ui.scrollEnd + 2
	startIdx := len(queuedDisplayLines) - displayQueuedLines
	if startIdx < 0 {
		startIdx = 0
	}
	for i := startIdx; i < len(queuedDisplayLines); i++ {
		fmt.Printf("\033[%d;1H%s%s%s", row, Gray, queuedDisplayLines[i], Reset)
		row++
	}

	// Draw input (prompt + text, terminal auto-wraps)
	inputStartRow := row
	displayInput := inputStr
	maxChars := displayInputRows*ui.cols - promptWidth
	if maxChars < 0 {
		maxChars = ui.cols
	}
	if len(displayInput) > maxChars {
		// Truncate from the left so the cursor stays visible at the end
		displayInput = displayInput[len(displayInput)-maxChars:]
	}
	fmt.Printf("\033[%d;1H%s%s", inputStartRow, promptStr, displayInput)

	// Position cursor at end of displayed text
	dispTotal := promptWidth + len(displayInput)
	cursorRow := inputStartRow + dispTotal/ui.cols
	cursorCol := dispTotal%ui.cols + 1
	fmt.Printf("\033[%d;%dH", cursorRow, cursorCol)
}

// UpdateInput updates the UI's copy of the input state for redrawing.
func (ui *TerminalUI) UpdateInput(prompt string, buf []byte) {
	ui.mu.Lock()
	ui.prompt = prompt
	ui.inputBuf = make([]byte, len(buf))
	copy(ui.inputBuf, buf)
	ui.mu.Unlock()
}

// RedrawInput redraws just the input line.
func (ui *TerminalUI) RedrawInput() {
	ui.mu.Lock()
	ui.drawInputLocked()
	ui.mu.Unlock()
}

// WriteToInputLine writes directly to the input line area (for character echo).
func (ui *TerminalUI) WriteToInputLine(s string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	// Just output the character - next redraw will handle positioning
	fmt.Print(s)
}

// ClearInputLine clears the entire bottom area (queued messages + input).
func (ui *TerminalUI) ClearInputLine() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	for r := ui.scrollEnd + 2; r <= ui.rows; r++ {
		fmt.Printf("\033[%d;1H\033[2K", r)
	}
}

// RefreshSize updates terminal size and redraws layout.
func (ui *TerminalUI) RefreshSize() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	if cols, rows, err := term.GetSize(ui.fd); err == nil && (rows != ui.rows || cols != ui.cols) {
		ui.rows = rows
		ui.cols = cols
		// drawInputLocked recalculates scrollEnd, scroll region, divider
		ui.drawInputLocked()
	}
}

// AddQueuedMessage adds a message to the queued display below the divider.
func (ui *TerminalUI) AddQueuedMessage(text string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.queuedMsgs = append(ui.queuedMsgs, text)
	ui.drawInputLocked()
}

// RemoveQueuedMessage removes the oldest queued message from the display.
func (ui *TerminalUI) RemoveQueuedMessage() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	if len(ui.queuedMsgs) > 0 {
		ui.queuedMsgs = ui.queuedMsgs[1:]
		ui.drawInputLocked()
	}
}

