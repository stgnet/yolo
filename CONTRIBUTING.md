# Contributing to YOLO

Thank you for your interest in contributing to YOLO (Your Own Living Operator)! This document provides guidelines for contributors. For general usage, see [README.md](README.md).

## Quick Reference

- **Main Documentation**: [README.md](README.md) - Tools, features, usage
- **Architecture Deep Dive**: [ARCHITECTURE.md](ARCHITECTURE.md) - Technical details
- **Email System**: [EMAIL_PROCESSING.md](EMAIL_PROCESSING.md) - Email integration

## Prerequisites

- Go 1.21 or later
- Git
- Access to Ollama with compatible models (qwen3.5:27b recommended)
- Optional: Google Workspace credentials for GOG integration

## Getting Started

```bash
# Clone and setup
git clone https://github.com/yourorg/yolo.git
cd yolo
go mod download

# Run tests
go test -v ./...

# Build
go build -o yolo .
```

## Code Style

- Follow Go best practices and idioms
- Use `gofmt` for formatting:
  ```bash
  gofmt -w .
  go vet ./...
  ```
- Add tests for new features (aim for >90% coverage)
- Write clear commit messages with actionable descriptions

## Testing Guidelines

### Run Tests

```bash
# All tests
go test -v ./...

# With coverage report
go test -coverprofile=cov.out ./...
go tool cover -html=cov.out  # Opens in browser

# Race detection (required for PRs)
go test -race ./...
```

### Coverage Requirements

- **New code**: Must have >90% test coverage
- **Existing packages**: Average ~85%+ coverage maintained
- **Critical paths** (file I/O, tool execution): Fully tested
- **Edge cases**: Invalid input, errors, boundary conditions

## Adding New Tools

Tools are YOLO's core capabilities. To add a new tool:

1. **Create tool function** in `tools.go` or dedicated file (`tools_xxx.go`)
2. **Add to ToolDefinitions slice** in `agent.go`
3. **Write unit tests** covering all code paths and error cases
4. **Update documentation** in README.md Tools Reference section

Example:
```go
func myNewTool(args string) string {
    // Parse arguments with validation
    var input struct {
        Query string `json:"query"`
        Count int    `json:"count"`
    }
    if err := json.Unmarshal([]byte(args), &input); err != nil {
        return fmt.Sprintf("Error parsing args: %v", err)
    }
    
    // Validate inputs
    if input.Count < 1 || input.Count > 10 {
        return "Count must be between 1 and 10"
    }
    
    // Execute logic with error handling
    result, err := doWork(input.Query, input.Count)
    if err != nil {
        return fmt.Sprintf("Operation failed: %v", err)
    }
    
    return fmt.Sprintf("Success: %s", result)
}
```

## Development Workflow

1. **Start a sub-agent** for tasks (if you're YOLO itself):
   ```
   spawn_subagent prompt="Describe the feature to implement"
   ```

2. **Implement changes**:
   - Write code following Go style guides
   - Add comprehensive tests
   - Run full test suite with race detection
   
3. **Verify**:
   - `go build` passes without errors
   - `go test ./...` all pass
   - `go test -race ./...` no races detected
   - `gofmt -l .` shows clean output

4. **Commit and restart**:
   - Commit changes with descriptive message
   - Use `restart` tool to rebuild and exec new binary

## Documentation Standards

- Update README.md for user-facing features
- Add ARCHITECTURE.md sections for system design changes
- Document complex logic with inline comments
- Keep examples current and tested
- Remove outdated information (check TODO list for doc cleanup tasks)

## Pull Request Process

1. Fork the repository
2. Create feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes following guidelines above
4. Ensure all tests pass including race detection
5. Update relevant documentation
6. Commit with clear messages: `git commit -m 'Add amazing feature'`
7. Push: `git push origin feature/amazing-feature`
8. Open Pull Request with description of changes

## Code Review Checklist

All submissions require review. Reviewers check:

- ✅ Code quality and Go idioms
- ✅ Test coverage (>90% for new code)
- ✅ Race-free concurrency (pass `-race` flag)
- ✅ Documentation updates
- ✅ Performance impact acceptable
- ✅ Security considerations addressed
- ✅ No breaking changes to existing features

## Community Guidelines

- Be respectful and inclusive
- Provide clear, detailed descriptions in PRs/issues
- Help others when possible
- Report security vulnerabilities responsibly (not publicly)

## Questions?

Open an issue or reach out to maintainers.

---

**Remember**: YOLO continuously improves its own codebase. Many improvements come from autonomous operation following the same guidelines!
