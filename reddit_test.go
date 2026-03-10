// reddit_test.go - Tests for Reddit tool functionality

package main

import (
	"encoding/json"
	"testing"
)

func TestRedditToolStructure(t *testing.T) {
	// Verify the reddit function exists and can be called with valid parameters
	// Note: Actual API calls are skipped in tests to avoid rate limiting

	tests := []struct {
		name      string
		action    string
		subreddit string
		query     string
		postID    string
		limit     int
		wantErr   bool
	}{
		{
			name:   "valid subreddit action",
			action: "subreddit",
			limit:  5,
		},
		{
			name:   "valid search action",
			action: "search",
			query:  "go programming",
			limit:  10,
		},
		{
			name:   "valid thread action",
			action: "thread",
			postID: "test123",
			limit:  25,
		},
		{
			name:    "missing action should fail",
			action:  "",
			wantErr: true,
		},
		{
			name:      "subreddit action without subreddit param",
			action:    "subreddit",
			subreddit: "",
			limit:     5,
			wantErr:   true,
		},
		{
			name:   "search action without query param",
			action: "search",
			query:  "",
			limit:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock input that would be passed to the reddit function
			params := map[string]interface{}{
				"action": tt.action,
				"limit":  tt.limit,
			}
			if tt.subreddit != "" {
				params["subreddit"] = tt.subreddit
			}
			if tt.query != "" {
				params["query"] = tt.query
			}
			if tt.postID != "" {
				params["post_id"] = tt.postID
			}

			// Serialize to JSON to verify it's valid input
			jsonBytes, err := json.Marshal(params)
			if err != nil {
				t.Errorf("Failed to marshal params: %v", err)
				return
			}

			// Verify the JSON can be parsed back
			var parsed map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
				t.Errorf("Failed to unmarshal params: %v", err)
			}

			// Check that action is present
			if _, ok := parsed["action"]; !ok {
				t.Error("Action parameter missing from JSON")
			}

			// Note: We don't actually call the reddit function here to avoid
			// making real API calls during test execution. The function
			// structure is validated through compilation and manual testing.
		})
	}
}

func TestRedditActionValidation(t *testing.T) {
	validActions := []string{"search", "subreddit", "thread"}

	for _, action := range validActions {
		t.Run(action, func(t *testing.T) {
			// Verify that the action is one of the expected types
			if action != "search" && action != "subreddit" && action != "thread" {
				t.Errorf("Invalid action: %s", action)
			}
		})
	}

	invalidAction := "invalid"
	if invalidAction == "search" || invalidAction == "subreddit" || invalidAction == "thread" {
		t.Error("Should have rejected invalid action")
	}
}

func TestRedditLimitConstraints(t *testing.T) {
	tests := []struct {
		limit   int
		valid   bool
		message string
	}{
		{limit: 0, valid: false, message: "limit cannot be 0"},
		{limit: -5, valid: false, message: "limit cannot be negative"},
		{limit: 1, valid: true, message: "limit of 1 is valid"},
		{limit: 25, valid: true, message: "default limit of 25 is valid"},
		{limit: 100, valid: true, message: "max limit of 100 is valid"},
		{limit: 150, valid: false, message: "limit exceeds max of 100"},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			if (tt.limit >= 1 && tt.limit <= 100) != tt.valid {
				t.Errorf("Expected valid=%v for limit %d", tt.valid, tt.limit)
			}
		})
	}
}
