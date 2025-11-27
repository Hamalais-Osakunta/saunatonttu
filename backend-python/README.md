# Sauna Monitoring Service (Python)

A robust Python-based sauna monitoring service that receives temperature data from RuuviTag sensors and sends Telegram notifications when the sauna is warming up or ready.

## Features

- 🌡️ **Real-time Temperature Monitoring**: Receives and processes RuuviTag RAWv2 format sensor data
- 🔥 **Smart Notifications**: Sends Telegram alerts when sauna starts warming and when it's ready
- 📊 **Temperature Trend Analysis**: Calculates warming rate and estimates ready time
- 🔄 **Automatic Reset**: Resets notification state when sauna cools down
- 🛡️ **Error-Proof Architecture**: Retry logic, proper error handling, and graceful degradation
- 🐳 **Docker Support**: Easy containerized deployment

## Architecture

The application is built with a modular, robust architecture:

```
backend-python/
├── main.py                  # Application entry point with HTTP server
├── config.py                # Configuration management
├── kiuas.py                 # Sauna state model and logic
├── ruuvitag_parser.py       # RuuviTag RAWv2 data parser
├── telegram_bot.py          # Telegram bot wrapper with retry logic
├── notification_service.py  # Notification logic
├── test_main.py            # Comprehensive unit tests
├── requirements.txt        # Python dependencies
├── Dockerfile              # Container build file
└── README.md              # This file
```

### Key Design Principles

- **Separation of Concerns**: Each module has a single, well-defined responsibility
- **Error Handling**: Comprehensive try-catch blocks with proper logging
- **Retry Logic**: Automatic retries for Telegram API calls with exponential backoff
- **Thread Safety**: Proper synchronization for background monitoring tasks
- **Testability**: Mock-friendly interfaces and comprehensive unit tests

## Requirements

- Python 3.11+
- Dependencies (see `requirements.txt`):
  - `requests` - HTTP client for Telegram API
  - `python-dotenv` - Environment variable management

## Configuration

The service is configured via environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `TELEGRAM_BOT_TOKEN` | ✅ Yes | - | Telegram bot API token |
| `SAUNA_READY_THRESHOLD` | ❌ No | 75.0 | Temperature threshold (°C) for "ready" state |
| `MAINTENANCE_CHAT_ID` | ✅ Yes | - | Chat ID for maintenance alerts |
| `NOTIFICATION_CHAT_ID` | ✅ Yes | - | Chat ID for user notifications |
| `SERVER_PORT` | ❌ No | 1337 | HTTP server port for data reception |

### Example .env File

```bash
TELEGRAM_BOT_TOKEN=your_bot_token_here
SAUNA_READY_THRESHOLD=75.0
MAINTENANCE_CHAT_ID=123456789
NOTIFICATION_CHAT_ID=987654321
SERVER_PORT=1337
```

## Installation & Running

### Option 1: Local Python

```bash
# Install dependencies
pip install -r requirements.txt

# Create .env file with your configuration
cp .env.example .env
nano .env

# Run the service
python main.py
```

### Option 2: Docker

```bash
# Build the image
docker build -t sauna-monitor .

# Run with environment variables
docker run -d \
  --name sauna-monitor \
  -p 1337:1337 \
  -e TELEGRAM_BOT_TOKEN=your_token \
  -e MAINTENANCE_CHAT_ID=123456789 \
  -e NOTIFICATION_CHAT_ID=987654321 \
  -e SAUNA_READY_THRESHOLD=75.0 \
  sauna-monitor

# View logs
docker logs -f sauna-monitor
```

### Option 3: Docker Compose

```yaml
version: '3.8'
services:
  sauna-monitor:
    build: ./backend-python
    ports:
      - "1337:1337"
    environment:
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - MAINTENANCE_CHAT_ID=${MAINTENANCE_CHAT_ID}
      - NOTIFICATION_CHAT_ID=${NOTIFICATION_CHAT_ID}
      - SAUNA_READY_THRESHOLD=75.0
    restart: unless-stopped
```

## API Endpoints

### POST /api/receive-bt

Receives RuuviTag sensor data in RAWv2 binary format.

**Request:**
- Method: POST
- Content-Type: application/octet-stream
- Body: Raw binary RuuviTag RAWv2 data (24-26 bytes)

**Response:**
```json
{
  "status": "ok"
}
```

**Example:**
```bash
# Send test data (replace with actual RuuviTag data)
curl -X POST http://localhost:1337/api/receive-bt \
  --data-binary @ruuvitag_data.bin
```

## Telegram Commands

The bot responds to the following commands:

- `/kiuas` - Show current sauna status (temperature, humidity, on/off state)
- `/info` - Show detailed maintenance information (maintenance chat only)

## Running Tests

```bash
# Run all tests
python -m unittest test_main.py

# Run with verbose output
python -m unittest test_main.py -v

# Run specific test
python -m unittest test_main.TestKiuas.test_add_temperature_record
```

### Test Coverage

The test suite includes:
- ✅ Temperature record management
- ✅ Temperature change rate calculation
- ✅ Warming detection logic
- ✅ Ready time estimation
- ✅ Notification triggering (warming and ready)
- ✅ Notification deduplication
- ✅ State reset when cooled

## Monitoring & Logging

The service logs important events to stdout:

- Data reception from sensors
- Notification sending
- Configuration errors
- API failures and retries
- Background task status

**View logs:**
```bash
# Local Python
python main.py 2>&1 | tee sauna.log

# Docker
docker logs -f sauna-monitor
```

## Error Handling

The service includes comprehensive error handling:

1. **Configuration Errors**: Validates all required environment variables at startup
2. **Parse Errors**: Gracefully handles invalid RuuviTag data
3. **Network Errors**: Retries Telegram API calls with exponential backoff (3 attempts)
4. **Missing Data**: Alerts maintenance chat if no data received for 1 hour
5. **Graceful Shutdown**: Handles SIGINT/SIGTERM signals cleanly

## Differences from Go Version

This Python implementation maintains feature parity with the original Go service while adding:

- ✨ More modular architecture with clear separation of concerns
- ✨ Enhanced error handling with retry logic
- ✨ Comprehensive unit test suite
- ✨ Better logging and observability
- ✨ Simpler deployment (no external Telegram library needed)
- ✨ Type hints for better code clarity

## Development

### Project Structure

```python
# Config management
config.py           # Environment variable loading and validation

# Core models
kiuas.py           # Sauna state and temperature analysis

# External integrations
ruuvitag_parser.py # Binary data parsing
telegram_bot.py    # Telegram API wrapper with retry logic

# Business logic
notification_service.py  # Notification decision logic

# Application
main.py            # HTTP server and background tasks

# Testing
test_main.py       # Unit tests
```

### Adding Features

1. **New notification types**: Extend `notification_service.py`
2. **New sensor types**: Add parser in `ruuvitag_parser.py`
3. **New commands**: Update `handle_telegram_commands()` in `main.py`

## License

[Your License Here]

## Contributing

Contributions are welcome! Please ensure:
- All tests pass: `python -m unittest test_main.py`
- Code follows PEP 8 style guide
- New features include tests
- Documentation is updated
