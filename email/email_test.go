// Email package tests
//
// **************************************************************************
// ** WARNING: SENDING REAL EMAILS IN TESTS IS STRICTLY FORBIDDEN.         **
// ** All tests in this file MUST validate logic WITHOUT invoking sendmail  **
// ** or any real email transport. Tests that need to send actual emails    **
// ** belong ONLY in email_integration_test.go, gated behind               **
// ** YOLO_TEST_EMAIL=1. DO NOT bypass this restriction.                   **
// **************************************************************************

package email

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.SendmailPath == "" {
		t.Error("SendmailPath should have a default value")
	}

	if cfg.From == "" {
		t.Error("From should have a default value")
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()

	expectedSendmail := "/usr/sbin/sendmail"
	if cfg.SendmailPath != expectedSendmail {
		t.Errorf("Expected SendmailPath %q, got %q", expectedSendmail, cfg.SendmailPath)
	}

	expectedFrom := "yolo@b-haven.org"
	if cfg.From != expectedFrom {
		t.Errorf("Expected From %q, got %q", expectedFrom, cfg.From)
	}
}

func TestConfigCustomValues(t *testing.T) {
	cfg := &Config{
		From:         "custom@example.com",
		UseSendmail:  true,
		SendmailPath: "/usr/local/sbin/sendmail",
	}

	if cfg.From != "custom@example.com" {
		t.Errorf("Expected From 'custom@example.com', got %q", cfg.From)
	}

	if !cfg.UseSendmail {
		t.Error("UseSendmail should be true")
	}

	if cfg.SendmailPath != "/usr/local/sbin/sendmail" {
		t.Errorf("Expected custom SendmailPath, got %q", cfg.SendmailPath)
	}
}

func TestMessageCreation(t *testing.T) {
	msg := &Message{
		To:      []string{"test@example.com"},
		Subject: "Test Subject",
		Body:    "Test Body",
	}

	if len(msg.To) != 1 {
		t.Error("Message should have one recipient")
	}

	if msg.To[0] != "test@example.com" {
		t.Errorf("Expected recipient test@example.com, got %s", msg.To[0])
	}

	if msg.Subject != "Test Subject" {
		t.Errorf("Expected subject 'Test Subject', got %q", msg.Subject)
	}

	if msg.Body != "Test Body" {
		t.Errorf("Expected body 'Test Body', got %q", msg.Body)
	}
}

func TestMessageMultipleRecipients(t *testing.T) {
	recipients := []string{"user1@example.com", "user2@example.com", "user3@example.com"}
	msg := &Message{
		To:      recipients,
		Subject: "Multi-recipient test",
		Body:    "Test body",
	}

	if len(msg.To) != 3 {
		t.Errorf("Expected 3 recipients, got %d", len(msg.To))
	}

	for i, expected := range recipients {
		if msg.To[i] != expected {
			t.Errorf("Recipient %d: expected %q, got %q", i, expected, msg.To[i])
		}
	}
}

func TestNewClient(t *testing.T) {
	cfg := DefaultConfig()
	client := New(cfg)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.config == nil {
		t.Fatal("Expected non-nil config in client")
	}

	if client.config.From != "yolo@b-haven.org" {
		t.Errorf("Expected client config From 'yolo@b-haven.org', got %q", client.config.From)
	}
}

// **************************************************************************
// ** VALIDATION TESTS BELOW — THESE MUST NEVER SEND REAL EMAILS.         **
// ** They test error paths that reject the message before sendmail runs.  **
// **************************************************************************

func TestClientSendNoRecipients(t *testing.T) {
	cfg := DefaultConfig()
	client := New(cfg)

	msg := &Message{
		To:      []string{},
		Subject: "Test",
		Body:    "Test body",
	}

	err := client.Send(msg)
	if err == nil {
		t.Error("Expected error for empty recipients")
	}

	expectedErr := "no recipients specified"
	if err.Error() != expectedErr {
		t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestClientSendNoSubject(t *testing.T) {
	cfg := DefaultConfig()
	client := New(cfg)

	msg := &Message{
		To:      []string{"test@example.com"},
		Subject: "",
		Body:    "Test body",
	}

	err := client.Send(msg)
	if err == nil {
		t.Error("Expected error for empty subject")
	}

	expectedErr := "subject and body are required"
	if err.Error() != expectedErr {
		t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestClientSendNoBody(t *testing.T) {
	cfg := DefaultConfig()
	client := New(cfg)

	msg := &Message{
		To:      []string{"test@example.com"},
		Subject: "Test",
		Body:    "",
	}

	err := client.Send(msg)
	if err == nil {
		t.Error("Expected error for empty body")
	}

	expectedErr := "subject and body are required"
	if err.Error() != expectedErr {
		t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestClientSendNoSMTP(t *testing.T) {
	cfg := &Config{
		From:         "yolo@b-haven.org",
		UseSendmail:  false,
		SendmailPath: "/usr/sbin/sendmail",
	}
	client := New(cfg)

	msg := &Message{
		To:      []string{"test@example.com"},
		Subject: "Test",
		Body:    "Test body",
	}

	err := client.Send(msg)
	if err == nil {
		t.Error("Expected error when SMTP not implemented")
	}

	expectedErr := "SMTP transport not implemented - use sendmail"
	if err.Error() != expectedErr {
		t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestGetRFC2822Date(t *testing.T) {
	date := getRFC2822Date()

	if date == "" {
		t.Error("Expected non-empty RFC 2822 date")
	}

	// RFC 2822 date format example: "Mon, 02 Jan 2006 15:04:05 -0700"
	// Should contain comma and timezone
	if len(date) < 20 {
		t.Errorf("RFC 2822 date too short: %q", date)
	}
}

// TestSendViaSendmailError verifies error handling when sendmail binary is
// missing. This uses a non-existent path so NO real email is sent.
func TestSendViaSendmailError(t *testing.T) {
	cfg := &Config{
		From:         "yolo@b-haven.org",
		UseSendmail:  true,
		SendmailPath: "/nonexistent/sendmail",
	}
	client := New(cfg)

	msg := &Message{
		To:      []string{"test@example.com"},
		Subject: "Test",
		Body:    "Test body",
	}

	err := client.sendViaSendmail(msg)
	if err == nil {
		t.Error("Expected error when sendmail path doesn't exist")
	}

	if err.Error()[:16] != "sendmail failed:" {
		t.Errorf("Expected 'sendmail failed:' prefix, got %q", err.Error())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
