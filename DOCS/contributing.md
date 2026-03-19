# Contributing to YOLO

Thank you for your interest in contributing to YOLO (Your Own Living Operator)! This document provides guidelines for contributing to the project.

## Overview

YOLO is a self-evolving AI agent designed for autonomous software development. It continuously improves itself by reading and modifying its own source code, adding tests, fixing bugs, and implementing new features.

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- Access to Ollama with compatible models (qwen3.5:27b recommended)
- Optional: Google Workspace credentials for GOG integration

### Getting Started

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourorg/yolo.git
   cd yolo
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Run tests**
   ```bash
   go test -v ./...
   ```

4. **Build the project**
   ```bash
   go build -o yolo .
   ```

## Code Style

- Follow Go best practices and idioms
- Use `gofmt` for formatting:
  ```bash
  gofmt -w .
  ```
- Add tests for new features (aim for >90% coverage)
- Write clear commit messages

## Testing

### Run All Tests
```bash
go test -v ./...
```

### Run with Coverage
```bash
go test -v -coverprofile=cov.out ./...
go tool cover -html=cov.out
```

### Test Coverage Requirements
- New code should have >90% test coverage
- Existing packages average ~85%+ coverage
- Critical paths must be fully tested

## Adding New Tools

Tools are the core capabilities of YOLO. To add a new tool:

1. **Create the tool function** in `tools.go` or a dedicated file (`tools_xxx.go`)
2. **Add to ToolDefinitions** slice in `agent.go`
3. **Write unit tests** for all code paths
4. **Update documentation** in YOLO.md

Example tool structure:
```go
func newToolName(args string) string {
    // Parse arguments
    // Execute logic
    // Return formatted result
}
```

## Adding New Features

1. **Start a sub-agent** for the task (if you're YOLO itself):
   ```
   spawn_subagent prompt="Describe the feature to implement"
   ```

2. **Follow the workflow**:
   - Implement the feature
   - Add tests
   - Run full test suite
   - Commit changes
   - Restart if needed

3. **Test thoroughly**:
   - Unit tests for new code
   - Integration tests if affecting multiple components
   - Manual testing for UI/UX features

## Documentation

- Keep `YOLO.md` up to date with all capabilities
- Add README.md for new packages
- Document complex logic with comments
- Update examples when adding features

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add/update tests
5. Ensure all tests pass
6. Update documentation if needed
7. Commit your changes (`git commit -m 'Add amazing feature'`)
8. Push to the branch (`git push origin feature/amazing-feature`)
9. Open a Pull Request

## Code Review

All submissions require review. Maintainers will review:
- Code quality and style
- Test coverage
- Documentation updates
- Performance impact
- Security considerations

## Community Guidelines

- Be respectful and inclusive
- Provide clear, detailed descriptions
- Help others with their issues when possible
- Report security vulnerabilities responsibly

## Questions?

If you have questions or need help, open an issue or reach out to the maintainers.

---

Thank you for contributing to YOLO! 🚀
