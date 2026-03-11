// Email Tool Implementation
// Allows YOLO to send emails via SMTP from yolo@b-haven.org

package main

import (
	"fmt"
	"strings"

	"yolo/email"
)

func (t *ToolExecutor) sendEmail(args map[string]any) string {
	// Get email configuration (uses local SMTP relay by default, no auth needed)
	cfg := email.DefaultConfig()

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

	client := email.New(cfg)
	err := client.Send(msg)
	if err != nil {
		return fmt.Sprintf("Error sending email: %v", err)
	}

	var sb strings.Builder
	sb.WriteString("✅ Email sent successfully\n")
	sb.WriteString(fmt.Sprintf("   To: %s\n", to))
	sb.WriteString(fmt.Sprintf("   From: yolo@b-haven.org\n"))
	sb.WriteString(fmt.Sprintf("   Subject: %s\n", subject))
	return sb.String()
}

func (t *ToolExecutor) sendReport(args map[string]any) string {
	// Convenience function for sending progress reports
	cfg := email.DefaultConfig()

	subject := getStringArg(args, "subject", "YOLO Progress Report")
	body := getStringArg(args, "body", "")

	if body == "" {
		return "Error: body parameter is required"
	}

	msg := &email.Message{
		To:      []string{"scott@stg.net"},
		Subject: subject,
		Body:    body,
	}

	client := email.New(cfg)
	err := client.Send(msg)
	if err != nil {
		return fmt.Sprintf("Error sending report: %v", err)
	}

	var sb strings.Builder
	sb.WriteString("✅ Progress report sent successfully\n")
	sb.WriteString(fmt.Sprintf("   To: scott@stg.net\n"))
	sb.WriteString(fmt.Sprintf("   From: yolo@b-haven.org\n"))
	sb.WriteString(fmt.Sprintf("   Subject: %s\n", subject))
	return sb.String()
}
