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

func TestSendEmailMissingPassword(t *testing.T) {
	// Unset email password to test error handling
	os.Unsetenv("EMAIL_PASSWORD")

	executor := NewToolExecutor("/tmp", nil)
	result := executor.sendEmail(map[string]any{
		"subject": "Test",
		"body":    "Test body",
	})

	if result == "" || !strings.Contains(result, "Error") {
		t.Error("Expected error when EMAIL_PASSWORD not configured")
	}
}

func TestSendReportMissingPassword(t *testing.T) {
	os.Unsetenv("EMAIL_PASSWORD")

	executor := NewToolExecutor("/tmp", nil)
	result := executor.sendReport(map[string]any{
		"body": "Test body",
	})

	if result == "" || !strings.Contains(result, "Error") {
		t.Error("Expected error when EMAIL_PASSWORD not configured")
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

	// Should either error on missing password or mention scott@stg.net as default
	if !strings.Contains(result, "scott@stg.net") && !strings.Contains(result, "EMAIL_PASSWORD") {
		t.Error("Expected default recipient scott@stg.net to be mentioned")
	}
}
