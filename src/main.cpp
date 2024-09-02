#include <Arduino.h>
#include <BLEDevice.h>
#include <BLEUtils.h>
#include <BLEScan.h>
#include <WiFi.h>
#include <HTTPClient.h>
#include ".env.h"

#define SCAN_TIME 5      // seconds
#define CYCLE_TIME 10000 // milliseconds

BLEAddress targetAddress = BLEAddress(RUUVI_TAG_MAC);
BLEScan *pBLEScan;
std::string receivedAdvertisement;

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

void setup()
{
    Serial.begin(9600);
}

void loop()
{
    HTTPClient http;
    int httpResponseCode;

    unsigned long startTime = millis();

    BLEDevice::init("");
    pBLEScan = BLEDevice::getScan();
    pBLEScan->setAdvertisedDeviceCallbacks(new MyAdvertisedDeviceCallbacks());
    pBLEScan->setActiveScan(true);
    pBLEScan->setInterval(100);
    pBLEScan->setWindow(99);

    // 1. Start Bluetooth and scan for advertisement
    Serial.println("Starting BLE scan...");
    BLEDevice::getScan()->start(SCAN_TIME, false);
    delay(SCAN_TIME * 1000);
    BLEDevice::getScan()->stop(); //  not needed
    esp_bt_controller_disable();  // Disable the Bluetooth controller (optional, if necessary)
    
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
        unsigned long startTime = millis();
        while (WiFi.status() != WL_CONNECTED)
        {
            delay(500);
            Serial.print(".");
            if (millis() - startTime > 5000)
            {
                break;
            }
        }
        if (WiFi.status() == WL_CONNECTED)
        {
            Serial.println("\nWiFi connected");

            // 4. Send data to server

            http.begin((std::string(API_URL) + "/api/receive-bt").c_str());
            http.addHeader("Content-Type", "application/octet-stream");
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

            // 5. Stop WiFi
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
}