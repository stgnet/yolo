// Email tool tests

package main

import (
	"os"
	"strings"
	"testing"
)

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
	// Unset email password (no longer needed with sendmail)
	os.Unsetenv("EMAIL_PASSWORD")

	executor := NewToolExecutor("/tmp", nil)
	result := executor.sendEmail(map[string]any{
		"subject": "YOLO Test Email",
		"body":    "This is a test email from YOLO.",
		"to":      "scott@stg.net",
	})

	// Should succeed with sendmail/postfix configured
	if result == "" || !strings.Contains(result, "Email sent successfully") {
		t.Logf("Result: %s", result)
		t.Error("Expected email to be sent successfully via sendmail")
	}
}

func TestSendReportIntegration(t *testing.T) {
	os.Unsetenv("EMAIL_PASSWORD")

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
	// This test verifies that the default recipient is scott@stg.net
	executor := NewToolExecutor("/tmp", nil)

	result := executor.sendEmail(map[string]any{
		"subject": "Test",
		"body":    "Test body",
	})

	// Should succeed and use default recipient scott@stg.net
	if !strings.Contains(result, "scott@stg.net") || !strings.Contains(result, "Email sent successfully") {
		t.Logf("Result: %s", result)
		t.Error("Expected default recipient scott@stg.net to be used and email to send successfully")
	}
}
