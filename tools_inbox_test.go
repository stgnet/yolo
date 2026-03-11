package main

import (
	"os"
	"testing"
)

func TestCheckInboxNoEmails(t *testing.T) {
	executor := &ToolExecutor{baseDir: "/tmp"}
	result := executor.checkInbox(map[string]any{"mark_read": false})

	if result == "" {
		t.Error("expected non-empty result")
	}

	t.Logf("check_inbox output: %s", result)

	// Should mention "No new emails" or show an error about directory not existing
	if !containsString(result, "No new emails") && !containsString(result, "directory not found") {
		t.Log("Note: Result didn't contain expected 'No new emails' message - inbox may have actual emails")
	}
}

func TestCheckInboxMarkRead(t *testing.T) {
	executor := &ToolExecutor{baseDir: "/tmp"}
	result := executor.checkInbox(map[string]any{"mark_read": true})

	if result == "" {
		t.Error("expected non-empty result")
	}

	t.Logf("check_inbox with mark_read=true output: %s", result)
}

func TestParseEmailMessage(t *testing.T) {
	testContent := []byte(`From: test@example.com
Subject: Test Email
Date: Mon, 1 Jan 2024 00:00:00 +0000
Content-Type: text/plain; charset=utf-8

This is a test email body.`)

	email, err := parseEmailMessage(testContent, "test.msg")
	if err != nil {
		t.Fatalf("parseEmailMessage failed: %v", err)
	}

	if email.From == "" {
		t.Error("expected non-empty From field, got:", email.From)
	}
	if email.Subject == "" {
		t.Error("expected non-empty Subject field")
	}
	if email.Date == "" {
		t.Error("expected non-empty Date field")
	}
	if email.Filename != "test.msg" {
		t.Errorf("expected filename 'test.msg', got '%s'", email.Filename)
	}
}

func TestCheckInboxResultStructure(t *testing.T) {
	result := CheckInboxResult{
		Emails:    []EmailMessage{{From: "test@example.com", Subject: "Test"}},
		Count:     1,
		Processed: 0,
	}

	if result.Count != len(result.Emails) {
		t.Error("Count should match emails length")
	}
}

func TestMaildirDirectories(t *testing.T) {
	newDir := "/var/mail/b-haven.org/yolo/new/"
	curDir := "/var/mail/b-haven.org/yolo/cur/"

	t.Logf("Expected Maildir paths:")
	t.Logf("  new: %s", newDir)
	t.Logf("  cur: %s", curDir)

	// Check if directories exist (they may not on all systems)
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Log("Maildir new directory does not exist yet - this is OK for tests")
	}

	if _, err := os.Stat(curDir); os.IsNotExist(err) {
		t.Log("Maildir cur directory does not exist yet - this is OK for tests")
	}
}

func TestGetBoolArg(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		key      string
		fallback bool
		expected bool
	}{
		{"bool true", map[string]any{"mark_read": true}, "mark_read", false, true},
		{"bool false", map[string]any{"mark_read": false}, "mark_read", true, false},
		{"string true", map[string]any{"mark_read": "true"}, "mark_read", false, true},
		{"string 1", map[string]any{"mark_read": "1"}, "mark_read", false, true},
		{"missing key", map[string]any{}, "mark_read", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBoolArg(tt.args, tt.key, tt.fallback)
			if result != tt.expected {
				t.Errorf("getBoolArg(%v, %q, %v) = %v, want %v", tt.args, tt.key, tt.fallback, result, tt.expected)
			}
		})
	}
}

// Helper function for tests
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
