// Package terminalui provides terminal output rendering and buffering for both
// split-screen terminal mode and linear buffer mode.
package terminalui

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/term"
)

// ─── Output Sanitization ─────────────────────────

// sanitizeOutput strips terminal escape sequences and control characters from
// external text (LLM output, tool results) that could corrupt terminal state.
// It preserves printable text, newlines, tabs, and carriage returns.
// ANSI CSI SGR sequences (colors/styles, ending in 'm') are allowed through
// since they're used for display, but all other CSI sequences are stripped.
func SanitizeOutput(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))
	i := 0
	for i < len(s) {
		b := s[i]

		// Check for ESC (0x1B) - start of escape sequence
		if b == 0x1B && i+1 < len(s) {
			next := s[i+1]
			switch {
			case next == '[': // CSI sequence: \033[ ... <final byte>
				j := i + 2
				for j < len(s) && s[j] >= 0x20 && s[j] <= 0x3F {
					j++ // parameter bytes and intermediate bytes
				}
				if j < len(s) && s[j] >= 0x40 && s[j] <= 0x7E {
					finalByte := s[j]
					if finalByte == 'm' {
						// SGR (Select Graphic Rendition) - allow colors/styles
						buf.WriteString(s[i : j+1])
					}
					i = j + 1
					continue
				}
				i += 2
				continue

			case next == ']': // OSC sequence: \033] ... (ST or BEL)
				j := i + 2
				for j < len(s) {
					if s[j] == 0x07 { // BEL terminates OSC
						j++
						break
					}
					if s[j] == 0x1B && j+1 < len(s) && s[j+1] == '\\' { // ST
						j += 2
						break
					}
					j++
				}
				i = j
				continue

			case next == 'P' || next == 'X' || next == '^' || next == '_':
				// DCS, SOS, PM, APC sequences - skip until ST
				j := i + 2
				for j < len(s) {
					if s[j] == 0x1B && j+1 < len(s) && s[j+1] == '\\' {
						j += 2
						break
					}
					j++
				}
				i = j
				continue

			default:
				// Other ESC + single char sequences - drop them
				i += 2
				continue
			}
		}

		// Allow printable ASCII, newline, tab, carriage return
		if b == '\n' || b == '\t' || b == '\r' || (b >= 0x20 && b < 0x7F) {
			buf.WriteByte(b)
			i++
			continue
		}

		// Allow valid UTF-8 multibyte sequences
		if b >= 0xC0 && b < 0xFE {
			size := 2
			if b >= 0xE0 {
				size = 3
			}
			if b >= 0xF0 {
				size = 4
			}
			if i+size <= len(s) {
				valid := true
				for k := 1; k < size; k++ {
					if s[i+k]&0xC0 != 0x80 {
						valid = false
						break
					}
				}
				if valid {
					buf.WriteString(s[i : i+size])
					i += size
					continue
				}
			}
		}

		// Drop all other control characters and invalid bytes
		i++
	}
	return buf.String()
}

// ─── ANSI Colors ────────────────────────────────

const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Gray    = "\033[90m"
)

// Color constants for UI output
var colorPrefixes = map[string]string{
	"reset":   Reset,
	"bold":    Bold,
	"dim":     Dim,
	"red":     Red,
	"green":   Green,
	"yellow":  Yellow,
	"blue":    Blue,
	"magenta": Magenta,
	"cyan":    Cyan,
	"gray":    Gray,
}

// ─── Subagent Window Constants ──────────────────

const (
	SubagentContentRows = 4
	SubagentWindowRows  = SubagentContentRows + 1
	MaxVisibleSubWindows = 3
	SubagentWindowTimeout = 300 * time.Second
	MinScrollRows       = 4
)

// AgentWindow represents a subagent's output window displayed at the top.
type AgentWindow struct {
	ID          int
	Label       string
	TextBuffer  strings.Builder
	Completed   bool
	CompletedAt time.Time
}

// TerminalUI represents the split-screen terminal UI with scrollable output and input.
type TerminalUI struct {
	rows          int
	cols          int
	outRow        int
	outCol        int
	promptRow     int
	inputRow      int
	subagentRows  int
	scrollEnd     int
	agentRows     int
	lastRows      int
	outWin        *term.Window
	termWin       *term.Window
	mu            sync.Mutex
	cond          *sync.Cond
	subagents     map[int]*AgentWindow
	subagentOrder []int
	outputBuffer  strings.Builder
}

// NewTerminalUI creates a new TerminalUI instance.
func NewTerminalUI() *TerminalUI {
	return &TerminalUI{
		rows:        24,
		cols:        80,
		outRow:      1,
		outCol:      1,
		promptRow:   24,
		inputRow:    23,
		subagentRows: 0,
		scrollEnd:   20,
		agentRows:   2,
		outWin:      &term.Window{},
		termWin:     &term.Window{},
		subagents:   make(map[int]*AgentWindow),
	}
}

// Setup initializes the terminal UI. Call this after creating a TerminalUI.
func (ui *TerminalUI) Setup() {
	rows, cols, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil && rows > 0 && cols > 0 {
		ui.rows = rows
		ui.cols = cols
	}

	ui.cond = sync.NewCond(&ui.mu)
	ui.updateSubagentWindows()
	ui.establishScrollRegion()
	ui.clearScreen()
}

// Teardown cleans up the terminal UI. Call when done using the UI.
func (ui *TerminalUI) Teardown() {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	// Reset scroll region to full screen
	fmt.Fprintf(os.Stdout, "\033[%d;1r", ui.rows)
	fmt.Fprint(os.Stdout, "\033[H\033[2J")
}

// RefreshSize refreshes the terminal size. Call on window resize.
func (ui *TerminalUI) RefreshSize() {
	rows, cols, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil && rows > 0 && cols > 0 {
		ui.mu.Lock()
		oldRows := ui.rows
		oldCols := ui.cols
		ui.rows = rows
		ui.cols = cols
		ui.updateSubagentWindows()

		if rows != oldRows || cols != oldCols {
			ui.establishScrollRegion()
			ui.redrawSubagentWindows()
		}
		ui.mu.Unlock()
	}
}

// AddSubagentWindow creates a new subagent window with the given ID and label.
func (ui *TerminalUI) AddSubagentWindow(id int, label string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	ui.subagents[id] = &AgentWindow{
		ID:        id,
		Label:     label,
		TextBuffer: strings.Builder{},
	}
	ui.subagentOrder = append(ui.subagentOrder, id)
	ui.updateSubagentWindows()
}

// WriteToSubagentWindow writes text to a subagent's window.
func (ui *TerminalUI) WriteToSubagentWindow(id int, text string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	if win, ok := ui.subagents[id]; ok {
		sanitized := SanitizeOutput(text)
		rawWriteTo(&win.TextBuffer, sanitized)
	}
}

// MarkSubagentComplete marks a subagent window as complete.
func (ui *TerminalUI) MarkSubagentComplete(id int) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	if win, ok := ui.subagents[id]; ok {
		win.Completed = true
		win.CompletedAt = time.Now()
	}
}

// OutputPrint prints text to the terminal output area.
func (ui *TerminalUI) OutputPrint(text string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	sanitized := SanitizeOutput(text)
	rawWriteTo(&ui.outputBuffer, sanitized)
	ui.renderOutput(sanitized)
}

// UpdateInput updates the input buffer display.
func (ui *TerminalUI) UpdateInput(prompt string, buf []byte) {
	// Input update handled by separate component
}

// RedrawInput refreshes the input area display.
func (ui *TerminalUI) RedrawInput() {
	// Input redraw handled by separate component
}

// ClearScreen clears and resets the terminal.
func (ui *TerminalUI) ClearScreen() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Fprint(os.Stdout, "\033[H\033[2J")
}

// establishScrollRegion sets up scroll regions for output display.
func (ui *TerminalUI) establishScrollRegion() {
	subagentRows := len(ui.subagents) * SubagentWindowRows
	if subagentRows > 0 && subagentRows < ui.rows-MinScrollRows-1 {
		scrollEnd := ui.rows - MinScrollRows - subagentRows
		if scrollEnd < 4 {
			scrollEnd = 4
		}
		fmt.Fprintf(os.Stdout, "\033[1;%dR", scrollEnd)
		ui.scrollEnd = scrollEnd
	} else {
		fmt.Fprintf(os.Stdout, "\033[%d;1r", ui.rows-MinScrollRows)
		ui.scrollEnd = ui.rows - MinScrollRows
	}
}

// updateSubagentWindows recalculates subagent window positions.
func (ui *TerminalUI) updateSubagentWindows() {
	subagentRows := len(ui.subagents) * SubagentWindowRows
	if subagentRows > 0 && subagentRows <= ui.rows-MinScrollRows-1 {
		ui.agentRows = subagentRows
	} else {
		ui.agentRows = 0
	}
}

// redrawSubagentWindows refreshes all subagent windows.
func (ui *TerminalUI) redrawSubagentWindows() {
	cursor := 0
	for i, id := range ui.subagentOrder {
		if i >= MaxVisibleSubWindows {
			break
		}

		win, ok := ui.subagents[id]
		if !ok {
			continue
		}

		var buf strings.Builder
		buf.WriteString(fmt.Sprintf("\033[%dH", cursor+1))
		if win.Completed {
			buf.WriteString(Bold + Gray + fmt.Sprintf("═══ [ %s ] (DONE) ═══\n%s\n", win.Label, Reset))
		} else {
			buf.WriteString(Bold + Gray + fmt.Sprintf("═══ [ %s ] ═══\n%s\n", win.Label, Reset))
		}

		var textBuf strings.Builder
		subagentText := truncateString(win.TextBuffer.String(), SubagentContentRows*ui.cols)
		lines := strings.Split(subagentText, "\n")
		for _, line := range lines {
			if len(textBuf.Bytes()) >= SubagentContentRows*ui.cols {
				break
			}
			textBuf.WriteString(Bold + Gray + truncateString(line, ui.cols-2) + Reset)
			textBuf.WriteByte('\n')
			cursor++
		}

		fmt.Fprintf(os.Stdout, "\033[%dH%s", cursor+1, textBuf.String())
	}
}

// renderOutput renders output within the scroll region.
func (ui *TerminalUI) renderOutput(s string) {
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		ui.trackCursorMovement(line)

		if ui.outRow > ui.scrollEnd-2 || (ui.outRow == ui.scrollEnd-1 && ui.outCol > 4) {
			fmt.Fprint(os.Stdout, "\033[A\033[K")
			for i := ui.inputRow; i < ui.rows-MinScrollRows; i++ {
				fmt.Fprintf(os.Stdout, "\033[%dA", i-ui.inputRow+1)
				fmt.Fprint(os.Stdout, "\033[2K")
			}
			ui.outRow = 1
			ui.outCol = 1
		}

		if len(line) > ui.cols {
			line = line[:ui.cols]
		}

		cursorAt := fmt.Sprintf("\033[%d;%dH", ui.outRow, ui.outCol)
		clearToEOL := "\033[K"
		fmt.Fprintf(os.Stdout, "%s%s%s", cursorAt, line, clearToEOL)
		ui.outCol += len(line)

		if ui.outCol > ui.cols {
			ui.outCol = 1
			ui.outRow++
		}
	}
}

// trackCursorMovement tracks cursor movement for scrolling.
func (ui *TerminalUI) trackCursorMovement(text string) {
	var atRow, atCol int
	for _, c := range text {
		switch c {
		case '\r': // Carriage return: cursor to column 1
			atCol = 1
		case '\n': // Line feed: move down one row
			if atRow < ui.rows-MinScrollRows {
				atRow++
			}
			atCol = 1
		default:
			atCol++
			if atCol > ui.cols {
				atCol = 1
				if atRow < ui.rows-MinScrollRows {
					atRow++
				}
			}
		}
	}
	ui.outRow = atRow
	ui.outCol = atCol
}

// clearScreen clears the terminal and resets cursor.
func (ui *TerminalUI) clearScreen() {
	fmt.Fprint(os.Stdout, "\033[H\033[2J")
}

// rawWriteTo appends text to a builder, converting lone \n to \r\n for raw terminal mode.
func rawWriteTo(buf *strings.Builder, s string) {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", "\r\n")
	buf.WriteString(s)
}

// TruncateString truncates a string to maxLen characters.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	runeCount := 0
	for i := range s {
		if runeCount >= maxLen-4 {
			return s[:i] + "..."
		}
		runeCount++
	}
	return s
}

// TruncateStringWithAnsi truncates a string considering ANSI codes.
func TruncateStringWithAnsi(s string, maxLen int) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	stripped := re.ReplaceAllString(s, "")

	if len(stripped) <= maxLen {
		return s
	}

	runeCount := 0
	result := strings.Builder{}
	sanitizedAnsi := false

	for i := range stripped {
		if runeCount >= maxLen-4 {
			break
		}
		startPos := i

		// Extract any ANSI codes before this character
		prefixMatches := re.FindAllString(s[startPos:], 1)
		if len(prefixMatches) > 0 {
			result.WriteString(prefixMatches[0])
			sanitizedAnsi = true
		}

		result.WriteByte(stripped[i])
		runeCount++
	}

	if sanitizedAnsi {
		result.WriteString(Reset)
	}

	return result.String() + "..."
}

// BreakWordAtVisibleLength attempts to break a word at maxLen visible characters.
func BreakWordAtVisibleLength(word string, maxLen int) (string, string) {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	stripped := re.ReplaceAllString(word, "")

	if len(stripped) <= maxLen {
		return word, ""
	}

	runeCount := 0
	resultPrefix := strings.Builder{}
	sanitizedAnsi := false

	for i := range stripped {
		if runeCount >= maxLen {
			break
		}

		prefixMatches := re.FindAllString(word[i:], 1)
		if len(prefixMatches) > 0 {
			resultPrefix.WriteString(prefixMatches[0])
			sanitizedAnsi = true
		}

		resultPrefix.WriteByte(stripped[i])
		runeCount++
	}

	resultSuffix := strings.Builder{}
	for i := range stripped[runeCount:] {
		prefixMatches := re.FindAllString(word[runeCount+i:], 1)
		if len(prefixMatches) > 0 {
			resultSuffix.WriteString(prefixMatches[0])
		}
		resultSuffix.WriteByte(stripped[runeCount+i])
	}

	if sanitizedAnsi {
		resultSuffix.WriteString(Reset)
	}

	return resultPrefix.String(), resultSuffix.String()
}

// StripAnsiCodes removes ANSI escape codes from a string.
func StripAnsiCodes(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(s, "")
}
