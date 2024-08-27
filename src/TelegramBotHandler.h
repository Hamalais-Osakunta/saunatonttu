#include <UniversalTelegramBot.h>
#include "<kiuas.h>"
#include "<.env.h>"

class TelegramBotHandler
{
private:
    UniversalTelegramBot &bot; // Reference to the bot object
    Kiuas &kiuas;              // Reference to the Kiuas object

    bool kiuas_before = false;
    int temperature_before = 0;
    bool ready_before = false;
    bool warming_notified = false;
    bool cooling_notified = false;

public:
    // Constructor to inject dependencies
    TelegramBotHandler(UniversalTelegramBot &bot, Kiuas &kiuas)
        : bot(bot), kiuas(kiuas) {}

    // Handle events based on Kiuas status
    void handleEvent()
    {
        bool kiuas_current = kiuas.isSaunaOn();
        int temperature_current = kiuas.getTemperature();

        // Check if sauna on/off status has changed
        if (kiuas_before != kiuas_current)
        {
            Serial.println("Kiuas changed from " + String(kiuas_before) + " to " + String(kiuas_current));

            // Handle Kiuas events
            if (kiuas_current)
            {
                if (temperature_current < 70 && !warming_notified)
                {
                    bot.sendMessage(SAUNA_CHAT, "Sauna lämpiää, lämpötila " + String(temperature_current) + "°C", "Markdown");
                    warming_notified = true;
                }
                else if (temperature_current >= 70 && !ready_before)
                {
                    bot.sendMessage(SAUNA_CHAT, "Sauna valmis, lämpötila " + String(temperature_current) + "°C", "Markdown");
                    ready_before = true;
                }
            }
            else
            {
                bot.sendMessage(SAUNA_CHAT, "Sauna sammui, lämpötila " + String(temperature_current) + "°C", "Markdown");
                ready_before = false;
                warming_notified = false; // Reset warming notification when sauna turns off
                cooling_notified = false; // Reset cooling notification when sauna turns off
            }

            kiuas_before = kiuas_current;
        }
    }

    void handleMessage(telegramMessage msg)
    {
        String chat_id = msg.chat_id;
        String text = msg.text;

        String chat_id_display = chat_id;
        if (chat_id_display == String(SAUNA_CHAT))
        {
            chat_id_display = "SAUNA_CHAT";
        }

        if (chat_id_display == String(MAINTENANCE_CHAT))
        {
            chat_id_display = "MAINTENANCE_CHAT";
        }

        if (chat_id != MAINTENANCE_CHAT or chat_id != SAUNA_CHAT)
        {
            // Skip unknown chats
            Serial.println("Message " + String(msg.message_id) + " skipped");
            return;
        }

        // Avaa komennot ja selitteet
        if (text == "/apua" or text == "/start")
        {
            Serial.println("Replaing to " + String(msg.message_id) + " as start");

            String keyboardJson = "[[{ \"text\" : \"Kiukaan kuulumiset\", \"callback_data\" : \"/kiuas\" }]]";
            bot.sendMessageWithInlineKeyboard(chat_id, "Choose from one of the following options", "", keyboardJson);
        }

        // Kertoo kiukaan lämpötilan ja tilan
        if (text == "/kiuas")
        {
            Serial.println("Replaing to " + String(msg.message_id) + " as kiuas");

            String on = getKiuas() ? "päällä" : "pois";
            bot.sendMessage(chat_id, "Kiuas on " + on + ", lämpötila " + String(getTemperature()) + "°C", "Markdown");
        }
    }
};
