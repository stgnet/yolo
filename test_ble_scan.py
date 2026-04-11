#!/usr/bin/env python3
"""Test BLE scanning to find Victron Glow device."""
import asyncio
from bleak import BleakScanner

async def main():
    print("Scanning for BLE devices...")
    
    def callback(device, adv_data):
        print(f"\nFound: {device.name or 'unnamed'} (MAC: {device.address})")
        print(f"  RSSI: {adv_data.rssi}")
        if hasattr(adv_data, 'manufacturer_data') and adv_data.manufacturer_data:
            print(f"  Manufacturer Data:")
            for company_id, data in adv_data.manufacturer_data.items():
                print(f"    Company ID {company_id}: {data.hex()}")
        if hasattr(adv_data, 'service_data') and adv_data.service_data:
            print(f"  Service Data:")
            for uuid, data in adv_data.service_data.items():
                print(f"    UUID {uuid}: {data.hex()}")
    
    devices = await BleakScanner.discover(
        timeout=10.0,
        service_uuids=None,  # Scan all services
        detection_callback=callback
    )
    
    print(f"\n=== Scan complete. Found {len(devices)} device(s) ===")

if __name__ == "__main__":
    asyncio.run(main())
