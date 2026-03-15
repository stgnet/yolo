package yolo

import (
	"testing"
)

func TestGitExecutor_Status(t *testing.T) {
	executor := NewGitExecutor()
	status, err := executor.Status(".")
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if status == "" {
		t.Error("Expected non-empty status output")
	}
}
