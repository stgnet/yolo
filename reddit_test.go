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
				Kind:   "t1",
				Author: "test_user",
				Score:  10,
				Body:   "Comment body text",
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
				Kind:   "t1",
				Author: "anonymous",
				Score:  5,
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

// TestParseListingResponse tests parsing of Reddit search/subreddit results
func TestParseListingResponse(t *testing.T) {
	executor := &ToolExecutor{}

	// Mock JSON response for search results - matching real Reddit API structure
	mockJSON := `{
		"data": {
			"children": [
				{
					"kind": "t3",
					"data": {
						"title": "Test Post Title",
						"author": "test_author",
						"score": 42,
						"num_comments": 10,
						"url": "/r/golang/comments/abc123/test/",
						"subreddit": "golang",
						"selftext": "This is a test post body.",
						"id": "abc123"
					}
				},
				{
					"kind": "t3",
					"data": {
						"title": "Another Post",
						"author": "another_user",
						"score": 15,
						"num_comments": 3,
						"url": "https://example.com",
						"subreddit": "golang",
						"selftext": "",
						"id": "def456"
					}
				}
			]
		}
	}`

	tests := []struct {
		name     string
		action   string
		jsonData string
		checks   []string
	}{
		{
			name:     "search action",
			action:   "search",
			jsonData: mockJSON,
			checks:   []string{"Test Post Title", "test_author", "Score: 42"},
		},
		{
			name:     "subreddit action",
			action:   "subreddit",
			jsonData: mockJSON,
			checks:   []string{"r/golang", "Another Post", "example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.parseListingResponse(tt.action, []byte(tt.jsonData))

			if result == "" {
				t.Error("Expected non-empty parsing result")
			}

			for _, check := range tt.checks {
				if !strings.Contains(result, check) {
					t.Errorf("Expected result to contain %q\nGot: %s", check, result)
				}
			}
		})
	}
}

// TestParseThreadResponse tests parsing of Reddit thread/post details with comments
func TestParseThreadResponse(t *testing.T) {
	executor := &ToolExecutor{}

	// Mock JSON response for a thread with comments - matching real Reddit API structure
	mockJSON := `{
		"data": {
			"children": [
				{
					"kind": "t3",
					"data": {
						"title": "Original Post Title",
						"author": "op_user",
						"score": 100,
						"created_utc": 1700000000,
						"url": "/r/golang/comments/xyz789/",
						"subreddit": "golang",
						"selftext": "This is the original post content.",
						"id": "xyz789"
					}
				},
				{
					"kind": "t1",
					"data": {
						"id": "comment1",
						"author": "commenter1",
						"score": 25,
						"body": "First comment here.",
						"parent_id": "t3_xyz789"
					}
				},
				{
					"kind": "t1",
					"data": {
						"id": "comment2",
						"author": "commenter2",
						"score": 15,
						"body": "Second comment with more text.",
						"parent_id": "t3_xyz789"
					}
				}
			]
		}
	}`

	result := executor.parseThreadResponse("thread", []byte(mockJSON))

	if result == "" {
		t.Fatal("Expected non-empty thread parsing result")
	}

	requiredContents := []string{
		"Original Post Title",
		"op_user",
		"Score: 100",
		"This is the original post content.",
		"Top Comments:",
		"commenter1",
		"commenter2",
	}

	for _, req := range requiredContents {
		if !strings.Contains(result, req) {
			t.Errorf("Expected result to contain %q\nGot: %s", req, result)
		}
	}
}

// TestRedditTool tests the full reddit tool with various actions
func TestRedditTool(t *testing.T) {
	executor := &ToolExecutor{}

	tests := []struct {
		name        string
		args        map[string]any
		expectErr   bool
		errContains string
	}{
		{
			name:        "missing action parameter",
			args:        map[string]any{},
			expectErr:   true,
			errContains: "action' parameter is required",
		},
		{
			name: "search without query",
			args: map[string]any{
				"action": "search",
			},
			expectErr:   true,
			errContains: "query' parameter is required",
		},
		{
			name: "subreddit without name",
			args: map[string]any{
				"action": "subreddit",
			},
			expectErr:   true,
			errContains: "subreddit' parameter is required",
		},
		{
			name: "thread without post_id",
			args: map[string]any{
				"action": "thread",
			},
			expectErr:   true,
			errContains: "post_id' parameter is required",
		},
		{
			name: "invalid action",
			args: map[string]any{
				"action": "invalid_action",
			},
			expectErr:   true,
			errContains: "unknown action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.reddit(tt.args)

			if !tt.expectErr && result == "" {
				t.Error("Expected non-empty result")
			}

			if tt.expectErr {
				if result == "" {
					t.Error("Expected error message but got empty result")
				}
				if !strings.Contains(result, tt.errContains) {
					t.Errorf("Expected error to contain %q, got: %s", tt.errContains, result)
				}
			}
		})
	}
}
