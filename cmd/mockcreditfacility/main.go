package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"
)

func main() {
	addr := envOrDefault("LISTEN_ADDR", ":8082")

	mux := http.NewServeMux()

	// POST /drawdown — accepts a drawdown request and returns confirmation
	mux.HandleFunc("/drawdown", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			CallID string  `json:"callId"`
			Amount float64 `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		// Simulate a 1.5% facility fee
		fee := math.Round(req.Amount*0.015*100) / 100
		confirmationID := fmt.Sprintf("BRIDGE-%s-%d", req.CallID, time.Now().UnixMilli())

		log.Printf("[MOCK-CREDIT] Drawdown: callId=%s amount=$%.2f fee=$%.2f confirmation=%s",
			req.CallID, req.Amount, fee, confirmationID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"confirmationId": confirmationID,
			"drawdownUSD":    req.Amount,
			"feeUSD":         fee,
		})
	})

	// GET / — health check
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"service": "mock-credit-facility", "status": "ok"})
	})

	log.Printf("Mock Credit Facility listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
