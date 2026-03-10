package main

import (
	"strings"
	"testing"
)

// TestGogToolDefinition verifies the gog tool is properly defined
func TestGogToolDefinition(t *testing.T) {
	found := false
	for _, tool := range ollamaTools {
		if tool.Function.Name == "gog" {
			found = true
			
			// Check required parameters
			params := tool.Function.Parameters.Properties
			if _, ok := params["command"]; !ok {
				t.Error("gog tool should have 'command' parameter")
			}
			
			// Check description mentions key capabilities
			desc := tool.Function.Description
			if !strings.Contains(desc, "Gmail") && !strings.Contains(desc, "Calendar") {
				t.Errorf("gog description should mention Gmail or Calendar, got: %s", desc)
			}
			
			break
		}
	}
	
	if !found {
		t.Error("gog tool not found in ollamaTools")
	}
}

// TestGogToolInValidTools verifies gog is in the valid tools list
func TestGogToolInValidTools(t *testing.T) {
	found := false
	for _, tool := range validTools {
		if tool == "gog" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("gog not found in validTools list")
	}
}
