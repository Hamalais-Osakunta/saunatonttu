package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type MockTelegramBot struct {
	SentMessages []string
}

func (m *MockTelegramBot) SendMessage(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error) {
	m.SentMessages = append(m.SentMessages, params.Text)
	return &models.Message{}, nil
}

func (m *MockTelegramBot) RegisterHandler(handlerType bot.HandlerType, pattern string, matchType bot.MatchType, handler bot.HandlerFunc) {
	// No-op for testing
}

func (m *MockTelegramBot) Start(ctx context.Context) {
	// No-op for testing
}

func (m *MockTelegramBot) SetMyCommands(ctx context.Context, params *bot.SetMyCommandsParams) error {
	// No-op for testing
	return nil
}

func TestAddTemperatureRecord(t *testing.T) {
	kiuas := &Kiuas{}
	now := time.Now()
	kiuas.AddTemperatureRecord(20.0, now)
	kiuas.AddTemperatureRecord(21.0, now.Add(1*time.Minute))
	kiuas.AddTemperatureRecord(22.0, now.Add(2*time.Minute))

	expectedTemps := [3]float64{20.0, 21.0, 22.0}
	if kiuas.TemperatureRecords != expectedTemps {
		t.Errorf("Expected %v, got %v", expectedTemps, kiuas.TemperatureRecords)
	}
}

func TestCheckAndNotify_SaunaReady(t *testing.T) {
	kiuas := &Kiuas{
		Temperature: 80.0,
		TemperatureRecords: [3]float64{
			70.0, 75.0, 80.0,
		},
		TimestampRecords: [3]time.Time{
			time.Now().Add(-3 * time.Minute),
			time.Now().Add(-2 * time.Minute),
			time.Now().Add(-1 * time.Minute),
		},
	}

	mockBot := &MockTelegramBot{}

	config := &Config{
		ReadyThreshold: 75.0,
		LowerBound:     0.01,
		ResetThreshold: 40.0,
	}

	currentTime := time.Now()

	checkAndNotify(mockBot, context.Background(), kiuas, config, currentTime)

	if !kiuas.ReadyNotificationSent {
		t.Errorf("Expected ReadyNotificationSent to be true")
	}
	if len(mockBot.SentMessages) != 1 {
		t.Fatalf("Expected 1 message to be sent, got %d", len(mockBot.SentMessages))
	}
	expectedMessage := fmt.Sprintf("*Sauna valmis!*🔥\nLämpötila: %.1f °C 🌡️", kiuas.Temperature)
	// replace . wthi \. to escape the dot
	expectedMessage = strings.Replace(expectedMessage, ".", "\\.", -1)
	if mockBot.SentMessages[0] != expectedMessage {
		t.Errorf("Expected message: %s, got: %s", expectedMessage, mockBot.SentMessages[0])
	}
}

func TestCheckAndNotify_SaunaWarming(t *testing.T) {
	currentTime := time.Now()
	kiuas := &Kiuas{
		Temperature: 60.0,
		TemperatureRecords: [3]float64{
			55.0, 57.5, 60.0,
		},
		TimestampRecords: [3]time.Time{
			currentTime.Add(-6 * time.Minute),
			currentTime.Add(-3 * time.Minute),
			currentTime,
		},
	}

	mockBot := &MockTelegramBot{}

	config := &Config{
		ReadyThreshold: 75.0,
		LowerBound:     0.01,
		ResetThreshold: 40.0,
	}

	checkAndNotify(mockBot, context.Background(), kiuas, config, currentTime)

	if !kiuas.WarmingNotificationSent {
		t.Errorf("Expected WarmingNotificationSent to be true")
	}
	if len(mockBot.SentMessages) != 1 {
		t.Fatalf("Expected 1 message to be sent, got %d", len(mockBot.SentMessages))
	}

	// Calculate the expected estimated ready time
	timeDiff := kiuas.TimestampRecords[2].Sub(kiuas.TimestampRecords[0]).Seconds()
	tempChange := kiuas.TemperatureRecords[2] - kiuas.TemperatureRecords[0]
	tempChangeRate := tempChange / timeDiff
	if tempChangeRate <= 0 {
		t.Fatalf("Temperature change rate should be positive")
	}
	tempRemaining := config.ReadyThreshold - kiuas.Temperature
	timeToReadySeconds := tempRemaining / tempChangeRate
	estimatedReadyTime := currentTime.Add(time.Duration(timeToReadySeconds) * time.Second)
	estimatedReadyTimeStr := estimatedReadyTime.Format("15:04")

	expectedMessage := fmt.Sprintf("🔥*Sauna lämpiää!*🔥\nValmis klo %s", estimatedReadyTimeStr)
	if mockBot.SentMessages[0] != expectedMessage {
		t.Errorf("Expected message: %s, got: %s", expectedMessage, mockBot.SentMessages[0])
	}
}

func TestCheckAndNotify_NoNotification(t *testing.T) {
	currentTime := time.Now()
	kiuas := &Kiuas{
		Temperature: 30.0,
		TemperatureRecords: [3]float64{
			30.0, 30.0, 30.0,
		},
		TimestampRecords: [3]time.Time{
			currentTime.Add(-3 * time.Minute),
			currentTime.Add(-2 * time.Minute),
			currentTime.Add(-1 * time.Minute),
		},
	}

	mockBot := &MockTelegramBot{}

	config := &Config{
		ReadyThreshold: 75.0,
		LowerBound:     0.01,
		ResetThreshold: 40.0,
	}

	checkAndNotify(mockBot, context.Background(), kiuas, config, currentTime)

	if kiuas.WarmingNotificationSent || kiuas.ReadyNotificationSent {
		t.Errorf("No notifications should be sent")
	}
	if len(mockBot.SentMessages) != 0 {
		t.Fatalf("Expected 0 messages to be sent, got %d", len(mockBot.SentMessages))
	}
}

func TestCheckAndNotify_WarmingNotificationSentOnlyOnce(t *testing.T) {
	currentTime := time.Now()
	kiuas := &Kiuas{
		Temperature: 60.0,
		TemperatureRecords: [3]float64{
			55.0, 57.5, 60.0,
		},
		TimestampRecords: [3]time.Time{
			currentTime.Add(-6 * time.Minute),
			currentTime.Add(-3 * time.Minute),
			currentTime,
		},
	}

	mockBot := &MockTelegramBot{}

	config := &Config{
		ReadyThreshold: 75.0,
		LowerBound:     0.01,
		ResetThreshold: 40.0,
	}

	ctx := context.Background()

	// Call checkAndNotify multiple times
	for i := 0; i < 5; i++ {
		kiuas.Temperature += 2.0
		kiuas.AddTemperatureRecord(kiuas.Temperature, currentTime.Add(time.Duration(i)*time.Minute))
		checkAndNotify(mockBot, ctx, kiuas, config, currentTime.Add(time.Duration(i)*time.Minute))
	}

	if !kiuas.WarmingNotificationSent {
		t.Errorf("Expected WarmingNotificationSent to be true")
	}
	if len(mockBot.SentMessages) != 1 {
		t.Fatalf("Expected 1 message to be sent, got %d", len(mockBot.SentMessages))
	}
}

func TestCheckAndNotify_ResetNotifications(t *testing.T) {
	currentTime := time.Now()
	kiuas := &Kiuas{
		Temperature: 80.0,
		TemperatureRecords: [3]float64{
			70.0, 75.0, 80.0,
		},
		TimestampRecords: [3]time.Time{
			currentTime.Add(-3 * time.Minute),
			currentTime.Add(-2 * time.Minute),
			currentTime.Add(-1 * time.Minute),
		},
		WarmingNotificationSent: true,
		ReadyNotificationSent:   true,
	}

	mockBot := &MockTelegramBot{}

	config := &Config{
		ReadyThreshold: 75.0,
		LowerBound:     0.01,
		ResetThreshold: 40.0,
	}

	ctx := context.Background()

	// Simulate temperature dropping below ResetThreshold
	kiuas.Temperature = 35.0
	kiuas.AddTemperatureRecord(kiuas.Temperature, currentTime)

	checkAndNotify(mockBot, ctx, kiuas, config, currentTime)

	if kiuas.WarmingNotificationSent || kiuas.ReadyNotificationSent {
		t.Errorf("Expected notifications to be reset")
	}

	if len(mockBot.SentMessages) != 0 {
		t.Fatalf("Expected 0 new messages to be sent during reset, got %d", len(mockBot.SentMessages))
	}
}

func TestCheckAndNotify_WarmingStoppedBeforeReady_ResetNotifications(t *testing.T) {
	currentTime := time.Now()
	kiuas := &Kiuas{
		Temperature: 60.0,
		TemperatureRecords: [3]float64{
			55.0, 57.5, 60.0,
		},
		TimestampRecords: [3]time.Time{
			currentTime.Add(-6 * time.Minute),
			currentTime.Add(-3 * time.Minute),
			currentTime,
		},
		WarmingNotificationSent: true,
	}

	mockBot := &MockTelegramBot{}

	config := &Config{
		ReadyThreshold: 75.0,
		LowerBound:     0.01,
		ResetThreshold: 40.0,
	}

	ctx := context.Background()

	// Simulate temperature dropping before reaching ReadyThreshold
	kiuas.Temperature = 35.0
	kiuas.AddTemperatureRecord(kiuas.Temperature, currentTime)

	checkAndNotify(mockBot, ctx, kiuas, config, currentTime)

	if kiuas.WarmingNotificationSent && kiuas.ReadyNotificationSent {
		t.Errorf("Expected notifications to be reset")
	}

	if len(mockBot.SentMessages) != 0 {
		t.Fatalf("Expected 0 new messages to be sent during reset, got %d", len(mockBot.SentMessages))
	}
}
