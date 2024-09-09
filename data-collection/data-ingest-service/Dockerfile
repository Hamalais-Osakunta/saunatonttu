# Dockerfile for Data Ingestion Service
FROM python:3.10-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Copy ingestion script
COPY data_ingestion.py .

# Run the data ingestion service
CMD ["python", "-u", "data_ingestion.py"]
