// Package email provides functionality for sending emails via SMTP.
package email

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Config holds SMTP configuration settings.
type Config struct {
	SMTPHost     string // SMTP server host (default: uses sendmail)
	SMTPPort     int    // SMTP server port (default: 25, not used with sendmail)
	Username     string // SMTP username (not used with sendmail)
	Password     string // SMTP password (not used with sendmail)
	UseTLS       bool   // Enable TLS (default: false for local relay)
	SendmailPath string // Path to sendmail binary (default: /usr/sbin/sendmail)
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		SMTPHost:     "localhost", // Not used with sendmail
		SMTPPort:     25,         // Not used with sendmail
		Username:     "",         // Not used with sendmail
		Password:     "",         // Not used with sendmail
		UseTLS:       false,      // Not used with sendmail
		SendmailPath: "/usr/sbin/sendmail",
	}
}

// Client handles email sending via sendmail or SMTP.
type Client struct {
	config *Config
}

// New creates a new email client with the given configuration.
func New(config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}
	// Set defaults for empty values
	if config.SendmailPath == "" {
		config.SendmailPath = "/usr/sbin/sendmail"
	}
	return &Client{config: config}
}

// Message represents an email to be sent.
type Message struct {
	From      string   // Sender address
	To        []string // Recipient addresses (can include CC/BCC)
	Subject   string   // Email subject
	Body      string   // Email body (plain text)
	HTMLBody  string   // Optional HTML version of the body
	Cc        []string // Carbon copy recipients
	Bcc       []string // Blind carbon copy recipients
	Headers   map[string]string // Custom headers
}

// Send sends an email message.
func (c *Client) Send(msg *Message) error {
	if len(msg.To) == 0 && len(msg.Cc) == 0 && len(msg.Bcc) == 0 {
		return fmt.Errorf("at least one recipient required")
	}

	// Create the email body with headers
	body := c.prepareMessage(msg)

	// Get all recipients (including CC and BCC for sendmail)
	recipients := make([]string, 0, len(msg.To)+len(msg.Cc)+len(msg.Bcc))
	recipients = append(recipients, msg.To...)
	recipients = append(recipients, msg.Cc...)
	recipients = append(recipients, msg.Bcc...)

	// Use sendmail if available (preferred for local MTA)
	if c.config.SendmailPath != "" {
		return c.sendViaSendmail(body, recipients)
	}

	// Fall back to direct SMTP connection
	return c.sendViaSMTP(msg.From, recipients, body)
}

// prepareMessage formats the email with proper headers.
func (c *Client) prepareMessage(msg *Message) string {
	var buf strings.Builder

	// From address
	if msg.From == "" {
		msg.From = "yolo@b-haven.org"
	}
	buf.WriteString(fmt.Sprintf("From: %s\r\n", msg.From))

	// To addresses
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))

	// CC addresses (if any)
	if len(msg.Cc) > 0 {
		buf.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.Cc, ", ")))
	}

	// Subject
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))

	// Content-Type header
	if msg.HTMLBody != "" {
		buf.WriteString("MIME-Version: 1.0\r\n")
		buf.WriteString("Content-Type: multipart/alternative; boundary=\"----=_Part_1234567890\"\r\n")
	} else {
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}

	// Custom headers
	for key, value := range msg.Headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// Empty line to separate headers from body
	buf.WriteString("\r\n")

	// Body content
	if msg.HTMLBody != "" {
		// Multipart plain + HTML
		buf.WriteString("------=_Part_1234567890\r\n")
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(msg.Body)
		buf.WriteString("\r\n\r\n------=_Part_1234567890\r\n")
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(msg.HTMLBody)
		buf.WriteString("\r\n\r\n------=_Part_1234567890--\r\n")
	} else {
		buf.WriteString(msg.Body)
	}

	return buf.String()
}

// sendViaSendmail sends the email using the system's sendmail binary.
func (c *Client) sendViaSendmail(body string, recipients []string) error {
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients provided")
	}

	// Sendmail accepts recipients as arguments
	args := append([]string{"-t"}, recipients...) // -t reads from headers, but we also pass explicitly
	cmd := exec.Command(c.config.SendmailPath, args...)
	cmd.Stdin = strings.NewReader(body)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("sendmail failed: %v, stderr: %s", err, stderr.String())
	}

	return nil
}

// sendViaSMTP sends the email using a direct SMTP connection.
func (c *Client) sendViaSMTP(from string, to []string, body string) error {
	host := c.config.SMTPHost
	port := strconv.Itoa(c.config.SMTPPort)

	// Connect to SMTP server
	conn, err := net.DialTimeout("tcp", host+":"+port, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %v", err)
	}
	defer conn.Close()

	// Upgrade to TLS if configured
	if c.config.UseTLS {
		tlsConn := tls.Client(conn, &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: false,
		})
		err = tlsConn.Handshake()
		if err != nil {
			return fmt.Errorf("failed TLS handshake: %v", err)
		}
		conn = tlsConn
	}

	// Read SMTP greeting
	buf := make([]byte, 1024)
	n, _ := conn.Read(buf)
	greeting := string(buf[:n])
	if !strings.HasPrefix(greeting, "220") {
		return fmt.Errorf("invalid SMTP greeting: %s", greeting)
	}

	// Helper to send command and read response
	sendCmd := func(cmd string) (string, error) {
		_, err := io.WriteString(conn, cmd+"\r\n")
		if err != nil {
			return "", fmt.Errorf("failed to write: %v", err)
		}
		n, err := conn.Read(buf)
		if err != nil {
			return "", fmt.Errorf("failed to read: %v", err)
		}
		return string(buf[:n]), nil
	}

	// HELO/EHLO
	response, _ := sendCmd("EHLO " + c.config.SMTPHost)
	if !strings.HasPrefix(response, "250") {
		return fmt.Errorf("EHLO failed: %s", response)
	}

	// AUTH (if credentials provided)
	if c.config.Username != "" && c.config.Password != "" {
		authCmd := fmt.Sprintf("AUTH PLAIN %s", base64.StdEncoding.EncodeToString([]byte("\x00"+c.config.Username+"\x00"+c.config.Password)))
		response, err := sendCmd(authCmd)
		if err != nil || !strings.HasPrefix(response, "235") {
			return fmt.Errorf("authentication failed: %v, response: %s", err, response)
		}
	}

	// MAIL FROM
	response, _ = sendCmd("MAIL FROM:<" + from + ">")
	if !strings.HasPrefix(response, "250") {
		return fmt.Errorf("MAIL FROM failed: %s", response)
	}

	// RCPT TO (for each recipient)
	for _, recipient := range to {
		response, _ = sendCmd("RCPT TO:<" + recipient + ">")
		if !strings.HasPrefix(response, "250") && !strings.HasPrefix(response, "251") {
			return fmt.Errorf("RCPT TO failed for %s: %s", recipient, response)
		}
	}

	// DATA
	response, _ = sendCmd("DATA")
	if !strings.HasPrefix(response, "354") {
		return fmt.Errorf("DATA command failed: %s", response)
	}

	// Send message body
	_, err = io.WriteString(conn, body+"\r\n.\r\n")
	if err != nil {
		return fmt.Errorf("failed to send data: %v", err)
	}

	// Read final response
	n, _ = conn.Read(buf)
	finalResponse := string(buf[:n])
	if !strings.HasPrefix(finalResponse, "250") {
		return fmt.Errorf("email not accepted: %s", finalResponse)
	}

	return nil
}
