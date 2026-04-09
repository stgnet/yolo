# Victron BLE Tool Usage Examples

This document provides practical examples of using the Victron BLE tool to interact with Victron energy monitoring devices.

## Prerequisites

- Device with Bluetooth Low Energy (BLE) support (macOS, Linux with BlueZ, or Windows with appropriate drivers)
- Victron device with VE.Direct over BLE support:
  - SmartSolar MPPT charge controllers
  - SmartShunt battery monitors
  - Other VE.Direct enabled devices
- Go 1.21+

## Quick Start

### 1. Scan for Nearby Devices

```bash
# Using the YOLO agent (recommended)
victron scan --timeout 15

# Or directly with the library
yolo victron-scan --duration 15
```

**Example Output:**
```json
{
  "device_count": 2,
  "devices": [
    {
      "address": "AA:BB:CC:DD:EE:FF",
      "name": "SmartSolar MPPT 75/15",
      "signal_strength": -45
    },
    {
      "address": "11:22:33:44:55:66", 
      "name": "SmartShunt 500A",
      "signal_strength": -52
    }
  ]
}
```

### 2. Get Device Information

```bash
victron device-info --address AA:BB:CC:DD:EE:FF
```

**Example Output:**
```json
{
  "device_type": "SmartSolar MPPT",
  "firmware_version": "4.63.1_76896",
  "hardware_revision": "0x502",
  "product_code": "VCMPPT050140011"
}
```

### 3. Read Current Values (One-shot)

```bash
victron get-values --address AA:BB:CC:DD:EE:FF --timeout 30
```

**Example Output:**
```json
{
  "values": {
    "PV Voltage": {"value": 45.2, "unit": "V"},
    "Battery Voltage": {"value": 13.8, "unit": "V"},
    "Charge Current": {"value": 2.5, "unit": "A"},
    "Power": {"value": 35.6, "unit": "W"},
    "State": {"value": "Bulk", "unit": ""},
    "Temperature": {"value": 28.5, "unit": "°C"}
  }
}
```

### 4. Subscribe to Real-Time Updates

```bash
# Monitor for 60 seconds with automatic reconnection
victron subscribe --address AA:BB:CC:DD:EE:FF --duration 60 --reconnect true
```

**Example Output (streaming):**
```json
{"timestamp": "2024-01-15T10:30:15Z", "values": {"PV Voltage": 45.3, "Battery Voltage": 13.8}}
{"timestamp": "2024-01-15T10:30:18Z", "values": {"PV Voltage": 45.4, "Charge Current": 2.6}}
{"timestamp": "2024-01-15T10:30:21Z", "values": {"Power": 36.1, "State": "Bulk"}}
```

## Common Use Cases

### Solar System Health Monitoring

Monitor your solar charge controller to ensure it's operating correctly:

```bash
# Check if the system is charging
victron get-values --address YOUR_DEVICE_MAC \
  | jq '.values["State"].value'
  
# Expected values: "Off", "Bulk", "Absorption", "Float", "Equalize"
```

### Battery Voltage Alerts

Track battery voltage over time to detect potential issues:

```bash
# Record battery voltage every minute for an hour
for i in {1..60}; do
  result=$(victron get-values --address YOUR_DEVICE_MAC)
  voltage=$(echo $result | jq -r '.values["Battery Voltage"].value')
  echo "$(date): ${voltage}V" >> battery-voltage.log
  sleep 60
done
```

### Power Production Tracking

Log your solar power production for analysis:

```bash
# Subscribe to updates and save to file
victron subscribe --address YOUR_DEVICE_MAC --duration 3600 \
  | tee solar-production.log
```

## Value Reference

### SmartSolar MPPT Charge Controller Values

| VE.Direct Code | Description | Unit | Example |
|---------------|-------------|------|---------|
| rRv | PV Voltage | V | 45.2 |
| rRp | Battery Voltage | V | 13.8 |
| cAc | Charge Current | A | 2.5 |
| pPo | Power | W | 35.6 |
| sSt | State | - | Bulk |
| tTe | Temperature | °C | 28.5 |
| eEr | Error Code | - | No fault |
| fFi | Firmware Version | - | 4.63.1_76896 |

### SmartShunt Battery Monitor Values

| VE.Direct Code | Description | Unit | Example |
|---------------|-------------|------|---------|
| rRp | Battery Voltage | V | 12.6 |
| cCc | Current | A | -0.5 |
| sSoC | State of Charge | % | 85.3 |
| pWh | Energy Consumed | Wh | 1234.5 |
| dDs | Days to Empty | days | 7.2 |

## Troubleshooting

### Device Not Found

**Problem:** Scan returns no devices  
**Solutions:**
- Ensure the device is powered on and within Bluetooth range (~10 meters)
- Check if your OS has Bluetooth enabled
- Verify the device supports VE.Direct over BLE (check Victron documentation)
- On Linux, ensure BlueZ is installed: `sudo apt install bluez`

### Connection Timeout

**Problem:** Get "connection timed out" error  
**Solutions:**
- Increase timeout value: `--timeout 60`
- Move closer to the device (Bluetooth signal strength matters)
- Reduce interference from other Bluetooth devices
- Try disconnecting and reconnecting: `victron disconnect --address YOUR_MAC`

### Incomplete Value Data

**Problem:** Not all expected values are returned  
**Solutions:**
- Some values may not be available depending on device state (e.g., error codes only appear during faults)
- Subscribe for longer to capture more values: `--duration 120`
- Check Victron VE.Direct specification for value availability conditions

### Permission Denied (Linux)

**Problem:** "Operation not permitted" or similar errors  
**Solutions:**
```bash
# Add user to bluetooth group
sudo usermod -aG bluetooth $USER

# Restart Bluetooth service
sudo systemctl restart bluetooth

# May need to disable Secure Simple Pairing:
sudo apt install blueman
blueman-manager  # Configure in GUI
```

## Programmatic Usage

### Go Library Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/scottstg/yolo/victron"
)

func main() {
    // Create client with production backend
    client := victron.NewClient(victron.WithRealBackend())
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Connect to device
    err := client.Connect(ctx, "AA:BB:CC:DD:EE:FF")
    if err != nil {
        log.Fatalf("Connection failed: %v", err)
    }
    defer client.Disconnect()
    
    // Get device info
    info, err := client.GetDeviceInfo(ctx)
    if err != nil {
        log.Fatalf("Device info failed: %v", err)
    }
    fmt.Printf("Connected to: %s v%s\n", 
        info.DeviceType, info.FirmwareVersion)
    
    // Subscribe to updates
    ch := client.Subscribe(ctx)
    timeout := time.After(60 * time.Second)
    
    for {
        select {
        case update := <-ch:
            fmt.Printf("Update: %+v\n", update.Values)
        case <-timeout:
            return
        }
    }
}
```

### Shell Script Example

```bash
#!/bin/bash
# Monitor solar system health

DEVICE_MAC="AA:BB:CC:DD:EE:FF"

check_solar_health() {
    echo "Checking solar system status..."
    
    # Get current values
    result=$(yolo victron-get-values --address $DEVICE_MAC)
    
    # Extract key metrics
    state=$(echo $result | jq -r '.values.State.value')
    pv_voltage=$(echo $result | jq -r '.values["PV Voltage"].value')
    battery_voltage=$(echo $result | jq -r '.values["Battery Voltage"].value')
    
    echo "State: $state"
    echo "PV Voltage: ${pv_voltage}V"
    echo "Battery Voltage: ${battery_voltage}V"
    
    # Alert if not charging during daylight hours
    hour=$(date +%H)
    if [[ $hour -ge 8 && $hour -le 16 ]]; then
        if [[ "$state" == "Off" ]]; then
            echo "⚠️  WARNING: System is Off during daylight hours!"
            return 1
        fi
    fi
    
    return 0
}

# Run check
check_solar_health
exit $?
```

## Performance Tips

1. **Batch Reads:** Use `get-values` for one-time reads, `subscribe` for continuous monitoring
2. **Connection Reuse:** Keep connections open when making multiple reads to reduce connection overhead
3. **Timeout Tuning:** Adjust timeouts based on your environment (5-10s typically sufficient)
4. **Signal Strength:** Values with RSSI > -60 dBm provide most reliable data

## Security Considerations

- BLE communication is encrypted but not authenticated by default
- Device addresses can be spoofed in compromised environments
- Use only on trusted networks
- Monitor for unexpected value changes that may indicate tampering

## Additional Resources

- [Victron VE.Direct Protocol Specification](https://github.com/VictronEnergy/ve-direct-docker/blob/master/protocol.md)
- [Victron Energy Community Forum](https://community.victronenergy.com/)
- [BlueZ Documentation (Linux)](https://www.bluez.org/)
- [CoreBluetooth Documentation (macOS/iOS)](https://developer.apple.com/documentation/corebluetooth)

## Contributing

Found a bug or want to add support for additional Victron devices? Please open an issue or submit a pull request!

```
