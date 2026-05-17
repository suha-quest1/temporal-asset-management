package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/vishworks/assetmgmt/internal/models"
)

func main() {
	apiServerURL := envOrDefault("API_SERVER_URL", "http://localhost:8090")

	// Poll the API server until the frontend starts a capital call
	log.Println("Waiting for frontend to start a Capital Call...")
	var callID string
	var lps []models.LP

	for {
		resp, err := http.Get(apiServerURL + "/api/capital-calls/latest")
		if err == nil && resp.StatusCode == http.StatusOK {
			var latestCall models.CapitalCallInput
			if err := json.NewDecoder(resp.Body).Decode(&latestCall); err == nil {
				callID = latestCall.CallID
				lps = latestCall.LPList
				resp.Body.Close()
				break
			}
		}
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}

	log.Printf("Detected new Capital Call: %s (Target: $%.2f)", callID, 0.0) // To fix compiler for target amount later if needed

	// Give the workflow a moment to start child workflows
	time.Sleep(3 * time.Second)

	// 7 LPs respond immediately (LP 01-07)
	log.Println("═══ Sending 7 immediate LP responses ═══")
	for i := 0; i < 7; i++ {
		lp := lps[i]
		sendLPResponse(apiServerURL, callID, lp.LPID, lp.CommitmentUSD)
		time.Sleep(300 * time.Millisecond) // stagger for visibility
	}

	// LP 08 responds (will be flagged as high-risk by ML scorer)
	log.Println()
	log.Println("═══ LP-08 commits (will be flagged high-risk) ═══")
	time.Sleep(1 * time.Second)
	sendLPResponse(apiServerURL, callID, "lp-08", lps[7].CommitmentUSD)

	// LP 09 and LP 10 do NOT respond → triggers autoFollowUp
	log.Println()
	log.Printf("═══ LP-09 and LP-10 will NOT respond (triggering auto follow-up) ═══")
	log.Println("Waiting for follow-up escalation sequence...")
	log.Println()

	// Wait for escalation to GP (LP-08 high risk)
	// The ML scorer returns high risk for lp-08.
	// After child workflows complete and risk scoring runs, the parent
	// will escalate lp-08 and wait for a GP decision.
	log.Println("═══ Waiting for GP escalation, then sending GP decision ═══")
	time.Sleep(40 * time.Second) // wait for child timeouts + risk scoring

	sendGPDecision(apiServerURL, callID, "lp-08", "waive", "Vishy Iyer")

	log.Println()
	log.Println("════════════════════════════════════════")
	log.Println("  DEMODRIVER SIMULATION COMPLETE")
	log.Println("  Workflow will continue progressing in the background.")
	log.Println("════════════════════════════════════════")
}

func sendLPResponse(apiURL, callID, lpID string, amount float64) {
	payload := map[string]interface{}{
		"lpId":   lpID,
		"amount": amount,
	}
	data, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/api/capital-calls/%s/lp-response", apiURL, callID)
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("  [ERROR] Failed to send LP response for %s: %v", lpID, err)
		return
	}
	resp.Body.Close()
	log.Printf("  ✓ %s committed $%.2f (HTTP %d)", lpID, amount, resp.StatusCode)
}

func sendGPDecision(apiURL, callID, lpID, action, gpName string) {
	payload := map[string]interface{}{
		"lpId":   lpID,
		"action": action,
		"gpName": gpName,
	}
	data, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/api/capital-calls/%s/gp-decision", apiURL, callID)
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("  [ERROR] Failed to send GP decision for %s: %v", lpID, err)
		return
	}
	resp.Body.Close()
	log.Printf("  ✓ GP decision for %s: %s (HTTP %d)", lpID, action, resp.StatusCode)
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
