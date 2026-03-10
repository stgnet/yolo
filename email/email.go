package email

import (
	"crypto/tls"
	"net/smtp"
	"os"
	"strconv"
)

// Config holds email configuration
type Config struct {
	SMTPHost    string
	SMTPPort    int
	Username    string
	Password    string
	FromAddress string
	FromName    string
}

// DefaultConfig returns a config suitable for b-haven.org
func DefaultConfig() *Config {
	return &Config{
		SMTPHost:    getEnv("EMAIL_SMTP_HOST", "b-haven.org"),
		SMTPPort:    587,
		Username:    getEnv("EMAIL_USERNAME", "yolo@b-haven.org"),
		Password:    getEnv("EMAIL_PASSWORD", ""),
		FromAddress: getEnv("EMAIL_FROM_ADDRESS", "yolo@b-haven.org"),
		FromName:    getEnv("EMAIL_FROM_NAME", "YOLO"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Message represents an email to send
type Message struct {
	To      []string
	Subject string
	Body    string
}

// Send sends an email using the configured SMTP server
func Send(cfg *Config, msg *Message) error {
	auth := smtp.PlainAuth(
		"",
		cfg.Username,
		cfg.Password,
		cfg.SMTPHost,
	)

	from := cfg.FromAddress
	if cfg.FromName != "" {
		from = `"` + cfg.FromName + `" <` + cfg.FromAddress + `>`
	}

	address := cfg.SMTPHost + ":" + strconv.Itoa(cfg.SMTPPort)

	header := make(map[string]string)
	header["From"] = from
	header["To"] = "" // Will be set in loop
	header["Subject"] = msg.Subject
	header["Content-Type"] = "text/plain; charset=UTF-8"

	body := ""
	for key, value := range header {
		body += key + ": " + value + "\r\n"
	}
	body += "\r\n" + msg.Body

	// Connect to SMTP server
	conn, err := smtp.Dial(address)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Start TLS
	if err := conn.StartTLS(&tls.Config{ServerName: cfg.SMTPHost}); err != nil {
		return err
	}

	// Authenticate
	if err := conn.Auth(auth); err != nil {
		return err
	}

	// Set sender and recipient
	if err := conn.Mail(from); err != nil {
		return err
	}

	for _, addr := range msg.To {
		if err := conn.Rcpt(addr); err != nil {
			return err
		}
	}

	// Send data
	wr, err := conn.Data()
	if err != nil {
		return err
	}

	_, err = wr.Write([]byte(body))
	if err != nil {
		return err
	}

	err = wr.Close()
	if err != nil {
		return err
	}

	return conn.Quit()
}

// SendReport sends a progress report email from YOLO
func SendReport(cfg *Config, subject, body string) error {
	msg := &Message{
		To:      []string{"scott@stg.net"},
		Subject: subject,
		Body:    body,
	}
	return Send(cfg, msg)
}
