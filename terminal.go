package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// ─── Terminal Output ──────────────────────────────────────────────────

// globalUI is set once the split UI is active. Before that, output goes to stdout directly.
var globalUI *TerminalUI

// rawWrite writes text to stdout, converting lone \n to \r\n for raw terminal mode.
// In raw mode, OPOST is disabled so \n only moves the cursor down without returning
// to column 1. This function ensures proper carriage return + line feed behavior.
func rawWrite(s string) {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
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
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(s, "")
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
	mu       sync.Mutex
	fd       int
	rows     int
	cols     int
	inputBuf []byte // mirrors InputManager's buffer for redraw
	prompt   string
	outRow   int // tracked row of cursor in output region
	outCol   int // tracked col of cursor in output region
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
		fd:     fd,
		rows:   rows,
		cols:   cols,
		outRow: 1,
		outCol: 1,
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

	// Clear screen, set scroll region, draw divider
	fmt.Print("\033[2J")
	fmt.Printf("\033[1;%dr", ui.rows-2)
	ui.drawDividerLocked()
	ui.outRow = 1
	ui.outCol = 1
	fmt.Printf("\033[%d;%dH", ui.outRow, ui.outCol)
}

func (ui *TerminalUI) drawDividerLocked() {
	divider := strings.Repeat("─", ui.cols)
	fmt.Printf("\033[%d;1H%s%s%s", ui.rows-1, Gray, divider, Reset)
}

// Teardown restores the full scroll region.
func (ui *TerminalUI) Teardown() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("\033[1;%dr", ui.rows)
	fmt.Printf("\033[%d;1H\n", ui.rows)
}

func (ui *TerminalUI) trackCursorMovement(stripped string) {
	for _, ch := range stripped {
		switch ch {
		case '\n':
			ui.outRow++
			ui.outCol = 1
			if ui.outRow > ui.rows-2 {
				ui.outRow = ui.rows - 2 // scroll region keeps cursor at bottom
			}
		case '\r':
			ui.outCol = 1
		default:
			ui.outCol++
			if ui.outCol > ui.cols {
				ui.outCol = 1
				ui.outRow++
				if ui.outRow > ui.rows-2 {
					ui.outRow = ui.rows - 2
				}
			}
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
	// Move to input row and clear it
	fmt.Printf("\033[%d;1H\033[2K", ui.rows) // Clear entire line

	promptStr := ui.prompt
	inputStr := string(ui.inputBuf)

	// Calculate available space after prompt
	promptWidth := len(stripAnsiCodes(promptStr))
	availableWidth := ui.cols - promptWidth - 1 // -1 for cursor space

	if availableWidth <= 0 {
		availableWidth = ui.cols - 2
	}

	// If input fits on one line, show it all (current behavior)
	var displayInput string
	cursorCol := promptWidth + len(inputStr) + 1
	if len(inputStr) <= availableWidth {
		displayInput = inputStr
	} else {
		// Show rightmost portion that fits (horizontal scrolling)
		startPos := len(inputStr) - availableWidth
		displayInput = inputStr[startPos:]
		cursorCol = promptWidth + len(displayInput) + 1
	}

	// Draw prompt and input, then position cursor
	fmt.Printf("%s%s\033[%d;%dH", promptStr, displayInput, ui.rows, cursorCol)
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

// ClearInputLine clears the input line.
func (ui *TerminalUI) ClearInputLine() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("\033[%d;1H\033[K", ui.rows)
}

// RefreshSize updates terminal size and redraws layout.
func (ui *TerminalUI) RefreshSize() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	if cols, rows, err := term.GetSize(ui.fd); err == nil && (rows != ui.rows || cols != ui.cols) {
		ui.rows = rows
		ui.cols = cols
		fmt.Printf("\033[1;%dr", ui.rows-2)
		ui.drawDividerLocked()
		ui.drawInputLocked()
	}
}

// ─── Spinner ──────────────────────────────────────────────────────────

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type Spinner struct {
	prefix string
	color  string
	stop   chan struct{}
	done   chan struct{}
}

func NewSpinner(prefix, color string) *Spinner {
	return &Spinner{
		prefix: prefix,
		color:  color,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

func (s *Spinner) Start() {
	go func() {
		defer close(s.done)
		i := 0
		for {
			select {
			case <-s.stop:
				return
			default:
				frame := spinnerFrames[i%len(spinnerFrames)]
				text := fmt.Sprintf("\r%s%s%s thinking...%s", s.color, s.prefix, frame, Reset)
				if globalUI != nil {
					globalUI.OutputPrintInline(text)
				} else {
					rawWrite(text)
				}
				i++
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

func (s *Spinner) Stop() {
	close(s.stop)
	<-s.done
	clearLen := len(s.prefix) + 20
	text := fmt.Sprintf("\r%s\r", strings.Repeat(" ", clearLen))
	if globalUI != nil {
		globalUI.OutputPrintInline(text)
	} else {
		rawWrite(text)
	}
}
