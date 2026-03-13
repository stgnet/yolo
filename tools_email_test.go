// Email tool tests
//
// **************************************************************************
// ** WARNING: SENDING REAL EMAILS IN TESTS IS STRICTLY FORBIDDEN.         **
// ** All tests in this file MUST validate logic WITHOUT invoking sendmail  **
// ** or any real email transport. Tests that send actual emails belong     **
// ** ONLY in integration tests gated behind YOLO_TEST_EMAIL=1.            **
// ** DO NOT add test cases that call sendEmail() or sendReport() with     **
// ** valid arguments that would reach the sendmail binary.                 **
// **************************************************************************

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// skipUnlessEmailEnabled skips integration tests that would send real emails.
// Set YOLO_TEST_EMAIL=1 to run these (requires sendmail).
//
// **************************************************************************
// ** ANY TEST THAT COULD SEND A REAL EMAIL MUST CALL THIS FUNCTION.       **
// ** NEVER SEND REAL EMAILS IN UNIT TESTS.                                **
// **************************************************************************
func skipUnlessEmailEnabled(t *testing.T) {
	t.Helper()
	if os.Getenv("YOLO_TEST_EMAIL") != "1" {
		t.Skip("Skipping email integration test: set YOLO_TEST_EMAIL=1 to enable")
	}
	if _, err := exec.LookPath("/usr/sbin/sendmail"); err != nil {
		t.Skip("Skipping email integration test: sendmail not available")
	}
}

func TestSendEmailToolDefinition(t *testing.T) {
	found := false
	for _, tool := range ollamaTools {
		if tool.Function.Name == "send_email" {
			found = true
			if tool.Function.Description == "" {
				t.Error("send_email tool missing description")
			}
			if len(tool.Function.Parameters.Properties) != 3 {
				t.Errorf("send_email should have 3 parameters, got %d", len(tool.Function.Parameters.Properties))
			}
			expectedParams := []string{"to", "subject", "body"}
			for _, param := range expectedParams {
				if _, ok := tool.Function.Parameters.Properties[param]; !ok {
					t.Errorf("send_email missing parameter: %s", param)
				}
			}
			break
		}
	}
	if !found {
		t.Error("send_email tool not found in ollamaTools")
	}
}

func TestSendReportToolDefinition(t *testing.T) {
	found := false
	for _, tool := range ollamaTools {
		if tool.Function.Name == "send_report" {
			found = true
			if tool.Function.Description == "" {
				t.Error("send_report tool missing description")
			}
			if len(tool.Function.Parameters.Properties) != 2 {
				t.Errorf("send_report should have 2 parameters, got %d", len(tool.Function.Parameters.Properties))
			}
			expectedParams := []string{"subject", "body"}
			for _, param := range expectedParams {
				if _, ok := tool.Function.Parameters.Properties[param]; !ok {
					t.Errorf("send_report missing parameter: %s", param)
				}
			}
			break
		}
	}
	if !found {
		t.Error("send_report tool not found in ollamaTools")
	}
}

// **************************************************************************
// ** INTEGRATION TESTS BELOW — GATED BEHIND YOLO_TEST_EMAIL=1            **
// ** These are the ONLY tests allowed to send real emails.                **
// **************************************************************************

func TestSendEmailIntegration(t *testing.T) {
	skipUnlessEmailEnabled(t)

	executor := NewToolExecutor("/tmp", nil)
	result := executor.sendEmail(map[string]any{
		"subject": "YOLO Test Email",
		"body":    "This is a test email from YOLO.",
		"to":      "scott@stg.net",
	})

	if result == "" || !strings.Contains(result, "Email sent successfully") {
		t.Logf("Result: %s", result)
		t.Error("Expected email to be sent successfully via sendmail")
	}
}

func TestSendReportIntegration(t *testing.T) {
	skipUnlessEmailEnabled(t)

	executor := NewToolExecutor("/tmp", nil)
	result := executor.sendReport(map[string]any{
		"body": "This is a test progress report from YOLO.",
	})

	if result == "" || !strings.Contains(result, "Progress report sent successfully") {
		t.Logf("Result: %s", result)
		t.Error("Expected report to be sent successfully via sendmail")
	}
}

func TestSendEmailDefaultRecipient(t *testing.T) {
	skipUnlessEmailEnabled(t)

	executor := NewToolExecutor("/tmp", nil)

	result := executor.sendEmail(map[string]any{
		"subject": "Test",
		"body":    "Test body",
	})

	if !strings.Contains(result, "scott@stg.net") || !strings.Contains(result, "Email sent successfully") {
		t.Logf("Result: %s", result)
		t.Error("Expected default recipient scott@stg.net to be used and email to send successfully")
	}
}

func TestSendReportDefaultSubject(t *testing.T) {
	skipUnlessEmailEnabled(t)

	executor := NewToolExecutor("/tmp", nil)
	result := executor.sendReport(map[string]any{
		"body": "Test report",
	})

	if !strings.Contains(result, "YOLO Progress Report") || !strings.Contains(result, "Progress report sent successfully") {
		t.Logf("Result: %s", result)
		t.Error("Expected default subject 'YOLO Progress Report' to be used")
	}
}

// **************************************************************************
// ** UNIT TESTS BELOW — THESE MUST NEVER SEND REAL EMAILS.               **
// ** Only test input validation and error paths here.                     **
// **************************************************************************

func TestSendEmailMissingRequiredFields(t *testing.T) {
	executor := NewToolExecutor("/tmp", nil)

	// Missing subject
	result := executor.sendEmail(map[string]any{
		"body": "Test body",
	})
	if !strings.Contains(result, "subject") && !strings.Contains(result, "Error") {
		t.Error("Expected error for missing subject")
	}

	// Missing body
	result = executor.sendEmail(map[string]any{
		"subject": "Test",
	})
	if !strings.Contains(result, "body") && !strings.Contains(result, "Error") {
		t.Error("Expected error for missing body")
	}
}

func TestSendReportMissingBody(t *testing.T) {
	executor := NewToolExecutor("/tmp", nil)
	result := executor.sendReport(map[string]any{})

	if !strings.Contains(result, "body") && !strings.Contains(result, "Error") {
		t.Error("Expected error for missing body in sendReport")
	}
}

// TestSendEmailValidation tests input validation WITHOUT sending any emails.
//
// **************************************************************************
// ** DO NOT ADD TEST CASES WITH VALID SUBJECT+BODY — THEY WILL SEND      **
// ** REAL EMAILS VIA SENDMAIL. ONLY TEST ERROR/VALIDATION PATHS HERE.     **
// **************************************************************************
func TestSendEmailValidation(t *testing.T) {
	executor := NewToolExecutor("/tmp", nil)

	tests := []struct {
		name     string
		args     map[string]any
		errorMsg string
	}{
		{
			name:     "missing subject and body",
			args:     map[string]any{},
			errorMsg: "subject",
		},
		{
			name: "empty subject",
			args: map[string]any{
				"subject": "",
				"body":    "Test body",
			},
			errorMsg: "subject",
		},
		{
			name: "empty body",
			args: map[string]any{
				"subject": "Test subject",
				"body":    "",
			},
			errorMsg: "body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.sendEmail(tt.args)

			if !strings.Contains(result, "Error") {
				t.Errorf("Expected error but got: %s", result)
			}
			if tt.errorMsg != "" && !strings.Contains(result, tt.errorMsg) {
				t.Errorf("Expected error mentioning %q, got: %s", tt.errorMsg, result)
			}
		})
	}
}

// TestSendReportValidation tests input validation WITHOUT sending any emails.
//
// **************************************************************************
// ** DO NOT ADD TEST CASES WITH A VALID BODY — THEY WILL SEND REAL       **
// ** EMAILS VIA SENDMAIL. ONLY TEST ERROR/VALIDATION PATHS HERE.          **
// **************************************************************************
func TestSendReportValidation(t *testing.T) {
	executor := NewToolExecutor("/tmp", nil)

	tests := []struct {
		name     string
		args     map[string]any
		errorMsg string
	}{
		{
			name:     "missing body",
			args:     map[string]any{},
			errorMsg: "body",
		},
		{
			name: "empty body",
			args: map[string]any{
				"body": "",
			},
			errorMsg: "body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.sendReport(tt.args)

			if !strings.Contains(result, "Error") {
				t.Errorf("Expected error but got: %s", result)
			}
			if tt.errorMsg != "" && !strings.Contains(result, tt.errorMsg) {
				t.Errorf("Expected error mentioning %q, got: %s", tt.errorMsg, result)
			}
		})
	}
}

// **************************************************************************
// ** UNIT TESTS FOR composeResponseToEmail                              **
// ** These tests validate email response generation WITHOUT relying on    **
// ** external Ollama service. They use mocked LLM responses to test the   **
// ** email handling logic comprehensively.                               **
// **************************************************************************

func TestComposeResponseToEmailWithMock(t *testing.T) {
	// Save original generator and restore after test
	originalGenerator := llmResponseGenerator
	defer func() { llmResponseGenerator = originalGenerator }()

	// Set up mock generator
	callCount := 0
	llmResponseGenerator = func(prompt string) string {
		callCount++
		// Verify the prompt contains expected data
		if !strings.Contains(prompt, "test@example.com") {
			t.Error("Prompt should contain sender email")
		}
		if !strings.Contains(prompt, "Greeting") {
			t.Error("Prompt should contain subject")
		}
		if !strings.Contains(prompt, "Hello YOLO") {
			t.Error("Prompt should contain body content")
		}
		return "Hi! I'm doing great, thanks for asking. How can I help you today?"
	}

	response := composeResponseToEmail("Hello YOLO, how are you?", "Greeting", "test@example.com")

	if callCount != 1 {
		t.Errorf("Expected llmResponseGenerator to be called once, got %d calls", callCount)
	}

	if response == "" {
		t.Error("Expected non-empty response")
	}

	expected := "Hi! I'm doing great, thanks for asking. How can I help you today?"
	if response != expected {
		t.Errorf("Expected %q, got %q", expected, response)
	}
}

func TestComposeResponseToEmailEmptyBody(t *testing.T) {
	// Save original generator and restore after test
	originalGenerator := llmResponseGenerator
	defer func() { llmResponseGenerator = originalGenerator }()

	// Set up mock generator that tracks the prompt
	var capturedPrompt string
	llmResponseGenerator = func(prompt string) string {
		capturedPrompt = prompt
		return "Thank you for your email."
	}

	response := composeResponseToEmail("", "Test Subject", "sender@example.com")

	if response == "" {
		t.Error("Expected non-empty response even with empty body")
	}

	// Verify empty body is replaced with "No content"
	if !strings.Contains(capturedPrompt, "No content") {
		t.Error("Empty body should be replaced with 'No content' in prompt")
	}
}

func TestComposeResponseToEmailLongMessage(t *testing.T) {
	// Save original generator and restore after test
	originalGenerator := llmResponseGenerator
	defer func() { llmResponseGenerator = originalGenerator }()

	longBody := strings.Repeat("This is a test sentence. ", 100)

	llmResponseGenerator = func(prompt string) string {
		// Verify the long message is included in prompt
		if !strings.Contains(prompt, longBody) {
			t.Error("Long body should be included in prompt")
		}
		return "I've received your detailed message."
	}

	response := composeResponseToEmail(longBody, "Long Message Test", "sender@example.com")

	if response == "" {
		t.Error("Expected non-empty response for long message")
	}
}

func TestComposeResponseToEmailNoSendmail(t *testing.T) {
	// This test verifies that composeResponseToEmail does NOT invoke sendmail
	// It only generates the response text, it doesn't send anything

	// Save original generator and restore after test
	originalGenerator := llmResponseGenerator
	defer func() { llmResponseGenerator = originalGenerator }()

	llmResponseGenerator = func(prompt string) string {
		return "This is a mock response"
	}

	response := composeResponseToEmail("Test body", "Test Subject", "test@example.com")

	if response == "" {
		t.Error("Expected non-empty response")
	}

	// The function should return a generated response, not attempt to send it
	// If we got here without actually sending an email, the test passes
	if response != "This is a mock response" {
		t.Errorf("Expected mock response, got %q", response)
	}
}

func TestComposeResponseToEmailTrimWhitespace(t *testing.T) {
	// Save original generator and restore after test
	originalGenerator := llmResponseGenerator
	defer func() { llmResponseGenerator = originalGenerator }()

	// Mock returns response with leading/trailing whitespace
	llmResponseGenerator = func(prompt string) string {
		return "  \n\nThis is the response.\n\n  "
	}

	response := composeResponseToEmail("Test", "Subject", "test@example.com")

	if strings.HasPrefix(response, " ") || strings.HasSuffix(response, " ") {
		t.Errorf("Response should have whitespace trimmed, got %q", response)
	}

	if strings.HasPrefix(response, "\n") || strings.HasSuffix(response, "\n") {
		t.Errorf("Response should have newlines trimmed, got %q", response)
	}
}

func TestComposeResponseToEmailPromptStructure(t *testing.T) {
	// Save original generator and restore after test
	originalGenerator := llmResponseGenerator
	defer func() { llmResponseGenerator = originalGenerator }()

	// Capture the full prompt to verify structure
	var capturedPrompt string
	llmResponseGenerator = func(prompt string) string {
		capturedPrompt = prompt
		return "Mock response"
	}

	body := "I have a question about your capabilities"
	subject := "YOLO Questions"
	from := "curious@user.com"

	composeResponseToEmail(body, subject, from)

	// Verify prompt contains all required sections (updated for improved prompt structure)
	expectedElements := []string{
		"You are YOLO, an autonomous AI assistant running on a Mac",
		"INCOMING EMAIL CONTEXT:",
		fmt.Sprintf("Sender: %s", from),
		fmt.Sprintf("Subject: %s", subject),
		"THREAD/TOPIC BEING DISCUSSED:",
		subject,
		"EMAIL BODY CONTENT:",
		body,
		"RESPONSE GUIDELINES:",
		"ACKNOWLEDGE THE SENDER",
		"REFERENCE THE ORIGINAL SUBJECT",
		"INCLUDE EMAIL METADATA",
		"BE PROFESSIONAL YET CONVERSATIONAL",
		"ANSWER SPECIFICALLY",
		"PROVIDE CONTEXT AWARENESS",
		"NO PLACEHOLDERS",
		"RESPONSE FORMAT:",
		"Write your email response now:",
	}

	for _, element := range expectedElements {
		if !strings.Contains(capturedPrompt, element) {
			t.Errorf("Prompt should contain %q, got:\n%s", element, capturedPrompt)
		}
	}
}
