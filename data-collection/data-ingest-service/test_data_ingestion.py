import unittest
from unittest.mock import MagicMock
from data_ingestion import ingest_data

class TestIngestData(unittest.TestCase):
    def test_ingest_data(self):
        # Mock Redis connection
        redis_conn = MagicMock()

        # Mock TimescaleDB connection
        db_conn = MagicMock()
        db_cursor = MagicMock()
        db_conn.cursor.return_value = db_cursor

        # Mock data from Redis queue
        data = ("ruuvi_data_queue", "sensor_mac,25.5,50.0,3.7,2022-01-01 00:00:00")
        redis_conn.blpop.return_value = data

        # Override time.sleep in the test to avoid delay
        with unittest.mock.patch('time.sleep', return_value=None):
            # Limit to 1 iteration for the test
            ingest_data(redis_conn, db_conn, max_iterations=1)
            
        # Check that the data was inserted into the database
        db_cursor.execute.assert_called_with(
            "INSERT INTO sauna_data (sensor_mac, temperature, humidity, battery, timestamp) VALUES (%s, %s, %s, %s, %s)",
            ("sensor_mac", 25.5, 50.0, 3.7, "2022-01-01 00:00:00")
        )
        
        db_conn.commit.assert_called_once()
        
        
    def test_ingest_data_no_data(self):
        # Mock Redis connection
        redis_conn = MagicMock()

        # Mock TimescaleDB connection
        db_conn = MagicMock()
        db_cursor = MagicMock()
        db_conn.cursor.return_value = db_cursor

        # Mock no data from Redis queue
        redis_conn.blpop.return_value = None

        # Override time.sleep in the test to avoid delay
        with unittest.mock.patch('time.sleep', return_value=None):
            # Limit to 1 iteration for the test
            ingest_data(redis_conn, db_conn, max_iterations=1)
            
        # Check that no data was inserted into the database
        db_cursor.execute.assert_not_called()
        db_conn.commit.assert_not_called()

# Run the test
if __name__ == '__main__':
    unittest.main()
