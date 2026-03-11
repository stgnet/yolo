package email

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Config holds email configuration settings
type Config struct {
	From         string
	UseSendmail  bool
	SendmailPath string
}

// Message represents an email to send
type Message struct {
	To      []string
	Subject string
	Body    string
}

// DefaultConfig returns default email configuration using sendmail
func DefaultConfig() *Config {
	return &Config{
		From:         "yolo@b-haven.org",
		UseSendmail:  true,
		SendmailPath: "/usr/sbin/sendmail",
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
		return fmt.Errorf("sendmail failed: %w", err)
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
