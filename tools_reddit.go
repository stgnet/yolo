package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ─── Reddit Tool Implementation ─────────────────────────────────────

type redditPost struct {
	Kind        string     `json:"kind"`
	Data        redditData `json:"data"`
	IsSelf      bool       `json:"is_self"`
	Subreddit   string     `json:"subreddit"`
	Title       string     `json:"title,omitempty"`
	Selftext    string     `json:"selftext,omitempty"`
	Body        string     `json:"body,omitempty"`
	URL         string     `json:"url,omitempty"`
	Score       int        `json:"score,omitempty"`
	NumComments int        `json:"num_comments,omitempty"`
	Created     float64    `json:"created_utc,omitempty"`
	ID          string     `json:"id,omitempty"`
	Author      string     `json:"author,omitempty"`
}

type redditData struct {
	Title       string  `json:"title,omitempty"`
	Selftext    string  `json:"selftext,omitempty"`
	Body        string  `json:"body,omitempty"`
	URL         string  `json:"url,omitempty"`
	Score       int     `json:"score,omitempty"`
	NumComments int     `json:"num_comments,omitempty"`
	Created     float64 `json:"created_utc,omitempty"`
	ID          string  `json:"id,omitempty"`
	Author      string  `json:"author,omitempty"`
	Subreddit   string  `json:"subreddit,omitempty"`
}

type redditListing struct {
	Kind string         `json:"kind"`
	Data redditChildren `json:"data"`
}

type redditChildren struct {
	Children []redditPostWrapper `json:"children"`
	After    string              `json:"after"`
	Before   string              `json:"before"`
}

type redditPostWrapper struct {
	Kind string     `json:"kind"`
	Data redditPost `json:"data"`
}

func (t *ToolExecutor) reddit(args map[string]any) string {
	action := getStringArg(args, "action", "")
	if action == "" {
		return errorMessage("'action' parameter is required. Options: 'search', 'subreddit', 'thread'")
	}

	limit := getIntArg(args, "limit", 25)
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 25
	}

	var requestURL string

	switch action {
	case "search":
		query := getStringArg(args, "query", "")
		if query == "" {
			return errorMessage("'query' parameter is required for 'search' action")
		}
		requestURL = fmt.Sprintf("https://www.reddit.com/search.json?q=%s&limit=%d",
			url.QueryEscape(query), limit)

	case "subreddit":
		subreddit := getStringArg(args, "subreddit", "")
		if subreddit == "" {
			return errorMessage("'subreddit' parameter is required for 'subreddit' action")
		}
		subreddit = strings.TrimPrefix(subreddit, "r/")
		requestURL = fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=%d", subreddit, limit)

	case "thread":
		postID := getStringArg(args, "post_id", "")
		if postID == "" {
			return errorMessage("'post_id' parameter is required for 'thread' action")
		}
		requestURL = fmt.Sprintf("https://www.reddit.com/comments/%s/.json", postID)

	default:
		return errorMessage("unknown action '%s'. Options: 'search', 'subreddit', 'thread'", action)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return errorMessage("could not create request: %v", err)
	}

	req.Header.Set("User-Agent", "YOLO-Agent/1.0 (by /u/yolo)")

	resp, err := client.Do(req)
	if err != nil {
		return errorMessage("could not fetch from Reddit: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return errorMessage("Reddit returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errorMessage("could not read response: %v", err)
	}

	switch action {
	case "thread":
		return t.parseThreadResponse(action, body)
	default:
		return t.parseListingResponse(action, body)
	}
}

func (t *ToolExecutor) parseListingResponse(action string, data []byte) string {
	var listing redditListing

	if err := json.Unmarshal(data, &listing); err != nil {
		return errorMessage("parsing JSON: %v", err)
	}

	if len(listing.Data.Children) == 0 {
		return "No results found"
	}

	var sb strings.Builder

	switch action {
	case "search":
		sb.WriteString(fmt.Sprintf("Search results (showing %d of available):\n\n", len(listing.Data.Children)))
	case "subreddit":
		subreddit := listing.Data.Children[0].Data.Subreddit
		sb.WriteString(fmt.Sprintf("Hot posts in r/%s (showing %d):\n\n", subreddit, len(listing.Data.Children)))
	}

	for i, post := range listing.Data.Children {
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}

		title := post.Data.Title
		if title == "" && post.Data.Selftext != "" {
			lines := strings.SplitN(post.Data.Selftext, "\n", 2)
			title = "[Self Post] " + lines[0]
		}

		sb.WriteString(fmt.Sprintf("**%s**\n", title))

		if post.Data.Author != "" {
			sb.WriteString(fmt.Sprintf("By: u/%s | Score: %d | Comments: %d\n",
				post.Data.Author, post.Data.Score, post.Data.NumComments))
		}

		if post.Data.URL != "" && !strings.Contains(post.Data.URL, "reddit.com") {
			sb.WriteString(fmt.Sprintf("URL: %s\n", post.Data.URL))
		}

		if post.Data.Selftext != "" {
			text := strings.TrimSpace(post.Data.Selftext)
			if len(text) > 500 {
				text = text[:500] + "..."
			}
			sb.WriteString(fmt.Sprintf("\n%s\n", text))
		}

		postURL := fmt.Sprintf("https://www.reddit.com%s", post.Data.URL)
		if !strings.HasPrefix(post.Data.URL, "/") {
			postURL = fmt.Sprintf("https://www.reddit.com/r/%s/comments/%s/",
				post.Data.Subreddit, post.Data.ID)
		}
		sb.WriteString(fmt.Sprintf("\n[Read more](%s)", postURL))
	}

	return sb.String()
}

func (t *ToolExecutor) parseThreadResponse(action string, data []byte) string {
	var listing redditListing

	if err := json.Unmarshal(data, &listing); err != nil {
		return errorMessage("parsing JSON: %v", err)
	}

	if len(listing.Data.Children) == 0 {
		return "No results found"
	}

	originalPost := listing.Data.Children[0]

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n", originalPost.Data.Title))
	sb.WriteString(fmt.Sprintf("By: u/%s | Score: %d | Posted: %s\n\n",
		originalPost.Data.Author,
		originalPost.Data.Score,
		formatRedditTimestamp(originalPost.Data.Created)))

	if originalPost.Data.Selftext != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", strings.TrimSpace(originalPost.Data.Selftext)))
	}

	if originalPost.Data.URL != "" && !strings.Contains(originalPost.Data.URL, "reddit.com") {
		sb.WriteString(fmt.Sprintf("[External Link](%s)\n\n", originalPost.Data.URL))
	}

	if len(listing.Data.Children) > 1 {
		sb.WriteString("\n## Top Comments:\n\n")

		for i := 1; i < len(listing.Data.Children); i++ {
			comment := listing.Data.Children[i]
			t.appendComment(&sb, comment.Data, 0)
			if i < len(listing.Data.Children)-1 {
				sb.WriteString("\n---\n\n")
			}
		}
	}

	return sb.String()
}

func (t *ToolExecutor) appendComment(sb *strings.Builder, post redditPost, depth int) {
	if depth > 3 {
		return
	}

	indent := strings.Repeat("  ", depth)

	body := strings.TrimSpace(post.Body)
	if body == "" {
		body = strings.TrimSpace(post.Data.Body)
	}
	if body == "" {
		body = strings.TrimSpace(post.Selftext)
	}
	if body == "" && post.Kind != "t1" {
		body = "[Deleted or removed]"
	}

	author := post.Author
	score := post.Score

	sb.WriteString(fmt.Sprintf("%s**%s** (%d points)\n", indent, author, score))
	if body != "" {
		if len(body) > 1000 {
			body = body[:1000] + "..."
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", indent, body))
	}
}

func formatRedditTimestamp(timestamp float64) string {
	t := time.Unix(int64(timestamp), 0)
	return t.Format("January 2, 2006 at 3:04 PM MST")
}
