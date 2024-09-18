package main

import (
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
	"os"
)

const (
	validTimeWindow = 300 // Time window in seconds (e.g., 5 minutes)
)

var apiKey = os.Getenv("API_KEY")

// In-memory nonce store with mutex for thread safety
var (
	usedNonces = make(map[string]time.Time)
	mutex      sync.Mutex
)

// Authentication handler for ForwardAuth
func authHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the original request URI (optional, for logging)
	originalURI := r.Header.Get("X-Forwarded-Uri")
	if originalURI == "" {
		log.Printf("Bad Request: Missing X-Forwarded-Uri header")
	}
	log.Printf("Authenticating request for %s", originalURI)

	// 1. Validate API Key
	apiKeyHeader := r.Header.Get("API-Key")
	if apiKeyHeader != apiKey {
		log.Printf("Forbidden: Invalid API Key for request %s", originalURI)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 2. Validate Timestamp
	timestampStr := r.Header.Get("Timestamp")
	if timestampStr == "" {
		log.Printf("Bad Request: Missing Timestamp for request %s", originalURI)
		http.Error(w, "Bad Request: Missing Timestamp", http.StatusBadRequest)
		return
	}
	timestampInt, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		log.Printf("Bad Request: Invalid Timestamp for request %s", originalURI)
		http.Error(w, "Bad Request: Invalid Timestamp", http.StatusBadRequest)
		return
	}
	now := time.Now().Unix()
	if abs(now-timestampInt) > validTimeWindow {
		log.Printf("Unauthorized: Timestamp Outside Allowed Window for request %s", originalURI)
		http.Error(w, "Unauthorized: Timestamp Outside Allowed Window", http.StatusUnauthorized)
		return
	}

	// 3. Validate Nonce
	nonce := r.Header.Get("Nonce")
	if nonce == "" {
		log.Printf("Bad Request: Missing Nonce for request %s", originalURI)
		http.Error(w, "Bad Request: Missing Nonce", http.StatusBadRequest)
		return
	}
	if isNonceUsed(nonce) {
		log.Printf("Unauthorized: Nonce Already Used for request %s", originalURI)
		http.Error(w, "Unauthorized: Nonce Already Used", http.StatusUnauthorized)
		return
	}
	markNonceAsUsed(nonce)

	// 4. Authentication Successful
	// ForwardAuth requires a 2xx response to proceed
	w.WriteHeader(http.StatusOK)
}

func isNonceUsed(nonce string) bool {
	mutex.Lock()
	defer mutex.Unlock()
	_, exists := usedNonces[nonce]
	return exists
}

func markNonceAsUsed(nonce string) {
	mutex.Lock()
	defer mutex.Unlock()
	usedNonces[nonce] = time.Now()
	// Optionally implement cleanup of old nonces
}

func abs(a int64) int64 {
	if a < 0 {
		return -a
	}
	return a
}

func cleanupNonces() {
	for {
		time.Sleep(time.Minute*5)
		mutex.Lock()
		for nonce, timestamp := range usedNonces {
			if time.Since(timestamp) > (time.Duration(validTimeWindow) * time.Second) {
				delete(usedNonces, nonce)
			}
		}
		mutex.Unlock()
	}
}

func main() {
	go cleanupNonces()
	http.HandleFunc("/auth", authHandler)
	log.Println("Authentication service started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
