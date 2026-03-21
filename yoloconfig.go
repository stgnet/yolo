package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ─── Yolo Config ─────────────────────────────────────────────────────
//
// YoloConfig is the single configuration system for YOLO. It manages both
// persistent settings (stored in .yolo/config.json) and runtime paths
// derived from environment variables and working directory.

// YoloConfigData is the top-level JSON structure for config.json.
type YoloConfigData struct {
	Version      int    `json:"version"`
	Model        string `json:"model,omitempty"`         // currently selected Ollama model
	TerminalMode bool   `json:"terminal_mode,omitempty"` // true = classic split-screen UI; false (default) = buffer mode
	DebugMode    *bool  `json:"debug_mode,omitempty"`    // false (default) = cleaner output; true = show full tool args/results verbatim
	AutoMode     *bool  `json:"auto_mode,omitempty"`     // false (default) = wait for user input; true = enable autonomous mode
	ThinkMode    *bool  `json:"think_mode,omitempty"`    // true (default) = show thinking output; false = hide thinking blocks
}

// YoloConfig owns the in-memory config and handles reading/writing to disk.
type YoloConfig struct {
	yoloDir        string // working directory
	configFile     string // path to .yolo/config.json
	ollamaURL      string // Ollama API base URL (from OLLAMA_URL env, default http://localhost:11434)
	numCtxOverride string // context window override (from YOLO_NUM_CTX env)
	subagentDir    string // path to .yolo/subagents/
	Data           YoloConfigData
	mu             sync.Mutex
}

// NewYoloConfig creates a config manager rooted in the given working directory.
// Runtime paths are derived from the working directory and environment variables.
func NewYoloConfig(workDir string) *YoloConfig {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	return &YoloConfig{
		yoloDir:        workDir,
		configFile:     filepath.Join(workDir, ".yolo", "config.json"),
		ollamaURL:      ollamaURL,
		numCtxOverride: os.Getenv("YOLO_NUM_CTX"),
		subagentDir:    filepath.Join(workDir, ".yolo", "subagents"),
		Data:           YoloConfigData{Version: 1},
	}
}

// GetOllamaURL returns the Ollama API base URL.
func (c *YoloConfig) GetOllamaURL() string {
	return c.ollamaURL
}

// GetNumCtxOverride returns the context window override from YOLO_NUM_CTX, or "".
func (c *YoloConfig) GetNumCtxOverride() string {
	return c.numCtxOverride
}

// GetSubagentDir returns the subagent results directory.
func (c *YoloConfig) GetSubagentDir() string {
	return c.subagentDir
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

	if err := os.MkdirAll(filepath.Dir(c.configFile), 0o755); err != nil {
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

// GetDebugMode returns whether debug mode is enabled. Defaults to false
// when not explicitly set for cleaner output in normal operation.
func (c *YoloConfig) GetDebugMode() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Data.DebugMode == nil {
		return false // default off
	}
	return *c.Data.DebugMode
}

// SetDebugMode updates the debug mode setting and persists to disk.
func (c *YoloConfig) SetDebugMode(enabled bool) {
	c.mu.Lock()
	c.Data.DebugMode = &enabled
	c.mu.Unlock()
	c.Save()
}

// GetAutoMode returns whether autonomous mode is enabled. Defaults to false
// when not explicitly set, requiring user input for operation.
func (c *YoloConfig) GetAutoMode() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Data.AutoMode == nil {
		return false // default off
	}
	return *c.Data.AutoMode
}

// SetAutoMode updates the auto mode setting and persists to disk.
func (c *YoloConfig) SetAutoMode(enabled bool) {
	c.mu.Lock()
	c.Data.AutoMode = &enabled
	c.mu.Unlock()
	c.Save()
}

// GetThinkMode returns whether thinking output is shown. Defaults to true
// when not explicitly set, showing thinking blocks by default.
func (c *YoloConfig) GetThinkMode() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Data.ThinkMode == nil {
		return true // default on
	}
	return *c.Data.ThinkMode
}

// SetThinkMode updates the think mode setting and persists to disk.
func (c *YoloConfig) SetThinkMode(enabled bool) {
	c.mu.Lock()
	c.Data.ThinkMode = &enabled
	c.mu.Unlock()
	c.Save()
}
