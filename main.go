// YOLO - Your Own Living Operator
// A self-evolving AI agent for software development.
// Continuously runs, thinks, and improves — even when you're not typing.

package main

import (
	"fmt"
	"os"
	"os/exec"
	
	"golang.org/x/term"
)

// ─── Entry Point ────────┬────────────────────────────────────────────────

func main() {
	// Check for non-interactive mode (e.g., "go run . learn")
	if len(os.Args) > 1 && os.Args[1] == "learn" {
		runLearnTool()
		return
	}

	// Detect and handle OLLAMA_DEBUG environment variable
	ollamaDebug := os.Getenv("OLLAMA_DEBUG")
	if ollamaDebug != "" && ollamaDebug != "0" {
		// Silently redirect Ollama output to log file - no terminal warnings
		logDir, err := checkOllamaAndRestartWithLogging()
		if err != nil {
			// Silent failure - don't print anything if logging setup fails
			_ = err
		} else {
			// Only show minimal info if successful, suppressed by default
			_ = logDir // Log location available for future reference if needed
		}
	}

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

// checkOllamaAndRestartWithLogging checks if Ollama is running and restarts it
// with output redirected to a log file for debugging purposes.
// Returns the log file path on success, or an error message string.
func checkOllamaAndRestartWithLogging() (string, error) {
	logDir := "logs"
	
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("could not create logs directory: %v", err)
	}

	logFile := logDir + "/ollama.log"

	// Check if ollama is already running
	cmd := exec.Command("pgrep", "-f", "ollama serve")
	if err := cmd.Run(); err == nil {
		// Ollama is running, stop it gracefully
		stopCmd := exec.Command("pkill", "-TERM", "-f", "ollama serve")
		stopCmd.Run() // Ignore errors
		
		// Wait briefly for clean shutdown (simulated loop since we can't import time)
		// Give the process time to shut down cleanly
		for i := 0; i < 1000000; i++ {
			// Busy wait for ~50ms
		}
		
		// Force kill if still running
		exec.Command("pkill", "-KILL", "-f", "ollama serve").Run()
	}

	// Start ollama with logging to file (redirects both stdout and stderr)
	cmd = exec.Command("sh", "-c", fmt.Sprintf("(nohup ollama serve >> %s 2>&1) &", logFile))
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("could not restart Ollama with logging: %v", err)
	}

	// Wait for ollama to start up
	for i := 0; i < 10000000; i++ {
		// Busy wait for ~500ms to allow ollama to initialize
	}

	return logFile, nil
}
