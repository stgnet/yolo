# YOLO Victron Tool Usage Examples

This guide demonstrates how to use the Victron Energy BLE integration tool in YOLO for monitoring solar panels, battery systems, and VE.Direct devices.

## Quick Start

### 1. Scan for Nearby Devices

```json
{
  "name": "victron",
  "arguments": {
    "action": "scan",
    "duration": "10"
  }
}
```

**Expected Response:**
```json
{
  "status": "success",
  "message": "Found 2 device(s)",
  "devices": [
    {
      "address": "CC:50:C3:29:7D:B5",
      "name": "SmartSolar MPPT 100/30",
      "rssi": -45,
      "is_victron": true
    },
    {
      "address": "A4:C3:F0:12:34:56",
      "name": "SmartShunt",
      "rssi": -62,
      "is_victron": true
    }
  ]
}
```

### 2. Connect to a Device

```json
{
  "name": "victron",
  "arguments": {
    "action": "connect",
    "address": "CC:50:C3:29:7D:B5",
    "timeout": "10"
  }
}
```

**Expected Response:**
```json
{
  "status": "success",
  "message": "Connected to CC:50:C3:29:7D:B5 (SmartSolar)",
  "device_info": {
    "address": "CC:50:C3:29:7D:B5",
    "name": "SmartSolar MPPT 100/30",
    "device_type": "SmartSolar",
    "connected": true
  }
}
```

### 3. Get Current Sensor Values

```json
{
  "name": "victron",
  "arguments": {
    "action": "get_values",
    "address": "CC:50:C3:29:7D:B5",
    "timeout": "5"
  }
}
```

**Expected Response:**
```json
{
  "status": "success",
  "message": "Received 8 value(s)",
  "values": [
    {
      "key": "V",
      "raw_value": "13.85",
      "float_value": 13.85,
      "timestamp": "2026-03-20T14:30:25Z"
    },
    {
      "key": "A",
      "raw_value": "5.2",
      "float_value": 5.2,
      "timestamp": "2026-03-20T14:30:25Z"
    },
    {
      "key": "W",
      "raw_value": "72.0",
      "float_value": 72.0,
      "timestamp": "2026-03-20T14:30:25Z"
    },
    {
      "key": "P.V",
      "raw_value": "18.2",
      "float_value": 18.2,
      "timestamp": "2026-03-20T14:30:25Z"
    },
    {
      "key": "P.A",
      "raw_value": "4.8",
      "float_value": 4.8,
      "timestamp": "2026-03-20T14:30:25Z"
    },
    {
      "key": "P.W",
      "raw_value": "87.36",
      "float_value": 87.36,
      "timestamp": "2026-03-20T14:30:25Z"
    },
    {
      "key": "SoC",
      "raw_value": "85",
      "float_value": 85.0,
      "timestamp": "2026-03-20T14:30:26Z"
    },
    {
      "key": "T",
      "raw_value": "35.5",
      "float_value": 35.5,
      "timestamp": "2026-03-20T14:30:26Z"
    }
  ]
}
```

### 4. Monitor Real-Time Updates (Subscribe)

Monitor a device for real-time sensor updates over a period of time:

```json
{
  "name": "victron",
  "arguments": {
    "action": "subscribe",
    "address": "CC:50:C3:29:7D:B5",
    "timeout": "30"
  }
}
```

This will stream values for 30 seconds, capturing changes as they occur. Useful for:
- Observing charging curves
- Tracking battery drain patterns
- Monitoring solar production fluctuations

### 5. Get Device Information

```json
{
  "name": "victron",
  "arguments": {
    "action": "device_info",
    "address": "CC:50:C3:29:7D:B5"
  }
}
```

**Expected Response:**
```json
{
  "status": "success",
  "message": "Device information retrieved",
  "device_info": {
    "address": "CC:50:C3:29:7D:B5",
    "name": "SmartSolar MPPT 100/30",
    "device_type": "SmartSolar",
    "connected": true
  }
}
```

### 6. Disconnect

```json
{
  "name": "victron",
  "arguments": {
    "action": "disconnect",
    "address": "CC:50:C3:29:7D:B5"
  }
}
```

Or disconnect all devices:
```json
{
  "name": "victron",
  "arguments": {
    "action": "disconnect"
  }
}
```

---

## Common Use Cases

### Solar Panel Monitoring

Track your solar installation's performance:

```json
// Scan and find SmartSolar device
{"name": "victron", "arguments": {"action": "scan"}}

// Connect to it
{"name": "victron", "arguments": {"action": "connect", "address": "<MAC_FROM_SCAN>"}}

// Get current solar production
{"name": "victron", "arguments": {"action": "get_values", "address": "<MAC>", "timeout": "5"}}
```

**Key metrics to monitor:**
- `P.W` - PV input power (current solar production)
- `P.V` - PV input voltage
- `P.A` - PV input current
- `W` - Charging power going to battery
- `V` - System voltage

### Battery Health Monitoring

Monitor battery state of charge and health:

```json
// Connect to SmartShunt or device with battery monitoring
{"name": "victron", "arguments": {"action": "connect", "address": "<MAC>"}}

// Monitor battery metrics for an hour
{"name": "victron", "arguments": {"action": "subscribe", "address": "<MAC>", "timeout": "3600"}}
```

**Key battery metrics:**
- `SoC` - State of charge percentage
- `B.V` - Battery voltage
- `B.A` - Battery current (negative = discharging)
- `V.Ar` - Energy yield today (Ah)
- `V.Wh.ar` - Energy yield today (Wh)

### Automated Daily Solar Report

Example autonomous workflow to generate daily solar reports:

```json
// 1. Scan for devices
{"name": "victron", "arguments": {"action": "scan"}}

// 2. Connect to SmartSolar
{"name": "victron", "arguments": {"action": "connect", "address": "<MAC>"}}

// 3. Get today's energy yield
{"name": "victron", "arguments": {"action": "get_values", "address": "<MAC>", "timeout": "5"}}

// Extract V.Wh.ar value and format report

// 4. Send email with daily summary
{
  "name": "send_email",
  "arguments": {
    "subject": "Daily Solar Report - March 20, 2026",
    "body": "Today's solar production: 87.36 Wh\nBattery SOC: 85%\nPeak power: 120W"
  }
}

// 5. Disconnect
{"name": "victron", "arguments": {"action": "disconnect"}}
```

### Scheduled Monitoring Task

Set up automatic monitoring with cron scheduling:

```json
{
  "name": "schedule_add",
  "arguments": {
    "name": "hourly-solar-check",
    "cron": "0 * * * *",
    "prompt": "Check solar production and battery status. Scan for Victron devices, connect to SmartSolar, get current values, log the readings with timestamp, then disconnect."
  }
}
```

---

## Sensor Key Reference

### Voltage Measurements
| Key | Description | Unit |
|-----|-------------|------|
| `V` | System voltage | V |
| `B.V` | Battery voltage | V |
| `P.V` | PV input voltage (panel 1) | V |
| `P.V(2)` | PV input voltage (panel 2) | V |

### Current Measurements
| Key | Description | Unit |
|-----|-------------|------|
| `A` | System current (charging) | A |
| `B.A` | Battery current | A |
| `P.A` | PV input current (panel 1) | A |
| `P.A(2)` | PV input current (panel 2) | A |

### Power Measurements
| Key | Description | Unit |
|-----|-------------|------|
| `W` | System power (charging) | W |
| `P.W` | PV input power (panel 1) | W |
| `P.W(2)` | PV input power (panel 2) | W |

### Battery State
| Key | Description | Unit |
|-----|-------------|------|
| `SoC` | State of charge | % |
| `V.Ar` | Energy yield today | Ah |
| `V.Wh.ar` | Energy yield today | Wh |

### Other Metrics
| Key | Description | Unit |
|-----|-------------|------|
| `T` | Device temperature | °C |
| `Alg.S` | Charging algorithm state | - |
| `S` | Stage | - |
| `E` | Error code | - |

---

## Platform Support

### Linux (Primary)
✅ Scanning works  
⚠️ GATT reading in progress  
**Requirements:**
- Bluetooth adapter with BLE 4.0+ support
- BlueZ 5.43 or higher
- Proper D-Bus permissions (may require `sudo` or udev rules)

### macOS / Windows
🔧 Mock backend available for testing  
**Limitations:** Real hardware requires platform-specific BLE implementations

---

## Troubleshooting

### No devices found in scan
- Ensure Bluetooth is enabled and working
- Check device is within range (Victron BLE range: ~10m indoor)
- Verify device has VE.Direct Bluetooth adapter or built-in BLE
- Increase scan duration to 20-30 seconds

### Connection fails
- Check no other app is connected to the device
- Try disconnecting all devices first, then reconnect
- Verify MAC address is correct (use `scan` action)

### No values received
- Ensure device supports VE.Direct over BLE
- Check timeout duration (increase to 10s if needed)
- Some devices may need to be woken up first

### Permission errors on Linux
```bash
# Add user to bluetooth group
sudo usermod -aG bluetooth $USER

# Restart Bluetooth service
sudo systemctl restart bluetooth

# Or run with sudo (not recommended for production)
sudo yolo
```

---

## More Information

- **Full Tool Documentation**: [DOCS/tools.md#victron-energy-ble-devices](./tools.md#victron-energy-ble-devices)
- **Implementation Details**: [DOCS/VICTRON_BLE_LIBRARIES.md](./VICTRON_BLE_LIBRARIES.md)
- **Source Code**: `tools_victron.go` and `victron/` package
