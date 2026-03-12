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
		expectSubjPrefix   string
		expectBodyContains string
	}{
		{
			name:               "general inquiry email",
			body:               "Hi YOLO, I heard you're autonomous. How's it going?",
			subject:            "Hello from outside!",
			from:               "user@example.com",
			expectSubjPrefix:   "", // Response is just body text, subject handled separately
			expectBodyContains: "MOCKED",
		},
		{
			name:               "empty body email",
			body:               "",
			subject:            "Just checking",
			from:               "checker@test.com",
			expectSubjPrefix:   "",
			expectBodyContains: "MOCKED",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response := composeResponseToEmail(tc.body, tc.subject, tc.from)

			if !strings.Contains(response, tc.expectBodyContains) {
				t.Errorf("response body should contain %q, got: %q", tc.expectBodyContains, response)
			}

			// For non-empty body, response should be the mocked LLM output
			if tc.body != "" && len(response) < 20 {
				t.Errorf("response too short: %q", response)
			}

			// For empty body, it should still generate something (not just error message)
			if tc.body == "" && !strings.Contains(response, "MOCKED") {
				t.Error("empty body should still get mocked response")
			}
		})
	}
}
