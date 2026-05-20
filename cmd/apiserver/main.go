package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"

	"github.com/vishworks/assetmgmt/internal/db"
	"github.com/vishworks/assetmgmt/internal/models"
	"github.com/vishworks/assetmgmt/internal/workflows"
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

	// Serve generated JSON reports directly
	reportDir := envOrDefault("REPORT_DIR", "./reports")
	r.Static("/reports", reportDir)

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

	// POST /api/capital-calls/:callId/force-settlement
	r.POST("/api/capital-calls/:callId/force-settlement", func(ctx *gin.Context) {
		callID := ctx.Param("callId")
		parentWorkflowID := "capital-call-" + callID

		err := c.SignalWorkflow(ctx, parentWorkflowID, "", workflows.SignalForceSettlement, nil)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"status": "signal_sent",
			"callId": callID,
			"signal": workflows.SignalForceSettlement,
		})
	})

	// POST /api/capital-calls/:callId/cancel-call
	r.POST("/api/capital-calls/:callId/cancel-call", func(ctx *gin.Context) {
		callID := ctx.Param("callId")
		parentWorkflowID := "capital-call-" + callID

		err := c.SignalWorkflow(ctx, parentWorkflowID, "", workflows.SignalCancelCall, nil)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"status": "signal_sent",
			"callId": callID,
			"signal": workflows.SignalCancelCall,
		})
	})

	// ─── Capital Call Workflow Initiation ────────────────────────────────────

	// POST /api/capital-calls — starts a new CapitalCallWorkflow.
	// Backend generates the callId; frontend provides fund, target, LPs, deadline.
	r.POST("/api/capital-calls", func(ctx *gin.Context) {
		var body struct {
			FundID          string      `json:"fundId" binding:"required"`
			TargetAmountUSD float64     `json:"targetAmountUSD" binding:"required"`
			LPList          []models.LP `json:"lpList" binding:"required"`
			DeadlineDays    int         `json:"deadlineDays"`
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

	// ─── Workflow Timeline ────────────────────────────────────────────────────

	// GET /api/capital-calls/:callId/timeline
	// Reads Temporal workflow history for the parent CapitalCallWorkflow and
	// returns a lightweight list of completed activities in chronological order.
	// No DB reads — Temporal history is the source of truth.
	r.GET("/api/capital-calls/:callId/timeline", func(ctx *gin.Context) {
		callID := ctx.Param("callId")
		workflowID := "capital-call-" + callID

		displayNames := map[string]string{
			"IssueCapitalCall":       "Capital Call Issued",
			"NotifyLPs":              "LP Notifications Dispatched",
			"AutoFollowUp":           "Auto Follow-Up Sent",
			"PredictDefaultRisk":     "Default Risk Predicted",
			"ScoreLPPortfolio":       "Portfolio Scored",
			"AggregateLiquidity":     "Liquidity Aggregated",
			"TriggerBridge":          "Bridge Facility Triggered",
			"EscalateToGP":           "Escalated to GP",
			"SendEnforcementWarning": "Enforcement Warning Sent",
			"SettleAndReconcile":     "Settlement & Reconciliation",
			"EmitLiquidityReport":    "Report Generated",
			"UpdateLiveAggregates":   "Live Aggregates Updated",
			"MarkCallCancelled":      "Call Cancelled",
		}

		activityColors := map[string]string{
			"IssueCapitalCall":       "#3b82f6",
			"NotifyLPs":              "#3b82f6",
			"AutoFollowUp":           "#f59e0b",
			"PredictDefaultRisk":     "#a855f7",
			"ScoreLPPortfolio":       "#a855f7",
			"AggregateLiquidity":     "#3b82f6",
			"TriggerBridge":          "#a855f7",
			"EscalateToGP":           "#ef4444",
			"SendEnforcementWarning": "#ef4444",
			"SettleAndReconcile":     "#10b981",
			"EmitLiquidityReport":    "#10b981",
			"UpdateLiveAggregates":   "#3b82f6",
			"MarkCallCancelled":      "#6b7280",
		}

		type TimelineEvent struct {
			Name      string `json:"name"`
			Title     string `json:"title"`
			Status    string `json:"status"`
			Timestamp string `json:"timestamp"`
			Color     string `json:"color"`
		}

		// Pass 1: build eventId → activityType map from all scheduled events.
		scheduledByID := make(map[int64]string)
		iter1 := c.GetWorkflowHistory(ctx, workflowID, "", false, 0)
		for iter1.HasNext() {
			ev, err := iter1.Next()
			if err != nil {
				log.Printf("timeline pass1 %s: %v", workflowID, err)
				break
			}
			if sa := ev.GetActivityTaskScheduledEventAttributes(); sa != nil {
				scheduledByID[ev.GetEventId()] = sa.GetActivityType().GetName()
			}
		}

		// Pass 2: walk history again, emit one entry per completed activity.
		var events []TimelineEvent
		iter2 := c.GetWorkflowHistory(ctx, workflowID, "", false, 0)
		for iter2.HasNext() {
			ev, err := iter2.Next()
			if err != nil {
				log.Printf("timeline pass2 %s: %v", workflowID, err)
				break
			}
			attrs := ev.GetActivityTaskCompletedEventAttributes()
			if attrs == nil {
				continue
			}
			actName, ok := scheduledByID[attrs.GetScheduledEventId()]
			if !ok {
				continue
			}

			ts := ""
			if ev.EventTime != nil {
				ts = ev.EventTime.AsTime().UTC().Format(time.RFC3339)
			}

			title := actName
			if d, ok := displayNames[actName]; ok {
				title = d
			}
			color := "#6b7280"
			if col, ok := activityColors[actName]; ok {
				color = col
			}

			events = append(events, TimelineEvent{
				Name:      actName,
				Title:     title,
				Status:    "completed",
				Timestamp: ts,
				Color:     color,
			})
		}

		if events == nil {
			events = []TimelineEvent{}
		}
		ctx.JSON(http.StatusOK, events)
	})

	// ─── Capital Calls List (DB-driven) ─────────────────────────────────────


	// GET /api/capital-calls — returns all capital calls from DB (optionally filtered by lpId)
	r.GET("/api/capital-calls", func(c *gin.Context) {
		lpID := c.Query("lpId")

		var query string
		var args []interface{}

		if lpID != "" {
			query = `SELECT cc.call_id, cc.fund_id, cc.target_amount_usd, cc.received_amount_usd,
				        COALESCE(cc.lp_completion_count, '0 / 0'), cc.deadline_date, cc.status,
				        cl.commitment_usd, cl.draw_amount_usd, cl.status AS lp_status
				 FROM capital_calls cc
				 JOIN capital_call_lps cl ON cc.call_id = cl.call_id
				 WHERE cl.lp_id = $1
				 ORDER BY cc.created_at DESC`
			args = append(args, lpID)
		} else {
			query = `SELECT call_id, fund_id, target_amount_usd, received_amount_usd,
				        COALESCE(lp_completion_count, '0 / 0'),
				        deadline_date, status,
				        0.0 AS commitment_usd, NULL AS draw_amount_usd, '' AS lp_status
				 FROM capital_calls
				 ORDER BY created_at DESC`
		}

		rows, err := pool.Query(context.Background(), query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var calls []map[string]interface{}
		for rows.Next() {
			var callId, fundId, status, lpCompletion, lpStatus string
			var targetAmt, receivedAmt, commitmentUSD float64
			var deadline *time.Time
			var drawAmountUSD *float64

			if err := rows.Scan(&callId, &fundId, &targetAmt, &receivedAmt, &lpCompletion, &deadline, &status, &commitmentUSD, &drawAmountUSD, &lpStatus); err != nil {
				log.Printf("Failed to scan capital_calls row: %v", err)
				continue
			}

			deadlineStr := ""
			if deadline != nil {
				deadlineStr = deadline.Format(time.RFC3339)
			}

			callObj := map[string]interface{}{
				"id":           callId,
				"fund":         fundId,
				"target":       targetAmt,
				"received":     receivedAmt,
				"lpCompletion": lpCompletion,
				"deadlineDate": deadlineStr,
				"status":       status,
			}

			if lpID != "" {
				drawAmt := 0.0
				if drawAmountUSD != nil {
					drawAmt = *drawAmountUSD
				}
				callObj["commitmentUSD"] = commitmentUSD
				callObj["drawAmountUSD"] = drawAmt
				callObj["lpStatus"] = lpStatus
			}

			calls = append(calls, callObj)
		}

		if calls == nil {
			calls = []map[string]interface{}{}
		}

		c.JSON(http.StatusOK, calls)
	})

	// GET /api/capital-calls/lps — unified endpoint for LPs across calls
	// Query params: callId, risk, callStatus
	r.GET("/api/capital-calls/lps", func(c *gin.Context) {
		callID := c.Query("callId")
		risk := c.Query("risk")
		callStatus := c.Query("callStatus")

		query := `SELECT cl.call_id, cl.lp_id, cl.commitment_usd, cl.draw_amount_usd, cl.status AS lp_status, cl.risk_score,
		                 cc.status AS call_status, cc.created_at AS flagged_at
		          FROM capital_call_lps cl
		          JOIN capital_calls cc ON cl.call_id = cc.call_id
		          WHERE 1=1`
		var args []interface{}
		argId := 1

		if callID != "" {
			query += fmt.Sprintf(" AND cl.call_id = $%d", argId)
			args = append(args, callID)
			argId++
		}
		if risk == "high" {
			query += fmt.Sprintf(" AND cl.risk_score > 0.7")
		}
		if callStatus != "" {
			query += fmt.Sprintf(" AND cc.status = $%d", argId)
			args = append(args, callStatus)
			argId++
		}

		query += " ORDER BY cl.risk_score DESC, cl.lp_id ASC"

		rows, err := pool.Query(context.Background(), query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var lps []map[string]interface{}
		for rows.Next() {
			var cid, lpId, lpStatusStr, callStatusStr string
			var commitmentUSD float64
			var drawAmountUSD *float64
			var riskScore *float64
			var flaggedAt *time.Time

			if err := rows.Scan(&cid, &lpId, &commitmentUSD, &drawAmountUSD, &lpStatusStr, &riskScore, &callStatusStr, &flaggedAt); err != nil {
				log.Printf("Failed to scan lps row: %v", err)
				continue
			}

			drawAmt := 0.0
			if drawAmountUSD != nil {
				drawAmt = *drawAmountUSD
			}

			var riskVal interface{} = nil
			if riskScore != nil {
				riskVal = *riskScore
			}

			flaggedAtStr := ""
			if flaggedAt != nil {
				flaggedAtStr = flaggedAt.Format(time.RFC3339)
			}

			lps = append(lps, map[string]interface{}{
				"callId":        cid,
				"lpId":          lpId,
				"commitmentUSD": commitmentUSD,
				"drawAmountUSD": drawAmt,
				"status":        lpStatusStr,
				"lpStatus":      lpStatusStr, // Provide both status and lpStatus for frontend compatibility
				"callStatus":    callStatusStr,
				"riskScore":     riskVal,
				"flaggedAt":     flaggedAtStr,
			})
		}

		if lps == nil {
			lps = []map[string]interface{}{}
		}

		c.JSON(http.StatusOK, lps)
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
