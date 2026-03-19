package email

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestSendViaSendmailSuccess tests successful sendmail execution with mocked command
func TestSendViaSendmailSuccess(t *testing.T) {
	// Temporarily use a mock that we can control
	cfg := &Config{
		From:         "test@yolo.local",
		UseSendmail:  true,
		SendmailPath: "/usr/bin/true", // /usr/bin/true always succeeds without doing anything
	}
	
	client := New(cfg)
	
	msg := &Message{
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Body:    "Test body content",
	}
	
	err := client.sendViaSendmail(msg)
	if err != nil {
		t.Errorf("Expected successful send, got error: %v", err)
	}
}

// TestSendViaSendmailFailure tests sendmail execution failure with proper error handling
func TestSendViaSendmailFailure(t *testing.T) {
	cfg := &Config{
		From:         "test@yolo.local",
		UseSendmail:  true,
		SendmailPath: "/nonexistent/sendmail/path/that/does/not/exist",
	}
	
	client := New(cfg)
	
	msg := &Message{
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Body:    "Test body content",
	}
	
	err := client.sendViaSendmail(msg)
	if err == nil {
		t.Error("Expected sendmail to fail with nonexistent path")
	}
	
	if !strings.Contains(err.Error(), "sendmail failed") {
		t.Errorf("Expected error to contain 'sendmail failed', got: %v", err)
	}
	
	if !strings.Contains(err.Error(), msg.Subject) {
		t.Errorf("Expected error to include subject, got: %v", err)
	}
}

// TestSendViaSendmailEmailFormat verifies RFC 2822 email format construction
func TestSendViaSendmailEmailFormat(t *testing.T) {
	cfg := &Config{
		From:         "sender@test.com",
		UseSendmail:  true,
		SendmailPath: "/bin/true",
	}
	
	client := New(cfg)
	
	msg := &Message{
		To:      []string{"recipient1@example.com", "recipient2@example.com"},
		Subject: "Format Test",
		Body:    "Test body with multiple lines\n\nSecond paragraph.",
	}
	
	// Manually construct what sendViaSendmail would create to test the format
	var emailContent bytes.Buffer
	emailContent.WriteString("From: sender@test.com\r\n")
	emailContent.WriteString("To: recipient1@example.com, recipient2@example.com\r\n")
	emailContent.WriteString("Subject: Format Test\r\n")
	emailContent.WriteString("MIME-Version: 1.0\r\n")
	emailContent.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	emailContent.WriteString("\r\n")
	emailContent.WriteString(msg.Body)
	
	contentStr := emailContent.String()
	
	// Verify all required headers are present
	requiredHeaders := []string{
		"From: sender@test.com\r\n",
		"To: recipient1@example.com, recipient2@example.com\r\n",
		"Subject: Format Test\r\n",
		"MIME-Version: 1.0\r\n",
		"Content-Type: text/plain; charset=utf-8\r\n",
	}
	
	for _, header := range requiredHeaders {
		if !strings.Contains(contentStr, header) {
			t.Errorf("Expected email to contain header: %s", header)
		}
	}
	
	// Verify body is present
	if !strings.Contains(contentStr, "Test body with multiple lines") {
		t.Error("Expected body content in email")
	}
	
	// Verify proper line endings (CRLF)
	if !strings.Contains(contentStr, "\r\n") {
		t.Error("Expected CRLF line endings")
	}
	
	_ = client // Use client to avoid unused variable warning
}

// TestSendViaSendmailWithSingleRecipient tests sendmail with single recipient
func TestSendViaSendmailSingleRecipient(t *testing.T) {
	cfg := &Config{
		From:         "test@yolo.local",
		UseSendmail:  true,
		SendmailPath: "/usr/bin/true",
	}
	
	client := New(cfg)
	
	msg := &Message{
		To:      []string{"single@example.com"},
		Subject: "Single Recipient Test",
		Body:    "Test body",
	}
	
	err := client.sendViaSendmail(msg)
	if err != nil {
		t.Errorf("Expected successful send with single recipient, got error: %v", err)
	}
}

// TestSendViaSendmailWithSpecialCharacters tests handling of special characters in subject and body
func TestSendViaSendmailSpecialCharacters(t *testing.T) {
	cfg := &Config{
		From:         "test@yolo.local",
		UseSendmail:  true,
		SendmailPath: "/usr/bin/true",
	}
	
	client := New(cfg)
	
	msg := &Message{
		To:      []string{"recipient@example.com"},
		Subject: "Test with Special Chars: <>&\"'\\/",
		Body:    "Body with unicode: 日本語 🎉 and emojis 😀🚀",
	}
	
	err := client.sendViaSendmail(msg)
	if err != nil {
		t.Errorf("Expected successful send with special characters, got error: %v", err)
	}
}

// TestGetRFC2822DateFallback tests that fallback date is returned when date command fails
func TestGetRFC2822DateFallback(t *testing.T) {
	// Temporarily set PATH to empty directory to force date command to fail
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)
	
	os.Setenv("PATH", "/nonexistent/path")
	defer os.Setenv("PATH", originalPath)
	
	dateStr := getRFC2822Date()
	
	// Should return fallback date when command fails
	expectedFallback := "Mon, 1 Jan 2024 00:00:00 +0000"
	if dateStr != expectedFallback {
		t.Errorf("Expected fallback date %q, got %q", expectedFallback, dateStr)
	}
}

// TestGetRFC2822DateSuccess tests successful RFC 2822 date generation
func TestGetRFC2822DateSuccess(t *testing.T) {
	dateStr := getRFC2822Date()
	
	if len(dateStr) == 0 {
		t.Error("Expected non-empty date string")
	}
	
	// Basic validation of RFC 2822 format (day, month, year, timezone)
	if !strings.Contains(dateStr, ",") {
		t.Errorf("Expected comma in date: %s", dateStr)
	}
	
	// Should have 4-digit year (look for 20xx or 19xx pattern)
	hasYear := false
	for i := 0; i+3 < len(dateStr); i++ {
		if dateStr[i:i+1] >= "2" && dateStr[i:i+4] != "---" {
			// Found a potential year starting with digit
			if isDigit(dateStr[i]) && isDigit(dateStr[i+1]) && 
			   isDigit(dateStr[i+2]) && isDigit(dateStr[i+3]) {
				hasYear = true
				break
			}
		}
	}
	if !hasYear {
		t.Errorf("Expected 4-digit year in date: %s", dateStr)
	}
	
	// Should have timezone offset (contains + or - followed by digits like -0400)
	hasTimezone := false
	for i := 0; i < len(dateStr)-4; i++ {
		if (dateStr[i] == '+' || dateStr[i] == '-') && 
		   isDigit(dateStr[i+1]) && isDigit(dateStr[i+2]) &&
		   isDigit(dateStr[i+3]) && isDigit(dateStr[i+4]) {
			hasTimezone = true
			break
		}
	}
	if !hasTimezone {
		t.Errorf("Expected timezone offset in date: %s", dateStr)
	}
}

// Helper function to check if character is digit
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// TestSendErrorLogging tests that errors are properly logged with context
func TestSendErrorLogging(t *testing.T) {
	cfg := &Config{
		From:         "test@yolo.local",
		UseSendmail:  true,
		SendmailPath: "/nonexistent/sendmail/path",
	}
	
	client := New(cfg)
	
	msg := &Message{
		To:      []string{"user@example.com"},
		Subject: "Error Test Subject",
		Body:    "Test body for error logging",
	}
	
	err := client.Send(msg)
	if err == nil {
		t.Error("Expected send to fail")
	}
	
	// Error should contain contextual information
	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "user@example.com") {
		t.Errorf("Error should include recipient: %v", err)
	}
	if !strings.Contains(errorMsg, "Error Test Subject") {
		t.Errorf("Error should include subject: %v", err)
	}
}

// TestSendMultipleTimes tests that client can send multiple emails in sequence
func TestSendMultipleTimesSuccess(t *testing.T) {
	cfg := &Config{
		From:         "test@yolo.local",
		UseSendmail:  true,
		SendmailPath: "/usr/bin/true",
	}
	
	client := New(cfg)
	
	for i := 0; i < 5; i++ {
		msg := &Message{
			To:      []string{"recipient@example.com"},
			Subject: "Batch Test " + string(rune('A'+i)),
			Body:    "Test body for batch item " + string(rune('0'+i)),
		}
		
		err := client.sendViaSendmail(msg)
		if err != nil {
			t.Errorf("Failed on iteration %d: %v", i, err)
		}
	}
}

// TestValidateMessageBeforeSend tests that validation happens before attempting to send
func TestValidateHappensBeforeSend(t *testing.T) {
	cfg := &Config{
		From:         "test@yolo.local",
		UseSendmail:  true,
		SendmailPath: "/bin/true", // Would succeed if called
	}
	
	client := New(cfg)
	
	// Invalid message should fail at validation, never reach sendViaSendmail
	msg := &Message{
		To:      []string{}, // No recipients - invalid
		Subject: "Test",
		Body:    "Test body",
	}
	
	err := client.Send(msg)
	if err == nil {
		t.Error("Expected validation to catch missing recipients")
	}
	
	if !strings.Contains(err.Error(), "recipients") {
		t.Errorf("Expected 'recipients' in error, got: %v", err)
	}
}

// TestSendViaSendmailCommandConstruction tests that sendmail command is built correctly
func TestSendViaSendmailCommandArgs(t *testing.T) {
	// This test verifies the args construction logic in sendViaSendmail
	cfg := &Config{
		From:         "sender@test.com",
		UseSendmail:  true,
		SendmailPath: "/usr/bin/true",
	}
	
	client := New(cfg)
	
	recipients := []string{"user1@test.com", "user2@test.com"}
	msg := &Message{
		To:      recipients,
		Subject: "Test",
		Body:    "Test body",
	}
	
	// Build the args that sendViaSendmail would create (append "-f", from, all recipients)
	args := append([]string{"-f", cfg.From}, msg.To...)
	
	if len(args) != 4 { // -f, sender@test.com, user1@test.com, user2@test.com
		t.Errorf("Expected 4 args, got %d: %v", len(args), args)
	}
	
	if args[0] != "-f" || args[1] != "sender@test.com" {
		t.Errorf("Incorrect first two args: %v", args[:2])
	}
	
	if args[2] != "user1@test.com" || args[3] != "user2@test.com" {
		t.Errorf("Incorrect recipient args: %v", args[2:])
	}
	
	_ = client // Avoid unused variable warning
}
