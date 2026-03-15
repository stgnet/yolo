package main

import "testing"

func TestGitExecutorBranches(t *testing.T) {
	executor := NewToolExecutor("", nil)
	result, err := executor.gitListBranches(nil)
	if err != nil {
		t.Fatalf("gitListBranches failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty branch list")
	}
}

func TestGitExecutorDiff(t *testing.T) {
	executor := NewToolExecutor("", nil)
	result, err := executor.gitDiff(nil)
	if err != nil {
		t.Fatalf("gitDiff failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty diff output")
	}
}

func TestGitExecutorStatus(t *testing.T) {
	executor := NewToolExecutor("", nil)
	result, err := executor.gitStatus(nil)
	if err != nil {
		t.Fatalf("gitStatus failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty status output")
	}
}
