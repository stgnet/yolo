// Package tools provides tests for communication tools
package tools

import (
	"context"
	"testing"
)

// TestSendEmailToolMissingSubject tests that missing subject returns an error
func TestSendEmailToolMissingSubject(t *testing.T) {
	tool := &SendEmailTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"body": "Test email body",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when subject is missing")
	}
	
	if result.Error != "subject is required" {
		t.Errorf("Expected 'subject is required' error, got '%s'", result.Error)
	}
}

// TestSendEmailToolMissingBody tests that missing body returns an error
func TestSendEmailToolMissingBody(t *testing.T) {
	tool := &SendEmailTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"subject": "Test Subject",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when body is missing")
	}
	
	if result.Error != "body is required" {
		t.Errorf("Expected 'body is required' error, got '%s'", result.Error)
	}
}

// TestSendEmailToolInvalidSubjectType tests that wrong subject type returns an error
func TestSendEmailToolInvalidSubjectType(t *testing.T) {
	tool := &SendEmailTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"subject": 123, // Wrong type
		"body":    "Test body",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when subject has wrong type")
	}
	
	if result.Error != "subject is required" {
		t.Errorf("Expected 'subject is required' error, got '%s'", result.Error)
	}
}

// TestSendEmailToolEmptySubject tests that empty subject returns an error
func TestSendEmailToolEmptySubject(t *testing.T) {
	tool := &SendEmailTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"subject": "",
		"body":    "Test body",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when subject is empty")
	}
	
	if result.Error != "subject is required" {
		t.Errorf("Expected 'subject is required' error, got '%s'", result.Error)
	}
}

// TestSendEmailToolEmptyBody tests that empty body returns an error
func TestSendEmailToolEmptyBody(t *testing.T) {
	tool := &SendEmailTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"subject": "Test Subject",
		"body":    "",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when body is empty")
	}
	
	if result.Error != "body is required" {
		t.Errorf("Expected 'body is required' error, got '%s'", result.Error)
	}
}

// TestSendEmailToolDefaultRecipient tests that default recipient is used when not provided
func TestSendEmailToolDefaultRecipient(t *testing.T) {
	tool := &SendEmailTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"subject": "Test Subject",
		"body":    "Test body",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	// The function will fail to send (no actual mail server), but it should have tried
	// with the default recipient, so Success will be false due to send failure
	// We're mainly checking that the tool is functional and doesn't crash
	if result.Error == "" {
		t.Log("Note: Email was sent successfully (may indicate test environment has mail)")
	}
}

// TestSendReportToolMissingBody tests that missing body returns an error
func TestSendReportToolMissingBody(t *testing.T) {
	tool := &SendReportTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when body is missing")
	}
	
	if result.Error != "body is required" {
		t.Errorf("Expected 'body is required' error, got '%s'", result.Error)
	}
}

// TestSendReportToolInvalidBodyType tests that wrong body type returns an error
func TestSendReportToolInvalidBodyType(t *testing.T) {
	tool := &SendReportTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"body": 123, // Wrong type
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when body has wrong type")
	}
	
	if result.Error != "body is required" {
		t.Errorf("Expected 'body is required' error, got '%s'", result.Error)
	}
}

// TestSendReportToolEmptyBody tests that empty body returns an error
func TestSendReportToolEmptyBody(t *testing.T) {
	tool := &SendReportTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"body": "",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when body is empty")
	}
	
	if result.Error != "body is required" {
		t.Errorf("Expected 'body is required' error, got '%s'", result.Error)
	}
}

// TestSendReportToolDefaultSubject tests that default subject is used when not provided
func TestSendReportToolDefaultSubject(t *testing.T) {
	tool := &SendReportTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"body": "Test report body",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	// The function will fail to send (no actual mail server), but it should have tried
	// with the default subject, so Success will be false due to send failure
	if result.Error == "" {
		t.Log("Note: Report was sent successfully (may indicate test environment has mail)")
	}
}

// TestCheckInboxTool tests basic execution
func TestCheckInboxTool(t *testing.T) {
	tool := &CheckInboxTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	// The function may fail if the Maildir doesn't exist in test environment
	// but it should not crash
	if result.Error != "" {
		t.Logf("Expected behavior in test env: %s", result.Error)
	}
}

// TestCheckInboxToolWithMarkRead tests with mark_read option
func TestCheckInboxToolWithMarkRead(t *testing.T) {
	tool := &CheckInboxTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"mark_read": true,
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	// Should execute without crashing
	if result.Error == "" {
		t.Logf("Found emails: %s", result.Output[:min(len(result.Output), 100)])
	}
}

// TestProcessInboxTool tests basic execution
func TestProcessInboxTool(t *testing.T) {
	tool := &ProcessInboxTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	// The function may fail if there are no emails to process
	// but it should not crash
	if result.Error != "" {
		t.Logf("Expected behavior in test env: %s", result.Error)
	}
}

// TestGOGToolMissingCommand tests that missing command returns an error
func TestGOGToolMissingCommand(t *testing.T) {
	tool := &GOGTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when command is missing")
	}
	
	if result.Error != "command is required" {
		t.Errorf("Expected 'command is required' error, got '%s'", result.Error)
	}
}

// TestGOGToolInvalidCommandType tests that wrong command type returns an error
func TestGOGToolInvalidCommandType(t *testing.T) {
	tool := &GOGTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": 123, // Wrong type
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when command has wrong type")
	}
	
	if result.Error != "command is required" {
		t.Errorf("Expected 'command is required' error, got '%s'", result.Error)
	}
}

// TestGOGToolEmptyCommand tests that empty command returns an error
func TestGOGToolEmptyCommand(t *testing.T) {
	tool := &GOGTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when command is empty")
	}
	
	if result.Error != "command is required" {
		t.Errorf("Expected 'command is required' error, got '%s'", result.Error)
	}
}

// TestGOGToolWithCommand tests that a valid command executes without crash
func TestGOGToolWithCommand(t *testing.T) {
	tool := &GOGTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "help",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	// The function may fail if gog is not installed or configured
	// but it should not crash
	if result.Error != "" {
		t.Logf("Expected behavior in test env: %s", result.Error)
	}
}

// TestLearnTool tests basic execution
func TestLearnTool(t *testing.T) {
	tool := &LearnTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	// The function may fail if web searches are not available
	// but it should not crash
	if result.Error != "" {
		t.Logf("Expected behavior in test env: %s", result.Error)
	}
}

// TestImplementTool tests basic execution
func TestImplementTool(t *testing.T) {
	tool := &ImplementTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	// The function may fail if there are no improvements to implement
	// but it should not crash
	if result.Error != "" {
		t.Logf("Expected behavior in test env: %s", result.Error)
	}
}
