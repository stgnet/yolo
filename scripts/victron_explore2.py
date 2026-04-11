#!/usr/bin/env python3
"""
Explore BLE services and characteristics of a specific device with more details.
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
                props_str = str(char.properties)
                
                print(f"  ├─ Characteristic: {char.uuid}")
                print(f"    │   Properties object: {props_str}")
                
                # Try to understand the properties better by looking at common patterns
                try:
                    # Print full characteristic info
                    print(f"    │   Full char: {dir(char)}")
                    
                    # Try to read anyway
                    data = await client.read_gatt_char(char.uuid)
                    hex_data = data.hex()
                    print(f"    │   Read value: {hex_data}")
                    
                    # Try various interpretations
                    try:
                        str_val = data.decode('utf-8', errors='ignore')
                        if str_val.strip():
                            print(f"    │   String: {str_val}")
                    except:
                        pass
                    
                    # Try as little-endian integer
                    try:
                        int_val = int.from_bytes(data, 'little')
                        float_val = int_val / 10.0  # Common scaling factor
                        print(f"    │   As number: {int_val} ({float_val})")
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
