# Errors Package

Custom error types and error handling utilities for YOLO agent.

## Usage

### Custom Error Types

```go
// File operations
err := errors.ErrFileNotFound{Path: "file.txt"}

// Tool execution
err := errors.ErrToolExecution{Tool: "gog", Code: 1, Output: "error output"}

// Configuration
err := errors.ErrInvalidConfig{Field: "api_key", Reason: "empty"}

// Network operations
err := errors.ErrNetworkTimeout{URL: "https://example.com", Duration: 30 * time.Second}

// JSON parsing
err := errors.ErrJSONParse{Input: `{"invalid": }`, Cause: json.SyntaxError{}}
```

### Error Wrapping with Context

```go
// Wrap error with additional context
wrapped := errors.WithContext(fmt.Errorf("original error"), "operation", "file_read")

// Check if wrapped error matches original type
if errors.IsFileNotFoundError(wrapped) {
    // Handle file not found
}

// Unwrap to get original error
original := errors.Unwrap(wrapped)
```

### Error Classification

```go
// Check error category
if errors.IsNetworkError(err) {
    // Retry with backoff
}

if errors.IsConfigurationError(err) {
    // Show user-friendly message
}

if errors.IsTimeoutError(err) {
    // Log timeout metrics
}
```

## Error Categories

- **File I/O**: `ErrFileNotFound`, `ErrPermissionDenied`, `ErrInvalidPath`
- **Tool Execution**: `ErrToolExecution`, `ErrMissingTool`, `ErrToolTimeout`
- **Configuration**: `ErrInvalidConfig`, `ErrMissingEnv`, `ErrValidationFailed`
- **Network**: `ErrNetworkTimeout`, `ErrHTTPError`, `ErrConnectionRefused`
- **JSON Operations**: `ErrJSONParse`, `ErrJSONMarshal`
- **Generic Wrapped**: `WrappedError` for adding context to any error

## Error Messages

All custom errors include:
- Descriptive message with context
- Error type identifier for programmatic checks
- Optional cause field for wrapped errors

Use standard Go error functions:
```go
errors.Is(err, target)      // Check error type
errors.As(err, &target)     // Extract typed error
errors.Unwrap(err)          // Get wrapped error
```
