// Package config provides centralized configuration management with thread-safe access.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
)

// Config holds all YOLO application configuration in a thread-safe manner.
type Config struct {
	ModelName       atomic.Value // string
	OllamaEndpoint  atomic.Value // string
	OllamaURL       atomic.Value // string (for compatibility)
	WorkingDir      atomic.Value // string (yolo directory)
	TodoFile        atomic.Value // string
	LearnFile       atomic.Value // string
	EmailDir        atomic.Value // string
	SubagentDir     atomic.Value // string
	PromptDir       atomic.Value // string
	ContextFilePath atomic.Value // string
}

// cfg is the global configuration singleton.
var cfg *Config

func init() {
	cfg = DefaultConfig()
}

// DefaultConfig returns a new Config with default values initialized.
func DefaultConfig() *Config {
	c := &Config{}
	
	// Get home directory for defaults
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	
	defaults := map[string]string{
		"ModelName":       "qwen2.5-coder:7b",
		"OllamaEndpoint":  "http://localhost:11434/api/generate",
		"OllamaURL":       "http://localhost:11434",
		"WorkingDir":      ".",
		"TodoFile":        filepath.Join(homeDir, ".yolo_todo.json"),
		"LearnFile":       filepath.Join(homeDir, ".yolo_learn.json"),
		"EmailDir":        "/var/mail/b-haven.org/yolo/new/",
		"SubagentDir":     filepath.Join(homeDir, ".yolo_subagents"),
		"PromptDir":       "prompts",
		"ContextFilePath": "context.txt",
	}
	
	for key, value := range defaults {
		switch key {
		case "ModelName":
			c.ModelName.Store(value)
		case "OllamaEndpoint":
			c.OllamaEndpoint.Store(value)
		case "OllamaURL":
			c.OllamaURL.Store(value)
		case "WorkingDir":
			c.WorkingDir.Store(value)
		case "TodoFile":
			c.TodoFile.Store(value)
		case "LearnFile":
			c.LearnFile.Store(value)
		case "EmailDir":
			c.EmailDir.Store(value)
		case "SubagentDir":
			c.SubagentDir.Store(value)
		case "PromptDir":
			c.PromptDir.Store(value)
		case "ContextFilePath":
			c.ContextFilePath.Store(value)
		}
	}
	
	return c
}

// GetModelName returns the current model name.
func (c *Config) GetModelName() string {
	if v := c.ModelName.Load(); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "qwen2.5-coder:7b"
}

// SetModelName sets the model name.
func (c *Config) SetModelName(name string) {
	c.ModelName.Store(name)
}

// GetOllamaEndpoint returns the Ollama API endpoint.
func (c *Config) GetOllamaEndpoint() string {
	if v := c.OllamaEndpoint.Load(); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "http://localhost:11434/api/generate"
}

// SetOllamaEndpoint sets the Ollama API endpoint.
func (c *Config) SetOllamaEndpoint(endpoint string) {
	c.OllamaEndpoint.Store(endpoint)
}

// GetOllamaURL returns the base Ollama URL for compatibility.
func (c *Config) GetOllamaURL() string {
	if v := c.OllamaURL.Load(); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "http://localhost:11434"
}

// SetOllamaURL sets the base Ollama URL.
func (c *Config) SetOllamaURL(url string) {
	c.OllamaURL.Store(url)
}

// GetYoloDir returns the working directory (yolo dir).
func (c *Config) GetYoloDir() string {
	if v := c.WorkingDir.Load(); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "."
}

// SetYoloDir sets the working directory.
func (c *Config) SetYoloDir(path string) {
	c.WorkingDir.Store(path)
}

// GetTodoFile returns the todo file path.
func (c *Config) GetTodoFile() string {
	if v := c.TodoFile.Load(); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return filepath.Join(os.Getenv("HOME"), ".yolo_todo.json")
}

// SetTodoFile sets the todo file path.
func (c *Config) SetTodoFile(path string) {
	c.TodoFile.Store(path)
}

// GetLearnFile returns the learn file path.
func (c *Config) GetLearnFile() string {
	if v := c.LearnFile.Load(); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return filepath.Join(os.Getenv("HOME"), ".yolo_learn.json")
}

// SetLearnFile sets the learn file path.
func (c *Config) SetLearnFile(path string) {
	c.LearnFile.Store(path)
}

// GetEmailDir returns the email directory path.
func (c *Config) GetEmailDir() string {
	if v := c.EmailDir.Load(); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "/var/mail/b-haven.org/yolo/new/"
}

// SetEmailDir sets the email directory path.
func (c *Config) SetEmailDir(path string) {
	c.EmailDir.Store(path)
}

// GetSubagentDir returns the subagent directory path.
func (c *Config) GetSubagentDir() string {
	if v := c.SubagentDir.Load(); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return filepath.Join(os.Getenv("HOME"), ".yolo_subagents")
}

// SetSubagentDir sets the subagent directory path.
func (c *Config) SetSubagentDir(path string) {
	c.SubagentDir.Store(path)
}

// GetPromptDir returns the prompts directory.
func (c *Config) GetPromptDir() string {
	if v := c.PromptDir.Load(); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "prompts"
}

// SetPromptDir sets the prompts directory.
func (c *Config) SetPromptDir(path string) {
	c.PromptDir.Store(path)
}

// GetContextFilePath returns the context file path.
func (c *Config) GetContextFilePath() string {
	if v := c.ContextFilePath.Load(); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "context.txt"
}

// SetContextFilePath sets the context file path.
func (c *Config) SetContextFilePath(path string) {
	c.ContextFilePath.Store(path)
}

// Global accessor functions for backward compatibility during migration.
// These will be removed once all code is migrated to dependency injection.

// GetModelName returns the current model name from global config.
func GetModelName() string {
	return cfg.GetModelName()
}

// SetModelName sets the model name in global config.
func SetModelName(name string) {
	cfg.SetModelName(name)
}

// GetOllamaEndpoint returns the Ollama API endpoint from global config.
func GetOllamaEndpoint() string {
	return cfg.GetOllamaEndpoint()
}

// SetOllamaEndpoint sets the Ollama API endpoint in global config.
func SetOllamaEndpoint(endpoint string) {
	cfg.SetOllamaEndpoint(endpoint)
}

// GetOllamaURL returns the base Ollama URL from global config.
func GetOllamaURL() string {
	return cfg.GetOllamaURL()
}

// SetOllamaURL sets the base Ollama URL in global config.
func SetOllamaURL(url string) {
	cfg.SetOllamaURL(url)
}

// GetYoloDir returns the working directory from global config.
func GetYoloDir() string {
	return cfg.GetYoloDir()
}

// SetYoloDir sets the working directory in global config.
func SetYoloDir(path string) {
	cfg.SetYoloDir(path)
}

// GetTodoFile returns the todo file path from global config.
func GetTodoFile() string {
	return cfg.GetTodoFile()
}

// SetTodoFile sets the todo file path in global config.
func SetTodoFile(path string) {
	cfg.SetTodoFile(path)
}

// GetLearnFile returns the learn file path from global config.
func GetLearnFile() string {
	return cfg.GetLearnFile()
}

// SetLearnFile sets the learn file path in global config.
func SetLearnFile(path string) {
	cfg.SetLearnFile(path)
}

// GetEmailDir returns the email directory from global config.
func GetEmailDir() string {
	return cfg.GetEmailDir()
}

// SetEmailDir sets the email directory in global config.
func SetEmailDir(path string) {
	cfg.SetEmailDir(path)
}

// GetSubagentDir returns the subagent directory from global config.
func GetSubagentDir() string {
	return cfg.GetSubagentDir()
}

// SetSubagentDir sets the subagent directory in global config.
func SetSubagentDir(path string) {
	cfg.SetSubagentDir(path)
}

// GetPromptDir returns the prompts directory from global config.
func GetPromptDir() string {
	return cfg.GetPromptDir()
}

// SetPromptDir sets the prompts directory in global config.
func SetPromptDir(path string) {
	cfg.SetPromptDir(path)
}

// GetContextFilePath returns the context file path from global config.
func GetContextFilePath() string {
	return cfg.GetContextFilePath()
}

// SetContextFilePath sets the context file path in global config.
func SetContextFilePath(path string) {
	cfg.SetContextFilePath(path)
}

// WorkingDir is an alias for GetYoloDir (for backward compatibility).
func WorkingDir() string {
	return cfg.GetYoloDir()
}

// SetWorkingDir sets the working directory in global config.
func SetWorkingDir(path string) {
	cfg.SetYoloDir(path)
}
// String returns a human-readable representation of current config.
func (c *Config) String() string {
	return fmt.Sprintf(`YOLO Configuration:
  Model: %s
  Ollama Endpoint: %s
  Ollama URL: %s
  Working Dir: %s
  Todo: %s
  Learn: %s
  Email: %s
  Subagents: %s
  Prompts: %s
  Context: %s`,
		c.GetModelName(),
		c.GetOllamaEndpoint(),
		c.GetOllamaURL(),
		c.GetYoloDir(),
		c.GetTodoFile(),
		c.GetLearnFile(),
		c.GetEmailDir(),
		c.GetSubagentDir(),
		c.GetPromptDir(),
		c.GetContextFilePath(),
	)
}
