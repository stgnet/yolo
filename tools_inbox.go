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
)

type EmailMessage struct {
	From        string   `json:"from"`
	Subject     string   `json:"subject"`
	Date        string   `json:"date"`
	Content     string   `json:"content"`
	Filename    string   `json:"filename"`
	ContentType string   `json:"content_type"`
	Size        int64    `json:"size"`
	To          []string `json:"to,omitempty"`
}

type CheckInboxResult struct {
	Emails    []EmailMessage `json:"emails"`
	Count     int            `json:"count"`
	Processed int            `json:"processed"`
	Message   string         `json:"message,omitempty"`
}

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

func extractBodyFromBytes(data []byte, contentType string) string {
	reader := bytes.NewReader(data)
	return extractBody(reader, contentType)
}

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
