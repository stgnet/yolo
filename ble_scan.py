#!/usr/bin/env python3
"""Quick BLE scanner using bleak to find Victron/Glow devices."""
import asyncio
from bleak import BleakScanner, BleakClient

async def scan_without_adv():
    """First try simple discovery without adv data."""
    print("=" * 60)
    print("METHOD 1: Simple scan (no adv data)")
    print("=" * 60)
    
    devices = await BleakScanner.discover(timeout=5.0, return_adv=False)
    
    if not devices:
        print("No devices found!")
        return None
    
    print(f"\nFound {len(devices)} device(s):\n")
    
    glow_device = None
    for device in devices:
        name = getattr(device, 'name', None) or str(device)
        addr = getattr(device, 'address', None)
        
        # Check if this might be a Glow/Victron device
        name_str = str(name).lower()
        is_glow = 'glow' in name_str or 'victron' in name_str
        
        print(f"  Address: {addr}")
        print(f"  Name: {name}")
        
        if is_glow:
            print(f"  ** POTENTIAL GLOW/VICTRON DEVICE! **")
            glow_device = device
            
        print()
    
    return glow_device

async def connect_and_explore(device):
    """Connect to a device and get its services."""
    print("=" * 60)
    print(f"CONNECTING TO: {device}")
    print("=" * 60)
    
    try:
        async with BleakClient(device.address, timeout=10.0) as client:
            print(f"\nConnected! Address: {client.address}")
            
            # Get services
            services = client.services
            print(f"\nFound {len(services)} service(s):\n")
            
            for service in services:
                print(f"  Service UUID: {service.uuid}")
                
                # Check characteristics
                if hasattr(service, 'characteristics') and service.characteristics:
                    for char in service.characteristics:
                        props = []
                        if hasattr(char, 'properties'):
                            props = str(char.properties)
                        
                        print(f"    - Char: {char.uuid}")
                        print(f"      Properties: {props}")
                        
            # Try to read any characteristics that support reading
            print("\nAttempting to read characteristics...")
            for service in services:
                if hasattr(service, 'characteristics') and service.characteristics:
                    for char in service.characteristics:
                        if 'read' in str(char.properties).lower():
                            try:
                                data = await client.read_gatt_char(char.uuid)
                                print(f"    Read {char.uuid}: {data.hex()}")
                            except Exception as e:
                                print(f"    Read failed for {char.uuid}: {e}")
                                
    except Exception as e:
        print(f"Error connecting: {e}")

async def main():
    # First do a simple scan
    glow = await scan_without_adv()
    
    if glow:
        print("\nFound potential Glow device, exploring it...")
        await connect_and_explore(glow)
    else:
        print("\nNo Glow/Victron device found in initial scan.")
        
        # If nothing obvious, try scanning with full details and check manufacturer data
        print("\n" + "=" * 60)
        print("METHOD 2: Detailed scan checking manufacturer data")
        print("=" * 60)
        
        def callback(device, adv_data):
            """Callback for each advertisement."""
            addr = getattr(device, 'address', str(device))
            
            # Check manufacturer data
            if hasattr(adv_data, 'manufacturer_data') and adv_data.manufacturer_data:
                for comp_id, data in adv_data.manufacturer_data.items():
                    print(f"\nDevice with manufacturer data:")
                    print(f"  Address: {addr}")
                    print(f"  Company ID: {comp_id} (0x{comp_id:04x})")
                    print(f"  Data: {data.hex()}")
                    
                    # Victron uses company ID 0x6422 (25634)
                    if comp_id == 25634 or comp_id == 0x6422:
                        print("  ** VICTRON MANUFACTURER DETECTED! **")

        # Use notify callback approach
        scanner = BleakScanner(callback=callback)
        await scanner.start()
        
        # Wait a few seconds for advertisements
        import time
        time.sleep(5)
        
        await scanner.stop()

if __name__ == "__main__":
    asyncio.run(main())
