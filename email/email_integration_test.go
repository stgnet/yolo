// Integration tests for email package (skipped by default)

package email

import (
	"os"
	"testing"
)

func TestSend_Integration(t *testing.T) {
	// Skip unless EMAIL_PASSWORD is set
	password := os.Getenv("EMAIL_PASSWORD")
	if password == "" {
		t.Skip("Skipping integration test: EMAIL_PASSWORD not set")
	}

	cfg := DefaultConfig()
	msg := &Message{
		To:      []string{cfg.Username}, // Send to self
		Subject: "YOLO Email Integration Test",
		Body:    "This is a test email from YOLO's email package.",
	}

	err := Send(cfg, msg)
	if err != nil {
		t.Errorf("Failed to send email: %v", err)
	}
}

func TestSendReport_Integration(t *testing.T) {
	// Skip unless EMAIL_PASSWORD is set
	password := os.Getenv("EMAIL_PASSWORD")
	if password == "" {
		t.Skip("Skipping integration test: EMAIL_PASSWORD not set")
	}

	cfg := DefaultConfig()
	subject := "YOLO Progress Report Test"
	body := "This is a test progress report from YOLO."

	err := SendReport(cfg, subject, body)
	if err != nil {
		t.Errorf("Failed to send report: %v", err)
	}
}
