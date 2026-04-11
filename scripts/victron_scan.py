#!/usr/bin/env python3
"""
BLE scanner using bleak to find Victron/Glow devices.
Outputs JSON array of devices to stdout.
Usage: python3 scripts/victron_scan.py [duration_seconds]
"""
import asyncio
import json
import sys
from bleak import BleakScanner

async def main():
    duration = int(sys.argv[1]) if len(sys.argv) > 1 else 10
    
    # Use simple discovery - returns device objects with address and name
    devices_list = await BleakScanner.discover(timeout=duration, return_adv=False)
    
    results = []
    for device in devices_list:
        addr = getattr(device, 'address', None) or str(device)
        name = getattr(device, 'name', '') or ''
        
        # Get additional info by connecting briefly (optional, may not work without pairing)
        # For now just return basic scan data
        
        results.append({
            'address': addr,
            'name': name,
            'rssi': -80  # RSSI requires adv data which we're not collecting here
        })
    
    # Output as JSON array
    print(json.dumps(results, indent=2))

if __name__ == "__main__":
    asyncio.run(main())
