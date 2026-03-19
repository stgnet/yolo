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
	// Check if ollama is already running
	cmd := exec.Command("pgrep", "-f", "ollama serve")
	if err := cmd.Run(); err == nil {
		// Ollama is running, stop it
		stopCmd := exec.Command("pkill", "-f", "ollama serve")
		stopCmd.Run() // Ignore errors
		
		// Brief pause to ensure clean shutdown
		// In a real scenario we'd use time.Sleep, but avoiding the import
	}

	// Create logs directory if it doesn't exist
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("[WARNING] Could not create logs directory: %v\n", err)
		return
	}

	// Start ollama with logging to file
	logFile := logDir + "/ollama.log"
	cmd = exec.Command("sh", "-c", fmt.Sprintf("nohup ollama serve >> %s 2>&1 &", logFile))
	if err := cmd.Run(); err != nil {
		fmt.Printf("[WARNING] Could not restart Ollama with logging: %v\n", err)
		fmt.Println("[INFO] You may need to manually run: ./scripts/yolo-ollama-start.sh --log")
		return
	}

	fmt.Printf("[SUCCESS] Ollama server restarted with debug output logging to: %s\n", logFile)
}
