package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

import _ "time/tzdata"

func InitializeTelegramBot(ctx context.Context , token string, kiuas *Kiuas) (*bot.Bot, error) {
	maintenanceChatID, err := strconv.ParseInt(os.Getenv("MAINTENANCE_CHAT_ID"), 10, 64)
	if err != nil {
		log.Fatalf("Error parsing MAINTENANCE_CHAT_ID: %v", err)
	}

	opts := []bot.Option{}

	b, err := bot.New(token, opts...)
	if err != nil {
		return nil, err
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/kiuas", bot.MatchTypePrefix, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Sauna on %s\nLämpötila: %.1f °C\nKosteus: %.1f%%", GetSaunaStatus(kiuas.IsOn()), kiuas.Temperature, kiuas.Humidity),
		})
		if err != nil {
			fmt.Printf("Failed to send message: %v\n", err)
		}
	})

	b.RegisterHandler(bot.HandlerTypeMessageText, "/info", bot.MatchTypePrefix, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		if update.Message.Chat.ID == maintenanceChatID {
			loc, err := time.LoadLocation("Europe/Bucharest")
			if err != nil {
				fmt.Printf("Error loading location: %v", err)
			}
			_, err = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text: fmt.Sprintf(
					"Sauna Info:\nTemperature: %.1f °C\nHumidity: %.1f%%\nBattery: %d V\nLast Data Received: %s",
					saunaKiuas.Temperature,
					saunaKiuas.Humidity,
					saunaKiuas.Battery,
					saunaKiuas.lastDataReceived.In(loc))})
			if err != nil {
				fmt.Printf("Failed to send message: %v\n", err)
			}
		}
	})

	b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{
				Command:     "kiuas",
				Description: "Näytä saunan tila",
			},
		},
	})

	return b, nil

}

func SendTelegramMessage(b *bot.Bot, ctx context.Context, message string, chatID ...int64) {
	var targetChatID int64
	var err error

	if len(chatID) > 0 {
		targetChatID = chatID[0]
	} else {
		targetChatID, err = strconv.ParseInt(os.Getenv("NOTIFICATION_CHAT_ID"), 10, 64)
		if err != nil {
			log.Fatalf("Error parsing NOTIFICATION_CHAT_ID: %v", err)
		}
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: targetChatID,
		Text:   message,
	})
	if err != nil {
		fmt.Printf("Failed to send message: %v\n", err)
	}
}
