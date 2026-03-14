// Package config provides configuration management for YOLO agent.
// It handles loading from YAML/JSON config files and environment variables.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// YoloConfig is the main configuration structure for the YOLO agent.
type YoloConfig struct {
	// Paths and directories
	BaseDir     string        `yaml:"base_dir,omitempty"`
	YoloDir     string        `yaml:"yolo_dir,omitempty"`
	HistoryFile string        `yaml:"history_file,omitempty"`
	SubagentDir string        `yaml:"subagent_dir,omitempty"`
	KnowledgeBasePath string  `yaml:"knowledge_base_path,omitempty"`

	// Ollama configuration
	OllamaURL     string `yaml:"ollama_url,omitempty"`
	Model         string `yaml:"model,omitempty"`
	DefaultNumCtx int    `yaml:"default_num_ctx,omitempty"`
	NumCtxOverride string `yaml:"num_ctx_override,omitempty"`

	// Timeout and limits
	CommandTimeout    time.Duration `yaml:"command_timeout,omitempty"`
	ToolTimeout       time.Duration `yaml:"tool_timeout,omitempty"`
	MaxContextMessages int          `yaml:"max_context_messages,omitempty"`
	MaxToolOutput     int           `yaml:"max_tool_output,omitempty"`
	MaxSubagentRounds int           `yaml:"max_subagent_rounds,omitempty"`

	// Input settings
	DefaultInputDelay time.Duration `yaml:"default_input_delay,omitempty"`

	// Tool settings
	UseRestartTool bool `yaml:"use_restart_tool,omitempty"`

	// Source code location (for tools that need to reference files)
	SourceCodeLocation string `yaml:"source_code_location,omitempty"`

	// Internal state (not saved, reset on startup)
	fileNameRegex *regexp.Regexp
}

// DefaultConfig returns a new YoloConfig with sensible defaults.
func DefaultConfig() *YoloConfig {
	return &YoloConfig{
		BaseDir:            ".",
		YoloDir:            ".yolo",
		OllamaURL:          "http://localhost:11434",
		Model:              "", // User must select on first run
		DefaultNumCtx:      8192,
		CommandTimeout:     30 * time.Second,
		ToolTimeout:        60 * time.Second,
		MaxContextMessages: 40,
		MaxToolOutput:      0, // 0 = unlimited
		MaxSubagentRounds:  20,
		DefaultInputDelay:  10 * time.Second,
		UseRestartTool:     true,
		SourceCodeLocation: ".",
	}
}

// NewYoloConfig creates a new configuration rooted at the given directory.
func NewYoloConfig(baseDir string) *YoloConfig {
	cfg := DefaultConfig()
	cfg.BaseDir = baseDir
	cfg.setupPaths()
	return cfg
}

// setupPaths initializes file paths based on YoloDir and BaseDir.
func (c *YoloConfig) setupPaths() {
	c.HistoryFile = filepath.Join(c.YoloDir, "history.json")
	c.SubagentDir = filepath.Join(c.YoloDir, "subagents")
	if c.KnowledgeBasePath == "" {
		c.KnowledgeBasePath = filepath.Join(c.YoloDir, "knowledge.md")
	}

	// Initialize the regex for subagent file names
	c.fileNameRegex = regexp.MustCompile(`agent_(\S+)\.json`)
}

// LoadConfig attempts to load configuration from a config file.
// It tries .yolo/config.yaml first, then falls back to defaults.
func LoadConfig(baseDir string) (*YoloConfig, error) {
	cfg := DefaultConfig()
	cfg.BaseDir = baseDir

	// Try to load from YAML config
	yamlPath := filepath.Join(baseDir, cfg.YoloDir, "config.yaml")
	if data, err := os.ReadFile(yamlPath); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
		cfg.setupPaths()

		// Override with environment variables
		cfg.applyEnvOverrides()
		return cfg, nil
	}

	// Try JSON config as fallback
	jsonPath := filepath.Join(baseDir, cfg.YoloDir, "config.json")
	if data, err := os.ReadFile(jsonPath); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
		cfg.setupPaths()
		cfg.applyEnvOverrides()
		return cfg, nil
	}

	// No config file found, use defaults with env overrides
	cfg.applyEnvOverrides()
	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the config.
func (c *YoloConfig) applyEnvOverrides() {
	if v := os.Getenv("OLLAMA_URL"); v != "" {
		c.OllamaURL = v
	}

	if v := os.Getenv("YOLO_NUM_CTX"); v != "" {
		c.NumCtxOverride = v
	}

	// Path overrides
	if v := os.Getenv("YOLO_DIR"); v != "" {
		c.YoloDir = v
		c.setupPaths()
	}

	if v := os.Getenv("HISTORY_FILE"); v != "" {
		c.HistoryFile = v
	}

	if v := os.Getenv("SUBAGENT_DIR"); v != "" {
		c.SubagentDir = v
	}

	// Other settings can be added here as needed
}

// GetContextLength determines the context window size for the current model.
func (c *YoloConfig) GetContextLength() int {
	if c.NumCtxOverride != "" {
		// Try to parse as integer
		var ctxLen int
		if _, err := fmt.Sscanf(c.NumCtxOverride, "%d", &ctxLen); err == nil {
			return ctxLen
		}
	}
	return c.DefaultNumCtx
}

// GetFileNameRegex returns the compiled regex for matching subagent files.
func (c *YoloConfig) GetFileNameRegex() *regexp.Regexp {
	return c.fileNameRegex
}

// Save writes the current configuration to a YAML file.
func (c *YoloConfig) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filepath.Join(c.BaseDir, c.YoloDir, "config.yaml"), data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// IsModelSelected returns true if a model has been configured.
func (c *YoloConfig) IsModelSelected() bool {
	return c.Model != ""
}

// SetModel updates the model name and saves the configuration.
func (c *YoloConfig) SetModel(modelName string) error {
	c.Model = modelName
	return c.Save()
}

// GetModel returns the current model name.
func (c *YoloConfig) GetModel() string {
	return c.Model
}

// Validate checks if the configuration is valid.
// Returns an error if required settings are missing or invalid.
func (c *YoloConfig) Validate() error {
	if c.OllamaURL == "" {
		return fmt.Errorf("ollama_url is required")
	}

	if c.ToolTimeout <= 0 {
		return fmt.Errorf("tool_timeout must be positive")
	}

	if c.MaxContextMessages <= 0 {
		return fmt.Errorf("max_context_messages must be positive")
	}

	if !filepath.IsAbs(c.YoloDir) && !filepath.IsAbs(c.BaseDir) {
		// This is OK - both are relative paths which is expected for development
	}

	return nil
}
