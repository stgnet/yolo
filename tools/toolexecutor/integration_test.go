package yolo

import "testing"

func TestGitExecutor(t *testing.T) {
	executor := NewToolExecutor("", nil)
	if executor == nil {
		t.Fatal("Expected non-nil executor")
	}

	branchResult, ok := executor.gitListBranches(map[string]any{})
	if !ok {
		t.Fatal("gitListBranches should not return false for default case")
	}
	if _, ok := branchResult.(map[string]any); !ok {
		t.Errorf("Expected map result from gitListBranches, got %T", branchResult)
	}

	diffResult, ok := executor.gitDiff(map[string]any{})
	if !ok {
		t.Fatal("gitDiff should not return false for default case")
	}
	if _, ok := diffResult.(map[string]any); !ok {
		t.Errorf("Expected map result from gitDiff, got %T", diffResult)
	}

	statusResult, ok := executor.gitStatus(map[string]any{})
	if !ok {
		t.Fatal("gitStatus should not return false for default case")
	}
	if _, ok := statusResult.(map[string]any); !ok {
		t.Errorf("Expected map result from gitStatus, got %T", statusResult)
	}

	commandResult, ok := executor.gitCommand(map[string]any{"command": "init"})
	if !ok {
		t.Fatal("gitCommand should not return false for init case")
	}
	if _, ok := commandResult.(map[string]any); !ok {
		t.Errorf("Expected map result from gitCommand, got %T", commandResult)
	}

	commandResult2, ok := executor.gitCommand(map[string]any{"command": "clone"})
	if !ok {
		t.Fatal("gitCommand should not return false for clone case")
	}
	if _, ok := commandResult2.(map[string]any); !ok {
		t.Errorf("Expected map result from gitCommand, got %T", commandResult2)
	}

	commandResult3, ok := executor.gitCommand(map[string]any{"command": "status"})
	if !ok {
		t.Fatal("gitCommand should not return false for status case")
	}
	if _, ok := commandResult3.(map[string]any); !ok {
		t.Errorf("Expected map result from gitCommand, got %T", commandResult3)
	}

	_, ok = executor.gitCommand(map[string]any{"command": "unknown"})
	if ok {
		t.Error("gitCommand should return false for unknown command")
	}
}
