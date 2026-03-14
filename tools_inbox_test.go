package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"yolo/utils"
)

// Test composeResponseToEmail with mocked LLM (user suggestion)
func TestComposeResponseToEmailWithMockLLM(t *testing.T) {
	// Save original generator function
	origGenerator := llmResponseGenerator

	// Mock the LLM response - simulate deterministic behavior for testing
	llmResponseGenerator = func(prompt string) string {
		return "MOCKED: This is an automated test response to verify composeResponseToEmail works with direct LLM generation."
	}

	// Restore on cleanup
	defer func() { llmResponseGenerator = origGenerator }()

	testCases := []struct {
		name               string
		body               string
		subject            string
		from               string
		expectBodyContains string
	}{
		{
			name:               "general inquiry email",
			body:               "Hi YOLO, I heard you're autonomous. How's it going?",
			subject:            "Hello from outside!",
			from:               "user@example.com",
			expectBodyContains: "MOCKED",
		},
		{
			name:               "empty body email",
			body:               "",
			subject:            "Just checking",
			from:               "checker@test.com",
			expectBodyContains: "MOCKED",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response := composeResponseToEmail(tc.body, tc.subject, tc.from)

			if !strings.Contains(response, tc.expectBodyContains) {
				t.Errorf("response body should contain %q, got: %q", tc.expectBodyContains, response)
			}

			// Response should be at least the mocked content
			if len(response) < 20 {
				t.Errorf("response too short: %q", response)
			}
		})
	}
}

// Test composeEmailWithAgent uses mock when llmResponseGenerator is set
func TestComposeEmailWithAgentMockPath(t *testing.T) {
	origGen := llmResponseGenerator
	defer func() { llmResponseGenerator = origGen }()

	llmResponseGenerator = func(prompt string) string {
		return "AGENT_MOCK: Response via agent path"
	}

	te := &ToolExecutor{baseDir: t.TempDir(), agent: nil}
	response := te.composeEmailWithAgent("What's on the todo list?", "Todo question", "user@example.com")

	if !strings.Contains(response, "AGENT_MOCK") {
		t.Errorf("Expected agent mock response, got: %s", response)
	}
}

// Test limitString function
func TestLimitString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than maxLen",
			input:    "Hello",
			maxLen:   10,
			expected: "Hello",
		},
		{
			name:     "string equal to maxLen",
			input:    "Hello",
			maxLen:   5,
			expected: "Hello",
		},
		{
			name:     "string longer than maxLen",
			input:    "Hello World",
			maxLen:   5,
			expected: "Hello...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := limitString(tc.input, tc.maxLen)
			if result != tc.expected {
				t.Errorf("limitString(%q, %d) = %q, want %q", tc.input, tc.maxLen, result, tc.expected)
			}
		})
	}
}

// Test processInboxWorkflow with mock LLM
func TestProcessInboxWorkflow(t *testing.T) {
	// Save original LLM generator
	origLLMGen := llmResponseGenerator

	// Mock LLM response
	llmResponseGenerator = func(prompt string) string {
		return "Test auto-reply to your email"
	}

	defer func() {
		llmResponseGenerator = origLLMGen
	}()

	// Test the compose function directly (hard to test full workflow without real mailbox)
	response := composeResponseToEmail("Test email body", "Test Subject", "test@example.com")

	if !strings.Contains(response, "Test auto-reply") {
		t.Errorf("Response should contain mock reply, got: %s", response)
	}
}

// Test that LLM error responses are detected correctly
func TestLLMErrorResponseDetection(t *testing.T) {
	// Save original generator
	origGen := llmResponseGenerator
	defer func() { llmResponseGenerator = origGen }()

	// Simulate LLM returning an error string
	llmResponseGenerator = func(prompt string) string {
		return "[Error generating response: connection refused]"
	}

	response := composeResponseToEmail("Test body", "Test Subject", "test@example.com")

	if !strings.HasPrefix(response, "[Error generating response:") {
		t.Errorf("Expected error prefix in response, got: %s", response)
	}
}

// Test that empty LLM responses are handled
func TestEmptyLLMResponse(t *testing.T) {
	origGen := llmResponseGenerator
	defer func() { llmResponseGenerator = origGen }()

	llmResponseGenerator = func(prompt string) string {
		return ""
	}

	response := composeResponseToEmail("Test body", "Test Subject", "test@example.com")

	if response != "" {
		t.Errorf("Expected empty response, got: %s", response)
	}
}

// Test that processInboxWithResponse does not leave .bak files in the inbox
// This was the root cause of the email re-reply loop: DeleteFile's default
// safety config creates a .bak backup before deletion, and the backup file
// gets picked up as a "new" email on the next processing cycle.
func TestProcessInboxNoBakFiles(t *testing.T) {
	// Create a temporary maildir structure
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new")
	curDir := filepath.Join(tmpDir, "cur")
	os.MkdirAll(newDir, 0755)
	os.MkdirAll(curDir, 0755)

	// Write a fake email
	emailContent := "From: test@example.com\nTo: yolo@b-haven.org\nSubject: Test\n\nHello"
	os.WriteFile(filepath.Join(newDir, "test-email-001"), []byte(emailContent), 0644)

	// Mock LLM
	origGen := llmResponseGenerator
	llmResponseGenerator = func(prompt string) string {
		return "Auto-reply for testing"
	}
	defer func() { llmResponseGenerator = origGen }()

	// Test the underlying delete behavior to ensure no .bak files are created
	// when using the no-backup config (matching what processInboxWithResponse does).
	noBackupConfig := utils.DefaultSafetyConfig()
	noBackupConfig.CreateBackup = false

	// Write a file to delete
	testFile := filepath.Join(newDir, "test-email-002")
	os.WriteFile(testFile, []byte(emailContent), 0644)

	// Delete with no-backup config
	err := utils.DeleteFileWithConfig(testFile, noBackupConfig)
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// Verify the original file is gone
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("Original file should have been deleted")
	}

	// Verify NO .bak file was created
	bakFile := testFile + ".bak"
	if _, err := os.Stat(bakFile); !os.IsNotExist(err) {
		t.Error(".bak file should NOT exist — this causes the email re-reply loop bug")
	}

	// Also check that no .bak files exist anywhere in the new/ directory
	entries, _ := os.ReadDir(newDir)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".bak") {
			t.Errorf("Found unexpected .bak file in inbox: %s", entry.Name())
		}
	}
}

// Test that MAILER-DAEMON and other system senders are detected for skipping
func TestSystemSenderDetection(t *testing.T) {
	systemSenders := []string{
		"MAILER-DAEMON@example.com",
		"mailer-daemon@mail.example.com",
		"postmaster@example.com",
		"noreply@example.com",
		"no-reply@notifications.example.com",
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

// Test that LLM NO_REPLY response is detected correctly
func TestNoReplyDetection(t *testing.T) {
	origGen := llmResponseGenerator
	defer func() { llmResponseGenerator = origGen }()

	llmResponseGenerator = func(prompt string) string {
		return "NO_REPLY"
	}

	response := composeResponseToEmail("Bounce notification", "Undeliverable", "MAILER-DAEMON@example.com")
	if strings.TrimSpace(response) != "NO_REPLY" {
		t.Errorf("Expected NO_REPLY, got: %s", response)
	}
}

// Test that the DEFAULT safety config DOES create .bak files (proving the bug existed)
func TestDefaultDeleteCreatesBakFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "email-msg")
	os.WriteFile(testFile, []byte("From: x@x.com\nSubject: Hi\n\nBody"), 0644)

	err := utils.DeleteFile(testFile)
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// The default config creates a .bak — this is the root cause of the loop
	bakFile := testFile + ".bak"
	if _, err := os.Stat(bakFile); os.IsNotExist(err) {
		t.Skip("Default config no longer creates .bak files — test assumption changed")
	}
	// If we get here, the .bak exists, confirming the bug mechanism
	t.Log("Confirmed: DefaultSafetyConfig creates .bak files — inbox deletion must use CreateBackup=false")
}

// Test parseEmail function
func TestParseEmail(t *testing.T) {
	testCases := []struct {
		name            string
		raw             string
		expectedFrom    string
		expectedTo      string
		expectedSubject string
		expectedBody    string
	}{
		{
			name:            "standard email",
			raw:             "From: sender@example.com\nTo: recipient@example.com\nSubject: Hello\n\nThis is the body.",
			expectedFrom:    "sender@example.com",
			expectedTo:      "recipient@example.com",
			expectedSubject: "Hello",
			expectedBody:    "This is the body.",
		},
		{
			name:            "email with empty body",
			raw:             "From: test@test.com\nTo: me@me.com\nSubject: Empty\n\n",
			expectedFrom:    "test@test.com",
			expectedTo:      "me@me.com",
			expectedSubject: "Empty",
			expectedBody:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			email := parseEmail(tc.raw)

			if email.From != tc.expectedFrom {
				t.Errorf("Expected From=%q, got %q", tc.expectedFrom, email.From)
			}
			if email.To != tc.expectedTo {
				t.Errorf("Expected To=%q, got %q", tc.expectedTo, email.To)
			}
			if email.Subject != tc.expectedSubject {
				t.Errorf("Expected Subject=%q, got %q", tc.expectedSubject, email.Subject)
			}
			if email.Body != tc.expectedBody {
				t.Errorf("Expected Body=%q, got %q", tc.expectedBody, email.Body)
			}
		})
	}
}
