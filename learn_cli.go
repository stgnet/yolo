// CLI support for running learn tool without interactive terminal
package main

import (
	"fmt"
	"os"
)

// runLearnTool executes the learn tool directly from command line
func runLearnTool() {
	baseDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	// Create minimal agent for context
	agent := &YoloAgent{
		baseDir: baseDir,
	}

	// Initialize tool executor
	executor := NewToolExecutor(baseDir, agent)

	// Run the learn tool
	fmt.Println("🔍 Starting autonomous research for self-improvement...")
	result := executor.learn(make(map[string]any))
	fmt.Print(result)
}
