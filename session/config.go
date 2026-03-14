// Package session provides configuration management for YOLO.
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

const (
	// YoloDir is the directory where history, sub-agent results, and other state are stored.
	YoloDir = ".yolo"

	// SourceCodeLocation indicates source code is in current directory.
	SourceCodeLocation = "."

	// UseRestartTool indicates to use restart tool instead of os.Exit().
	UseRestartTool = true

	// MaxContextMessages caps how many history messages are included in context window.
	MaxContextMessages = 40

	// MaxToolOutput limits tool output length (0 = unlimited).
	MaxToolOutput = 0

	// CommandTimeout is the maximum seconds a shell command can run.
	CommandTimeout = 30

	// ToolTimeout is the maximum seconds any tool execution is allowed to run.
	ToolTimeout = 60

	// MaxSubagentRounds is the maximum rounds a sub-agent is allowed.
	MaxSubagentRounds = 20

	// DefaultNumCtx is the default context-window size for Ollama.
	DefaultNumCtx = 8192

	// DefaultInputDelay is seconds to wait after user stops typing before sending input.
	DefaultInputDelay = 10
)

var (
	// HistoryFile is the default path to the conversation history JSON file.
	HistoryFile = filepath.Join(YoloDir, "history.json")

	// OllamaURL is the Ollama API base URL.
	OllamaURL = getEnvDefault("OLLAMA_URL", "http://localhost:11434")

	// NumCtxOverride forces context-window size for Ollama.
	NumCtxOverride = os.Getenv("YOLO_NUM_CTX")

	// SubagentDir is the directory where sub-agent result files are written.
	SubagentDir = filepath.Join(YoloDir, "subagents")

	// FileNameRegex matches sub-agent result files.
	FileNameRegex = regexp.MustCompile(`agent_(\S+)\.json`)
)

// getEnvDefault returns the value of an environment variable or fallback if unset.
func getEnvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// YoloConfigData is the top-level JSON structure for config.json.
type YoloConfigData struct {
	Version      int    `json:"version"`
	Model        string `json:"model,omitempty"`
	TerminalMode bool   `json:"terminal_mode,omitempty"`
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
