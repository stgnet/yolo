# Go Bluetooth Low Energy (BLE) & GATT Libraries Research

## Executive Summary

After exhaustive research, here are the available open-source Go libraries for Bluetooth Low Energy with GATT support:

---

## 1. muka/go-bluetooth ⭐ **RECOMMENDED**

### Overview
- **GitHub**: https://github.com/muka/go-bluetooth
- **Status**: Archived (July 2024) but functional
- **Stars**: 674 | **Forks**: 134
- **Platform**: Linux only (BlueZ via D-Bus)
- **Last Update**: July 1, 2024

### Features
✅ Full GATT API (services, characteristics, descriptors)  
✅ Device discovery and scanning  
✅ Pairing and authentication support  
✅ iBeacon and Eddystone beaconing  
✅ Mesh API support (BlueZ 5.53+)  
✅ Already in your go.mod dependencies!

### Installation
```bash
go get github.com/muka/go-bluetooth@v0.0.0-20240701044517-04c4f09c514e
```

### Victron Use Case
**Best choice for Linux-based YOLO deployments**. Provides complete BlueZ D-Bus wrapper with GATT support needed to read VE.Direct data from Victron devices.

### Example Usage
```go
import (
    "github.com/muka/go-bluetooth/api"
    "github.com/muka/go-bluetooth/bluez/profile/adapter"
    "github.com/muka/go-bluetooth/bluez/profile/gatt"
)

// Discover GATT services
services, err := device.GetProperties()
gattServices := services.GetValue("Services")

// Find specific service by UUID
for _, servicePath := range gattServices {
    service, _ := gatt.NewGattService1(servicePath)
    props, _ := service.GetProperties()
    // Check UUID and find characteristics
}
```

### Limitations
- Linux only (requires BlueZ 5.43+)
- Repository archived (no new features expected)
- Requires proper D-Bus permissions (may need root or udev rules)

---

## 2. go-ble/ble

### Overview
- **GitHub**: https://github.com/go-ble/ble
- **Status**: Forked from moogle19/ble, macOS portion unmaintained
- **Platform**: Linux/macOS (macOS not actively maintained)
- **Last Activity**: Minimal recent commits

### Features
✅ Cross-platform BLE API  
✅ GATT support (services, characteristics)  
✅ Lower-level abstraction than muka/go-bluetooth

### Installation
```bash
go get github.com/go-ble/ble
```

### Victron Use Case
Alternative to muka/go-bluetooth for Linux. Less mature GATT implementation. Used in YOLO's macOS backend.

### Limitations
- macOS support not maintained
- Smaller community (fewer stars/forks)
- Less comprehensive than muka/go-bluetooth

---

## 3. Other Libraries Mentioned

### go-bluez (Various Authors)
Several smaller implementations exist but are either:
- Unmaintained (last commit >2 years ago)
- Incomplete GATT support
- No Victron-specific testing

### Platform-Specific Solutions
- **macOS**: Requires cgo bindings to CoreBluetooth framework (no maintained Go wrapper)
- **Windows**: Would require WinRT Bluetooth API bindings (no maintained Go wrapper)

---

## Comparison Table

| Library | Linux | macOS | Windows | GATT | Active Maint | Stars |
|---------|-------|-------|---------|------|--------------|-------|
| muka/go-bluetooth | ✅ | ❌ | ❌ | ✅ Full | Archived* | 674 |
| go-ble/ble | ✅ | ⚠️ | ❌ | ✅ Basic | Minimal | ~50 |
| Custom cgo | ❌ | ✅ | ✅ | ✅ | N/A | N/A |

*Archived but functional and well-tested

---

## Recommendations for YOLO Victron Tool

### Short Term (Immediate)
1. **Use Mock Backend**: Already fully functional for testing on all platforms
2. **Complete Linux BlueZ**: Finish GATT characteristic discovery in `victron/bluez/backend.go`
3. **Document Requirements**: Clearly state hardware/permission needs

### Long Term
1. **macOS Implementation**: Either implement cgo bindings or use go-ble/ble as base
2. **Windows Support**: Would require significant work on WinRT bindings
3. **Consider Rust Binding**: SimpleBLE is actively maintained and could be wrapped

---

## Victron-Specific Information

### VE.Direct Protocol over BLE
Victron devices expose VE.Direct data via a GATT characteristic:
- **Service UUID**: Typically `0000fff0-0000-1000-8000-00805f9b34fb` (VE.Direct Service)
- **Characteristic UUID**: `0000fff1-0000-1000-8000-00805f9b34fb` (Notification characteristic)
- **Data Format**: ASCII strings with `/` prefix and checksum suffix

### Required Implementation Steps
1. Scan for devices advertising Victron names ("VE.Direct", "SmartSolar", etc.)
2. Connect to device
3. Discover GATT services
4. Find VE.Direct service by UUID
5. Enable notifications on characteristic
6. Parse incoming VE.Direct protocol messages (already implemented in `victron/parser.go`)

---

## Victron Glow LED Indicator Device

### Overview
The Victron Glow is a Bluetooth LE LED indicator module that displays battery system status through color-coded lights. It's designed to provide visual feedback about battery state of charge, faults, and charging status.

### Features
- **Bluetooth LE**: Advertises with name containing "glow" (case-insensitive)
- **LED Colors**: 
  - 🔴 Red: Low battery or fault conditions
  - 🟡 Yellow/Orange: Charging or medium charge
  - 🟢 Green: Full charge
  - ⚪ White: Standby/idle
- **Power Saving**: May enter sleep mode when not actively transmitting
- **No Battery**: Powered directly from the system it monitors

### Technical Specifications
- **Protocol**: VE.Direct over BLE GATT
- **Service UUID**: `0000fff0-0000-1000-8000-00805f9b34fb`
- **Characteristic UUID**: `0000fff1-0000-1000-8000-00805f9b34fb`
- **Discovery Name Pattern**: Contains "glow", "Glow", or "glow-" prefix

### Connecting to Glow Device

#### Prerequisites
1. Ensure Bluetooth is enabled on macOS
2. Make sure the Glow device is:
   - Powered on (connected to battery system)
   - Within ~10 meters range
   - Not in deep sleep mode
3. Grant Bluetooth permissions to the application

#### Scanning for Devices
```bash
# Using the victron MCP tool
victron scan --duration=20

# Or programmatically using the example:
go run examples/victron-ble-example.go
```

#### Common Issues

**Problem**: No devices found during scan
**Solutions**:
1. Check if Glow device is powered (LED should be lit)
2. Wake device by pressing button or triggering battery event
3. Move closer to the device (< 5 meters)
4. Ensure Bluetooth is enabled in macOS System Settings
5. Check that YOLO has Bluetooth permissions (System Preferences > Privacy > Bluetooth)

**Problem**: Device found but can't connect
**Solutions**:
1. Verify MAC address/UUID is correct
2. Try scanning again - some devices advertise intermittently
3. Ensure no other device is connected to the Glow
4. Reset the Glow device if possible

#### Reading Measurements from Victron Devices (Including Glow)

Once connected to a Victron device, you can read various measurements via the VE.Direct protocol over BLE GATT notifications.

**Supported Measurement Types:**

| Category | Data Key | Description | Unit |
|----------|----------|-------------|------|
| **Voltage** | `V` | System voltage | Volts (V) |
| | `B.V` | Battery voltage | Volts (V) |
| | `P.V` | PV input voltage (panel 1) | Volts (V) |
| | `P.V(2)` | PV input voltage (panel 2) | Volts (V) |

| **Current** | `A` | System current (charging) | Amps (A) |
| | `B.A` | Battery current | Amps (A) |
| | `P.A` | PV input current (panel 1) | Amps (A) |
| | `P.A(2)` | PV input current (panel 2) | Amps (A) |

| **Power** | `W` | System power | Watts (W) |
| | `P.W` | PV input power (panel 1) | Watts (W) |
| | `P.W(2)` | PV input power (panel 2) | Watts (W) |

| **State of Charge** | `SoC` | Battery state of charge | Percentage (%) |
| | `V.Ar` | Energy yield today (Ah) | Amp-hours (Ah) |
| | `V.Wh.ar` | Energy yield today (Wh) | Watt-hours (Wh) |

| **Other** | `T` | Device temperature | Celsius (°C) |
| | `Alg.S` | Charging algorithm state | Enum |
| | `S` | Stage | Enum |
| | `E` | Error code | Code |
| | `/Dev` | Device type string | String |
| | `/Rev` | Firmware revision | String |
| | `/Sn` | Serial number | String |

**How to Read Measurements:**

1. **Connect to the device** using its BLE address
2. **Discover GATT services** and find the Victron service (UUID: `0000fff0-...`)
3. **Subscribe to notifications** on the characteristic (UUID: `0000fff1-...`)
4. **Parse incoming VE.Direct frames** - each frame contains one data point
5. **Build a complete picture** by collecting multiple frames over time

**Example Flow:**
```
Frame 1: /V(V=12.650)A  → System Voltage = 12.650V
Frame 2: /A(A=2.500)B   → System Current = 2.500A
Frame 3: /P.V(P.V=38.100)C → PV Voltage Panel 1 = 38.100V
Frame 4: /SoC(SoC=85)D  → State of Charge = 85%
```

**Note on Glow LED Indicator:**
The Victron Glow device primarily provides **visual status indicators** via its LEDs. For detailed voltage/current measurements, you'll typically want to connect to:
- **SmartSolar MPPT** - for solar charging data
- **SmartShunt** - for battery monitoring data
- **VE.Direct Bluetooth adapter** - to add BLE to any VE.Direct device

The Glow can be connected but may provide more limited telemetry compared to dedicated monitoring devices.

---

## Current YOLO Victron Tool Status

✅ **Integrated into YOLO**: `tools_victron.go` fully implemented  
✅ **MCP Tool Registered**: Available with proper schema  
✅ **Mock Backend**: Works without hardware for testing  
⚠️ **Linux BlueZ**: Scanning works, GATT reading incomplete  
❌ **macOS/Windows**: Not fully implemented  

### How to Use Now
```bash
# Scan for devices (uses mock backend by default)
yolo victron --action scan --duration 10

# With real hardware on Linux:
yolo victron --action connect --address "XX:XX:XX:XX:XX:XX"
yolo victron --action get_values --address "XX:XX:XX:XX:XX:XX"
```

---

## Conclusion

**muka/go-bluetooth** is the best available option for Linux-based Victron support. The library is comprehensive, well-documented, and already in your dependencies. Complete the BlueZ backend implementation to enable real hardware support on Linux systems.

For cross-platform support, significant work would be needed to implement platform-specific backends or find/wrap alternative libraries from other languages (Rust's SimpleBLE being the most promising).

---

## Example Usage with YOLO Victron Package

See `victron/examples/victron-ble-example.go` for:
1. Scanning for nearby Victron devices (including Glow)
2. Connecting to discovered devices
3. Discovering GATT services and characteristics
4. Reading VE.Direct data from the device

```bash
# Run the example
go run victron/examples/victron-ble-example.go

# Or scan with a specific duration
victron scan --duration=15
```
