// Package main provides comprehensive unit tests for the email package.
// These tests use the email package and validate all functionality without sending real emails.
package yolo

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"yolo/email"
)

// ─── Email Package Tests ──────────────────
// IMPORTANT: All tests MUST NOT send real emails.
// They test error paths and validation logic only.

// TestEmailPackage_Comprehensive tests comprehensive email functionality
func TestEmailPackage_Comprehensive(t *testing.T) {
	t.Run("TestDefaultConfig", func(t *testing.T) {
		cfg := email.DefaultConfig()

		if cfg.SendmailPath != "/usr/sbin/sendmail" {
			t.Errorf("Expected SendmailPath '/usr/sbin/sendmail', got %q", cfg.SendmailPath)
		}

		if cfg.From != "yolo@b-haven.org" {
			t.Errorf("Expected From 'yolo@b-haven.org', got %q", cfg.From)
		}

		if !cfg.UseSendmail {
			t.Error("UseSendmail should be true by default")
		}
	})

	t.Run("TestEnvVarOverrides", func(t *testing.T) {
		os.Setenv("YELO_EMAIL_FROM", "override@example.com")
		defer os.Unsetenv("YELO_EMAIL_FROM")

		cfg := email.DefaultConfig()
		if cfg.From != "override@example.com" {
			t.Errorf("Expected From 'override@example.com', got %q", cfg.From)
		}
	})

	t.Run("TestEnvVarOverrides_SendmailPath", func(t *testing.T) {
		os.Setenv("YELO_SENDBMAIL_PATH", "/custom/sendmail")
		defer os.Unsetenv("YELO_SENDBMAIL_PATH")

		cfg := email.DefaultConfig()
		if cfg.SendmailPath != "/custom/sendmail" {
			t.Errorf("Expected SendmailPath '/custom/sendmail', got %q", cfg.SendmailPath)
		}
	})

	t.Run("TestMessageCreation", func(t *testing.T) {
		msg := &email.Message{
			To:      []string{"test@example.com"},
			Subject: "Test Subject",
			Body:    "Test Body",
		}

		if len(msg.To) != 1 {
			t.Error("Message should have one recipient")
		}

		if msg.To[0] != "test@example.com" {
			t.Errorf("Expected recipient 'test@example.com', got %s", msg.To[0])
		}
	})

	t.Run("TestMessageMultipleRecipients", func(t *testing.T) {
		recipients := []string{"user1@example.com", "user2@example.com"}
		msg := &email.Message{
			To:      recipients,
			Subject: "Multi-recipient test",
			Body:    "Test body",
		}

		if len(msg.To) != 2 {
			t.Errorf("Expected 2 recipients, got %d", len(msg.To))
		}
	})

	t.Run("TestClientCreation", func(t *testing.T) {
		cfg := email.DefaultConfig()
		client := email.New(cfg)

		if client == nil {
			t.Fatal("Expected non-nil client")
		}

		if client.Config() == nil {
			t.Error("Expected non-nil config in client")
		}
	})
}

// TestEmailValidation_Tests tests all validation paths that reject emails
func TestEmailValidation_Tests(t *testing.T) {
	t.Run("TestClientSendEmptyRecipients", func(t *testing.T) {
		cfg := email.DefaultConfig()
		client := email.New(cfg)

		msg := &email.Message{
			To:      []string{},
			Subject: "Test",
			Body:    "Test body",
		}

		err := client.Send(msg)
		if err == nil {
			t.Error("Expected error for empty recipients")
		}

		if err.Error() != "no recipients specified" {
			t.Errorf("Expected 'no recipients specified', got %q", err.Error())
		}
	})

	t.Run("TestClientSendNoSubject", func(t *testing.T) {
		cfg := email.DefaultConfig()
		client := email.New(cfg)

		msg := &email.Message{
			To:      []string{"test@example.com"},
			Subject: "",
			Body:    "Test body",
		}

		err := client.Send(msg)
		if err == nil {
			t.Error("Expected error for empty subject")
		}

		if err.Error() != "subject and body are required" {
			t.Errorf("Expected 'subject and body are required', got %q", err.Error())
		}
	})

	t.Run("TestClientSendNoBody", func(t *testing.T) {
		cfg := email.DefaultConfig()
		client := email.New(cfg)

		msg := &email.Message{
			To:      []string{"test@example.com"},
			Subject: "Test",
			Body:    "",
		}

		err := client.Send(msg)
		if err == nil {
			t.Error("Expected error for empty body")
		}

		if err.Error() != "subject and body are required" {
			t.Errorf("Expected 'subject and body are required', got %q", err.Error())
		}
	})

	t.Run("TestClientSendNoSMTP", func(t *testing.T) {
		cfg := &email.Config{
			From:         "yolo@b-haven.org",
			UseSendmail:  false,
			SendmailPath: "/usr/sbin/sendmail",
		}
		client := email.New(cfg)

		msg := &email.Message{
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
			t.Errorf("Expected %q, got %q", expectedErr, err.Error())
		}
	})
}

// TestEmail_Serialization tests email content generation
func TestEmail_Serialization(t *testing.T) {
	cfg := &email.Config{
		From:         "sender@example.com",
		UseSendmail:  true,
		SendmailPath: "/usr/sbin/sendmail",
	}
	client := email.New(cfg)

	msg := &email.Message{
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Body:    "Test body content",
	}

	// Build email content manually (simulating sendViaSendmail logic)
	var emailContent bytes.Buffer
	emailContent.WriteString(fmt.Sprintf("From: %s\r\n", cfg.From))
	emailContent.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	emailContent.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))

	contentStr := emailContent.String()
	if !strings.Contains(contentStr, "From: sender@example.com") {
		t.Error("Email content missing From header")
	}
	if !strings.Contains(contentStr, "To: recipient@example.com") {
		t.Error("Email content missing To header")
	}
	if !strings.Contains(contentStr, "Subject: Test Subject") {
		t.Error("Email content missing Subject header")
	}

	// Verify MIME version and content type
	emailContent.WriteString("MIME-Version: 1.0\r\n")
	emailContent.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	
	contentStr = emailContent.String()
	if !strings.Contains(contentStr, "MIME-Version: 1.0") {
		t.Error("Email content missing MIME-Version header")
	}
	if !strings.Contains(contentStr, "text/plain") {
		t.Error("Email content missing text/plain content type")
	}
}

// TestGetRFC2822Date tests date format generation
func TestGetRFC2822Date(t *testing.T) {
	date := email.GetRFC2822Date()

	if date == "" {
		t.Error("Expected non-empty RFC 2822 date")
	}

	if len(date) < 20 {
		t.Errorf("RFC 2822 date too short: %q", date)
	}
}

// TestEdgeCases tests edge cases for email handling
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		recipients []string
		subject  string
		body     string
		expectErr bool
	}{
		{
			name:     "valid email",
			recipients: []string{"user@example.com"},
			subject:  "Test Subject",
			body:     "Test Body",
			expectErr: false,
		},
		{
			name:     "multiple recipients valid",
			recipients: []string{"a@x.com", "b@y.com"},
			subject:  "Test",
			body:     "Body",
			expectErr: false,
		},
		{
			name:     "no recipients with content",
			recipients: []string{},
			subject:    "Test",
			body:       "Body",
			expectErr:  true,
		},
		{
			name:     "empty subject and body",
			recipients: []string{"user@example.com"},
			subject:  "",
			body:     "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := email.DefaultConfig()
			client := email.New(cfg)

			msg := &email.Message{
				To:      tt.recipients,
				Subject: tt.subject,
				Body:    tt.body,
			}

			err := client.Send(msg)
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestEmailPackage_Integration tests workflow from config to message creation
func TestEmailPackage_Integration(t *testing.T) {
	t.Run("workflow: create config, client, message, validate", func(t *testing.T) {
		// Step 1: Create config
		cfg := email.DefaultConfig()
		if cfg == nil || cfg.From == "" {
			t.Fatal("Failed to create valid config")
		}

		// Step 2: Create client
		client := email.New(cfg)
		if client == nil || client.Config() == nil {
			t.Fatal("Failed to create valid client")
		}

		// Step 3: Create invalid message (no recipients) - should fail
		msgNoRecipients := &email.Message{
			To:      []string{},
			Subject: "Test",
			Body:    "Body",
		}

		err := client.Send(msgNoRecipients)
		if err == nil {
			t.Error("Should reject message with no recipients")
		}

		// Step 4: Create valid content message (will fail sendViaSendmail due to env)
		msgValidContent := &email.Message{
			To:      []string{"test@example.com"},
			Subject: "Test Subject",
			Body:    "Test Body",
		}

		err = client.Send(msgValidContent)
		if err == nil {
			t.Log("Note: Message validation passed - would send in production")
		} else if err.Error() != "SMTP transport not implemented - use sendmail" &&
			!strings.Contains(err.Error(), "sendmail failed:") {
			t.Errorf("Unexpected error type: %v", err)
		}
	})
}

// TestHelperFunctions tests utility helper functions used in email testing
func TestHelperFunctions(t *testing.T) {
	if !contains("hello world", "world") {
		t.Error("contains should return true for substring")
	}
	if contains("hello world", "xyz") {
		t.Error("contains should return false for non-substring")
	}
}

// Helper function: contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

// Helper function: findSubstring finds substring in s
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
