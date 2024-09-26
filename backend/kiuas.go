package main

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Kiuas struct {
	Temperature             float64
	Humidity                float64
	Battery                 uint16
	WarmingNotificationSent bool
	ReadyNotificationSent   bool
	lastDataReceived        time.Time
	TemperatureRecords      [3]float64
	TimestampRecords        [3]time.Time
}

func (k *Kiuas) IsOn() bool {
	warmingThreshold, err := strconv.ParseFloat(os.Getenv("SAUNA_WARMING_THRESHOLD"), 64)
	if err != nil {
		log.Fatalf("Error parsing SAUNA_WARMING_THRESHOLD: %v", err)
	}
	return k.Temperature > warmingThreshold
}

// Shift the old values and add a new temperature record
func (s *Kiuas) AddTemperatureRecord(newTemp float64, newTime time.Time) {
	// Shift the array to remove the oldest record and add the new one
	s.TemperatureRecords[0] = s.TemperatureRecords[1]
	s.TemperatureRecords[1] = s.TemperatureRecords[2]
	s.TemperatureRecords[2] = newTemp

	// Similarly update the timestamps
	s.TimestampRecords[0] = s.TimestampRecords[1]
	s.TimestampRecords[1] = s.TimestampRecords[2]
	s.TimestampRecords[2] = newTime
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
