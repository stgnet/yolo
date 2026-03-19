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

	// Warn about OLLAMA_DEBUG if set - it causes verbose output
	ollamaDebug := os.Getenv("OLLAMA_DEBUG")
	if ollamaDebug != "" && ollamaDebug != "0" {
		fmt.Printf("Note: OLLAMA_DEBUG=%s is set. This causes verbose debug messages from ollama server.\n", ollamaDebug)
		fmt.Println("To suppress these messages, unset the variable: unset OLLAMA_DEBUG")
		fmt.Println("Or set it to 0: export OLLAMA_DEBUG=0")
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
