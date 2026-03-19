// YOLO - Your Own Living Operator
// A self-evolving AI agent for software development.
// Continuously runs, thinks, and improves — even when you're not typing.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

// ─── Ollama Log Management ────────────────

// setupOllamaLogging ensures ollama is running with output logged to a file.
// If OLLAMA_DEBUG is set, it redirects ollama to logs/ollama.log so YOLO can read it.
func setupOllamaLogging() {
	// Check if OLLAMA_DEBUG is set or if we want logging enabled
	debugSet := os.Getenv("OLLAMA_DEBUG") != "" || os.Getenv("YOLO_OLLAMA_LOG") == "1"

	if !debugSet {
		return
	}

	// Create logs directory if it doesn't exist
	logDir := "./logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not create logs directory: %v\n", err)
		return
	}

	logFile := logDir + "/ollama.log"

	// Check if ollama is already running
	cmd := exec.Command("pgrep", "-f", "ollama serve")
	ollamaRunning := cmd.Run() == nil

	if !ollamaRunning {
		// Ollama not running, start it with logging
		fmt.Println("Starting ollama server with logging...")

		// Kill any stale ollama processes first
		exec.Command("pkill", "-f", "ollama serve").Run()

		// Give it a moment to fully stop
		exec.Command("sleep", "1").Run()

		// Start new ollama with output redirected to log file
		// We disable debug mode to reduce log noise since we're logging anyway
		newEnv := os.Environ()
		for i, env := range newEnv {
			if strings.HasPrefix(env, "OLLAMA_DEBUG") {
				newEnv[i] = "OLLAMA_DEBUG=0"
			}
		}

		ollamaCmd := exec.Command("nohup", "ollama", "serve")
		ollamaCmd.Env = newEnv
		ollamaCmd.Stdout, _ = os.Create(logFile)
		ollamaCmd.Stderr, _ = os.Create(logFile+".err")
		ollamaCmd.Start() // Don't block waiting for it

		fmt.Println("Ollama started. Logs will be written to", logFile)

		// Give ollama time to start
		exec.Command("sleep", "3").Run()
	} else {
		// Ollama is already running - we can't redirect its output easily
		// Just inform the user and suggest they restart manually if needed
		fmt.Println("Ollama is already running. To enable logging:")
		fmt.Println("  1. Run: pkill -f 'ollama serve'")
		fmt.Println("  2. Then run: nohup ollama serve > logs/ollama.log 2>&1 &")
		fmt.Println("  3. Or restart YOLO after killing ollama to have it auto-start with logging")
	}
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
