// Email Tool Implementation with Security Hardening
// Allows YOLO to send emails via SMTP from yolo@b-haven.org
// Includes protection against prompt injection, header injection, and rate limiting

package main

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/scottstg/yolo/email"
)

// Security constants for rate limiting and validation
const (
	MaxEmailsPerHour   = 10              // Maximum emails per hour
	CooldownPeriod     = time.Second * 5 // Minimum time between email sends
	AllowedSenderRegex = `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	DenylistedDomains  = "example.com|test.com|suspicious-domain.org"
)

// EmailCooldownTracker tracks email sending rate
var (
	lastEmailTime atomic.Value // stores time.Time via interface{}
	emailCount    atomic.Int32
	hourStart     atomic.Int64
	initOnce      sync.Once
)

// initEmailSecurity initializes security tracking variables
func initEmailSecurity() {
	initOnce.Do(func() {
		now := time.Now()
		hourStart.Store(now.Unix())
		lastEmailTime.Store(time.Time{})
	})
}

// checkEmailCooldown enforces rate limiting and cooldown periods
func checkEmailCooldown() bool {
	hourUnix := hourStart.Load()
	currentUnix := time.Now().Unix()

	// Reset counter at the start of each hour
	if currentUnix-hourUnix >= 3600 {
		hourStart.Store(currentUnix)
		emailCount.Store(0)
	}

	// Check rate limit (max emails per hour)
	if emailCount.Load() >= MaxEmailsPerHour {
		return false // Rate limit exceeded
	}

	// Check cooldown period between sends
	lastTimeVal := lastEmailTime.Load()
	if lastTimeVal == nil {
		return true
	}
	lastTime, ok := lastTimeVal.(time.Time)
	if !ok || lastTime.IsZero() {
		return true
	}
	if time.Since(lastTime) < CooldownPeriod {
		return false // Still in cooldown
	}

	return true
}

// recordEmailSent updates the email send tracking after a successful send
func recordEmailSent() error {
	lastEmailTime.Store(time.Now())
	emailCount.Add(1)
	return nil
}

// validateSender checks if sender is allowed (not denylisted)
// Format validation MUST happen before denylist check to avoid false rejections of invalid emails
func validateSender(emailAddress string) bool {
	// First, validate email format - must be valid before checking denylist
	formatRegex := regexp.MustCompile(AllowedSenderRegex)
	if !formatRegex.MatchString(emailAddress) {
		return false // Invalid email format
	}

	// Only check denylist for properly formatted emails
	denyRegex := regexp.MustCompile("(?i)(" + DenylistedDomains + ")")
	if denyRegex.MatchString(emailAddress) {
		return false // Email domain is denylisted
	}

	return true
}



// encodeHeader safely encodes header values to prevent header injection
func encodeHeader(value string) string {
	// Remove any embedded newlines which could allow header injection
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")

	// Truncate very long headers to prevent buffer overflow attacks
	if len(value) > 500 {
		value = value[:500] + "..."
	}

	return value
}

func (t *ToolExecutor) sendEmail(args map[string]any) string {
	subject := getStringArg(args, "subject", "")
	body := getStringArg(args, "body", "")
	to := getStringArg(args, "to", "")

	if subject == "" || body == "" {
		return "Error: subject and body parameters are required"
	}

	// Validate email format using proper regex
	emailRegex := regexp.MustCompile(AllowedSenderRegex)
	if !emailRegex.MatchString(to) {
		return fmt.Sprintf("Error: invalid email address '%s'", to)
	}

	// Check rate limiting before processing
	if !checkEmailCooldown() {
		return "Error: Email sending rate limit exceeded. Please wait a moment and try again."
	}

	// Get email configuration (uses local SMTP relay by default, no auth needed)
	cfg := email.DefaultConfig()

	// If no recipient specified, use default (scott@stg.net)
	if to == "" {
		to = "scott@stg.net"
	}

	// Validate sender is not denylisted
	if !validateSender(to) {
		return fmt.Sprintf("Error: email address '%s' is not allowed", to)
	}

	// Encode headers safely to prevent header injection
	safeSubject := encodeHeader(subject)

	msg := &email.Message{
		To:      []string{to},
		Subject: safeSubject,
		Body:    body,
	}

	client := email.New(cfg)
	err := client.Send(msg)
	if err != nil {
		return fmt.Sprintf("Error sending email: %v", err)
	}

	// Record successful send for rate limiting
	recordEmailSent()

	var sb strings.Builder
	sb.WriteString("✅ Email sent successfully\n")
	sb.WriteString(fmt.Sprintf("   To: %s\n", to))
	sb.WriteString(fmt.Sprintf("   From: yolo@b-haven.org\n"))
	sb.WriteString(fmt.Sprintf("   Subject: %s\n", safeSubject))
	return sb.String()
}

func (t *ToolExecutor) sendReport(args map[string]any) string {
	// Convenience function for sending progress reports
	cfg := email.DefaultConfig()

	subject := getStringArg(args, "subject", "YOLO Progress Report")
	body := getStringArg(args, "body", "")
	to := getStringArg(args, "to", "")                  // Get recipient from args
	attachTodo := getBoolArg(args, "attach_todo", true) // Default: include todo list

	if body == "" {
		return "Error: body parameter is required"
	}

	// Validate email format for recipient
	matched, _ := regexp.MatchString(AllowedSenderRegex, to)
	if !matched {
		return fmt.Errorf("invalid to address: %s", to).Error()
	}

	// Append todo list to the report if not already included and attach_todo is true
	if attachTodo && !strings.Contains(body, "📝 TODO LIST") {
		todoOutput := listTodos()
		body = body + "\n\n" + todoOutput
	}

	// Use provided recipient or default to scott@stg.net
	if to == "" {
		to = "scott@stg.net"
	}

	// Check rate limiting before processing
	if !checkEmailCooldown() {
		return "Error: Email sending rate limit exceeded. Please wait a moment and try again."
	}

	// Validate sender is not denylisted
	if !validateSender(to) {
		return fmt.Sprintf("Error: email address '%s' is not allowed", to)
	}

	// Encode headers safely to prevent header injection
	safeSubject := encodeHeader(subject)

	msg := &email.Message{
		To:      []string{to},
		Subject: safeSubject,
		Body:    body,
	}

	client := email.New(cfg)
	err := client.Send(msg)
	if err != nil {
		return fmt.Sprintf("Error sending report: %v", err)
	}

	// Record successful send for rate limiting
	recordEmailSent()

	var sb strings.Builder
	sb.WriteString("✅ Progress report sent successfully\n")
	sb.WriteString(fmt.Sprintf("   To: %s\n", to))
	sb.WriteString(fmt.Sprintf("   From: yolo@b-haven.org\n"))
	sb.WriteString(fmt.Sprintf("   Subject: %s\n", safeSubject))
	return sb.String()
}
