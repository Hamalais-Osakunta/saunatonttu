package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"os"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func InitializeTelegramBot(token string, kiuas *Kiuas) (*bot.Bot, error) {
	maintenanceChatID, err := strconv.ParseInt(os.Getenv("MAINTENANCE_CHAT_ID"), 10, 64)
	if err != nil {
		log.Fatalf("Error parsing MAINTENANCE_CHAT_ID: %v", err)
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   update.Message.Text,
			})
		}),
	}

	b, err := bot.New(token, opts...)
	if err != nil {
		return nil, err
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/kiuas", bot.MatchTypeExact, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Sauna on %s\nLämpötila: %.1f °C\nKosteus: %.1f%%", GetSaunaStatus(kiuas.IsOn()), kiuas.Temperature, kiuas.Humidity),
		})
		if err != nil {
			fmt.Printf("Failed to send message: %v\n", err)
		}
	})

	b.RegisterHandler(bot.HandlerTypeMessageText, "/info", bot.MatchTypeExact, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		if update.Message.Chat.ID == maintenanceChatID {
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   fmt.Sprintf("Temperature: %.1f °C\nHumidity: %.1f%%\nBattery: %d V", kiuas.Temperature, kiuas.Humidity, kiuas.Battery),
			})
			if err != nil {
				fmt.Printf("Failed to send message: %v\n", err)
			}
		}
	})

	return b, nil
}

func SendTelegramMessage(b *bot.Bot, ctx context.Context, message string) {
	chatID, err := strconv.ParseInt(os.Getenv("NOTIFICATION_CHAT_ID"), 10, 64)
	if err != nil {
		log.Fatalf("Error parsing NOTIFICATION_CHAT_ID: %v", err)
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   message,
	})
	if err != nil {
		fmt.Printf("Failed to send message: %v\n", err)
	}
}
