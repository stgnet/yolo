package main

import "testing"

func TestGitListBranches(t *testing.T) {
	te := NewToolExecutor()
	output, err := te.GitListBranches()
	if err != nil {
		t.Fatalf("git list-branches failed: %v", err)
	}
	expectedPrefix := "  master\n"
	if output != expectedPrefix {
		t.Errorf("unexpected output from git list-branches:\nexpected: %q\ngot:      %q", expectedPrefix, output)
	}
}
