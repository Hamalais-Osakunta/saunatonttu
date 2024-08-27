#include <UniversalTelegramBot.h>

#include <functions.h>
#include <connection.h>
#include <measurement.h>
#include <kiuas.h>
#include <TelegramBotHandler.h>

#include <.env.h>

// Mean time between scan messages
const unsigned long BOT_MTBS = 1000;

// Mean time between scan events
const unsigned long EVENT_MTBS = 400;

X509List cert(TELEGRAM_CERTIFICATE_ROOT);
WiFiClientSecure secured_client;
UniversalTelegramBot bot(BOT_TOKEN, secured_client);
Kiuas kiuas;

TelegramBotHandler botHandler(bot, kiuas);

// Last time messages scan has been done
unsigned long bot_lasttime;

// Last time events scan has been done
unsigned long event_lasttime;

void setup()
{
  Serial.begin(9600);
  Serial.println();

  setupLed();

  Connection::turn_ON_WIFI(); // Turn on Wifi

  // Attempt to connect to Wifi network:
  Connection::updateNTP();
  secured_client.setTrustAnchors(&cert); // Add root certificate for api.telegram.org

  bot.sendMessage(MAINTENANCE_CHAT, "Saunatonttu on käynnistynyt.", "Markdown");
}

void loop()
{

  NimBLEAdvertisedDevice device = Connection::turn_ON_BLE();

  if (device.getAddress().toString() != "")
  {
    Measurement::RuuviMeasurement data = Measurement::readDataFromDevice(device);
    kiuas.updateStatus(data);
  }
  else
  {
    Serial.println("No device found.");
  }

  Connection::turn_OFF_BLE();

  if (millis() - event_lasttime > EVENT_MTBS)
  {
    botHandler.handleEvent();
    event_lasttime = millis();
  }

  // Handle incoming messages
  if (millis() - bot_lasttime > BOT_MTBS)
  {
    int numNewMessages = bot.getUpdates(bot.last_message_received + 1);

    while (numNewMessages)
    {
      for (int i = 0; i < numNewMessages; i++)
      {
        botHandler.handleMessage(bot.messages[i]);
      }
      numNewMessages = bot.getUpdates(bot.last_message_received + 1);
    }

    bot_lasttime = millis();
  }
}
