// Email Inbox Tool Implementation
// Provides tools to read and process emails from Maildir

package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// checkInbox reads emails from the Maildir inbox
func (t *ToolExecutor) checkInbox(args map[string]any) string {
	markRead := getBoolArg(args, "mark_read", false)

	newDir := "/var/mail/b-haven.org/yolo/new/"
	curDir := "/var/mail/b-haven.org/yolo/cur/"

	emails, processedCount, err := readMaildir(newDir, curDir, markRead)
	if err != nil {
		if os.IsNotExist(err) {
			return "📭 No new emails (inbox directory not found - may need to create /var/mail/b-haven.org/yolo/)"
		}
		return fmt.Sprintf("❌ Error reading inbox: %v", err)
	}

	var sb strings.Builder

	if len(emails) == 0 {
		sb.WriteString("📭 No new emails in inbox\n")
		if markRead {
			sb.WriteString("   (No emails to process)\n")
		}
	} else {
		sb.WriteString(fmt.Sprintf("📬 Found %d new email(s)\n", len(emails)))
		if markRead && processedCount > 0 {
			sb.WriteString(fmt.Sprintf("   Moved %d email(s) to cur/ (marked as read)\n", processedCount))
		}
		sb.WriteString("\n")

		for i, email := range emails {
			sb.WriteString(fmt.Sprintf("--- Email %d of %d ---\n", i+1, len(emails)))
			sb.WriteString(fmt.Sprintf("From: %s\n", email.From))
			sb.WriteString(fmt.Sprintf("Subject: %s\n", email.Subject))
			sb.WriteString(fmt.Sprintf("Date: %s\n", email.Date))
			if email.ContentType != "" {
				sb.WriteString(fmt.Sprintf("Content-Type: %s\n", email.ContentType))
			}
			sb.WriteString("\nBody:\n")
			sb.WriteString(email.Content)
			sb.WriteString("\n")
			if i < len(emails)-1 {
				sb.WriteString(strings.Repeat("-", 50) + "\n")
			}
		}
	}

	return sb.String()
}

// readMaildir reads emails from the new directory, optionally moving to cur/ if markRead is true

func readMaildir(newDir, curDir string, markRead bool) ([]EmailMessage, int, error) {
	var emails []EmailMessage
	processedCount := 0

	files, err := os.ReadDir(newDir)
	if err != nil {
		return nil, 0, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(newDir, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		email, err := parseEmailMessage(content, file.Name())
		if err != nil {
			continue
		}

		emails = append(emails, email)

		if markRead {
			curPath := filepath.Join(curDir, file.Name())
			if err := os.Rename(filePath, curPath); err == nil {
				processedCount++
			}
		}
	}

	return emails, processedCount, nil
}

// parseEmailMessage parses the raw email content into an EmailMessage struct
func parseEmailMessage(content []byte, filename string) (EmailMessage, error) {
	email := EmailMessage{Filename: filename}

	// Parse as RFC 2822 email using textproto
	reader := bufio.NewReader(bytes.NewReader(content))
	msgReader, err := textproto.NewReader(reader).ReadMIMEHeader()
	if err != nil {
		return email, fmt.Errorf("failed to parse headers: %w", err)
	}

	email.From = msgReader.Get("From")
	email.Subject = msgReader.Get("Subject")
	email.Date = msgReader.Get("Date")
	contentType := msgReader.Get("Content-Type")
	if contentType != "" {
		fullType := contentType
		if len(fullType) > 50 {
			fullType = fullType[:50] + "..."
		}
		email.ContentType = fullType
	}

	// Extract body from the remaining content (after headers)
	bodyContent, err := io.ReadAll(reader)
	if err != nil {
		return email, fmt.Errorf("failed to read body: %w", err)
	}

	// Parse the body based on content type
	email.Content = extractBodyFromBytes(bodyContent, contentType)

	return email, nil
}

// extractBodyFromBytes extracts text content from raw email bytes
func extractBodyFromBytes(data []byte, contentType string) string {
	reader := bytes.NewReader(data)
	return extractBody(reader, contentType)
}

// extractBody extracts the text body from an email based on content type
func extractBody(reader io.Reader, contentType string) string {
	// Handle multipart messages
	mediatype, params, err := mime.ParseMediaType(contentType)
	if err == nil && strings.HasPrefix(mediatype, "multipart/") {
		mpReader := multipart.NewReader(reader, params["boundary"])
		var textParts []string

		for {
			part, err := mpReader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return ""
			}

			partContent := strings.ToLower(part.Header.Get("Content-Type"))

			// Prefer text/plain over other content types
			partBody := extractBody(part, partContent)
			if strings.Contains(partContent, "text/plain") && !strings.Contains(partContent, "alternative") {
				textParts = append(textParts, partBody)
				break // Found plain text, prefer this one
			} else if len(textParts) == 0 {
				textParts = append(textParts, partBody)
			}
		}

		if len(textParts) > 0 {
			return strings.Join(textParts, "\n\n")
		}
	}

	// Check for charset encoding (text/plain or text/html)
	if strings.HasPrefix(contentType, "text/plain; charset=") || strings.HasPrefix(contentType, "text/html; charset=") {
		data, err := io.ReadAll(reader)
		if err != nil {
			return ""
		}

		// Try quoted-printable decoding
		dec := quotedprintable.NewReader(bytes.NewReader(data))
		decoded, err := io.ReadAll(dec)
		if err == nil {
			return string(decoded)
		}

		return string(data)
	}

	// Check for base64 encoding
	if strings.Contains(contentType, "base64") || (strings.Contains(contentType, "charset=") && !strings.Contains(contentType, "multipart")) {
		data, err := io.ReadAll(reader)
		if err != nil {
			return ""
		}

		decoded, err := base64.StdEncoding.DecodeString(string(data))
		if err == nil {
			return string(decoded)
		}
		return string(data)
	}

	// Fallback: read as plain text
	data, err := io.ReadAll(reader)
	if err != nil {
		return ""
	}

	return string(data)
}

// processInboxWithResponse checks inbox, composes responses for qualifying emails, and deletes them
func (t *ToolExecutor) processInboxWithResponse(args map[string]any) string {
	newDir := "/var/mail/b-haven.org/yolo/new/"
	curDir := "/var/mail/b-haven.org/yolo/cur/"

	emails, _, err := readMaildir(newDir, curDir, false)
	if err != nil {
		if os.IsNotExist(err) {
			return "📭 No inbox directory found - may need to create /var/mail/b-haven.org/yolo/"
		}
		return fmt.Sprintf("❌ Error reading inbox: %v", err)
	}

	if len(emails) == 0 {
		return "📭 No emails in inbox"
	}

	var results []string
	for _, email := range emails {
		response, deleted, err := t.processSingleEmail(email)
		if err != nil {
			results = append(results, fmt.Sprintf("❌ Error processing email '%s': %v", email.Subject, err))
			continue
		}
		if deleted {
			results = append(results, fmt.Sprintf("✅ Responded to: '%s' from %s - Email deleted after response", email.Subject, email.From))
		} else {
			results = append(results, fmt.Sprintf("⚠️ No response sent for: '%s'", email.Subject))
		}
		results = append(results, response)
	}

	var sb strings.Builder
	sb.WriteString("📧 Email processing results:\n\n")
	for _, result := range results {
		sb.WriteString(result + "\n")
	}

	return sb.String()
}

// processSingleEmail processes one email: compose response and delete if appropriate
func (t *ToolExecutor) processSingleEmail(email EmailMessage) (string, bool, error) {
	// Check if this email needs a response (heuristic: short messages, questions, requests)
	if !emailShouldRespond(email) {
		return "ℹ️ No action needed", false, nil
	}

	// Compose response
	response := t.composeResponseToEmail(email)

	// Send the response
	sentMsg := t.sendEmail(map[string]any{
		"to":      email.From,
		"subject": fmt.Sprintf("Re: %s", email.Subject),
		"body":    response,
	})

	if !strings.HasPrefix(sentMsg, "✅ Email sent") {
		return fmt.Sprintf("❌ Failed to send response: %s", sentMsg), false, nil
	}

	// Try to delete the original email file
	emailDeleted := t.deleteEmailFile(email.Filename)

	if emailDeleted {
		return fmt.Sprintf("ℹ️ Auto-response sent:\n%s\n✓ Deleted from inbox", response), true, nil
	} else {
		return fmt.Sprintf("ℹ️ Auto-response sent:\n%s\n⚠ Email file not deleted (may already be in cur/)", response), true, nil
	}
}

// emailShouldRespond determines if an email needs a response based on content analysis
func emailShouldRespond(email EmailMessage) bool {
	// Always respond to emails that look like they need attention:
	// - Subject contains question marks or requests
	// - Body is relatively short (likely human communication, not system logs)
	// - From address looks like a real person

	subject := strings.ToLower(email.Subject)

	if strings.Contains(subject, "?") ||
		strings.Contains(subject, "please") ||
		strings.Contains(subject, "help") ||
		strings.Contains(subject, "need") ||
		strings.Contains(subject, "when") {
		return true
	}

	// Respond to emails under 5000 chars (likely human communication)
	if len(email.Content) < 5000 && email.From != "" {
		return true
	}

	return false
}

// getCurrentTime returns the current time for use in email responses
func (t *ToolExecutor) getCurrentTime() time.Time {
	return time.Now()
}

// composeResponseToEmail creates an appropriate response to the given email
func (t *ToolExecutor) composeResponseToEmail(email EmailMessage) string {
	now := t.getCurrentTime()

	response := fmt.Sprintf("Thank you for your message regarding '%s'.\n\n", email.Subject)
	response += fmt.Sprintf("YOLO received your email on %s.\n\n", now.Format(time.RFC1123))
	response += "I'm currently in autonomous operation mode. If this was a question or request:\n"
	response += "- I'll process it according to my current priorities\n"
	response += "- You should see activity/results within your normal monitoring windows\n\n"

	if email.From == "scott@stg.net" {
		response += "Since this is from Scott, I'll prioritize any task-related content.\n\n"
	}

	response += "Best regards,\nYOLO (Your Own Living Operator)"

	return response
}

// deleteEmailFile attempts to delete the email file from both new/ and cur/ directories
func (t *ToolExecutor) deleteEmailFile(filename string) bool {
	newDir := "/var/mail/b-haven.org/yolo/new/"
	curDir := "/var/mail/b-haven.org/yolo/cur/"

	newPath := filepath.Join(newDir, filename)
	curPath := filepath.Join(curDir, filename)

	// Try to delete from new/ first
	if _, err := os.Stat(newPath); err == nil {
		if err := os.Remove(newPath); err == nil {
			return true
		}
	}

	// Fall back to cur/
	if _, err := os.Stat(curPath); err == nil {
		if err := os.Remove(curPath); err == nil {
			return true
		}
	}

	return false
}
