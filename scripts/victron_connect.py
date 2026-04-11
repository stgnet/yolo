#!/usr/bin/env python3
"""
BLE connector using bleak to connect to Victron/Glow devices.
Discovers services and characteristics, or reads specific values.
Usage: python3 scripts/victron_connect.py <mac_address> [command]
Commands: discover (default), read_battery, notify, connect, read_char
"""
import asyncio
import json
import sys
from bleak import BleakClient

def output_result(result_dict):
    """Output result as JSON"""
    print(json.dumps(result_dict, indent=2))

async def discover_services(address):
    """Connect and discover GATT services/characteristics"""
    try:
        async with BleakClient(address, timeout=10.0) as client:
            services = client.services
            
            result = {
                "address": address,
                "services": []
            }
            
            for service in services:
                service_info = {
                    "uuid": str(service.uuid),
                    "characteristics": []
                }
                
                chars = service.characteristics
                for char in chars:
                    char_info = {
                        "uuid": str(char.uuid),
                        "properties": char.properties
                    }
                    service_info["characteristics"].append(char_info)
                
                result["services"].append(service_info)
            
            return result
    except Exception as e:
        return {"error": str(e)}

async def read_battery_voltage(address):
    """Read battery voltage from Glow device"""
    try:
        async with BleakClient(address, timeout=10.0) as client:
            services = client.services
            
            results = []
            for service in services:
                chars = service.characteristics
                for char in chars:
                    if "read" in char.properties:
                        try:
                            value = await client.read_gatt_char(char.uuid)
                            byte_list = list(value)
                            
                            # Try to interpret as battery voltage
                            voltage = None
                            if len(byte_list) >= 2:
                                uint16_val = int.from_bytes(value[:2], 'little')
                                if uint16_val > 100 and uint16_val < 10000:
                                    voltage = uint16_val / 100.0
                            
                            results.append({
                                "service_uuid": str(service.uuid),
                                "char_uuid": str(char.uuid),
                                "value_hex": value.hex(),
                                "value_bytes": byte_list,
                                "voltage_v": voltage
                            })
                        except Exception as e:
                            pass  # Skip characteristics that can't be read
            
            return {"results": results}
    except Exception as e:
        return {"error": str(e)}

async def notify_battery(address):
    """Subscribe to battery notifications from Glow"""
    print(f"Connecting to {address}...")
    
    async with BleakClient(address, timeout=10.0) as client:
        services = client.services
        
        found_notify_chars = []
        for service in services:
            chars = service.characteristics
            for char in chars:
                if "notify" in char.properties:
                    found_notify_chars.append((str(service.uuid), char))
        
        print(f"\nFound {len(found_notify_chars)} characteristics with notify:")
        
        def callback(sender, data):
            val = int.from_bytes(data[:2], 'little') if len(data) >= 2 else 0
            voltage = val / 100.0 if val > 100 and val < 10000 else None
            print(f"Notification: {data.hex()} ({list(data)})")
            if voltage:
                print(f"  Voltage: {voltage}V")
        
        for service_uuid, char in found_notify_chars:
            print(f"  Subscribing to {char.uuid}...")
            try:
                await client.start_notify(char.uuid, callback)
                await asyncio.sleep(5)
                await client.stop_notify(char.uuid)
            except Exception as e:
                print(f"    Subscribe failed: {e}")

async def connect_command(address):
    """Connect and discover services (for Go backend compatibility)"""
    try:
        async with BleakClient(address, timeout=10.0) as client:
            services = client.services
            
            result_services = []
            for service in services:
                service_info = {
                    "uuid": str(service.uuid),
                    "characteristics": []
                }
                
                chars = service.characteristics
                for char in chars:
                    char_info = {
                        "uuid": str(char.uuid),
                        "properties": char.properties
                    }
                    service_info["characteristics"].append(char_info)
                
                result_services.append(service_info)
            
            return {
                "success": True,
                "message": f"Connected to {client.address}",
                "services": result_services
            }
    except Exception as e:
        return {"success": False, "error": str(e)}

async def read_char_command(address, char_uuid):
    """Read a specific characteristic (for Go backend compatibility)"""
    try:
        async with BleakClient(address, timeout=10.0) as client:
            value = await client.read_gatt_char(char_uuid)
            
            return {
                "success": True,
                "message": f"Read characteristic {char_uuid}",
                "value_hex": value.hex(),
                "value_bytes": list(value)
            }
    except Exception as e:
        return {"success": False, "error": str(e)}

async def main():
    if len(sys.argv) < 2:
        print("Usage: python3 scripts/victron_connect.py <mac_address> [command]")
        print("Commands: discover, read_battery, notify, connect, read_char <uuid>")
        sys.exit(1)
    
    address = sys.argv[1]
    command = sys.argv[2].lower() if len(sys.argv) > 2 else "discover"
    
    try:
        if command == "discover":
            result = await discover_services(address)
            output_result(result)
        elif command == "read_battery":
            result = await read_battery_voltage(address)
            output_result(result)
        elif command == "notify":
            await notify_battery(address)
        elif command == "connect":
            result = await connect_command(address)
            output_result(result)
        elif command == "read_char":
            if len(sys.argv) < 4:
                print("Error: read_char requires characteristic UUID")
                sys.exit(1)
            char_uuid = sys.argv[3]
            result = await read_char_command(address, char_uuid)
            output_result(result)
        else:
            print(f"Unknown command: {command}")
            print("Commands: discover, read_battery, notify, connect, read_char <uuid>")
    except Exception as e:
        print(f"Error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

if __name__ == "__main__":
    asyncio.run(main())
