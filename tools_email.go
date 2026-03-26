// Email Tool Implementation with Security Hardening
// Allows YOLO to send emails via SMTP using configured sender address
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
)

// validateEmailFormat checks if an email address has valid format
func validateEmailFormat(emailAddress string) bool {
	formatRegex := regexp.MustCompile(AllowedSenderRegex)
	return formatRegex.MatchString(emailAddress)
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

// parseAttachmentArg extracts attachment paths from tool arguments.
func parseAttachmentArg(args map[string]any) []string {
	attachmentArg, ok := args["attachments"]
	if !ok || attachmentArg == nil {
		return nil
	}
	var paths []string
	switch v := attachmentArg.(type) {
	case []string:
		paths = v
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok && str != "" {
				paths = append(paths, str)
			}
		}
	}
	return paths
}

func (t *ToolExecutor) sendEmail(args map[string]any) string {
	subject := getStringArg(args, "subject", "")
	body := getStringArg(args, "body", "")
	to := getStringArg(args, "to", "")

	if subject == "" || body == "" {
		return errorMessage("subject and body parameters are required")
	}

	// Use configured default recipient if none provided
	if to == "" {
		if t.agent != nil {
			to = t.agent.config.GetEmailTo()
		}
	}
	if to == "" {
		return errorMessage("'to' parameter is required (no default recipient configured)")
	}

	// Validate email format
	if !validateEmailFormat(to) {
		return errorMessage("invalid email address '%s'", to)
	}

	// Get sender from config
	var emailFrom string
	if t.agent != nil {
		emailFrom = t.agent.config.GetEmailFrom()
	}
	cfg := email.ConfigWithFrom(emailFrom)

	// Encode headers safely to prevent header injection
	safeSubject := encodeHeader(subject)

	// Load attachments if specified
	var attachments []email.Attachment
	if paths := parseAttachmentArg(args); len(paths) > 0 {
		var err error
		attachments, err = loadAttachments(paths)
		if err != nil {
			return errorMessage("loading attachments: %v", err)
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
		return errorMessage("sending email: %v", err)
	}

	var sb strings.Builder
	sb.WriteString("Email sent successfully\n")
	sb.WriteString(fmt.Sprintf("   To: %s\n", to))
	sb.WriteString(fmt.Sprintf("   From: %s\n", cfg.From))
	sb.WriteString(fmt.Sprintf("   Subject: %s\n", safeSubject))
	if len(attachments) > 0 {
		sb.WriteString(fmt.Sprintf("   Attachments: %d file(s)\n", len(attachments)))
	}
	return sb.String()
}

func (t *ToolExecutor) sendReport(args map[string]any) string {
	subject := getStringArg(args, "subject", "YOLO Progress Report")
	body := getStringArg(args, "body", "")
	to := getStringArg(args, "to", "")
	attachTodo := getBoolArg(args, "attach_todo", true)

	if body == "" {
		return errorMessage("body parameter is required")
	}

	// Use configured default recipient if none provided
	if to == "" {
		if t.agent != nil {
			to = t.agent.config.GetEmailTo()
		}
	}
	if to == "" {
		return errorMessage("'to' parameter is required (no default recipient configured)")
	}

	// Validate email format
	if !validateEmailFormat(to) {
		return errorMessage("invalid to address: %s", to)
	}

	// Append todo list to the report if not already included and attach_todo is true
	if attachTodo && !strings.Contains(body, "TODO LIST") {
		todoOutput := listTodos()
		body = body + "\n\n" + todoOutput
	}

	// Get sender from config
	var emailFrom string
	if t.agent != nil {
		emailFrom = t.agent.config.GetEmailFrom()
	}
	cfg := email.ConfigWithFrom(emailFrom)

	// Encode headers safely to prevent header injection
	safeSubject := encodeHeader(subject)

	// Load attachments if specified
	var attachments []email.Attachment
	if paths := parseAttachmentArg(args); len(paths) > 0 {
		var err error
		attachments, err = loadAttachments(paths)
		if err != nil {
			return errorMessage("loading attachments: %v", err)
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
		return errorMessage("sending report: %v", err)
	}

	var sb strings.Builder
	sb.WriteString("Progress report sent successfully\n")
	sb.WriteString(fmt.Sprintf("   To: %s\n", to))
	sb.WriteString(fmt.Sprintf("   From: %s\n", cfg.From))
	sb.WriteString(fmt.Sprintf("   Subject: %s\n", safeSubject))
	if len(attachments) > 0 {
		sb.WriteString(fmt.Sprintf("   Attachments: %d file(s)\n", len(attachments)))
	}
	return sb.String()
}
