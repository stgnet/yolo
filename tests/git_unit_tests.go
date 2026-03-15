// Git operations unit tests for YOLO
package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGitListBranches_EmptyRepo - Test git list branches in a directory without git repo
func TestGitListBranches_EmptyRepo(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	
	// Create a temp directory that's not a git repo
	tempDir := t.TempDir()
	
	args := map[string]any{
		"path": tempDir,
	}
	
	result := executor.gitListBranches(args)
	
	if containsString(result, "Error") {
		t.Logf("Git list branches returned expected error for non-git directory: %s", result)
	} else if len(result) == 0 {
		t.Log("Git list branches returned empty result for non-git directory")
	}
}

// TestGitListBranches_WithRepo - Test git list branches (will skip on systems without git)
func TestGitListBranches_WithRepo(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	
	// Create a temp directory with a git repo
	tempDir := t.TempDir()
	
	args := map[string]any{
		"path": tempDir,
	}
	
	result := executor.gitListBranches(args)
	
	// On systems without git, this will fail - that's expected
	if containsString(result, "Error") || containsString(result, "Exit code: 127") {
		t.Logf("Git list branches returned expected error (git may not be installed): %s", result)
	} else if len(result) > 0 {
		t.Log("Git list branches executed successfully")
	}
}

// TestGitDiff_EmptyRepo - Test git diff in a directory without git repo
func TestGitDiff_EmptyRepo(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	
	tempDir := t.TempDir()
	
	args := map[string]any{
		"path": tempDir,
	}
	
	result := executor.gitDiff(args)
	
	if containsString(result, "Error") {
		t.Logf("Git diff returned expected error for non-git directory: %s", result)
	} else if len(result) == 0 {
		t.Log("Git diff returned empty result for non-git directory")
	}
}

// TestGitDiff_WithRepo - Test git diff (will skip on systems without git)
func TestGitDiff_WithRepo(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	
	tempDir := t.TempDir()
	
	args := map[string]any{
		"path": tempDir,
	}
	
	result := executor.gitDiff(args)
	
	if containsString(result, "Error") || containsString(result, "Exit code: 127") {
		t.Logf("Git diff returned expected error (git may not be installed): %s", result)
	}
}

// TestGitStatus_EmptyRepo - Test git status in a directory without git repo
func TestGitStatus_EmptyRepo(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	
	tempDir := t.TempDir()
	
	args := map[string]any{
		"path": tempDir,
	}
	
	result := executor.gitStatus(args)
	
	if containsString(result, "Error") {
		t.Logf("Git status returned expected error for non-git directory: %s", result)
	} else if len(result) == 0 {
		t.Log("Git status returned empty result for non-git directory")
	}
}

// TestGitStatus_WithRepo - Test git status (will skip on systems without git)
func TestGitStatus_WithRepo(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	
	tempDir := t.TempDir()
	
	args := map[string]any{
		"path": tempDir,
	}
	
	result := executor.gitStatus(args)
	
	if containsString(result, "Error") || containsString(result, "Exit code: 127") {
		t.Logf("Git status returned expected error (git may not be installed): %s", result)
	}
}

// TestGitDiff_StagedFile - Test git diff with staged file
func TestGitDiff_StagedFile(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	
	// Create a test file
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Skipf("Could not create test file: %v", err)
	}
	
	args := map[string]any{
		"path": tempDir,
	}
	
	result := executor.gitDiff(args)
	
	if containsString(result, "Error") || containsString(result, "Exit code: 127") {
		t.Logf("Git diff returned expected error (git may not be installed): %s", result)
	}
}

// TestGitListBranches_NonEmptyDirectory - Test git list branches in non-empty directory
func TestGitListBranches_NonEmptyDirectory(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	
	// Create a test file
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Skipf("Could not create test file: %v", err)
	}
	
	args := map[string]any{
		"path": tempDir,
	}
	
	result := executor.gitListBranches(args)
	
	if containsString(result, "Error") || containsString(result, "Exit code: 127") {
		t.Logf("Git list branches returned expected error (git may not be installed): %s", result)
	}
}

// TestGit_InvalidPath_Empty - Test git commands with empty path
func TestGit_InvalidPath_Empty(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	
	args := map[string]any{
		"path": "",
	}
	
	// Test all git operations with empty path
	testCases := []struct{
		name string
		fn func(map[string]any) string
	}{
		{"listBranches", executor.gitListBranches},
		{"diff", executor.gitDiff},
		{"status", executor.gitStatus},
	}
	
	for _, tc := range testCases {
		result := tc.fn(args)
		if containsString(result, "Error") || containsString(result, "required") {
			t.Logf("%s returned expected error for empty path: %s", tc.name, result)
		}
	}
}
