// Package tools provides tool definitions and execution for the YOLO agent.
package tools

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Tool represents a callable operation that the agent can perform.
type Tool struct {
	Name        string
	Description string
	Params      map[string]ToolParam
	Required    []string
}

// ToolParam describes a parameter for a tool.
type ToolParam struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ToolExecutor dispatches tool calls to their implementations.
type ToolExecutor struct {
	mu           sync.RWMutex
	tools        map[string]*Tool
	toolFuncs    map[string]func(context.Context, map[string]interface{}) (string, error)
	timeout      time.Duration
	baseDir      string
	executor     *YoloExecutor // Reference back to orchestration layer
}

// ToolConfig holds configuration for the ToolExecutor.
type ToolConfig struct {
	BaseDir    string
	Timeout    time.Duration
	ToolFuncs  []string // Which tools to enable
}

// DefaultToolConfig returns a default tool executor configuration.
func DefaultToolConfig() ToolConfig {
	return ToolConfig{
		BaseDir:   ".",
		Timeout:   60 * time.Second,
		ToolFuncs: []string{"all"}, // Enable all tools by default
	}
}

// NewToolExecutor creates a new tool executor with the given configuration.
func NewToolExecutor(cfg ToolConfig) *ToolExecutor {
	executor := &ToolExecutor{
		tools:     make(map[string]*Tool),
		toolFuncs: make(map[string]func(context.Context, map[string]interface{}) (string, error)),
		timeout:   cfg.Timeout,
		baseDir:   cfg.BaseDir,
	}

	// Register standard tools
	executor.RegisterTool(executor.tool_readFile)
	executor.RegisterTool(executor.tool_listFiles)
	executor.RegisterTool(executor.tool_writeFile)
	executor.RegisterTool(executor.tool_editFile)
	executor.RegisterTool(executor.tool_shellCommand)
	executor.RegisterTool(executor.tool_searchFiles)
	executor.RegisterTool(executor.tool_readWebpage)
	executor.RegisterTool(executor.tool_webSearch)
	executor.RegisterTool(executor.tool_listSubagents)
	executor.RegisterTool(executor.tool_readSubagentResult)
	executor.RegisterTool(executor.tool_restart)

	return executor
}

// RegisterTool adds a tool to the executor.
func (e *ToolExecutor) RegisterTool(tool *Tool, fn func(map[string]interface{}) (string, error)) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.tools[tool.Name] = tool
	e.toolFuncs[tool.Name] = func(ctx context.Context, args map[string]interface{}) (string, error) {
		return fn(args)
	}
}

// SetExecutor sets the orchestration layer reference for sub-agent management.
func (e *ToolExecutor) SetExecutor(executor *YoloExecutor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	executor.executor = executor
}

// Execute runs a tool with the given name and arguments.
func (e *ToolExecutor) Execute(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	e.mu.RLock()
	toolFunc, ok := e.toolFuncs[name]
	e.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}

	// Create a context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Execute the tool with cancellation support
	resultChan := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		result, err := toolFunc(ctxWithTimeout, args)
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return "", fmt.Errorf("tool execution failed: %w", err)
	case <-ctxWithTimeout.Done():
		return "", fmt.Errorf("tool execution timed out after %v", e.timeout)
	}
}

// ListTools returns all available tools.
func (e *ToolExecutor) ListTools() []*Tool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*Tool, 0, len(e.tools))
	for _, tool := range e.tools {
		result = append(result, tool)
	}
	return result
}

// GetTool returns a specific tool by name.
func (e *ToolExecutor) GetTool(name string) (*Tool, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	tool, ok := e.tools[name]
	return tool, ok
}
