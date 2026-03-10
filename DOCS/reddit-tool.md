# Reddit Tool Documentation

## Overview

The `reddit` tool provides access to Reddit's public API without requiring authentication. It can search posts, list subreddit content, and retrieve thread details with comments.

## Quick Reference

```json
{
  "name": "reddit",
  "arguments": {
    "action": "search",
    "query": "go programming tips",
    "limit": 10
  }
}
```

## Actions

### 1. Search (action: "search")

Search Reddit globally for posts matching a query.

**Parameters:**
- `action`: "search" (required)
- `query`: Search terms (required)
- `limit`: Results to return (optional, default: 25, max: 100)

**Example:**
```json
{
  "name": "reddit",
  "arguments": {
    "action": "search",
    "query": "golang concurrency best practices",
    "limit": 10
  }
}
```

### 2. Subreddit Posts (action: "subreddit")

List recent posts from a specific subreddit.

**Parameters:**
- `action`: "subreddit" (required)
- `subreddit`: Subreddit name without 'r/' (required, e.g., "golang", not "r/golang")
- `limit`: Posts to return (optional, default: 25, max: 100)

**Example:**
```json
{
  "name": "reddit",
  "arguments": {
    "action": "subreddit",
    "subreddit": "golang",
    "limit": 15
  }
}
```

### 3. Thread Details (action: "thread")

Get a specific post and its comments by ID.

**Parameters:**
- `action`: "thread" (required)
- `post_id`: Reddit post/comment ID (required)
- `limit`: Comments to include (optional, default: 25, max: 100)

**Example:**
```json
{
  "name": "reddit",
  "arguments": {
    "action": "thread",
    "post_id": "abc123"
  }
}
```

## Use Cases

### Research Community Discussions
Find what developers are saying about specific topics:
```json
{
  "name": "reddit",
  "arguments": {
    "action": "search",
    "query": "Go vs Rust performance",
    "limit": 20
  }
}
```

### Monitor Specific Communities
Check recent activity in relevant subreddits:
```json
{
  "name": "reddit",
  "arguments": {
    "action": "subreddit",
    "subreddit": "programming",
    "limit": 30
  }
}
```

### Analyze Hot Topics
Get full thread with comments to understand discussions:
```json
{
  "name": "reddit",
  "arguments": {
    "action": "thread",
    "post_id": "xyz789"
  }
}
```

## Response Format

### Search Results
```
Reddit search results for "golang testing":

1. [How to write good Go tests]
   r/golang • Posted by u/tester • 25 comments • 45 points
   Writing effective unit tests in Go requires understanding...

2. [Testing patterns that scale]
   r/golang • Posted by u/devops_guru • 18 comments • 32 points
   Here are some testing patterns I've found useful...
```

### Subreddit Listing
```
Recent posts from r/golang:

1. [Announcing Go 1.24]
   Posted 2 hours ago • 156 comments • 2.3k points

2. [Understanding Go generics]
   Posted 5 hours ago • 89 comments • 1.1k points
```

### Thread with Comments
```
Thread: Understanding Go interfaces

Post by u/gomaster • 45 comments • 234 points
Interfaces in Go are implicitly implemented...

Comments:
- user1: Great explanation! This helped me understand...
- user2: One thing to add - you can also use empty interfaces...
```

## Limitations

- No authentication required (public API only)
- Rate limited by Reddit's public API
- Maximum 100 results per request
- Comments in threads may be truncated based on limit parameter

## Best Practices

1. **Use specific queries** for search to get relevant results
2. **Set appropriate limits** - don't fetch more than you need
3. **Check subreddit spelling** - common misspellings won't work
4. **Post IDs from URLs** - extract from share links or URLs like `reddit.com/comments/abc123`

## Error Handling

The tool returns descriptive error messages for:
- Missing required parameters
- Invalid action types
- Network errors
- Reddit API errors
- Invalid post IDs

Example error response:
```
Error: 'query' parameter is required for 'search' action
```

## Integration Examples

### Before Implementing New Feature
Research community experiences with similar tools:
```json
{
  "name": "reddit",
  "arguments": {
    "action": "search",
    "query": "AI coding assistant experiences",
    "limit": 15
  }
}
```

### Debugging Issues
Find if others have encountered similar problems:
```json
{
  "name": "reddit",
  "arguments": {
    "action": "subreddit",
    "subreddit": "golang",
    "limit": 50
  }
}
```

## Related Tools

- `web_search` - For broader internet research
- `think` - For planning before making Reddit queries

---

*Last updated: 2026-03-10*
