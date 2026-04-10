package email

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestGetRFC2822Date tests RFC 2822 date formatting
func TestGetRFC2822Date(t *testing.T) {
	dateStr := getRFC2822Date()
	
	if dateStr == "" {
		t.Error("Expected non-empty date string")
		return
	}

	// RFC 2822 format: "02 Jan 2006 15:04:05 -0700"
	// Check for basic format components
	if len(dateStr) < 20 {
		t.Errorf("Date string too short: %q (expected at least 20 chars)", dateStr)
	}

	// Should contain a year (4 digits)
	if !strings.Contains(dateStr, fmt.Sprintf("%d", time.Now().Year())) {
		t.Errorf("Date should contain current year: expected to find %d in %q", 
			time.Now().Year(), dateStr)
	}

	// Should contain valid month abbreviation
	months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	foundMonth := false
	for _, month := range months {
		if strings.Contains(dateStr, month) {
			foundMonth = true
			break
		}
	}
	if !foundMonth {
		t.Errorf("Date should contain valid month abbreviation in %q", dateStr)
	}

	// Should contain timezone offset (contains + or - followed by digits)
	hasTimezone := false
	for i, c := range dateStr {
		if (c == '+' || c == '-') && i > 0 && i < len(dateStr)-4 {
			// Check if there are digits after the sign
			remaining := dateStr[i+1:]
			if len(remaining) >= 4 {
				hasTimezone = true
				break
			}
		}
	}
	if !hasTimezone {
		t.Errorf("Date should contain timezone offset in %q", dateStr)
	}
}

// TestSanitizeFilenameEdgeCases tests edge cases for filename sanitization
func TestSanitizeFilenameEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "filename with quotes",
			filename: "file\"name.pdf",
			expected: "filename.pdf",
		},
		{
			name:     "filename with newline",
			filename: "file\nname.pdf",
			expected: "file_name.pdf",
		},
		{
			name:     "filename with carriage return",
			filename: "file\rname.pdf",
			expected: "file_name.pdf",
		},
		{
			name:     "filename with CRLF",
			filename: "file\r\nname.pdf",
			expected: "file__name.pdf",
		},
		{
			name:     "filename with valid chars preserved",
			filename: "test-file_v2.0.pdf",
			expected: "test-file_v2.0.pdf",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeFilename(tc.filename)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// TestClientSend tests the Client.Send method with various scenarios
func TestClientSend(t *testing.T) {
	testCases := []struct {
		name        string
		msg         *Message
		expectError bool
		errorSubstr string
	}{
		{
			name: "empty recipients should fail validation",
			msg: &Message{
				To:      []string{},
				Subject: "Test Subject",
				Body:    "Test body content",
			},
			expectError: true,
			errorSubstr: "no recipients specified",
		},
		{
			name: "empty subject should fail validation",
			msg: &Message{
				To:      []string{"test@example.com"},
				Subject: "",
				Body:    "Test body content",
			},
			expectError: true,
			errorSubstr: "subject and body are required",
		},
		{
			name: "empty body should fail validation",
			msg: &Message{
				To:      []string{"test@example.com"},
				Subject: "Test Subject",
				Body:    "",
			},
			expectError: true,
			errorSubstr: "subject and body are required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := New(DefaultConfig())
			
			err := client.Send(tc.msg)
			
			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}
			
			if tc.errorSubstr != "" && tc.expectError {
				if err == nil || !strings.Contains(err.Error(), tc.errorSubstr) {
					t.Errorf("Error should contain %q, got: %v", tc.errorSubstr, err)
				}
			}
		})
	}
	
	// Test that valid messages don't crash (may succeed or fail depending on sendmail availability)
	t.Run("valid message does not panic", func(t *testing.T) {
		client := New(DefaultConfig())
		
		msg := &Message{
			To:      []string{"test@example.com"},
			Subject: "Test Subject",
			Body:    "Test body content",
		}
		
		// This should not panic, even if sendmail succeeds or fails
		err := client.Send(msg)
		// We don't assert on error here since sendmail may or may not be available
		_ = err
	})
}

// TestClientValidateMessage tests message validation logic
func TestClientValidateMessage(t *testing.T) {
	client := New(DefaultConfig())
	
	testCases := []struct {
		name        string
		msg         *Message
		expectError bool
	}{
		{
			name: "valid message",
			msg: &Message{
				To:      []string{"test@example.com"},
				Subject: "Test Subject",
				Body:    "Test body",
			},
			expectError: false,
		},
		{
			name: "multiple recipients",
			msg: &Message{
				To:      []string{"test1@example.com", "test2@example.com"},
				Subject: "Test Subject",
				Body:    "Test body",
			},
			expectError: false,
		},
		{
			name: "empty recipients",
			msg: &Message{
				To:      []string{},
				Subject: "Test Subject",
				Body:    "Test body",
			},
			expectError: true,
		},
		{
			name: "empty subject",
			msg: &Message{
				To:      []string{"test@example.com"},
				Subject: "",
				Body:    "Test body",
			},
			expectError: true,
		},
		{
			name: "empty body",
			msg: &Message{
				To:      []string{"test@example.com"},
				Subject: "Test Subject",
				Body:    "",
			},
			expectError: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := client.ValidateMessage(tc.msg)
			
			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}
		})
	}
}

// TestConfigCreation tests configuration creation functions
func TestConfigCreation(t *testing.T) {
	// Test DefaultConfig returns valid config
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	
	if !cfg.UseSendmail {
		t.Error("UseSendmail should be true by default")
	}
	
	if cfg.SendmailPath == "" {
		t.Error("SendmailPath should not be empty")
	}
	
	// Test ConfigWithFrom creates config with custom from address
	customFrom := "custom@example.com"
	cfg2 := ConfigWithFrom(customFrom)
	if cfg2.From != customFrom {
		t.Errorf("Expected From to be %q, got %q", customFrom, cfg2.From)
	}
	
	// Test ConfigWithFrom with empty string falls back to default/ENV
	cfg3 := ConfigWithFrom("")
	if cfg3.From == "" {
		t.Error("ConfigWithFrom with empty string should fall back to ENV or default")
	}
}

// TestNewClient tests client creation
func TestNewClient(t *testing.T) {
	config := DefaultConfig()
	client := New(config)
	
	if client == nil {
		t.Fatal("New returned nil client")
	}
	
	if client.config != config {
		t.Error("Client should store the provided config")
	}
}