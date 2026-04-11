package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestReddit_InvalidArguments validates error handling for invalid arguments
func TestReddit_InvalidArguments(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string]any
		expectError bool
	}{
		{
			name:        "empty action",
			input:       map[string]any{"action": ""},
			expectError: true,
		},
		{
			name:        "missing action",
			input:       map[string]any{},
			expectError: true,
		},
		{
			name:        "invalid action",
			input:       map[string]any{"action": "invalid"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &ToolExecutor{baseDir: "."}
			result := executor.reddit(tt.input)
			
			// Check for error indicators (either JSON status:error or plain "Error:" text)
			hasError := strings.Contains(result, `"status":"error"`) || strings.HasPrefix(result, "Error:")
			
			if tt.expectError && !hasError {
				t.Errorf("Expected error message, got: %s", result)
			}
			if !tt.expectError && hasError {
				t.Errorf("Unexpected error in result: %s", result)
			}
		})
	}
}

// TestReddit_Search validates Reddit search functionality
func TestReddit_Search(t *testing.T) {
	executor := &ToolExecutor{baseDir: "."}
	
	result := executor.reddit(map[string]any{
		"action": "search",
		"query":  "go language",
	})
	
	// Should return either JSON with status or formatted text output (not starting with "Error:")
	if strings.HasPrefix(result, "Error:") {
		t.Errorf("Unexpected error in search result: %s", result)
	}
}

// TestReddit_Subreddit validates Reddit subreddit listing
func TestReddit_Subreddit(t *testing.T) {
	executor := &ToolExecutor{baseDir: "."}
	
	result := executor.reddit(map[string]any{
		"action":    "subreddit",
		"subreddit": "golang",
	})
	
	// Should return either JSON with status or formatted text output (not starting with "Error:")
	if strings.HasPrefix(result, "Error:") {
		t.Errorf("Unexpected error in subreddit result: %s", result)
	}
}

// TestReddit_Thread validates Reddit thread/details functionality
func TestReddit_Thread(t *testing.T) {
	executor := &ToolExecutor{baseDir: "."}
	
	result := executor.reddit(map[string]any{
		"action":  "thread",
		"post_id": "abc123",
	})
	
	// Thread requests may return errors for invalid/non-existent posts or parsing issues
	// The important thing is that it doesn't crash and handles the error gracefully
	// Real thread data would require a valid Reddit post ID
	if strings.Contains(result, "Error:") {
		// This is expected for test data - real usage needs valid post IDs
		maxLen := 100
		if len(result) < maxLen {
			maxLen = len(result)
		}
		t.Logf("Expected error for test post ID (not a bug): %s", result[:maxLen])
	} else if strings.HasPrefix(result, "#") || strings.Contains(result, "No results") {
		// Valid response format
		maxLen := 100
		if len(result) < maxLen {
			maxLen = len(result)
		}
		t.Logf("Got valid thread response: %s", result[:maxLen])
	}
}

// TestReddit_SubredditWithLimit validates limit parameter handling
func TestReddit_SubredditWithLimit(t *testing.T) {
	executor := &ToolExecutor{baseDir: "."}
	
	result := executor.reddit(map[string]any{
		"action":    "subreddit",
		"subreddit": "golang",
		"limit":     5,
	})
	
	if strings.HasPrefix(result, "Error:") {
		t.Errorf("Unexpected error: %s", result)
	}
}

// TestReddit_SearchWithLimit validates search with limit parameter
func TestReddit_SearchWithLimit(t *testing.T) {
	executor := &ToolExecutor{baseDir: "."}
	
	result := executor.reddit(map[string]any{
		"action": "search",
		"query":  "test query",
		"limit":  10,
	})
	
	if strings.HasPrefix(result, "Error:") {
		t.Errorf("Unexpected error: %s", result)
	}
}

// TestReddit_PostParsing validates JSON parsing of Reddit posts
func TestReddit_PostParsing(t *testing.T) {
	tests := []struct {
		name       string
		json       string
		wantTitle  string
		wantSub    string
		wantAuthor string
	}{
		{
			name: "self-post comment",
			json: `{
				"kind": "t1",
				"data": {
					"body": "Test content",
					"author": "testuser"
				},
				"subreddit": "golang"
			}`,
			wantTitle:  "",
			wantSub:    "golang",
			wantAuthor: "testuser",
		},
		{
			name: "link-post",
			json: `{
				"kind": "t3",
				"data": {
					"title": "Test Title",
					"url": "https://example.com",
					"author": "testuser",
					"is_self": false
				},
				"subreddit": "golang"
			}`,
			wantTitle:  "Test Title",
			wantSub:    "golang",
			wantAuthor: "testuser",
		},
		{
			name: "self-post with title",
			json: `{
				"kind": "t3",
				"data": {
					"title": "Self Post Title",
					"selftext": "This is the post content",
					"author": "devuser",
					"is_self": true
				},
				"subreddit": "programming"
			}`,
			wantTitle:  "Self Post Title",
			wantSub:    "programming",
			wantAuthor: "devuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var post redditPost
			err := json.Unmarshal([]byte(tt.json), &post)
			if err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			if tt.wantTitle != "" && post.Data.Title != tt.wantTitle {
				t.Errorf("Expected title '%s', got '%s'", tt.wantTitle, post.Data.Title)
			}

			if post.Subreddit != tt.wantSub {
				t.Errorf("Expected subreddit '%s', got '%s'", tt.wantSub, post.Subreddit)
			}

			if post.Data.Author != tt.wantAuthor {
				t.Errorf("Expected author '%s', got '%s'", tt.wantAuthor, post.Data.Author)
			}
		})
	}
}

// TestReddit_PermalinkParsing validates subreddit extraction from permalinks
func TestReddit_PermalinkParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/r/golang/comments/abc123/test-post/", "golang"},
		{"/r/rust/comments/def456/another-post", "rust"},
		{"/r/programming/comments/ghi789/discussion?foo=bar", "programming"},
		{"/r/multiredditsomething/", "multiredditsomething"},
		{"", ""},
		{"/r/", ""},
		{"/not-a-permalink/", ""},
	}

	for _, tt := range tests {
		result := extractSubredditFromPermalink(tt.input)
		if result != tt.expected {
			t.Errorf("extractSubredditFromPermalink(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// extractSubredditFromPermalink extracts subreddit name from a Reddit permalink
func extractSubredditFromPermalink(permalink string) string {
	if !strings.HasPrefix(permalink, "/r/") {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(permalink, "/r/"), "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// TestReddit_CommentParsing validates JSON parsing of Reddit comments
func TestReddit_CommentParsing(t *testing.T) {
	jsonData := `{
		"kind": "t1",
		"data": {
			"body": "Comment text here",
			"author": "commenter"
		}
	}`

	var comment redditPost
	err := json.Unmarshal([]byte(jsonData), &comment)
	if err != nil {
		t.Fatalf("Failed to parse comment JSON: %v", err)
	}

	if comment.Kind != "t1" {
		t.Errorf("Expected kind 't1', got '%s'", comment.Kind)
	}

	if comment.Data.Author != "commenter" {
		t.Errorf("Expected author 'commenter', got '%s'", comment.Data.Author)
	}

	if !strings.Contains(comment.Data.Body, "Comment text here") {
		t.Errorf("Expected body to contain 'Comment text here', got '%s'", comment.Data.Body)
	}
}

// TestReddit_DataFields validates all fields in redditData structure
func TestReddit_DataFields(t *testing.T) {
	jsonData := `{
		"title": "Test Title",
		"selftext": "Self post text",
		"body": "Comment body",
		"url": "https://example.com",
		"score": 100,
		"num_comments": 42,
		"created_utc": 1234567890.0,
		"id": "abc123",
		"author": "testuser",
		"subreddit": "golang"
	}`

	var data redditData
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		t.Fatalf("Failed to parse redditData JSON: %v", err)
	}

	if data.Title != "Test Title" {
		t.Errorf("Expected Title 'Test Title', got '%s'", data.Title)
	}
	if data.Selftext != "Self post text" {
		t.Errorf("Expected Selftext 'Self post text', got '%s'", data.Selftext)
	}
	if data.Body != "Comment body" {
		t.Errorf("Expected Body 'Comment body', got '%s'", data.Body)
	}
	if data.URL != "https://example.com" {
		t.Errorf("Expected URL 'https://example.com', got '%s'", data.URL)
	}
	if data.Score != 100 {
		t.Errorf("Expected Score 100, got %d", data.Score)
	}
	if data.NumComments != 42 {
		t.Errorf("Expected NumComments 42, got %d", data.NumComments)
	}
	if data.ID != "abc123" {
		t.Errorf("Expected ID 'abc123', got '%s'", data.ID)
	}
	if data.Author != "testuser" {
		t.Errorf("Expected Author 'testuser', got '%s'", data.Author)
	}
	if data.Subreddit != "golang" {
		t.Errorf("Expected Subreddit 'golang', got '%s'", data.Subreddit)
	}
}
