package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"

	// "strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/peterhellberg/ruuvitag"
)

// Globally store last temperature value
var lastTemperature float64

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Create a channel to send data from HTTP server to Telegram bot
	// msgChan := make(chan ruuvitag.RAWv2)

	// Set up the Telegram bot
	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			// Basic echo functionality
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   update.Message.Text,
			})
		}),
	}

	b, err := bot.New("6820973406:AAE8ZM4v6OPDmeADFgqqhP-Vd6IB5-iKHXA", opts...)
	if err != nil {
		panic(err)
	}

	// Handler for /temperature command
	b.RegisterHandler(bot.HandlerTypeMessageText, "/temperature", bot.MatchTypeExact, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		// Send the message to the specific Telegram user
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Temperature: %.1f °C", lastTemperature),
		})
		if err != nil {
			fmt.Printf("Failed to send message: %v\n", err)
		}
	})

	// Start the Telegram bot in a separate goroutine
	go b.Start(ctx)

	// Start the HTTP server in a separate goroutine
	go func() {
		http.HandleFunc("/api/receive-bt", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
				return
			}

			// Read the request body
			body, err := io.ReadAll(r.Body)
			if err != nil {
				// Print the error and return
				println(err)
				fmt.Println(err)

				http.Error(w, "Failed to read request body", http.StatusInternalServerError)
				return
			}

			defer r.Body.Close()
			if err != nil {
				// Print the error and return
				println(err)
				fmt.Println(err)
			}

			// Print the body in hex
			// println(hex.Dump(body))

			// Parse ruuvitag data from the HTTP request body
			ruuviTag, err := ruuvitag.ParseRAWv2(body)
			if err != nil {
				// The parser returns error for tag variants without all sensors, so error checking is skipped
				// since there is always an error present... bruh...
			}

			lastTemperature = ruuviTag.Temperature
			// "Received new temperature value: %.1f °C", lastTemperature
			println("Received new temperature value:", lastTemperature)
		})

		if err := http.ListenAndServe(":1337", nil); err != nil {
			panic(err)
		}
	}()

	// Handle incoming messages from the HTTP server channel
	for {
		select {
		case <-ctx.Done():
			return
			// case data := <-msgChan:
			// 	// Send the message to the specific Telegram user
			// 	fmt.Println(data)

			// 	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			// 		ChatID: 516822194,
			// 		Text:   fmt.Sprintf("Temperature: %.1f °C", data.Temperature),
			// 	})
			// 	if err != nil {
			// 		fmt.Printf("Failed to send message: %v\n", err)
			// 	}
			// }
		}
	}
}
