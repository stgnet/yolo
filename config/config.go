package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

// ─── Core Constants ─────────────────────────────────────────────────────

const (
	// YoloDir is the directory (relative to the working directory) where
	// history, sub-agent results, and other state files are stored.
	YoloDir = ".yolo"

	// IMPORTANT: Source code is in current directory (.), NOT in yolo/
	// File paths should be relative to: /Users/sgriepentrog/src/yolo
	// Example: Use "tools_inbox.go", NOT "yolo/tools_inbox.go"
	SourceCodeLocation = "."

	// CRITICAL: Use the restart tool to rebuild, NEVER call os.Exit()
	// The restart tool handles: go build → go test → git commit → clean restart
	UseRestartTool = true

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
	FileNameRegex = regexp.MustCompile(`agent_(\S+)\.json`)
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

// ─── Configuration Management ─────────────────────────────────────────

// YoloConfigData is the top-level JSON structure for config.json.
type YoloConfigData struct {
	Version      int  `json:"version"`
	Model        string `json:"model,omitempty"`         // currently selected Ollama model
	TerminalMode bool   `json:"terminal_mode,omitempty"` // true = classic split-screen UI; false (default) = buffer mode
}

// YoloConfig owns the in-memory config and handles reading/writing to disk.
type YoloConfig struct {
	yoloDir    string
	configFile string
	Data       YoloConfigData
	mu         sync.Mutex
}

// NewYoloConfig creates a config manager that stores its file in yoloDir.
func NewYoloConfig(yoloDir string) *YoloConfig {
	return &YoloConfig{
		yoloDir:    yoloDir,
		configFile: filepath.Join(yoloDir, "config.json"),
		Data:       YoloConfigData{Version: 1},
	}
}

// Load reads config.json from disk. Returns true on success.
func (c *YoloConfig) Load() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.configFile)
	if err != nil {
		return false
	}
	if err := json.Unmarshal(data, &c.Data); err != nil {
		c.Data = YoloConfigData{Version: 1}
		return false
	}
	return true
}

// Save writes the current config to config.json atomically.
func (c *YoloConfig) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := os.MkdirAll(c.yoloDir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(c.Data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	tmp := c.configFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Rename(tmp, c.configFile); err != nil {
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}

// GetModel returns the configured model name.
func (c *YoloConfig) GetModel() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Data.Model
}

// SetModel updates the model and persists to disk.
func (c *YoloConfig) SetModel(model string) {
	c.mu.Lock()
	c.Data.Model = model
	c.mu.Unlock()
	c.Save()
}

// GetTerminalMode returns whether classic split-screen terminal mode is enabled.
func (c *YoloConfig) GetTerminalMode() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Data.TerminalMode
}

// SetTerminalMode updates the terminal mode setting and persists to disk.
func (c *YoloConfig) SetTerminalMode(enabled bool) {
	c.mu.Lock()
	c.Data.TerminalMode = enabled
	c.mu.Unlock()
	c.Save()
}
