package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestRedditPostParsing validates JSON parsing of Reddit posts
func TestRedditPostParsing(t *testing.T) {
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

// TestRedditPermalinkParsing validates subreddit extraction from permalinks
func TestRedditPermalinkParsing(t *testing.T) {
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

// TestRedditCommentParsing validates JSON parsing of Reddit comments
func TestRedditCommentParsing(t *testing.T) {
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

// TestRedditDataFields validates all fields in redditData structure
func TestRedditDataFields(t *testing.T) {
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

// TestRedditEmptyFields validates handling of missing/null fields
func TestRedditEmptyFields(t *testing.T) {
	jsonData := `{
		"kind": "t3",
		"data": {}
	}`

	var post redditPost
	err := json.Unmarshal([]byte(jsonData), &post)
	if err != nil {
		t.Fatalf("Failed to parse empty data JSON: %v", err)
	}

	// All fields should have zero values
	if post.Data.Title != "" {
		t.Errorf("Expected empty Title, got '%s'", post.Data.Title)
	}
	if post.Data.Author != "" {
		t.Errorf("Expected empty Author, got '%s'", post.Data.Author)
	}
	if post.Data.URL != "" {
		t.Errorf("Expected empty URL, got '%s'", post.Data.URL)
	}
	if post.Data.Score != 0 {
		t.Errorf("Expected Score 0, got %d", post.Data.Score)
	}
}

// TestRedditListingParsing validates parsing of Reddit listing responses
func TestRedditListingParsing(t *testing.T) {
	jsonData := `{
		"kind": "Listing",
		"data": {
			"children": [
				{
					"kind": "t3",
					"data": {
						"data": {
							"title": "First post",
							"author": "user1"
						},
						"subreddit": "golang"
					}
				},
				{
					"kind": "t3",
					"data": {
						"data": {
							"title": "Second post",
							"author": "user2"
						},
						"subreddit": "rust"
					}
				}
			],
			"after": "t3_abc123",
			"before": ""
		}
	}`

	var listing redditListing
	err := json.Unmarshal([]byte(jsonData), &listing)
	if err != nil {
		t.Fatalf("Failed to parse listing JSON: %v", err)
	}

	if listing.Kind != "Listing" {
		t.Errorf("Expected kind 'Listing', got '%s'", listing.Kind)
	}

	if len(listing.Data.Children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(listing.Data.Children))
	}

	if listing.Data.Children[0].Data.Data.Title != "First post" {
		t.Errorf("Expected first title 'First post', got '%s'", listing.Data.Children[0].Data.Data.Title)
	}

	if listing.Data.After != "t3_abc123" {
		t.Errorf("Expected after 't3_abc123', got '%s'", listing.Data.After)
	}
}

// TestRedditPostWrapper validates nested post structure in listings
func TestRedditPostWrapper(t *testing.T) {
	jsonData := `{
		"kind": "t3",
		"data": {
			"title": "Nested post",
			"author": "testuser",
			"score": 50,
			"is_self": true,
			"selftext": "Post content",
			"url": "https://example.com/post",
			"subreddit": "testing"
		}
	}`

	var wrapper redditPostWrapper
	err := json.Unmarshal([]byte(jsonData), &wrapper)
	if err != nil {
		t.Fatalf("Failed to parse post wrapper JSON: %v", err)
	}

	if wrapper.Kind != "t3" {
		t.Errorf("Expected kind 't3', got '%s'", wrapper.Kind)
	}

	// The title is in the inner Data.Data.Title path
	// redditPostWrapper -> Data (redditPost) -> Title field
	if wrapper.Data.Title != "Nested post" {
		t.Errorf("Expected title 'Nested post', got '%s'", wrapper.Data.Title)
	}

	if !wrapper.Data.IsSelf {
		t.Error("Expected IsSelf to be true")
	}

	if wrapper.Data.Selftext != "Post content" {
		t.Errorf("Expected selftext 'Post content', got '%s'", wrapper.Data.Selftext)
	}
}
