#!/usr/bin/env python3
"""
BLE GATT client for reading device characteristics
Uses bleak library to connect and discover services/characteristics
"""

import sys
import asyncio
import json
from bleak import BleakClient, BleakScanner
from bleak.backends.characteristic import BleakGATTCharacteristic

async def scan_bluetooth_devices(duration=10):
    """Scan for nearby BLE devices"""
    print("Scanning for BLE devices...")
    devices = await BleakScanner.discover(timeout=duration)
    
    result = {"devices": [], "count": len(devices)}
    
    for device in devices:
        # Extract address and name
        addr = device.address if hasattr(device, 'address') else str(device)
        name = device.name if hasattr(device, 'name') and device.name else ""
        
        # Determine if likely Victron device
        is_victron = any(keyword.lower() in name.lower() for keyword in [
            "glow", "smartshunt", "smartsolar", "mppt", "victron"
        ])
        
        result["devices"].append({
            "address": addr,
            "name": name,
            "is_victron": is_victron,
            "device_type": "Unknown"
        })
    
    print(json.dumps(result, indent=2))
    return result

async def discover_gatt_services(address: str):
    """Connect and discover GATT services/characteristics"""
    print(f"Connecting to {address}...")
    
    async with BleakClient(address) as client:
        # Check if connected
        if not client.is_connected:
            print(json.dumps({"error": "Failed to connect"}))
            return None
        
        print("Connected! Discovering services...")
        
        services = await client.discover_services()
        
        result = {
            "address": address,
            "connected": True,
            "services": []
        }
        
        for service in services:
            service_info = {
                "uuid": str(service.uuid),
                "characteristics": []
            }
            
            for char in service.characteristics:
                char_info = {
                    "uuid": str(char.uuid),
                    "properties": []
                }
                
                # Get properties
                if char.properties.read:
                    char_info["properties"].append("read")
                if char.properties.write:
                    char_info["properties"].append("write")
                if char.properties.notify:
                    char_info["properties"].append("notify")
                if char.properties.indicate:
                    char_info["properties"].append("indicate")
                
                service_info["characteristics"].append(char_info)
            
            result["services"].append(service_info)
        
        print(json.dumps(result, indent=2))
        return result

async def read_characteristic(address: str, service_uuid: str, char_uuid: str):
    """Read a specific characteristic value"""
    print(f"Reading {char_uuid} from service {service_uuid} on {address}...")
    
    async with BleakClient(address) as client:
        if not client.is_connected:
            print(json.dumps({"error": "Failed to connect"}))
            return None
        
        try:
            value = await client.read_gatt_char(f"{char_uuid}")
            result = {
                "address": address,
                "service_uuid": service_uuid,
                "characteristic_uuid": char_uuid,
                "value_hex": value.hex(),
                "value_bytes": list(value)
            }
            
            # Try to interpret as common types
            if len(value) == 2:
                # Could be uint16 (battery voltage * 10 or similar)
                result["value_uint16"] = int.from_bytes(value[:2], 'little')
                result["value_float_approx"] = result["value_uint16"] / 10.0
            
            print(json.dumps(result, indent=2))
            return result
        except Exception as e:
            print(json.dumps({"error": str(e)}))
            return None

async def read_battery_level(address: str):
    """Try to read battery level using standard Battery Service"""
    # Standard Battery Service UUID
    BATTERY_SERVICE = "0000180f-0000-1000-8000-00805f9b34fb"
    BATTERY_LEVEL_CHAR = "00002a19-0000-1000-8000-00805f9b34fb"
    
    print(f"Attempting to read battery level from {address}...")
    
    async with BleakClient(address) as client:
        if not client.is_connected:
            print(json.dumps({"error": "Failed to connect"}))
            return None
        
        try:
            value = await client.read_gatt_char(BATTERY_LEVEL_CHAR)
            battery_pct = value[0]  # First byte is percentage (0-100)
            
            result = {
                "address": address,
                "battery_percentage": battery_pct,
                "raw_value_hex": value.hex()
            }
            
            print(json.dumps(result, indent=2))
            return result
        except Exception as e:
            # Battery service not available, try to discover what is
            print(f"Battery service not available: {e}")
            return await discover_gatt_services(address)

async def main():
    if len(sys.argv) < 2:
        print("Usage: victron_client.py <command> [args]")
        print("Commands:")
        print("  scan [duration]     - Scan for BLE devices")
        print("  connect <address>   - Connect and discover services")
        print("  battery <address>   - Read battery level")
        print("  read <addr> <svc> <char> - Read characteristic")
        sys.exit(1)
    
    command = sys.argv[1].lower()
    
    if command == "scan":
        duration = int(sys.argv[2]) if len(sys.argv) > 2 else 10
        await scan_bluetooth_devices(duration)
    
    elif command == "connect" and len(sys.argv) > 2:
        await discover_gatt_services(sys.argv[2])
    
    elif command == "battery" and len(sys.argv) > 2:
        await read_battery_level(sys.argv[2])
    
    elif command == "read" and len(sys.argv) > 4:
        await read_characteristic(sys.argv[2], sys.argv[3], sys.argv[4])
    
    else:
        print("Invalid command or arguments")
        sys.exit(1)

if __name__ == "__main__":
    asyncio.run(main())
