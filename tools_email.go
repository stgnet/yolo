// Email Tool Implementation
// Allows YOLO to send emails via SMTP from yolo@b-haven.org

package main

import (
	"fmt"
	"strings"

	"yolo/email"
)

func (t *ToolExecutor) sendEmail(args map[string]any) string {
	// Get email configuration
	cfg := email.DefaultConfig()

	// Check if password is configured
	if cfg.Password == "" {
		return "Error: EMAIL_PASSWORD not configured. Set the environment variable or configure SMTP credentials."
	}

	subject := getStringArg(args, "subject", "")
	body := getStringArg(args, "body", "")
	to := getStringArg(args, "to", "")

	if subject == "" || body == "" {
		return "Error: subject and body parameters are required"
	}

	// If no recipient specified, use default (scott@stg.net)
	if to == "" {
		to = "scott@stg.net"
	}

	msg := &email.Message{
		To:      []string{to},
		Subject: subject,
		Body:    body,
	}

	err := email.Send(cfg, msg)
	if err != nil {
		return fmt.Sprintf("Error sending email: %v", err)
	}

	var sb strings.Builder
	sb.WriteString("✅ Email sent successfully\n")
	sb.WriteString(fmt.Sprintf("   To: %s\n", to))
	sb.WriteString(fmt.Sprintf("   From: %s\n", cfg.FromAddress))
	sb.WriteString(fmt.Sprintf("   Subject: %s\n", subject))
	return sb.String()
}

func (t *ToolExecutor) sendReport(args map[string]any) string {
	// Convenience function for sending progress reports
	cfg := email.DefaultConfig()

	if cfg.Password == "" {
		return "Error: EMAIL_PASSWORD not configured. Set the environment variable or configure SMTP credentials."
	}

	subject := getStringArg(args, "subject", "YOLO Progress Report")
	body := getStringArg(args, "body", "")

	if body == "" {
		return "Error: body parameter is required"
	}

	err := email.SendReport(cfg, subject, body)
	if err != nil {
		return fmt.Sprintf("Error sending report: %v", err)
	}

	var sb strings.Builder
	sb.WriteString("✅ Progress report sent successfully\n")
	sb.WriteString(fmt.Sprintf("   To: scott@stg.net\n"))
	sb.WriteString(fmt.Sprintf("   From: %s\n", cfg.FromAddress))
	sb.WriteString(fmt.Sprintf("   Subject: %s\n", subject))
	return sb.String()
}
