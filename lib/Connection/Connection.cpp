#include "Connection.h"
#include <Arduino.h> // Assuming you're using this in an Arduino environment

namespace Connection
{
    // Initialize global variables
    time_t measurementTimeNow;
    time_t prevNtpUpdateTime{0};
    bool ntpTimeIsSet{false};

    /**
     * Turn on WIFI
     */
    void turn_ON_WIFI()
    {
        Serial.print("Connecting to Wifi SSID ");
        Serial.print(WIFI_SSID);
        WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
        while (WiFi.status() != WL_CONNECTED)
        {
            blink(50);
            Serial.print(".");
            delay(50);
        }
        Serial.println("\nWiFi connected. IP address: ");
        Serial.println(WiFi.localIP());
    }

    /**
     * Turn off WIFI
     */
    void turn_OFF_WIFI()
    {
        Serial.println("WIFI OFF");
        WiFi.mode(WIFI_OFF);
    }

    /**
     * MyAdvertisedDeviceCallbacks: Called for each advertising BLE server.
     */
    void MyAdvertisedDeviceCallbacks::onResult(NimBLEAdvertisedDevice advertisedDevice)
    {
        if (advertisedDevice.getAddress().toString() == RUUVI_TAG_MAC)
        {
            Serial.print("Found Tag");
            foundDevice = advertisedDevice; // Store the found device
            deviceFound = true;
        }
    }

    /**
     * MyAdvertisedDeviceCallbacks: Function to check if a device was found
     */
    bool MyAdvertisedDeviceCallbacks::isDeviceFound()
    {
        return deviceFound;
    }

    /**
     * MyAdvertisedDeviceCallbacks: Function to return the stored device
     */
    NimBLEAdvertisedDevice MyAdvertisedDeviceCallbacks::getFoundDevice()
    {
        deviceFound = false; // Reset after returning the device
        return foundDevice;
    }

    /**
     * Turn on BLE and scan for RuuviTags
     */
    NimBLEAdvertisedDevice turn_ON_BLE()
    {
        Serial.println("BLE ON");

        NimBLEDevice::init("");
        NimBLEScan *pBLEScan = NimBLEDevice::getScan(); // create new scan
        MyAdvertisedDeviceCallbacks *myCallbacks = new MyAdvertisedDeviceCallbacks();
        pBLEScan->setAdvertisedDeviceCallbacks(myCallbacks);
        pBLEScan->setActiveScan(true); // active scan uses more power, but gets results faster
        pBLEScan->setInterval(100);
        pBLEScan->setWindow(99); // less or equal to setInterval value

        delay(500);
        time(&measurementTimeNow);
        NimBLEScanResults foundDevices = pBLEScan->start(Config::BLEscanTime, false);
        pBLEScan->clearResults();

        // Check if a device was found and return it
        if (myCallbacks->isDeviceFound())
        {
            return myCallbacks->getFoundDevice();
        }

        // Return a default NimBLEAdvertisedDevice if no device was found
        return NimBLEAdvertisedDevice();
    }

    /**
     * Turn off BLE
     */
    void turn_OFF_BLE()
    {
        Serial.println("BLE OFF");
        NimBLEDevice::deinit(false);
    }

    /**
     * Update NTP time if it is not updated in the last hour
     */
    void updateNTP()
    {
        time_t timeNow;
        time(&timeNow);
        if (timeNow > 1500000000 && (timeNow - prevNtpUpdateTime) < (60 * 60))
        {
            return;
        }

        Serial.println("Try to update NTP");
        if (WiFi.status() == WL_CONNECTED)
        {
            Serial.println("NTP update starting... ");

            configTime(0, 0, "pool.ntp.org", "fi.pool.ntp.org", "time.mikes.fi");
            delay(2000);
            time(&timeNow);
            Serial.println("NTP update completed.");
            if (timeNow > 1500000000)
            {                                             // is there some status that would tell if the update was successful
                Serial.println("NTP update successful."); // perhaps we could wait until we have correct time until we send the measurements to the server
                time(&prevNtpUpdateTime);
                ntpTimeIsSet = true;
            }
        }
        else
        {
            Serial.println("No WIFI available, NTP not updated");
        }
    }
}
