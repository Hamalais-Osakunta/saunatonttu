package main

import (
	"context"
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
		ReadyThreshold:  75.0,
		ChangeThreshold: 0.0123,
		LowerBound:      0.0123 * 0.9,
	}

	checkAndNotify(mockBot, context.Background(), kiuas, config)

	if !kiuas.ReadyNotificationSent {
		t.Errorf("Expected ReadyNotificationSent to be true")
	}
	if len(mockBot.SentMessages) != 1 {
		t.Fatalf("Expected 1 message to be sent, got %d", len(mockBot.SentMessages))
	}
	expectedMessage := "*Sauna valmis\\!*🔥\nLämpötila: 80.0 °C 🌡️"
	if mockBot.SentMessages[0] != expectedMessage {
		t.Errorf("Expected message: %s, got: %s", expectedMessage, mockBot.SentMessages[0])
	}
}

func TestCheckAndNotify_SaunaWarming(t *testing.T) {
	kiuas := &Kiuas{
		Temperature: 60.0,
		TemperatureRecords: [3]float64{
			55.0, 57.5, 60.0,
		},
		TimestampRecords: [3]time.Time{
			time.Now().Add(-3 * time.Minute),
			time.Now().Add(-2 * time.Minute),
			time.Now().Add(-1 * time.Minute),
		},
	}

	mockBot := &MockTelegramBot{}

	config := &Config{
		ReadyThreshold:  75.0,
		ChangeThreshold: 0.0123,
		LowerBound:      0.0123 * 0.9,
	}

	checkAndNotify(mockBot, context.Background(), kiuas, config)

	if !kiuas.WarmingNotificationSent {
		t.Errorf("Expected WarmingNotificationSent to be true")
	}
	if len(mockBot.SentMessages) != 1 {
		t.Fatalf("Expected 1 message to be sent, got %d", len(mockBot.SentMessages))
	}
	expectedMessage := "🔥*Sauna lämpiää\\!*🔥"
	if mockBot.SentMessages[0] != expectedMessage {
		t.Errorf("Expected message: %s, got: %s", expectedMessage, mockBot.SentMessages[0])
	}
}

func TestCheckAndNotify_NoNotification(t *testing.T) {
	kiuas := &Kiuas{
		Temperature: 30.0,
		TemperatureRecords: [3]float64{
			30.0, 30.0, 30.0,
		},
		TimestampRecords: [3]time.Time{
			time.Now().Add(-3 * time.Minute),
			time.Now().Add(-2 * time.Minute),
			time.Now().Add(-1 * time.Minute),
		},
	}

	mockBot := &MockTelegramBot{}

	config := &Config{
		ReadyThreshold:  75.0,
		ChangeThreshold: 0.0123,
		LowerBound:      0.0123 * 0.9,
	}

	checkAndNotify(mockBot, context.Background(), kiuas, config)

	if kiuas.WarmingNotificationSent || kiuas.ReadyNotificationSent {
		t.Errorf("No notifications should be sent")
	}
	if len(mockBot.SentMessages) != 0 {
		t.Fatalf("Expected 0 messages to be sent, got %d", len(mockBot.SentMessages))
	}
}

func TestCheckAndNotify_WarmingNotificationSentOnlyOnce(t *testing.T) {
	kiuas := &Kiuas{
		Temperature: 60.0,
		TemperatureRecords: [3]float64{
			55.0, 57.5, 60.0,
		},
		TimestampRecords: [3]time.Time{
			time.Now().Add(-3 * time.Minute),
			time.Now().Add(-2 * time.Minute),
			time.Now().Add(-1 * time.Minute),
		},
	}

	mockBot := &MockTelegramBot{}

	config := &Config{
		ReadyThreshold:  75.0,
		ChangeThreshold: 0.0123,
		LowerBound:      0.0123 * 0.9,
		ResetThreshold:  40.0,
	}

	ctx := context.Background()

	// Call checkAndNotify multiple times and increase the temperature
	for i := 0; i < 5; i++ {
		kiuas.Temperature += 2.0
		kiuas.AddTemperatureRecord(kiuas.Temperature, time.Now().Add(time.Duration(i)*time.Minute))
		checkAndNotify(mockBot, ctx, kiuas, config)
	}

	// Warming notification should only be sent once
	if !kiuas.WarmingNotificationSent {
		t.Errorf("Expected WarmingNotificationSent to be true")
	}
	if len(mockBot.SentMessages) != 1 {
		t.Fatalf("Expected 1 message to be sent, got %d", len(mockBot.SentMessages))
	}
	expectedMessage := "🔥*Sauna lämpiää\\!*🔥"
	if mockBot.SentMessages[0] != expectedMessage {
		t.Errorf("Expected message: %s, got: %s", expectedMessage, mockBot.SentMessages[0])
	}
}

func TestCheckAndNotify_ResetNotifications(t *testing.T) {
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
		ReadyThreshold:  75.0,
		ChangeThreshold: 0.0123,
		LowerBound:      0.0123 * 0.9,
		ResetThreshold:  40.0,
	}

	ctx := context.Background()

	// First, send ready notification
	checkAndNotify(mockBot, ctx, kiuas, config)
	if !kiuas.ReadyNotificationSent {
		t.Errorf("Expected ReadyNotificationSent to be true")
	}

	// Simulate temperature dropping below ResetThreshold
	kiuas.Temperature = 35.0
	kiuas.AddTemperatureRecord(35.0, time.Now())

	checkAndNotify(mockBot, ctx, kiuas, config)
	if kiuas.WarmingNotificationSent || kiuas.ReadyNotificationSent {
		t.Errorf("Expected notifications to be reset")
	}

	// No new messages should have been sent during reset
	if len(mockBot.SentMessages) != 1 {
		t.Fatalf("Expected 1 message to be sent, got %d", len(mockBot.SentMessages))
	}
}

func TestCheckAndNotify_WarmingStoppedBeforeReady_ResetNotifications(t *testing.T) {
    kiuas := &Kiuas{
        Temperature: 60.0,
        TemperatureRecords: [3]float64{
            55.0, 57.5, 60.0,
        },
        TimestampRecords: [3]time.Time{
            time.Now().Add(-3 * time.Minute),
            time.Now().Add(-2 * time.Minute),
            time.Now().Add(-1 * time.Minute),
        },
    }

    mockBot := &MockTelegramBot{}

    config := &Config{
        ReadyThreshold:  75.0,
        ChangeThreshold: 0.0123,
        LowerBound:      0.0123 * 0.9,
        ResetThreshold:  40.0,
    }

    ctx := context.Background()

    // First, send warming notification
    checkAndNotify(mockBot, ctx, kiuas, config)
    if !kiuas.WarmingNotificationSent {
        t.Errorf("Expected WarmingNotificationSent to be true")
    }
    if len(mockBot.SentMessages) != 1 {
        t.Fatalf("Expected 1 message to be sent, got %d", len(mockBot.SentMessages))
    }
    expectedWarmingMessage := "🔥*Sauna lämpiää\\!*🔥"
    if mockBot.SentMessages[0] != expectedWarmingMessage {
        t.Errorf("Expected message: %s, got: %s", expectedWarmingMessage, mockBot.SentMessages[0])
    }

    // Simulate temperature dropping before reaching ReadyThreshold
    kiuas.Temperature = 35.0
    kiuas.AddTemperatureRecord(35.0, time.Now())

    // Call checkAndNotify again after temperature drop
    checkAndNotify(mockBot, ctx, kiuas, config)

    // Notifications should be reset
    if kiuas.WarmingNotificationSent || kiuas.ReadyNotificationSent {
        t.Errorf("Expected notifications to be reset")
    }

    // No new messages should have been sent during reset
    if len(mockBot.SentMessages) != 1 {
        t.Fatalf("Expected 1 message to be sent (warming notification only), got %d", len(mockBot.SentMessages))
    }
}