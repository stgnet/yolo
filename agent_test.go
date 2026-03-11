package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewYoloAgent verifies agent initialization
func TestNewYoloAgent(t *testing.T) {
	a := NewYoloAgent()
	if a == nil {
		t.Fatal("Expected non-nil agent")
	}

	baseDir, _ := os.Getwd()
	if a.baseDir != baseDir {
		t.Errorf("baseDir = %q, want %q", a.baseDir, baseDir)
	}

	if a.scriptPath == "" {
		t.Error("scriptPath should not be empty")
	}

	if a.ollama == nil {
		t.Error("ollama client should not be nil")
	}

	if a.history == nil {
		t.Error("history manager should not be nil")
	}

	if a.tools == nil {
		t.Error("tools executor should not be nil")
	}

	// Note: inputMgr is set in Run(), not constructor, so it's nil here
	if a.inputMgr != nil {
		t.Error("inputMgr should be nil initially (set in Run)")
	}
}

// TestShowHelpHint verifies the help hint is displayed
func TestShowHelpHint(t *testing.T) {
	a := NewYoloAgent()

	// Just verify it doesn't panic
	a.showHelpHint()
}

// TestDrainQueuedInput tests draining of queued input
func TestDrainQueuedInput(t *testing.T) {
	a := NewYoloAgent()

	// inputMgr is nil until Run() is called, so drainQueuedInput should not panic
	a.drainQueuedInput()
}

// TestSetupFirstRun verifies first run setup
func TestSetupFirstRun(t *testing.T) {
	t.Skip("setupFirstRun requires interactive input and network call to Ollama, skipping in automated tests")

	a := NewYoloAgent()

	// Create a temp dir for testing to avoid modifying actual repo
	tmpDir, err := os.MkdirTemp("", "yolo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set baseDir to temp dir
	a.baseDir = tmpDir

	// Setup should not panic
	a.setupFirstRun()

	// Check that .yolo directory was created
	yoloDir := filepath.Join(tmpDir, ".yolo")
	if _, err := os.Stat(yoloDir); os.IsNotExist(err) {
		t.Error(".yolo directory should be created")
	}
}

// TestDisplaySessionResumption tests session resumption display
func TestDisplaySessionResumption(t *testing.T) {
	a := NewYoloAgent()

	// Just verify it doesn't panic (will show minimal or no history in fresh agent)
	a.displaySessionResumption()
}

// TestShowCacheStatus tests cache status display
func TestShowCacheStatus(t *testing.T) {
	a := NewYoloAgent()

	// Just verify it doesn't panic with a test value
	a.showCacheStatus("test")
}

// TestShowPrompt tests prompt display - skip because requires inputMgr initialization
func TestShowPrompt(t *testing.T) {
	t.Skip("showPrompt requires inputMgr to be initialized (done in Run), skipping in automated tests")
	a := NewYoloAgent()

	// Just verify it doesn't panic
	a.showPrompt()
}

// TestSpawnSubagentErrorPath tests subagent spawning with invalid path
func TestSpawnSubagentErrorPath(t *testing.T) {
	t.Skip("spawnSubagent requires Ollama connection, skipping in automated tests")

	a := NewYoloAgent()

	// Override baseDir with invalid location to trigger error path
	a.baseDir = "/nonexistent/path/that/does/not/exist"

	result := a.spawnSubagent("test prompt", "test name")

	if !strings.Contains(result, "error") {
		t.Error("Expected result to contain 'error' when baseDir is invalid")
	}
}

// TestSpawnSubagentNoResultFile tests subagent spawning without result file
func TestSpawnSubagentNoResultFile(t *testing.T) {
	t.Skip("spawnSubagent requires Ollama connection, skipping in automated tests")

	a := NewYoloAgent()

	result := a.spawnSubagent("test prompt", "test name")

	// Result should mention the results will be sent to file
	if !strings.Contains(result, "result") {
		t.Error("Expected result to mention where results will be sent")
	}
}

// TestHandoffRemainingToolsEmpty tests handoff with empty tool list
func TestHandoffRemainingToolsEmpty(t *testing.T) {
	a := NewYoloAgent()

	hr := a.handoffRemainingTools([]ParsedToolCall{})

	if hr == nil {
		t.Fatal("Expected non-nil handoffResult")
	}

	if hr.ID <= 0 {
		t.Errorf("Expected positive ID, got %d", hr.ID)
	}

	if hr.Results != nil && len(hr.Results) > 0 {
		t.Error("Expected empty results for empty input")
	}

	// Don't wait on Done channel to avoid timeout - just verify structure is correct
	a.mu.Lock()
	pendingCount := len(a.pendingHandoffs)
	a.mu.Unlock()

	if pendingCount != 1 {
		t.Errorf("Expected 1 pending handoff, got %d", pendingCount)
	}
}

// TestIngestHandoffResultsEmpty tests ingestion with no pending handoffs
func TestIngestHandoffResultsEmpty(t *testing.T) {
	a := NewYoloAgent()

	count := a.ingestHandoffResults()

	if count != 0 {
		t.Errorf("Expected 0 ingested results, got %d", count)
	}
}

// TestIngestHandoffResults tests ingestion of completed handoffs
func TestIngestHandoffResults(t *testing.T) {
	a := NewYoloAgent()

	// Create a fake completed handoff
	hr := &handoffResult{
		ID:   123,
		Done: make(chan struct{}),
		Results: []toolExecResult{
			{Name: "test_tool", Args: map[string]any{"key": "value"}, Result: "test result"},
		},
	}
	close(hr.Done) // Mark as complete

	a.mu.Lock()
	a.pendingHandoffs = append(a.pendingHandoffs, hr)
	a.mu.Unlock()

	// Ingest should pick it up
	count := a.ingestHandoffResults()

	if count != 1 {
		t.Errorf("Expected 1 ingested result, got %d", count)
	}

	// Check that message was added to history
	messages := a.history.GetContextMessages(10)
	found := false
	for _, msg := range messages {
		if strings.Contains(msg.Content, "handoff #123") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected handoff result to be added to history")
	}
}

// TestGetSystemPrompt verifies system prompt generation
func TestGetSystemPrompt(t *testing.T) {
	a := NewYoloAgent()

	prompt := a.getSystemPrompt()

	if len(prompt) == 0 {
		t.Error("System prompt should not be empty")
	}

	if !strings.Contains(prompt, "YOLO") {
		t.Error("System prompt should mention YOLO")
	}
}

// TestRestartAgent tests restart setup
func TestRestartAgent(t *testing.T) {
	a := NewYoloAgent()

	// Verify agent is configured correctly (actual restart would exit)
	if a.scriptPath == "" {
		t.Error("scriptPath should be set")
	}
}

// TestDisplaySessionResumptionWithHistory tests with actual history
func TestDisplaySessionResumptionWithHistory(t *testing.T) {
	a := NewYoloAgent()

	// Add some test messages
	a.history.AddMessage("user", "Test user message", nil)
	a.history.AddMessage("assistant", "Test assistant response", nil)

	// Should not panic and should display something
	a.displaySessionResumption()
}

func TestAgentStateReset(t *testing.T) {
	a := NewYoloAgent()

	// Verify initial state - inputMgr is set in Run(), so it will be nil here
	if a.inputMgr != nil {
		t.Error("inputMgr should be nil initially (set in Run)")
	}

	// The rest are initialized in constructor
	if a.ollama == nil {
		t.Error("ollama not initialized")
	}
	if a.history == nil {
		t.Error("history not initialized")
	}
	if a.tools == nil {
		t.Error("tools not initialized")
	}
}

func TestAgentRunPreparation(t *testing.T) {
	a := NewYoloAgent()

	// Verify all components are ready (inputMgr is set in Run, so check without it)
	if a.ollama == nil {
		t.Error("ollama not initialized")
	}
	if a.history == nil {
		t.Error("history not initialized")
	}
	if a.tools == nil {
		t.Error("tools not initialized")
	}
}
