import unittest
from unittest.mock import MagicMock, patch
from fastapi.testclient import TestClient
from data_ingestion import app, push_data_to_redis, decode_ruuvitag_bytestream
import redis
from ruuvitag_sensor.decoder import UrlDecoder
import random
import os

# TestClient is used to simulate requests to the FastAPI app
client = TestClient(app)

# Mock function to simulate Ruuvitag binary data
def generate_mock_ruuvitag_bytestream():
    """
    Generate mock Ruuvitag bytestream to simulate incoming sensor data.
    """
    # Simulate random sensor values
    temperature = round(random.uniform(-10, 40), 2)
    humidity = round(random.uniform(0, 100), 2)
    battery = round(random.uniform(2.5, 3.7), 2)
    
    # Mock MAC address
    mac = "AA:BB:CC:DD:EE:FF"
    
    # Return data as hex-encoded string, simulating a Ruuvitag bytestream
    mock_data = f"990403{int(temperature * 100):04x}{int(humidity * 100):04x}{int(battery * 1000):04x}"
    return bytes.fromhex(mock_data)

class TestIngestEndpoint(unittest.TestCase):

    @patch('data_ingestion.r')  # Mock Redis connection in the main module
    def test_ingest_data(self, mock_redis):
        # Mock Redis rpush function to avoid real Redis interaction
        mock_redis.rpush = MagicMock()
        
        # Generate mock Ruuvitag bytestream
        bytestream = generate_mock_ruuvitag_bytestream()
        
        # Send POST request to /ingest/ endpoint with the mock bytestream
        response = client.post("/ingest/", data=bytestream)
        
        # Assert that the response is successful
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json()["status"], "success")
        
        # Check if Redis rpush was called with expected arguments
        self.assertTrue(mock_redis.rpush.called)
        
        # Extract the arguments with which Redis was called
        args, kwargs = mock_redis.rpush.call_args
        
        # Check that the key in Redis is correct
        self.assertEqual(args[0], "ruuvi_data_queue")
        
        # Ensure data string contains expected fields
        # This is checking that the mocked data was pushed to Redis properly
        self.assertIn("AA:BB:CC:DD:EE:FF", args[1])  # MAC Address
        self.assertIn("temperature", args[1])
        self.assertIn("humidity", args[1])
        self.assertIn("battery", args[1])

if __name__ == '__main__':
    unittest.main()
