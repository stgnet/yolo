// Tests for processInboxWithResponse improvement - checking both new/ and cur/ directories
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessInboxWithResponseReadsBothDirectories(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	newDir := filepath.Join(tmpDir, "new")
	curDir := filepath.Join(tmpDir, "cur")

	// Create both directories
	if err := os.MkdirAll(newDir, 0755); err != nil {
		t.Fatalf("Failed to create new dir: %v", err)
	}
	if err := os.MkdirAll(curDir, 0755); err != nil {
		t.Fatalf("Failed to create cur dir: %v", err)
	}

	// Create a test email in new/ directory
	newEmailPath := filepath.Join(newDir, "12345.new")
	newEmailContent := `From: test@example.com
Subject: New Email Test
Date: Wed, 11 Mar 2026 12:00:00 +0000

Test message from new directory.`

	if err := os.WriteFile(newEmailPath, []byte(newEmailContent), 0644); err != nil {
		t.Fatalf("Failed to write test email: %v", err)
	}

	// Create a test email in cur/ directory (simulating read but not responded)
	curEmailPath := filepath.Join(curDir, "12346.cur")
	curEmailContent := `From: read@example.com
Subject: Read Email Test
Date: Wed, 11 Mar 2026 12:01:00 +0000

Test message from cur directory.`

	if err := os.WriteFile(curEmailPath, []byte(curEmailContent), 0644); err != nil {
		t.Fatalf("Failed to write test email: %v", err)
	}

	// Test that readMaildir can find emails in both directories
	newEmails, _, err := readMaildir(newDir, curDir, false)
	if err != nil {
		t.Errorf("readMaildir for new/ failed: %v", err)
	}

	curEmails, _, err := readMaildir(curDir, newDir, false)
	if err != nil {
		t.Errorf("readMaildir for cur/ failed: %v", err)
	}

	// Verify we found emails in both directories
	if len(newEmails) == 0 {
		t.Error("Expected to find at least one email in new/ directory")
	}
	if len(curEmails) == 0 {
		t.Error("Expected to find at least one email in cur/ directory")
	}

	// Verify content was read correctly
	foundNew := false
	foundCur := false

	for _, email := range newEmails {
		if strings.Contains(email.Content, "new directory") {
			foundNew = true
		}
	}

	for _, email := range curEmails {
		if strings.Contains(email.Content, "cur directory") {
			foundCur = true
		}
	}

	if !foundNew {
		t.Error("Expected to find email with 'new directory' content")
	}
	if !foundCur {
		t.Error("Expected to find email with 'cur directory' content")
	}
}

func TestProcessInboxWithResponseHandlesEmptyDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	newDir := filepath.Join(tmpDir, "new")
	curDir := filepath.Join(tmpDir, "cur")

	// Create both directories but leave them empty
	if err := os.MkdirAll(newDir, 0755); err != nil {
		t.Fatalf("Failed to create new dir: %v", err)
	}
	if err := os.MkdirAll(curDir, 0755); err != nil {
		t.Fatalf("Failed to create cur dir: %v", err)
	}

	// Test that the function handles empty directories gracefully
	newEmails, _, err := readMaildir(newDir, curDir, false)
	if err != nil && !os.IsNotExist(err) {
		t.Errorf("readMaildir should not error on empty dirs: %v", err)
	}

	curEmails, _, err := readMaildir(curDir, newDir, false)
	if err != nil && !os.IsNotExist(err) {
		t.Errorf("readMaildir should not error on empty dirs: %v", err)
	}

	// Should return empty slices, not error
	if len(newEmails) != 0 {
		t.Errorf("Expected empty slice for new/, got %d emails", len(newEmails))
	}
	if len(curEmails) != 0 {
		t.Errorf("Expected empty slice for cur/, got %d emails", len(curEmails))
	}
}

func TestProcessInboxWithResponseHandlesMissingDirectories(t *testing.T) {
	// Test with directories that don't exist
	nonExistentDir := "/tmp/definitely_does_not_exist_12345"

	_, _, err := readMaildir(nonExistentDir, nonExistentDir+"_also_missing", false)
	if err == nil {
		t.Error("Expected error when directory doesn't exist")
	}

	// Verify it's the correct type of error
	if !os.IsNotExist(err) {
		t.Errorf("Expected IsNotExist error, got: %v", err)
	}
}
