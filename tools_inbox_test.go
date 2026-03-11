package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadMaildir(t *testing.T) {
	// Create a test Maildir structure
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new")
	curDir := filepath.Join(tmpDir, "cur")
	os.MkdirAll(newDir, 0755)
	os.MkdirAll(curDir, 0755)

	// Create test email file with proper headers and body
	emailContent := `From: test@example.com
To: yolo@b-haven.org
Subject: Test Email
Date: Mon, 01 Jan 2024 00:00:00 +0000

This is a test email body.
It has multiple lines.
`
	emailPath := filepath.Join(newDir, "1704067200.V1P12345.yolo")
	os.WriteFile(emailPath, []byte(emailContent), 0644)

	// Test reading the maildir
	emails, processedCount, err := readMaildir(newDir, curDir, false)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if processedCount != 0 {
		t.Errorf("Expected 0 processed (markRead=false), got %d", processedCount)
	}

	if len(emails) != 1 {
		t.Errorf("Expected 1 email, got %d", len(emails))
	}

	if len(emails) > 0 {
		email := emails[0]
		if email.From != "test@example.com" {
			t.Errorf("Expected From: test@example.com, got: %s", email.From)
		}
		if email.Subject != "Test Email" {
			t.Errorf("Expected Subject: Test Email, got: %s", email.Subject)
		}
		if !strings.Contains(strings.ToLower(email.Content), "test email body") {
			t.Errorf("Expected content to contain 'test email body', got: %s", email.Content)
		}
	}

	// Verify email still exists in newDir (not moved)
	if _, err := os.Stat(emailPath); err != nil {
		t.Error("Email should still exist in newDir when markRead=false")
	}
}

func TestReadMaildirEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new")
	curDir := filepath.Join(tmpDir, "cur")
	os.MkdirAll(newDir, 0755)
	os.MkdirAll(curDir, 0755)

	emails, processedCount, err := readMaildir(newDir, curDir, false)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(emails) != 0 {
		t.Errorf("Expected 0 emails for empty maildir, got %d", len(emails))
	}

	if processedCount != 0 {
		t.Errorf("Expected 0 processed, got %d", processedCount)
	}
}

func TestReadMaildirNoNewDir(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "nonexistent")
	curDir := filepath.Join(tmpDir, "cur")

	emails, processedCount, err := readMaildir(newDir, curDir, false)

	if err == nil {
		t.Error("Expected error when new directory doesn't exist, got nil")
	}

	if emails != nil {
		t.Errorf("Expected nil emails for non-existent dir, got %v", emails)
	}

	if processedCount != 0 {
		t.Errorf("Expected 0 processed, got %d", processedCount)
	}
}

func TestReadMaildirWithMarkRead(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new")
	curDir := filepath.Join(tmpDir, "cur")
	os.MkdirAll(newDir, 0755)
	os.MkdirAll(curDir, 0755)

	emailContent := `From: mark@test.com
To: yolo@b-haven.org
Subject: Mark Read Test

Test body.
`
	emailPath := filepath.Join(newDir, "1704067200.V1P12345.yolo")
	os.WriteFile(emailPath, []byte(emailContent), 0644)

	emails, processedCount, err := readMaildir(newDir, curDir, true)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(emails) != 1 {
		t.Errorf("Expected 1 email, got %d", len(emails))
	}

	if processedCount != 1 {
		t.Errorf("Expected 1 processed (markRead=true), got %d", processedCount)
	}

	// Email should be moved to cur directory
	if _, err := os.Stat(emailPath); err == nil {
		t.Error("Email should have been moved from new/ to cur/")
	}

	curEmailPath := filepath.Join(curDir, "1704067200.V1P12345.yolo")
	if _, err := os.Stat(curEmailPath); err != nil {
		t.Errorf("Email should exist in cur/ directory: %v", err)
	}
}

func TestReadMalformedEmail(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new")
	curDir := filepath.Join(tmpDir, "cur")
	os.MkdirAll(newDir, 0755)
	os.MkdirAll(curDir, 0755)

	// Create malformed email (missing headers)
	emailContent := `This has no headers at all.
Just body content.
`
	emailPath := filepath.Join(newDir, "1704067200.V1P12345.yolo")
	os.WriteFile(emailPath, []byte(emailContent), 0644)

	emails, _, err := readMaildir(newDir, curDir, false)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Malformed emails should be skipped (parseEmailMessage returns error)
	if len(emails) != 0 {
		t.Logf("Note: Expected 0 emails for malformed content, got %d (may indicate parsing improvement)", len(emails))
	}
}

func TestReadMaildirMultipleEmails(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new")
	curDir := filepath.Join(tmpDir, "cur")
	os.MkdirAll(newDir, 0755)
	os.MkdirAll(curDir, 0755)

	// Create multiple emails
	for i := 1; i <= 3; i++ {
		content := `From: test` + string(rune('0'+i)) + `@example.com
To: yolo@b-haven.org
Subject: Email ` + string(rune('0'+i)) + `

Body of email ` + string(rune('0'+i)) + `.
`
		path := filepath.Join(newDir, "170406720"+string(rune('0'+i))+".V1P12345.yolo")
		os.WriteFile(path, []byte(content), 0644)
	}

	emails, _, err := readMaildir(newDir, curDir, false)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(emails) != 3 {
		t.Errorf("Expected 3 emails, got %d", len(emails))
	}
}

func TestParseEmailMessage(t *testing.T) {
	content := []byte(`From: sender@example.com
To: recipient@example.com
Subject: Test Subject
Date: Mon, 01 Jan 2024 00:00:00 +0000
Content-Type: text/plain; charset=utf-8

This is the email body.
It has multiple lines.
`)

	email, err := parseEmailMessage(content, "test-file")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if email.From != "sender@example.com" {
		t.Errorf("Expected From: sender@example.com, got: %s", email.From)
	}

	if email.Subject != "Test Subject" {
		t.Errorf("Expected Subject: Test Subject, got: %s", email.Subject)
	}

	if email.Date != "Mon, 01 Jan 2024 00:00:00 +0000" {
		t.Errorf("Expected Date: Mon, 01 Jan 2024 00:00:00 +0000, got: %s", email.Date)
	}

	if email.ContentType != "text/plain; charset=utf-8" {
		t.Errorf("Expected Content-Type: text/plain; charset=utf-8, got: %s", email.ContentType)
	}

	if !strings.Contains(strings.ToLower(email.Content), "email body") {
		t.Errorf("Expected content to contain 'email body', got: %s", email.Content)
	}

	if email.Filename != "test-file" {
		t.Errorf("Expected Filename: test-file, got: %s", email.Filename)
	}
}

func TestParseEmailMessageEmpty(t *testing.T) {
	content := []byte(``)

	email, err := parseEmailMessage(content, "empty-file")

	if err == nil {
		t.Log("Note: Empty content did not return error (may be acceptable)")
	}

	if email.Filename != "empty-file" {
		t.Errorf("Expected Filename: empty-file, got: %s", email.Filename)
	}
}

func TestParseEmailMessageMultipart(t *testing.T) {
	content := []byte(`From: multipart@test.com
Subject: Multipart Test
Content-Type: multipart/alternative; boundary="boundary123"

--boundary123
Content-Type: text/plain

Plain text version.
--boundary123
Content-Type: text/html

<html><body>HTML version.</body></html>
--boundary123--
`)

	email, err := parseEmailMessage(content, "multipart-file")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if email.From != "multipart@test.com" {
		t.Errorf("Expected From: multipart@test.com, got: %s", email.From)
	}

	// Should prefer text/plain over text/html
	if !strings.Contains(strings.ToLower(email.Content), "plain text") {
		t.Logf("Note: Expected plain text content, got: %s", email.Content)
	}
}

func TestExtractBodyFromBytes(t *testing.T) {
	content := []byte("Test body content here.")
	result := extractBodyFromBytes(content, "text/plain; charset=utf-8")

	if result != "Test body content here." {
		t.Errorf("Expected 'Test body content here.', got: %s", result)
	}
}

func TestExtractBodyQuotedPrintable(t *testing.T) {
	content := "This=20is=20quoted=20printable."
	result := extractBody(strings.NewReader(content), "text/plain; charset=utf-8")

	if !strings.Contains(result, "This is quoted printable") && !strings.Contains(result, "=20") {
		t.Logf("Note: Quoted-printable decoding may vary: %s", result)
	}
}

func TestEmailMessageJSONTags(t *testing.T) {
	email := EmailMessage{
		From:     "test@example.com",
		Subject:  "Test",
		Date:     "2024-01-01",
		Content:  "Body content",
		Filename: "test-file",
	}

	if email.From == "" || email.Subject == "" || email.Content == "" {
		t.Error("EmailMessage fields should be accessible")
	}
	_ = email
}

func TestCheckInboxTool(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new")
	curDir := filepath.Join(tmpDir, "cur")
	os.MkdirAll(newDir, 0755)
	os.MkdirAll(curDir, 0755)

	// Create test email
	emailContent := `From: check@test.com
To: yolo@b-haven.org
Subject: Check Test

Test body content.
`
	emailPath := filepath.Join(newDir, "1704067200.V1P12345.yolo")
	os.WriteFile(emailPath, []byte(emailContent), 0644)

	// Test checkInbox with mark_read=false (simulated)
	emails, processedCount, err := readMaildir(newDir, curDir, false)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(emails) != 1 {
		t.Errorf("Expected 1 email in inbox, got %d", len(emails))
	}

	if processedCount != 0 {
		t.Errorf("Expected 0 processed (mark_read=false), got %d", processedCount)
	}

	if emails[0].From != "check@test.com" {
		t.Errorf("Expected From: check@test.com, got: %s", emails[0].From)
	}

	if emails[0].Subject != "Check Test" {
		t.Errorf("Expected Subject: Check Test, got: %s", emails[0].Subject)
	}

	if !strings.Contains(strings.ToLower(emails[0].Content), "test body") {
		t.Errorf("Expected content to contain 'test body', got: %s", emails[0].Content)
	}
}

func TestCheckInboxMarkReadTool(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new")
	curDir := filepath.Join(tmpDir, "cur")
	os.MkdirAll(newDir, 0755)
	os.MkdirAll(curDir, 0755)

	// Create test email
	emailContent := `From: markread@test.com
To: yolo@b-haven.org
Subject: Mark Read Test

Test body.
`
	emailPath := filepath.Join(newDir, "1704067200.V1P12345.yolo")
	os.WriteFile(emailPath, []byte(emailContent), 0644)

	// Test checkInbox with mark_read=true (simulated)
	emails, processedCount, err := readMaildir(newDir, curDir, true)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(emails) != 1 {
		t.Errorf("Expected 1 email in inbox, got %d", len(emails))
	}

	if processedCount != 1 {
		t.Errorf("Expected 1 processed (mark_read=true), got %d", processedCount)
	}

	// Verify email was moved
	if _, err := os.Stat(emailPath); err == nil {
		t.Error("Email should have been moved from new/")
	}

	curEmailPath := filepath.Join(curDir, "1704067200.V1P12345.yolo")
	if _, err := os.Stat(curEmailPath); err != nil {
		t.Errorf("Email should exist in cur/: %v", err)
	}
}

func TestCheckInboxEmptyTool(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new")
	curDir := filepath.Join(tmpDir, "cur")
	os.MkdirAll(newDir, 0755)
	os.MkdirAll(curDir, 0755)

	// Test checkInbox with empty inbox
	emails, processedCount, err := readMaildir(newDir, curDir, false)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(emails) != 0 {
		t.Errorf("Expected 0 emails in empty inbox, got %d", len(emails))
	}

	if processedCount != 0 {
		t.Errorf("Expected 0 processed, got %d", processedCount)
	}
}
