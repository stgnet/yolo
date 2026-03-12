package main

import (
	"os"
	"path/filepath"
	"regexp"
)

// ─── Configuration ────────────────────────────────────────────────────

const (
	// YoloDir is the directory (relative to the working directory) where
	// history, sub-agent results, and other state files are stored.
	YoloDir = ".yolo"

	// IMPORTANT: Source code is in current directory (.), NOT in yolo/
	// File paths should be relative to: /Users/sgriepentrog/src/yolo
	// Example: Use "tools_inbox.go", NOT "yolo/tools_inbox.go"
	_SourceCodeLocation = "."

	// CRITICAL: Use the restart tool to rebuild, NEVER call os.Exit()
	// The restart tool handles: go build → go test → git commit → clean restart
	_UseRestartTool = true

	// MaxContextMessages caps how many history messages are included in the
	// context window sent to the LLM.
	MaxContextMessages = 40

	// MaxToolOutput limits tool output length (0 = unlimited).
	MaxToolOutput = 0

	// CommandTimeout is the maximum wall-clock seconds a shell command
	// (run_command tool) is allowed to run before being killed.
	CommandTimeout = 30

	// MaxSubagentRounds is the maximum number of LLM ↔ tool-execution
	// rounds a sub-agent is allowed before it must return.
	MaxSubagentRounds = 20

	// DefaultNumCtx is the default context-window size passed to Ollama
	// when auto-detection fails or is not available.
	DefaultNumCtx = 8192

	// DefaultInputDelay is the number of seconds to wait after the user
	// presses Enter (cursor at start of blank line) before sending the
	// input buffer to the agent. Overridable via YOLO_INPUT_DELAY env var.
	DefaultInputDelay = 10
)

var (
	// HistoryFile is the default path to the conversation history JSON file.
	HistoryFile = filepath.Join(YoloDir, "history.json")

	// OllamaURL is the Ollama API base URL, overridable via the OLLAMA_URL
	// environment variable.
	OllamaURL = getEnvDefault("OLLAMA_URL", "http://localhost:11434")

	// NumCtxOverride, when non-empty, forces the context-window size sent
	// to Ollama instead of auto-detecting from the model metadata.
	NumCtxOverride = os.Getenv("YOLO_NUM_CTX")

	// SubagentDir is the directory where sub-agent result JSON files are
	// written (one file per spawned sub-agent).
	SubagentDir = filepath.Join(YoloDir, "subagents")

	// fileNameRegex matches sub-agent result files (agent_1.json, agent_test_123.json, etc.).
	fileNameRegex = regexp.MustCompile(`agent_(\S+)\.json`)
)

// getEnvDefault returns the value of the environment variable key, or
// fallback if the variable is unset or empty.
func getEnvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ─── ANSI Colors ──────────────────────────────────────────────────────

// ANSI escape sequences for terminal color output.
const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Gray    = "\033[90m"
)
