package main

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Kiuas struct {
	Temperature          float64
	Humidity             float64
	Battery              uint16
	WarmingNotificationSent bool
	ReadyNotificationSent    bool
	lastDataReceived	 time.Time
}

func (k *Kiuas) IsOn() bool {
	warmingThreshold, err := strconv.ParseFloat(os.Getenv("SAUNA_WARMING_THRESHOLD"), 64)
	if err != nil {
		log.Fatalf("Error parsing SAUNA_WARMING_THRESHOLD: %v", err)
	}
	return k.Temperature > warmingThreshold
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
