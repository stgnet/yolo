# YOLO Unit Tests

This directory contains comprehensive unit tests for the YOLO (Your Own Living Operator) agent project.

## Test Files

- **agent_unit_tests.go** - Tests for main agent functions, file operations, todo management, and email sending
- **email_unit_tests.go** - Tests for email parsing, MIME decoding, and multipart email handling
- **search_and_reddit_tests.go** - Tests for web search, Reddit API integration, and GOG (Google CLI) tool
- **git_unit_tests.go** - Tests for Git operations (list branches, diff, status)

## Running Tests

Run all tests with:

```bash
go test ./tests/...
```

Run specific test files:

```bash
go test -v ./tests/agent_unit_tests.go ./tests/helpers.go
go test -v ./tests/email_unit_tests.go ./tests/helpers.go
go test -v ./tests/search_and_reddit_tests.go ./tests/helpers.go
go test -v ./tests/git_unit_tests.go ./tests/helpers.go
```

Run with coverage:

```bash
go test -cover ./tests/...
```

## Test Categories

### Agent Functions
- ToolExecutor initialization
- File operations (read, write, edit)
- Todo management (add, complete, delete, list)
- Email sending (send_email, send_report)

### Email Handling
- MIME word decoding (UTF-8 Base64, Quoted-Printable)
- Email parsing (full format, minimal format, multipart)
- Special character handling
- Unicode content support

### External Tools
- Web search with various query types
- Reddit API integration (search, subreddit listing)
- GOG (Google CLI) tool commands
- Error handling for missing parameters

### Git Operations
- List branches
- Show diff
- Show status
- Error handling for non-git directories

## Notes

- Some tests may fail or return errors due to external dependencies:
  - SMTP configuration for email sending
  - Network connectivity for web search and Reddit
  - GOG tool installation for Google CLI commands
  - Git installation for Git operations

- Tests use temporary directories (`t.TempDir()`) for file operations to ensure isolation
- Helper function `containsString()` is used throughout for result validation
- All tests follow Go testing best practices with `Test*` naming convention
