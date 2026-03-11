// Email Tool Implementation
// Allows YOLO to send emails via SMTP from yolo@b-haven.org

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"yolo/email"
)

const (
	emailCooldownFile   = ".yolo_email_cooldown.txt"
	emailCooldownPeriod = time.Hour * 2 // Minimum 2 hours between emails in autonomous mode
)

// checkEmailCooldown checks if enough time has passed since the last email
// Returns true if allowed to send, false if on cooldown
func checkEmailCooldown() bool {
	cooldownPath := "." + string(os.PathSeparator) + emailCooldownFile

	data, err := os.ReadFile(cooldownPath)
	if err != nil {
		// No cooldown file exists, allow sending
		return true
	}

	var unixTime int64
	_, err = fmt.Sscanf(string(data), "%d", &unixTime)
	if err != nil {
		// Invalid data, allow sending
		return true
	}

	lastSend := time.Unix(unixTime, 0)

	if time.Since(lastSend) < emailCooldownPeriod {
		return false
	}

	return true
}

// recordEmailSent records the current time as the last email sent
func recordEmailSent() error {
	cooldownPath := "." + string(os.PathSeparator) + emailCooldownFile

	now := time.Now().Unix()
	data := fmt.Sprintf("%d", now)
	return os.WriteFile(cooldownPath, []byte(data), 0644)
}

func (t *ToolExecutor) sendEmail(args map[string]any) string {
	subject := getStringArg(args, "subject", "")
	body := getStringArg(args, "body", "")
	to := getStringArg(args, "to", "")

	if subject == "" || body == "" {
		return "Error: subject and body parameters are required"
	}

	// Check cooldown in autonomous mode to prevent spam (after validation)
	if !checkEmailCooldown() {
		return "⚠️ Email on cooldown - too many emails sent recently. Waiting before next send."
	}

	// Get email configuration (uses local SMTP relay by default, no auth needed)
	cfg := email.DefaultConfig()

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

	// Record that we sent an email (for cooldown tracking)
	recordEmailSent()

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

	// Check cooldown in autonomous mode to prevent spam (after validation)
	if !checkEmailCooldown() {
		return "⚠️ Progress report on cooldown - too many reports sent recently. Waiting before next send."
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

	// Record that we sent an email (for cooldown tracking)
	recordEmailSent()

	var sb strings.Builder
	sb.WriteString("✅ Progress report sent successfully\n")
	sb.WriteString(fmt.Sprintf("   To: scott@stg.net\n"))
	sb.WriteString(fmt.Sprintf("   From: yolo@b-haven.org\n"))
	sb.WriteString(fmt.Sprintf("   Subject: %s\n", subject))
	return sb.String()
}
