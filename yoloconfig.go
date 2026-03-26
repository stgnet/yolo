package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	TTSVoice     string `json:"tts_voice,omitempty"`     // TTS voice name (default: platform-specific)
	TTSEnabled   *bool  `json:"tts_enabled,omitempty"`   // nil = default (enabled if backend found); true/false = explicit
	EmailFrom    string `json:"email_from,omitempty"`    // sender email address (default: yolo@localhost)
	EmailTo      string `json:"email_to,omitempty"`      // default recipient email address
	InboxPath    string `json:"inbox_path,omitempty"`    // Maildir inbox path (default: empty = no inbox)
	UserAgent    string `json:"user_agent,omitempty"`    // HTTP User-Agent string (default: YOLO-Agent/1.0)
	TempDir      string `json:"temp_dir,omitempty"`      // temporary file directory (default: os.TempDir())
}

// YoloConfig owns the in-memory config and handles reading/writing to disk.
type YoloConfig struct {
	yoloDir        string // working directory
	configFile     string // path to .yolo/config.json
	ollamaURL      string // Ollama API base URL (from OLLAMA_URL env, default http://localhost:11434)
	numCtxOverride string // context window override (from YOLO_NUM_CTX env)
	subagentDir    string // path to .yolo/subagents/
	Data           YoloConfigData
	loaded         bool // true after Load() succeeds or first Save() on new config
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
// Must be called before any Set*/Save to avoid overwriting saved settings.
func (c *YoloConfig) Load() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.configFile)
	if err != nil {
		// File doesn't exist yet — mark as loaded so Save() knows this is
		// intentional (first run) rather than a missed Load().
		c.loaded = true
		return false
	}
	if err := json.Unmarshal(data, &c.Data); err != nil {
		c.Data = YoloConfigData{Version: 1}
		return false
	}
	c.loaded = true
	return true
}

// Save writes the current config to config.json atomically.
// If Load() was never called, Save reads the existing file first and merges
// in-memory changes on top, so that fields set by a prior session are not
// silently dropped.
func (c *YoloConfig) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.saveLocked()
}

// saveLocked performs the actual save. Must be called with c.mu held.
func (c *YoloConfig) saveLocked() error {
	// Guard: if Load() was never called, read the existing file and merge
	// our in-memory fields on top so we don't wipe saved settings.
	if !c.loaded {
		if existing, err := os.ReadFile(c.configFile); err == nil {
			var disk YoloConfigData
			if json.Unmarshal(existing, &disk) == nil {
				c.mergeOnto(&disk)
				c.Data = disk
			}
		}
		c.loaded = true
	}

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

// mergeOnto applies non-zero in-memory fields from c.Data onto disk,
// preserving any disk fields that are not set in memory.
// Must be called with c.mu held.
func (c *YoloConfig) mergeOnto(disk *YoloConfigData) {
	if c.Data.Model != "" {
		disk.Model = c.Data.Model
	}
	if c.Data.TerminalMode {
		disk.TerminalMode = c.Data.TerminalMode
	}
	if c.Data.DebugMode != nil {
		disk.DebugMode = c.Data.DebugMode
	}
	if c.Data.AutoMode != nil {
		disk.AutoMode = c.Data.AutoMode
	}
	if c.Data.ThinkMode != nil {
		disk.ThinkMode = c.Data.ThinkMode
	}
	if c.Data.TTSVoice != "" {
		disk.TTSVoice = c.Data.TTSVoice
	}
	if c.Data.TTSEnabled != nil {
		disk.TTSEnabled = c.Data.TTSEnabled
	}
	if c.Data.EmailFrom != "" {
		disk.EmailFrom = c.Data.EmailFrom
	}
	if c.Data.EmailTo != "" {
		disk.EmailTo = c.Data.EmailTo
	}
	if c.Data.InboxPath != "" {
		disk.InboxPath = c.Data.InboxPath
	}
	if c.Data.UserAgent != "" {
		disk.UserAgent = c.Data.UserAgent
	}
	if c.Data.TempDir != "" {
		disk.TempDir = c.Data.TempDir
	}
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
	defer c.mu.Unlock()
	c.Data.Model = model
	c.saveLocked()
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
	defer c.mu.Unlock()
	c.Data.TerminalMode = enabled
	c.saveLocked()
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
	defer c.mu.Unlock()
	c.Data.DebugMode = &enabled
	c.saveLocked()
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
	defer c.mu.Unlock()
	c.Data.AutoMode = &enabled
	c.saveLocked()
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
	defer c.mu.Unlock()
	c.Data.ThinkMode = &enabled
	c.saveLocked()
}

// GetTTSVoice returns the configured TTS voice name, or "" for platform default.
func (c *YoloConfig) GetTTSVoice() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Data.TTSVoice
}

// SetTTSVoice updates the TTS voice and persists to disk.
func (c *YoloConfig) SetTTSVoice(voice string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Data.TTSVoice = voice
	c.saveLocked()
}

// GetTTSEnabled returns the configured TTS enabled state.
// Returns nil if not explicitly set (use platform default).
func (c *YoloConfig) GetTTSEnabled() *bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Data.TTSEnabled
}

// SetTTSEnabled updates the TTS enabled state and persists to disk.
func (c *YoloConfig) SetTTSEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Data.TTSEnabled = &enabled
	c.saveLocked()
}

// GetEmailFrom returns the configured sender email address.
func (c *YoloConfig) GetEmailFrom() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Data.EmailFrom != "" {
		return c.Data.EmailFrom
	}
	return "yolo@localhost"
}

// SetEmailFrom updates the sender email and persists to disk.
func (c *YoloConfig) SetEmailFrom(from string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Data.EmailFrom = from
	c.saveLocked()
}

// GetEmailTo returns the configured default recipient email address.
func (c *YoloConfig) GetEmailTo() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Data.EmailTo
}

// SetEmailTo updates the default recipient email and persists to disk.
func (c *YoloConfig) SetEmailTo(to string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Data.EmailTo = to
	c.saveLocked()
}

// GetInboxPath returns the configured Maildir inbox path, or "" if not set.
func (c *YoloConfig) GetInboxPath() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Data.InboxPath
}

// SetInboxPath updates the inbox path and persists to disk.
func (c *YoloConfig) SetInboxPath(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Data.InboxPath = path
	c.saveLocked()
}

// GetUserAgent returns the configured HTTP User-Agent string.
func (c *YoloConfig) GetUserAgent() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Data.UserAgent != "" {
		return c.Data.UserAgent
	}
	return "YOLO-Agent/1.0"
}

// SetUserAgent updates the User-Agent string and persists to disk.
func (c *YoloConfig) SetUserAgent(ua string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Data.UserAgent = ua
	c.saveLocked()
}

// GetTempDir returns the configured temporary directory.
func (c *YoloConfig) GetTempDir() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Data.TempDir != "" {
		return c.Data.TempDir
	}
	return os.TempDir()
}

// SetTempDir updates the temp directory and persists to disk.
func (c *YoloConfig) SetTempDir(dir string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Data.TempDir = dir
	c.saveLocked()
}

// SetByName sets a config field by its JSON key name. Returns an error
// for unknown keys. Boolean fields accept "true"/"false"/"on"/"off".
func (c *YoloConfig) SetByName(key, value string) error {
	switch key {
	case "model":
		c.SetModel(value)
	case "email_from":
		c.SetEmailFrom(value)
	case "email_to":
		c.SetEmailTo(value)
	case "inbox_path":
		c.SetInboxPath(value)
	case "user_agent":
		c.SetUserAgent(value)
	case "temp_dir":
		c.SetTempDir(value)
	case "tts_voice":
		c.SetTTSVoice(value)
	case "terminal_mode":
		c.SetTerminalMode(parseBool(value))
	case "debug_mode":
		c.SetDebugMode(parseBool(value))
	case "auto_mode":
		c.SetAutoMode(parseBool(value))
	case "think_mode":
		c.SetThinkMode(parseBool(value))
	case "tts_enabled":
		c.SetTTSEnabled(parseBool(value))
	default:
		return fmt.Errorf("unknown config key '%s'", key)
	}
	return nil
}

// GetAll returns all config fields as a map for display.
func (c *YoloConfig) GetAll() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	m := map[string]string{
		"model":         c.Data.Model,
		"email_from":    c.Data.EmailFrom,
		"email_to":      c.Data.EmailTo,
		"inbox_path":    c.Data.InboxPath,
		"user_agent":    c.Data.UserAgent,
		"temp_dir":      c.Data.TempDir,
		"tts_voice":     c.Data.TTSVoice,
		"terminal_mode": fmt.Sprintf("%v", c.Data.TerminalMode),
	}
	if c.Data.DebugMode != nil {
		m["debug_mode"] = fmt.Sprintf("%v", *c.Data.DebugMode)
	} else {
		m["debug_mode"] = "false"
	}
	if c.Data.AutoMode != nil {
		m["auto_mode"] = fmt.Sprintf("%v", *c.Data.AutoMode)
	} else {
		m["auto_mode"] = "false"
	}
	if c.Data.ThinkMode != nil {
		m["think_mode"] = fmt.Sprintf("%v", *c.Data.ThinkMode)
	} else {
		m["think_mode"] = "true"
	}
	if c.Data.TTSEnabled != nil {
		m["tts_enabled"] = fmt.Sprintf("%v", *c.Data.TTSEnabled)
	} else {
		m["tts_enabled"] = "(auto)"
	}
	return m
}

// parseBool converts common boolean string representations to bool.
func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "on", "yes", "1":
		return true
	default:
		return false
	}
}
