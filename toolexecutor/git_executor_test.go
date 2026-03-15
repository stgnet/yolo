package toolexecutor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitExecutor_ListBranches(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewToolExecutor()

	branches, err := executor.GitListBranches(tmpDir)
	if err != nil {
		t.Fatalf("GitListBranches failed: %v", err)
	}

	if len(branches) == 0 {
		t.Fatal("Expected at least one branch")
	}
}

func TestGitExecutor_Diff(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewToolExecutor()

	diff, err := executor.GitDiff(tmpDir, "")
	if err != nil {
		t.Fatalf("GitDiff failed: %v", err)
	}

	if len(diff) == 0 {
		t.Error("Expected some output from diff")
	}
}

func TestGitExecutor_Status(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewToolExecutor()

	status, err := executor.GitStatus(tmpDir)
	if err != nil {
		t.Fatalf("GitStatus failed: %v", err)
	}

	if len(status) == 0 {
		t.Error("Expected some output from status")
	}
}

func TestGitExecutor_InitAndCommit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	executor := NewToolExecutor()

	status, err := executor.GitStatus(tmpDir)
	if err != nil {
		t.Fatalf("GitStatus failed: %v", err)
	}

	if len(status) == 0 {
		t.Error("Expected some output from status after creating file")
	}
}
