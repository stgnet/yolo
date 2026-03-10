package main

import (
	"strings"
	"testing"
)

// TestAppendComment tests comment formatting for Reddit threads
func TestAppendComment(t *testing.T) {
	executor := &ToolExecutor{}

	tests := []struct {
		name     string
		postData redditPost
	}{
		{
			name: "comment with author and body",
			postData: redditPost{
				Kind: "t1",
				Data: redditData{
					Author:   "test_user",
					Score:    10,
					Selftext: "Comment body text",
				},
			},
		},
		{
			name: "comment without selftext",
			postData: redditPost{
				Kind: "t1",
				Data: redditData{
					Author: "anonymous",
					Score:  5,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sb strings.Builder
			executor.appendComment(&sb, tt.postData, 0)
			result := sb.String()

			if result == "" {
				t.Error("Expected non-empty comment result")
			}

			// Check for author and score in output
			if !strings.Contains(result, tt.postData.Data.Author) {
				t.Errorf("Expected result to contain author %q", tt.postData.Data.Author)
			}
		})
	}
}

// TestFormatRedditTimestamp tests timestamp formatting for Reddit posts
func TestFormatRedditTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		timestamp float64
		contains  string
	}{
		{
			name:      "recent timestamp",
			timestamp: 1700000000, // Nov 2023
			contains:  "November",
		},
		{
			name:      "old timestamp",
			timestamp: 1000000000, // Sep 2012
			contains:  "September",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRedditTimestamp(tt.timestamp)
			if result == "" {
				t.Error("Expected non-empty timestamp format")
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Expected result to contain %q, got: %q", tt.contains, result)
			}
		})
	}
}
