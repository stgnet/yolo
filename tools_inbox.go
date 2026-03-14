package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yolo/utils"
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
	bodyStarted := false
	var bodyLines []string

	for _, line := range lines {
		// Check for header fields (but skip the raw email envelope from maildir)
		if !bodyStarted && (strings.HasPrefix(line, "Date:") || strings.HasPrefix(line, "From:") ||
			strings.HasPrefix(line, "To:") || strings.HasPrefix(line, "Subject:") ||
			strings.HasPrefix(line, "Content-Type:") || strings.HasPrefix(line, "MIME-Version:")) {
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

			if strings.HasPrefix(line, "Content-Type:") && strings.Contains(line, "text/plain") {
				// This is the first content-type indicating body starts after blank line
			}
			continue
		}

		if line == "" || strings.HasPrefix(line, "--") {
			bodyStarted = true
			continue
		}

		if bodyStarted {
			bodyLines = append(bodyLines, line)
		}
	}

	email.Body = strings.Join(bodyLines, "\n")

	return email
}

// checkInbox reads emails from Maildir inbox at /var/mail/b-haven.org/yolo/new/
func (t *ToolExecutor) checkInbox(args map[string]any) string {
	markRead := getBoolArg(args, "mark_read", false)
	inboxPath := "/var/mail/b-haven.org/yolo/new/"

	files, err := utils.ListFiles(inboxPath)
	if err != nil {
		return fmt.Sprintf("❌ Error reading inbox: %v", err)
	}

	if len(files) == 0 {
		return "No new emails in inbox."
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("📬 Found %d new email(s).\n\n", len(files)))

	for _, filename := range files {
		filePath := fmt.Sprintf("%s%s", inboxPath, filename)
		emailContent, err := utils.ReadFile(filePath)
		if err != nil {
			output.WriteString(fmt.Sprintf("Error reading email %s: %v\n", filename, err))
			continue
		}

		parsedEmail := parseEmail(string(emailContent))

		// Display email summary
		output.WriteString(fmt.Sprintf("=== Email from: %s ===\n", parsedEmail.From))
		output.WriteString(fmt.Sprintf("Subject: %s\n", parsedEmail.Subject))

		if markRead {
			curPath := fmt.Sprintf("/var/mail/b-haven.org/yolo/cur/%s", filename)
			// Use no-backup config to avoid leaving .bak files in the inbox
			// directory, which would be picked up as new emails on the next run.
			noBackupConfig := utils.DefaultSafetyConfig()
			noBackupConfig.CreateBackup = false
			err = utils.MoveFileWithConfig(filePath, curPath, noBackupConfig)
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

	files, err := utils.ListFiles(inboxPath)
	if err != nil {
		return fmt.Sprintf("❌ Error reading inbox: %v", err)
	}

	if len(files) == 0 {
		return "No new emails to process."
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Processing %d email(s)...\n\n", len(files)))

	// Process each email with individual timeout handling
	for i, filename := range files {
		filePath := fmt.Sprintf("%s%s", inboxPath, filename)
		emailContent, err := utils.ReadFile(filePath)
		if err != nil {
			output.WriteString(fmt.Sprintf("Error reading email %s: %v\n\n", filename, err))
			continue
		}

		parsedEmail := parseEmail(string(emailContent))

		output.WriteString(fmt.Sprintf("--- Email %d/%d from: %s ---\n", i+1, len(files), parsedEmail.From))
		output.WriteString(fmt.Sprintf("Subject: %s\n\n", parsedEmail.Subject))

		// Skip emails from system/automated senders where a reply is pointless
		fromLower := strings.ToLower(parsedEmail.From)
		if strings.Contains(fromLower, "mailer-daemon") ||
			strings.Contains(fromLower, "postmaster@") ||
			strings.Contains(fromLower, "noreply@") ||
			strings.Contains(fromLower, "no-reply@") {
			output.WriteString("⏭ Skipping automated/system email — no reply needed\n")
			// Delete the message so it doesn't get reprocessed
			noBackupConfig := utils.DefaultSafetyConfig()
			noBackupConfig.CreateBackup = false
			if err := utils.DeleteFileWithConfig(filePath, noBackupConfig); err != nil {
				output.WriteString(fmt.Sprintf("Warning: Could not delete skipped email: %v\n", err))
			} else {
				output.WriteString("✓ Skipped email removed\n")
			}
			output.WriteString("---\n\n")
			continue
		}

		// Generate response using LLM with per-email timeout
		type emailResponseResult struct {
			response string
			err      error
		}
		var response string
		done := make(chan emailResponseResult, 1)
		go func() {
			resp := composeResponseToEmail(parsedEmail.Body, parsedEmail.Subject, parsedEmail.From)
			done <- emailResponseResult{response: resp, err: nil}
		}()

		select {
		case r := <-done:
			response = r.response
		case <-time.After(15 * time.Minute):
			output.WriteString(fmt.Sprintf("⚠️ Warning: Email response generation timed out after 15 minutes\n"))
			// Move to cur/ so it's not reprocessed every cycle
			curPath := fmt.Sprintf("/var/mail/b-haven.org/yolo/cur/%s", filename)
			noBackupCfg := utils.DefaultSafetyConfig()
			noBackupCfg.CreateBackup = false
			if mvErr := utils.MoveFileWithConfig(filePath, curPath, noBackupCfg); mvErr != nil {
				output.WriteString(fmt.Sprintf("⚠️ Could not move to cur/: %v — email left in inbox\n", mvErr))
			} else {
				output.WriteString("⚠️ Email moved to cur/ for retry.\n")
			}
			output.WriteString("---\n\n")
			continue
		}

		if strings.HasPrefix(response, "[Error generating response:") {
			output.WriteString(fmt.Sprintf("⚠️ Warning: Failed to generate response for this email\n"))
			// Move to cur/ so it's not reprocessed every cycle
			curPath := fmt.Sprintf("/var/mail/b-haven.org/yolo/cur/%s", filename)
			noBackupCfg := utils.DefaultSafetyConfig()
			noBackupCfg.CreateBackup = false
			if mvErr := utils.MoveFileWithConfig(filePath, curPath, noBackupCfg); mvErr != nil {
				output.WriteString(fmt.Sprintf("⚠️ Could not move to cur/: %v — email left in inbox\n", mvErr))
			} else {
				output.WriteString("⚠️ Email moved to cur/ for retry.\n")
			}
			output.WriteString("---\n\n")
			continue
		}

		if response == "" {
			output.WriteString("Warning: Empty response generated, skipping send.\n\n")
			continue
		}

		// LLM can signal that no reply should be sent by returning "NO_REPLY"
		if strings.TrimSpace(response) == "NO_REPLY" {
			output.WriteString("⏭ LLM indicated no reply needed for this email\n")
			// Delete the message so it doesn't get reprocessed
			noBackupConfig := utils.DefaultSafetyConfig()
			noBackupConfig.CreateBackup = false
			if err := utils.DeleteFileWithConfig(filePath, noBackupConfig); err != nil {
				output.WriteString(fmt.Sprintf("Warning: Could not delete skipped email: %v\n", err))
			} else {
				output.WriteString("✓ Skipped email removed\n")
			}
			output.WriteString("---\n\n")
			continue
		}

		output.WriteString(fmt.Sprintf("Generated response preview:\n%s\n", limitString(response, 200)))

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
			// Move failed email to cur/ so it's not reprocessed every cycle,
			// but preserved for manual retry or inspection.
			curPath := fmt.Sprintf("/var/mail/b-haven.org/yolo/cur/%s", filename)
			noBackupConfig := utils.DefaultSafetyConfig()
			noBackupConfig.CreateBackup = false
			if mvErr := utils.MoveFileWithConfig(filePath, curPath, noBackupConfig); mvErr != nil {
				output.WriteString(fmt.Sprintf("⚠️ Could not move to cur/: %v — email left in inbox\n", mvErr))
			} else {
				output.WriteString("⚠️ Email moved to cur/ for retry.\n")
			}
		} else {
			output.WriteString("✓ Response sent successfully\n")

			// Only delete original email after response was sent successfully
			// Use no-backup config to avoid creating .bak files in the inbox
			// directory, which would be picked up as new emails on the next run.
			noBackupConfig := utils.DefaultSafetyConfig()
			noBackupConfig.CreateBackup = false
			err = utils.DeleteFileWithConfig(filePath, noBackupConfig)
			if err != nil {
				output.WriteString(fmt.Sprintf("Warning: Could not delete processed email: %v\n", err))
			} else {
				output.WriteString("✓ Original email removed\n")
			}
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

CANCELLING A REPLY:
- If the email is from a system address (e.g. MAILER-DAEMON, postmaster, noreply) or is an
  automated bounce/notification where a reply would be pointless, respond with EXACTLY the
  text "NO_REPLY" (nothing else) to indicate no reply should be sent.

Write your email response now:`, from, subject, currentDateTime, subject, body)

	// Generate response using LLM directly without timeout
	return strings.TrimSpace(llmResponseGenerator(prompt))
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

	result, err := client.Chat(context.Background(), "qwen3.5:27b", messages, nil, nil)
	if err != nil {
		return fmt.Sprintf("[Error generating response: %v]", err)
	}

	return result.ContentText
}
