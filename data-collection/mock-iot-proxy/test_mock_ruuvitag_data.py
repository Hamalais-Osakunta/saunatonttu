import unittest
from mock_ruuvitag_data import generate_mock_ruuvitag_data
import struct


class TestGenerateMockRuuvitagData(unittest.TestCase):

    def test_generate_with_random_values(self):
        data = generate_mock_ruuvitag_data()
        # Ensure the length of the data is correct
        self.assertEqual(len(data), 26)
        self.assertEqual(data[:2], b'\x99\x04')  # Check the manufacturer ID

    def test_generate_with_specific_values(self):
        specific_data = generate_mock_ruuvitag_data(
            temperature=24.3,
            humidity=53.49,
            pressure=100044,
            acc_x=0.004,
            acc_y=-0.004,
            acc_z=1.036,
            battery_voltage=2.977,
            tx_power=4,
            movement_counter=66,
            measurement_seq=205,
            mac_address=[0xCB, 0xB8, 0x33, 0x4C, 0x88, 0x4F]
        )
        expected_data = "99040512fc5394c37c0004fffc040cac364200cdcbb8334c884f"
        self.assertEqual(specific_data.hex(), expected_data)

    def test_generate_with_partial_values(self):
        data = generate_mock_ruuvitag_data(temperature=25.0)
        self.assertEqual(len(data), 26)
        self.assertEqual(data[:2], b'\x99\x04')
        temperature = struct.unpack(">h", data[3:5])[0] / 200
        self.assertAlmostEqual(temperature, 25.0, places=2)


if __name__ == '__main__':
    unittest.main()
