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
✅ Real-time monitoring via channels/callbacks
✅ Automatic VE.Direct protocol parsing with checksum validation
✅ Cross-platform architecture with backend interface
✅ Comprehensive test suite with mock backend for testing without hardware
✅ Example CLI tool (`victron-read`) demonstrating complete workflow

## Installation

```bash
go get github.com/scottstg/yolo/victron
```

## Examples

### Command-Line Tool

A complete example CLI tool is available at [`victron/cmd/victron-read/`](./cmd/victron-read/):

```bash
# Scan for and connect to Victron devices
go run ./victron/cmd/victron-read

# Connect to specific device by MAC address
go run ./victron/cmd/victron-read AA:BB:CC:DD:EE:FF
```

See [`victron/cmd/victron-read/README.md`](./cmd/victron-read/README.md) for full documentation.

### Basic Usage in Your Code

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/scottstg/yolo/victron"
)

func main() {
    // Initialize BLE backend (required first step)
    if err := victron.InitializeBackend(); err != nil {
        log.Fatalf("Failed to initialize BLE: %v", err)
    }

    // Create client
    client := victron.NewClient()

    // Scan for nearby devices (10 second scan)
    results, err := client.Scan(10 * time.Second)
    if err != nil {
        log.Fatal(err)
    }

    // Filter Victron devices and connect to strongest
    for _, device := range results {
        if device.IsVictron {
            fmt.Printf("Found: %s at %s (RSSI: %d)\n", device.Name, device.Address, device.RSSI)
            
            // Connect to device
            victronDevice, err := client.Connect(device.Address)
            if err != nil {
                log.Fatal(err)
            }
            defer victronDevice.Disconnect()

            // Establish BLE connection
            if err := victronDevice.Connect(); err != nil {
                log.Fatal(err)
            }

            // Get all current values
            values := victronDevice.GetAllValues()
            for key, val := range values {
                fmt.Printf("%s: %s\n", key, val.RawValue)
            }

            break // Connect to first device only
        }
    }
}
```

### Direct Connection (Known Device)

If you already know the MAC address of your Victron device:

```go
// Initialize backend
if err := victron.InitializeBackend(); err != nil {
    log.Fatal(err)
}

client := victron.NewClient()
device, _ := client.Connect("AA:BB:CC:DD:EE:FF")
defer device.Disconnect()

if err := device.Connect(); err != nil {
    log.Fatal(err)
}

// Subscribe to real-time updates
go func() {
    for values := range device.SubscribeBatched() {
        voltage, exists := values[victron.KeySystemVoltage]
        if exists {
            fmt.Printf("Voltage: %.2f V\n", voltage.FloatValue)
        }
    }
}()

// Keep running to receive updates
time.Sleep(30 * time.Second)
```

### Using with Mock Backend (Testing)

For development and testing without hardware:

```go
import "github.com/scottstg/yolo/victron"

// Set mock backend before initializing
victron.SetBackend(&victron.MockBackend{})
if err := victron.InitializeBackend(); err != nil {
    log.Fatal(err)
}

// Rest of code works the same - mock simulates device behavior
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
