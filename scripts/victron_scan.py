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
    
    results = []
    
    # Detection callback to capture devices as they're discovered
    async def detection_callback(device, adv_data):
        addr = str(device.address) if hasattr(device, 'address') and device.address else None
        
        # Extract name from advertisement data or device object
        name = ''
        if adv_data and hasattr(adv_data, 'local_name'):
            name = adv_data.local_name or ''
        
        # Fallback: try device.name attribute
        if not name and hasattr(device, 'name') and device.name:
            name = str(device.name)
        
        # Get RSSI from advertisement data if available  
        rssi = -80  # Default fallback
        if adv_data and hasattr(adv_data, 'rssi'):
            rssi = adv_data.rssi
        
        results.append({
            'address': addr,
            'name': name,
            'rssi': rssi
        })

    
    # Scan using detection callback which works better on macOS
    await BleakScanner.discover(timeout=duration, detection_callback=detection_callback)
    
    # Output as JSON array
    print(json.dumps(results, indent=2))

if __name__ == "__main__":
    asyncio.run(main())
