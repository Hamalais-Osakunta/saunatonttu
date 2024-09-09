import redis
import psycopg2
import time
import os

# Function to create Redis connection
def create_redis_connection():
    redis_host = os.getenv("REDIS_HOST", "redis")
    redis_port = int(os.getenv("REDIS_PORT", 6379))
    return redis.StrictRedis(host=redis_host, port=redis_port, decode_responses=True)

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
    #check if the connection is successful
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

# Function to ingest data from Redis and store in TimescaleDB
def ingest_data(redis_conn, db_conn, max_iterations=None):
    print("Starting data ingestion...")
    cursor = db_conn.cursor()
    iterations = 0
    while True:
        # Blocking pop from Redis queue
        data = redis_conn.blpop("ruuvi_data_queue", timeout=10)
        if data:
            measurement = data[1].split(",")
            sensor_mac = measurement[0]
            temperature = float(measurement[1])
            humidity = float(measurement[2])
            battery = float(measurement[3])
            timestamp = measurement[4]

            cursor.execute(
                "INSERT INTO sauna_data (sensor_mac, temperature, humidity, battery, timestamp) VALUES (%s, %s, %s, %s, %s)",
                (sensor_mac, temperature, humidity, battery, timestamp)
            )
            db_conn.commit()

        time.sleep(10)  # Adjust this to your needs

        if max_iterations is not None:
            iterations += 1
            if iterations >= max_iterations:
                break

if __name__ == '__main__':
    print("Starting data ingestion service...")

    # Create Redis and TimescaleDB connections
    redis_conn = create_redis_connection()
    db_conn = create_timescale_connection()

    # Ensure table exists
    ensure_table_exists(db_conn.cursor())

    # Start data ingestion
    ingest_data(redis_conn, db_conn)
