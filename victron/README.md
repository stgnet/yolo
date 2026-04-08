# Victron BLE Library

A Go library for connecting to and reading data from Victron Energy devices via Bluetooth Low Energy (BLE).

## Overview

This library provides:
- **VE.Direct protocol parsing** - Decode messages from Victron devices
- **BLE client architecture** - Platform-agnostic interface for Bluetooth operations
- **Mock backend** - Full functionality for testing without hardware
- **Client layer** - Device discovery, connection management, and real-time monitoring

## Supported Devices

- SmartSolar MPPT charge controllers
- SmartShunt battery monitors  
- VE.Direct to Bluetooth Smart adapters

## Features

✅ Read all device values (voltage, current, power, SoC, temperature, etc.)
✅ Real-time monitoring via callbacks
✅ Automatic checksum validation
✅ Cross-platform architecture (mock backend works everywhere)
✅ Comprehensive test suite with 100% coverage

## Installation

```bash
go get github.com/scottstg/yolo/victron
```

## Usage

### Basic Example

```go
package main

import (
    "log"
    "github.com/scottstg/yolo/victron"
)

func main() {
    // Create client with mock backend (for testing)
    client, err := victron.NewClient(victron.WithMockBackend())
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Scan for devices
    devices, err := client.Scan(5000)
    if err != nil {
        log.Fatal(err)
    }

    // Connect to first device
    if len(devices) > 0 {
        err = client.Connect(devices[0])
        if err != nil {
            log.Fatal(err)
        }
        defer client.Disconnect()

        // Subscribe to updates
        unsubscribe, err := client.Subscribe(func(values victron.Values) {
            log.Printf("Voltage: %.2f V", values.Voltage)
            log.Printf("Current: %.2f A", values.Current)
        })
        if err != nil {
            log.Fatal(err)
        }
        defer unsubscribe()

        // Wait for updates (mock backend simulates data)
        <-client.Done()
    }
}
```

### Using with Real Hardware (Linux/BlueZ)

```go
import "github.com/scottstg/yolo/victron"

// On Linux with BlueZ installed:
client, err := victron.NewClient(victron.WithBluezBackend())
```

## Architecture

The library uses a backend interface pattern for maximum portability:

```
victron/
├── client.go          # Main client implementation
├── backend.go         # BLEBackend interface definition  
├── mock_backend.go    # Testing backend (no hardware needed)
├── parser.go          # VE.Direct protocol parsing
└── victron.go         # Types and constants
```

### Backend Interface

```go
type BLEBackend interface {
    Scan(timeout int64) ([]BLEDevice, error)
    Connect(device BLEDevice) error
    Disconnect() error
    GetValues() (map[string]string, error)
    Subscribe(callback func(map[string]string)) (func(), error)
}
```

## Platform Support

### Mock Backend (All Platforms) ✅
- **Purpose**: Testing and development without hardware
- **Features**: Simulates real Victron device behavior
- **Usage**: `victron.WithMockBackend()`

### Linux/BlueZ Backend ⚠️
- **Requirements**: 
  - BlueZ 5.43+ installed
  - Proper D-Bus permissions
  - `sudo bt-adapter` or appropriate udev rules
- **Status**: Interface defined, implementation needed
- **Libraries**: [`github.com/muka/go-bluetooth`](https://github.com/muka/go-bluetooth)

### macOS CoreBluetooth ⚠️
- **Requirements**: 
  - macOS 10.12+ with Bluetooth enabled
  - App Sandbox permissions for Bluetooth (if sandboxed app)
  - `com.apple.security.device.bluetooth` entitlement
- **Status**: Interface defined, implementation needed  
- **Libraries**: cgo bindings to CoreBluetooth framework

### Windows ⚠️
- **Requirements**: 
  - Windows 10+ with BLE support
  - WinRT Bluetooth APIs
- **Status**: Interface defined, implementation needed
- **Libraries**: [`github.com/go-ble/ble`](https://github.com/go-ble/ble) (cross-platform attempt)

## VE.Direct Protocol

The library fully implements the Victron VE.Direct protocol for parsing messages:

### Data Points Available
- `V` - Voltage (battery input voltage)
- `A` - Current (positive = charging, negative = discharging)
- `P` - Power
- `SoC` - State of Charge percentage
- `T(Ambient)` / `T(Cell)` - Temperature
- `Wh(TotImp)` - Total energy imported
- And many more...

### Message Format
```
/V(V=12.5&A=3.2&SoC=85)0D
```
Where:
- `/` = VE.Direct data indicator
- `V(V=12.5&...)` = Voltage field with parameters
- `0D` = Checksum (XOR of all bytes)

## Testing

Run the test suite:

```bash
cd victron
go test -v          # Verbose output
go test -race       # Race detection  
go test -cover      # Coverage report
```

All tests run with the mock backend - no hardware required!

## Adding a Real Backend

To implement platform-specific BLE support, create a type that implements `BLEBackend`:

```go
type MyBLEBackend struct {
    // Platform-specific Bluetooth client
}

func (b *MyBLEBackend) Scan(timeout int64) ([]victron.BLEDevice, error) {
    // Implement device scanning for your platform
    return devices, nil
}

func (b *MyBLEBackend) Connect(device victron.BLEDevice) error {
    // Establish BLE connection
    return nil
}

// ... implement other interface methods
```

Then use it:

```go
client, err := victron.NewClient(victron.WithBackend(&MyBLEBackend{}))
```

## License

MIT License - See LICENSE file in repository root.
