package main

import (
	"strings"
	"testing"
	"time"
)

// TestSanitizeContentCommandInjection tests that command injection patterns are removed
func TestSanitizeContentCommandInjection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no injection",
			input:    "This is safe content",
			expected: "This is safe content",
		},
		{
			name:     "semicolon injection",
			input:    `rm -rf /; echo "injected"`,
			expected: `rm -rf / echo "injected"`,
		},
		{
			name:     "pipe injection",
			input:    `cat file.txt | malicious_cmd`,
			expected: `cat file.txt  malicious_cmd`,
		},
		{
			name:     "dollar command substitution",
			input:    `echo $(rm -rf /)`,
			expected: `echo [COMMAND_REDACTED]`,
		},
		{
			name:     "backtick injection",
			input:    `echo ` + "`rm -rf /`" + ``,
			expected: `echo [COMMAND_REDACTED]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeContent(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeContent(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSanitizeContentTemplateInjection tests that template injection markers are removed
func TestSanitizeContentTemplateInjection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no injection",
			input:    "Normal content",
			expected: "Normal content",
		},
		{
			name:     "simple template injection",
			input:    `{{ malicious code }}`,
			expected: `[REDACTED]`,
		},
		{
			name:     "nested template injection",
			input:    `Hello {{ user.name }}! How are you?`,
			expected: `Hello [REDACTED]! How are you?`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeContent(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeContent(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSanitizeContentTruncation tests that very long content is truncated
func TestSanitizeContentTruncation(t *testing.T) {
	longContent := strings.Repeat("A", 15000)
	result := sanitizeContent(longContent)

	if len(result) >= 15000 || !strings.Contains(result, "[CONTENT TRUNCATED") {
		t.Errorf("Content not properly truncated: length=%d", len(result))
	}
}

// TestEncodeHeader tests header injection prevention
func TestEncodeHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal header",
			input:    "Test Subject",
			expected: "Test Subject",
		},
		{
			name:     "newline injection",
			input:    "Test\r\nBcc: attacker@evil.com",
			expected: "Test Bcc: attacker@evil.com",
		},
		{
			name:     "carriage return injection",
			input:    "Test\rHeader continuation",
			expected: "Test Header continuation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeHeader(tt.input)
			if result != tt.expected {
				t.Errorf("encodeHeader(%q) = %q, want %q", tt.input, result, tt.expected)
			}

			// Ensure no newlines remain
			if strings.Contains(result, "\n") || strings.Contains(result, "\r") {
				t.Errorf("encodeHeader still contains newlines: %q", result)
			}
		})
	}
}

// TestValidateSender tests sender validation and denylist
func TestValidateSender(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid email",
			input:    "user@example.com",
			expected: true,
		},
		{
			name:     "valid email with subdomain",
			input:    "user@mail.example.com",
			expected: true,
		},
		{
			name:     "invalid email (no domain)",
			input:    "notanemail",
			expected: false,
		},
		{
			name:     "valid looking but denylisted",
			input:    "user@test.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateSender(tt.input)
			if result != tt.expected {
				t.Errorf("validateSender(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestCheckEmailCooldown tests rate limiting functionality
func TestCheckEmailCooldown(t *testing.T) {
	tests := []struct {
		name     string
		setup    func()
		expected bool
	}{
		{
			name: "no emails sent yet",
			setup: func() {
				hourStart.Store(time.Now().Unix())
				emailCount.Store(0)
				lastEmailTime.Store(nil)
			},
			expected: true,
		},
		{
			name: "within cooldown period",
			setup: func() {
				hourStart.Store(time.Now().Unix())
				emailCount.Store(5)
				lastEmailTime.Store(time.Now())
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result := checkEmailCooldown()
			if result != tt.expected {
				t.Errorf("checkEmailCooldown() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestEncodeHeaderTruncation tests that very long headers are truncated
func TestEncodeHeaderTruncation(t *testing.T) {
	longHeader := strings.Repeat("A", 1000)
	result := encodeHeader(longHeader)

	if len(result) > 503 || !strings.HasSuffix(result, "..") {
		t.Errorf("Header not properly truncated: length=%d, ends_with='..': %v", len(result), strings.HasSuffix(result, ".."))
	}
}

// TestSanitizeContentMultipleInjection tests multiple injection attempts
func TestSanitizeContentMultipleInjection(t *testing.T) {
	input := `
Hello {{ user.name }},
You can execute: $(rm -rf /)
Or use backticks: ` + "`cat /etc/passwd`" + `
Separator: ; rm -rf /
Pipe: | malicious_cmd
`

	result := sanitizeContent(input)

	// Check all injection markers are removed
	injectionPatterns := []string{"{{", "$(rm", "`cat", "; rm", "| malicious"}
	for _, pattern := range injectionPatterns {
		if strings.Contains(result, pattern) {
			t.Errorf("Injection pattern '%s' still present in sanitized content", pattern)
		}
	}
}

// TestRateLimitingOverHour tests that rate limit resets after an hour
func TestRateLimitingOverHour(t *testing.T) {
	now := time.Now().Unix()
	hourStart.Store(now)
	emailCount.Store(MaxEmailsPerHour)

	// Initially should be rate limited
	if checkEmailCooldown() {
		t.Errorf("Expected rate limiting to block send")
	}

	// Simulate hour passing
	hourStart.Store(now + 3601)
	emailCount.Store(0)

	// Should now allow send
	if !checkEmailCooldown() {
		t.Errorf("Expected rate limit reset after hour")
	}
}
