package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/go-telegram/bot"
	"github.com/joho/godotenv"
	"github.com/peterhellberg/ruuvitag"
)

var saunaKiuas Kiuas
var lastDataReceived time.Time

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatalf("TELEGRAM_BOT_TOKEN is not set in the environment")
	}
	b, err := InitializeTelegramBot(botToken, &saunaKiuas)
	if err != nil {
		log.Fatalf("Failed to initialize Telegram bot: %v", err)
	}

	go b.Start(ctx)

	go startHTTPServer(b, ctx)

	go monitorDataReception(b, ctx)

	<-ctx.Done()
	fmt.Println("Shutting down...")
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
			fmt.Println("Failed to parse RuuviTag data. are all the sensors enabled?", err)
		}

		saunaKiuas.Temperature = ruuviTag.Temperature
		saunaKiuas.Humidity = ruuviTag.Humidity
		saunaKiuas.Battery = ruuviTag.Battery
		fmt.Printf("Received new temperature value: %.1f °C, Humidity: %.1f%%, Voltage: %d V\n", saunaKiuas.Temperature, saunaKiuas.Humidity, saunaKiuas.Battery)

		lastDataReceived = time.Now()

		checkAndNotify(b, ctx)
	})

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "1337"
	}

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func monitorDataReception(b *bot.Bot, ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            if time.Since(lastDataReceived) > 10*time.Minute {
                maintenanceChatID, err := strconv.ParseInt(os.Getenv("MAINTENANCE_CHAT_ID"), 10, 64)
                if err != nil {
                    log.Fatalf("Error parsing MAINTENANCE_CHAT_ID: %v", err)
                }
                SendTelegramMessage(b, ctx, "No data received for over 10 minutes", maintenanceChatID)
            }
        case <-ctx.Done():
            return
        }
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

	if saunaKiuas.Temperature >= readyThreshold {
		if !saunaKiuas.ReadyNotificationSent {
			SendTelegramMessage(b, ctx, fmt.Sprintf("Sauna valmis!\nLämpötila: %.1f °C", saunaKiuas.Temperature))
			saunaKiuas.ReadyNotificationSent = true
		}
	} else if saunaKiuas.Temperature >= warmingThreshold {
		if !saunaKiuas.WarmingNotificationSent && !saunaKiuas.ReadyNotificationSent {
			SendTelegramMessage(b, ctx, "🔥 Sauna lämpiää")
			saunaKiuas.WarmingNotificationSent = true
		}
	} else {
		// Reset notifications if temperature drops below warming threshold
		saunaKiuas.ResetNotifications()
	}
}
