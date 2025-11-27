FROM python:3.12-slim

# Set working directory
WORKDIR /app

# Copy requirements first for better layer caching
COPY requirements.txt .

# Install dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy application files
COPY *.py .

# Create empty .env file placeholder
RUN touch .env

# Set timezone
ENV TZ=Europe/Helsinki

# Expose the server port
EXPOSE 1337

# Run the application
CMD ["python", "-u", "main.py"]
