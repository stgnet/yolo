# Victron BLE Client Library

A Go library for connecting to and reading data from Victron Energy devices over Bluetooth Low Energy (BLE).

## Overview

This package provides functionality to:
- Scan for nearby Victron Energy devices advertising over BLE
- Connect to discovered devices
- Read VE.Direct protocol messages in real-time
- Parse and interpret telemetry data (voltage, current, power, etc.)

## Supported Devices

Tested with Victron Energy SmartSolar MPPT charge controllers. The library should work with any Victron device supporting:
- Bluetooth Low Energy (BLE) advertising
- VE.Direct protocol over BLE

## Installation

```bash
go get github.com/scottstg/yolo/victron
```

## Quick Start

### Scanning for Devices

```go
package main

import (
    "fmt"
    "time"
    "github.com/scottstg/yolo/victron"
)

func main() {
    client := victron.NewClient()
    
    // Scan for 10 seconds
    devices, err := client.Scan(10 * time.Second)
    if err != nil {
        fmt.Printf("Scan error: %v\n", err)
        return
    }
    
    for _, device := range devices {
        if device.IsVictron {
            fmt.Printf("Found: %s (%s, RSSI: %d)\n", 
                device.Name, device.Address, device.RSSI)
        }
    }
}
```

### Connecting and Reading Values

```go
package main

import (
    "context"
    "fmt"
    "github.com/scottstg/yolo/victron"
)

func main() {
    client := victron.NewClient()
    
    // Connect to a specific device
    device, err := client.Connect("AA:BB:CC:DD:EE:FF")
    if err != nil {
        fmt.Printf("Connect error: %v\n", err)
        return
    }
    defer client.Disconnect(device.Address)
    
    // Subscribe to real-time value updates
    ctx := context.Background()
    go func() {
        for updates := range device.Subscribe(ctx) {
            for key, value := range updates {
                fmt.Printf("%s: %v\n", key, value.FloatValue)
            }
        }
    }()
    
    // Keep connection alive
    <-make(chan struct{})
}
```

### Using Auto-Discovery

```go
// Automatically find and connect to the strongest Victron device
device, err := client.Discover()
if err != nil {
    fmt.Printf("Discover error: %v\n", err)
    return
}
defer client.Disconnect(device.Address)
```

## VE.Direct Protocol

Victron devices communicate using the VE.Direct protocol, an ASCII-based message format:

```
/[device][field(value)checksum
```

### Example Messages

```
/V(V=12.345)A      # Voltage = 12.345V
/A(A=-5.678)B      # Current = -5.678A (negative = discharging)
/P.W(W=301)C       # Power = 301W
```

### Parsing Messages

```go
// Parse a single VE.Direct message
frame, err := victron.ParseVEDirectMessage("/V(V=12.345)A")
if err != nil {
    // Handle error
}
fmt.Printf("Key: %s, Value: %.3f\n", frame.DataKey, frame.Value)

// Parse multiple messages from a stream
frames, err := victron.ParseVEDirectStream("/V(V=12.345)A\n/A(A=-5.678)B")

// Format values with human-readable names
formatted := victron.FormatValue(frame)
// Output: "System Voltage: 12.345 V"
```

## Known VE.Direct Data Keys

The library recognizes these common data keys:

### Voltages
- `V` - System voltage
- `Vin` - Battery voltage  
- `Pn.Vin` - PV input voltage (input N)

### Currents
- `A` - System current
- `Pn.Iin` - PV input current (input N)

### Power
- `P.W` - System power
- `Pn.P` - PV input power (input N)

### State & Status
- `SoC` - State of charge (%)
- `T.Temp` - Device temperature (°C)
- `A.State` - Charging algorithm state

### Identification
- `/Dev` - Device type
- `/Ver` - Firmware version
- `/SN` - Serial number

## API Reference

### Client Methods

| Method | Description |
|--------|-------------|
| `NewClient()` | Create a new BLE client |
| `Scan(duration)` | Scan for nearby devices |
| `Connect(address)` | Connect to a specific device |
| `Disconnect(address)` | Disconnect from a device |
| `Discover()` | Auto-discover and connect to strongest signal |
| `GetAllConnected()` | List all connected devices |

### Device Methods

| Method | Description |
|--------|-------------|
| `Subscribe(ctx)` | Receive real-time value updates |
| `GetValue(key)` | Get current value for a data key |
| `GetAllValues()` | Get all current values |
| `Disconnect()` | Disconnect this device |

### Parser Functions

| Function | Description |
|----------|-------------|
| `ParseVEDirectMessage(msg)` | Parse single VE.Direct message |
| `ParseVEDirectStream(data)` | Parse multiple messages |
| `ValidateVEDirectMessage(msg)` | Check message checksum |
| `FormatValue(frame)` | Format value with unit |
| `GetValueType(key)` | Get info about a data key |

## Error Handling

The library defines specific error types:

```go
type (
    ParseError    struct{} // Message parsing failed
    ConnectError  struct{} // BLE connection failed
    NotFoundError struct{} // Device not found during scan
)
```

## Troubleshooting

### No Devices Found

- Ensure Bluetooth is enabled on your system
- Verify Victron device has BLE enabled (some require configuration)
- Move closer to the device for better signal strength
- Try scanning for longer duration

### Connection Fails

- Check if device is already connected to another controller
- Some devices only allow one active connection at a time
- Ensure you're using the correct MAC address format

### Invalid Checksum Errors

- VE.Direct messages include checksum validation
- If many messages fail, there may be interference or connection issues
- The library automatically filters invalid messages

## Testing

The library includes comprehensive tests covering parser functions, BLE operations, and retry logic:

```bash
# Run all tests
go test ./victron/... -v

# Check coverage
go test ./victron/... -coverprofile=coverage.out
go tool cover -html=coverage.out  # Open in browser
go tool cover -func=coverage.out  # Print coverage by file

# Run specific test groups
go test ./victron -run "^TestParse"     # Parser tests
go test ./victron -run "^TestRetry"     # Retry module tests
go test ./victron/cmd/...               # Command integration tests
```

### Coverage Targets

- Core parser (`parser.go`): 95%+ coverage
- Helper functions: 100% coverage  
- Retry module (`retry.go`): 100% coverage
- Overall package: 85%+ coverage

## Retry Module

For unreliable BLE operations, use the built-in retry logic with exponential backoff:

```go
import "github.com/scottstg/yolo/victron"

// Use default retry configuration (3 retries, 100ms base delay)
config := victron.DefaultRetryConfig()

// Retry a potentially flaky operation
result, err := victron.Retry(ctx, config, func() (string, error) {
    return client.Connect(deviceAddress)
})

if err != nil {
    // Error includes number of attempts
    log.Printf("Failed after retries: %v", err)
}
```

See [RETRY.md](RETRY.md) for detailed documentation on configuration and usage.

## Examples

See `examples/` directory for complete working examples:
- `scan.go` - Device discovery
- `read_values.go` - Real-time monitoring
- `mcp_handler.go` - MCP server integration

## License

MIT License - see LICENSE file in repository root.
