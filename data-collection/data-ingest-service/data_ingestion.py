import psycopg2
import os
from fastapi import FastAPI, Request
from ruuvitag_sensor.decoder import get_decoder
from datetime import datetime

app = FastAPI()

# Function to create TimescaleDB connection
def create_timescale_connection():
    db_host = os.getenv("DB_HOST", "localhost")
    db_name = os.getenv("DB_NAME", "meter_data")
    db_user = os.getenv("DB_USER", "admin")
    db_password = os.getenv("DB_PASSWORD", "adminpassword")

    conn = psycopg2.connect(
        dbname=db_name,
        user=db_user,
        password=db_password,
        host=db_host
    )
    # Check if the connection is successful
    if conn:
        print("Connection Successful")
    else:
        print("Connection Unsuccessful")
    return conn

# Function to ensure the table exists
def ensure_table_exists(cursor):
    cursor.execute("""
    CREATE TABLE IF NOT EXISTS sauna_data (
        id SERIAL PRIMARY KEY,
        sensor_mac TEXT NOT NULL,
        temperature FLOAT NOT NULL,
        humidity FLOAT NOT NULL,
        battery FLOAT NOT NULL,
        timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
    """)
    cursor.connection.commit()

# Function to decode the bytestream (Ruuvitag data)
def decode_ruuvitag_bytestream(bytestream: bytes):
    full_data = bytestream.hex()
    
    # Check manufacturer ID
    if full_data[0:4] != "9904":
        raise ValueError("Invalid manufacturer ID")

    sensor_data = get_decoder(5).decode_data(full_data[4:])

    return sensor_data

# Function to insert data into TimescaleDB
def insert_data_to_db(cursor, mac, temperature, humidity, battery, timestamp):
    cursor.execute(
        "INSERT INTO sauna_data (sensor_mac, temperature, humidity, battery, timestamp) VALUES (%s, %s, %s, %s, %s)",
        (mac, temperature, humidity, battery, timestamp)
    )
    cursor.connection.commit()

# POST endpoint to receive bytestream data and store it in TimescaleDB
@app.post("/api/receive-bt")
async def ingest_data(request: Request):
    try:
        # Read the raw bytestream from the POST request
        bytestream = await request.body()
        
        # Decode the bytestream into usable sensor data
        decoded_data = decode_ruuvitag_bytestream(bytestream)
        
        print(f"Decoded data: {decoded_data}")
        
        # Get current timestamp
        timestamp = datetime.now()

        # Create TimescaleDB connection
        db_conn = create_timescale_connection()
        cursor = db_conn.cursor()

        # Ensure the table exists
        ensure_table_exists(cursor)
        
        # Insert the decoded data into the database
        insert_data_to_db(
            cursor, 
            decoded_data["mac"], 
            decoded_data["temperature"], 
            decoded_data["humidity"], 
            decoded_data["battery"], 
            timestamp
        )
        
        return {"status": "success", "message": "Data ingested successfully"}
    
    except Exception as e:
        return {"status": "error", "message": str(e)}
