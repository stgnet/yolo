package main

import (
	"os"
	"testing"
)

// TestGitStatus tests the git_status tool
func TestGitStatus(t *testing.T) {
	if !isGitRepo() {
		t.Skip("Not in a git repository")
	}

	executor := NewToolExecutor(os.Getenv("PWD"), nil)
	result := executor.gitStatus(nil)

	if result == "" {
		t.Error("git_status returned empty result")
	}

	if !containsString(result, "Branch:") {
		t.Errorf("Expected 'Branch:' in output, got: %s", result)
	}

	t.Logf("Git status output:\n%s", result)
}

// TestGitDiff tests the git_diff tool
func TestGitDiff(t *testing.T) {
	if !isGitRepo() {
		t.Skip("Not in a git repository")
	}

	executor := NewToolExecutor(os.Getenv("PWD"), nil)
	result := executor.gitDiff(nil)

	// Should return either "No changes" or actual diff content
	if result != "No changes (working tree clean)" && result == "" {
		t.Error("git_diff returned empty string when it should indicate no changes")
	}

	t.Logf("Git diff output:\n%s", result)
}

// TestGitLog tests the git_log tool
func TestGitLog(t *testing.T) {
	if !isGitRepo() {
		t.Skip("Not in a git repository")
	}

	executor := NewToolExecutor(os.Getenv("PWD"), nil)
	result := executor.gitLog(map[string]any{"limit": 5})

	if result == "" {
		t.Error("git_log returned empty result")
	}

	if !containsString(result, "Recent commits") {
		t.Errorf("Expected 'Recent commits' in output, got: %s", result)
	}

	t.Logf("Git log output:\n%s", result)
}

// TestGitBranch tests the git_branch tool
func TestGitBranch(t *testing.T) {
	if !isGitRepo() {
		t.Skip("Not in a git repository")
	}

	executor := NewToolExecutor(os.Getenv("PWD"), nil)
	result := executor.gitBranch(nil)

	if result == "" {
		t.Error("git_branch returned empty result")
	}

	if !containsString(result, "Branches:") {
		t.Errorf("Expected 'Branches:' in output, got: %s", result)
	}

	t.Logf("Git branch output:\n%s", result)
}

// TestGitAdd tests the git_add tool
func TestGitAdd(t *testing.T) {
	if !isGitRepo() {
		t.Skip("Not in a git repository")
	}

	executor := NewToolExecutor(os.Getenv("PWD"), nil)
	
	// Create a test file
	testFile := ".git_add_test"
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	result := executor.gitAdd(map[string]any{"file": testFile})

	if !containsString(result, "Staged") && !containsString(result, "added to index") {
		t.Errorf("Expected success message, got: %s", result)
	}

	t.Logf("Git add output:\n%s", result)
}

// TestGitRemote tests the git_remote tool
func TestGitRemote(t *testing.T) {
	if !isGitRepo() {
		t.Skip("Not in a git repository")
	}

	executor := NewToolExecutor(os.Getenv("PWD"), nil)
	result := executor.gitRemote(nil)

	// May be empty if no remotes configured, but shouldn't error
	if result != "" && !containsString(result, "Error:") {
		t.Logf("Git remote output:\n%s", result)
	}
}

// TestGitShow tests the git_show tool
func TestGitShow(t *testing.T) {
	if !isGitRepo() {
		t.Skip("Not in a git repository")
	}

	executor := NewToolExecutor(os.Getenv("PWD"), nil)
	result := executor.gitShow(nil)

	if result == "" {
		t.Error("git_show returned empty result")
	}

	t.Logf("Git show output:\n%s", result)
}

// isGitRepo checks if current directory is a git repository
func isGitRepo() bool {
	_, err := os.Stat(".git")
	return err == nil
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && contains(s, substr))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
