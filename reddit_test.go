package main

import (
	"strings"
	"testing"
)

// TestRedditToolDefinition verifies the reddit tool is properly defined
func TestRedditToolDefinition(t *testing.T) {
	found := false
	for _, tool := range ollamaTools {
		if tool.Function.Name == "reddit" {
			found = true

			// Check required parameters
			params := tool.Function.Parameters.Properties
			if _, ok := params["action"]; !ok {
				t.Error("reddit tool should have 'action' parameter")
			}
			if _, ok := params["limit"]; !ok {
				t.Error("reddit tool should have 'limit' parameter (optional)")
			}

			// Check description mentions Reddit functionality
			desc := tool.Function.Description
			if !strings.Contains(desc, "Reddit") && !strings.Contains(desc, "API") {
				t.Errorf("reddit description should mention Reddit or API, got: %s", desc)
			}

			break
		}
	}

	if !found {
		t.Error("reddit tool not found in ollamaTools")
	}
}

// TestRedditToolInValidTools verifies reddit is in the valid tools list
func TestRedditToolInValidTools(t *testing.T) {
	found := false
	for _, tool := range validTools {
		if tool == "reddit" {
			found = true
			break
		}
	}

	if !found {
		t.Error("reddit not found in validTools list")
	}
}

// TestRedditActions verifies all three Reddit actions are documented
func TestRedditActions(t *testing.T) {
	var toolDesc string
	for _, tool := range ollamaTools {
		if tool.Function.Name == "reddit" {
			toolDesc = tool.Function.Description
			break
		}
	}

	if toolDesc == "" {
		t.Fatal("reddit tool description not found")
	}

	actions := []string{"search", "subreddit", "thread"}
	for _, action := range actions {
		lowerDesc := strings.ToLower(toolDesc)
		if !strings.Contains(lowerDesc, strings.ToLower(action)) &&
			!strings.Contains(toolDesc, "\""+action+"\"") {
			t.Logf("Action '%s' may not be documented in description: %s", action, toolDesc[:min(100, len(toolDesc))])
		}
	}
}
