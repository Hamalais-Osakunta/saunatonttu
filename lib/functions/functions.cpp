#include <ESP8266WiFi.h>

const uint8_t ledPin = LED_BUILTIN;

/**
 * Setup led
 */
void setupLed()
{
    pinMode(ledPin, OUTPUT); // Initialize digital ledPin as an output.
}

// Use blink to blink led ;)
void blink(long time)
{
    digitalWrite(ledPin, LOW);
    delay(time);
    digitalWrite(ledPin, HIGH);
}