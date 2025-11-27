"""Configuration management for the sauna monitoring service."""
import os
from dataclasses import dataclass
from typing import Optional


@dataclass
class Config:
    """Application configuration loaded from environment variables."""
    
    ready_threshold: float
    change_threshold: float
    lower_bound: float
    reset_threshold: float
    maintenance_chat_id: int
    notification_chat_id: int
    server_port: int
    telegram_bot_token: str
    timezone: str = "Europe/Bucharest"
    
    @classmethod
    def from_env(cls) -> "Config":
        """Load configuration from environment variables.
        
        Raises:
            ValueError: If required environment variables are missing or invalid.
        """
        try:
            ready_threshold = float(os.getenv("SAUNA_READY_THRESHOLD", "75.0"))
            maintenance_chat_id = int(os.getenv("MAINTENANCE_CHAT_ID", "0"))
            notification_chat_id = int(os.getenv("NOTIFICATION_CHAT_ID", "0"))
            server_port = int(os.getenv("SERVER_PORT", "1337"))
            telegram_bot_token = os.getenv("TELEGRAM_BOT_TOKEN", "")
            
            if not telegram_bot_token:
                raise ValueError("TELEGRAM_BOT_TOKEN environment variable is required")
            
            if maintenance_chat_id == 0:
                raise ValueError("MAINTENANCE_CHAT_ID environment variable is required")
            
            if notification_chat_id == 0:
                raise ValueError("NOTIFICATION_CHAT_ID environment variable is required")
            
            # Derived values
            change_threshold = 0.0123
            lower_bound = change_threshold * 0.9
            reset_threshold = 40.0
            
            return cls(
                ready_threshold=ready_threshold,
                change_threshold=change_threshold,
                lower_bound=lower_bound,
                reset_threshold=reset_threshold,
                maintenance_chat_id=maintenance_chat_id,
                notification_chat_id=notification_chat_id,
                server_port=server_port,
                telegram_bot_token=telegram_bot_token,
            )
        except ValueError as e:
            raise ValueError(f"Configuration error: {e}")
