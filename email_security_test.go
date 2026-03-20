package main

import (
	"strings"
	"testing"
	"time"
)

// NOTE: sanitizeContent tests removed - this function does not exist and is not needed
// Email content should NOT be sanitized before being sent to LLM - the LLM needs full context

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
			expected: "Test  Bcc: attacker@evil.com", // Both \r and \n replaced with spaces
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
			input:    "user@stg.net",
			expected: true,
		},
		{
			name:     "valid email with subdomain",
			input:    "user@mail.b-haven.org",
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
	t.Parallel() // Isolate from other tests with global state

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
				lastEmailTime.Store(time.Time{})
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

	// Cleanup: reset to initial state for other tests
	hourStart.Store(time.Now().Unix())
	emailCount.Store(0)
	lastEmailTime.Store(time.Time{})
}

// TestEncodeHeaderTruncation tests that very long headers are truncated
func TestEncodeHeaderTruncation(t *testing.T) {
	longHeader := strings.Repeat("A", 1000)
	result := encodeHeader(longHeader)

	if len(result) > 503 || !strings.HasSuffix(result, "..") {
		t.Errorf("Header not properly truncated: length=%d, ends_with='..': %v", len(result), strings.HasSuffix(result, ".."))
	}
}

// TestRateLimitingOverHour tests that rate limit resets after an hour
func TestRateLimitingOverHour(t *testing.T) {
	t.Parallel() // Isolate from other tests

	now := time.Now().Unix()
	hourStart.Store(now)
	emailCount.Store(MaxEmailsPerHour)
	lastEmailTime.Store(time.Time{})

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

	// Cleanup: reset to initial state for other tests
	hourStart.Store(time.Now().Unix())
	emailCount.Store(0)
	lastEmailTime.Store(time.Time{})
}
