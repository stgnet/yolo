// Integration tests for process_inbox_with_response tool
package main

import (
	"os"
	"testing"
)

// TestProcessInboxWithResponse_RequiresEmail verifies the email directory exists
func TestProcessInboxWithResponse_RequiresEmail(t *testing.T) {
	emailDir := "/var/mail/b-haven.org/yolo/new/"

	if _, err := os.Stat(emailDir); os.IsNotExist(err) {
		t.Skip("Email inbox directory does not exist - skipping integration test")
	}

	executor := NewToolExecutor("/tmp", nil)
	result := executor.processInboxWithResponse(nil)

	t.Logf("Process inbox with response result: %s", result)

	// Should return success (even if no emails)
	if result == "" {
		t.Error("Expected non-empty result from process_inbox_with_response")
	}
}

// TestProcessInboxWithResponse_RespondToQuestions tests email response heuristics
func TestProcessInboxWithResponse_RespondToQuestions(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		content string
		expect  bool
	}{
		{"Question mark", "Can you help?", "Please assist", true},
		{"Request word", "Need help", "Please help me", true},
		{"System log", "Build completed", "Build finished successfully at 10:00 AM", false},
		{"Scott from scott@stg.net", "Test", "Hello Scott", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := EmailMessage{
				From:    tt.subject,
				Subject: tt.subject,
				Content: tt.content,
			}

			// Test heuristic for human email addresses only
			if tt.name != "Scott from scott@stg.net" {
				email.From = "test@example.com"
			} else {
				email.From = "scott@stg.net"
			}

			result := emailShouldRespond(email)

			if result != tt.expect {
				t.Errorf("emailShouldRespond(%s, %s) = %v; want %v",
					tt.subject, tt.content, result, tt.expect)
			}
		})
	}
}
