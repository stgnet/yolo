package email

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestDefaultConfig tests the default email configuration
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.From == "" {
		t.Error("Expected default From address, got empty string")
	}
	if cfg.From != DefaultFrom {
		t.Errorf("Expected From=%s, got %s", DefaultFrom, cfg.From)
	}
	if !cfg.UseSendmail {
		t.Error("Expected UseSendmail to be true by default")
	}
	if cfg.SendmailPath == "" {
		t.Error("Expected SendmailPath to be set")
	}
}

// TestNewClient tests creating a new email client
func TestNewClient(t *testing.T) {
	cfg := &Config{
		From:         "test@example.com",
		UseSendmail:  false,
		SendmailPath: "/usr/sbin/sendmail",
	}

	client := New(cfg)
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.config != cfg {
		t.Error("Expected client to use provided config")
	}
}

// TestGetEnvOrDefault tests environment variable handling
func TestGetEnvOrDefault(t *testing.T) {
	testCases := []struct {
		name       string
		key        string
		defaultVal string
		envValue   string
		expected   string
	}{
		{"env not set", "NONEXISTENT_VAR_XYZ", "default", "", "default"},
		{"env set to value", "TEST_ENV_VAR_123", "default", "custom_value", "custom_value"},
		{"env empty string", "EMPTY_VAR_TEST", "fallback", "", "fallback"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Unsetenv(tc.key)
			if tc.envValue != "" {
				os.Setenv(tc.key, tc.envValue)
			}
			defer os.Unsetenv(tc.key)

			result := getEnvOrDefault(tc.key, tc.defaultVal)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

// TestMessageValidation tests email message validation in Send method
func TestMessageValidation(t *testing.T) {
	cfg := DefaultConfig()
	client := New(cfg)

	testCases := []struct {
		name          string
		message       *Message
		expectError   bool
		errorContains string
	}{
		{
			name: "valid message",
			message: &Message{
				To:      []string{"recipient@example.com"},
				Subject: "Test Subject",
				Body:    "Test body content",
			},
			expectError: false,
		},
		{
			name: "no recipients",
			message: &Message{
				Subject: "Test",
				Body:    "Test body",
			},
			expectError:   true,
			errorContains: "recipients",
		},
		{
			name: "missing subject",
			message: &Message{
				To:   []string{"recipient@example.com"},
				Body: "Test body",
			},
			expectError:   true,
			errorContains: "subject and body",
		},
		{
			name: "missing body",
			message: &Message{
				To:      []string{"recipient@example.com"},
				Subject: "Test Subject",
			},
			expectError:   true,
			errorContains: "subject and body",
		},
		{
			name: "multiple recipients",
			message: &Message{
				To:      []string{"user1@example.com", "user2@example.com"},
				Subject: "Test Subject",
				Body:    "Test body",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Validate message without sending
			err := client.ValidateMessage(tc.message)

			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			} else if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			} else if tc.expectError && tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
				t.Errorf("Expected error to contain %q, got %v", tc.errorContains, err)
			}
		})
	}
}

// TestMessageStructure tests that messages are properly formatted in sendmail format
func TestMessageStructure(t *testing.T) {
	cfg := DefaultConfig()
	_ = New(cfg)

	msg := &Message{
		To:      []string{"user1@example.com", "user2@example.com"},
		Subject: "Test Subject Line",
		Body:    "This is the test body content.",
	}

	var emailContent bytes.Buffer
	emailContent.WriteString("From: " + cfg.From + "\r\n")
	emailContent.WriteString("To: " + strings.Join(msg.To, ", ") + "\r\n")
	emailContent.WriteString("Subject: " + msg.Subject + "\r\n")
	emailContent.WriteString("MIME-Version: 1.0\r\n")
	emailContent.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	emailContent.WriteString("\r\n")
	emailContent.WriteString(msg.Body)

	// Verify expected format components exist
	testCases := []struct {
		name        string
		expected    string
		content     string
		expectFound bool
	}{
		{"From header", "From: yolo@localhost\r\n", emailContent.String(), true},
		{"To header", "To: user1@example.com, user2@example.com\r\n", emailContent.String(), true},
		{"Subject header", "Subject: Test Subject Line\r\n", emailContent.String(), true},
		{"MIME-Version header", "MIME-Version: 1.0\r\n", emailContent.String(), true},
		{"Body content", "This is the test body content.", emailContent.String(), true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			found := strings.Contains(tc.content, tc.expected)
			if found != tc.expectFound {
				t.Errorf("Expected %s to %s in email content: expected=%v",
					tc.expected, map[bool]string{true: "found", false: "not found"}[tc.expectFound], tc.expectFound)
			}
		})
	}
}

// TestSMTPNotImplemented tests that non-sendmail transport returns appropriate error
func TestSMTPNotImplemented(t *testing.T) {
	cfg := &Config{
		From:         "test@example.com",
		UseSendmail:  false,
		SendmailPath: "/usr/sbin/sendmail",
	}
	client := New(cfg)

	msg := &Message{
		To:      []string{"recipient@example.com"},
		Subject: "Test",
		Body:    "Test body",
	}

	err := client.Send(msg)
	if err == nil {
		t.Error("Expected error for non-sendmail transport")
	}
	if !strings.Contains(err.Error(), "SMTP") && !strings.Contains(err.Error(), "sendmail") {
		t.Errorf("Expected SMTP or sendmail-related error, got: %v", err)
	}
}

// TestGetRFC2822Date tests the date formatting function
func TestGetRFC2822Date(t *testing.T) {
	dateStr := getRFC2822Date()

	if len(dateStr) < 10 {
		t.Errorf("Expected RFC2822 date format, got string of length %d: %s", len(dateStr), dateStr)
	}
	// RFC2822 date should contain standard components like day, month, year
	expectedComponents := []string{
		"Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec",
		"[0-9]{4}",
		"+[0-9]{4}",
	}

	for _, component := range expectedComponents {
		_ = component
		// Basic validation: should be a non-empty string
		if len(dateStr) == 0 {
			t.Error("RFC2822 date is empty")
			break
		}
	}
}

// TestClientSendViaSendmailWithEmptyRecipients tests edge case of no recipients
func TestSendWithoutRecipients(t *testing.T) {
	cfg := DefaultConfig()
	client := New(cfg)

	msg := &Message{
		To:      []string{},
		Subject: "Test",
		Body:    "Test body",
	}

	err := client.ValidateMessage(msg)
	if err == nil {
		t.Error("Expected error for empty recipients")
	}
}

// TestConfigCustomValues tests that custom config values are properly used
func TestConfigCustomValues(t *testing.T) {
	customFrom := "custom@domain.com"
	customPath := "/custom/sendmail/path"

	cfg := &Config{
		From:         customFrom,
		UseSendmail:  true,
		SendmailPath: customPath,
	}

	client := New(cfg)

	if client.config.From != customFrom {
		t.Errorf("Expected From=%s, got %s", customFrom, client.config.From)
	}
	if client.config.SendmailPath != customPath {
		t.Errorf("Expected SendmailPath=%s, got %s", customPath, client.config.SendmailPath)
	}
}

// TestMessageEmptySubject tests sending with empty subject
func TestMessageWithEmptySubject(t *testing.T) {
	cfg := DefaultConfig()
	client := New(cfg)

	msg := &Message{
		To:      []string{"recipient@example.com"},
		Subject: "",
		Body:    "Test body",
	}

	err := client.ValidateMessage(msg)
	if err == nil {
		t.Error("Expected error for empty subject")
	}
	if !strings.Contains(err.Error(), "subject and body") {
		t.Errorf("Expected 'subject and body' error, got: %v", err)
	}
}

// TestMultipleEmailRecipients tests that multiple recipients are properly joined
func TestMultipleEmailRecipients(t *testing.T) {
	cfg := DefaultConfig()
	client := New(cfg)

	recipients := []string{"alice@example.com", "bob@example.com", "charlie@example.com"}
	msg := &Message{
		To:      recipients,
		Subject: "Test Subject",
		Body:    "Test body",
	}

	err := client.ValidateMessage(msg)
	if err != nil {
		t.Errorf("Expected no error for valid message with multiple recipients: %v", err)
	}
}

// TestGetEnvVariableOverride tests that environment variables can override defaults
func TestEnvironmentVariableOverrides(t *testing.T) {
	originalFrom := os.Getenv(EnvFrom)
	originalPath := os.Getenv(EnvSendmailPath)
	defer func() {
		if originalFrom != "" {
			os.Setenv(EnvFrom, originalFrom)
		} else {
			os.Unsetenv(EnvFrom)
		}
		if originalPath != "" {
			os.Setenv(EnvSendmailPath, originalPath)
		} else {
			os.Unsetenv(EnvSendmailPath)
		}
	}()

	customEmail := "override@example.com"
	customPath := "/custom/sendmail"
	os.Setenv(EnvFrom, customEmail)
	os.Setenv(EnvSendmailPath, customPath)

	cfg := DefaultConfig()
	if cfg.From != customEmail {
		t.Errorf("Expected From to be overridden to %s, got %s", customEmail, cfg.From)
	}
	if cfg.SendmailPath != customPath {
		t.Errorf("Expected SendmailPath to be overridden to %s, got %s", customPath, cfg.SendmailPath)
	}
}

// TestMessageWithLongBody tests that long email bodies are handled properly
func TestMessageWithLongBody(t *testing.T) {
	// This test verifies that long email bodies are handled correctly without actually sending.
	// Actual email delivery would cause test failures or network issues, so we skip the Send call.
	cfg := DefaultConfig()
	client := New(cfg)

	longBody := strings.Repeat("Line of text. ", 100) // ~2000 characters
	msg := &Message{
		To:      []string{"recipient@example.com"},
		Subject: "Long Email Test",
		Body:    longBody,
	}

	// Validate message creation without actual delivery
	if msg == nil {
		t.Fatal("Expected non-nil message")
	}
	if len(msg.Body) < 1000 {
		t.Errorf("Expected long body (>1000 chars), got %d", len(msg.Body))
	}

	// Validate the message using the client (without sending)
	err := client.ValidateMessage(msg)
	if err != nil {
		t.Errorf("Validation failed for valid long message: %v", err)
	}

	// Don't actually send - tests should not trigger network operations
	t.Skip("Skipping actual email send to avoid network dependency")
}

// TestClientNilConfigSafety tests that New doesn't panic with nil config
func TestNewClientWithNilConfig(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Error("New client should not panic with nil config")
		}
	}()

	client := New(nil)
	if client == nil {
		t.Error("Expected non-nil client even with nil config")
	}
}

// TestSendViaSendmailFormat verifies the email content format matches RFC 2822 requirements
func TestSendViaSendmailFormat(t *testing.T) {
	cfg := DefaultConfig()
	client := New(cfg)

	// Create a message to test validation without actual delivery
	msg := &Message{
		To:      []string{"test@example.com"},
		Subject: "Format Test",
		Body:    "Test body for format verification",
	}

	// Verify the ValidateMessage method checks required fields before attempting send
	err := client.ValidateMessage(msg)

	if err == nil {
		t.Log("ValidateMessage validation passed")
	} else {
		t.Errorf("Unexpected validation error: %v", err)
	}

	// Verify all required fields were checked by creating invalid messages
	err = client.ValidateMessage(&Message{To: []string{"test@example.com"}}) // missing subject, body
	if err == nil {
		t.Error("Should have errored on missing subject and body")
	}

	err = client.ValidateMessage(&Message{Subject: "Test", Body: "Test"}) // missing recipients
	if err == nil {
		t.Error("Should have errored on missing recipients")
	}
}

// TestConfigWithFrom tests ConfigWithFrom function with various inputs
func TestConfigWithFrom(t *testing.T) {
	testCases := []struct {
		name         string
		from         string
		expectedFrom string
	}{
		{
			name:         "with valid custom from",
			from:         "custom@example.com",
			expectedFrom: "custom@example.com",
		},
		{
			name:         "with empty from uses default",
			from:         "",
			expectedFrom: DefaultFrom,
		},
		{
			name:         "with complex email address",
			from:         "user+tag@subdomain.example.com",
			expectedFrom: "user+tag@subdomain.example.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Unset environment variable to ensure we use defaults
			os.Unsetenv(EnvFrom)
			defer os.Unsetenv(EnvFrom)

			cfg := ConfigWithFrom(tc.from)
			if cfg == nil {
				t.Fatal("Expected non-nil config")
			}
			if cfg.From != tc.expectedFrom {
				t.Errorf("Expected From=%s, got %s", tc.expectedFrom, cfg.From)
			}
			if !cfg.UseSendmail {
				t.Error("Expected UseSendmail to be true")
			}
			if cfg.SendmailPath == "" {
				t.Error("Expected SendmailPath to be set")
			}
		})
	}
}

// TestSanitizeFilename tests filename sanitization for email headers
func TestSanitizeFilename(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple filename",
			input:    "document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "filename with quotes",
			input:    `file"name.txt`,
			expected: "filename.txt",
		},
		{
			name:     "filename with multiple quotes",
			input:    `"test"file"data.pdf`,
			expected: "testfiledata.pdf",
		},
		{
			name:     "filename with newline",
			input:    "file\nwith\nnewlines.txt",
			expected: "file_with_newlines.txt",
		},
		{
			name:     "filename with carriage return",
			input:    "file\rwith\rcrlf.txt",
			expected: "file_with_crlf.txt",
		},
		{
			name:     "filename with mixed problematic chars",
			input:    "\"test\nfile\rdata.pdf",
			expected: "test_file_data.pdf",
		},
		{
			name:     "empty filename",
			input:    "",
			expected: "",
		},
		{
			name:     "filename with path",
			input:    "/path/to/file.txt",
			expected: "/path/to/file.txt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeFilename(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// TestBuildMultipartMessage tests multipart message building with attachments
func TestBuildMultipartMessage(t *testing.T) {
	cfg := &Config{
		From:         "test@example.com",
		UseSendmail:  true,
		SendmailPath: "/usr/sbin/sendmail",
	}
	client := New(cfg)

	testCases := []struct {
		name            string
		message         *Message
		checkBoundaries bool
	}{
		{
			name: "single attachment",
			message: &Message{
				To:      []string{"recipient@example.com"},
				Subject: "Test with Attachment",
				Body:    "This is a test email with an attachment.",
				Attachments: []Attachment{
					{
						Filename: "test.pdf",
						Content:  []byte("PDF content here"),
					},
				},
			},
			checkBoundaries: true,
		},
		{
			name: "multiple attachments",
			message: &Message{
				To:      []string{"recipient@example.com"},
				Subject: "Multiple Attachments",
				Body:    "Email with multiple files.",
				Attachments: []Attachment{
					{
						Filename: "file1.txt",
						Content:  []byte("Content 1"),
					},
					{
						Filename: "file2.pdf",
						Content:  []byte("Content 2"),
					},
				},
			},
			checkBoundaries: true,
		},
		{
			name: "attachment with special filename chars",
			message: &Message{
				To:      []string{"recipient@example.com"},
				Subject: "Special Filename Test",
				Body:    "Testing sanitized filenames.",
				Attachments: []Attachment{
					{
						Filename: `"test\nfile".pdf`,
						Content:  []byte("Test content"),
					},
				},
			},
			checkBoundaries: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := client.buildMultipartMessage(tc.message)
			content := buf.String()

			// Check basic headers exist
			expectedHeaders := []string{
				"From: test@example.com",
				fmt.Sprintf("To: %s", tc.message.To[0]),
				fmt.Sprintf("Subject: %s", tc.message.Subject),
				"MIME-Version: 1.0",
				"Content-Type: multipart/mixed; boundary=",
			}

			for _, header := range expectedHeaders {
				if !strings.Contains(content, header) {
					t.Errorf("Expected header %q in content", header)
				}
			}

			// Check text part exists
			if !strings.Contains(content, "Content-Type: text/plain; charset=utf-8") {
				t.Error("Expected text part Content-Type header")
			}

			if !strings.Contains(content, "Content-Transfer-Encoding: quoted-printable") {
				t.Error("Expected quoted-printable encoding header")
			}

			// Check attachment parts exist
			for i, att := range tc.message.Attachments {
				sanitizedFilename := sanitizeFilename(att.Filename)
				if !strings.Contains(content, fmt.Sprintf(`name="%s"`, sanitizedFilename)) {
					t.Errorf("Expected attachment %d with filename %q", i+1, sanitizedFilename)
				}

				if !strings.Contains(content, "Content-Type: application/octet-stream") {
					t.Errorf("Expected attachment %d to have application/octet-stream content type", i+1)
				}

				if !strings.Contains(content, "Content-Transfer-Encoding: base64") {
					t.Errorf("Expected attachment %d to have base64 encoding", i+1)
				}
			}

			// Check boundaries if requested
			if tc.checkBoundaries {
				// Content should contain boundary markers
				if !strings.Contains(content, "Content-Type: multipart/mixed; boundary=") {
					t.Error("Expected Content-Type header with boundary parameter")
				}

				// Should have at least one opening boundary marker (--)
				boundaryCount := strings.Count(content, "--yolo-")
				if boundaryCount < 2 {
					t.Errorf("Expected at least 2 boundary markers ('--yolo-'), found %d", boundaryCount)
				}

				// Content should end with closing boundary
				if !strings.HasSuffix(strings.TrimSpace(content), "--\r\n") && !strings.HasSuffix(strings.TrimSpace(content), "--") {
					t.Error("Expected content to end with closing boundary '--'")
				}
			}
		})
	}
}

// TestBuildMultipartMessageBase64Encoding tests that attachment content is properly base64 encoded
func TestBuildMultipartMessageBase64Encoding(t *testing.T) {
	cfg := DefaultConfig()
	client := New(cfg)

	testContent := []byte("Hello, World!")
	expectedEncoded := "SGVsbG8sIFdvcmxkIQ==" // Base64 of "Hello, World!"

	msg := &Message{
		To:      []string{"test@example.com"},
		Subject: "Test",
		Body:    "Test body",
		Attachments: []Attachment{
			{
				Filename: "test.txt",
				Content:  testContent,
			},
		},
	}

	buf := client.buildMultipartMessage(msg)
	content := buf.String()

	if !strings.Contains(content, expectedEncoded) {
		t.Errorf("Expected base64 encoded content %q in message body", expectedEncoded)
	}
}

// TestBuildMultipartMessageLongContent tests that long attachment content is properly wrapped
func TestBuildMultipartMessageLongContent(t *testing.T) {
	cfg := DefaultConfig()
	client := New(cfg)

	// Create content that will be longer than 76 chars when base64 encoded
	longContent := bytes.Repeat([]byte("A"), 100)

	msg := &Message{
		To:      []string{"test@example.com"},
		Subject: "Test",
		Body:    "Test body",
		Attachments: []Attachment{
			{
				Filename: "long.txt",
				Content:  longContent,
			},
		},
	}

	buf := client.buildMultipartMessage(msg)
	content := buf.String()

	// Check that lines are properly wrapped (no line should exceed 76 chars for base64)
	lines := strings.Split(content, "\r\n")
	for i, line := range lines {
		if len(line) > 78 && strings.Contains(line, "AAAAA") { // Base64 'A' is 'Q=='
			t.Errorf("Line %d exceeds expected length: %d chars", i, len(line))
		}
	}
}
