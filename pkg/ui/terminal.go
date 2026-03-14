// Package ui provides terminal user interface components for the YOLO agent.
package ui

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/term"
)

// TerminalUI handles all terminal output and display.
type TerminalUI struct {
	mu           sync.Mutex
	outputBuffer []string // Buffer for accumulating output
	currentLine  string   // Current line being written to
	isTerminal   bool     // Whether we're running in a terminal
	bufferedMode bool     // If true, buffer all output instead of printing immediately
}

// OutputConfig holds configuration for TerminalUI.
type OutputConfig struct {
	BufferedMode bool // If true, buffer output until flushed
}

// DefaultOutputConfig returns default UI configuration.
func DefaultOutputConfig() OutputConfig {
	return OutputConfig{
		BufferedMode: false,
	}
}

// NewTerminalUI creates a new terminal UI handler.
func NewTerminalUI(cfg OutputConfig) *TerminalUI {
	ui := &TerminalUI{
		outputBuffer: make([]string, 0),
		isTerminal:   true,
	}

	// Check if stdout is a terminal
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		ui.isTerminal = false
	}

	if cfg.BufferedMode {
		ui.bufferedMode = true
	}

	return ui
}

// ANSI escape sequences for terminal colors.
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

	// Bold variants
	BoldRed    = "\033[1m\033[31m"
	BoldGreen  = "\033[1m\033[32m"
	BoldYellow = "\033[1m\033[33m"
	BoldBlue   = "\033[1m\033[34m"
)

// OutputPrint prints text to stdout with optional color.
func (ui *TerminalUI) OutputPrint(text string, colors ...string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	if len(colors) > 0 {
		text = colors[0] + text + Reset
	}

	if ui.bufferedMode {
		ui.outputBuffer = append(ui.outputBuffer, text)
		return
	}

	fmt.Print(text)
}

// OutputPrintln prints text to stdout with newline.
func (ui *TerminalUI) OutputPrintln(text string, colors ...string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	if len(colors) > 0 {
		text = colors[0] + text + Reset
	}

	if ui.bufferedMode {
		ui.outputBuffer = append(ui.outputBuffer, text+"\n")
		return
	}

	fmt.Println(text)
}

// OutputPrintf formats and prints to stdout.
func (ui *TerminalUI) OutputPrintf(format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	ui.OutputPrintln(text)
}

// OutputWrite writes text directly without formatting.
func (ui *TerminalUI) OutputWrite(text string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	if ui.bufferedMode {
		ui.outputBuffer = append(ui.outputBuffer, text)
		return
	}

	fmt.Print(text)
}

// ClearLine clears the current line and moves cursor to beginning.
func (ui *TerminalUI) ClearLine() {
	if !ui.isTerminal {
		return
	}

	fmt.Print("\033[2K\r")
}

// OutputFinishLine finalizes output for a complete line.
func (ui *TerminalUI) OutputFinishLine() {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	if ui.bufferedMode {
		// Flush the buffer
		for _, line := range ui.outputBuffer {
			fmt.Print(line)
		}
		ui.outputBuffer = ui.outputBuffer[:0]
		return
	}

	fmt.Println()
}

// OutputInlinePrint prints inline without newline.
func (ui *TerminalUI) OutputInlinePrint(text string, colors ...string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	if len(colors) > 0 {
		text = colors[0] + text + Reset
	}

	if ui.bufferedMode {
		ui.outputBuffer = append(ui.outputBuffer, text)
		return
	}

	fmt.Print(text)
}

// OutputInlineFinish finishes inline output.
func (ui *TerminalUI) OutputInlineFinish() {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	if ui.bufferedMode {
		// Flush the buffer
		for _, text := range ui.outputBuffer {
			fmt.Print(text)
		}
		ui.outputBuffer = ui.outputBuffer[:0]
		return
	}

	fmt.Println()
}

// SetBufferedMode changes between immediate and buffered output.
func (ui *TerminalUI) SetBufferedMode(enabled bool) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.bufferedMode = enabled
}

// GetBufferSize returns the current buffer size.
func (ui *TerminalUI) GetBufferSize() int {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	return len(ui.outputBuffer)
}

// ClearBuffer clears any buffered output.
func (ui *TerminalUI) ClearBuffer() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.outputBuffer = ui.outputBuffer[:0]
}
