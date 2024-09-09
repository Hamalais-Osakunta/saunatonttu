import random
import struct

def generate_mock_ruuvitag_data(
    temperature=None,    # Temperature in degrees Celsius
    humidity=None,       # Humidity in percentage (0-100%)
    pressure=None,       # Pressure in Pascals
    acc_x=None,          # Acceleration-X in G units
    acc_y=None,          # Acceleration-Y in G units
    acc_z=None,          # Acceleration-Z in G units
    battery_voltage=None,# Battery voltage in volts (1.6V to 3.646V)
    tx_power=None,       # TX power in dBm (-40 to +20 dBm)
    movement_counter=None, # Movement counter (0-254)
    measurement_seq=None,  # Measurement sequence number (0-65534)
    mac_address=None     # MAC address as a list of 6 bytes
):
    """
    Generates mock Ruuvitag data according to the specified Bluetooth Manufacturer Specific Data format.
    
    The returned data will be in the format expected from the 0x9904 Manufacturer ID.
    All values can either be passed explicitly or randomly generated if None.
    Real-world units should be provided.
    """
    
    # Manufacturer ID: 0x9904
    manufacturer_id = 0x9904
    
    # Data format (8-bit): Only '5' is allowed as per spec.
    data_format = 5
    
    # Temperature (16-bit signed): Real temperature in degrees Celsius, steps of 0.005°C
    if temperature is None:
        temperature = random.uniform(-163.835, 163.835)
    encoded_temperature = int(temperature * 200)  # Convert to 0.005°C steps
    
    # Humidity (16-bit unsigned): Real humidity in percentage (0-100%), steps of 0.0025%
    if humidity is None:
        humidity = random.uniform(0, 100)
    encoded_humidity = int(humidity / 0.0025)  # Convert to 0.0025% steps
    
    # Pressure (16-bit unsigned): Real pressure in Pascals, offset by -50000 Pa
    if pressure is None:
        pressure = random.uniform(30000, 110000)  # Realistic pressure range (Pa)
    encoded_pressure = int(pressure - 50000)  # Convert to format with -50000 Pa offset
    
    # Acceleration-X, Y, Z (16-bit signed): Real acceleration in G, convert to milli-G
    if acc_x is None:
        acc_x = random.uniform(-32.767, 32.767)  # Range in Gs
    if acc_y is None:
        acc_y = random.uniform(-32.767, 32.767)
    if acc_z is None:
        acc_z = random.uniform(-32.767, 32.767)
    encoded_acc_x = int(acc_x * 1000)  # Convert G to milli-G
    encoded_acc_y = int(acc_y * 1000)
    encoded_acc_z = int(acc_z * 1000)
    
    # Power info (11+5 bits): Battery voltage in volts (1.6V to 3.646V) and TX power (-40 to +20 dBm)
    if battery_voltage is None:
        battery_voltage = random.uniform(1.6, 3.646)  # Voltage in volts
    if tx_power is None:
        tx_power = random.choice([-40, -38, -36, -34, -32, -30, -28, -26, -24, -22, -20, -18, -16, -14, -12, -10, -8, -6, -4, -2, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20])
    
    encoded_battery_voltage = int(battery_voltage * 1000 - 1600)  # Align voltage encoding
    encoded_tx_power = (tx_power + 40) // 2  # Convert TX power from dBm to the encoded 5-bit value
    power_info = (encoded_battery_voltage << 5) | encoded_tx_power
    
    # Movement counter (8-bit unsigned): Random number between 0 and 254
    if movement_counter is None:
        movement_counter = random.randint(0, 254)
    
    # Measurement sequence number (16-bit unsigned): Random number between 0 and 65534
    if measurement_seq is None:
        measurement_seq = random.randint(0, 65534)
    
    # MAC Address (48-bit): Random MAC address (in bytes)
    if mac_address is None:
        mac_address = [random.randint(0, 255) for _ in range(6)]

    # Pack data into a byte array, using struct.pack for correct byte representation
    # ">H" means big-endian unsigned short (2 bytes)
    # ">h" means big-endian signed short (2 bytes)
    # ">B" means unsigned char (1 byte)
    # ">6B" means 6 unsigned chars (6 bytes for the MAC address)
    
    data = struct.pack(
        ">B h H H h h h H B H 6B", 
        data_format,       # 1 byte
        encoded_temperature,  # 2 bytes
        encoded_humidity,   # 2 bytes
        encoded_pressure,   # 2 bytes
        encoded_acc_x,      # 2 bytes
        encoded_acc_y,      # 2 bytes
        encoded_acc_z,      # 2 bytes
        power_info,         # 2 bytes (battery voltage + tx power)
        movement_counter,   # 1 byte
        measurement_seq,    # 2 bytes
        *mac_address        # 6 bytes
    )
    
    #Prepend the manufacturer ID (0x9904)
    manufacturer_specific_data = struct.pack(">H", manufacturer_id) + data
    
    return manufacturer_specific_data

# Example usage:

def run_example():
    # Using random values
    random_data = generate_mock_ruuvitag_data()
    print(f"Random Data: {random_data.hex()}")

    # Using specific real-world values
    specific_data = generate_mock_ruuvitag_data(
        temperature=24.3,      # 25°C
        humidity=53.49,         # 50%
        pressure=100044,       # 101325 Pa
        acc_x=0.004,             # 1 G
        acc_y=-0.004,            # -1 G
        acc_z=1.036,             # 0.5 G
        battery_voltage=2.977,   # 3.0 V
        tx_power=4,           # 10 dBm
        movement_counter=66,  # Movement counter 100
        measurement_seq=205,  # Measurement sequence 5000
        mac_address=[0xCB, 0xB8, 0x33, 0x4C, 0x88, 0x4F]  # MAC address
    )
    from ruuvitag_sensor.decoder import get_decoder
    real_data = "0512FC5394C37C0004FFFC040CAC364200CDCBB8334C884F"
    print(real_data == specific_data.hex())
    sensor_data = get_decoder(5).decode_data(specific_data[2:26].hex())
    print(sensor_data)
    print(f"Specific Data: {specific_data.hex()}")
