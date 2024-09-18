#include <Arduino.h>
#include <BLEDevice.h>
#include <BLEUtils.h>
#include <BLEScan.h>
#include <WiFi.h>
#include <HTTPClient.h>
#include <time.h>       // Include time.h for NTP
#include ".env.h"       // Contains sensitive information like WiFi credentials and API key

#define SCAN_TIME 5      // seconds
#define CYCLE_TIME 10000 // milliseconds
#define RESTART_INTERVAL 600000 // 10 minutes in milliseconds

BLEAddress targetAddress = BLEAddress(RUUVI_TAG_MAC);
BLEScan *pBLEScan;
std::string receivedAdvertisement;

unsigned long lastRestartTime = 0;

class MyAdvertisedDeviceCallbacks : public BLEAdvertisedDeviceCallbacks
{
    void onResult(BLEAdvertisedDevice advertisedDevice)
    {
        if (advertisedDevice.getAddress() == targetAddress)
        {
            receivedAdvertisement = advertisedDevice.getManufacturerData();
            pBLEScan->stop();
        }
    }
};

MyAdvertisedDeviceCallbacks* callbacks;

// NTP server settings
const char* ntpServer = "pool.ntp.org";
const long  gmtOffset_sec = 0;    // Adjust according to your timezone
const int   daylightOffset_sec = 0;

void setup()
{
    Serial.begin(9600);
    callbacks = new MyAdvertisedDeviceCallbacks();
    lastRestartTime = millis();

    // Initialize WiFi to sync time
    WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
    Serial.println("Connecting to WiFi for NTP...");
    unsigned long wifiStartTime = millis();
    while (WiFi.status() != WL_CONNECTED)
    {
        delay(500);
        Serial.print(".");
        if (millis() - wifiStartTime > 10000)  // 10 seconds timeout
        {
            Serial.println("Failed to connect to WiFi for NTP");
            break;
        }
    }
    if (WiFi.status() == WL_CONNECTED)
    {
        Serial.println("\nWiFi connected for NTP");

        // Sync time with NTP server
        configTime(gmtOffset_sec, daylightOffset_sec, ntpServer);
        struct tm timeinfo;
        if(!getLocalTime(&timeinfo)){
            Serial.println("Failed to obtain time");
        } else {
            Serial.println("Time synchronized with NTP");
        }

        // Disconnect WiFi after time sync
        WiFi.disconnect(true);
        WiFi.mode(WIFI_OFF);
    }

    // Seed the random number generator for nonce generation
    randomSeed(esp_random());
}

void loop()
{
    HTTPClient http;
    int httpResponseCode;

    unsigned long startTime = millis();

    BLEDevice::init("");
    pBLEScan = BLEDevice::getScan();
    pBLEScan->setAdvertisedDeviceCallbacks(callbacks);
    pBLEScan->setActiveScan(true);
    pBLEScan->setInterval(100);
    pBLEScan->setWindow(99);

    // 1. Start Bluetooth and scan for advertisement
    Serial.println("Starting BLE scan...");
    pBLEScan->start(SCAN_TIME, false);

    if (receivedAdvertisement.empty())
    {
        Serial.println("Target device not found");
        BLEDevice::deinit(false);
    }
    else
    {
        Serial.println("Advertisement received");

        // 2. Stop Bluetooth, start WiFi
        BLEDevice::deinit(false);
        WiFi.begin(WIFI_SSID, WIFI_PASSWORD);

        // 3. Connect to WiFi
        Serial.println("Connecting to WiFi...");

        // Try connecting to WiFi for 5 seconds
        unsigned long wifiStartTime = millis();
        while (WiFi.status() != WL_CONNECTED)
        {
            delay(500);
            Serial.print(".");
            if (millis() - wifiStartTime > 5000)
            {
                break;
            }
        }
        if (WiFi.status() == WL_CONNECTED)
        {
            Serial.println("\nWiFi connected");

            // 4. Get current timestamp
            time_t now;
            struct tm timeinfo;
            if(!getLocalTime(&timeinfo)){
                Serial.println("Failed to obtain time");
                now = 0;
            } else {
                time(&now);
            }
            String timestamp = String((unsigned long)now);

            // 5. Generate nonce
            String nonce = "";
            for (int i = 0; i < 16; i++)
            {
                nonce += String(random(0, 16), HEX);
            }

            // 6. Send data to server with API key, timestamp, and nonce
            http.begin((std::string(API_URL) + "/api/receive-bt").c_str());
            http.addHeader("Content-Type", "application/octet-stream");
            http.addHeader("API-Key", API_KEY);
            http.addHeader("Timestamp", timestamp);
            http.addHeader("Nonce", nonce);

            httpResponseCode = http.POST((uint8_t *)receivedAdvertisement.c_str(), receivedAdvertisement.length());

            if (httpResponseCode > 0)
            {
                Serial.printf("HTTP Response code: %d\n", httpResponseCode);
            }
            else
            {
                Serial.printf("HTTP Request failed: %s\n", http.errorToString(httpResponseCode).c_str());
            }

            http.end();

            // 7. Stop WiFi
            WiFi.disconnect(true);
            WiFi.mode(WIFI_OFF);
        }
        else
        {
            Serial.println("WiFi connection failed");
        }
    }

    // Calculate sleep time
    unsigned long elapsedTime = millis() - startTime;
    long sleepTime = CYCLE_TIME - elapsedTime;

    if (sleepTime > 0)
    {
        Serial.printf("Sleeping for %ld ms\n", sleepTime);
        delay(sleepTime);
    }
    else
    {
        Serial.println("Cycle took longer than 10 seconds");
    }

    // Clear the received advertisement for the next cycle
    receivedAdvertisement.clear();

    // Optional: Clear scan results to free memory
    if (pBLEScan)
    {
        pBLEScan->clearResults();  // Free memory
    }

    // Restart ESP32 every 10 minutes
    if (millis() - lastRestartTime >= RESTART_INTERVAL)
    {
        Serial.println("Restarting ESP32...");
        ESP.restart();
    }
}
