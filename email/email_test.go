// Email package tests

package email

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.SMTPHost == "" {
		t.Error("SMTPHost should have a default value")
	}

	// Sendmail is the default, SMTP settings are fallbacks
	if cfg.SendmailPath == "" {
		t.Error("SendmailPath should have a default value")
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()

	expectedHost := "localhost"
	if cfg.SMTPHost != expectedHost {
		t.Errorf("Expected SMTPHost %q, got %q", expectedHost, cfg.SMTPHost)
	}

	expectedSendmail := "/usr/sbin/sendmail"
	if cfg.SendmailPath != expectedSendmail {
		t.Errorf("Expected SendmailPath %q, got %q", expectedSendmail, cfg.SendmailPath)
	}

	expectedPort := 25
	if cfg.SMTPPort != expectedPort {
		t.Errorf("Expected SMTPPort %d, got %d", expectedPort, cfg.SMTPPort)
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

func TestPrepareMessage(t *testing.T) {
	c := Client{config: DefaultConfig()}
	msg := &Message{
		To:      []string{"test@example.com"},
		Subject: "Test Subject",
		Body:    "Test Body",
	}

	body := c.prepareMessage(msg)

	if !contains(body, "From: yolo@b-haven.org") {
		t.Error("Message should contain default From address")
	}

	if !contains(body, "To: test@example.com") {
		t.Error("Message should contain To address")
	}

	if !contains(body, "Subject: Test Subject") {
		t.Error("Message should contain Subject")
	}

	if !contains(body, "Test Body") {
		t.Error("Message should contain Body")
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
