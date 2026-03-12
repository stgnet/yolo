package main

import (
	"strings"
	"testing"
)

// TestEmailResponseShowsActionsTaken verifies that actions taken are shown in responses.
//
// SKIPPED: This test calls composeResponseToEmail which internally invokes
// runCommand to execute shell commands like "go test -v -cover ./..." and
// "sendmail". When run inside "go test", the "go test" command recurses
// infinitely (the inner go test compiles and runs the same tests, which
// again invoke composeResponseToEmail, which again runs "go test", etc.).
// The sendmail command also blocks waiting for input on stdin.
// These behaviors cause the test to hang indefinitely in CI environments.
// Testing composeResponseToEmail requires mocking the runCommand function
// or running in an environment where these shell commands won't block.
func TestEmailResponseShowsActionsTaken(t *testing.T) {
	t.Skip("Skipping: composeResponseToEmail runs shell commands (go test, sendmail) that block in CI")
	agent := &YoloAgent{config: NewYoloConfig(".")}
	tex := NewToolExecutor(".", agent)

	email := EmailMessage{
		From:    "scott@stg.net",
		Subject: "How is it going?",
		Content: "What's the current test coverage? Are all tests passing?",
	}

	response := tex.composeResponseToEmail(email)

	t.Logf("Response:\n%s\n", response)

	// Should show actions taken
	if !strings.Contains(response, "ACTIONS TAKEN:") {
		t.Error("Expected response to include ACTIONS TAKEN section showing what was done")
	}

	// Should show at least one action
	hasAction := strings.Contains(response, "Generating system status") ||
		strings.Contains(response, "Gathering current status") ||
		strings.Contains(response, "Checking test coverage")
	if !hasAction {
		t.Error("Expected response to show at least one action taken")
	}

	// Should provide actual answers with data
	hasData := strings.Contains(response, "Test Coverage:") ||
		strings.Contains(response, "tests") && strings.Contains(response, "passing") ||
		strings.Contains(response, "status")
	if !hasData {
		t.Error("Expected response to include actual system data/status information")
	}

	// Should be personalized for Scott
	if !strings.Contains(response, "Hi Scott") {
		t.Error("Expected personalized greeting for Scott")
	}
}
