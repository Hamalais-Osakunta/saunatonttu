"""Kiuas (sauna stove) monitoring and state management."""
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from typing import List, Tuple
import logging

logger = logging.getLogger(__name__)


@dataclass
class Kiuas:
    """Sauna stove state and temperature monitoring.
    
    Tracks temperature history and manages notification state for warming
    and ready events.
    """
    
    temperature: float = 0.0
    humidity: float = 0.0
    battery: int = 0
    warming_notification_sent: bool = False
    ready_notification_sent: bool = False
    last_data_received: datetime = field(default_factory=datetime.now)
    
    # Temperature and timestamp records (circular buffer of last 3 readings)
    temperature_records: List[float] = field(default_factory=lambda: [0.0, 0.0, 0.0])
    timestamp_records: List[datetime] = field(default_factory=lambda: [
        datetime.now(), datetime.now(), datetime.now()
    ])
    
    def is_on(self, ready_threshold: float, lower_bound: float) -> bool:
        """Check if sauna is currently on (warming or ready).
        
        Args:
            ready_threshold: Temperature threshold for ready state
            lower_bound: Minimum warming rate threshold
            
        Returns:
            True if sauna is on (temperature >= ready threshold or warming)
        """
        return self.temperature >= ready_threshold or self.is_warming(ready_threshold, lower_bound)
    
    def temp_change_rate(self) -> float:
        """Calculate the rate of temperature change in degrees per second.
        
        Returns:
            Temperature change rate in °C/s, or 0 if insufficient data
        """
        # Ensure there are at least three valid records
        if self.temperature_records[0] == 0:
            logger.debug("Not enough valid temperature records")
            return 0.0
        
        # Calculate time difference between first and last record
        time_diff = (self.timestamp_records[2] - self.timestamp_records[0]).total_seconds()
        if time_diff == 0:
            return 0.0
        
        # Calculate temperature change over the three records
        temp_change = self.temperature_records[2] - self.temperature_records[0]
        temp_change_rate = temp_change / time_diff  # degrees per second
        
        # Check if the temperature change rate is positive
        if temp_change_rate <= 0:
            logger.debug("Temperature change rate should be positive")
            return 0.0
        
        return temp_change_rate
    
    def is_warming(self, ready_threshold: float, lower_bound: float) -> bool:
        """Check if the sauna is warming up.
        
        Args:
            ready_threshold: Temperature threshold for ready state
            lower_bound: Minimum warming rate threshold
            
        Returns:
            True if sauna is actively warming (positive rate above threshold)
        """
        temp_rate = self.temp_change_rate()
        return (
            temp_rate > 0 
            and temp_rate >= lower_bound 
            and self.temperature < ready_threshold
        )
    
    def get_estimated_ready_seconds(self, ready_threshold: float) -> float:
        """Calculate estimated time in seconds until sauna is ready.
        
        Args:
            ready_threshold: Temperature threshold for ready state
            
        Returns:
            Estimated seconds until ready, or 0 if not warming
        """
        rate = self.temp_change_rate()
        if rate <= 0:
            return 0.0
        
        temp_remaining = ready_threshold - self.temperature
        time_to_ready_seconds = temp_remaining / rate
        
        return time_to_ready_seconds
    
    def add_temperature_record(self, new_temp: float, new_time: datetime) -> None:
        """Add a new temperature record, shifting old values.
        
        Args:
            new_temp: New temperature reading in Celsius
            new_time: Timestamp of the reading
        """
        # Shift the array to remove the oldest record and add the new one
        self.temperature_records[0] = self.temperature_records[1]
        self.temperature_records[1] = self.temperature_records[2]
        self.temperature_records[2] = new_temp
        
        # Similarly update the timestamps
        self.timestamp_records[0] = self.timestamp_records[1]
        self.timestamp_records[1] = self.timestamp_records[2]
        self.timestamp_records[2] = new_time
    
    def reset_notifications(self) -> None:
        """Reset notification flags when sauna cools down."""
        self.warming_notification_sent = False
        self.ready_notification_sent = False
    
    def get_status_text(self, ready_threshold: float, lower_bound: float) -> str:
        """Get human-readable status text.
        
        Args:
            ready_threshold: Temperature threshold for ready state
            lower_bound: Minimum warming rate threshold
            
        Returns:
            Status string in Finnish ("päällä" or "pois päältä")
        """
        if self.is_on(ready_threshold, lower_bound):
            return "päällä"
        return "pois päältä"
