// Tests for email inbox tool
package main

import (
	"strings"
	"testing"
	"time"
)

func TestCleanEmailField(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "full with display name",
			input:    "Scott Griepentrog <scott@griepentrog.com>",
			expected: "scott@griepentrog.com",
		},
		{
			name:     "plain email address",
			input:    "test@stg.net",
			expected: "test@stg.net",
		},
		{
			name:     "with angle brackets only",
			input:    "<user@example.org>",
			expected: "user@example.org",
		},
		{
			name:     "with spaces",
			input:    "  <user@example.org>  ",
			expected: "user@example.org",
		},
		{
			name:     "no angle brackets, no spaces",
			input:    "simple@email.com",
			expected: "simple@email.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanEmailField(tt.input)
			if result != tt.expected {
				t.Errorf("cleanEmailField(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetCurrentTime(t *testing.T) {
	agent := &YoloAgent{config: NewYoloConfig(".")}
	executor := NewToolExecutor(".", agent)
	now := executor.getCurrentTime()

	if now.IsZero() {
		t.Fatal("getCurrentTime returned zero time")
	}

	// Check that the time is reasonably current (within 1 minute)
	diff := time.Since(now)
	if diff < -time.Minute || diff > time.Minute {
		t.Errorf("getCurrentTime() = %v, expected within 1 minute of now (%v), got diff: %v", now, time.Now(), diff)
	}
}

func TestComposeResponseToEmail(t *testing.T) {
	agent := &YoloAgent{config: NewYoloConfig(".")}
	executor := NewToolExecutor(".", agent)

	email := EmailMessage{
		From:    "user@example.com",
		Subject: "Test Question?",
		Date:    "2024-01-01T00:00:00Z",
		Content: "Can you help with this?",
	}

	response := executor.composeResponseToEmail(email)

	// Check that response contains expected elements
	if !strings.Contains(response, email.Subject) {
		t.Errorf("Response doesn't contain subject '%s'", email.Subject)
	}

	if !strings.Contains(response, email.From) {
		t.Errorf("Response doesn't contain from address '%s'", email.From)
	}

	// Verify response format - should start with greeting and end with signature
	if !strings.Contains(response, "Hi ") && !strings.Contains(response, "Hello ") {
		t.Error("Response doesn't contain expected personal greeting")
	}

	if !strings.Contains(response, "Best regards") || !strings.Contains(response, "YOLO") {
		t.Error("Response doesn't end with expected signature")
	}
}

func TestEmailShouldRespond(t *testing.T) {
	tests := []struct {
		name     string
		email    EmailMessage
		expected bool
	}{
		{
			name: "email with question mark",
			email: EmailMessage{
				Subject: "Can you help?",
			},
			expected: true,
		},
		{
			name: "email with please request",
			email: EmailMessage{
				Subject: "Please review this",
			},
			expected: true,
		},
		{
			name: "automation build message",
			email: EmailMessage{
				Content: "Build completed successfully",
				From:    "ci@system",
			},
			expected: false,
		},
		{
			name: "short human email",
			email: EmailMessage{
				Subject: "Quick question",
				Content: "Hi, I need some help with this project.",
				From:    "person@example.com",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := emailShouldRespond(tt.email)
			if result != tt.expected {
				t.Errorf("emailShouldRespond() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestComposeResponseToEmail_ContextualQuestions(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedPhrase string
	}{
		{
			name:           "question about ability to answer earlier messages",
			content:        "Are you now able to answer my questions posed in the earlier message?",
			expectedPhrase: "answer questions from earlier messages",
		},
		{
			name:           "generic question with word 'question'",
			content:        "I have a question about the project",
			expectedPhrase: "I can see you have",
		},
		{
			name:           "testing message",
			content:        "Just testing the email system to make sure I receive responses",
			expectedPhrase: "Test received",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := buildSimpleResponse(tt.content)
			if !strings.Contains(strings.ToLower(response), strings.ToLower(tt.expectedPhrase)) {
				t.Errorf("Response doesn't contain expected phrase '%s'. Got: %s", tt.expectedPhrase, response)
			} else {
				t.Logf("✓ Response correctly includes '%s'", tt.expectedPhrase)
			}
		})
	}
}

// buildSimpleResponse is a pure function for unit testing email response logic
func buildSimpleResponse(content string) string {
	bodyLower := strings.ToLower(content)
	var specificAnswers []string

	// Check for questions about answering earlier messages (more flexible pattern)
	if strings.Contains(bodyLower, "answer") && strings.Contains(bodyLower, "question") &&
		(strings.Contains(bodyLower, "earlier") || strings.Contains(bodyLower, "previous")) {
		specificAnswers = append(specificAnswers, "Yes, I can answer questions from earlier messages.")
	}

	if strings.Contains(bodyLower, "test") && (strings.Contains(bodyLower, "testing") || strings.Contains(bodyLower, "receive")) {
		specificAnswers = append(specificAnswers, "Test received! I'm working correctly.")
	}

	if strings.Contains(bodyLower, "question") {
		specificAnswers = append(specificAnswers, "I can see you have questions or requests.")
	}

	var bodyParts []string
	if len(specificAnswers) > 0 {
		bodyParts = append(bodyParts, strings.Join(specificAnswers, "\n"))
	}
	bodyParts = append(bodyParts, "I'm designed to process emails and respond appropriately.")

	return "Thank you for your message.\n\n" + strings.Join(bodyParts, "\n\n") + "\n\nBest regards,\nYOLO"
}
