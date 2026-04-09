# Contributing to YOLO

Thank you for your interest in contributing to the YOLO project! This document provides guidelines and information for contributors.

## Quick Start

### Setting Up Your Environment

1. **Fork the repository** on GitHub
2. **Clone your fork**:
   ```bash
   git clone https://github.com/YOUR_USERNAME/yolo.git
   cd yolo
   ```

3. **Install dependencies**:
   ```bash
   go mod download
   ```

4. **Verify setup** - Run all tests:
   ```bash
   go test -race ./...
   ```

## Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-description
```

### 2. Make Your Changes

Follow these guidelines:
- Write clean, idiomatic Go code
- Add comments for complex logic
- Keep functions focused and small
- Follow existing code style

### 3. Write Tests

**All new code must include tests:**

```bash
# Run tests with race detection
go test -race ./...

# Check coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

**Test Requirements:**
- ✅ Test public functions
- ✅ Include edge cases
- ✅ Use table-driven tests where appropriate
- ✅ Run with `-race` flag to detect data races

### 4. Format and Lint Your Code

```bash
# Auto-format code
go fmt ./...

# Run linter (install golangci-lint first)
golangci-lint run
```

### 5. Commit Your Changes

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```bash
git add .
git commit -m "feat: add new feature"
# or
git commit -m "fix: resolve bug in component"
# or
git commit -m "test: add test coverage for X"
```

**Commit Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test additions/modifications
- `refactor`: Code refactoring
- `chore`: Maintenance tasks

### 6. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a PR on GitHub with:
- Clear title describing the change
- Description of what and why
- Reference to related issues (if any)
- Test results showing all tests pass

## Code Style Guide

### General Principles

1. **Idiomatic Go**: Follow [Effective Go](https://golang.org/doc/effective_go.html)
2. **Readability**: Prioritize clear, understandable code over cleverness
3. **Consistency**: Match existing code style in the file/package

### Specific Guidelines

**File Organization:**
- One package per directory (with exceptions for internal packages)
- Files organized by functionality
- Test files: `*_test.go` with same base name as source

**Naming Conventions:**
- Exported identifiers: PascalCase (`MyFunction`)
- Unexported identifiers: camelCase (`myFunction`)
- Constants: MixedCaps (`MaxConnections`)
- Structs: PascalCase (`UserProfile`)
- Test functions: `TestFeatureName_Subcase`

**Error Handling:**
- Return errors, don't panic (except in tests/init)
- Wrap errors with context using `fmt.Errorf("%w", err)`
- Document error conditions in function comments

**Documentation:**
```go
// MyFunction does something useful. It takes input and returns output.
//
// Example:
//
//      result, err := MyFunction("input")
func MyFunction(input string) (string, error) {
    // implementation
}
```

## Testing Guidelines

### Writing Good Tests

1. **Test Name Structure**: `Test{Feature}_{Behavior}_{Condition}`
   ```go
   func TestVictronParse_DecodeSuccess_StandardDevice(t *testing.T) {}
   ```

2. **Table-Driven Tests** for multiple cases:
   ```go
   func TestFunctionName(t *testing.T) {
       tests := []struct {
           name     string
           input    InputType
           want     OutputType
           wantErr  bool
       }{
           {"valid case", validInput, expectedOutput, false},
           {"invalid case", invalidInput, zeroOutput, true},
       }
       
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               got, err := FunctionName(tt.input)
               // assertions...
           })
       }
   }
   ```

3. **Setup and Teardown**:
   - Use `t.Cleanup()` for resource cleanup
   - Use subtests for independent test cases
   - Mock external dependencies

4. **Coverage Requirements**:
   - Minimum 80% coverage for new code
   - All public functions must have tests
   - Edge cases and error paths tested

### Running Tests

```bash
# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Run specific package
go test ./victron

# Verbosity + timing
go test -v -count=1 ./...

# Coverage report
go test -coverprofile=c.out ./... && go tool cover -html=c.out
```

## Pull Request Process

### Before Submitting

- [ ] Code follows style guide
- [ ] All tests pass (`go test -race ./...`)
- [ ] New code has appropriate test coverage
- [ ] Documentation updated (if applicable)
- [ ] Code reviewed locally or with team member
- [ ] No linting errors
- [ ] Commit messages follow conventions

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing Done
- [ ] Added/updated tests
- [ ] All existing tests pass
- [ ] Race detection clean (`go test -race ./...`)
- [ ] Coverage verified

## Related Issues
Closes #XXX

## Additional Notes
Any other context or information
```

## Code Review Guidelines

**Reviewers will check:**
1. ✅ Logic correctness
2. ✅ Error handling completeness
3. ✅ Test coverage adequacy
4. ✅ Code style compliance
5. ✅ Performance considerations
6. ✅ Security implications

**Be prepared to:**
- Address feedback promptly
- Explain design decisions if questioned
- Make requested changes or discuss alternatives
- Update documentation as needed

## Getting Help

- **Questions**: Open a GitHub issue with "question" label
- **Discussions**: Participate in existing issues/PRs
- **Complex Features**: Discuss design before implementation (create issue)

## Resources

- [Go Documentation](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Testing in Go](https://go.dev/doc/tutorial/tests)

## License

By contributing, you agree that your contributions will be licensed under the MIT License. See LICENSE file for details.

---

**Thank you for contributing to YOLO!** 🎉
