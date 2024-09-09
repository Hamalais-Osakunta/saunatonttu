import time
import random
import requests
import os
from mock_ruuvitag_data import generate_mock_ruuvitag_data

def generate_sauna_data(is_on, prev_temperature, prev_humidity):
    """
    Generates realistic sauna data, adjusting temperature and humidity based on whether the sauna is on or off.

    Args:
        is_on (bool): True if the sauna is on, False if the sauna is off.
        prev_temperature (float): The previous temperature to simulate gradual change.
        prev_humidity (float): The previous humidity to simulate gradual change.

    Returns:
        tuple: (new_temperature, new_humidity)
    """
    if is_on:
        # Sauna is on: Increase temperature and humidity
        new_temperature = min(prev_temperature + random.uniform(0.5, 2.0), 90.0)  # Max ~90°C in sauna
        new_humidity = min(prev_humidity + random.uniform(0.1, 1.0), 40.0)  # Max ~40% humidity (depending on steam)
    else:
        # Sauna is off: Gradually cool down and lower humidity
        new_temperature = max(prev_temperature - random.uniform(0.5, 1.5), 20.0)  # Min ~20°C (room temp)
        new_humidity = max(prev_humidity - random.uniform(0.1, 0.5), 10.0)  # Min ~10% humidity

    return new_temperature, new_humidity

def send_mock_sauna_data(interval=10):
    """
    This service generates mock sauna sensor data and sends it to the /ingest endpoint.
    It simulates the sauna being turned on and off, with realistic temperature and humidity changes.

    Args:
        interval (int): The interval between sending data, in seconds. Default is 10 seconds.
    """
    
    # Get the URL from environment variable or use default
    url = os.getenv("INGEST_URL", "http://localhost") + "/ingest/"

    # Initial conditions
    temperature = 20.0  # Room temperature in °C
    humidity = 20.0  # Room humidity in %
    is_sauna_on = False  # Initially, the sauna is off
    toggle_counter = 0  # Counter to toggle the sauna state periodically

    while True:
        try:
            # Toggle sauna state every 10 cycles (~100 seconds if interval is 10 seconds)
            toggle_counter += 1
            if toggle_counter >= 10:
                is_sauna_on = not is_sauna_on
                toggle_counter = 0
                state = "on" if is_sauna_on else "off"
                print(f"Sauna turned {state}")

            # Generate new temperature and humidity based on whether the sauna is on or off
            temperature, humidity = generate_sauna_data(is_sauna_on, temperature, humidity)

            # Other sensor data remains relatively constant
            battery_voltage = 3.0  # Battery voltage is constant
            tx_power = 0  # TX power remains constant
            mac_address = [0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF]  # Example MAC address
            
            print(f"Temperature: {temperature}°C, Humidity: {humidity}%, Sauna is {'on' if is_sauna_on else 'off'}")

            # Generate mock Ruuvitag data for the sauna
            mock_data = generate_mock_ruuvitag_data(
                temperature=temperature,
                humidity=humidity,
                pressure=101325,  # Constant pressure
                battery_voltage=battery_voltage,
                tx_power=tx_power,
                mac_address=mac_address
            )

            # Send the generated data to the server as a POST request
            response = requests.post(url, data=mock_data)
            print(f"Data sent: {mock_data.hex()}")

            # Log the server's response
            if response.status_code == 200:
                print(f"Data sent successfully: {response.json()}")
            else:
                print(f"Failed to send data: {response.status_code} - {response.text}")

        except Exception as e:
            print(f"Error occurred: {e}")

        # Wait for the specified interval before sending the next batch of data
        time.sleep(interval)

# Start the service (sending data every 10 seconds by default)
if __name__ == "__main__":
    send_mock_sauna_data(interval=10)
