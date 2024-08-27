#include "Kiuas.h"
#include <Arduino.h> // Assuming this is for an Arduino-based project
#include ".env.h"


Kiuas::Kiuas()
{
    // Initialize the temperature to 0
    temperature = 0.0;
    sauna_on = false;
    temperature_before = 0;
    lastSignificantChangeTime = 0;
    tempDuringLastChange = 0;
    
}

double Kiuas::getTemperature() const
{
    return temperature;
}

bool Kiuas::isSaunaOn() const
{
    return sauna_on;
}

void Kiuas::updateStatus(const Measurement::RuuviMeasurement &deviceData)
{
    // Update the temperature from the device data
    temperature = deviceData.temperature;

    unsigned long currentTime = millis(); // Current time in milliseconds

    // Determine the sauna status based on some logic
    // Check if temperature has changed significantly
    if (abs(temperature_before - temperature) >= SAUNA_CHANGE_THRESHOLD)
    {
        if (lastSignificantChangeTime == 0)
        {
            // Start timing the significant change
            lastSignificantChangeTime = currentTime;
            tempDuringLastChange = temperature;
        }
        else if (currentTime - lastSignificantChangeTime >= TIME_THRESHOLD)
        {
            // If the change has persisted for the threshold time, consider it real
            Serial.println("Temperature change persisted for 3 minutes.");

            // Handle warming event (temperature rising above a certain threshold)
            if (temperature > SAUNA_WARMING_TEMP)
            {
                sauna_on = true;
            }

            // Handle cooling event (temperature falling below a certain threshold)
            if (temperature < SAUNA_READY_TEMP && sauna_on)
            {
                sauna_on = false;
            }

            // Update the temperature and reset the timing for the next significant change
            temperature_before = temperature;
            lastSignificantChangeTime = 0;
        }
    }
}