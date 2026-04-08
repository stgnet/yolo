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
Alternative to muka/go-bluetooth for Linux. Less mature GATT implementation.

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

## Current YOLO Victron Tool Status

✅ **Integrated into YOLO**: `tools_victron.go` fully implemented  
✅ **MCP Tool Registered**: Available with proper schema  
✅ **Mock Backend**: Works without hardware for testing  
⚠️ **Linux BlueZ**: Scanning works, GATT reading incomplete  
❌ **macOS/Windows**: Not implemented  

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
