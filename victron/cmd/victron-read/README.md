# Victron Reader - Example Application

A command-line tool demonstrating how to scan for and connect to Victron Energy devices via Bluetooth Low Energy (BLE).

## Features

- **Scan Mode**: Automatically discover nearby Victron VE.Direct devices
- **Direct Connect**: Connect to a specific device by MAC address
- **Real-time Monitoring**: Display live sensor values with automatic updates
- **Smart Detection**: Identifies SmartSolar MPPT and SmartShunt battery monitors

## Prerequisites

- System with Bluetooth Low Energy capability (Linux, macOS)
- Victron VE.Direct device (SmartSolar MPPT, SmartShunt, or VE.Direct adapter)
- Go 1.25+ installed

## Installation

```bash
# Build from source
go build -o victron-read ./victron/cmd/victron-read

# Or install globally
go install github.com/scottstg/yolo/victron/cmd/victron-read@latest
```

## Usage

### Scan for Nearby Devices (Recommended)

Discover all Victron devices in range:

```bash
./victron-read
```

This will:
1. Scan for nearby BLE devices advertising as Victron products
2. List discovered devices with signal strength
3. Automatically connect to the strongest device (or let you choose)
4. Display real-time sensor values for 10 seconds

### Connect to Specific Device

If you know the MAC address of your Victron device:

```bash
./victron-read AA:BB:CC:DD:EE:FF
```

Replace `AA:BB:CC:DD:EE:FF` with the actual MAC address.

## Example Output

```
==========================================
  Victron Energy BLE Device Reader
==========================================

Connecting to device: AA:BB:CC:DD:EE:FF

✓ Connected successfully!

Current Sensor Values:
----------------------------------
  V                              : 13.76 V
  A                              : -2.45 A
  P                              : -33.8 W
  P.V                            : 45.2 V
  P.A                            : 0.62 A
  P.W                            : 28.0 W

Monitoring live updates (10 seconds)...
----------------------------------
  V: 13.77 V | A: -2.46 A | P: -33.9 W

Monitoring complete! Total updates received: 10

Session complete. Disconnecting...
```

## Supported Devices

### SmartSolar MPPT Charge Controllers
- System voltage, current, and power
- PV panel voltage, current, and power (single or dual input)
- Temperature monitoring
- Error codes and charging algorithm state

### SmartShunt Battery Monitors
- Battery voltage, current, and state of charge
- Energy yield tracking (Ah)
- Coulomb counting data

## Technical Details

This example demonstrates:
- BLE backend initialization
- Device discovery via advertising packets
- Connection establishment with Victron VE.Direct protocol
- Real-time value monitoring through channels
- Graceful disconnection

See the source code in `victron/cmd/victron-read/main.go` for a complete working example of the victron package API.

## Troubleshooting

### "Failed to initialize BLE backend"
Ensure your system has Bluetooth support enabled:
- **Linux**: Install BlueZ (`sudo apt install bluez`)
- **macOS**: Enable Bluetooth in System Preferences

### "No Victron devices found"
1. Ensure your Victron device is powered on and within 10 meters
2. Check that the device supports VE.Direct over BLE (requires SmartSolar MPPT or SmartShunt)
3. Try scanning again - advertising may be intermittent

### "Connection timeout"
1. Move closer to the device for better signal strength
2. Ensure no other applications are connected to the same device
3. Restart your Victron device and try again

## API Reference

The example uses these key victron package functions:

| Function | Description |
|----------|-------------|
| `InitializeBackend()` | Initialize BLE support (required first step) |
| `NewClient()` | Create a new client instance |
| `client.Scan(duration)` | Discover nearby devices |
| `client.Connect(address)` | Connect to specific device by MAC |
| `device.Connect()` | Establish BLE connection |
| `device.GetAllValues()` | Get all current sensor readings |
| `device.SubscribeBatched()` | Receive real-time updates via channel |
| `device.Disconnect()` | Clean up and disconnect |

## License

MIT License - See LICENSE file in the project root.
