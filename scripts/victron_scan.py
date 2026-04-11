#!/usr/bin/env python3
"""
BLE Scanner for Victron and Generic Devices
Scans for all nearby BLE devices, not just Victron-specific ones.
Uses bleak library which provides cross-platform CoreBluetooth support on macOS.
"""

import json
import sys
import asyncio
from bleak import BleakScanner

async def scan_for_devices(duration: int = 10):
    """
    Scan for BLE devices for the specified duration (seconds).
    Returns all discovered devices, not just Victron ones.
    """
    devices = []
    
    async def discovery_handler(device, advertisement_data):
        """Callback when a device is discovered."""
        device_info = {
            "address": device.address,
            "name": device.name or "",
            "rssi": advertisement_data.rssi if hasattr(advertisement_data, 'rssi') else -99,
            "is_victron": check_if_victron(device.name or ""),
        }
        devices.append(device_info)
    
    # Scan for ALL BLE devices (not filtered by service UUID)
    print(f"Scanning for BLE devices for {duration} seconds...", file=sys.stderr)
    
    try:
        await BleakScanner.discover(
            timeout=duration,
            service_uuids=None,  # None = discover all devices, not just those advertising specific services
            detection_callback=discovery_handler,
            scanning_mode="active"  # Active scanning for better discovery
        )
    except Exception as e:
        print(f"Error during scan: {e}", file=sys.stderr)
    
    # Output results as JSON
    result = {
        "devices": devices,
        "count": len(devices),
        "victron_count": sum(1 for d in devices if d["is_victron"])
    }
    
    print(json.dumps(result, indent=2))

def check_if_victron(name: str) -> bool:
    """Check if device name suggests it's a Victron device."""
    if not name:
        return False
    
    victron_patterns = [
        "VE.Direct",
        "SmartSolar", 
        "SmartShunt",
        "Victron",
        "BMV",
        "Phoenix",
        "BlueSolar"
    ]
    
    name_lower = name.lower()
    for pattern in victron_patterns:
        if pattern.lower() in name_lower:
            return True
    
    return False

def main():
    """Main entry point."""
    duration = 10  # Default scan duration
    
    # Parse command line argument for duration
    if len(sys.argv) > 1:
        try:
            duration = int(sys.argv[1])
        except ValueError:
            print(f"Invalid duration: {sys.argv[1]}", file=sys.stderr)
            sys.exit(1)
    
    # Ensure duration is reasonable
    duration = max(1, min(duration, 60))
    
    asyncio.run(scan_for_devices(duration))

if __name__ == "__main__":
    main()
