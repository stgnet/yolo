// Email Tool Implementation with Security Hardening
// Allows YOLO to send emails via SMTP from yolo@b-haven.org
// Includes protection against prompt injection and header injection

package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/scottstg/yolo/email"
)

// Security constants for validation
const (
	AllowedSenderRegex = `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	DenylistedDomains  = "example.com|test.com|suspicious-domain.org"
)

// validateSender checks if sender is allowed (not denylisted)
// Format validation MUST happen before denylist check to avoid false rejections of invalid emails
func validateSender(emailAddress string) bool {
	// First, validate email format - must be valid before checking denylist
	formatRegex := regexp.MustCompile(AllowedSenderRegex)
	if !formatRegex.MatchString(emailAddress) {
		return false // Invalid email format
	}

	// Only check denylist for properly formatted emails
	denyRegex := regexp.MustCompile("(?i)(" + DenylistedDomains + ")")
	if denyRegex.MatchString(emailAddress) {
		return false // Email domain is denylisted
	}

	return true
}



// encodeHeader safely encodes header values to prevent header injection
func encodeHeader(value string) string {
	// Remove any embedded newlines which could allow header injection
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")

	// Truncate very long headers to prevent buffer overflow attacks
	if len(value) > 500 {
		value = value[:500] + "..."
	}

	return value
}

// loadAttachments reads file contents for attachment paths and returns email.Attachments
func loadAttachments(paths []string) ([]email.Attachment, error) {
	var attachments []email.Attachment
	
	for _, path := range paths {
		if path == "" {
			continue // Skip empty paths
		}
		
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read attachment %s: %v", path, err)
		}
		
		// Extract filename from path
		filename := path
		for i := len(path) - 1; i >= 0; i-- {
			if path[i] == '/' || path[i] == '\\' {
				filename = path[i+1:]
				break
			}
		}
		
		// Security: prevent path traversal in filename
		filename = sanitizeFilename(filename)
		
		attachments = append(attachments, email.Attachment{
			Filename: filename,
			Content:  content,
		})
	}
	
	return attachments, nil
}

// sanitizeFilename removes dangerous characters from filenames for email headers
func sanitizeFilename(filename string) string {
	filename = strings.ReplaceAll(filename, "\"", "")
	filename = strings.ReplaceAll(filename, "\n", "_")
	filename = strings.ReplaceAll(filename, "\r", "_")
	return filename
}

func (t *ToolExecutor) sendEmail(args map[string]any) string {
	subject := getStringArg(args, "subject", "")
	body := getStringArg(args, "body", "")
	to := getStringArg(args, "to", "")

	if subject == "" || body == "" {
		return "Error: subject and body parameters are required"
	}

	// Validate email format using proper regex
	emailRegex := regexp.MustCompile(AllowedSenderRegex)
	if !emailRegex.MatchString(to) {
		return fmt.Sprintf("Error: invalid email address '%s'", to)
	}

	// Get email configuration (uses local SMTP relay by default, no auth needed)
	cfg := email.DefaultConfig()

	// If no recipient specified, use default (scott@stg.net)
	if to == "" {
		to = "scott@stg.net"
	}

	// Validate sender is not denylisted
	if !validateSender(to) {
		return fmt.Sprintf("Error: email address '%s' is not allowed", to)
	}

	// Encode headers safely to prevent header injection
	safeSubject := encodeHeader(subject)

	// Load attachments if specified
	var attachments []email.Attachment
	if attachmentPaths, ok := args["attachments"].([]string); ok && len(attachmentPaths) > 0 {
		var err error
		attachments, err = loadAttachments(attachmentPaths)
		if err != nil {
			return fmt.Sprintf("Error loading attachments: %v", err)
		}
	}

	msg := &email.Message{
		To:          []string{to},
		Subject:     safeSubject,
		Body:        body,
		Attachments: attachments,
	}

	client := email.New(cfg)
	err := client.Send(msg)
	if err != nil {
		return fmt.Sprintf("Error sending email: %v", err)
	}

	var sb strings.Builder
	sb.WriteString("✅ Email sent successfully\n")
	sb.WriteString(fmt.Sprintf("   To: %s\n", to))
	sb.WriteString(fmt.Sprintf("   From: yolo@b-haven.org\n"))
	sb.WriteString(fmt.Sprintf("   Subject: %s\n", safeSubject))
	if len(attachments) > 0 {
		sb.WriteString(fmt.Sprintf("   Attachments: %d file(s)\n", len(attachments)))
	}
	return sb.String()
}

func (t *ToolExecutor) sendReport(args map[string]any) string {
	// Convenience function for sending progress reports
	cfg := email.DefaultConfig()

	subject := getStringArg(args, "subject", "YOLO Progress Report")
	body := getStringArg(args, "body", "")
	to := getStringArg(args, "to", "")                  // Get recipient from args
	attachTodo := getBoolArg(args, "attach_todo", true) // Default: include todo list

	if body == "" {
		return "Error: body parameter is required"
	}

	// Validate email format for recipient
	matched, _ := regexp.MatchString(AllowedSenderRegex, to)
	if !matched {
		return fmt.Errorf("invalid to address: %s", to).Error()
	}

	// Append todo list to the report if not already included and attach_todo is true
	if attachTodo && !strings.Contains(body, "📝 TODO LIST") {
		todoOutput := listTodos()
		body = body + "\n\n" + todoOutput
	}

	// Use provided recipient or default to scott@stg.net
	if to == "" {
		to = "scott@stg.net"
	}

	// Validate sender is not denylisted
	if !validateSender(to) {
		return fmt.Sprintf("Error: email address '%s' is not allowed", to)
	}

	// Encode headers safely to prevent header injection
	safeSubject := encodeHeader(subject)

	msg := &email.Message{
		To:      []string{to},
		Subject: safeSubject,
		Body:    body,
	}

	client := email.New(cfg)
	err := client.Send(msg)
	if err != nil {
		return fmt.Sprintf("Error sending report: %v", err)
	}

	var sb strings.Builder
	sb.WriteString("✅ Progress report sent successfully\n")
	sb.WriteString(fmt.Sprintf("   To: %s\n", to))
	sb.WriteString(fmt.Sprintf("   From: yolo@b-haven.org\n"))
	sb.WriteString(fmt.Sprintf("   Subject: %s\n", safeSubject))
	return sb.String()
}
