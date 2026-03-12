package main

import (
	"strings"
	"testing"
)

// Test composeResponseToEmail with mocked LLM (user suggestion)
func TestComposeResponseToEmailWithMockLLM(t *testing.T) {
	// Save original generator function
	origGenerator := llmResponseGenerator

	// Mock the LLM response - simulate deterministic behavior for testing
	llmResponseGenerator = func(prompt string) string {
		return "MOCKED: This is an automated test response to verify composeResponseToEmail works with direct LLM generation."
	}

	// Restore on cleanup
	defer func() { llmResponseGenerator = origGenerator }()

	testCases := []struct {
		name               string
		body               string
		subject            string
		from               string
		expectBodyContains string
	}{
		{
			name:               "general inquiry email",
			body:               "Hi YOLO, I heard you're autonomous. How's it going?",
			subject:            "Hello from outside!",
			from:               "user@example.com",
			expectBodyContains: "MOCKED",
		},
		{
			name:               "empty body email",
			body:               "",
			subject:            "Just checking",
			from:               "checker@test.com",
			expectBodyContains: "MOCKED",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response := composeResponseToEmail(tc.body, tc.subject, tc.from)

			if !strings.Contains(response, tc.expectBodyContains) {
				t.Errorf("response body should contain %q, got: %q", tc.expectBodyContains, response)
			}

			// Response should be at least the mocked content
			if len(response) < 20 {
				t.Errorf("response too short: %q", response)
			}
		})
	}
}

// Test limitString function
func TestLimitString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than maxLen",
			input:    "Hello",
			maxLen:   10,
			expected: "Hello",
		},
		{
			name:     "string equal to maxLen",
			input:    "Hello",
			maxLen:   5,
			expected: "Hello",
		},
		{
			name:     "string longer than maxLen",
			input:    "Hello World",
			maxLen:   5,
			expected: "Hello...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := limitString(tc.input, tc.maxLen)
			if result != tc.expected {
				t.Errorf("limitString(%q, %d) = %q, want %q", tc.input, tc.maxLen, result, tc.expected)
			}
		})
	}
}

// Test processInboxWorkflow with mock LLM
func TestProcessInboxWorkflow(t *testing.T) {
	// Save original LLM generator
	origLLMGen := llmResponseGenerator

	// Mock LLM response
	llmResponseGenerator = func(prompt string) string {
		return "Test auto-reply to your email"
	}

	defer func() {
		llmResponseGenerator = origLLMGen
	}()

	// Test the compose function directly (hard to test full workflow without real mailbox)
	response := composeResponseToEmail("Test email body", "Test Subject", "test@example.com")

	if !strings.Contains(response, "Test auto-reply") {
		t.Errorf("Response should contain mock reply, got: %s", response)
	}
}
