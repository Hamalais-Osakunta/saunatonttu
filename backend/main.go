package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
	"github.com/peterhellberg/ruuvitag"
)

// Kiuas struct to track temperature, humidity, and voltage
type Kiuas struct {
	Temperature float64
	Humidity    float64
	Battery     uint16
}

var kiuas Kiuas

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Initialize Telegram bot
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatalf("TELEGRAM_BOT_TOKEN is not set in the environment")
	}
	b, err := initializeTelegramBot(botToken)
	if err != nil {
		log.Fatalf("Failed to initialize Telegram bot: %v", err)
	}

	// Start Telegram bot
	go b.Start(ctx)

	// Start HTTP server
	go startHTTPServer(b, ctx)

	// Wait for context cancellation (e.g., Ctrl+C)
	<-ctx.Done()
	fmt.Println("Shutting down...")
}

func initializeTelegramBot(token string) (*bot.Bot, error) {
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

	// Register /temperature command handler
	b.RegisterHandler(bot.HandlerTypeMessageText, "/temperature", bot.MatchTypeExact, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Temperature: %.1f °C, Humidity: %.1f%%", kiuas.Temperature, kiuas.Humidity),
		})
		if err != nil {
			fmt.Printf("Failed to send message: %v\n", err)
		}
	})

	// Register /info command handler for maintenance chat only
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

func startHTTPServer(b *bot.Bot, ctx context.Context) {
	http.HandleFunc("/api/receive-bt", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		ruuviTag, err := ruuvitag.ParseRAWv2(body)
		if err != nil {
			fmt.Println("Failed to parse RuuviTag data:", err)
			return
		}

		kiuas.Temperature = ruuviTag.Temperature
		kiuas.Humidity = ruuviTag.Humidity
		kiuas.Battery = ruuviTag.Battery
		fmt.Printf("Received new temperature value: %.1f °C, Humidity: %.1f%%, Voltage: %.2f V\n", kiuas.Temperature, kiuas.Humidity, kiuas.Battery)

		checkAndNotify(b, ctx)
	})

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "1337" // Default port
	}

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func checkAndNotify(b *bot.Bot, ctx context.Context) {
	warmingThreshold, err := strconv.ParseFloat(os.Getenv("SAUNA_WARMING_THRESHOLD"), 64)
	if err != nil {
		log.Fatalf("Error parsing SAUNA_WARMING_THRESHOLD: %v", err)
	}

	readyThreshold, err := strconv.ParseFloat(os.Getenv("SAUNA_READY_THRESHOLD"), 64)
	if err != nil {
		log.Fatalf("Error parsing SAUNA_READY_THRESHOLD: %v", err)
	}

	if kiuas.Temperature >= readyThreshold {
		sendTelegramMessage(b, ctx, fmt.Sprintf("Sauna valmis!\nLämpötila: %.1f °C", kiuas.Temperature))
	} else if kiuas.Temperature >= warmingThreshold {
		sendTelegramMessage(b, ctx, "🔥 Sauna lämpiää")
	}
}

func sendTelegramMessage(b *bot.Bot, ctx context.Context, message string) {
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
