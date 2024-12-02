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
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
	"github.com/peterhellberg/ruuvitag"
)

import _ "time/tzdata"

type Kiuas struct {
	Temperature             float64
	Humidity                float64
	Battery                 uint16
	WarmingNotificationSent bool
	ReadyNotificationSent   bool
	LastDataReceived        time.Time
	TemperatureRecords      [3]float64
	TimestampRecords        [3]time.Time
	WarmingStartTime        [3]time.Time
}

func (k *Kiuas) IsOn(config *Config) bool {
	return k.Temperature >= config.ReadyThreshold || k.IsWarming(config)
}

// Calculate the rate of temperature change in degrees per second
func (k *Kiuas) tempChangeRate() float64 {
	// Ensure there are at least three valid records
	if k.TemperatureRecords[0] == 0 {
		log.Println("Not enough valid temperature records")
		return 0
	}

	// Calculate the time difference and average temperature change over the last three records
	timeDiff := k.TimestampRecords[2].Sub(k.TimestampRecords[0]).Seconds()
	if timeDiff == 0 {
		// Avoid division by zero if the timestamps are identical (unlikely but possible)
		return 0
	}

	// Calculate the temperature change over the three records
	tempChange := k.TemperatureRecords[2] - k.TemperatureRecords[0]
	tempChangeRate := tempChange / timeDiff // degrees per second

	// Check if the temperature change rate is within the acceptable range
	if tempChangeRate <= 0 {
		log.Println("Temperature change rate should be positive")
		return 0
	}

	return tempChangeRate

}

// Check if the sauna is warming up
func (k *Kiuas) IsWarming(config *Config) bool {
	tempChangeRate := k.tempChangeRate()

	return tempChangeRate > 0 && tempChangeRate >= config.LowerBound && k.Temperature < config.ReadyThreshold

}

func (k *Kiuas) getEstimateReadySeconds(config *Config) float64 {
	// Calculate the estimated time until the sauna is ready
	tempRemaining := config.ReadyThreshold - k.Temperature
	timeToReadySeconds := tempRemaining / k.tempChangeRate()

	return timeToReadySeconds
}

// Shift the old values and add a new temperature record
func (k *Kiuas) AddTemperatureRecord(newTemp float64, newTime time.Time) {
	// Shift the array to remove the oldest record and add the new one
	k.TemperatureRecords[0] = k.TemperatureRecords[1]
	k.TemperatureRecords[1] = k.TemperatureRecords[2]
	k.TemperatureRecords[2] = newTemp

	// Similarly update the timestamps
	k.TimestampRecords[0] = k.TimestampRecords[1]
	k.TimestampRecords[1] = k.TimestampRecords[2]
	k.TimestampRecords[2] = newTime
}

func GetSaunaStatus(isOn bool) string {
	if isOn {
		return "päällä"
	}
	return "pois päältä"
}

func (k *Kiuas) ResetNotifications() {
	k.WarmingNotificationSent = false
	k.ReadyNotificationSent = false
}

type TelegramBot interface {
	SendMessage(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error)
	RegisterHandler(handlerType bot.HandlerType, pattern string, matchType bot.MatchType, handler bot.HandlerFunc)
	Start(ctx context.Context)
	SetMyCommands(ctx context.Context, params *bot.SetMyCommandsParams) error
}

type BotWrapper struct {
	Bot *bot.Bot
}

func (b *BotWrapper) SendMessage(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error) {
	return b.Bot.SendMessage(ctx, params)
}

func (b *BotWrapper) RegisterHandler(handlerType bot.HandlerType, pattern string, matchType bot.MatchType, handler bot.HandlerFunc) {
	b.Bot.RegisterHandler(handlerType, pattern, matchType, handler)
}

func (b *BotWrapper) Start(ctx context.Context) {
	b.Bot.Start(ctx)
}

func (b *BotWrapper) SetMyCommands(ctx context.Context, params *bot.SetMyCommandsParams) error {
	_, err := b.Bot.SetMyCommands(ctx, params)
	return err
}

type Config struct {
	ReadyThreshold     float64
	ChangeThreshold    float64
	LowerBound         float64
	ResetThreshold     float64
	MaintenanceChatID  int64
	NotificationChatID int64
	ServerPort         string
	TelegramBotToken   string
}

func InitializeTelegramBot(ctx context.Context, token string, kiuas *Kiuas, config *Config) (TelegramBot, error) {
	opts := []bot.Option{}

	botInstance, err := bot.New(token, opts...)
	if err != nil {
		return nil, err
	}

	botWrapper := &BotWrapper{Bot: botInstance}

	botWrapper.RegisterHandler(bot.HandlerTypeMessageText, "/kiuas", bot.MatchTypePrefix, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		_, err := botWrapper.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Sauna on %s\nLämpötila: %.1f °C\nKosteus: %.1f%%", GetSaunaStatus(kiuas.IsOn(config)), kiuas.Temperature, kiuas.Humidity),
		})
		if err != nil {
			fmt.Printf("Failed to send message: %v\n", err)
		}
	})

	botWrapper.RegisterHandler(bot.HandlerTypeMessageText, "/info", bot.MatchTypePrefix, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		if update.Message.Chat.ID == config.MaintenanceChatID {
			loc, err := time.LoadLocation("Europe/Bucharest")
			if err != nil {
				fmt.Printf("Error loading location: %v", err)
			}
			_, err = botWrapper.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text: fmt.Sprintf(
					"Sauna Info:\nTemperature: %.1f °C\nHumidity: %.1f%%\nBattery: %d V\nLast Data Received: %s",
					kiuas.Temperature,
					kiuas.Humidity,
					kiuas.Battery,
					kiuas.LastDataReceived.In(loc))})
			if err != nil {
				fmt.Printf("Failed to send message: %v\n", err)
			}
		}
	})

	err = botWrapper.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{
				Command:     "kiuas",
				Description: "Näytä saunan tila",
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return botWrapper, nil
}

func FmtTelegram(input string) string {
	return strings.NewReplacer(
	  ".", "\\.",
	).Replace(input)
  }

func SendTelegramMessage(b TelegramBot, ctx context.Context, config *Config, message string, chatID ...int64) {
	var targetChatID int64

	if len(chatID) > 0 {
		targetChatID = chatID[0]
	} else {
		targetChatID = config.NotificationChatID
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    targetChatID,
		Text:      FmtTelegram(message),
		ParseMode: "MarkdownV2",
	})
	if err != nil {
		fmt.Printf("Failed to send message: %v\n", err)
	}
}

func main() {
	os.Setenv("TZ", "Europe/Bucharest")

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

	readyThreshold, err := strconv.ParseFloat(os.Getenv("SAUNA_READY_THRESHOLD"), 64)
	if err != nil {
		log.Fatalf("Error parsing SAUNA_READY_THRESHOLD: %v", err)
	}

	maintenanceChatID, err := strconv.ParseInt(os.Getenv("MAINTENANCE_CHAT_ID"), 10, 64)
	if err != nil {
		log.Fatalf("Error parsing MAINTENANCE_CHAT_ID: %v", err)
	}

	notificationChatID, err := strconv.ParseInt(os.Getenv("NOTIFICATION_CHAT_ID"), 10, 64)
	if err != nil {
		log.Fatalf("Error parsing NOTIFICATION_CHAT_ID: %v", err)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "1337"
	}

	config := &Config{
		ReadyThreshold:     readyThreshold,
		ChangeThreshold:    0.0123,
		LowerBound:         0.0123 * 0.9,
		ResetThreshold:     40.0,
		MaintenanceChatID:  maintenanceChatID,
		NotificationChatID: notificationChatID,
		ServerPort:         port,
		TelegramBotToken:   botToken,
	}

	kiuas := &Kiuas{
		TemperatureRecords: [3]float64{0.0, 0.0, 0.0},
		TimestampRecords:   [3]time.Time{time.Now(), time.Now(), time.Now()},
	}

	botInstance, err := InitializeTelegramBot(ctx, botToken, kiuas, config)
	if err != nil {
		log.Fatalf("Failed to initialize Telegram bot: %v", err)
	}

	go botInstance.Start(ctx)

	go startHTTPServer(botInstance, ctx, kiuas, config)

	go monitorDataReception(botInstance, ctx, kiuas, config)

	<-ctx.Done()
	fmt.Println("Shutting down...")
}

func startHTTPServer(b TelegramBot, ctx context.Context, kiuas *Kiuas, config *Config) {
	http.HandleFunc("/api/receive-bt", func(w http.ResponseWriter, r *http.Request) {
		handleReceiveBT(w, r, b, ctx, kiuas, config)
	})

	if err := http.ListenAndServe(":"+config.ServerPort, nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func handleReceiveBT(w http.ResponseWriter, r *http.Request, b TelegramBot, ctx context.Context, kiuas *Kiuas, config *Config) {
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
		fmt.Println("Failed to parse RuuviTag data. Are all the sensors enabled?", err)
	}

	kiuas.Temperature = ruuviTag.Temperature
	kiuas.Humidity = ruuviTag.Humidity
	kiuas.Battery = ruuviTag.Battery
	fmt.Printf("Received new temperature value: %.1f °C, Humidity: %.1f%%, Voltage: %d V\n", kiuas.Temperature, kiuas.Humidity, kiuas.Battery)

	kiuas.LastDataReceived = time.Now()
	kiuas.AddTemperatureRecord(kiuas.Temperature, time.Now())

	checkAndNotify(b, ctx, kiuas, config, time.Now())
}

func monitorDataReception(b TelegramBot, ctx context.Context, kiuas *Kiuas, config *Config) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	notificationSent := false

	for {
		select {
		case <-ticker.C:
			if time.Since(kiuas.LastDataReceived) > time.Hour && !notificationSent {
				SendTelegramMessage(b, ctx, config, "No data received for over 1 hour", config.MaintenanceChatID)
				notificationSent = true
			} else if time.Since(kiuas.LastDataReceived) <= time.Hour {
				notificationSent = false
			}
		case <-ctx.Done():
			return
		}
	}
}

// Function to check temperature change and send notifications
func checkAndNotify(b TelegramBot, ctx context.Context, kiuas *Kiuas, config *Config, currentTime time.Time) {

	// Ready notification check
	if kiuas.Temperature >= config.ReadyThreshold {
		if !kiuas.ReadyNotificationSent {
			SendTelegramMessage(b, ctx, config, fmt.Sprintf("*Sauna valmis\\!*🔥\nLämpötila: %.1f °C 🌡️", kiuas.Temperature))
			kiuas.ReadyNotificationSent = true
		}
	} else if !kiuas.WarmingNotificationSent && !kiuas.ReadyNotificationSent {
		if kiuas.IsWarming(config) {
			// Check if warming started, if not, initialize warming start time
			if kiuas.WarmingStartTime.IsZero() {
				kiuas.WarmingStartTime = currentTime
			}
			timeToReadySeconds := kiuas.getEstimateReadySeconds(config)
			estimatedReadyTime := currentTime.Add(time.Duration(timeToReadySeconds) * time.Second)
			fmt.Printf("Estimated ready time: %s\n", estimatedReadyTime)
			fmt.Printf("Current time: %s\n", currentTime)

			// Format the estimated ready time
			estimatedReadyTimeStr := estimatedReadyTime.Format("15:04")
			fmt.Printf("Estimated ready time string: %s\n", estimatedReadyTimeStr)

			SendTelegramMessage(b, ctx, config, fmt.Sprintf("🔥*Sauna lämpiää\\!*🔥\nValmis klo %s", estimatedReadyTimeStr))
			kiuas.WarmingNotificationSent = true
		}
	}

	// Check if 2 hours have passed since warming started and temperature is below ReadyThreshold
	// Reset notifications if temperature has cooled down
	if !kiuas.ReadyNotificationSent && !kiuas.WarmingStartTime.IsZero() {
		if currentTime.Sub(kiuas.WarmingStartTime) > 2*time.Hour {

			SendTelegramMessage(b, ctx, config, "⚠️ *Sauna ei saavuttanut tavoitelämpötilaa kahdessa tunnissa\\!* Tarkista kiuas.")
			// Reset notifications and warming start time
			kiuas.ResetNotifications()
			kiuas.WarmingStartTime = time.Time{}

		}
	}

	// Reset notifications if temperature has cooled down
	if kiuas.Temperature < config.ResetThreshold {
		if kiuas.WarmingNotificationSent && kiuas.ReadyNotificationSent {
			kiuas.ResetNotifications()
			kiuas.WarmingStartTime = time.Time{} // Ensure warming start time is reset
		}
	}
}
