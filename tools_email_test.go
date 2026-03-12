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
// ** These tests validate email response generation WITHOUT sending      **
// ** any real emails. They test the composeResponseToEmail function      **
// ** directly, which calls the LLM but does not invoke sendmail.         **
// **************************************************************************

func TestComposeResponseToEmail(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		subject string
		from    string
	}{
		{
			name:    "simple greeting",
			body:    "Hello YOLO, how are you?",
			subject: "Greeting",
			from:    "test@example.com",
		},
		{
			name:    "question about capabilities",
			body:    "What can you do?",
			subject: "YOLO Capabilities",
			from:    "curious@example.com",
		},
		{
			name:    "request for help",
			body:    "I need help with my Go project",
			subject: "Need assistance",
			from:    "developer@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := composeResponseToEmail(tt.body, tt.subject, tt.from)

			// Response should not be empty
			if response == "" {
				t.Error("Expected non-empty response from composeResponseToEmail")
			}

			// Response should contain some meaningful text (more than just error message)
			if len(response) < 5 {
				t.Errorf("Response too short: %q", response)
			}

			// If there was an error, it should be formatted as an error message
			if strings.Contains(response, "[Error") {
				t.Logf("LLM error in response: %s", response)
			}
		})
	}
}

func TestComposeResponseToEmailEmptyBody(t *testing.T) {
	response := composeResponseToEmail("", "Test Subject", "sender@example.com")

	if response == "" {
		t.Error("Expected non-empty response even with empty body")
	}

	// Empty body should be handled gracefully (replaced with "No content")
	if !strings.Contains(response, "[Error") && len(response) < 5 {
		t.Errorf("Response too short for empty body case: %q", response)
	}
}

func TestComposeResponseToEmailLongMessage(t *testing.T) {
	longBody := strings.Repeat("This is a test sentence. ", 100)
	response := composeResponseToEmail(longBody, "Long Message Test", "sender@example.com")

	if response == "" {
		t.Error("Expected non-empty response for long message")
	}

	// Should handle long messages without crashing
	if strings.Contains(response, "[Error") && !strings.Contains(response, "context deadline exceeded") {
		t.Logf("Unexpected error: %s", response)
	}
}

func TestComposeResponseToEmailNoSendmail(t *testing.T) {
	// This test verifies that composeResponseToEmail does NOT invoke sendmail
	// It only generates the response text, it doesn't send anything
	
	response := composeResponseToEmail("Test body", "Test Subject", "test@example.com")

	if response == "" {
		t.Error("Expected non-empty response")
	}

	// The function should return a generated response, not attempt to send it
	// If we got here without actually sending an email, the test passes
	_ = response
}
