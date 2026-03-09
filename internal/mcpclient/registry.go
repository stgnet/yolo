package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// ToolRegistry manages MCP tools available to YOLO
type ToolRegistry struct {
	tools   map[string]*Client
	configs map[string]ToolConfig
}

// ToolConfig represents a configured MCP tool
type ToolConfig struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	Description string `json:"description"`
	Priority    int    `json:"priority,omitempty"`
	Enabled     bool   `json:"enabled"`
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:   make(map[string]*Client),
		configs: make(map[string]ToolConfig),
	}
}

// LoadTools loads tools from configuration file
func (r *ToolRegistry) LoadTools(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	var configs []ToolConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}

		r.configs[cfg.Name] = cfg

		client, err := NewServer(context.Background(), cfg.Command, nil, os.Environ())
		if err != nil {
			fmt.Printf("Warning: failed to create client for %s: %v\n", cfg.Name, err)
			continue
		}

		r.tools[cfg.Name] = client
		fmt.Printf("Registered MCP tool: %s (%s)\n", cfg.Name, cfg.Description)
	}

	return nil
}

// GetTool returns a tool by name
func (r *ToolRegistry) GetTool(name string) (*Client, bool) {
	client, ok := r.tools[name]
	return client, ok
}

// CallTool invokes a tool with the given arguments
func (r *ToolRegistry) CallTool(ctx context.Context, name string, args map[string]any) (*CallToolResultParams, error) {
	client, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	result, err := client.CallTool(ctx, "", args)
	if err != nil {
		return nil, fmt.Errorf("calling tool %s: %w", name, err)
	}

	return result, nil
}

// ListTools returns names of all registered tools
func (r *ToolRegistry) ListTools() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		config, ok := r.configs[name]
		if ok && config.Enabled {
			names = append(names, fmt.Sprintf("%s:%d", name, config.Priority))
		}
	}
	return names
}

// GetToolDescription returns description for a tool
func (r *ToolRegistry) GetToolDescription(name string) string {
	config, ok := r.configs[name]
	if !ok {
		return ""
	}
	return config.Description
}
