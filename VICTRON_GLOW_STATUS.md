# Victron Glow BLE - Implementation Status

## ✅ Completed

### 1. BLE Scanning (Working)
- **Status**: Fully functional
- **Device Found**: "Glow" at address `CB60561D-CF87-0055-E7B1-009FFA19F942`
- **Implementation**: Python helper script using bleak library
  - File: `scripts/victron_scan.py`
  - Outputs JSON array of discovered devices
  - Go code in `victron/ble_backend_macos.go` calls this script

### 2. Device Discovery & Connection (Working)
- **Status**: Successfully connects and discovers GATT services
- **Implementation**: Python helper script `scripts/victron_connect.py`
- **Discovered Services** on Glow device:
  - Service `68c10001-b17f-4d3a-a290-34ad6499937c`: 2 characteristics (write/notify, write)
  - Service `97580001-ddf1-48be-b73e-182664615d8e`: 4 characteristics (read, write/notify, write, read/write/notify)
  - Service `306b0001-b081-4037-83dc-e59fcc3cdfd0`: 3 characteristics

### 3. Battery Data Reading (Working)
- **Status**: Successfully reads raw battery data
- **Characteristic**: `97580002-ddf1-48be-b73e-182664615d8e` (read)
- **Raw Data**: `ff180100ff5202008c0091a15000f2ffffff0000` (20 bytes)
- **Decoded Voltage**: 14.0V
  - Extracted from bytes 8-9: `0x8c 0x00` = 140 (little-endian uint16) / 10 = 14.0V

### 4. Code Updates
- Added "Glow", "ECO-", "Solar48" patterns to `VictronAdvertisementPatterns` in `victron/ble.go`
- Updated `ble_backend_macos.go` with Connect() and ReadCharacteristic() function signatures (need Python helper integration)

## 🔄 In Progress / TODO

### 1. Complete Go Integration
- Need to add `discoverServicesWithPython()` and `readCharacteristicWithPython()` functions to `victron/ble_backend_macos.go`
- These will call the Python helpers and parse JSON responses into Go structs

### 2. Create Go-Based Battery Decoder
- Add decoder function for Glow binary protocol in `victron/victron.go` or new file
- Format: Parse 20-byte response, extract voltage from bytes 8-9 (LE uint16 / 10)
- May need to decode additional values (state of charge, current, etc.)

### 3. Add Glow Device Type
- Add `DeviceTypeGlow` to DeviceType enum in `victron/victron.go`
- Update device detection logic to identify Glow devices by name

### 4. MCP Tool Integration
- Ensure the victron MCP action properly exposes:
  - `scan`: List available BLE devices (✅ working)
  - `connect`: Connect to a device by address
  - `get_values`: Read battery voltage and other metrics

## 📝 Technical Details

### Glow Device GATT Profile
```
Service: 97580001-ddf1-48be-b73e-182664615d8e (Main battery service)
├── 97580002-ddf1-48be-b73e-182664615d8e [READ] - Battery status data
├── 97580003-ddf1-48be-b73e-182664615d8e [WRITE, NOTIFY] - Command/notifications
├── 97580004-ddf1-48be-b73e-182664615d8e [WRITE] - Control
└── 97580006-ddf1-48be-b73e-182664615d8e [READ, WRITE, NOTIFY] - Configuration/data
```

### Binary Data Format (from characteristic read)
```
Offset  Size  Description
------  ---   -----------
0       2     Header/magic (0xFF 0x18)
2       2     Version/flags (0x01 0x00)
4       2     Unknown (0xFF 0x52)
6       2     Unknown (0x02 0x00)
8       2     Voltage * 10 (0x8C 0x00 = 14.0V) ✓
10      2     Unknown (0x91 0xA1)
12      2     Unknown (0x50 0x00)
14      4     Unknown/padding (0xF2 0xFF 0xFF 0xFF)
18      2     Padding (0x00 0x00)
```

### Python Helper Scripts

**scripts/victron_scan.py** - Scans for BLE devices, outputs JSON:
```bash
python3 scripts/victron_scan.py [duration_seconds]
```

**scripts/victron_connect.py** - Connects and reads data:
```bash
python3 scripts/victron_connect.py <address> <action> [char_uuid]
# Actions: connect, read_char, subscribe
```

## 🔧 Next Steps (Priority Order)

1. Add `discoverServicesWithPython()` function to parse service discovery JSON
2. Add `readCharacteristicWithPython()` function to parse read value JSON  
3. Create `DecodeGlowBatteryData()` function in Go for binary decoding
4. Add Glow device type detection and String() method
5. Build and test full end-to-end flow through MCP tool
6. Update TODO items once battery voltage reading is complete
