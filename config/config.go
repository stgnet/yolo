// Package config provides centralized configuration management for YOLO.
package config

import (
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

// Config holds all runtime configuration for YOLO.
type Config struct {
	mu sync.RWMutex

	// Immutable constants
	YoloDir             string
	HistoryFile         string
	SubagentDir         string
	DefaultOllamaURL    string
	CommandTimeout      int
	ToolTimeout         int
	MaxSubagentRounds   int
	DefaultNumCtx       int
	DefaultInputDelay   int

	// Mutable configuration (protected by mutex)
	OllamaURL      string
	NumCtxOverride string
}

var fileNameRegex = regexp.MustCompile(`agent_(\S+)\.json`)

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

	YoloDirDefault        = ".yolo"
	MaxContextMessages    = 40
	MaxToolOutput         = 0
	CommandTimeoutDefault = 30
	ToolTimeoutDefault    = 60
	MaxSubagentRoundsDef  = 20
	DefaultNumCtxDefault  = 8192
	DefaultInputDelayDef  = 10
	DefaultOllamaURLDef   = "http://localhost:11434"
)

func DefaultConfig() *Config {
	return &Config{
		YoloDir:           YoloDirDefault,
		HistoryFile:       filepath.Join(YoloDirDefault, "history.json"),
		SubagentDir:       filepath.Join(YoloDirDefault, "subagents"),
		DefaultOllamaURL:  DefaultOllamaURLDef,
		CommandTimeout:    CommandTimeoutDefault,
		ToolTimeout:       ToolTimeoutDefault,
		MaxSubagentRounds: MaxSubagentRoundsDef,
		DefaultNumCtx:     DefaultNumCtxDefault,
		DefaultInputDelay: DefaultInputDelayDef,
		OllamaURL:         getEnvDefault("OLLAMA_URL", DefaultOllamaURLDef),
		NumCtxOverride:    os.Getenv("YOLO_NUM_CTX"),
	}
}

func (c *Config) ReloadFromEnv() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if url := os.Getenv("OLLAMA_URL"); url != "" {
		c.OllamaURL = url
	}
	c.NumCtxOverride = os.Getenv("YOLO_NUM_CTX")
}

func (c *Config) GetOllamaURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.OllamaURL
}

func (c *Config) SetOllamaURL(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.OllamaURL = url
}

func (c *Config) GetNumCtxOverride() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.NumCtxOverride
}

func (c *Config) SetNumCtxOverride(val string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.NumCtxOverride = val
}

func (c *Config) GetHistoryFile() string { return c.HistoryFile }
func (c *Config) GetSubagentDir() string  { return c.SubagentDir }
func (c *Config) GetYoloDir() string      { return c.YoloDir }

func getEnvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
