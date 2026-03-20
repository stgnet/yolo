package main

import (
	"strings"
	"testing"
)

// TestParseEmailHeaderSanitization tests that email headers are properly sanitized
func TestParseEmailHeaderSanitization(t *testing.T) {
	emailContent := `From: attacker@evil.com
Subject: Test\r\nBcc: victim@target.com
Date: Mon, 1 Jan 2024 00:00:00 +0000
To: yolo@b-haven.org

Test email content`

	email := parseEmail(emailContent, "test_email")

	// Header should be sanitized (no newlines)
	if strings.Contains(email.From, "\n") || strings.Contains(email.Subject, "\n") {
		t.Errorf("Headers still contain newlines: From=%q, Subject=%q", email.From, email.Subject)
	}
}

// TestIsBounceMessage tests bounce message detection
func TestIsBounceMessage(t *testing.T) {
	tests := []struct {
		name     string
		email    *EmailMessage
		expected bool
	}{
		{
			name: "bounce message - delivery failed",
			email: &EmailMessage{
				From:    "mailer-daemon@example.com",
				Subject: "Delivery Failed",
				Content: "Your email delivery failed",
			},
			expected: true,
		},
		{
			name: "bounce message - postmaster",
			email: &EmailMessage{
				From:    "postmaster@example.com",
				Subject: "Undeliverable Message",
				Content: "This message could not be delivered",
			},
			expected: true,
		},
		{
			name: "normal message",
			email: &EmailMessage{
				From:    "user@example.com",
				Subject: "Hello there!",
				Content: "Just checking in",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBounceMessage(tt.email)
			if result != tt.expected {
				t.Errorf("isBounceMessage() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestSanitizeEmailField tests email field sanitization
func TestSanitizeEmailField(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool // True if field should be truncated
	}{
		{
			name:     "normal field",
			input:    "user@example.com",
			expected: false,
		},
		{
			name:     "field with newline injection",
			input:    "user@example.com\r\nBcc: attacker@evil.com",
			expected: true,
		},
		{
			name:     "very long field",
			input:    strings.Repeat("A", 600),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeEmailField(tt.input)

			// Should not contain newlines
			if strings.Contains(result, "\n") || strings.Contains(result, "\r") {
				t.Errorf("sanitizeEmailField still contains newlines")
			}

			// Should be truncated if too long (max 500 chars including suffix)
			if tt.expected && len(result) > 500 {
				t.Errorf("sanitizeEmailField not properly truncated: length=%d", len(result))
			}
		})
	}
}

// TestTruncateString tests content truncation helper
func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		limit    int
		expected bool // True if truncated
	}{
		{
			name:     "within limit",
			input:    "Short text",
			limit:    50,
			expected: false,
		},
		{
			name:     "at limit",
			input:    strings.Repeat("A", 50),
			limit:    50,
			expected: false,
		},
		{
			name:     "over limit",
			input:    strings.Repeat("A", 100),
			limit:    50,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.limit)

			if len(result) > tt.limit && !tt.expected {
				t.Errorf("Truncate incorrectly truncated string of length %d with limit %d", len(tt.input), tt.limit)
			}

			if tt.expected && len(result) <= tt.limit {
				t.Errorf("Truncate did not truncate string that should have been truncated")
			}
		})
	}
}
