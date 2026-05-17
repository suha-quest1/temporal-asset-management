package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"

	"github.com/vishworks/assetmgmt/internal/db"
	"github.com/vishworks/assetmgmt/internal/models"
	"github.com/vishworks/assetmgmt/internal/workflows"
)

var (
	latestCallMu sync.Mutex
	latestCall   models.CapitalCallInput
)

func main() {
	temporalAddr := envOrDefault("TEMPORAL_ADDRESS", "localhost:7233")
	listenAddr := envOrDefault("LISTEN_ADDR", ":8090")
	databaseURL := envOrDefault("DATABASE_URL", "postgres://assetmgmt:assetmgmt@localhost:5432/assetmgmt?sslmode=disable")

	ctx := context.Background()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	c, err := client.Dial(client.Options{HostPort: temporalAddr})
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	defer c.Close()

	r := gin.Default()

	// ─── LP Responses & Signals ──────────────────────────────────────────────

	// POST /api/capital-calls/:callId/lp-response
	// Sends an lpCommitment signal to the child LPResponseWorkflow.
	r.POST("/api/capital-calls/:callId/lp-response", func(ctx *gin.Context) {
		callID := ctx.Param("callId")

		var body struct {
			LPID      string  `json:"lpId" binding:"required"`
			AmountUSD float64 `json:"amount" binding:"required"`
		}
		if err := ctx.ShouldBindJSON(&body); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		childWorkflowID := "lp-response-" + callID + "-" + body.LPID

		err := c.SignalWorkflow(ctx, childWorkflowID, "", workflows.SignalLPCommitment, models.LPCommitmentSignal{
			LPID:      body.LPID,
			AmountUSD: body.AmountUSD,
		})
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"status": "signal_sent",
			"lpId":   body.LPID,
			"callId": callID,
			"signal": workflows.SignalLPCommitment,
		})
	})

	// POST /api/capital-calls/:callId/gp-decision
	// Sends a gpDecision signal to the parent CapitalCallWorkflow.
	r.POST("/api/capital-calls/:callId/gp-decision", func(ctx *gin.Context) {
		callID := ctx.Param("callId")

		var body struct {
			LPID   string `json:"lpId" binding:"required"`
			Action string `json:"action" binding:"required"` // "waive" or "enforce"
			GPName string `json:"gpName"`
		}
		if err := ctx.ShouldBindJSON(&body); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		parentWorkflowID := "capital-call-" + callID

		err := c.SignalWorkflow(ctx, parentWorkflowID, "", workflows.SignalGPDecision, models.GPDecisionSignal{
			LPID:   body.LPID,
			Action: body.Action,
			GPName: body.GPName,
		})
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"status": "signal_sent",
			"lpId":   body.LPID,
			"callId": callID,
			"action": body.Action,
			"signal": workflows.SignalGPDecision,
		})
	})

	// ─── Capital Call Workflow Initiation ────────────────────────────────────

	// POST /api/capital-calls — starts a new CapitalCallWorkflow.
	// Backend generates the callId; frontend provides fund, target, LPs, deadline.
	r.POST("/api/capital-calls", func(ctx *gin.Context) {
		var body struct {
			FundID          string       `json:"fundId" binding:"required"`
			TargetAmountUSD float64      `json:"targetAmountUSD" binding:"required"`
			LPList          []models.LP  `json:"lpList" binding:"required"`
			DeadlineDays    int          `json:"deadlineDays"`
		}
		if err := ctx.ShouldBindJSON(&body); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Backend owns CallID generation
		callID := fmt.Sprintf("CC-%d-%04d", time.Now().Year(), rand.Intn(9000)+1000)

		input := models.CapitalCallInput{
			CallID:          callID,
			FundID:          body.FundID,
			TargetAmountUSD: body.TargetAmountUSD,
			LPList:          body.LPList,
			DeadlineDays:    body.DeadlineDays,
		}

		latestCallMu.Lock()
		latestCall = input
		latestCallMu.Unlock()

		workflowID := "capital-call-" + callID
		opts := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: workflows.TaskQueue,
		}

		run, err := c.ExecuteWorkflow(ctx, opts, workflows.CapitalCallWorkflow, input)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusAccepted, gin.H{
			"workflowId": run.GetID(),
			"runId":      run.GetRunID(),
			"callId":     callID,
		})
	})

	// ─── Capital Calls List (DB-driven) ─────────────────────────────────────

	// GET /api/capital-calls — returns all capital calls from DB
	r.GET("/api/capital-calls", func(c *gin.Context) {
		rows, err := pool.Query(context.Background(),
			`SELECT call_id, fund_id, target_amount_usd, received_amount_usd,
			        COALESCE(lp_completion_count, '0 / 0'),
			        deadline_date, status
			 FROM capital_calls
			 ORDER BY created_at DESC`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var calls []map[string]interface{}
		for rows.Next() {
			var callId, fundId, status, lpCompletion string
			var targetAmt, receivedAmt float64
			var deadline *time.Time // pointer so NULL scans as nil

			if err := rows.Scan(&callId, &fundId, &targetAmt, &receivedAmt, &lpCompletion, &deadline, &status); err != nil {
				log.Printf("Failed to scan capital_calls row: %v", err)
				continue
			}

			deadlineStr := ""
			if deadline != nil {
				deadlineStr = deadline.Format(time.RFC3339)
			}

			calls = append(calls, map[string]interface{}{
				"id":           callId,
				"fund":         fundId,
				"target":       targetAmt,
				"received":     receivedAmt,
				"lpCompletion": lpCompletion,
				"deadlineDate": deadlineStr,
				"status":       status,
			})
		}

		// Return empty array rather than null when no rows
		if calls == nil {
			calls = []map[string]interface{}{}
		}

		c.JSON(http.StatusOK, calls)
	})

	// ─── LP Master List ──────────────────────────────────────────────────────

	// GET /api/lps — returns all seeded LPs from the lps master table
	r.GET("/api/lps", func(c *gin.Context) {
		rows, err := pool.Query(context.Background(),
			`SELECT lp_id, commitment_usd, email FROM lps ORDER BY lp_id`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var lps []map[string]interface{}
		for rows.Next() {
			var lpId, email string
			var commitmentUSD float64

			if err := rows.Scan(&lpId, &commitmentUSD, &email); err != nil {
				log.Printf("Failed to scan lps row: %v", err)
				continue
			}

			lps = append(lps, map[string]interface{}{
				"lpId":          lpId,
				"commitmentUSD": commitmentUSD,
				"email":         email,
			})
		}

		if lps == nil {
			lps = []map[string]interface{}{}
		}

		c.JSON(http.StatusOK, lps)
	})

	// ─── Dashboard Stats ─────────────────────────────────────────────────────

	// GET /api/dashboard/stats — returns aggregated stats for the dashboard
	r.GET("/api/dashboard/stats", func(c *gin.Context) {
		var totalCalledYTD float64
		var pendingLiquidity float64
		var activeCalls int

		// Total called YTD: sum of target_amount_usd for this calendar year
		yearStart := fmt.Sprintf("%d-01-01", time.Now().Year())
		row := pool.QueryRow(context.Background(),
			`SELECT COALESCE(SUM(target_amount_usd), 0) FROM capital_calls WHERE created_at >= $1`,
			yearStart)
		_ = row.Scan(&totalCalledYTD)

		// Pending liquidity: sum of target_amount_usd for all 'issued' calls
		row = pool.QueryRow(context.Background(),
			`SELECT COALESCE(SUM(target_amount_usd - received_amount_usd), 0)
			 FROM capital_calls WHERE status = 'issued'`)
		_ = row.Scan(&pendingLiquidity)

		// Active calls count
		row = pool.QueryRow(context.Background(),
			`SELECT COUNT(*) FROM capital_calls WHERE status = 'issued'`)
		_ = row.Scan(&activeCalls)

		activeCallsStr := fmt.Sprintf("%02d", activeCalls)

		c.JSON(http.StatusOK, gin.H{
			"totalCalledYTD":   fmt.Sprintf("$%.1fM", totalCalledYTD/1_000_000),
			"pendingLiquidity": fmt.Sprintf("$%.1fM", pendingLiquidity/1_000_000),
			"avgLPResponse":    "—",
			"activeCalls":      activeCallsStr,
		})
	})

	// ─── Demodriver transitional endpoint ────────────────────────────────────

	// GET /api/capital-calls/latest — used by demodriver to poll for active call
	r.GET("/api/capital-calls/latest", func(ctx *gin.Context) {
		latestCallMu.Lock()
		defer latestCallMu.Unlock()

		if latestCall.CallID == "" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "no active call"})
			return
		}

		ctx.JSON(http.StatusOK, latestCall)
	})

	log.Printf("API server listening on %s", listenAddr)
	if err := r.Run(listenAddr); err != nil {
		log.Fatalf("API server failed: %v", err)
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
