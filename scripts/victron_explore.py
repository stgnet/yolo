#!/usr/bin/env python3
"""
Explore BLE services and characteristics of a specific device.
Usage: python3 scripts/victron_explore.py [device_address]
"""
import asyncio
import sys
from bleak import BleakClient

async def explore_device(address):
    """Connect to device and list all services and characteristics."""
    print(f"Connecting to {address}...")
    
    async with BleakClient(address, timeout=10.0) as client:
        print(f"\n✓ Connected to {address}")
        
        # Get services after connection (it's a property, not a coroutine)
        services = list(client.services)
        
        if not services:
            print("No services found")
            return
            
        print(f"Found {len(services)} service(s):\n")
        
        for service in services:
            print(f"Service: {service.uuid}")
            
            # Get characteristics from the service
            characteristics = list(service.characteristics)
            for char in characteristics:
                props = []
                if hasattr(char.properties, 'read') and char.properties.read:
                    props.append("read")
                if hasattr(char.properties, 'write') and char.properties.write:
                    props.append("write")
                if hasattr(char.properties, 'write_without_response') and char.properties.write_without_response:
                    props.append("write_no_response")
                if hasattr(char.properties, 'notify') and char.properties.notify:
                    props.append("notify")
                if hasattr(char.properties, 'indicate') and char.properties.indicate:
                    props.append("indicate")
                    
                props_str = ", ".join(props) if props else "none"
                
                print(f"  ├─ Characteristic: {char.uuid}")
                print(f"    │   Properties: {props_str}")
                
                # Try to read if readable
                if "read" in props:
                    try:
                        data = await client.read_gatt_char(char.uuid)
                        hex_data = data.hex()
                        print(f"    │   Read value: {hex_data}")
                        
                        # Try to interpret as string or number
                        try:
                            str_val = data.decode('utf-8', errors='ignore')
                            if str_val.strip():
                                print(f"    │   String: {str_val}")
                        except:
                            pass
                    except Exception as e:
                        print(f"    │   Read error: {e}")

async def main():
    if len(sys.argv) < 2:
        # Default to the Glow device
        target_address = "CB60561D-CF87-0055-E7B1-009FFA19F942"
        print(f"No device address specified. Using Glow device: {target_address}")
        print("To scan for devices, run: python3 scripts/victron_scan.py 10")
    else:
        target_address = sys.argv[1]
    
    await explore_device(target_address)

if __name__ == "__main__":
    asyncio.run(main())
