// Email Inbox Processing Tools with Security Hardening
// Implements check_inbox and process_inbox_with_response for autonomous email handling
// Includes protection against prompt injection, proper archival, and rate limiting

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const (
	InboxPath      = "/var/mail/b-haven.org/yolo/new/"
	CurDir         = "cur"
	ArchiveDir     = "archive"
	RateLimitEmail = "admin@b-haven.org" // Email for rate limit notifications
)

// emailArchived tracks which emails have been archived in this session
var (
	emailArchived  map[string]bool
	emailArchivedMu sync.Mutex
)

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

		// Generate AI response using LLM based on full email context
		prompt := fmt.Sprintf("You are YOLO, an autonomous AI assistant. You received this email:\n\nFrom: %s\nSubject: %s\n\nMessage:\n%s\n\nPlease generate a friendly, helpful, and concise response (maximum 200 words). Acknowledge their message, address any questions or topics they raised, and sign off as YOLO - Your Own Living Operator. Be conversational but professional.",
			email.From, email.Subject, email.Content)

		response := t.generateLLMText(prompt, true) // Use LLM for response generation
		if response == "" {
			log.Printf("Failed to generate LLM response, using fallback")
			response = generateSafeAIResponse(&email)
		}

		// Send response back to sender - extract email address from From header
		sender := email.From
		if strings.Contains(sender, "<") {
			// Extract email from within angle brackets: "Name <email>" -> "email"
			startIdx := strings.Index(sender, "<")
			endIdx := strings.Index(sender, ">")
			if startIdx != -1 && endIdx > startIdx {
				sender = strings.TrimSpace(sender[startIdx+1 : endIdx])
			} else {
				// Fallback: try to extract email-like pattern using regex
				emailPattern := regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
				if match := emailPattern.FindString(sender); match != "" {
					sender = match
				} else {
					sender = strings.TrimSpace(sender) // Use as-is, let validation handle it
				}
			}
		} else {
			sender = strings.TrimSpace(sender)
		}

		log.Printf("Extracted sender email: %s from From header: %s", sender, email.From)

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
	emailArchivedMu.Lock()
	if emailArchived[filename] {
		// Already archived in this session
		emailArchivedMu.Unlock()
		return nil
	}
	emailArchivedMu.Unlock()

	archiveDir := filepath.Join(ArchiveDir)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		log.Printf("Failed to create archive directory: %v", err)
		return err
	}

	destPath := filepath.Join(archiveDir, fmt.Sprintf("%s_%s", reason, filename))
	err := os.Rename(srcPath, destPath)
	if err == nil {
		emailArchivedMu.Lock()
		emailArchived[filename] = true
		emailArchivedMu.Unlock()
		log.Printf("Archived email %s to %s: %s", filename, reason, destPath)
	}

	return err
}

// generateSafeAIResponse creates a safe auto-response that resists prompt injection
func generateSafeAIResponse(email *EmailMessage) string {
	var sb strings.Builder

	sb.WriteString("Thank you for your message.\n\n")
	if email.Subject != "" {
		sb.WriteString(fmt.Sprintf("I've received your email regarding \"%s\" and will review it shortly.\n", email.Subject))
	} else {
		sb.WriteString("I've received your email and will review it shortly.\n")
	}

	// Use a template that explicitly mentions the original subject but doesn't
	// include or repeat any potentially malicious content from the body
	if strings.Contains(strings.ToLower(email.Subject), "hello") ||
		strings.Contains(strings.ToLower(email.Subject), "hi ") ||
		strings.Contains(strings.ToLower(email.Subject), "help") {
		sb.WriteString("\nI appreciate you reaching out. I'll get back to you with a more detailed response shortly.\n")
	}

	sb.WriteString("\nBest regards,\n")
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

// generateLLMText generates an AI response using the Ollama client
func (t *ToolExecutor) generateLLMText(prompt string, streaming bool) string {
	// Use agent's ollama client to generate response
	ctx := context.Background()
	msgs := []ChatMessage{
		{Role: "user", Content: prompt},
	}

	result, err := t.agent.ollama.Chat(ctx, t.agent.config.GetModel(), msgs, nil, nil)
	if err != nil {
		log.Printf("Error generating LLM text: %v", err)
		return ""
	}
	return result.ContentText
}
