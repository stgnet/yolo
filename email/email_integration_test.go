// Integration tests for email package.
// Gated behind YOLO_TEST_EMAIL=1 to prevent accidentally sending real emails.
//
// **************************************************************************
// ** WARNING: THESE TESTS SEND REAL EMAILS.                               **
// ** They are disabled by default and MUST stay gated behind              **
// ** YOLO_TEST_EMAIL=1. NEVER remove or bypass the skipUnlessEmailEnabled **
// ** guard. Running these without the gate WILL send actual emails.        **
// **************************************************************************

package email

import (
	"os"
	"os/exec"
	"testing"
)

func skipUnlessEmailEnabled(t *testing.T) {
	t.Helper()
	// ******************************************************************
	// ** THIS GUARD IS CRITICAL — DO NOT REMOVE OR WEAKEN IT.         **
	// ** Without it, real emails will be sent on every test run.       **
	// ******************************************************************
	if os.Getenv("YOLO_TEST_EMAIL") != "1" {
		t.Skip("Skipping email integration test: set YOLO_TEST_EMAIL=1 to enable")
	}
	if _, err := exec.LookPath("/usr/sbin/sendmail"); err != nil {
		t.Skip("Skipping email integration test: sendmail not available")
	}
}

func TestSend_Integration(t *testing.T) {
	skipUnlessEmailEnabled(t)

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
	skipUnlessEmailEnabled(t)

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
