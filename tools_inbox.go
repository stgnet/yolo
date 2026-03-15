package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"mime/multipart"
	"net/mail"
	"strconv"
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

// decodeMIMEHeader decodes MIME encoded words in header fields using mime.WordDecoder
func decodeMIMEHeader(raw string) string {
	decoder := &mime.WordDecoder{}
	decoded, err := decoder.DecodeString(raw)
	if err != nil {
		return raw
	}
	return decoded
}

// parseEmail parses raw email content using Go's net/mail package for proper MIME handling
func parseEmail(raw string) Email {
	email := Email{Raw: raw}

	// First, try to use net/mail to parse the headers
	msg, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		// If parsing fails, fall back to simple string splitting
		return parseEmailFallback(raw)
	}

	// Extract From header with proper address parsing and MIME decoding
	fromAddr := msg.Header.Get("From")
	if fromAddr != "" {
		addr, err := mail.ParseAddress(fromAddr)
		if err == nil {
			// Use the display name if available (it will be decoded by ParseAddress)
			email.From = addr.Address
		} else {
			// Fallback: decode any MIME encoding in the raw address
			email.From = decodeMIMEHeader(strings.TrimSpace(fromAddr))
		}
	}

	// Extract To header with MIME decoding
	toAddr := msg.Header.Get("To")
	if toAddr != "" {
		email.To = decodeMIMEHeader(strings.TrimSpace(toAddr))
	}

	// Extract Subject header with MIME decoding
	subject := msg.Header.Get("Subject")
	if subject != "" {
		email.Subject = decodeMIMEHeader(strings.TrimSpace(subject))
	}

	// Extract body content - handle multipart emails properly
	contentType, _, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err == nil && contentType == "multipart" {
		// Handle multipart email (could contain both text and attachments)
		boundary := msg.Header.Get("Content-Type")
		if idx := strings.Index(boundary, "boundary="); idx != -1 {
			boundary = boundary[idx+9:] // Skip "boundary=" prefix
			// Remove quotes and trailing semicolon
			boundary = strings.Trim(boundary, "\"; \t")
		} else if idx := strings.Index(contentType, ";"); idx != -1 {
			// Try to extract boundary from Content-Type header value
			contentTypeStr := msg.Header.Get("Content-Type")
			if mediaType, params, parseErr := mime.ParseMediaType(contentTypeStr); parseErr == nil {
				boundary = params["boundary"]
			}
		}
		reader := multipart.NewReader(msg.Body, boundary)

		var bodyBuilder strings.Builder
		hasTextPlain := false
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}

			partContentType := part.Header.Get("Content-Type")
			contentTypeStr, _, _ := mime.ParseMediaType(partContentType)

			// Prefer text/plain over text/html for the main body
			if contentTypeStr == "text/plain" && !hasTextPlain {
				hasTextPlain = true
				partBody, err := io.ReadAll(part)
				if err == nil {
					bodyBuilder.WriteString(string(partBody))
				}
			} else if contentTypeStr == "text/html" && bodyBuilder.Len() == 0 {
				// No text/plain found, use text/html as fallback
				partBody, err := io.ReadAll(part)
				if err == nil {
					bodyBuilder.WriteString(string(partBody))
				}
			}
		}

		email.Body = strings.TrimSpace(bodyBuilder.String())
	} else if contentType == "text/plain" || contentType == "text/html" {
		// Handle single-part email with proper encoding detection
		contentTypeValue := msg.Header.Get("Content-Type")
		if idx := strings.Index(contentTypeValue, ";"); idx != -1 {
			params, _, parseErr := mime.ParseMediaType(contentTypeValue)
			if parseErr == nil {
				// Check for charset and transfer encoding
				if enc := params["charset"]; enc != "" {
					// Charset is specified, the body should be UTF-8 or the specified charset
				}

				transferEnc := strings.ToLower(params["content-transfer-encoding"])
				switch transferEnc {
				case "quoted-printable", "qp":
					decodedReader := quotedprintable.NewReader(msg.Body)
					body, err := io.ReadAll(decodedReader)
					if err == nil {
						email.Body = strings.TrimSpace(string(body))
					} else {
						// If decoding fails, read raw body
						rawBody, _ := io.ReadAll(msg.Body)
						email.Body = strings.TrimSpace(string(rawBody))
					}
				case "base64":
					buf := new(bytes.Buffer)
					buf.ReadFrom(msg.Body)
					decodedBytes, err := base64.StdEncoding.DecodeString(buf.String())
					if err == nil {
						email.Body = strings.TrimSpace(string(decodedBytes))
					} else {
						rawBody, _ := io.ReadAll(msg.Body)
						email.Body = strings.TrimSpace(string(rawBody))
					}
				default:
					// Assume raw text, try quoted-printable anyway for safety
					decodedReader := quotedprintable.NewReader(msg.Body)
					body, err := io.ReadAll(decodedReader)
					if err == nil {
						email.Body = strings.TrimSpace(string(body))
					} else {
						rawBody, _ := io.ReadAll(msg.Body)
						email.Body = strings.TrimSpace(string(rawBody))
					}
				}
			}
		}

		// If body is still empty, try raw read with fallback
		if email.Body == "" {
			msg.Body.Close() // Reset for re-read attempt
			body, _ := io.ReadAll(msg.Body)
			email.Body = strings.TrimSpace(string(body))
		}
	} else {
		// Unknown content type, try to read whatever we can get
		var buf bytes.Buffer
		io.Copy(&buf, msg.Body)
		email.Body = strings.TrimSpace(buf.String())
	}

	return email
}

// parseMultipartBoundary extracts the boundary parameter from Content-Type header
func parseMultipartBoundary(contentType string) string {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	if !strings.HasPrefix(mediaType, "multipart/") {
		return ""
	}
	return params["boundary"]
}

// parseEmailFallback is a simple fallback parser when net/mail fails
func parseEmailFallback(raw string) Email {
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

		// Generate response using the full agent loop (with tools, history, system prompt)
		type emailResponseResult struct {
			response string
			err      error
		}
		var response string
		done := make(chan emailResponseResult, 1)
		go func() {
			resp := t.composeEmailWithAgent(parsedEmail.Body, parsedEmail.Subject, parsedEmail.From)
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

// llmResponseGenerator is a function type for generating LLM responses.
// This allows tests to inject mock generators without needing actual Ollama.
// When set to a non-nil custom function, composeEmailWithAgent falls back to
// the simple single-turn path for testability.
var llmResponseGenerator func(prompt string) string

// MaxEmailRounds caps the number of tool-calling iterations for email responses.
const MaxEmailRounds = 10

// composeEmailWithAgent generates a response to an incoming email using the
// full agent capabilities: system prompt, conversation history, configured
// model, and tool access (including todos, file reads, web search, etc.).
// This ensures email replies have the same knowledge as the console agent.
func (t *ToolExecutor) composeEmailWithAgent(body, subject, from string) string {
	if body == "" {
		body = "No content"
	}

	// If a test mock is installed, use the simple single-turn path
	if llmResponseGenerator != nil {
		return strings.TrimSpace(llmResponseGenerator(composeEmailPrompt(body, subject, from)))
	}

	agent := t.agent
	if agent == nil {
		// Fallback: no agent available (shouldn't happen in production)
		return composeResponseToEmailLegacy(body, subject, from)
	}

	currentDateTime := time.Now().Format("January 2, 2006 at 3:04 PM MST")

	// Build messages: system prompt (with knowledge base + todos) + recent
	// history for context + the email as a user message.
	systemPrompt := agent.getSystemPrompt()

	emailInstruction := fmt.Sprintf(`You have received an email that you need to reply to.

INCOMING EMAIL:
- From: %s
- Subject: %s
- Date/Time: %s

EMAIL BODY:
%s

INSTRUCTIONS:
1. Use your tools to look up any information needed to answer the email accurately.
   For example, if they ask about your todo list, use the list_todos tool.
   If they ask about files or code, use read_file or search_files.
2. Write a professional yet conversational email reply that directly answers their questions.
3. Address the sender by name if available. Reference the subject/topic.
4. Do NOT use placeholders like [ACTION_NEEDED]. Write complete, specific answers.
5. If the email is from a system address (MAILER-DAEMON, postmaster, noreply) or is an
   automated notification, respond with EXACTLY "NO_REPLY".
6. Your final response (after any tool calls) should be ONLY the email reply text — no
   explanations, no markdown headers, no "Here is the reply:" prefix. Just the reply itself.`, from, subject, currentDateTime, body)

	msgs := []ChatMessage{
		{Role: "system", Content: systemPrompt},
	}
	// Include recent conversation history so the email responder knows what
	// has been discussed on the console.
	msgs = append(msgs, agent.history.GetContextMessages(MaxContextMessages)...)
	msgs = append(msgs, ChatMessage{Role: "user", Content: emailInstruction})

	emailTools := EmailTools()
	model := agent.config.GetModel()

	// Tool-calling loop (mirrors subagent pattern from agent.go)
	var roundMsgs []ChatMessage
	var finalText string

	for round := 0; round < MaxEmailRounds; round++ {
		allMsgs := make([]ChatMessage, 0, len(msgs)+len(roundMsgs))
		allMsgs = append(allMsgs, msgs...)
		allMsgs = append(allMsgs, roundMsgs...)

		result, err := agent.ollama.Chat(context.Background(), model, allMsgs, emailTools, nil)
		if err != nil {
			return fmt.Sprintf("[Error generating response: %v]", err)
		}

		toolCalls := result.ToolCalls
		if len(toolCalls) == 0 {
			toolCalls = agent.parseTextToolCalls(result.DisplayText)
		}
		toolCalls = deduplicateToolCalls(toolCalls)

		if len(toolCalls) == 0 {
			finalText = result.DisplayText
			break
		}

		// Build assistant message with tool_calls
		var nativeTCs []ToolCall
		for i, tc := range toolCalls {
			argsJSON, _ := json.Marshal(tc.Args)
			nativeTCs = append(nativeTCs, ToolCall{
				ID: fmt.Sprintf("email_call_%d_%d", round, i),
				Function: ToolCallFunc{
					Name:      tc.Name,
					Arguments: json.RawMessage(argsJSON),
				},
			})
		}
		roundMsgs = append(roundMsgs, ChatMessage{
			Role:      "assistant",
			Content:   result.ContentText,
			ToolCalls: nativeTCs,
		})

		// Execute each tool — abort remaining if a file-mutation tool fails
		inboxFileMutationFailed := false
		for i, call := range toolCalls {
			args := call.Args
			if args == nil {
				args = map[string]any{}
			}

			if inboxFileMutationFailed {
				abortMsg := fmt.Sprintf("Error: skipped — a prior file operation failed. "+
					"Review earlier errors before retrying this tool call (%s).", call.Name)
				roundMsgs = append(roundMsgs, ChatMessage{
					Role:       "tool",
					Content:    abortMsg,
					ToolCallID: fmt.Sprintf("email_call_%d_%d", round, i),
				})
				continue
			}

			resultStr := executeWithTimeout(t, call.Name, args)
			cleanResult := filterToolActivityMarkers(resultStr)
			roundMsgs = append(roundMsgs, ChatMessage{
				Role:       "tool",
				Content:    cleanResult,
				ToolCallID: fmt.Sprintf("email_call_%d_%d", round, i),
			})

			if strings.HasPrefix(resultStr, "Error: ") && isFileMutationTool(call.Name) {
				inboxFileMutationFailed = true
			}
		}
	}

	if finalText == "" {
		finalText = "(no response generated)"
	}

	return strings.TrimSpace(finalText)
}

// composeEmailPrompt builds the email prompt string for the simple single-turn path.
func composeEmailPrompt(body, subject, from string) string {
	currentDateTime := time.Now().Format("January 2, 2006 at 3:04 PM MST")
	return fmt.Sprintf(`You are YOLO, an autonomous AI assistant running on a Mac.
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
}

// composeResponseToEmail generates a response using the simple single-turn path.
// Kept for backward compatibility; production code uses composeEmailWithAgent.
func composeResponseToEmail(body, subject, from string) string {
	if body == "" {
		body = "No content"
	}
	prompt := composeEmailPrompt(body, subject, from)
	if llmResponseGenerator != nil {
		return strings.TrimSpace(llmResponseGenerator(prompt))
	}
	return strings.TrimSpace(composeResponseToEmailLegacy(body, subject, from))
}

// composeResponseToEmailLegacy is the original single-turn LLM call without tools.
func composeResponseToEmailLegacy(body, subject, from string) string {
	if body == "" {
		body = "No content"
	}
	prompt := composeEmailPrompt(body, subject, from)
	return strings.TrimSpace(generateLLMText(prompt))
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
