package yolo

import (
	"testing"
)

func TestGitListBranches(t *testing.T) {
	executor := NewToolExecutor("", nil)
	result, err := executor.GitListBranches(nil)
	if err != nil {
		t.Fatalf("GitListBranches failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty branch list")
	}
}

func TestGitDiff(t *testing.T) {
	executor := NewToolExecutor("", nil)
	result, err := executor.GitDiff(nil)
	if err != nil {
		t.Fatalf("GitDiff failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty diff output")
	}
}

func TestGitStatus(t *testing.T) {
	executor := NewToolExecutor("", nil)
	result, err := executor.GitStatus(nil)
	if err != nil {
		t.Fatalf("GitStatus failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty status output")
	}
}
