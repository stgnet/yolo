package mcpclient

import (
	"context"
	"os"
	"testing"
)

func TestToolRegistry(t *testing.T) {
	t.Run("New registry created", func(t *testing.T) {
		registry := NewToolRegistry()
		if registry == nil {
			t.Fatal("Expected non-nil registry")
		}
		if registry.tools == nil {
			t.Error("Expected non-nil tools map")
		}
	})

	t.Run("Load valid config", func(t *testing.T) {
		configContent := `[
			{
				"name": "test-tool-1",
				"command": "node /path/to/tool.mjs",
				"description": "A test tool for testing",
				"priority": 1,
				"enabled": true
			}
		]`

		tmpfile := t.TempDir() + "/tools.json"
		if err := os.WriteFile(tmpfile, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write temp config: %v", err)
		}

		registry := NewToolRegistry()
		// LoadTools is resilient - it logs warnings for failed tools but doesn't return errors
		err := registry.LoadTools(tmpfile)
		if err != nil {
			t.Errorf("LoadTools should not error on failed tool startup, got: %v", err)
		}
		// Since the command doesn't exist, the tool won't be registered (but no error returned)
		tools := registry.ListTools()
		if len(tools) != 0 {
			t.Error("Expected no tools loaded for non-existent command")
		}
	})

	t.Run("Get unknown tool", func(t *testing.T) {
		registry := NewToolRegistry()
		client, ok := registry.GetTool("nonexistent")
		if ok {
			t.Error("Expected false for non-existent tool")
		}
		if client != nil {
			t.Error("Expected nil client for non-existent tool")
		}
	})

	t.Run("Call unknown tool", func(t *testing.T) {
		ctx := context.Background()
		registry := NewToolRegistry()
		result, err := registry.CallTool(ctx, "nonexistent", map[string]any{})
		if err == nil {
			t.Error("Expected error calling non-existent tool")
		}
		if result != nil {
			t.Error("Expected nil result for non-existent tool")
		}
	})

	t.Run("List tools empty", func(t *testing.T) {
		registry := NewToolRegistry()
		tools := registry.ListTools()
		if len(tools) != 0 {
			t.Errorf("Expected 0 tools, got %d", len(tools))
		}
	})

	t.Run("Get tool description unknown", func(t *testing.T) {
		registry := NewToolRegistry()
		desc := registry.GetToolDescription("nonexistent")
		if desc != "" {
			t.Errorf("Expected empty description, got %q", desc)
		}
	})

	t.Run("Load disabled tool", func(t *testing.T) {
		configContent := `[
			{
				"name": "disabled-tool",
				"command": "echo test",
				"description": "A disabled tool",
				"enabled": false
			}
		]`

		tmpfile := t.TempDir() + "/tools.json"
		if err := os.WriteFile(tmpfile, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write temp config: %v", err)
		}

		registry := NewToolRegistry()
		// LoadTools should not error - disabled tools are silently skipped
		err := registry.LoadTools(tmpfile)
		if err != nil {
			t.Errorf("LoadTools should not error on disabled tool, got: %v", err)
		}
		// Should have 0 tools since disabled tool was skipped
		tools := registry.ListTools()
		if len(tools) != 0 {
			t.Errorf("Expected 0 tools after loading disabled tool, got %d", len(tools))
		}
	})
}
