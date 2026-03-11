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
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprintln(os.Stderr, "Error: yolo requires an interactive terminal (stdin is not a TTY)")
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
