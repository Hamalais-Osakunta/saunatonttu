from fastapi import FastAPI, Request
import redis
import os
from ruuvitag_sensor.data_formats import DataFormats
from ruuvitag_sensor.decoder import get_decoder
from datetime import datetime

app = FastAPI()


# Redis connection setup
redis_host = os.getenv("REDIS_HOST", "redis")
redis_port = int(os.getenv("REDIS_PORT", 6379))
r = redis.StrictRedis(host=redis_host, port=redis_port, decode_responses=True)

# Function to decode the bytestream (Ruuvitag data)
# Assuming the bytestream contains binary data that needs to be decoded
def decode_ruuvitag_bytestream(bytestream: bytes):
    full_data = bytestream.hex()
    
    # check manufacturer ID
    if full_data[0:4] != "9904":
        raise ValueError("Invalid manufacturer ID")

    sensor_data = get_decoder(5).decode_data(full_data[4:])

    return sensor_data

# Function to push decoded data to Redis
def push_data_to_redis(mac, measurement_data):
    # Create a CSV-like string to store in Redis
    timestamp = datetime.now()
    measurement_str = f"{mac},{measurement_data['temperature']},{measurement_data['humidity']},{measurement_data['battery']},{timestamp}"
    
    # Push data to Redis queue
    r.rpush("ruuvi_data_queue", measurement_str)

# POST endpoint to receive bytestream data
@app.post("/ingest")
async def ingest_data(request: Request):
    try:
        # Read the raw bytestream from the POST request
        bytestream = await request.body()
        
        # Decode the bytestream into usable sensor data
        decoded_data = decode_ruuvitag_bytestream(bytestream)
        
        print(f"Decoded data: {decoded_data}")
        
        # Push the decoded data to Redis
        push_data_to_redis(decoded_data["mac"], decoded_data)
        
        return {"status": "success", "message": "Data ingested successfully"}
    
    except Exception as e:
        return {"status": "error", "message": str(e)}
