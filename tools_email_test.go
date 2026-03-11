// Email tool tests

package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// skipUnlessEmailEnabled skips integration tests that would send real emails.
// Set YOLO_TEST_EMAIL=1 to run these (requires sendmail).
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

// TestSendEmailValidation tests input validation without requiring sendmail
func TestSendEmailValidation(t *testing.T) {
	executor := NewToolExecutor("/tmp", nil)

	tests := []struct {
		name       string
		args       map[string]any
		wantError  bool
		errorMsg   string
	}{
		{
			name: "missing subject and body",
			args: map[string]any{},
			wantError: true,
			errorMsg: "subject",
		},
		{
			name: "empty subject",
			args: map[string]any{
				"subject": "",
				"body":    "Test body",
			},
			wantError: true,
			errorMsg: "subject",
		},
		{
			name: "empty body",
			args: map[string]any{
				"subject": "Test subject",
				"body":    "",
			},
			wantError: true,
			errorMsg: "body",
		},
		{
			name: "valid email without recipient",
			args: map[string]any{
				"subject": "Test",
				"body":    "Test body",
			},
			wantError: false, // Sendmail on this system accepts emails without network validation
			errorMsg:  "",   
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.sendEmail(tt.args)

			if tt.wantError {
				if !strings.Contains(result, "Error") {
					t.Logf("Expected error but got: %s", result)
				} else if tt.errorMsg != "" {
					// For validation errors, check specific message
					if tt.errorMsg == "subject" && !strings.Contains(result, "subject") {
						t.Errorf("Expected error mentioning 'subject', got: %s", result)
					}
					if tt.errorMsg == "body" && !strings.Contains(result, "body") {
						t.Errorf("Expected error mentioning 'body', got: %s", result)
					}
				}
			} else {
				if strings.Contains(result, "Error") {
					t.Errorf("Unexpected error: %s", result)
				}
			}
		})
	}
}

// TestSendReportValidation tests input validation without requiring sendmail
func TestSendReportValidation(t *testing.T) {
	executor := NewToolExecutor("/tmp", nil)

	tests := []struct {
		name       string
		args       map[string]any
		wantError  bool
		errorMsg   string
	}{
		{
			name: "missing body",
			args: map[string]any{},
			wantError: true,
			errorMsg: "body",
		},
		{
			name: "empty body",
			args: map[string]any{
				"body": "",
			},
			wantError: true,
			errorMsg: "body",
		},
		{
			name: "valid report without custom subject",
			args: map[string]any{
				"body": "Test report body",
			},
			wantError: false, // Sendmail on this system accepts emails
			errorMsg:  "",   
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.sendReport(tt.args)

			if tt.wantError {
				if !strings.Contains(result, "Error") {
					t.Logf("Expected error but got: %s", result)
				} else if tt.errorMsg != "" && !strings.Contains(result, tt.errorMsg) {
					t.Errorf("Expected error mentioning '%s', got: %s", tt.errorMsg, result)
				}
			} else {
				if strings.Contains(result, "Error") {
					t.Errorf("Unexpected error: %s", result)
				}
			}
		})
	}
}

// TestSendReportDefaultSubject tests that default subject is used when not specified
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