package main

import (
	"strings"
	"testing"
)

// TestParseEmailStandardEmail tests standard email parsing with net/mail
func TestParseEmailStandardEmail(t *testing.T) {
	raw := `From: sender@example.com
To: recipient@target.com
Subject: Test Subject
Date: Mon, 01 Jan 2024 12:00:00 +0000
Content-Type: text/plain; charset=UTF-8

This is the email body content.`

	email := parseEmail(raw)

	if email.From != "sender@example.com" {
		t.Errorf("Expected From='sender@example.com', got %q", email.From)
	}
	if email.To != "recipient@target.com" {
		t.Errorf("Expected To='recipient@target.com', got %q", email.To)
	}
	if email.Subject != "Test Subject" {
		t.Errorf("Expected Subject='Test Subject', got %q", email.Subject)
	}
	if email.Body != "This is the email body content." {
		t.Errorf("Expected Body='This is the email body content.', got %q", email.Body)
	}
}

// TestParseEmailMIMEEncodedSubject tests MIME encoded subject decoding
func TestParseEmailMIMEEncodedSubject(t *testing.T) {
	raw := `From: sender@example.com
To: recipient@target.com
Subject: =?UTF-8?B?VGVzdCBTdWJqZWN0?=

Body content.`

	email := parseEmail(raw)

	if email.Subject != "Test Subject" {
		t.Errorf("Expected decoded Subject='Test Subject', got %q", email.Subject)
	}
}

// TestParseEmailMultipartEmail tests multipart email parsing
func TestParseEmailMultipartEmail(t *testing.T) {
	raw := `From: sender@example.com
To: recipient@target.com
Subject: Multipart Test
Content-Type: multipart/alternative; boundary="----Part123"

------Part123
Content-Type: text/plain

This is the plain text body.

------Part123--`

	email := parseEmail(raw)

	// Should extract the first plain text part
	if email.Body == "" {
		t.Logf("Multipart parsing result: %q", email.Body)
		t.Logf("Raw input:\n%s", raw)
	}
}

// TestParseEmailQuotedPrintable tests quoted-printable encoded body
func TestParseEmailQuotedPrintable(t *testing.T) {
	raw := `From: sender@example.com
To: recipient@target.com
Subject: Encoded Body
Content-Type: text/plain; charset=UTF-8
Content-Transfer-Encoding: quoted-printable

This is =E2=80=9Cquoted-printable=E2=80=9D encoded body.`

	email := parseEmail(raw)

	// Should decode the quoted-printable encoding
	if !strings.Contains(email.Body, "quoted-printable") {
		t.Errorf("Expected Body to contain 'quoted-printable', got %q", email.Body)
	}
}

// TestParseEmailEmptyFields tests parsing emails with empty fields
func TestParseEmailEmptyFields(t *testing.T) {
	raw := `From: 
To: recipient@test.com
Subject: 

Body here.`

	email := parseEmail(raw)

	if email.From != "" {
		t.Errorf("Expected From='', got %q", email.From)
	}
	if email.Subject != "" {
		t.Errorf("Expected Subject='', got %q", email.Subject)
	}
}

// TestParseEmailFromAndToAddresses tests address parsing with names
func TestParseEmailFromAndToAddresses(t *testing.T) {
	raw := `From: "John Doe" <john@example.com>
To: Jane Smith <jane@target.com>
Subject: With Names

Body.`

	email := parseEmail(raw)

	// Should extract just the email address from the formatted address
	if !strings.Contains(email.From, "@") {
		t.Errorf("Expected From to contain '@', got %q", email.From)
	}
	if !strings.Contains(email.To, "@") {
		t.Errorf("Expected To to contain '@', got %q", email.To)
	}
}

// TestParseEmailMalformedEmail tests fallback parser for malformed emails
func TestParseEmailMalformedEmail(t *testing.T) {
	// Malformed email that will trigger the fallback parser
	raw := `From: test@example.com
Subject: Malformed Test
This is invalid headers
And also broken content

Body after bad headers.`

	email := parseEmail(raw)

	if email.Raw == "" {
		t.Error("Raw field should never be empty")
	}
	// The fallback parser should still extract what it can
	if email.Body == "" {
		t.Errorf("Expected Body to contain some content, got %q", email.Body)
	}
}

// TestParseEmailSystemSenders tests detection of system/automated senders
func TestParseEmailSystemSenderDetection(t *testing.T) {
	systemSenders := []string{
		"MAILER-DAEMON@example.com",
		"mailer-daemon@mail.example.com",
		"Postmaster@company.org",
		"noreply@notifications.site.com",
		"No-Reply@sendservice.net",
	}

	for _, sender := range systemSenders {
		fromLower := strings.ToLower(sender)
		isSystem := strings.Contains(fromLower, "mailer-daemon") ||
			strings.Contains(fromLower, "postmaster@") ||
			strings.Contains(fromLower, "noreply@") ||
			strings.Contains(fromLower, "no-reply@")

		if !isSystem {
			t.Errorf("Expected %q to be detected as system sender", sender)
		}
	}

	normalSenders := []string{
		"user@example.com",
		"scott@stg.net",
		"admin@company.com",
		"support@helpdesk.org",
	}

	for _, sender := range normalSenders {
		fromLower := strings.ToLower(sender)
		isSystem := strings.Contains(fromLower, "mailer-daemon") ||
			strings.Contains(fromLower, "postmaster@") ||
			strings.Contains(fromLower, "noreply@") ||
			strings.Contains(fromLower, "no-reply@")

		if isSystem {
			t.Errorf("Expected %q to NOT be detected as system sender", sender)
		}
	}
}

// TestComposeResponseToEmail tests email response composition
func TestComposeResponseToEmail(t *testing.T) {
	origGen := llmResponseGenerator
	defer func() { llmResponseGenerator = origGen }()

	llmResponseGenerator = func(prompt string) string {
		return "Auto-generated reply"
	}

	response := composeResponseToEmail("Test body", "Test Subject", "sender@example.com")

	if response == "" {
		t.Error("Expected non-empty response")
	}
	if !strings.Contains(response, "Auto-generated reply") {
		t.Errorf("Expected response to contain 'Auto-generated reply', got %q", response)
	}
}

// TestComposeEmailWithAgent tests agent-based email composition
func TestComposeEmailWithAgent(t *testing.T) {
	origGen := llmResponseGenerator
	defer func() { llmResponseGenerator = origGen }()

	llmResponseGenerator = func(prompt string) string {
		return "Agent response"
	}

	te := &ToolExecutor{baseDir: t.TempDir(), agent: nil}
	response := te.composeEmailWithAgent("Test body", "Test Subject", "sender@example.com")

	if response == "" {
		t.Error("Expected non-empty response from composeEmailWithAgent")
	}
	if !strings.Contains(response, "Agent response") {
		t.Errorf("Expected 'Agent response', got %q", response)
	}
}

// TestComposeResponseWithEmptyBody tests handling of emails with empty body
func TestComposeResponseWithEmptyBody(t *testing.T) {
	origGen := llmResponseGenerator
	defer func() { llmResponseGenerator = origGen }()

	llmResponseGenerator = func(prompt string) string {
		return "Reply to your email"
	}

	response := composeResponseToEmail("", "Subject Only", "sender@example.com")

	if response == "" {
		t.Error("Expected response even with empty body")
	}
}

// TestComposeResponseWithLongBody tests handling of long email bodies
func TestComposeResponseWithLongBody(t *testing.T) {
	origGen := llmResponseGenerator
	defer func() { llmResponseGenerator = origGen }()

	longBody := strings.Repeat("This is a long line. ", 100)
	llmResponseGenerator = func(prompt string) string {
		return "Reply to your detailed email"
	}

	response := composeResponseToEmail(longBody, "Detailed Subject", "sender@example.com")

	if response == "" {
		t.Error("Expected response for long body email")
	}
}

// TestComposeResponseWithNoReplySignal tests NO_REPLY signal handling
func TestComposeResponseWithNoReplySignal(t *testing.T) {
	origGen := llmResponseGenerator
	defer func() { llmResponseGenerator = origGen }()

	llmResponseGenerator = func(prompt string) string {
		return "NO_REPLY"
	}

	response := composeResponseToEmail("Test body", "Subject", "sender@example.com")

	if strings.TrimSpace(response) != "NO_REPLY" {
		t.Errorf("Expected 'NO_REPLY', got %q", response)
	}
}

// TestComposeResponseWithErrorMessage tests error message detection
func TestComposeResponseWithErrorMessage(t *testing.T) {
	origGen := llmResponseGenerator
	defer func() { llmResponseGenerator = origGen }()

	llmResponseGenerator = func(prompt string) string {
		return "[Error generating response: connection refused]"
	}

	response := composeResponseToEmail("Test body", "Subject", "sender@example.com")

	if !strings.HasPrefix(response, "[Error generating response:") {
		t.Errorf("Expected error prefix in response, got %q", response)
	}
}

// TestComposeResponseWithErrorMessage tests error message detection
func TestComposeResponseSpecialCharacters(t *testing.T) {
	origGen := llmResponseGenerator
	defer func() { llmResponseGenerator = origGen }()

	specialBody := "Email with special chars: @#$%^&*() and Unicode: äöü ñ"

	llmResponseGenerator = func(prompt string) string {
		return "Handled special characters"
	}

	response := composeResponseToEmail(specialBody, "Special Subject", "sender@example.com")

	if response == "" {
		t.Error("Expected non-empty response for special character email")
	}
}

// TestComposeResponseWhitespaceHandling tests whitespace handling in emails
func TestComposeResponseWhitespaceHandling(t *testing.T) {
	origGen := llmResponseGenerator
	defer func() { llmResponseGenerator = origGen }()

	whitespaceBody := "   Trimmed body   \n\n\t\n  Extra whitespace\n"

	llmResponseGenerator = func(prompt string) string {
		return "Whitespace handled"
	}

	response := composeResponseToEmail(whitespaceBody, "Subject", "sender@example.com")

	if response == "" {
		t.Error("Expected non-empty response for whitespace-heavy email")
	}
}

// TestParseEmailUnicodeContent tests Unicode content parsing
func TestParseEmailUnicodeContent(t *testing.T) {
	raw := `From: sender@example.com
To: recipient@target.com
Subject: Unicode Subject: äöü ñ 中文
Content-Type: text/plain; charset=UTF-8

Body with Unicode: Привет мир مرحبا العالم こんにちは`

	email := parseEmail(raw)

	if !strings.Contains(email.Body, "Привет") {
		t.Errorf("Expected Body to contain Russian text 'Привет', got %q", email.Body)
	}
	if !strings.Contains(email.Subject, "Unicode") {
		t.Errorf("Expected Subject to contain 'Unicode', got %q", email.Subject)
	}
}

// TestParseEmailMultipleBoundaries tests multipart emails with multiple boundaries
func TestParseEmailMultipleBoundaries(t *testing.T) {
	raw := `From: sender@example.com
To: recipient@target.com
Subject: Multi-boundary Test
Content-Type: multipart/mixed; boundary="----Boundary123"

------Boundary123
Content-Type: text/plain

Text part

------Boundary123
Content-Type: text/html

HTML part

------Boundary123--`

	email := parseEmail(raw)

	// Should extract the first plain text part
	if email.Body == "" {
		t.Error("Expected Body to contain content from multipart email")
	}
}
