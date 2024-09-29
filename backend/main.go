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

var saunaKiuas  = Kiuas{
	TemperatureRecords:  [3]float64{0.0, 0.0, 0.0},
	TimestampRecords:    [3]time.Time{time.Now(), time.Now(), time.Now()},
}

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
	b, err := InitializeTelegramBot(ctx, botToken, &saunaKiuas)
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

		saunaKiuas.lastDataReceived = time.Now()
		// Update the temperature records with the latest temperature and timestamp
		saunaKiuas.AddTemperatureRecord(saunaKiuas.Temperature, time.Now())

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

	notificationSent := false

	for {
		select {
		case <-ticker.C:
			if time.Since(saunaKiuas.lastDataReceived) > time.Hour && !notificationSent {
				maintenanceChatID, err := strconv.ParseInt(os.Getenv("MAINTENANCE_CHAT_ID"), 10, 64)
				if err != nil {
					log.Fatalf("Error parsing MAINTENANCE_CHAT_ID: %v", err)
				}
				SendTelegramMessage(b, ctx, "No data received for over 1 hour", maintenanceChatID)
				notificationSent = true
			} else if time.Since(saunaKiuas.lastDataReceived) <= time.Hour {
				notificationSent = false
			}
		case <-ctx.Done():
			return
		}
	}
}

// Function to check temperature change and send notifications
func checkAndNotify(b *bot.Bot, ctx context.Context) {
	readyThreshold, err := strconv.ParseFloat(os.Getenv("SAUNA_READY_THRESHOLD"), 64)
	if err != nil {
		log.Fatalf("Error parsing SAUNA_READY_THRESHOLD: %v", err)
	}

	// Ensure there are at least three valid records
	if saunaKiuas.TemperatureRecords[0] == 0 {
		log.Println("Not enough valid timestamp records")
		return
	}

	// Calculate the time difference and average temperature change over the last three records
	timeDiff := saunaKiuas.TimestampRecords[2].Sub(saunaKiuas.TimestampRecords[0]).Seconds()
	if timeDiff == 0 {
		// Avoid division by zero if the timestamps are identical (unlikely but possible)
		return
	}

	// Calculate the temperature change over the three records
	tempChange := saunaKiuas.TemperatureRecords[2] - saunaKiuas.TemperatureRecords[0]
	tempChangeRate := tempChange / timeDiff

	// Threshold for change rate considered as sauna warming up
	changeThreshold := 0.0123 // avg temperature change / second

	// Introduce lower bound for the change rate to catch the case where the temperature is rising slowly
	lowerBound := changeThreshold * 0.9

	// Ready notification check
	if saunaKiuas.Temperature >= readyThreshold {
		if !saunaKiuas.ReadyNotificationSent {
			SendTelegramMessage(b, ctx, fmt.Sprintf("*Sauna valmis\!*🔥\nLämpötila: %.1f °C 🌡️", saunaKiuas.Temperature))
			saunaKiuas.ReadyNotificationSent = true
		}
	} else if !saunaKiuas.WarmingNotificationSent && !saunaKiuas.ReadyNotificationSent {
			if tempChangeRate >= lowerBound {
				SendTelegramMessage(b, ctx, "🔥*Sauna lämpiää\!*🔥")
				saunaKiuas.WarmingNotificationSent = true
			}
	} else {
		// Reset notifications if temperature drops below warming threshold
		saunaKiuas.ResetNotifications()
	}
}
