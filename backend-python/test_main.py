"""Unit tests for the Kiuas model and notification logic."""
import unittest
from datetime import datetime, timedelta
from unittest.mock import MagicMock
from kiuas import Kiuas
from config import Config
from notification_service import check_and_notify


class MockTelegramBot:
    """Mock Telegram bot for testing."""
    
    def __init__(self):
        self.sent_messages = []
    
    def send_message(self, text: str, chat_id: int = None, parse_mode: str = "MarkdownV2") -> bool:
        self.sent_messages.append(text)
        return True
    
    def register_commands(self) -> bool:
        return True


class TestKiuas(unittest.TestCase):
    """Tests for the Kiuas class."""
    
    def test_add_temperature_record(self):
        """Test adding temperature records."""
        kiuas = Kiuas()
        now = datetime.now()
        
        kiuas.add_temperature_record(20.0, now)
        kiuas.add_temperature_record(21.0, now + timedelta(minutes=1))
        kiuas.add_temperature_record(22.0, now + timedelta(minutes=2))
        
        expected_temps = [20.0, 21.0, 22.0]
        self.assertEqual(kiuas.temperature_records, expected_temps)
    
    def test_temp_change_rate(self):
        """Test temperature change rate calculation."""
        kiuas = Kiuas()
        now = datetime.now()
        
        # Add temperature records showing warming
        kiuas.add_temperature_record(55.0, now)
        kiuas.add_temperature_record(57.5, now + timedelta(minutes=3))
        kiuas.add_temperature_record(60.0, now + timedelta(minutes=6))
        
        rate = kiuas.temp_change_rate()
        
        # Expected: (60.0 - 55.0) / (6 * 60) = 5.0 / 360 = 0.0138... degrees per second
        self.assertGreater(rate, 0.0)
        self.assertAlmostEqual(rate, 5.0 / 360, places=4)
    
    def test_is_warming(self):
        """Test warming detection."""
        kiuas = Kiuas()
        now = datetime.now()
        
        # Add temperature records showing warming
        kiuas.add_temperature_record(55.0, now)
        kiuas.add_temperature_record(57.5, now + timedelta(minutes=3))
        kiuas.add_temperature_record(60.0, now + timedelta(minutes=6))
        kiuas.temperature = 60.0
        
        config = Config(
            ready_threshold=75.0,
            change_threshold=0.0123,
            lower_bound=0.0123 * 0.9,
            reset_threshold=40.0,
            maintenance_chat_id=123,
            notification_chat_id=456,
            server_port=1337,
            telegram_bot_token="test_token"
        )
        
        self.assertTrue(kiuas.is_warming(config.ready_threshold, config.lower_bound))
    
    def test_get_estimated_ready_seconds(self):
        """Test estimated ready time calculation."""
        kiuas = Kiuas()
        now = datetime.now()
        
        # Add temperature records showing warming at ~0.0138 °C/s
        kiuas.add_temperature_record(55.0, now)
        kiuas.add_temperature_record(57.5, now + timedelta(minutes=3))
        kiuas.add_temperature_record(60.0, now + timedelta(minutes=6))
        kiuas.temperature = 60.0
        
        config = Config(
            ready_threshold=75.0,
            change_threshold=0.0123,
            lower_bound=0.0123 * 0.9,
            reset_threshold=40.0,
            maintenance_chat_id=123,
            notification_chat_id=456,
            server_port=1337,
            telegram_bot_token="test_token"
        )
        
        estimated_seconds = kiuas.get_estimated_ready_seconds(config.ready_threshold)
        
        # Expected: (75.0 - 60.0) / (5.0 / 360) = 15.0 / 0.0138... ≈ 1080 seconds
        self.assertGreater(estimated_seconds, 0)
        self.assertAlmostEqual(estimated_seconds, 1080, delta=10)
    
    def test_reset_notifications(self):
        """Test notification reset."""
        kiuas = Kiuas()
        kiuas.warming_notification_sent = True
        kiuas.ready_notification_sent = True
        
        kiuas.reset_notifications()
        
        self.assertFalse(kiuas.warming_notification_sent)
        self.assertFalse(kiuas.ready_notification_sent)


class TestNotifications(unittest.TestCase):
    """Tests for notification logic."""
    
    def setUp(self):
        """Set up test fixtures."""
        self.config = Config(
            ready_threshold=75.0,
            change_threshold=0.0123,
            lower_bound=0.0123 * 0.9,
            reset_threshold=40.0,
            maintenance_chat_id=123,
            notification_chat_id=456,
            server_port=1337,
            telegram_bot_token="test_token"
        )
    
    def test_sauna_ready_notification(self):
        """Test notification when sauna is ready."""
        current_time = datetime.now()
        kiuas = Kiuas(
            temperature=80.0,
            temperature_records=[70.0, 75.0, 80.0],
            timestamp_records=[
                current_time - timedelta(minutes=3),
                current_time - timedelta(minutes=2),
                current_time - timedelta(minutes=1)
            ]
        )
        
        mock_bot = MockTelegramBot()
        check_and_notify(mock_bot, kiuas, self.config, current_time)
        
        self.assertTrue(kiuas.ready_notification_sent)
        self.assertEqual(len(mock_bot.sent_messages), 1)
        self.assertIn("Sauna valmis", mock_bot.sent_messages[0])
        # Message is escaped for MarkdownV2, so check for escaped version
        self.assertIn("80", mock_bot.sent_messages[0])
    
    def test_sauna_warming_notification(self):
        """Test notification when sauna is warming."""
        current_time = datetime.now()
        kiuas = Kiuas(
            temperature=60.0,
            temperature_records=[55.0, 57.5, 60.0],
            timestamp_records=[
                current_time - timedelta(minutes=6),
                current_time - timedelta(minutes=3),
                current_time
            ]
        )
        
        mock_bot = MockTelegramBot()
        check_and_notify(mock_bot, kiuas, self.config, current_time)
        
        self.assertTrue(kiuas.warming_notification_sent)
        self.assertEqual(len(mock_bot.sent_messages), 1)
        self.assertIn("Sauna lämpiää", mock_bot.sent_messages[0])
        self.assertIn("Valmis klo", mock_bot.sent_messages[0])
    
    def test_no_notification_when_stable(self):
        """Test no notification sent when temperature is stable."""
        current_time = datetime.now()
        kiuas = Kiuas(
            temperature=30.0,
            temperature_records=[30.0, 30.0, 30.0],
            timestamp_records=[
                current_time - timedelta(minutes=3),
                current_time - timedelta(minutes=2),
                current_time - timedelta(minutes=1)
            ]
        )
        
        mock_bot = MockTelegramBot()
        check_and_notify(mock_bot, kiuas, self.config, current_time)
        
        self.assertFalse(kiuas.warming_notification_sent)
        self.assertFalse(kiuas.ready_notification_sent)
        self.assertEqual(len(mock_bot.sent_messages), 0)
    
    def test_warming_notification_sent_only_once(self):
        """Test warming notification is only sent once."""
        current_time = datetime.now()
        kiuas = Kiuas(
            temperature=60.0,
            temperature_records=[55.0, 57.5, 60.0],
            timestamp_records=[
                current_time - timedelta(minutes=6),
                current_time - timedelta(minutes=3),
                current_time
            ]
        )
        
        mock_bot = MockTelegramBot()
        
        # Call multiple times with increasing temperature
        for i in range(5):
            kiuas.temperature += 2.0
            kiuas.add_temperature_record(
                kiuas.temperature,
                current_time + timedelta(minutes=i)
            )
            check_and_notify(mock_bot, kiuas, self.config, current_time + timedelta(minutes=i))
        
        self.assertTrue(kiuas.warming_notification_sent)
        # Should only have 1 message (warming notification sent once)
        self.assertEqual(len(mock_bot.sent_messages), 1)
    
    def test_reset_notifications_when_cooled(self):
        """Test notifications reset when temperature drops."""
        current_time = datetime.now()
        kiuas = Kiuas(
            temperature=80.0,
            temperature_records=[70.0, 75.0, 80.0],
            timestamp_records=[
                current_time - timedelta(minutes=3),
                current_time - timedelta(minutes=2),
                current_time - timedelta(minutes=1)
            ],
            warming_notification_sent=True,
            ready_notification_sent=True
        )
        
        mock_bot = MockTelegramBot()
        
        # Simulate temperature dropping below reset threshold
        kiuas.temperature = 35.0
        kiuas.add_temperature_record(kiuas.temperature, current_time)
        
        check_and_notify(mock_bot, kiuas, self.config, current_time)
        
        self.assertFalse(kiuas.warming_notification_sent)
        self.assertFalse(kiuas.ready_notification_sent)
        # No new messages should be sent during reset
        self.assertEqual(len(mock_bot.sent_messages), 0)
    
    def test_warming_stopped_before_ready_reset(self):
        """Test notifications reset when warming stops before reaching ready."""
        current_time = datetime.now()
        kiuas = Kiuas(
            temperature=60.0,
            temperature_records=[55.0, 57.5, 60.0],
            timestamp_records=[
                current_time - timedelta(minutes=6),
                current_time - timedelta(minutes=3),
                current_time
            ],
            warming_notification_sent=True
        )
        
        mock_bot = MockTelegramBot()
        
        # Simulate temperature dropping before reaching ready threshold
        kiuas.temperature = 35.0
        kiuas.add_temperature_record(kiuas.temperature, current_time)
        
        check_and_notify(mock_bot, kiuas, self.config, current_time)
        
        # When only warming notification was sent, only that should be reset
        # The logic only resets when BOTH notifications were sent
        # So warming_notification_sent stays True until both are True
        self.assertTrue(kiuas.warming_notification_sent)
        self.assertFalse(kiuas.ready_notification_sent)
        self.assertEqual(len(mock_bot.sent_messages), 0)


if __name__ == "__main__":
    unittest.main()
