import unittest
from unittest.mock import MagicMock, patch

from fastapi.testclient import TestClient
from ruuvitag_insert import app, decode_ruuvitag_bytestream, push_data_to_redis

# TestClient is used to simulate requests to the FastAPI app
client = TestClient(app)


class TestDecodeRuuvitagBytestream(unittest.TestCase):

    def test_decode_valid_bytestream(self):
        # Valid bytestream for testing
        bytestream = bytes.fromhex(
            "99040512FC5394C37C0004FFFC040CAC364200CDCBB8334C884F")

        # Expected decoded data
        expected_data = {'acceleration': 1036.015443900331,
                         'acceleration_x': 4,
                         'acceleration_y': -4,
                         'acceleration_z': 1036,
                         'battery': 2977,
                         'data_format': 5,
                         'humidity': 53.49,
                         'mac': 'cbb8334c884f',
                         'measurement_sequence_number': 205,
                         'movement_counter': 66,
                         'pressure': 1000.44,
                         'rssi': None,
                         'temperature': 24.3,
                         'tx_power': 4}

        # Call the function
        decoded_data = decode_ruuvitag_bytestream(bytestream)

        # Assert the decoded data matches expected data
        self.assertEqual(decoded_data, expected_data)

    def test_decode_invalid_bytestream(self):
        # Invalid bytestream (wrong manufacturer ID)
        bytestream = bytes.fromhex(
            "12340512FC5394C37C0004FFFC040CAC364200CDCBB8334C884F")

        # Call the function and assert it raises a ValueError
        with self.assertRaises(ValueError):
            decode_ruuvitag_bytestream(bytestream)


class TestPushDataToRedis(unittest.TestCase):

    @patch('ruuvitag_insert.r')
    def test_push_data_to_redis(self, mock_redis):
        # Mock data to push
        mac = "AA:BB:CC:DD:EE:FF"
        measurement_data = {
            "temperature": 24.3,
            "humidity": 53.49,
            "battery": 2900
        }

        # Call the function
        push_data_to_redis(mac, measurement_data)

        # Check if Redis rpush was called with expected arguments
        self.assertTrue(mock_redis.rpush.called)

        # Extract the arguments with which Redis was called
        args, kwargs = mock_redis.rpush.call_args

        # Check that the key in Redis is correct
        self.assertEqual(args[0], "ruuvi_data_queue")

        # Ensure data string contains expected fields
        self.assertIn(mac, args[1])
        self.assertIn(str(measurement_data["temperature"]), args[1])
        self.assertIn(str(measurement_data["humidity"]), args[1])
        self.assertIn(str(measurement_data["battery"]), args[1])


if __name__ == '__main__':
    unittest.main()
