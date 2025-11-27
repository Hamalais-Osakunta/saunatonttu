"""Main application entry point for sauna monitoring service."""
import logging
import os
import signal
import sys
import threading
import time
from datetime import datetime, timedelta
from http.server import HTTPServer, BaseHTTPRequestHandler
from typing import Optional

from config import Config
from kiuas import Kiuas
from ruuvitag_parser import parse_rawv2, RuuviTagParserError
from telegram_bot import TelegramBot, fmt_telegram
from notification_service import check_and_notify

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    handlers=[logging.StreamHandler(sys.stdout)]
)
logger = logging.getLogger(__name__)


class SaunaHTTPHandler(BaseHTTPRequestHandler):
    """HTTP request handler for sauna data reception."""
    
    # Class variables to share state across requests
    kiuas: Optional[Kiuas] = None
    bot: Optional[TelegramBot] = None
    config: Optional[Config] = None
    
    def log_message(self, format, *args):
        """Override to use our logger."""
        logger.info(f"{self.address_string()} - {format % args}")
    
    def do_POST(self):
        """Handle POST requests to /api/receive-bt endpoint."""
        if self.path != "/api/receive-bt":
            self.send_error(404, "Not Found")
            return
        
        try:
            # Read request body
            content_length = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(content_length)
            
            # Parse RuuviTag data
            try:
                ruuvi_data = parse_rawv2(body)
            except RuuviTagParserError as e:
                logger.error(f"Failed to parse RuuviTag data: {e}")
                self.send_error(400, f"Invalid RuuviTag data: {e}")
                return
            
            # Update kiuas state
            self.kiuas.temperature = ruuvi_data.temperature
            self.kiuas.humidity = ruuvi_data.humidity
            self.kiuas.battery = ruuvi_data.battery
            self.kiuas.last_data_received = datetime.now()
            
            logger.info(
                f"Received new temperature value: {self.kiuas.temperature:.1f} °C, "
                f"Humidity: {self.kiuas.humidity:.1f}%, "
                f"Voltage: {self.kiuas.battery} mV"
            )
            
            # Add temperature record
            self.kiuas.add_temperature_record(self.kiuas.temperature, datetime.now())
            
            # Check and send notifications
            check_and_notify(self.bot, self.kiuas, self.config)
            
            # Send success response
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(b'{"status": "ok"}')
            
        except Exception as e:
            logger.error(f"Error handling request: {e}", exc_info=True)
            self.send_error(500, f"Internal server error: {e}")


def start_http_server(kiuas: Kiuas, bot: TelegramBot, config: Config) -> HTTPServer:
    """Start the HTTP server for receiving data.
    
    Args:
        kiuas: Kiuas state object
        bot: Telegram bot instance
        config: Application configuration
        
    Returns:
        Running HTTPServer instance
    """
    # Set class variables for handler
    SaunaHTTPHandler.kiuas = kiuas
    SaunaHTTPHandler.bot = bot
    SaunaHTTPHandler.config = config
    
    server = HTTPServer(("0.0.0.0", config.server_port), SaunaHTTPHandler)
    logger.info(f"HTTP server listening on port {config.server_port}")
    
    # Run server in a separate thread
    server_thread = threading.Thread(target=server.serve_forever, daemon=True)
    server_thread.start()
    
    return server


def monitor_data_reception(kiuas: Kiuas, bot: TelegramBot, config: Config, stop_event: threading.Event) -> None:
    """Monitor data reception and alert if no data received for too long.
    
    Args:
        kiuas: Kiuas state object
        bot: Telegram bot instance
        config: Application configuration
        stop_event: Threading event to signal shutdown
    """
    notification_sent = False
    
    while not stop_event.is_set():
        time_since_last_data = datetime.now() - kiuas.last_data_received
        
        if time_since_last_data > timedelta(hours=1) and not notification_sent:
            message = "No data received for over 1 hour"
            bot.send_message(message, chat_id=config.maintenance_chat_id, parse_mode=None)
            notification_sent = True
            logger.warning("No data received for over 1 hour")
        elif time_since_last_data <= timedelta(hours=1):
            notification_sent = False
        
        # Check every minute
        stop_event.wait(60)


def handle_telegram_commands(kiuas: Kiuas, bot: TelegramBot, config: Config, stop_event: threading.Event) -> None:
    """Handle incoming Telegram commands.
    
    Args:
        kiuas: Kiuas state object
        bot: Telegram bot instance
        config: Application configuration
        stop_event: Threading event to signal shutdown
    """
    offset = None
    
    while not stop_event.is_set():
        try:
            updates = bot.get_updates(offset=offset, timeout=30)
            
            if updates:
                for update in updates:
                    # Update offset to acknowledge this update
                    offset = update.get("update_id", 0) + 1
                    
                    message = update.get("message", {})
                    text = message.get("text", "")
                    chat_id = message.get("chat", {}).get("id")
                    
                    if not text or not chat_id:
                        continue
                    
                    # Handle /kiuas command
                    if text.startswith("/kiuas"):
                        status = kiuas.get_status_text(config.ready_threshold, config.lower_bound)
                        response = (
                            f"Sauna on {status}\n"
                            f"Lämpötila: {kiuas.temperature:.1f} °C\n"
                            f"Kosteus: {kiuas.humidity:.1f}%"
                        )
                        bot.send_message(response, chat_id=chat_id, parse_mode=None)
                        logger.info(f"Handled /kiuas command from chat {chat_id}")
                    
                    # Handle /info command (maintenance only)
                    elif text.startswith("/info") and chat_id == config.maintenance_chat_id:
                        response = (
                            f"Sauna Info:\n"
                            f"Temperature: {kiuas.temperature:.1f} °C\n"
                            f"Humidity: {kiuas.humidity:.1f}%\n"
                            f"Battery: {kiuas.battery} mV\n"
                            f"Last Data Received: {kiuas.last_data_received.strftime('%Y-%m-%d %H:%M:%S')}"
                        )
                        bot.send_message(response, chat_id=chat_id, parse_mode=None)
                        logger.info(f"Handled /info command from chat {chat_id}")
                        
        except Exception as e:
            logger.error(f"Error handling Telegram commands: {e}", exc_info=True)
            time.sleep(5)  # Wait before retrying


def main():
    """Main application entry point."""
    # Set timezone
    os.environ["TZ"] = "Europe/Bucharest"
    time.tzset()
    
    # Load environment variables from .env file if it exists
    try:
        from dotenv import load_dotenv
        load_dotenv()
        logger.info("Loaded environment variables from .env file")
    except ImportError:
        logger.info("python-dotenv not available, using system environment variables")
    except Exception as e:
        logger.warning(f"Failed to load .env file: {e}")
    
    # Load configuration
    try:
        config = Config.from_env()
        logger.info("Configuration loaded successfully")
    except ValueError as e:
        logger.error(f"Configuration error: {e}")
        sys.exit(1)
    
    # Initialize Kiuas state
    kiuas = Kiuas()
    logger.info("Kiuas state initialized")
    
    # Initialize Telegram bot
    bot = TelegramBot(
        token=config.telegram_bot_token,
        default_chat_id=config.notification_chat_id
    )
    
    # Register bot commands
    if bot.register_commands():
        logger.info("Bot commands registered")
    else:
        logger.warning("Failed to register bot commands")
    
    # Start HTTP server
    server = start_http_server(kiuas, bot, config)
    
    # Start monitoring threads
    stop_event = threading.Event()
    
    monitor_thread = threading.Thread(
        target=monitor_data_reception,
        args=(kiuas, bot, config, stop_event),
        daemon=True
    )
    monitor_thread.start()
    logger.info("Data reception monitoring started")
    
    command_thread = threading.Thread(
        target=handle_telegram_commands,
        args=(kiuas, bot, config, stop_event),
        daemon=True
    )
    command_thread.start()
    logger.info("Telegram command handler started")
    
    # Setup signal handlers for graceful shutdown
    def signal_handler(sig, frame):
        logger.info("Shutdown signal received, stopping...")
        stop_event.set()
        server.shutdown()
        sys.exit(0)
    
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    logger.info("Sauna monitoring service started successfully")
    
    # Keep main thread alive
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        logger.info("Keyboard interrupt received, stopping...")
        stop_event.set()
        server.shutdown()


if __name__ == "__main__":
    main()
