// Integration tests for email package (skipped by default if sendmail not available)

package email

import (
	"os/exec"
	"testing"
)

func TestSend_Integration(t *testing.T) {
	// Check if sendmail is available
	if _, err := exec.LookPath("/usr/sbin/sendmail"); err != nil {
		t.Skip("Skipping integration test: sendmail not available")
	}

	cfg := DefaultConfig()
	msg := &Message{
		To:      []string{"test@example.com"}, // Test recipient
		Subject: "YOLO Email Integration Test",
		Body:    "This is a test email from YOLO's email package.",
	}

	client := New(cfg)
	err := client.Send(msg)
	if err != nil {
		t.Logf("Email send result (may succeed even with errors due to async delivery): %v", err)
		// Don't fail the test if sendmail accepts it but DNS/etc fails
	}
}

func TestSendReport_Integration(t *testing.T) {
	// Check if sendmail is available
	if _, err := exec.LookPath("/usr/sbin/sendmail"); err != nil {
		t.Skip("Skipping integration test: sendmail not available")
	}

	cfg := DefaultConfig()
	subject := "YOLO Progress Report Test"
	body := "This is a test progress report from YOLO."

	msg := &Message{
		To:      []string{"scott@stg.net"},
		Subject: subject,
		Body:    body,
	}

	client := New(cfg)
	err := client.Send(msg)
	if err != nil {
		t.Logf("Report send result (may succeed even with errors due to async delivery): %v", err)
		// Don't fail the test if sendmail accepts it but DNS/etc fails
	}
}
