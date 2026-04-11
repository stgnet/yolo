#!/usr/bin/env python3
"""
Decode Victron Glow battery data from VE.Direct format.
The Glow device returns binary data that needs parsing.
"""
import struct

def decode_glow_data(data_bytes):
    """Decode Glow battery monitoring data."""
    data = bytes(data_bytes) if isinstance(data_bytes, list) else data_bytes
    
    if len(data) < 20:
        return {'error': f'Invalid data length: {len(data)}'}
    
    # The data appears to be in a VE.Direct-like format
    # Common Victron values:
    # - VV (Battery voltage): typically at offset with 2-byte little-endian
    # - CS (State of charge): percentage
    
    result = {}
    
    # Try different offsets for voltage (little-endian uint16, scaled by 10 or 100)
    # Based on the hex: ff180100ff5202008c0091a15000f2ffffff0000
    
    # Byte 4-5: 0xff 0x18 = could be flags/version
    # Byte 8-9: 0x8c 0x00 = 140 (decimal) - likely voltage * 10 or * 100
    # Byte 10-11: 0x91 0xa1 = might be other measurements
    
    # Try offset 8 for voltage (most common in VE.Direct)
    if len(data) >= 10:
        voltage_raw = struct.unpack('<H', data[8:10])[0]  # Little-endian uint16
        result['voltage_10x'] = voltage_raw / 10.0  # Try /10 first
        result['voltage_100x'] = voltage_raw / 100.0  # Try /100
        
        # Also try big-endian
        voltage_be = struct.unpack('>H', data[8:10])[0]
        result['voltage_be_10x'] = voltage_be / 10.0
    
    # Check offset 2 (another common position)
    if len(data) >= 4:
        voltage_raw2 = struct.unpack('<H', data[2:4])[0]
        result['voltage_offset2_10x'] = voltage_raw2 / 10.0
    
    # The full hex dump for reference
    result['raw_hex'] = data.hex()
    result['byte_count'] = len(data)
    
    return result

def main():
    # Test with the data we got from Glow
    test_data = bytes.fromhex('ff180100ff5202008c0091a15000f2ffffff0000')
    
    decoded = decode_glow_data(test_data)
    
    print("Decoded Glow battery data:")
    for key, value in decoded.items():
        if isinstance(value, float):
            print(f"  {key}: {value:.2f}")
        else:
            print(f"  {key}: {value}")

if __name__ == "__main__":
    main()
