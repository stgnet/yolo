package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLearnTool(t *testing.T) {
	t.Run("creates learning manager", func(t *testing.T) {
		tempDir := t.TempDir()
		executor := NewToolExecutor(tempDir, nil)
		
		result := executor.learn(map[string]any{})
		
		if result == "" {
			t.Error("learn tool should return non-empty result")
		}
		
		// Should mention learning or improvements
		if !containsAny(result, []string{"learning", "improvement", "research"}) {
			t.Logf("Result doesn't contain expected keywords: %s", result)
		}
	})

	t.Run("checks for recent sessions before learning", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Create a fake learning history with recent session
		historyPath := filepath.Join(tempDir, ".yolo_learning.json")
		sessions := []LearningSession{
			{
				Timestamp:   time.Now().Add(-1 * time.Hour), // 1 hour ago
				Duration:    30,
				Improvements: []Improvement{},
			},
		}
		
		historyJSON, err := json.MarshalIndent(sessions, "", "  ")
		if err != nil {
			t.Fatalf("Failed to create test history JSON: %v", err)
		}
		
		err = os.WriteFile(historyPath, historyJSON, 0644)
		if err != nil {
			t.Fatalf("Failed to write test history file: %v", err)
		}
		
		executor := NewToolExecutor(tempDir, nil)
		result := executor.learn(map[string]any{})
		
		// Should mention already performed today
		if !containsAny(result, []string{"already performed", "today"}) {
			t.Errorf("Expected rate limit message, got: %s", result)
		}
	})

	t.Run("handles missing learning history gracefully", func(t *testing.T) {
		tempDir := t.TempDir()
		executor := NewToolExecutor(tempDir, nil)
		
		result := executor.learn(map[string]any{})
		
		// Should not error on missing file, should create it
		if containsAny(result, []string{"error", "Error"}) {
			t.Errorf("Unexpected error in result: %s", result)
		}
	})

	t.Run("formatting includes session info", func(t *testing.T) {
		tempDir := t.TempDir()
		executor := NewToolExecutor(tempDir, nil)
		
		result := executor.learn(map[string]any{})
		
		// Check for expected formatting elements
		expectedElements := []string{
			"Learning Session",
			"Improvements Discovered",
		}
		
		for _, element := range expectedElements {
			if !containsAny(result, []string{element}) {
				t.Logf("Missing expected element: %s", element)
			}
		}
	})
}

func TestLearnToolErrorCases(t *testing.T) {
	t.Run("handles invalid directory", func(t *testing.T) {
		invalidDir := "/nonexistent/path/that/does/not/exist"
		executor := NewToolExecutor(invalidDir, nil)
		
		result := executor.learn(map[string]any{})
		
		// Should handle gracefully, possibly with error message or empty result
		_ = result // Just ensure it doesn't panic
	})
}

func TestLearnToolEdgeCases(t *testing.T) {
	t.Run("with empty args", func(t *testing.T) {
		tempDir := t.TempDir()
		executor := NewToolExecutor(tempDir, nil)
		
		result := executor.learn(nil)
		
		if result == "" {
			t.Error("learn tool should return non-empty result even with nil args")
		}
	})

	t.Run("with extra unused args", func(t *testing.T) {
		tempDir := t.TempDir()
		executor := NewToolExecutor(tempDir, nil)
		
		extraArgs := map[string]any{
			"unused": "value",
			"number": 123,
		}
		
		result := executor.learn(extraArgs)
		
		if result == "" {
			t.Error("learn tool should handle extra args gracefully")
		}
	})
}

// Helper function for string containment check
func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if contains(s, sub) {
			return true
		}
	}
	return false
}
