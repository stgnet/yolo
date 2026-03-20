package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"
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

	// ToolTimeout is the maximum wall-clock seconds any tool execution
	// is allowed to run before being reported as hung to the agent.
	// This catches deadlocks, infinite loops, and other hangs.
	ToolTimeout = 60

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

// ─── Configuration ───────────────────────────────────────────────────────

var yoloConfig = &Config{}

// Config holds all mutable configuration with thread-safe access.
type Config struct {
	historyFile    atomic.Value // stores string
	ollamaURL      atomic.Value // stores string
	numCtxOverride atomic.Value // stores string
	subagentDir    atomic.Value // stores string
	fileNameRegex  *regexp.Regexp
	mu             sync.RWMutex // for future use if needed
}

// GetHistoryFile returns the current history file path.
func (c *Config) GetHistoryFile() string {
	if v, ok := c.historyFile.Load().(string); ok {
		return v
	}
	return filepath.Join(YoloDir, "history.json")
}

// SetHistoryFile updates the history file path.
func (c *Config) SetHistoryFile(path string) {
	c.historyFile.Store(path)
}

// GetOllamaURL returns the current Ollama API URL.
func (c *Config) GetOllamaURL() string {
	if v, ok := c.ollamaURL.Load().(string); ok {
		return v
	}
	return "http://localhost:11434"
}

// SetOllamaURL updates the Ollama API URL.
func (c *Config) SetOllamaURL(url string) {
	c.ollamaURL.Store(url)
}

// GetNumCtxOverride returns the current context window override.
func (c *Config) GetNumCtxOverride() string {
	if v, ok := c.numCtxOverride.Load().(string); ok {
		return v
	}
	return ""
}

// SetNumCtxOverride updates the context window override.
func (c *Config) SetNumCtxOverride(val string) {
	c.numCtxOverride.Store(val)
}

// GetSubagentDir returns the current subagent directory path.
func (c *Config) GetSubagentDir() string {
	if v, ok := c.subagentDir.Load().(string); ok {
		return v
	}
	return filepath.Join(YoloDir, "subagents")
}

// SetSubagentDir updates the subagent directory path.
func (c *Config) SetSubagentDir(path string) {
	c.subagentDir.Store(path)
}

// GetFileNameRegex returns the file name regex pattern.
func (c *Config) GetFileNameRegex() *regexp.Regexp {
	return c.fileNameRegex
}

// fileNameRegex matches sub-agent result files (agent_1.json, agent_test_123.json, etc.).
var filePattern = regexp.MustCompile(`agent_(\S+)\.json`)

// Legacy global variables for backward compatibility (deprecated)
var (
	HistoryFile    string
	OllamaURL      string
	NumCtxOverride string
	SubagentDir    string
)

// Initialize global config with proper atomic values
func init() {
	yoloConfig.historyFile.Store(filepath.Join(YoloDir, "history.json"))
	yoloConfig.ollamaURL.Store(getEnvDefault("OLLAMA_URL", "http://localhost:11434"))
	yoloConfig.numCtxOverride.Store(os.Getenv("YOLO_NUM_CTX"))
	yoloConfig.subagentDir.Store(filepath.Join(YoloDir, "subagents"))
	yoloConfig.fileNameRegex = filePattern

	// Initialize legacy globals from config after config is ready
	HistoryFile = yoloConfig.GetHistoryFile()
	OllamaURL = yoloConfig.GetOllamaURL()
	NumCtxOverride = yoloConfig.GetNumCtxOverride()
	SubagentDir = yoloConfig.GetSubagentDir()
}

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
