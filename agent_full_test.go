package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAgentGetSystemPrompt tests loading the system prompt template
func TestAgentGetSystemPrompt(t *testing.T) {
	t.Parallel()

	// Create a temp directory with SYSTEM_PROMPT.md
	tmpDir, err := os.MkdirTemp("", "yolo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	systemPromptPath := filepath.Join(tmpDir, "SYSTEM_PROMPT.md")
	promptContent := `# YOLO Test
Working directory: {baseDir}
Model: {model}
Timestamp: {timestamp}`

	if err := os.WriteFile(systemPromptPath, []byte(promptContent), 0644); err != nil {
		t.Fatal(err)
	}

	agent := &YoloAgent{
		baseDir:    tmpDir,
		scriptPath: "/tmp/test",
		history:    NewHistoryManager(".yolo"),
	}

	prompt := agent.getSystemPrompt()

	if !strings.Contains(prompt, "Working directory: "+tmpDir) {
		t.Errorf("Expected prompt to contain working directory, got: %s", prompt)
	}

	if !strings.Contains(prompt, "YOLO Test") {
		t.Errorf("Expected prompt to contain template content, got: %s", prompt)
	}
}

// TestAgentSetupFirstRun tests first-run setup when no history exists
func TestAgentSetupFirstRun(t *testing.T) {
	t.Skip("Skipping interactive test that requires user input")

	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "yolo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	agent := &YoloAgent{
		baseDir: tmpDir,
		history: NewHistoryManager(".yolo"),
		ollama:  NewOllamaClient("http://localhost:11434"),
	}

	// Setup should not panic even with no models available
	// We expect it to exit when Ollama is unreachable, but that's fine for this test
	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Recovered from panic: %v", r)
			}
			close(done)
		}()
		agent.setupFirstRun()
	}()

	select {
	case <-done:
		// Test passed if we reach here without crashing
	case <-time.After(2 * time.Second):
		t.Fatal("setupFirstRun took too long")
	}
}

// TestAgentChatWithAgent tests the main chat loop
func TestAgentChatWithAgent(t *testing.T) {
	t.Skip("Skipping integration test that requires Ollama connection")

	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "yolo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	systemPromptPath := filepath.Join(tmpDir, "SYSTEM_PROMPT.md")
	if err := os.WriteFile(systemPromptPath, []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	agent := &YoloAgent{
		baseDir: tmpDir,
		history: NewHistoryManager(".yolo"),
	}

	// ChatWithAgent requires Ollama connection, so we test that it doesn't panic
	// We expect it to fail gracefully when no models are available
	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Recovered from panic: %v", r)
			}
			close(done)
		}()
		agent.chatWithAgent("test message", false)
	}()

	select {
	case <-done:
		// Test passed if we reach here without crashing
	case <-time.After(2 * time.Second):
		t.Fatal("chatWithAgent took too long")
	}
}

// TestAgentSwitchModel tests model switching
func TestAgentSwitchModel(t *testing.T) {
	t.Skip("Skipping integration test that requires Ollama connection")

	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "yolo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	systemPromptPath := filepath.Join(tmpDir, "SYSTEM_PROMPT.md")
	if err := os.WriteFile(systemPromptPath, []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	agent := &YoloAgent{
		baseDir: tmpDir,
		history: NewHistoryManager(".yolo"),
		ollama:  NewOllamaClient("http://localhost:11434"),
	}

	// Try to switch to a non-existent model
	result := agent.switchModel("nonexistent-model")
	if !strings.Contains(result, "not found") {
		t.Errorf("Expected error message about model not found, got: %s", result)
	}
}

// TestAgentSpawnSubagent tests subagent spawning
func TestAgentSpawnSubagent(t *testing.T) {
	t.Skip("Skipping integration test that requires Ollama connection")

	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "yolo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	systemPromptPath := filepath.Join(tmpDir, "SYSTEM_PROMPT.md")
	if err := os.WriteFile(systemPromptPath, []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	os.MkdirAll(filepath.Join(tmpDir, ".yolo", "subagents"), 0755)

	agent := &YoloAgent{
		baseDir: tmpDir,
		history: NewHistoryManager(".yolo"),
		ollama:  NewOllamaClient("http://localhost:11434"),
	}

	result := agent.spawnSubagent("test task", "test-model")

	if !strings.Contains(result, "spawned") {
		t.Errorf("Expected 'spawned' in result, got: %s", result)
	}

	// Give subagent time to write result file
	time.Sleep(1 * time.Second)
}

// TestAgentHandleCommand tests slash command handling
func TestAgentHandleCommand(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "yolo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	systemPromptPath := filepath.Join(tmpDir, "SYSTEM_PROMPT.md")
	if err := os.WriteFile(systemPromptPath, []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	agent := &YoloAgent{
		baseDir:  tmpDir,
		history:  NewHistoryManager(".yolo"),
		inputMgr: &InputManager{}, // Minimal input manager to prevent nil pointer
	}

	// Test /help - should not panic
	agent.handleCommand("/help")

	// Test /model - should not panic
	agent.handleCommand("/model")

	// Test /history - should not panic
	agent.handleCommand("/history")

	// Test /status - should not panic
	agent.handleCommand("/status")
}

// TestAgentRun tests the main Run method (integration test)
func TestAgentRun(t *testing.T) {
	t.Parallel()
	t.Skip("Skipping integration test that requires terminal")

	tmpDir, err := os.MkdirTemp("", "yolo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	systemPromptPath := filepath.Join(tmpDir, "SYSTEM_PROMPT.md")
	if err := os.WriteFile(systemPromptPath, []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	agent := &YoloAgent{
		baseDir: tmpDir,
		history: NewHistoryManager(".yolo"),
	}

	// Run test would require terminal, skip for now
	_ = agent
}
