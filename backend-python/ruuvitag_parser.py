"""RuuviTag RAWv2 data format parser.

Decodes binary sensor data from RuuviTag devices using the RAWv2 format.
Reference: https://github.com/ruuvi/ruuvi-sensor-protocols
"""
import struct
from dataclasses import dataclass
from typing import Optional


@dataclass
class RuuviTagData:
    """Parsed RuuviTag sensor data."""
    
    temperature: float  # Celsius
    humidity: float  # Percentage
    pressure: float  # hPa
    acceleration_x: int  # mG
    acceleration_y: int  # mG
    acceleration_z: int  # mG
    battery: int  # mV
    tx_power: int  # dBm
    movement_counter: int
    measurement_sequence: int
    mac_address: Optional[str] = None


class RuuviTagParserError(Exception):
    """Raised when RuuviTag data parsing fails."""
    pass


def parse_rawv2(data: bytes) -> RuuviTagData:
    """Parse RuuviTag RAWv2 format data.
    
    Args:
        data: Binary data from RuuviTag sensor (24-26 bytes expected)
        
    Returns:
        RuuviTagData object with parsed sensor values
        
    Raises:
        RuuviTagParserError: If data is invalid or cannot be parsed
    """
    if len(data) < 24:
        raise RuuviTagParserError(f"Invalid data length: {len(data)} bytes (minimum 24 required)")
    
    try:
        # Format ID (should be 0x05 for RAWv2)
        format_id = data[0]
        if format_id != 0x05:
            raise RuuviTagParserError(f"Invalid format ID: 0x{format_id:02x} (expected 0x05)")
        
        # Temperature: int16, 0.005°C units
        temp_raw = struct.unpack(">h", data[1:3])[0]
        temperature = temp_raw * 0.005
        
        # Humidity: uint16, 0.0025% units
        humidity_raw = struct.unpack(">H", data[3:5])[0]
        humidity = humidity_raw * 0.0025
        
        # Pressure: uint16, 1 Pa units (offset from 50000 Pa)
        pressure_raw = struct.unpack(">H", data[5:7])[0]
        pressure = (pressure_raw + 50000) / 100.0  # Convert to hPa
        
        # Acceleration X, Y, Z: int16, mG units
        accel_x = struct.unpack(">h", data[7:9])[0]
        accel_y = struct.unpack(">h", data[9:11])[0]
        accel_z = struct.unpack(">h", data[11:13])[0]
        
        # Power info: uint16
        # - 11 MSB: battery voltage above 1.6V in mV
        # - 5 LSB: TX power in 2dBm steps
        power_info = struct.unpack(">H", data[13:15])[0]
        battery = 1600 + ((power_info >> 5) & 0x7FF)
        tx_power = (power_info & 0x1F) * 2 - 40
        
        # Movement counter: uint8
        movement_counter = data[15]
        
        # Measurement sequence: uint16
        measurement_sequence = struct.unpack(">H", data[16:18])[0]
        
        # MAC address (optional, 6 bytes)
        mac_address = None
        if len(data) >= 24:
            mac_bytes = data[18:24]
            mac_address = ":".join(f"{b:02x}" for b in mac_bytes)
        
        return RuuviTagData(
            temperature=temperature,
            humidity=humidity,
            pressure=pressure,
            acceleration_x=accel_x,
            acceleration_y=accel_y,
            acceleration_z=accel_z,
            battery=battery,
            tx_power=tx_power,
            movement_counter=movement_counter,
            measurement_sequence=measurement_sequence,
            mac_address=mac_address,
        )
        
    except (struct.error, IndexError) as e:
        raise RuuviTagParserError(f"Failed to parse RuuviTag data: {e}")
