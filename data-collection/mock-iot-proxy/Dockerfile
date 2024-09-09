# Use an official Python runtime as a parent image
FROM python:3.10-slim

# Set the working directory in the container
WORKDIR /app

# Copy the requirements.txt file (if you have one) and install dependencies
COPY requirements.txt .

# Install any dependencies listed in requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Copy the current directory contents into the container at /app
COPY . .

# Expose port 8000 for potential future use (in case of FastAPI service running)
EXPOSE 8000

# Run the service
CMD ["python", "-u", "mock_iot_proxy_service.py"]
