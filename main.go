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
	logDir := "./logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not create logs directory: %v\n", err)
		return
	}

	logFile := logDir + "/ollama.log"
	errLogFile := logDir + "/ollama.err.log"

	// Always restart ollama with logging when requested
	// This ensures we capture all output including [OLLAMA DEBUG] messages
	cmd := exec.Command("pgrep", "-f", "ollama serve")
	ollamaRunning := cmd.Run() == nil

	if ollamaRunning {
		fmt.Println("Stopping existing ollama server to enable logging...")
		exec.Command("pkill", "-f", "ollama serve").Run()
		exec.Command("sleep", "2").Run() // Wait for it to fully stop
	}

	fmt.Println("Starting ollama server with logging enabled...")

	// Prepare environment - keep OLLAMA_DEBUG if set so we capture those messages
	newEnv := os.Environ()
	
	// Start new ollama with output redirected to log files
	ollamaCmd := exec.Command("ollama", "serve")
	ollamaCmd.Env = newEnv
	
	// Redirect both stdout and stderr to log files
	stdoutFile, err := os.Create(logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not create stdout log file: %v\n", err)
		return
	}
	
	stderrFile, err := os.OpenFile(errLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not create stderr log file: %v\n", err)
		stdoutFile.Close()
		return
	}

	ollamaCmd.Stdout = stdoutFile
	ollamaCmd.Stderr = stderrFile
	
	err = ollamaCmd.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting ollama: %v\n", err)
		stdoutFile.Close()
		stderrFile.Close()
		return
	}

	fmt.Printf("Ollama started with logging enabled.\n")
	fmt.Printf("  Standard output: %s\n", logFile)
	fmt.Printf("  Error output: %s\n", errLogFile)
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
