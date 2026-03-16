// Email Inbox Processing Tools
// Implements check_inbox and process_inbox_with_response for autonomous email handling

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	InboxPath = "/var/mail/b-haven.org/yolo/new/"
	CurDir    = "cur"
)

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

// parseEmail extracts relevant fields from email content
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
			email.From = strings.TrimSpace(strings.TrimPrefix(line, "From:"))
		} else if strings.HasPrefix(line, "Subject:") {
			email.Subject = strings.TrimSpace(strings.TrimPrefix(line, "Subject:"))
		} else if strings.HasPrefix(line, "Date:") {
			email.Date = strings.TrimSpace(strings.TrimPrefix(line, "Date:"))
		} else if strings.HasPrefix(line, "To:") {
			toAddresses := strings.Split(strings.TrimSpace(strings.TrimPrefix(line, "To:")), ",")
			for i, addr := range toAddresses {
				toAddresses[i] = strings.TrimSpace(addr)
			}
			email.To = toAddresses
		}
	}

	return email
}

// isBounceMessage checks if an email appears to be a bounce/delivery failure notification
func isBounceMessage(email *EmailMessage) bool {
	bounceIndicators := []string{
		"delivery failed", "undeliverable", "bounce", "returned mail",
		"permanent failure", "temporary failure", "postmaster@",
		"mailer-daemon@", "failure notice", "notification system",
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

// processInboxWithResponse automates the email workflow: read → respond → delete
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
			continue
		}

		// Generate auto-response based on email content
		response := generateAIResponse(&email)

		// Send response back to sender
		sender := email.From
		if strings.Contains(sender, "<") {
			parts := strings.SplitN(sender, "<", 2)
			sender = strings.TrimSpace(strings.TrimSuffix(parts[0], ">"))
		}

		subjectResp := fmt.Sprintf("Re: %s", email.Subject)
		t.sendEmail(map[string]any{
			"to":      sender,
			"subject": subjectResp,
			"body":    response,
		})

		// Delete original email
		os.Remove(filePath)
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
		sb.WriteString(fmt.Sprintf("🗑️  Deleted: %s\n", strings.Join(processed, ", ")))
	}
	if len(skipped) > 0 {
		sb.WriteString(fmt.Sprintf("⚠️  Skipped %d potential bounce messages:\n", totalSkipped))
		for _, name := range skipped {
			sb.WriteString(fmt.Sprintf("    - %s\n", name))
		}
	}
	if len(processed) > 0 {
		sb.WriteString("📧 Sent auto-responses to all senders")
	}

	return sb.String()
}

// generateAIResponse creates an intelligent auto-response based on email content
func generateAIResponse(email *EmailMessage) string {
	var sb strings.Builder

	// Personalized greeting
	if email.From != "" {
		sb.WriteString(fmt.Sprintf("Hello %s,\n\n", truncateString(email.From, 50)))
	} else {
		sb.WriteString("Hello,\n\n")
	}

	// Acknowledge receipt and provide contextual response
	sb.WriteString(fmt.Sprintf("I've received your message regarding \"%s\".\n", email.Subject))
	sb.WriteString("\n")

	// Provide appropriate auto-response based on content type
	if strings.Contains(strings.ToLower(email.Content), "hello") ||
		strings.Contains(strings.ToLower(email.Content), "hi") {
		sb.WriteString("Thank you for reaching out! I'm YOLO, your autonomous AI agent.\n")
		sb.WriteString("I'm here to help with software development tasks and project automation.\n")
	} else if strings.Contains(strings.ToLower(email.Subject), "status") ||
		strings.Contains(strings.ToLower(email.Content), "status") {
		sb.WriteString("I can provide status updates on my current activities.\n")
		sb.WriteString("Would you like me to send a detailed progress report?\n")
	} else if strings.Contains(strings.ToLower(email.Subject), "todo") ||
		strings.Contains(strings.ToLower(email.Content), "todo") {
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
