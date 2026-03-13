package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// Email represents a parsed email message
type Email struct {
	From    string
	To      string
	Subject string
	Body    string
	Raw     string
}

// parseEmail extracts fields from raw email text
func parseEmail(raw string) Email {
	email := Email{Raw: raw}

	lines := strings.Split(raw, "\n")
	inBody := false
	var bodyLines []string

	for _, line := range lines {
		if inBody {
			bodyLines = append(bodyLines, line)
			continue
		}

		if strings.HasPrefix(line, "From: ") || strings.HasPrefix(line, "From\t") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				email.From = strings.TrimSpace(parts[1])
			} else {
				parts = strings.SplitN(line, "\t", 2)
				if len(parts) == 2 {
					email.From = strings.TrimSpace(parts[1])
				}
			}
		}

		if strings.HasPrefix(line, "To: ") || strings.HasPrefix(line, "To\t") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				email.To = strings.TrimSpace(parts[1])
			} else {
				parts = strings.SplitN(line, "\t", 2)
				if len(parts) == 2 {
					email.To = strings.TrimSpace(parts[1])
				}
			}
		}

		if strings.HasPrefix(line, "Subject: ") || strings.HasPrefix(line, "Subject\t") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				email.Subject = strings.TrimSpace(parts[1])
			} else {
				parts = strings.SplitN(line, "\t", 2)
				if len(parts) == 2 {
					email.Subject = strings.TrimSpace(parts[1])
				}
			}
		}

		if line == "" || strings.HasPrefix(line, "--") {
			inBody = true
		}
	}

	email.Body = strings.Join(bodyLines, "\n")
	return email
}

// checkInbox reads emails from Maildir inbox at /var/mail/b-haven.org/yolo/new/
func (t *ToolExecutor) checkInbox(args map[string]any) string {
	markRead := getBoolArg(args, "mark_read", false)
	inboxPath := "/var/mail/b-haven.org/yolo/new/"

	files, err := os.ReadDir(inboxPath)
	if err != nil {
		return fmt.Sprintf("❌ Error reading inbox: %v", err)
	}

	if len(files) == 0 {
		return "No new emails in inbox."
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("📬 Found %d new email(s).\n\n", len(files)))

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := fmt.Sprintf("%s%s", inboxPath, file.Name())
		emailContent, err := os.ReadFile(filePath)
		if err != nil {
			output.WriteString(fmt.Sprintf("Error reading email %s: %v\n", file.Name(), err))
			continue
		}

		parsedEmail := parseEmail(string(emailContent))

		// Display email summary
		output.WriteString(fmt.Sprintf("=== Email from: %s ===\n", parsedEmail.From))
		output.WriteString(fmt.Sprintf("Subject: %s\n", parsedEmail.Subject))

		if markRead {
			curPath := fmt.Sprintf("/var/mail/b-haven.org/yolo/cur/%s", file.Name())
			err = os.Rename(filePath, curPath)
			if err != nil {
				output.WriteString(fmt.Sprintf("Error marking email as read: %v\n", err))
			}
		}
		output.WriteString("\n")
	}

	return output.String()
}

// processInboxWithResponse handles the complete email workflow: read, respond, delete
func (t *ToolExecutor) processInboxWithResponse(args map[string]any) string {
	inboxPath := "/var/mail/b-haven.org/yolo/new/"

	files, err := os.ReadDir(inboxPath)
	if err != nil {
		return fmt.Sprintf("❌ Error reading inbox: %v", err)
	}

	if len(files) == 0 {
		return "No new emails to process."
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Processing %d email(s)...\n\n", len(files)))

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := fmt.Sprintf("%s%s", inboxPath, file.Name())
		emailContent, err := os.ReadFile(filePath)
		if err != nil {
			output.WriteString(fmt.Sprintf("Error reading email %s: %v\n\n", file.Name(), err))
			continue
		}

		parsedEmail := parseEmail(string(emailContent))

		output.WriteString(fmt.Sprintf("--- Processing email from: %s ---\n", parsedEmail.From))
		output.WriteString(fmt.Sprintf("Subject: %s\n\n", parsedEmail.Subject))

		// Generate response using LLM directly (no pattern matching)
		response := composeResponseToEmail(parsedEmail.Body, parsedEmail.Subject, parsedEmail.From)

		if response == "" {
			output.WriteString("Warning: Empty response generated, skipping send.\n\n")
			continue
		}

		output.WriteString(fmt.Sprintf("Generated response:\n%s\n", response))

		// Prepare send_email args
		emailArgs := map[string]any{
			"subject": "Re: " + parsedEmail.Subject,
			"body":    response,
			"to":      parsedEmail.From,
		}

		// Send the response using existing sendEmail method
		result := t.sendEmail(emailArgs)
		if strings.HasPrefix(result, "Error:") {
			output.WriteString(fmt.Sprintf("❌ Error sending response: %s\n", result))
		} else {
			output.WriteString("✓ Response sent successfully\n")
		}

		// Delete original email from inbox (move to trash or remove)
		err = os.Remove(filePath)
		if err != nil {
			output.WriteString(fmt.Sprintf("Warning: Could not delete processed email: %v\n", err))
		} else {
			output.WriteString("✓ Original email removed\n")
		}

		output.WriteString("---\n\n")
	}

	return output.String()
}

// llmResponseGenerator is a function type for generating LLM responses
// This allows tests to inject mock generators without needing actual Ollama
var llmResponseGenerator = func(prompt string) string {
	return generateLLMText(prompt)
}

// composeResponseToEmail generates a personalized response to an incoming email using LLM directly
// Includes email metadata (subject, timestamp, sender info) and thread context for professional responses
func composeResponseToEmail(body, subject, from string) string {
	if body == "" {
		body = "No content"
	}

	// Get current date/time for reference in response
	currentDateTime := time.Now().Format("January 2, 2006 at 3:04 PM MST")

	prompt := fmt.Sprintf(`You are YOLO, an autonomous AI assistant running on a Mac. 
Your job is to reply to emails in a professional, personalized manner with proper context.

INCOMING EMAIL CONTEXT:
- Sender: %s
- Subject: %s
- Received: (original email)
- Reply Date/Time: %s

THREAD/TOPIC BEING DISCUSSED:
%s

EMAIL BODY CONTENT:
%s

RESPONSE GUIDELINES:
1. ACKNOWLEDGE THE SENDER - Address them by name if available in their address, or reference who they are
2. REFERENCE THE ORIGINAL SUBJECT - Include context about what conversation/thread this is part of
3. INCLUDE EMAIL METADATA - Reference the original subject line and acknowledge when their email was received
4. BE PROFESSIONAL YET CONVERSATIONAL - Use a friendly but polished tone appropriate for email correspondence
5. ANSWER SPECIFICALLY - Address each point/question they raised directly without generic responses
6. PROVIDE CONTEXT AWARENESS - Show you understand the thread/topic being discussed and relate to it
7. NO PLACEHOLDERS - Write complete, actionable responses with no [ACTION_NEEDED] or similar markers

RESPONSE FORMAT:
- Start with an appropriate greeting referencing the sender
- Acknowledge their message and reference the subject/topic
- Provide specific answers or actions taken
- Close professionally with context about next steps if applicable

Write your email response now:`, from, subject, currentDateTime, subject, body)

	response := llmResponseGenerator(prompt)

	return strings.TrimSpace(response)
}

// limitString truncates string to maxLen chars, adding "..." if truncated
func limitString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// generateLLMText sends a prompt to Ollama and returns the text response
func generateLLMText(prompt string) string {
	client := NewOllamaClient("http://localhost:11434")

	messages := []ChatMessage{
		{Role: "user", Content: prompt},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := client.Chat(ctx, "qwen3.5:27b", messages, nil, nil)
	if err != nil {
		return fmt.Sprintf("[Error generating response: %v]", err)
	}

	return result.ContentText
}
