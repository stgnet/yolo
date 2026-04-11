#!/usr/bin/env python3
"""
BLE scanner for macOS using CoreBluetooth framework via pyobjc.
Designed to find Victron SmartSolar MPPT controllers (like "Glow") and other BLE devices.
"""

import sys
import json
import time
from Foundation import NSRunLoop, NSDate
from CoreBluetooth import CBCentralManager, CBAdvertisementDataLocalNameKey


class BLEScannerDelegate:
    """Delegate class for handling BLE scanning callbacks."""
    
    def __init__(self):
        self.devices = set()
        self.device_details = []
        self.found_target = False
        self.bt_ready = False
        self.scanning_complete = False
    
    def centralManagerDidUpdateState_(self, manager):
        """Called when Bluetooth state changes."""
        new_state = manager.state()
        
        state_names = {
            0: "unknown",
            1: "powered on",
            2: "powered off", 
            3: "unauthorized",
            4: "unsupported",
            5: "resetting"
        }
        
        status_msg = state_names.get(new_state, 'unknown')
        print(f"Bluetooth state: {status_msg}")
        
        if new_state == 1:  # powered on - start scanning
            self.bt_ready = True
            print("Starting BLE scan...")
            manager.scanForPeripheralsWithServices_options_(None, None)
    
    def centralManager_didDiscoverPeripheral_advertisementData_RSSI_(self, manager, peripheral, adv_data, rssi):
        """Called when a new peripheral is discovered."""
        device_name = None
        
        if CBAdvertisementDataLocalNameKey in adv_data:
            device_name = str(adv_data[CBAdvertisementDataLocalNameKey])
        
        if not device_name and peripheral.name():
            device_name = str(peripheral.name())
            
        if not device_name:
            device_name = "Unknown"
        
        device_id = str(peripheral.identifier().UUIDString())
        rssi_value = int(rssi)
        
        # Create unique key to avoid duplicates
        device_key = (device_name, device_id)
        
        if device_key in self.devices:
            return
        
        self.devices.add(device_key)
        
        # Check for Victron/Glow devices
        is_victron = "glow" in device_name.lower() or "victron" in device_name.lower()
        
        marker = "🎯 VICTRON!" if is_victron else "  →"
        print(f"{marker} {device_name} (RSSI: {rssi_value} dBm)")
        
        self.device_details.append({
            "name": device_name,
            "id": device_id,
            "rssi": rssi_value,
            "is_victron": is_victron
        })
        
        if is_victron:
            self.found_target = True


def scan_for_ble_devices(duration=10):
    """Scan for BLE devices for the specified duration."""
    
    delegate = BLEScannerDelegate()
    
    print(f"{'='*60}")
    print(f"BLE Scanner - Looking for Victron/Glow devices")
    print(f"Duration: {duration} seconds")
    print(f"{'='*60}\n")
    
    print("Initializing Bluetooth...")
    
    # Create CBCentralManager with our delegate
    manager = CBCentralManager.alloc().initWithDelegate_queue_options_(delegate, None, {})
    
    # Wait for Bluetooth to initialize (up to 20 seconds)
    start_init = time.time()
    
    while not delegate.bt_ready and (time.time() - start_init) < 20:
        elapsed = time.time() - start_init
        
        # Show progress every 5 seconds
        if int(elapsed / 5) * 5 > 0 and int(elapsed) % 5 == 0:
            print(f"Waiting for Bluetooth... {int(elapsed)}s")
        
        # Process events
        NSRunLoop.currentRunLoop().runUntilDate_(NSDate.dateWithTimeIntervalSinceNow_(0.1))
    
    # Check if Bluetooth is ready
    bt_state = manager.state()
    state_names = {
        0: "unknown",
        1: "powered on",
        2: "powered off - please enable Bluetooth",
        3: "unauthorized", 
        4: "unsupported",
        5: "resetting"
    }
    
    if delegate.bt_ready:
        print("\nBluetooth ready, scanning...\n")
    else:
        print(f"\n✗ Cannot scan. Bluetooth state: {state_names.get(bt_state, 'unknown')} ({bt_state})")
        return None
    
    # Run the event loop for the specified duration
    start_time = time.time()
    
    while (time.time() - start_time) < duration:
        elapsed = time.time() - start_time
        
        # Show progress every 5 seconds
        if int(elapsed / 5) * 5 > 0 and int(elapsed) % 5 == 0:
            print(f"Scanning... {int(elapsed)}s found {len(delegate.device_details)} devices")
        
        NSRunLoop.currentRunLoop().runUntilDate_(NSDate.dateWithTimeIntervalSinceNow_(0.1))
    
    # Stop scanning
    manager.stopScan()
    
    device_details = delegate.device_details
    
    # Print summary
    print(f"\n{'='*60}")
    print("SCAN RESULTS")
    print(f"{'='*60}")
    print(f"Total devices found: {len(device_details)}")
    
    victron_devices = [d for d in device_details if d['is_victron']]
    print(f"Victron/Glow devices: {len(victron_devices)}")
    
    if victron_devices:
        print("\nFound Victron devices:")
        for dev in victron_devices:
            print(f"  ✓ {dev['name']} (RSSI: {dev['rssi']})")
    
    return {
        "scan_duration": duration,
        "total_devices": len(device_details),
        "victron_devices": len(victron_devices),
        "found_target": delegate.found_target,
        "devices": device_details,
        "victron_device_list": victron_devices
    }


def main():
    """Main entry point."""
    import argparse
    
    parser = argparse.ArgumentParser(
        description='Scan for BLE devices (looking for Victron/Glow)'
    )
    parser.add_argument(
        '-d', '--duration', 
        type=int, 
        default=10, 
        help='Scan duration in seconds (default: 10)'
    )
    
    args = parser.parse_args()
    
    if args.duration > 60:
        print("Warning: Duration limited to 60 seconds maximum")
        args.duration = 60
    
    # Run the scan
    results = scan_for_ble_devices(duration=args.duration)
    
    if results is None:
        sys.exit(1)
    
    # Output JSON summary
    print(f"\nJSON Summary:")
    print(json.dumps(results, indent=2))
    
    # Exit with error if no target found
    sys.exit(0 if results['found_target'] else 1)


if __name__ == '__main__':
    main()
