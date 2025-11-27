"""Notification service for sauna state changes."""
import logging
from datetime import datetime
from typing import Optional
from kiuas import Kiuas
from config import Config
from telegram_bot import TelegramBotInterface, fmt_telegram

logger = logging.getLogger(__name__)


def check_and_notify(
    bot: TelegramBotInterface,
    kiuas: Kiuas,
    config: Config,
    current_time: Optional[datetime] = None
) -> None:
    """Check temperature changes and send notifications.
    
    Args:
        bot: Telegram bot interface for sending messages
        kiuas: Kiuas state object
        config: Application configuration
        current_time: Current time (defaults to now if not provided)
    """
    if current_time is None:
        current_time = datetime.now()
    
    # Ready notification check
    if kiuas.temperature >= config.ready_threshold:
        if not kiuas.ready_notification_sent:
            message = fmt_telegram(f"*Sauna valmis!*🔥\nLämpötila: {kiuas.temperature:.1f} °C 🌡️")
            bot.send_message(message, chat_id=config.notification_chat_id)
            kiuas.ready_notification_sent = True
            logger.info(f"Sent ready notification: {kiuas.temperature:.1f}°C")
    
    # Warming notification check
    elif not kiuas.warming_notification_sent and not kiuas.ready_notification_sent:
        if kiuas.is_warming(config.ready_threshold, config.lower_bound):
            time_to_ready_seconds = kiuas.get_estimated_ready_seconds(config.ready_threshold)
            estimated_ready_time = current_time + timedelta(seconds=time_to_ready_seconds)
            estimated_ready_time_str = estimated_ready_time.strftime("%H:%M")
            
            logger.info(f"Estimated ready time: {estimated_ready_time}")
            logger.info(f"Current time: {current_time}")
            logger.info(f"Estimated ready time string: {estimated_ready_time_str}")
            
            message = fmt_telegram(f"🔥*Sauna lämpiää!*🔥\nValmis klo {estimated_ready_time_str}")
            bot.send_message(message, chat_id=config.notification_chat_id)
            kiuas.warming_notification_sent = True
            logger.info(f"Sent warming notification: ready at {estimated_ready_time_str}")
    
    # Reset notifications if temperature has cooled down
    if kiuas.temperature < config.reset_threshold:
        if kiuas.warming_notification_sent and kiuas.ready_notification_sent:
            kiuas.reset_notifications()
            logger.info("Notifications reset due to temperature drop")


from datetime import timedelta  # Add this import at the top
