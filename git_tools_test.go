package main

import (
	"os"
	"strings"
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

	if !strings.Contains(result, "Branch:") {
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

	if !strings.Contains(result, "Recent commits") {
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

	if !strings.Contains(result, "Branches:") {
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

	// Note: We cannot actually test git add here because it operates on the real repo
	// and would stage files. Instead, we verify the tool exists and handles invalid args.
	result := executor.gitAdd(nil)

	// Should return some output or handle gracefully without making changes
	if result == "" {
		t.Log("git_add returned empty (expected when no file specified)")
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
	if result != "" && !strings.Contains(result, "Error:") {
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

// TestGitCheckoutBranch tests the git_checkout tool with a branch
func TestGitCheckoutBranch(t *testing.T) {
	t.Skip("Skipped to avoid changing git repository state - checkout operations modify working directory")
}

// TestGitCheckoutFile tests the git_checkout tool with a file restore
func TestGitCheckoutFile(t *testing.T) {
	t.Skip("Skipped to avoid changing git repository state - checkout operations modify working directory")
}

// TestGitCommit tests the git_commit tool
func TestGitCommit(t *testing.T) {
	t.Skip("Skipped to avoid creating commits in git repository - commit operations modify repository history")
}

// isGitRepo checks if current directory is a git repository
func isGitRepo() bool {
	_, err := os.Stat(".git")
	return err == nil
}
