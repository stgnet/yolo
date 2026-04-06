# Testing Guide

This document describes the testing practices and guidelines for the YOLO Agent project.

## Overview

The test suite is designed to be **production-safe** with zero side effects. All tests follow proper isolation patterns and can be safely run in CI/CD pipelines.

## Running Tests

### Run All Tests
```bash
go test ./...
```

### Run With Coverage
```bash
go test -cover ./...
# or
go test -coverprofile=cover.out ./... && go tool cover -func=cover.out
```

### Run Specific Package
```bash
go test -v ./tools_git_test.go
go test -v ./email/...
```

## Test Categories & Safety

All tests are verified to be **production-safe** with no unwanted side effects:

| Category | Test Files | Safety Level | Details |
|--|--|-|--|--|--|--|
| Git Tools | `tools_git_test.go` | ✅ Excellent | Uses in-memory git repos (`git.NewRepository()`) |
| Email | `email/email_test.go` | ✅ Excellent | Message construction only, never sends |
| Inbox | `tools_inbox_test.go` | ✅ Good | Temp directories with cleanup |
| Memory | `tools_memory*_test.go` | ✅ Good | Test scope operations with defer cleanup |
| History DB | `historydb_test.go` | ✅ Excellent | In-memory SQLite (`:memory:`) |
| Buffer UI | `bufferui_test.go` | ✅ Excellent | Pure in-memory TUI logic |

## Isolation Patterns

### 1. Temporary Directories
```go
func TestExample(t *testing.T) {
    tempDir := t.TempDir()
    // Use tempDir for file operations
    // Automatically cleaned up after test
}
```

### 2. In-Memory Git Repositories
```go
repo := git.NewRepository()
// Creates memory-only git repository
// Never touches actual user repository
```

### 3. In-Memory Databases
```go
db, err := sql.Open("sqlite", ":memory:")
// SQLite in-memory database
// No persistence to disk
```

### 4. Mock Dependencies
```go
// Email tests create structs but never call Send()
email := &Email{To: "test@example.com", ...}
validateEmail(email) // Validation only, no network activity
```

## Coverage Targets

- **Main package**: ~35% (CLI tools)
- **Email package**: ~48% (message construction/validation)
- **Overall goal**: Maintain existing coverage levels while improving quality

## Best Practices

### ✅ DO
- Use `t.TempDir()` for file operations
- Clean up resources with `defer` statements
- Mock external dependencies
- Keep tests deterministic and repeatable
- Test edge cases and error conditions

### ❌ DON'T
- Send real emails in tests
- Modify production data or repositories
- Leave files/directories behind after tests
- Rely on network connectivity in unit tests
- Test implementation details (test behavior instead)

## Audit Reports

Comprehensive safety audits have been completed:

- [`TEST_SAFETY_AUDIT.md`](TEST_SAFETY_AUDIT.md) - Detailed safety analysis
- [`TEST_AUDIT_REPORT.md`](TEST_AUDIT_REPORT.md) - Findings and recommendations

All tests verified production-safe with zero side effects.

## CI/CD Integration

The test suite is safe to run in:
- ✅ Continuous Integration pipelines
- ✅ Pre-commit hooks
- ✅ Automated testing environments
- ✅ Multiple concurrent runs (idempotent)

## Test Structure

Each test package follows this pattern:
```
package_name_test.go
├── TestFunctionName_Success()
├── TestFunctionName_Failure()
├── TestFunctionName_EdgeCase()
└── TestFunctionName_Validation()
```

## Adding New Tests

When adding new tests, ensure they:
1. Use proper isolation patterns (temp dirs, in-memory resources)
2. Clean up all created resources
3. Don't produce side effects outside test scope
4. Are deterministic (same input = same output)
5. Include both success and failure cases

For examples of proper test implementation, see:
- `tools_git_test.go` - In-memory git operations
- `email/email_test.go` - Struct validation without network calls
- `historydb_test.go` - In-memory database usage
