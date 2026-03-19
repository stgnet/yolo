// YOLO - Your Own Living Operator
// A self-evolving AI agent for software development.
// Continuously runs, thinks, and improves — even when you're not typing.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

// ─── Ollama Log Management ────────────────

// setupOllamaLogging ensures ollama is running with output logged to a file.
// If OLLAMA_DEBUG or YOLO_OLLAMA_LOG is set, it redirects ollama to logs/ollama.log so YOLO can read it.
func setupOllamaLogging() {
	// Check if logging is enabled
	logEnabled := os.Getenv("YOLO_OLLAMA_LOG") == "1" || os.Getenv("OLLAMA_DEBUG") != ""

	if !logEnabled {
		return
	}

	// Create logs directory if it doesn't exist
	logDir, _ := filepath.Abs("./logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not create logs directory: %v\n", err)
		return
	}

	logFile := filepath.Join(logDir, "ollama.log")
	errLogFile := filepath.Join(logDir, "ollama.err.log")

	// Always restart ollama with logging when requested
	// This ensures we capture all output including [OLLAMA DEBUG] messages
	cmd := exec.Command("pgrep", "-f", "ollama serve")
	ollamaRunning := cmd.Run() == nil

	if ollamaRunning {
		fmt.Println("Stopping existing ollama server to enable logging...")
		exec.Command("pkill", "-9", "-f", "ollama serve").Run()
		exec.Command("sleep", "2").Run() // Wait for it to fully stop
		
		// Double-check all ollama processes are stopped
		cmd2 := exec.Command("pgrep", "-f", "ollama")
		if cmd2.Run() == nil {
			fmt.Println("Warning: Some ollama processes may still be running. Trying to kill them all...")
			exec.Command("pkill", "-9", "-f", "ollama").Run()
			exec.Command("sleep", "1").Run()
		}
	}

	fmt.Println("Starting ollama server with logging enabled...")

	// Use shell redirection for reliable process detachment
	// This creates a background process that continues running after YOLO exits
	// and properly redirects output to log files without YOLO needing to manage file handles
	
	ollamaDebug := os.Getenv("OLLAMA_DEBUG")
	if ollamaDebug == "" {
		ollamaDebug = "1" // Enable debug if not already set
	}

	// Get absolute paths for log files to ensure they work from any directory
	absLogFile, _ := filepath.Abs(logFile)
	absErrLogFile, _ := filepath.Abs(errLogFile)

	// Truncate log files before starting to ensure clean output
	os.WriteFile(absLogFile, []byte(""), 0644)
	os.WriteFile(absErrLogFile, []byte(""), 0644)

	// Build the shell command with proper quoting for paths that might have spaces
	// Use > instead of >> to start fresh (files are truncated above)
	startCmd := fmt.Sprintf("( OLLAMA_DEBUG=%s nohup ollama serve > '%s' 2> '%s' ) &", 
		strings.ReplaceAll(ollamaDebug, "'", "'\\''"), 
		absLogFile, 
		absErrLogFile)

	exec.Command("sh", "-c", startCmd).Run()

	// Get absolute paths for display if not already absolute
	displayLogFile := absLogFile
	displayErrLogFile := absErrLogFile
	if !filepath.IsAbs(logFile) {
		displayLogFile, _ = filepath.Abs(logFile)
	}
	if !filepath.IsAbs(errLogFile) {
		displayErrLogFile, _ = filepath.Abs(errLogFile)
	}

	fmt.Printf("Ollama started with logging enabled.\n")
	fmt.Printf("  Standard output: %s\n", displayLogFile)
	fmt.Printf("  Error output: %s\n", displayErrLogFile)
	fmt.Println("YOLO can now read these logs to diagnose Ollama issues.")

	// Give ollama time to start
	exec.Command("sleep", "3").Run()
}

// ─── Entry Point ────────────────

func main() {
	// Check for non-interactive mode (e.g., "go run . learn")
	if len(os.Args) > 1 && os.Args[1] == "learn" {
		runLearnTool()
		return
	}

	// Setup Ollama logging if needed
	setupOllamaLogging()

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprintln(os.Stderr, "Error: yolo requires an interactive terminal (stdin is not a TTY)")
		os.Exit(1)
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Fprintln(os.Stderr, "Error: yolo requires an interactive terminal (stdout is not a TTY)")
		os.Exit(1)
	}
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		fmt.Fprintln(os.Stderr, "Error: yolo requires an interactive terminal (stderr is not a TTY)")
		os.Exit(1)
	}
	fmt.Println("YOLO - Your Own Living Operator")
	fmt.Println("Starting agent...")
	agent := NewYoloAgent()
	agent.Run()
}
