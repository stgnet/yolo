// Email package tests

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
