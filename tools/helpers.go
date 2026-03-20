// Package tools provides helper functions for tool implementations
package tools

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-message/mail"
	"github.com/scottstg/yolo/tools/todo"
)

// EmailMessage represents a parsed email from the inbox
type EmailMessage struct {
	Filename    string
	From        string
	To          []string
	Subject     string
	Date        string
	ContentType string
	Content     string
	Size        int64
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

// truncateString safely truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// parseEmail extracts relevant fields from email content using proper MIME parsing
func parseEmail(content, filename string) EmailMessage {
	email := EmailMessage{
		Filename:    filename,
		ContentType: "text/plain",
		Size:        int64(len(content)),
	}

	// Parse the email as a MIME message using go-message library
	reader, err := mail.CreateReader(bytes.NewReader([]byte(content)))
	if err != nil {
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
	
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		// Get content type
		contentType := part.Header.Get("Content-Type")
		
		// Skip attachments (application, image types) and HTML if we can get plain text first
		if strings.HasPrefix(contentType, "application/") || 
		   strings.HasPrefix(contentType, "image/") {
			continue
		}

		if strings.Contains(contentType, "html") {
			// Skip HTML parts - prefer plain text
			continue
		}

		// Read the content of this part (Part has a Body field that is io.Reader)
		data, err := io.ReadAll(part.Body)
		if err != nil {
			continue
		}

		body.WriteString(string(data))
		
		// Stop after finding a text/plain body (first match wins)
		if strings.HasPrefix(contentType, "text/plain") {
			break
		}
	}

	// Set Content to just the body text, preserving newlines
	// DO NOT truncate - preserve full email content for proper LLM context and response generation
	email.Content = strings.TrimSpace(body.String())

	if email.Content == "" {
		simpleEmail := parseEmailSimple(content, filename)
		email.Content = simpleEmail.Content
	}

	return email
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

// Email-related helpers

func sendEmail(to, subject, body string) error {
	return sendEmailInternal(to, subject, body, nil)
}

// sendEmailInternal is the internal function that handles email sending with optional attachments
func sendEmailInternal(to, subject, body string, attachments []string) error {
	if to == "" {
		to = "scott@stg.net"
	}
	
	// If no attachments, use simple sendmail approach
	if len(attachments) == 0 {
		cmd := exec.Command("sendmail", "-t")
		input := fmt.Sprintf("To: %s\nFrom: yolo@b-haven.org\nSubject: %s\n\n%s\n", to, subject, body)
		cmd.Stdin = strings.NewReader(input)
		return cmd.Run()
	}
	
	// With attachments, create a proper MIME multipart message
	return sendEmailWithAttachments(to, subject, body, attachments)
}

// sendEmailWithAttachments creates a MIME multipart message with attachments
func sendEmailWithAttachments(to, subject, body string, attachments []string) error {
	if to == "" {
		to = "scott@stg.net"
	}

	// Create the email buffer
	var emailBuf bytes.Buffer
	
	// Write headers
	fmt.Fprintf(&emailBuf, "To: %s\n", to)
	fmt.Fprintf(&emailBuf, "From: yolo@b-haven.org\n")
	fmt.Fprintf(&emailBuf, "Subject: %s\n", subject)
	fmt.Fprintf(&emailBuf, "MIME-Version: 1.0\n")
	
	// Create a boundary for multipart message
	boundary := fmt.Sprintf("Boundary_%d", time.Now().UnixNano())
	fmt.Fprintf(&emailBuf, "Content-Type: multipart/mixed; boundary=\"%s\"\n\n", boundary)
	
	// Write the text part
	fmt.Fprintf(&emailBuf, "--%s\n", boundary)
	fmt.Fprintf(&emailBuf, "Content-Type: text/plain; charset=\"UTF-8\"\n")
	fmt.Fprintf(&emailBuf, "Content-Transfer-Encoding: 7bit\n\n")
	fmt.Fprint(&emailBuf, body)
	fmt.Fprintf(&emailBuf, "\n\n")
	
	// Write attachment parts
	for _, attachPath := range attachments {
		data, err := os.ReadFile(attachPath)
		if err != nil {
			// Skip attachment if can't read, but continue with others
			continue
		}
		
		filename := filepath.Base(attachPath)
		
		fmt.Fprintf(&emailBuf, "--%s\n", boundary)
		fmt.Fprintf(&emailBuf, "Content-Type: application/octet-stream; name=\"%s\"\n", filename)
		fmt.Fprintf(&emailBuf, "Content-Disposition: attachment; filename=\"%s\"\n", filename)
	fmt.Fprintf(&emailBuf, "Content-Transfer-Encoding: base64\n\n")
		
		// Encode to base64 with line breaks every 76 chars (RFC 2045)
		encoded := base64.StdEncoding.EncodeToString(data)
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			fmt.Fprintf(&emailBuf, "%s\n", encoded[i:end])
		}
		fmt.Fprintf(&emailBuf, "\n")
	}
	
	// Write closing boundary
	fmt.Fprintf(&emailBuf, "--%s--\n", boundary)
	
	// Send the message via sendmail
	cmd := exec.Command("sendmail", "-t")
	cmd.Stdin = &emailBuf
	return cmd.Run()
}

func sendReport(subject, body string) error {
	if subject == "" {
		subject = "YOLO Progress Report"
	}
	
	to := "scott@stg.net"
	return sendEmail(to, subject, body)
}

const (
	InboxPath = "/var/mail/b-haven.org/yolo/new/"
	CurDir    = "cur"
)

func checkInbox(markRead bool) ([]string, error) {
	entries, err := os.ReadDir(InboxPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read inbox: %w", err)
	}
	
	var emails []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		emailPath := filepath.Join(InboxPath, entry.Name())
		content, err := os.ReadFile(emailPath)
		if err != nil {
			continue
		}
		
		// Parse the email and extract just the relevant parts
		emailMsg := parseEmail(string(content), entry.Name())
		
		// Format email for display - show only parsed fields, not raw content
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("From: %s\n", emailMsg.From))
		sb.WriteString(fmt.Sprintf("To: %s\n", strings.Join(emailMsg.To, ", ")))
		sb.WriteString(fmt.Sprintf("Subject: %s\n", emailMsg.Subject))
		sb.WriteString(fmt.Sprintf("Date: %s\n", emailMsg.Date))
		sb.WriteString(fmt.Sprintf("Size: %d bytes\n", emailMsg.Size))
		sb.WriteString("\n--- Message Body ---\n")
		sb.WriteString(emailMsg.Content)
		
		emails = append(emails, sb.String())
		
		if markRead {
			// Move to cur directory
			curDir := "/var/mail/b-haven.org/yolo/cur/"
			os.MkdirAll(curDir, 0755)
			destPath := filepath.Join(curDir, entry.Name())
			
			data, _ := os.ReadFile(emailPath)
			os.WriteFile(destPath, data, 0644)
			os.Remove(emailPath)
		}
	}
	
	return emails, nil
}

func processInboxWithResponse() (int, int, error) {
	processed := 0
	skipped := 0
	
	emails, err := checkInbox(false)
	if err != nil {
		return 0, 0, err
	}
	
	for _, emailContent := range emails {
		// For each email, generate LLM response and send it
		// This is a simplified version - full implementation in main package
		_ = emailContent // Parse the content to check if body exists
		processed++
	}
	
	return processed, skipped, nil
}

// GOG helper

func executeGOG(command string) (string, error) {
	// Placeholder for Google API commands
	// In production, this would call actual Google APIs
	return fmt.Sprintf("GOG command executed: %s", command), nil
}

// Learning helpers

type Improvement struct {
	Title      string `json:"title"`
	Priority   string `json:"priority"`
	Category   string `json:"category"`
	Source     string `json:"source"`
	Descraption string `json:"description"`
}

func runLearning() ([]Improvement, error) {
	// Placeholder for learning implementation
	return []Improvement{
		{
			Title:      "Improve error handling in HTTP handlers",
			Priority:   "HIGH",
			Category:   "Code Quality",
			Source:     "Web search",
			Descraption: "Add proper error logging and user-friendly error messages",
		},
	}, nil
}

func implementImprovements(count int) (string, error) {
	// Placeholder for implementation logic
	return fmt.Sprintf("Implementation logic for %d improvements", count), nil
}

// Model helpers

func listOllamaModels() ([]string, error) {
	cmd := exec.Command("ollama", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	
	var models []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines[1:] { // Skip header
		parts := strings.Fields(line)
		if len(parts) > 0 {
			models = append(models, parts[0])
		}
	}
	
	return models, nil
}

func switchToModel(model string) error {
	// Update current model in config
	// This would call config.SetCurrentModel(model)
	return nil
}

// Todo helpers

func addTodoItem(title string) error {
	_, err := todo.GetGlobalTodoList().Add(title)
	return err
}

func completeTodoItem(title string) error {
	found, err := todo.GetGlobalTodoList().Complete(title)
	if !found {
		return fmt.Errorf("todo not found: %s", title)
	}
	return err
}

func deleteTodoItem(title string) error {
	found, err := todo.GetGlobalTodoList().Delete(title)
	if !found {
		return fmt.Errorf("todo not found: %s", title)
	}
	return err
}

func listAllTodos() ([]todo.Todo, error) {
	return todo.GetGlobalTodoList().GetAllTodos(), nil
}

func formatTodos(todos []todo.Todo) string {
	var sb strings.Builder
	
	pendingCount := 0
	completedCount := 0
	
	for _, t := range todos {
		if t.Completed {
			completedCount++
		} else {
			pendingCount++
		}
	}
	
	sb.WriteString(fmt.Sprintf("Total: %d pending, %d completed\n\n", pendingCount, completedCount))
	
	if pendingCount > 0 {
		sb.WriteString("--- PENDING ---\n")
		for _, t := range todos {
			if !t.Completed {
				sb.WriteString(fmt.Sprintf("- [ ] %s (created: %s)\n", 
					t.Title, t.CreatedAt.Format("Jan 2, 2006 3:04PM")))
			}
		}
		sb.WriteString("\n")
	}
	
	if completedCount > 0 {
		sb.WriteString("--- COMPLETED ---\n")
		for _, t := range todos {
			if t.Completed {
				sb.WriteString(fmt.Sprintf("- [x] %s (created: %s, updated: %s)\n", 
					t.Title, t.CreatedAt.Format("Jan 2, 2006 3:04PM"), t.UpdatedAt.Format("Jan 2, 2006 3:04PM")))
			}
		}
	}
	
	return sb.String()
}

// Helper function for running send_command with buffering
func runSendmailCommand(to, subject, body string) error {
	cmd := exec.Command("sendmail", "-t")
	
	var stdin bytes.Buffer
	fmt.Fprintf(&stdin, "To: %s\n", to)
	fmt.Fprintf(&stdin, "From: yolo@b-haven.org\n")
	fmt.Fprintf(&stdin, "Subject: %s\n", subject)
	fmt.Fprintln(&stdin)
	fmt.Fprintln(&stdin, body)
	
	cmd.Stdin = &stdin
	return cmd.Run()
}
