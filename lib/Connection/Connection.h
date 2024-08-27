#ifndef CONNECTION_H
#define CONNECTION_H

#include <WiFi.h>
#include <WiFiClientSecure.h>
#include <NimBLEDevice.h>
#include <functions.h>
#include <env.h>

namespace Connection
{
    void turn_ON_WIFI();
    void turn_OFF_WIFI();
    NimBLEAdvertisedDevice turn_ON_BLE();
    void turn_OFF_BLE();
    void updateNTP();

    extern time_t measurementTimeNow;
    extern time_t prevNtpUpdateTime;
    extern bool ntpTimeIsSet;

    // MyAdvertisedDeviceCallbacks class definition
    class MyAdvertisedDeviceCallbacks : public NimBLEAdvertisedDeviceCallbacks
    {
    private:
        NimBLEAdvertisedDevice foundDevice;
        bool deviceFound;

    public:
        MyAdvertisedDeviceCallbacks() : deviceFound(false) {}

        void onResult(NimBLEAdvertisedDevice advertisedDevice) override;
        bool isDeviceFound();
        NimBLEAdvertisedDevice getFoundDevice();
    };
}

#endif // CONNECTION_H
