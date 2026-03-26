package main

import (
	"testing"
)

func TestHistoryDB_SearchAndPersistence(t *testing.T) {
	dir := t.TempDir()
	db := NewHistoryDB(dir)
	if !db.Load() {
		t.Log("Load returned false (expected for empty DB)")
	}
	if db.db == nil {
		t.Fatal("Database not opened after Load()")
	}
	defer db.Close()

	// Add messages
	db.AddMessage("user", "Please make a list of bugs to fix in the parser", nil)
	db.AddMessage("assistant", "Here is the bug list:\n1. Off-by-one error in tokenizer\n2. Missing null check in AST builder\n3. Incorrect precedence for ternary operator", nil)
	db.AddMessage("user", "Thanks, also check the compiler output", nil)
	db.AddMessage("assistant", "The compiler shows warnings about unused variables", nil)

	// Verify messages were stored
	count := db.MessageCount()
	if count != 4 {
		t.Fatalf("Expected 4 messages, got %d", count)
	}

	// Search should find the bug list
	results := db.Search("bug list parser", 10)
	if len(results) == 0 {
		// Debug: try simpler queries
		r1 := db.Search("bug", 10)
		r2 := db.Search("parser", 10)
		t.Fatalf("Expected to find results for 'bug list parser' (bug=%d, parser=%d)", len(r1), len(r2))
	}

	found := false
	for _, r := range results {
		if r.Message.Role == "assistant" && len(r.Message.Content) > 50 {
			found = true
		}
	}
	if !found {
		t.Error("Expected to find the assistant's bug list in results")
	}

	// Search for specific bug
	results2 := db.Search("tokenizer", 10)
	if len(results2) == 0 {
		t.Fatal("Expected to find results for 'tokenizer'")
	}

	// Close and reopen to verify persistence
	db.Close()
	db2 := NewHistoryDB(dir)
	db2.Load()
	defer db2.Close()

	if db2.MessageCount() != 4 {
		t.Errorf("After reopen, expected 4 messages, got %d", db2.MessageCount())
	}

	results3 := db2.Search("ternary operator", 10)
	if len(results3) == 0 {
		t.Fatal("After reopen, expected to find 'ternary operator'")
	}
}
