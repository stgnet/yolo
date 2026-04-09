# Victron BLE Quick Start Guide

This guide helps you get started with the Victron Bluetooth Low Energy (BLE) tool integrated into YOLO.

## Prerequisites

- A computer with Bluetooth capability (Bluetooth 4.0+ BLE support required)
- Victron devices nearby (SmartSolar MPPT, SmartShunt, etc.)
- The device must be advertising via BLE (usually enabled by default on newer Victron devices)

## Basic Usage

### 1. Scan for Nearby Devices

Find all Victron devices in range:

```
victron scan --duration 15
```

Parameters:
- `--duration`: Scan duration in seconds (default: 10, max: 60)

Example output:
```json
{
  "status": "success",
  "devices_found": [
    {
      "address": "C4:92:19:E3:F1:5A",
      "name": "SmartSolar-MPPT-100-30",
      "rssi": -45,
      "is_victron": true,
      "device_type": "SmartSolar MPPT"
    }
  ],
  "scan_duration_seconds": 15
}
```

### 2. Connect to a Device

Once you have a device address from the scan, connect to it:

```
victron connect --address C4:92:19:E3:F1:5A
```

Parameters:
- `--address`: MAC address of the Victron device (required)
- `--timeout`: Connection timeout in seconds (default: 10, max: 60)

### 3. Get Device Information

Retrieve detailed information about the connected device:

```
victron device-info --address C4:92:19:E3:F1:5A
```

Example output:
```json
{
  "device_address": "C4:92:19:E3:F1:5A",
  "firmware_version": "4.65",
  "product_type": "SmartSolar MPPT 100/30",
  "serial_number": "123-45678901234",
  "device_specific": {
    "charge_current_limit": 30.0,
    "max_power_point_voltage": 45.2
  }
}
```

### 4. Read Current Values

Get all available values from the device:

```
victron get-values --address C4:92:19:E3:F1:5A
```

Example output:
```json
{
  "device_address": "C4:92:19:E3:F1:5A",
  "values": [
    {
      "key": "V",
      "name": "System Voltage",
      "raw_value": "12.678",
      "float_value": 12.678,
      "unit": "V",
      "timestamp": "2024-04-08T22:39:55Z"
    },
    {
      "key": "A",
      "name": "System Current",
      "raw_value": "5.234",
      "float_value": 5.234,
      "unit": "A",
      "timestamp": "2024-04-08T22:39:55Z"
    }
  ]
}
```

### 5. Subscribe to Real-Time Updates

Monitor values continuously with automatic disconnection:

```
victron subscribe --address C4:92:19:E3:F1:5A --duration 60
```

Parameters:
- `--address`: MAC address of the Victron device (required)
- `--duration`: Subscription duration in seconds (default: 30, max: 300)

This will print updates as they arrive and automatically disconnect after the specified duration.

### 6. Disconnect from Device

Manually disconnect when done:

```
victron disconnect --address C4:92:19:E3:F1:5A
```

## Common Use Cases

### Monitor Solar Charge Controller

Track your solar panel performance:

```bash
# Scan for the device
victron scan --duration 10

# Get current readings (repeat as needed)
victron get-values --address YOUR_DEVICE_ADDRESS

# Monitor in real-time for 5 minutes
victron subscribe --address YOUR_DEVICE_ADDRESS --duration 300
```

### Check Battery Status with SmartShunt

Monitor battery state of charge:

```bash
# Get battery values
victron get-values --address BATTERY_DEVICE_ADDRESS

# Look for SoC (State of Charge), B.V (Battery Voltage), B.A (Current)
```

### Troubleshooting Connection Issues

If devices aren't showing up in scans:
1. Verify Bluetooth is enabled on your computer
2. Ensure the Victron device has power and is advertising via BLE
3. Move closer to the device (BLE range is typically 10-50 meters)
4. Check for interference from other Bluetooth devices
5. Try increasing scan duration

### Supported Devices

The tool supports all Victron devices that advertise over BLE:
- SmartSolar MPPT charge controllers
- SmartShunt battery monitors  
- VE.Direct to Bluetooth Smart adapters
- Any device responding to GATT service UUID `0000ffd200001000`

## Advanced Usage

### Value Keys Reference

Common value keys you'll encounter:

| Key | Name | Unit | Description |
|-----|------|------|-------------|
| V | System Voltage | V | Battery/DC system voltage |
| A | System Current | A | Positive = charging, Negative = discharging |
| W | System Power | W | Instantaneous power |
| P.V | PV Input Voltage 1 | V | Solar panel voltage (input 1) |
| P.A | PV Input Current 1 | A | Solar panel current (input 1) |
| P.W | PV Input Power 1 | W | Solar panel power (input 1) |
| Pi.V | PV Input Voltage 2 | V | Solar panel voltage (input 2) |
| Pi.A | PV Input Current 2 | A | Solar panel current (input 2) |
| Pi.W | PV Input Power 2 | W | Solar panel power (input 2) |
| SoC | State of Charge | % | Battery charge percentage (0-100) |
| T | Temperature | °C | Device temperature |
| B.V | Battery Voltage | V | Battery terminal voltage |
| B.A | Battery Current | A | Battery current |

### Integration Examples

You can use these commands in scripts or automation:

```bash
#!/bin/bash
# Example: Log solar production every 5 minutes

DEVICE="C4:92:19:E3:F1:5A"
LOGFILE="/var/log/solar-production.log"

while true; do
    TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    # Get power value
    POWER=$(victron get-values --address $DEVICE | \
            jq -r '.values[] | select(.key == "P.W") | .float_value')
    
    echo "$TIMESTAMP: $POWER W" >> $LOGFILE
    
    sleep 300
done
```

## Error Handling

Common errors and solutions:

| Error | Cause | Solution |
|-------|-------|----------|
| `No devices found` | Device out of range or not advertising | Move closer, check BLE is enabled on device |
| `Connection timeout` | Device busy or unreachable | Wait and retry, ensure device isn't connected elsewhere |
| `Service not found` | Not a Victron device | Verify the MAC address is correct |
| `Device already disconnected` | Connection lost | Reconnect before getting values |

## Troubleshooting Tips

1. **No devices found in scan:**
   - Ensure Bluetooth adapter is working
   - Check if the Victron device has power
   - Verify BLE advertising is enabled on the Victron device (default on newer models)
   - Remove other interfering Bluetooth devices temporarily

2. **Connection fails:**
   - Device may already be connected to another system
   - Disconnect from other connections first
   - Ensure sufficient signal strength (RSSI > -80 is good, < -90 is poor)

3. **No values returned:**
   - Some devices need a moment to initialize after connection
   - Retry get-values command
   - Check if the device type supports the requested data

## Additional Resources

- [Victron Energy Documentation](https://www.victronenergy.com/support-and-downloads/documents)
- [BLE GATT Specification](https://www.bluetooth.com/specifications/)
- [Full Victron Tool Reference](./victron-usage-examples.md)
