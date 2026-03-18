// Email Inbox Processing Tools with Security Hardening
// Implements check_inbox and process_inbox_with_response for autonomous email handling
// Includes protection against prompt injection, proper archival, and rate limiting

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	InboxPath      = "/var/mail/b-haven.org/yolo/new/"
	CurDir         = "cur"
	ArchiveDir     = "archive"
	RateLimitEmail = "admin@b-haven.org" // Email for rate limit notifications
)

// emailArchived tracks which emails have been archived in this session
var emailArchived map[string]bool

func init() {
	emailArchived = make(map[string]bool)
}

// checkInbox reads emails from the Maildir inbox
func (t *ToolExecutor) checkInbox(args map[string]any) string {
	markRead := getBoolArg(args, "mark_read", false)

	// Check if inbox directory exists
	if _, err := os.Stat(InboxPath); os.IsNotExist(err) {
		return fmt.Sprintf("No emails in inbox. Inbox path: %s", InboxPath)
	}

	// Read all files in the inbox directory
	files, err := os.ReadDir(InboxPath)
	if err != nil {
		return fmt.Sprintf("Error reading inbox: %v", err)
	}

	if len(files) == 0 {
		return "No emails found in inbox"
	}

	var emailList []EmailMessage
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(InboxPath, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Error reading email %s: %v", file.Name(), err)
			continue
		}

		email := parseEmail(string(content), file.Name())
		emailList = append(emailList, email)

		// Move to cur directory if mark_read is true
		if markRead {
			curDir := filepath.Join(CurDir)
			if err := os.MkdirAll(curDir, 0755); err == nil {
				destPath := filepath.Join(curDir, file.Name())
				os.Rename(filePath, destPath)
			}
		}
	}

	result, _ := json.MarshalIndent(emailList, "", "  ")
	return fmt.Sprintf("📧 Found %d email(s):\n\n%s", len(emailList), string(result))
}

// parseEmail extracts relevant fields from email content with security validation
func parseEmail(content, filename string) EmailMessage {
	email := EmailMessage{
		Filename:    filename,
		ContentType: "text/plain",
		Size:        int64(len(content)),
		Content:     truncateString(content, 500),
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "From:") {
			email.From = sanitizeEmailField(strings.TrimSpace(strings.TrimPrefix(line, "From:")))
		} else if strings.HasPrefix(line, "Subject:") {
			email.Subject = sanitizeEmailField(strings.TrimSpace(strings.TrimPrefix(line, "Subject:")))
		} else if strings.HasPrefix(line, "Date:") {
			email.Date = sanitizeEmailField(strings.TrimSpace(strings.TrimPrefix(line, "Date:")))
		} else if strings.HasPrefix(line, "To:") {
			toAddresses := strings.Split(strings.TrimSpace(strings.TrimPrefix(line, "To:")), ",")
			for i, addr := range toAddresses {
				toAddresses[i] = sanitizeEmailField(strings.TrimSpace(addr))
			}
			email.To = toAddresses
		}
	}

	return email
}

// sanitizeEmailField sanitizes email header fields to prevent injection attacks
func sanitizeEmailField(value string) string {
	// Remove embedded newlines that could enable header injection
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")

	// Truncate very long headers to prevent buffer overflow attacks
	if len(value) > 497 {
		value = value[:497] + "..."
	}

	return value
}

// isBounceMessage checks if an email appears to be a bounce/delivery failure notification
func isBounceMessage(email *EmailMessage) bool {
	bounceIndicators := []string{
		"delivery failed", "undeliverable", "bounce", "returned mail",
		"permanent failure", "temporary failure", "postmaster@",
		"mailer-daemon@", "daemon@", "failure notice", "notification system",
		"unable to deliver", "recipient rejected", "user unknown",
	}

	contentLower := strings.ToLower(email.Content)
	for _, indicator := range bounceIndicators {
		if strings.Contains(contentLower, indicator) {
			return true
		}
	}

	// Check from address for common bounce senders
	fromLower := strings.ToLower(email.From)
	bounceSenders := []string{
		"postmaster@", "mailer-daemon@", "daemon@", "automated@",
		"notification@", "nobody@", "no-reply@", "donotreply@",
	}
	for _, sender := range bounceSenders {
		if strings.Contains(fromLower, sender) {
			return true
		}
	}

	return false
}

// isRateLimited checks if we've exceeded the email sending rate limit
func isRateLimited() bool {
	emailCount.Load() // This is a no-op to check current count
	return emailCount.Load() >= MaxEmailsPerHour
}

// processInboxWithResponse automates the email workflow: read → respond → archive
func (t *ToolExecutor) processInboxWithResponse(args map[string]any) string {
	markRead := getBoolArg(args, "mark_read", true)

	// Check if inbox directory exists
	if _, err := os.Stat(InboxPath); os.IsNotExist(err) {
		return fmt.Sprintf("No emails in inbox. Inbox path: %s", InboxPath)
	}

	// Read all files in the inbox directory
	files, err := os.ReadDir(InboxPath)
	if err != nil {
		return fmt.Sprintf("Error reading inbox: %v", err)
	}

	if len(files) == 0 {
		return "No emails found in inbox to process"
	}

	totalProcessed := 0
	totalSkipped := 0
	var processed []string
	var skipped []string

	// Create archive directory if it doesn't exist
	archiveDir := filepath.Join(ArchiveDir)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		log.Printf("Failed to create archive directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(InboxPath, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Error reading email %s: %v", file.Name(), err)
			continue
		}

		email := parseEmail(string(content), file.Name())

		// Skip bounce/delivery failure messages to avoid infinite loops
		if isBounceMessage(&email) {
			log.Printf("Skipping bounce message from %s with subject %s", email.From, email.Subject)
			skipped = append(skipped, file.Name())

			// Still archive these for audit purposes
			archiveEmail(filePath, file.Name(), "bounce_skipped")
			continue
		}

		// Check rate limiting before generating response
		if isRateLimited() {
			log.Printf("Rate limit exceeded, skipping email %s", file.Name())
			skipped = append(skipped, file.Name())

			// Archive the email and notify admin
			archiveEmail(filePath, file.Name(), "rate_limited")

			// Send notification to admin about rate limit
			t.sendEmail(map[string]any{
				"to":      RateLimitEmail,
				"subject": "YOLO Email Rate Limit Exceeded",
				"body":    fmt.Sprintf("YOLO has reached its hourly email sending limit. %d emails processed, %d skipped due to rate limiting.", totalProcessed, len(skipped)),
			})

			break // Stop processing remaining emails
		}

		// Generate safe auto-response based on email content
		response := generateSafeAIResponse(&email)

		// Send response back to sender
		sender := email.From
		if strings.Contains(sender, "<") {
			parts := strings.SplitN(sender, "<", 2)
			sender = strings.TrimSpace(strings.TrimSuffix(parts[0], ">"))
		}

		// Validate sender before responding
		if !validateSender(sender) {
			log.Printf("Skipping response to untrusted sender: %s", sender)
			skipped = append(skipped, file.Name())
			continue
		}

		subjectResp := fmt.Sprintf("Re: %s", email.Subject)
		t.sendEmail(map[string]any{
			"to":      sender,
			"subject": subjectResp,
			"body":    response,
		})

		// Archive email instead of deleting (preserves audit trail)
		archiveEmail(filePath, file.Name(), "processed")

		totalProcessed++
		processed = append(processed, file.Name())

		// Mark as read (move to cur directory) if requested
		if markRead {
			curDir := filepath.Join(CurDir)
			if err := os.MkdirAll(curDir, 0755); err == nil {
				destPath := filepath.Join(curDir, "processed_"+file.Name())
				os.Rename(filePath, destPath)
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✅ Processed %d email(s)\n", totalProcessed))
	if len(processed) > 0 {
		sb.WriteString(fmt.Sprintf("📁 Archived: %s\n", strings.Join(processed, ", ")))
	}
	if len(skipped) > 0 {
		sb.WriteString(fmt.Sprintf("⚠️  Skipped %d messages:\n", totalSkipped))
		for _, name := range skipped {
			sb.WriteString(fmt.Sprintf("    - %s\n", name))
		}
	}
	if len(processed) > 0 {
		sb.WriteString("📧 Sent auto-responses to all senders")
	}

	return sb.String()
}

// archiveEmail moves an email to the archive directory for audit purposes
func archiveEmail(srcPath, filename string, reason string) error {
	if emailArchived[filename] {
		// Already archived in this session
		return nil
	}

	archiveDir := filepath.Join(ArchiveDir)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		log.Printf("Failed to create archive directory: %v", err)
		return err
	}

	destPath := filepath.Join(archiveDir, fmt.Sprintf("%s_%s", reason, filename))
	err := os.Rename(srcPath, destPath)
	if err == nil {
		emailArchived[filename] = true
		log.Printf("Archived email %s to %s: %s", filename, reason, destPath)
	}

	return err
}

// generateSafeAIResponse creates a safe auto-response that resists prompt injection
func generateSafeAIResponse(email *EmailMessage) string {
	var sb strings.Builder

	// Sanitized greeting - truncate sender name to prevent injection
	safeFrom := sanitizeInboxContent(truncateString(email.From, 50))
	if safeFrom != "" {
		sb.WriteString(fmt.Sprintf("Hello %s,\n\n", safeFrom))
	} else {
		sb.WriteString("Hello,\n\n")
	}

	// Sanitized subject acknowledgment
	safeSubject := sanitizeInboxContent(truncateString(email.Subject, 100))
	sb.WriteString(fmt.Sprintf("I've received your message regarding \"%s\".\n", safeSubject))
	sb.WriteString("\n")

	// Provide appropriate auto-response based on content type (using sanitized content)
	contentSanitized := sanitizeInboxContent(strings.ToLower(email.Content))
	if strings.Contains(contentSanitized, "hello") ||
		strings.Contains(contentSanitized, "hi") {
		sb.WriteString("Thank you for reaching out! I'm YOLO, your autonomous AI agent.\n")
		sb.WriteString("I'm here to help with software development tasks and project automation.\n")
	} else if strings.Contains(safeSubject, "status") ||
		strings.Contains(contentSanitized, "status") {
		sb.WriteString("I can provide status updates on my current activities.\n")
		sb.WriteString("Would you like me to send a detailed progress report?\n")
	} else if strings.Contains(safeSubject, "todo") ||
		strings.Contains(contentSanitized, "todo") {
		sb.WriteString("I maintain a todo list for tracking tasks and improvements.\n")
		sb.WriteString("Let me know what specific tasks you'd like to discuss!\n")
	} else {
		sb.WriteString("Thank you for your message. I'll review it and get back to you\n")
		sb.WriteString("with a more detailed response shortly.\n")
	}

	sb.WriteString("\n")
	sb.WriteString("Best regards,\n")
	sb.WriteString("YOLO - Your Own Living Operator\n")

	return sb.String()
}

// sanitizeContent removes potentially malicious content from email body
// This helps prevent prompt injection attacks
func sanitizeInboxContent(content string) string {
	// Remove potential command injection patterns (line breaks and separators only)
	content = regexp.MustCompile(`[;|&]\s*`).ReplaceAllString(content, " ")

	// Remove potential template injection markers
	content = regexp.MustCompile(`\{\{.*?\}\}`).ReplaceAllStringFunc(content, func(match string) string {
		return "[REDACTED]"
	})

	// Remove shell command patterns that could be interpreted by downstream systems
	// Process dollar-sign command substitution first: $(...)
	content = regexp.MustCompile(`\$\([^)]*\)`).ReplaceAllStringFunc(content, func(match string) string {
		return "[COMMAND_REDACTED]"
	})
	// Process backtick command substitution second: `...`
	content = regexp.MustCompile("`[^`]*`").ReplaceAllStringFunc(content, func(match string) string {
		return "[COMMAND_REDACTED]"
	})

	// Truncate very long content to prevent memory issues
	if len(content) > 10000 {
		content = content[:10000] + " [CONTENT TRUNCATED - ORIGINAL EXCEEDED 10KB LIMIT]"
	}

	return content
}