"""Telegram bot interface for sending notifications and handling commands."""
import logging
import time
from typing import Optional, Protocol
import requests
from config import Config

logger = logging.getLogger(__name__)


class TelegramBotInterface(Protocol):
    """Protocol defining the Telegram bot interface."""
    
    def send_message(self, text: str, chat_id: Optional[int] = None, parse_mode: str = "MarkdownV2") -> bool:
        """Send a message via Telegram."""
        ...
    
    def register_commands(self) -> bool:
        """Register bot commands."""
        ...


class TelegramBot:
    """Lightweight Telegram bot client with retry logic."""
    
    def __init__(self, token: str, default_chat_id: int, max_retries: int = 3):
        """Initialize Telegram bot.
        
        Args:
            token: Telegram bot API token
            default_chat_id: Default chat ID for notifications
            max_retries: Maximum number of retry attempts for failed requests
        """
        self.token = token
        self.default_chat_id = default_chat_id
        self.max_retries = max_retries
        self.base_url = f"https://api.telegram.org/bot{token}"
        self.session = requests.Session()
        self.session.headers.update({"Content-Type": "application/json"})
    
    def _make_request(self, method: str, endpoint: str, data: dict) -> Optional[dict]:
        """Make an API request with retry logic.
        
        Args:
            method: HTTP method (GET, POST)
            endpoint: API endpoint path
            data: Request payload
            
        Returns:
            Response JSON data or None on failure
        """
        url = f"{self.base_url}/{endpoint}"
        
        for attempt in range(self.max_retries):
            try:
                if method.upper() == "POST":
                    response = self.session.post(url, json=data, timeout=10)
                else:
                    response = self.session.get(url, params=data, timeout=10)
                
                response.raise_for_status()
                result = response.json()
                
                if result.get("ok"):
                    return result
                else:
                    logger.error(f"Telegram API error: {result.get('description')}")
                    
            except requests.exceptions.RequestException as e:
                logger.warning(f"Telegram request failed (attempt {attempt + 1}/{self.max_retries}): {e}")
                if attempt < self.max_retries - 1:
                    time.sleep(2 ** attempt)  # Exponential backoff
                    
        return None
    
    def send_message(
        self, 
        text: str, 
        chat_id: Optional[int] = None, 
        parse_mode: str = "MarkdownV2"
    ) -> bool:
        """Send a message via Telegram.
        
        Args:
            text: Message text to send
            chat_id: Target chat ID (uses default if not specified)
            parse_mode: Message parse mode (MarkdownV2, Markdown, HTML, or None)
            
        Returns:
            True if message was sent successfully, False otherwise
        """
        target_chat_id = chat_id if chat_id is not None else self.default_chat_id
        
        data = {
            "chat_id": target_chat_id,
            "text": text,
        }
        
        if parse_mode:
            data["parse_mode"] = parse_mode
        
        result = self._make_request("POST", "sendMessage", data)
        
        if result:
            logger.info(f"Message sent successfully to chat {target_chat_id}")
            return True
        else:
            logger.error(f"Failed to send message to chat {target_chat_id}")
            return False
    
    def register_commands(self) -> bool:
        """Register bot commands with Telegram.
        
        Returns:
            True if commands were registered successfully, False otherwise
        """
        commands = [
            {
                "command": "kiuas",
                "description": "Näytä saunan tila"
            }
        ]
        
        data = {"commands": commands}
        result = self._make_request("POST", "setMyCommands", data)
        
        if result:
            logger.info("Bot commands registered successfully")
            return True
        else:
            logger.error("Failed to register bot commands")
            return False
    
    def get_updates(self, offset: Optional[int] = None, timeout: int = 30) -> Optional[list]:
        """Get updates from Telegram (for command handling).
        
        Args:
            offset: Update ID offset for pagination
            timeout: Long polling timeout in seconds
            
        Returns:
            List of updates or None on failure
        """
        data = {
            "timeout": timeout,
            "allowed_updates": ["message"]
        }
        
        if offset is not None:
            data["offset"] = offset
        
        result = self._make_request("POST", "getUpdates", data)
        
        if result:
            return result.get("result", [])
        return None


def fmt_telegram(text: str) -> str:
    """Escape special characters for Telegram MarkdownV2 format.
    
    Args:
        text: Text to escape
        
    Returns:
        Escaped text safe for MarkdownV2 parsing
    """
    # Characters that need escaping in MarkdownV2
    special_chars = ['_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!']
    
    for char in special_chars:
        text = text.replace(char, f"\\{char}")
    
    return text
