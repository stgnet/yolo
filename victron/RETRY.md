# Retry Module

The `Retry` module provides exponential backoff retry logic for unreliable operations such as BLE connections, scans, and data reads. This helps handle transient failures gracefully without manual retry implementation in every operation.

## Features

- **Exponential Backoff**: Delays between retries increase exponentially to avoid overwhelming devices
- **Configurable Parameters**: Customize max retries, delays, and behavior
- **Context Support**: Respects cancellation for clean timeout handling
- **Jitter**: Optional random delay variation to prevent thundering herd problems
- **Generic Implementation**: Works with any return type using Go generics

## Usage

### Basic Example

```go
import "github.com/scottstg/yolo/victron"

ctx := context.Background()
config := victron.DefaultRetryConfig()

// Retry a BLE operation with automatic backoff
result, err := victron.Retry(ctx, config, func() (string, error) {
    return device.ReadCharacteristic(charUUID)
})

if err != nil {
    log.Printf("Operation failed after retries: %v", err)
} else {
    log.Printf("Success: %v", result)
}
```

### Custom Configuration

```go
config := victron.RetryConfig{
    MaxRetries:  5,              // Number of retry attempts
    BaseDelay:   200 * time.Millisecond,  // Initial delay
    MaxDelay:    10 * time.Second,        // Maximum delay cap
    Multiplier:  1.5,                 // Exponential growth factor
    Jitter:      true,                  // Add random variation to delays
}

result, err := victron.Retry(ctx, config, func() (int, error) {
    return sensor.GetValue("V")
})
```

### With Context Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Operation will be cancelled if it takes longer than 30 seconds total
result, err := victron.Retry(ctx, config, func() (Data, error) {
    return device.Connect(address)
})
```

## Configuration Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `MaxRetries` | int | 3 | Number of retry attempts after initial failure |
| `BaseDelay` | time.Duration | 100ms | Initial delay before first retry |
| `MaxDelay` | time.Duration | 5s | Maximum delay between retries |
| `Multiplier` | float64 | 2.0 | Factor by which delay increases each retry |
| `Jitter` | bool | true | Add ±25% random variation to delays |

## Delay Calculation

The delay before each retry is calculated as:

```
delay = min(BaseDelay × Multiplier^(attempt-1), MaxDelay)
```

With jitter enabled (default), a random variation of ±25% is added to prevent synchronized retries across multiple devices.

### Example Delay Sequence

With default config (`BaseDelay=100ms`, `Multiplier=2.0`):

- Attempt 1: immediate (no delay)
- Attempt 2: ~100ms delay
- Attempt 3: ~200ms delay
- Attempt 4: ~400ms delay

## Error Handling

The `Retry` function returns:
- The successful result on first success
- An error wrapping the last failure with attempt count information after all retries exhausted

```go
if err != nil {
    // Error message includes number of attempts
    // Example: "operation failed after 4 attempts: connection refused"
}
```

## Use Cases

The retry module is particularly useful for:

1. **BLE Operations**: Scanning, connecting, reading characteristics
2. **Network Requests**: HTTP calls with potential transient failures
3. **File Operations**: Accessing files that may be temporarily locked
4. **Database Queries**: Handling connection pool exhaustion
5. **Sensor Reads**: Dealing with intermittent sensor communication issues

## Best Practices

1. **Use Context**: Always pass a context with appropriate timeouts
2. **Reasonable Defaults**: `DefaultRetryConfig()` works for most BLE operations
3. **Idempotent Operations**: Retry is safest with operations that can be repeated safely
4. **Monitor Retries**: Log retry attempts to understand failure patterns
5. **Adjust for Use Case**: 
   - Fast operations: reduce BaseDelay
   - Critical operations: increase MaxRetries
   - Unstable connections: enable jitter

## Testing

The module includes comprehensive tests covering:

- Success on first attempt
- Success after retries
- Maximum attempts exceeded
- Context cancellation
- Exponential backoff calculation
- Delay caps and jitter

Run tests with:
```bash
go test -v ./victron -run Retry
```

## Integration Examples

See the `victron-read` command for practical examples of using retry logic with BLE operations.
