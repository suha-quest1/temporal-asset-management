package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"

	"github.com/vishworks/assetmgmt/internal/models"
	"github.com/vishworks/assetmgmt/internal/workflows"
)

func main() {
	temporalAddr := envOrDefault("TEMPORAL_ADDRESS", "localhost:7233")
	listenAddr := envOrDefault("LISTEN_ADDR", ":8090")

	c, err := client.Dial(client.Options{HostPort: temporalAddr})
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	defer c.Close()

	r := gin.Default()

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

		// The child workflow ID follows the convention: lp-response-{callId}-{lpId}
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
			"status":  "signal_sent",
			"lpId":    body.LPID,
			"callId":  callID,
			"signal":  workflows.SignalLPCommitment,
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

		// The parent workflow ID follows the convention: capital-call-{callId}
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

	// POST /api/capital-calls — starts a new CapitalCallWorkflow
	r.POST("/api/capital-calls", func(ctx *gin.Context) {
		var input models.CapitalCallInput
		if err := ctx.ShouldBindJSON(&input); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		workflowID := "capital-call-" + input.CallID
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
			"callId":     input.CallID,
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
