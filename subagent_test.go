package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestListSubagents tests the list_subagents tool
func TestListSubagents(t *testing.T) {
	t.Parallel()

	// Use the actual SubagentDir since it's a global constant
	subagentDir := filepath.Join(YoloDir, "subagents")
	if err := os.MkdirAll(subagentDir, 0755); err != nil {
		t.Fatal(err)
	}

	executor := NewToolExecutor(".", nil)

	// Clean up any existing test files first
	files, _ := filepath.Glob(filepath.Join(subagentDir, "agent_test_*.json"))
	for _, f := range files {
		os.Remove(f)
	}

	// Test with no subagents (should show either message or empty list)
	result := executor.listSubagents(nil)
	t.Logf("Empty result: %s", result)
	if result == "" {
		t.Error("Expected non-empty result from listSubagents")
	}

	// Create a test subagent file with unique name
	subagentData := map[string]any{
		"id":     "test_12345",
		"task":   "unique test task for list_subagents verification",
		"model":  "test-model",
		"status": "complete",
		"result": "test result",
		"ts":     "2024-01-01T00:00:00Z",
	}
	data, _ := json.MarshalIndent(subagentData, "", "  ")
	testFile := filepath.Join(subagentDir, "agent_test_12345.json")
	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile) // Clean up

	// Test with one subagent - note: task is truncated to 40 chars in output
	result = executor.listSubagents(nil)
	if !strings.Contains(result, "unique test task") {
		t.Errorf("Expected to find 'unique test task' in result, got: %s", result)
	}
}

// TestReadSubagentResult tests the read_subagent_result tool
func TestReadSubagentResult(t *testing.T) {
	t.Parallel()

	executor := NewToolExecutor(".", nil)

	// Test with missing ID argument
	result := executor.readSubagentResult(nil)
	if !strings.Contains(result, "Error") {
		t.Errorf("Expected error about missing id, got: %s", result)
	}

	// Test with non-existent subagent (use unique ID to avoid conflicts)
	result = executor.readSubagentResult(map[string]any{"id": 77777})
	if !strings.Contains(result, "Error") {
		t.Errorf("Expected error about subagent not found, got: %s", result)
	}

	// Create test subagent directory
	subagentDir := filepath.Join(YoloDir, "subagents")
	os.MkdirAll(subagentDir, 0755)

	// Create a test subagent file with unique ID
	subagentData := map[string]any{
		"id":     77777,
		"task":   "unique task for read_subagent_result",
		"model":  "test-model",
		"status": "complete",
		"result": "unique test result output for verification",
		"ts":     "2024-01-01T00:00:00Z",
	}
	data, _ := json.MarshalIndent(subagentData, "", "  ")
	testFile := filepath.Join(subagentDir, "agent_77777.json")
	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile) // Clean up

	// Test with existing subagent (use the unique ID)
	result = executor.readSubagentResult(map[string]any{"id": 77777})
	if !strings.Contains(result, "unique test result output for verification") {
		t.Errorf("Expected to find unique test result in output, got: %s", result)
	}
	if !strings.Contains(result, "Sub-agent #77777 Result:") {
		t.Errorf("Expected to find ID 77777 in output, got: %s", result)
	}

	// Test with in-progress subagent
	inProgressData := map[string]any{
		"id":     88888,
		"task":   "still running task",
		"model":  "test-model",
		"status": "in-progress",
		"result": "",
		"ts":     "2024-01-01T00:00:00Z",
	}
	ipData, _ := json.MarshalIndent(inProgressData, "", "  ")
	ipFile := filepath.Join(subagentDir, "agent_88888.json")
	if err := os.WriteFile(ipFile, ipData, 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(ipFile)

	result = executor.readSubagentResult(map[string]any{"id": 88888})
	if !strings.Contains(result, "in-progress") {
		t.Errorf("Expected 'in-progress' status in output, got: %s", result)
	}
	if !strings.Contains(result, "still running, check back later") {
		t.Errorf("Expected 'still running' message for in-progress subagent, got: %s", result)
	}
}

// TestSummarizeSubagents tests the summarize_subagents tool
func TestSummarizeSubagents(t *testing.T) {
	t.Parallel()

	executor := NewToolExecutor(".", nil)

	// Create test subagent directory
	subagentDir := filepath.Join(YoloDir, "subagents")
	os.MkdirAll(subagentDir, 0755)

	// Clean up any existing test files first
	files, _ := filepath.Glob(filepath.Join(subagentDir, "agent_test_sum_*.json"))
	for _, f := range files {
		os.Remove(f)
	}

	// Test summary - verify the function runs and produces output with expected format
	result := executor.summarizeSubagents(nil)
	t.Logf("Summary result: %s", result)

	// Verify the function runs and produces output with expected format
	if !strings.Contains(result, "Summary") {
		t.Errorf("Expected 'Summary' in output, got: %s", result)
	}
	if !strings.Contains(result, "Completed:") || !strings.Contains(result, "Errors:") {
		t.Errorf("Expected 'Completed:' and 'Errors:' in summary, got: %s", result)
	}

	// Create test subagent files with different statuses and unique IDs
	subagentData1 := map[string]any{
		"id":     "test_sum_001",
		"status": "complete",
	}
	data1, _ := json.Marshal(subagentData1)
	file1 := filepath.Join(subagentDir, "agent_test_sum_001.json")
	os.WriteFile(file1, data1, 0644)
	defer os.Remove(file1)

	subagentData2 := map[string]any{
		"id":     "test_sum_002",
		"status": "error",
	}
	data2, _ := json.Marshal(subagentData2)
	file2 := filepath.Join(subagentDir, "agent_test_sum_002.json")
	os.WriteFile(file2, data2, 0644)
	defer os.Remove(file2)

	subagentData3 := map[string]any{
		"id":     "test_sum_003",
		"status": "complete",
	}
	data3, _ := json.Marshal(subagentData3)
	file3 := filepath.Join(subagentDir, "agent_test_sum_003.json")
	os.WriteFile(file3, data3, 0644)
	defer os.Remove(file3)

	// Re-check after adding files - verify function doesn't crash
	result = executor.summarizeSubagents(nil)
	if result == "" {
		t.Error("Expected non-empty summary result")
	}
}
