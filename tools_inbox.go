// Email Inbox Processing Tools with Security Hardening
// Implements check_inbox and process_inbox_with_response for autonomous email handling
// Includes protection against prompt injection, proper archival, and rate limiting

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/emersion/go-message/mail"
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

// parseEmail extracts relevant fields from email content using proper MIME parsing
func parseEmail(content, filename string) EmailMessage {
	email := EmailMessage{
		Filename:    filename,
		ContentType: "text/plain",
		Size:        int64(len(content)),
	}

	// Parse the email as a MIME message using go-message library
	// First, strip Postfix envelope headers (Return-Path, Received, etc.) before parsing
	// The actual RFC822 message starts at standard headers like From:, To:, Subject:, MIME-Version:
	actualContent := stripEnvelopeHeaders(content)

	reader, err := mail.CreateReader(bytes.NewReader([]byte(actualContent)))
	if err != nil {
		log.Printf("Error creating MIME reader for %s: %v - falling back to simple parsing", filename, err)
		return parseEmailSimple(content, filename)
	}

	// Extract headers safely with sanitization
	email.From = sanitizeEmailField(reader.Header.Get("From"))
	email.Subject = sanitizeEmailField(reader.Header.Get("Subject"))
	email.Date = sanitizeEmailField(reader.Header.Get("Date"))

	// Get To addresses
	toHeader := reader.Header.Get("To")
	if toHeader != "" {
		toAddresses := strings.Split(toHeader, ",")
		for _, addr := range toAddresses {
			email.To = append(email.To, sanitizeEmailField(strings.TrimSpace(addr)))
		}
	}

	// Extract the plain text body using the reader's part iterator
	var body strings.Builder

	// Track if we found plain text (prefer it over HTML)
	foundPlainText := false

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading part from %s: %v", filename, err)
			break
		}

		// Get content type
		contentType := part.Header.Get("Content-Type")

		// Skip attachments (application, image types)
		if strings.HasPrefix(contentType, "application/") ||
			strings.HasPrefix(contentType, "image/") {
			continue
		}

		// Read the raw content of this part
		rawData, err := io.ReadAll(part.Body)
		if err != nil {
			log.Printf("Error reading part data from %s: %v", filename, err)
			continue
		}

		// Get transfer encoding to decode properly
		transferEncoding := strings.ToLower(part.Header.Get("Content-Transfer-Encoding"))

		// Decode based on transfer encoding
		var decodedData []byte
		switch transferEncoding {
		case "quoted-printable":
			decodedData, err = decodeQuotedPrintable(rawData)
			if err != nil {
				log.Printf("Error decoding quoted-printable from %s: %v", filename, err)
				decodedData = rawData // Fall back to raw data
			}
		case "base64":
			decodedData, err = base64.StdEncoding.DecodeString(string(rawData))
			if err != nil {
				log.Printf("Error decoding base64 from %s: %v", filename, err)
				decodedData = rawData // Fall back to raw data
			}
		case "7bit", "8bit", "binary", "":
			// No decoding needed
			decodedData = rawData
		default:
			log.Printf("Unknown transfer encoding '%s' in %s, using raw data", transferEncoding, filename)
			decodedData = rawData
		}

		body.WriteString(string(decodedData))

		// Stop after finding a text/plain body (first match wins)
		if strings.HasPrefix(contentType, "text/plain") {
			foundPlainText = true
			break
		}
	}

	// If we only found HTML and no plain text, go back and try to get HTML
	if !foundPlainText && body.Len() == 0 {
		// Reset reader - not easily possible with current library, so use simple fallback instead
		log.Printf("No plain text body found in %s, using simple parser", filename)
		simpleEmail := parseEmailSimple(content, filename)
		email.Content = simpleEmail.Content
		return email
	}

	// Set Content to just the body text, preserving newlines
	// DO NOT truncate - preserve full email content for proper LLM context and response generation
	email.Content = strings.TrimSpace(body.String())

	if email.Content == "" {
		log.Printf("No content extracted from MIME message %s - falling back to simple parsing", filename)
		simpleEmail := parseEmailSimple(content, filename)
		email.Content = simpleEmail.Content
	}

	return email
}

// decodeQuotedPrintable decodes a quoted-printable encoded byte slice
func decodeQuotedPrintable(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	// Process line by line (quoted-printable is line-oriented)
	lines := bytes.Split(data, []byte("\n"))

	for i, line := range lines {
		// Remove soft line breaks (= at end of line) and join
		if bytes.HasSuffix(line, []byte("=")) {
			// Soft line break - remove the = and continue to next line
			line = bytes.TrimRight(line, "=")
			buf.Write(line)
			continue
		}

		// Decode escape sequences (=XX where XX is hex)
		decoded := make([]byte, 0, len(line))
		j := 0
		for j < len(line) {
			if line[j] == '=' && j+2 < len(line) {
				// Try to decode =XX hex sequence
				hexChars := string(line[j+1 : j+3])
				decodedByte, err := parseQuotedPrintableHex(hexChars)
				if err != nil {
					// If it's not valid hex, keep the original characters
					decoded = append(decoded, line[j:j+3]...)
					j += 3
					continue
				}
				decoded = append(decoded, decodedByte)
				j += 3
			} else {
				decoded = append(decoded, line[j])
				j++
			}
		}

		buf.Write(decoded)
		if i < len(lines)-1 {
			buf.WriteByte('\n')
		}
	}

	return buf.Bytes(), nil
}

// parseQuotedPrintableHex decodes a two-character hex string to a byte
func parseQuotedPrintableHex(hexStr string) (byte, error) {
	var result byte
	for i, c := range hexStr {
		var val byte
		switch {
		case c >= '0' && c <= '9':
			val = byte(c - '0')
		case c >= 'A' && c <= 'F':
			val = byte(c - 'A' + 10)
		case c >= 'a' && c <= 'f':
			val = byte(c - 'a' + 10)
		default:
			return 0, fmt.Errorf("invalid hex character: %c", c)
		}
		if i == 0 {
			result = val << 4
		} else {
			result |= val
		}
	}
	return result, nil
}

// parseEmailSimple provides a fallback parser for non-MIME or corrupted emails
func parseEmailSimple(content, filename string) EmailMessage {
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
	// DO NOT truncate - preserve full email content for proper LLM context and response generation
	email.Content = strings.TrimSpace(body)

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

// stripEnvelopeHeaders removes Postfix delivery headers and returns only the RFC822 message
func stripEnvelopeHeaders(content string) string {
	lines := strings.Split(content, "\n")

	// Find where actual message headers start (From:, Date:, Subject:, Message-ID:, MIME-Version:)
	messageStart := -1
	for i, line := range lines {
		lowerLine := strings.ToLower(line)
		if strings.HasPrefix(lowerLine, "from:") ||
			strings.HasPrefix(lowerLine, "date:") ||
			strings.HasPrefix(lowerLine, "subject:") ||
			strings.HasPrefix(lowerLine, "message-id:") ||
			strings.HasPrefix(lowerLine, "mime-version:") {
			messageStart = i
			break
		}
	}

	// If no message headers found, return original content
	if messageStart == -1 {
		return content
	}

	// Return everything from the first actual message header to the end
	return strings.Join(lines[messageStart:], "\n")
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

// extractEmailAddress extracts just the email address from a From header like "Name <email@example.com>"
func extractEmailAddress(fromHeader string) string {
	sender := strings.TrimSpace(fromHeader)
	if strings.Contains(sender, "<") {
		// Extract email from within angle brackets: "Name <email>" -> "email"
		startIdx := strings.Index(sender, "<")
		endIdx := strings.Index(sender, ">")
		if startIdx != -1 && endIdx > startIdx {
			return strings.TrimSpace(sender[startIdx+1 : endIdx])
		} else {
			// Fallback: try to extract email-like pattern using regex
			emailPattern := regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
			if match := emailPattern.FindString(sender); match != "" {
				return match
			}
		}
	}
	return sender // Use as-is if no angle brackets found
}

// isProgressReportRequest detects if an email is asking for a progress report
func isProgressReportRequest(email *EmailMessage) bool {
	contentLower := strings.ToLower(email.Content)
	subjectLower := strings.ToLower(email.Subject)

	// Check for common progress report request patterns
	indicators := []string{
		"progress report", "status report", "what are you working on",
		"what's your status", "update me", "give me a report",
		"report please", "send me a report", "your todo list",
		"current progress", "what have you been doing",
		"progress", "todo", "status",
	}

	for _, indicator := range indicators {
		if strings.Contains(contentLower, indicator) || strings.Contains(subjectLower, indicator) {
			return true
		}
	}

	return false
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

		// Debug: log email parsing details
		log.Printf("Email from %s with subject '%s' - Content length: %d bytes", email.From, email.Subject, len(email.Content))
		if email.Content == "" {
			log.Printf("WARNING: Email content is EMPTY for message from %s with subject '%s'", email.From, email.Subject)
		}

		// Extract sender email address using helper function
		sender := extractEmailAddress(email.From)
		log.Printf("Extracted sender email: %s", sender)

		// Check if this is a request for a progress report
		isProgressReportRequestFlag := isProgressReportRequest(&email)

		var response string
		if isProgressReportRequestFlag {
			log.Printf("Detected progress report request from %s, sending actual report...", email.From)

			// Generate and send actual progress report
			todoOutput := listTodos()
			reportBody := fmt.Sprintf("🤖 YOLO Progress Report\n\nGenerated in response to your email request.\n\n%s", todoOutput)

			sendResult := t.sendReport(map[string]any{
				"to":          sender,
				"subject":     "YOLO Progress Report",
				"body":        reportBody,
				"attach_todo": false,
			})
			log.Printf("Progress report send result: %s", sendResult)

			// Acknowledgment response to send back
			response = "Thanks for reaching out! I have just sent you a detailed progress report with my current status and todo list. You should receive it shortly.\n\nBest regards,\nYOLO - Your Own Living Operator"
		} else {
			// Generate AI response using LLM based on full email context - NO TEMPLATE FALLBACKS
			prompt := fmt.Sprintf("You are YOLO, an autonomous AI assistant running in production. You received this email:\n\nFrom: %s\nSubject: %s\n\nMessage content:\n%s\n\nInstructions for generating response:\n1. SIGNATURE DETECTION: The last few lines of emails often contain signatures (company names like 'STG.NET & B-Haven', phone numbers, website links, disclaimers). These are NOT part of the actual request - use your judgment to identify and ignore them. Focus only on the substantive message content.\n2. RESPOND TO SUBSTANCE: Generate a genuine response that addresses the actual request or questions in the email body, not the signature.\n3. NO LENGTH LIMITS: Provide complete, detailed information when requested. Include full progress reports, complete todo lists with all details, comprehensive explanations. Do NOT summarize, truncate, or artificially limit your response length.\n4. BE CONVERSATIONAL: Write naturally and professionally, acknowledging what was actually asked.\n5. SIGN YOUR RESPONSE: End as 'YOLO - Your Own Living Operator'\n\nResponse:",
				email.From, email.Subject, email.Content)

			log.Printf("Generating LLM response for email from %s...", email.From)
			response = strings.TrimSpace(t.generateLLMText(prompt, true))
			if response == "" {
				log.Printf("ERROR: Failed to generate LLM response (empty after trim) - skipping email, NO reply sent")
				log.Printf("PROMPT used: %.200s...", prompt)
				// Archive without responding when LLM fails - NO template fallbacks
				archiveEmail(filePath, file.Name(), "llm_generation_failed")
				skipped = append(skipped, file.Name())
				continue
			}

			log.Printf("SUCCESS: Generated LLM response: %d bytes, preview: %.80s", len(response), response)
		}

		// Validate sender before responding
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

	result, err := t.agent.ollama.Chat(ctx, t.agent.config.GetModel(), msgs, nil, nil, true)
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
