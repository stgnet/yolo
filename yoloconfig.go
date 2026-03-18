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
// YoloConfig manages persistent configuration stored in .yolo/config.json,
// separate from conversation history. This survives history resets and
// can be extended with additional settings in the future.

// YoloConfigData is the top-level JSON structure for config.json.
type YoloConfigData struct {
	Version      int    `json:"version"`
	Model        string `json:"model,omitempty"`         // currently selected Ollama model
	TerminalMode bool   `json:"terminal_mode,omitempty"` // true = classic split-screen UI; false (default) = buffer mode
	DebugMode    *bool  `json:"debug_mode,omitempty"`    // false (default) = cleaner output; true = show full tool args/results verbatim
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
