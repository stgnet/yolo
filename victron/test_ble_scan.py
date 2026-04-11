#!/usr/bin/env python3
"""Test BLE scanning with bleak - quick version."""

import asyncio
from bleak import BleakScanner

async def quick_scan():
    """Quick scan for any BLE devices."""
    print("Starting quick BLE scan (5 seconds)...")
    
    devices = await BleakScanner.discover(timeout=5)
    
    print(f"\nFound {len(devices)} device(s):")
    for device in devices[:10]:  # Show first 10
        print(f"  - {device.address}: {device.name or 'Unknown'} (RSSI: ?)")
    
    return devices

if __name__ == "__main__":
    devices = asyncio.run(quick_scan())
    if devices:
        print("\nBleak is working! Devices detected.")
    else:
        print("\nNo devices found - BLE may not be available or no devices nearby.")
