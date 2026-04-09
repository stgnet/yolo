# Contributing to YOLO

Thank you for your interest in contributing to YOLO (Your Own Living Operator)! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Style Guidelines](#code-style-guidelines)
- [Testing Requirements](#testing-requirements)
- [Documentation Standards](#documentation-standards)
- [Commit Message Format](#commit-message-format)
- [Pull Request Process](#pull-request-process)

## Getting Started

### Prerequisites

- Go 1.26 or higher
- Git
- Ollama installed and running locally
- Optional: Bluetooth hardware for Victron testing (tests gracefully handle missing hardware)

### Setting Up Development Environment

```bash
# Clone the repository
git clone https://github.com/your-username/yolo.git
cd yolo

# Download dependencies
go mod download

# Build the project
go build -o yolo

# Run tests
go test ./...
```

### Verify Installation

```bash
./yolo --version
go test -v ./...
```

## Development Workflow

### 1. Fork and Branch

```bash
# Fork the repository on GitHub
# Clone your fork
git clone https://github.com/YOUR_USERNAME/yolo.git
cd yolo

# Create a feature branch
git checkout -b feature/your-feature-name
```

### 2. Make Changes

Follow the coding guidelines below when making changes:
- Write tests for new functionality
- Update documentation as needed
- Keep changes focused and atomic
- Run `go fmt` before committing

### 3. Test Your Changes

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### 4. Commit and Push

```bash
# Stage changes
git add .

# Commit with descriptive message
git commit -m "feat: add your feature description"

# Push to your fork
git push origin feature/your-feature-name
```

### 5. Submit Pull Request

- Go to the original repository on GitHub
- Click "Pull Requests" and then "New Pull Request"
- Select your branch and submit
- Wait for review and feedback

## Code Style Guidelines

### General Principles

- Follow standard Go idioms and conventions
- Use `gofmt` for formatting (no manual formatting)
- Keep functions focused and reasonably sized (< 100 lines when possible)
- Name variables and functions descriptively
- Prefer clarity over cleverness

### File Organization

```
yolo/
├── agent.go           # Core agent logic
├── tools_*.go         # Tool implementations
├── *_test.go          # Test files (same prefix as implementation)
├── email/             # Email-related code
├── victron/           # Victron BLE integration
└── DOCS/              # Documentation files
```

### Error Handling

- Return errors, don't panic (except in main for fatal errors)
- Use descriptive error messages that help debugging
- Chain errors with `fmt.Errorf("context: %w", err)` when adding context
- Document expected errors in function comments

Example:
```go
func ReadFile(path string) ([]byte, error) {
    if path == "" {
        return nil, fmt.Errorf("path cannot be empty")
    }
    
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read file %s: %w", path, err)
    }
    
    return data, nil
}
```

## Testing Requirements

### Test Safety (Critical!)

**All tests must be safe and non-destructive.** This is a core requirement for CI/CD compatibility.

#### DO:
- ✅ Use isolated temp directories for file operations
- ✅ Check file existence without creating/modifying files in working directory
- ✅ Handle missing hardware gracefully (e.g., no Bluetooth = test passes with message)
- ✅ Mock external dependencies where possible
- ✅ Clean up after tests using `defer os.RemoveAll(tempDir)`

#### DON'T:
- ❌ Create files or directories in the project's working directory
- ❌ Perform actual git operations that modify `.git` directory
- ❌ Make real network requests without mocking options
- ❌ Leave artifacts after test execution
- ❌ Assume hardware is present (Bluetooth, email server, etc.)

### Test Structure

```go
func TestExample(t *testing.T) {
    // Arrange: Set up test fixtures
    tempDir := t.TempDir()
    testFile := filepath.Join(tempDir, "test.txt")
    
    // Act: Execute the code being tested
    result, err := FunctionUnderTest(testFile)
    
    // Assert: Verify the outcome
    if err != nil {
        t.Errorf("Expected no error, got %v", err)
    }
    
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

### Coverage Requirements

- New features: Minimum 80% test coverage
- Critical paths (security, email handling): Aim for 90%+ coverage
- Use `go test -cover` to check coverage before submitting

## Documentation Standards

### Code Comments

- Every exported function/type must have a comment
- Explain WHY, not just WHAT (code shows what)
- Include examples for complex functions

```go
// ToolExecutor handles dispatch of tool calls to concrete implementations.
// It supports 40+ tools across categories: file operations, agent management,
// external services, version control, and hardware integration.
//
// Example:
//   executor := NewToolExecutor(config)
//   result, err := executor.Execute("read_file", args)
type ToolExecutor struct {
    // fields...
}

// Execute calls the specified tool with given arguments.
func (e *ToolExecutor) Execute(name string, args map[string]string) (string, error) {
    // implementation...
}
```

### Markdown Documentation

- Update README.md for user-facing changes
- Add entries to CHANGELOG.md following the format
- Document new tools in DOCS/tools.md
- Keep documentation close to code when possible

## Commit Message Format

Follow conventional commits format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Formatting, missing semicolons, etc.
- `refactor`: Code restructuring without behavior change
- `test`: Adding or fixing tests
- `ci`: CI/CD configuration changes
- `chore`: Maintenance tasks

### Examples:

```
feat(victron): add support for SmartShunt battery monitors

Implement SmartShunt device parsing and value extraction.
Supports voltage, current, and charge/discharge tracking.

Closes #123
```

```
test(email): make tests resilient to missing email server

Tests now skip gracefully when postfix is not configured.
No side effects or file creation in working directory.
```

```
ci: add GitHub Actions workflow for automated testing

- Run tests on push and PR
- Generate coverage reports with Codecov
- Build for multiple platforms
```

## Pull Request Process

### Before Submitting

1. ✅ All tests pass locally (`go test ./...`)
2. ✅ Code is formatted (`go fmt ./...`)
3. ✅ No linting errors (`golangci-lint run`)
4. ✅ Tests follow safety guidelines (no side effects)
5. ✅ Documentation updated if needed
6. ✅ CHANGELOG.md updated for user-visible changes
7. ✅ Commit messages follow the format above

### During Review

- Be responsive to reviewer feedback
- Make requested changes promptly
- Keep commits logical and atomic
- Squash unnecessary commits before merging
- Add tests if suggested by reviewers

### After Approval

- Maintainer will merge your PR (typically via squash merge)
- You'll be acknowledged in the CHANGELOG for significant contributions
- Your work helps improve YOLO for everyone!

## Questions?

If you have questions about:
- Implementation details: Check existing code and tests
- Architecture: See README.md architecture section
- Test safety: Review `TESTING.md` and audit reports
- Something else: Open an issue or ask in your PR description

## Thank You!

Contributions make the YOLO community amazing. We appreciate your time and effort to make this project better! 🙏

---

*Note: The test safety requirements are particularly important as they enable CI/CD automation. Tests that create side effects cannot run reliably in automated environments.*
