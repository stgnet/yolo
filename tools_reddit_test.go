package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedditPostParsing(t *testing.T) {
	tests := []struct {
		name       string
		json       string
		wantTitle  string
		wantSub    string
		wantAuthor string
	}{
		{
			name: "self-post",
			json: `{
				"kind": "t1",
				"data": {
					"body_html": "<div>Test content</div>",
					"permalink": "/r/golang/comments/abc123/test_post/",
					"author": "testuser"
				}
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
					"permalink": "/r/golang/comments/abc123/test_post/",
					"author": "testuser",
					"is_self": false
				}
			}`,
			wantTitle:  "Test Title",
			wantSub:    "golang",
			wantAuthor: "testuser",
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

			// Extract subreddit from permalink
			subreddit := extractSubreddit(post.Data.Permalink)
			if subreddit != tt.wantSub {
				t.Errorf("Expected subreddit '%s', got '%s'", tt.wantSub, subreddit)
			}

			if post.Data.Author != tt.wantAuthor {
				t.Errorf("Expected author '%s', got '%s'", tt.wantAuthor, post.Data.Author)
			}
		})
	}
}

func TestRedditPermalinkParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/r/golang/comments/abc123/test-post/", "golang"},
		{"/r/rust/comments/def456/another-post", "rust"},
		{"/r/programming/comments/ghi789/discussion?foo=bar", "programming"},
		{"", ""},
	}

	for _, tt := range tests {
		result := extractSubreddit(tt.input)
		if result != tt.expected {
			t.Errorf("extractSubreddit(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestRedditURLConstruction(t *testing.T) {
	tests := []struct {
		action   string
		params   map[string]string
		expected string
	}{
		{
			action:   "search",
			params:   map[string]string{"q": "golang", "sort": "relevance"},
			expected: "https://www.reddit.com/search.json?q=golang&sort=relevance",
		},
		{
			action:   "subreddit",
			params:   map[string]string{"sr": "golang"},
			expected: "https://www.reddit.com/r/golang/.json",
		},
		{
			action:   "thread",
			params:   map[string]string{"id": "abc123"},
			expected: "https://www.reddit.com/comments/abc123/.json",
		},
	}

	for _, tt := range tests {
		result := buildRedditURL(tt.action, tt.params)
		if !strings.HasPrefix(result, "https://www.reddit.com/") {
			t.Errorf("Expected URL to start with 'https://www.reddit.com/', got '%s'", result)
		}

		for _, param := range tt.params {
			if !strings.Contains(result, param) {
				t.Errorf("Expected URL to contain param '%s', got '%s'", param, result)
			}
		}
	}
}

func TestRedditRateLimiter(t *testing.T) {
	// Create a new limiter with very short cooldown for testing
	r := &redditClient{}
	r.rateLimiter = newRateLimiter(10*time.Millisecond, 5)

	// Should be able to make requests up to the limit
	for i := 0; i < 3; i++ {
		allowed := r.canMakeRequest()
		if !allowed {
			t.Errorf("Request %d should be allowed but was blocked", i)
		}
	}

	// Fourth request should still be OK (limit is 5 per second)
	allowed := r.canMakeRequest()
	if !allowed {
		t.Error("Fourth request should be allowed")
	}
}

func TestRedditErrorParsing(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantErr  bool
		errMsg   string
	}{
		{
			name: "rate limited",
			json: `{"error": "RATE LIMIT EXCEEDED"}`,
			wantErr: true,
			errMsg:  "rate limited",
		},
		{
			name: "invalid request",
			json: `{"error": "INVALID REQUEST"}`,
			wantErr: true,
			errMsg:  "invalid request",
		},
		{
			name:    "valid response",
			json:    `[{"kind": "t3", "data": {}}]`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &redditClient{}
			err := r.parseResponse([]byte(tt.json))
			
			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errMsg, err)
				}
			}
		})
	}
}
