// YOLO - Your Own Living Operator
// A self-evolving AI agent for software development.
// Continuously runs, thinks, and improves — even when you're not typing.

package main

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// ─── Entry Point ──────────────────────────────────────────────────────

func main() {
	// Check for non-interactive mode (e.g., "go run . learn")
	if len(os.Args) > 1 && os.Args[1] == "learn" {
		runLearnTool()
		return
	}

	// Detect and warn about OLLAMA_DEBUG environment variable
	ollamaDebug := os.Getenv("OLLAMA_DEBUG")
	if ollamaDebug != "" && ollamaDebug != "0" {
		fmt.Printf("\n⚠ [WARNING] OLLAMA_DEBUG=%s is set.\n", ollamaDebug)
		fmt.Println("[WARNING] This causes verbose [OLLAMA DEBUG] messages from the Ollama server.")
		fmt.Println()
		fmt.Println("To capture these messages to a log file for debugging:")
		fmt.Println("  ./scripts/yolo-ollama-start.sh --log")
		fmt.Println()
		fmt.Println("Then view logs with:")
		fmt.Println("  tail -f logs/ollama.log")
		fmt.Println()
		fmt.Println("To suppress debug messages entirely:")
		fmt.Println("  unset OLLAMA_DEBUG")
		fmt.Println("  export OLLAMA_DEBUG=0")
		fmt.Println()
		
		// Create a convenience script if it doesn't exist
		convenienceScript := "./scripts/fix-ollama-debug.sh"
		if _, err := os.Stat(convenienceScript); os.IsNotExist(err) {
			scriptContent := `#!/bin/bash
# fix-ollama-debug.sh - Convenience script to handle OLLAMA_DEBUG output
#
# This script helps redirect Ollama debug output to a log file so YOLO can read it.

echo "Setting up Ollama debug logging..."

# Stop any running ollama instances
pkill -f "ollama serve" 2>/dev/null
sleep 1

# Create logs directory
mkdir -p logs

# Start ollama with logging
nohup ollama serve >> logs/ollama.log 2>&1 &
OLLAMA_PID=$!

echo "Ollama server started with PID: $OLLAMA_PID"
echo "Debug output is being logged to: logs/ollama.log"
echo ""
echo "To view logs in real-time:"
echo "  tail -f logs/ollama.log"
echo ""
echo "To stop ollama:"
echo "  kill $OLLAMA_PID"
echo ""
echo "YOLO can now read the log file at: $(pwd)/logs/ollama.log"
`
			if err := os.WriteFile(convenienceScript, []byte(scriptContent), 0755); err == nil {
				fmt.Printf("✓ Created convenience script: %s\n", convenienceScript)
				fmt.Println("✓ Run it with: ./scripts/fix-ollama-debug.sh")
				fmt.Println()
			} else {
				fmt.Printf("⚠ Could not create convenience script: %v\n", err)
			}
		} else {
			fmt.Printf("ℹ Convenience script exists: %s\n", convenienceScript)
			fmt.Println("ℹ Run it with: ./scripts/fix-ollama-debug.sh")
			fmt.Println()
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
