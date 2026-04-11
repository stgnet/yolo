#!/usr/bin/env python3
"""Debug script to see what bleak provides on macOS."""
import asyncio
from bleak import BleakScanner

async def main():
    print("Scanning for BLE devices with debug info...")
    
    async def detection_callback(device, adv_data):
        print(f"\nDevice detected:")
        print(f"  Device type: {type(device)}")
        print(f"  Device repr: {repr(device)[:200]}")
        print(f"  Has address: {hasattr(device, 'address')}")
        if hasattr(device, 'address'):
            print(f"  Address: {device.address}")
        print(f"  Has rssi: {hasattr(device, 'rssi')}")
        if hasattr(device, 'rssi'):
            print(f"  RSSI: {device.rssi}")
        print(f"  Adv data type: {type(adv_data)}")
        print(f"  Adv data repr: {repr(adv_data)[:500] if adv_data else None}")
        
        # Check attributes
        if device:
            attrs = [attr for attr in dir(device) if not attr.startswith('_')]
            print(f"  Available attrs: {attrs[:20]}")

    await BleakScanner.discover(timeout=3, detection_callback=detection_callback)

if __name__ == "__main__":
    asyncio.run(main())
