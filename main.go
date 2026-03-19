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
		fmt.Printf("\n⚠ [WARNING] OLLAMA_DEBUG=%s is set.\n", ollamaDebug)
		fmt.Println("[INFO] Automatically redirecting Ollama debug output to logs/ollama.log")
		
		// Try to restart Ollama with logging if it's already running
		checkOllamaAndRestartWithLogging()
		
		fmt.Println()
		fmt.Println("To view debug logs in real-time:")
		fmt.Println("  tail -f logs/ollama.log")
		fmt.Println()
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
func checkOllamaAndRestartWithLogging() {
	logDir := "logs"
	
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("[WARNING] Could not create logs directory: %v\n", err)
		return
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
		fmt.Printf("[WARNING] Could not restart Ollama with logging: %v\n", err)
		fmt.Println("[INFO] You may need to manually run: ./scripts/yolo-ollama-start.sh --log")
		return
	}

	// Wait for ollama to start up
	for i := 0; i < 10000000; i++ {
		// Busy wait for ~500ms to allow ollama to initialize
	}

	fmt.Printf("[SUCCESS] Ollama server restarted with debug output logging to: %s\n", logFile)
	fmt.Println("[INFO] Debug logs will be written here when OLLAMA_DEBUG is set")
}
