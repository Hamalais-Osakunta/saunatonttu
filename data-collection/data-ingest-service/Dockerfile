# Dockerfile for Ruuvitag Data Insert Service
FROM python:3.10-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Copy the FastAPI script
COPY data_ingestion.py .

# Expose port for FastAPI
EXPOSE 8000

# Run the FastAPI application
CMD ["uvicorn", "data_ingestion:app", "--host", "0.0.0.0", "--port", "8000"]
