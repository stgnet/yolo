#!/usr/bin/env python3
"""Test script to see what bleak discovers."""
import asyncio
from bleak import BleakScanner

async def main():
    print("Scanning for BLE devices...")
    devices = await BleakScanner.discover(timeout=5, return_adv=True)
    print(f'\nFound {len(devices)} devices:')
    
    for i, device in enumerate(list(devices)[:5]):
        print(f"\nDevice {i+1}:")
        print(f"  Type: {type(device)}")
        print(f"  Address: {getattr(device, 'address', None)}")
        print(f"  Name: {getattr(device, 'name', None)}")
        print(f"  Has adv_data: {hasattr(device, 'adv_data')}")
        if hasattr(device, 'adv_data') and device.adv_data:
            print(f"  Adv data type: {type(device.adv_data)}")
            print(f"  Adv data: {device.adv_data}")

if __name__ == "__main__":
    asyncio.run(main())
