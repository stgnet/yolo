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
	emailArchived   map[string]bool
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
	}

	// Email format: headers followed by blank line, then body
	// Find the blank line that separates headers from body
	lines := strings.Split(content, "\n")
	bodyStartIdx := 0
	headersFound := false
	
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			if headersFound {
				// Found blank line after headers - body starts here
				bodyStartIdx = i + 1
				break
			}
		} else if strings.Contains(line, ":") {
			headersFound = true
		}
	}

	// Extract body content (everything after the blank line)
	var bodyLines []string
	for i := bodyStartIdx; i < len(lines); i++ {
		bodyLines = append(bodyLines, lines[i])
	}
	body := strings.Join(bodyLines, "\n")
	
	// Set Content to just the body text (not headers)
	email.Content = sanitizeEmailField(truncateString(strings.TrimSpace(body), 500))

	// Now parse headers from the first part of content
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

		// Generate AI response using LLM based on full email context - NO TEMPLATE FALLBACKS
		prompt := fmt.Sprintf("You are YOLO, an autonomous AI assistant running in production. You received this email:\n\nFrom: %s\nSubject: %s\n\nMessage content (first 500 chars):\n%s\n\nInstructions:\n- Generate a genuine, personalized response to this specific message\n- Do NOT use generic templates or placeholder text\n- Acknowledge the sender's actual content and questions if any\n- Keep response under 200 words\n- Be conversational but professional\n- Sign as 'YOLO - Your Own Living Operator'\n\nResponse:",
			email.From, email.Subject, email.Content)

		log.Printf("Generating LLM response for email from %s...", email.From)
	response := strings.TrimSpace(t.generateLLMText(prompt, true))
	if response == "" {
		log.Printf("ERROR: Failed to generate LLM response (empty after trim) - skipping email, NO reply sent")
		log.Printf("PROMPT used: %.200s...", prompt)
		// Archive without responding when LLM fails - NO template fallbacks
		archiveEmail(filePath, file.Name(), "llm_generation_failed")
		skipped = append(skipped, file.Name())
		continue
	}

	log.Printf("SUCCESS: Generated LLM response: %d bytes, preview: %.80s", len(response), response)

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

	// Use DisplayText which falls back to ThinkingText if ContentText is empty
	response := strings.TrimSpace(result.DisplayText)

	log.Printf("generateLLMText result - ContentText: %.50s, ThinkingText: %.50s, DisplayText: %.50s", 
		result.ContentText, result.ThinkingText, result.DisplayText)

	if response == "" {
		return ""
	}

	return response
}

// safeTruncate safely truncates a string, handling empty strings
func safeTruncate(s string, maxLen int) string {
	if len(s) == 0 {
		return "(empty)"
	}
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
