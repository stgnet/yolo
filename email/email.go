package email

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	// DefaultFrom is the default sender email address
	DefaultFrom = "yolo@b-haven.org"
	// DefaultSendmailPath is the default path to the sendmail binary
	DefaultSendmailPath = "/usr/sbin/sendmail"
	// EnvFrom is the environment variable for setting the sender email
	EnvFrom = "YELO_EMAIL_FROM"
	// EnvSendmailPath is the environment variable for setting the sendmail path
	EnvSendmailPath = "YELO_SENDBMAIL_PATH"
)

// Config holds email configuration settings
type Config struct {
	From         string
	UseSendmail  bool
	SendmailPath string
}

// getEnvOrDefault returns the value of an environment variable or the default if not set.
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// Message represents an email to send
type Message struct {
	To      []string
	Subject string
	Body    string
}

// DefaultConfig returns default email configuration using sendmail.
// Values can be overridden via environment variables YELO_EMAIL_FROM and YELO_SENDBMAIL_PATH.
func DefaultConfig() *Config {
	return &Config{
		From:         getEnvOrDefault(EnvFrom, DefaultFrom),
		UseSendmail:  true,
		SendmailPath: getEnvOrDefault(EnvSendmailPath, DefaultSendmailPath),
	}
}

// Client is an email client that supports sendmail transport
type Client struct {
	config *Config
}

// New creates a new email client
func New(config *Config) *Client {
	return &Client{config: config}
}

// Send sends an email using the configured transport (sendmail by default)
func (c *Client) Send(msg *Message) error {
	if len(msg.To) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	if msg.Subject == "" || msg.Body == "" {
		return fmt.Errorf("subject and body are required")
	}

	if c.config.UseSendmail {
		return c.sendViaSendmail(msg)
	}

	return fmt.Errorf("SMTP transport not implemented - use sendmail")
}

// sendViaSendmail sends email using the local sendmail command
func (c *Client) sendViaSendmail(msg *Message) error {
	// Build RFC 2822 email format
	var emailContent bytes.Buffer

	emailContent.WriteString(fmt.Sprintf("From: %s\r\n", c.config.From))
	emailContent.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	emailContent.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	emailContent.WriteString(fmt.Sprintf("Date: %s\r\n", getRFC2822Date()))
	emailContent.WriteString("MIME-Version: 1.0\r\n")
	emailContent.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	emailContent.WriteString("\r\n")
	emailContent.WriteString(msg.Body)

	// Use sendmail command with -f flag to set envelope sender
	args := append([]string{"-f", c.config.From}, msg.To...)
	cmd := exec.Command(c.config.SendmailPath, args...)
	cmd.Stdin = &emailContent

	err := cmd.Run()
	if err != nil {
		// Log error with context including recipient email, subject, and error details
		log.Printf("[EMAIL ERROR] Failed to send email:\n  Recipients: %s\n  Subject: %s\n  Error: %v",
			strings.Join(msg.To, ", "), msg.Subject, err)
		return fmt.Errorf("sendmail failed for email '%s' to %s: %w", msg.Subject, strings.Join(msg.To, ", "), err)
	}

	return nil
}

// getRFC2822Date returns current time in RFC 2822 format
func getRFC2822Date() string {
	cmd := exec.Command("date", "-R")
	output, err := cmd.Output()
	if err != nil {
		return "Mon, 1 Jan 2024 00:00:00 +0000"
	}
	return strings.TrimSpace(string(output))
}
