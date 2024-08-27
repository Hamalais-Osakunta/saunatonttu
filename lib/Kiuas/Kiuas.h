#ifndef KIUAS_H
#define KIUAS_H

#include <measurement.h>

class Kiuas
{
private:
    double temperature{0.0};          // Current temperature of the sauna
    bool sauna_on{false};             // Status of the sauna (on/off)
    int temperature_before{0};        // Temperature before the last significant change

    const unsigned long TIME_THRESHOLD{180000}; // 3 minutes in milliseconds

    unsigned long lastSignificantChangeTime{0}; // Time when a significant change was first detected
    int tempDuringLastChange{0};                // Temperature at the time of the last significant change

public:
    // Getter for temperature
    double getTemperature() const;

    // Getter for sauna_on
    bool isSaunaOn() const;

    // Method to update the status from a DeviceData object
    void updateStatus(const Measurement::RuuviMeasurement &deviceData);
};

#endif // KIUAS_H
