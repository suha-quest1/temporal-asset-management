package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.temporal.io/sdk/client"

	"github.com/vishworks/assetmgmt/internal/models"
	"github.com/vishworks/assetmgmt/internal/workflows"
)

func main() {
	temporalAddr := envOrDefault("TEMPORAL_ADDRESS", "localhost:7233")
	apiServerURL := envOrDefault("API_SERVER_URL", "http://localhost:8090")

	c, err := client.Dial(client.Options{HostPort: temporalAddr})
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	defer c.Close()

	callID := fmt.Sprintf("call-%d", time.Now().Unix())

	// Pre-seed 10 mock LPs
	lps := make([]models.LP, 10)
	for i := 0; i < 10; i++ {
		lps[i] = models.LP{
			LPID:          fmt.Sprintf("lp-%02d", i+1),
			CommitmentUSD: float64((i+1)*1_000_000),
			Email:         fmt.Sprintf("lp%02d@example.com", i+1),
		}
	}

	input := models.CapitalCallInput{
		CallID:          callID,
		FundID:          "fund-alpha-1",
		TargetAmountUSD: 50_000_000,
		LPList:          lps,
		DeadlineDays:    15,
		SecondsPerDay:   2, // Demo mode: 2 seconds per "day"
	}

	// Start the parent workflow
	workflowID := "capital-call-" + callID
	opts := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: workflows.TaskQueue,
	}
	run, err := c.ExecuteWorkflow(context.Background(), opts, workflows.CapitalCallWorkflow, input)
	if err != nil {
		log.Fatalf("Failed to start workflow: %v", err)
	}
	log.Printf("Started CapitalCallWorkflow: workflowId=%s runId=%s", run.GetID(), run.GetRunID())
	log.Printf("Call ID: %s", callID)
	log.Println()

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

	sendGPDecision(apiServerURL, callID, "lp-08", "waive", "Jane Smith")

	// Wait for workflow completion
	log.Println()
	log.Println("═══ Waiting for workflow completion ═══")
	var result models.CapitalCallResult
	err = run.Get(context.Background(), &result)
	if err != nil {
		log.Fatalf("Workflow failed: %v", err)
	}

	log.Println()
	log.Println("════════════════════════════════════════")
	log.Println("  CAPITAL CALL COMPLETE")
	log.Println("════════════════════════════════════════")
	log.Printf("  Call ID:         %s", result.CallID)
	log.Printf("  Target:          $%.2f", result.TargetAmountUSD)
	log.Printf("  Total Committed: $%.2f", result.TotalCommitted)
	log.Printf("  Gap:             $%.2f (%.1f%%)", result.GapUSD, result.GapPercent)
	log.Printf("  Bridge Used:     %v", result.BridgeTriggered)
	log.Printf("  Report:          %s", result.ReportPath)
	log.Println()
	log.Println("  LP Responses:")
	for _, lp := range result.LPResponses {
		log.Printf("    %-8s  status=%-10s  amount=$%-12.2f  risk=%.2f",
			lp.LPID, lp.Status, lp.AmountUSD, lp.RiskScore)
	}
	log.Println()
	log.Printf("  Portfolio Concentration Score: %.4f", result.PortfolioRisk.ConcentrationScore)
	if len(result.PortfolioRisk.TopRiskyLPs) > 0 {
		log.Printf("  Top Risky LPs: %v", result.PortfolioRisk.TopRiskyLPs)
	}
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
