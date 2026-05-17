package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/vishworks/assetmgmt/internal/workflows"

	"models"
	//"../workflows"

	"go.temporal.io/sdk/client"
)

type StartCapitalCallRequest struct {
	FundID          string                `json:"fundId"`
	TargetAmountUSD float64               `json:"targetAmountUSD"`
	LPList          []models.LPAllocation `json:"lpList"`
}

func StartCapitalCallHandler(tc client.Client) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		var req StartCapitalCallRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		callID := fmt.Sprintf("cc-%s", uuid.New().String())

		input := models.CapitalCallInput{
			CallID:          callID,
			FundID:          req.FundID,
			TargetAmountUSD: req.TargetAmountUSD,
			LPList:          req.LPList,
		}

		workflowOptions := client.StartWorkflowOptions{
			ID:        callID,
			TaskQueue: "capital-call-task-queue",
		}

		we, err := tc.ExecuteWorkflow(
			context.Background(),
			workflowOptions,
			workflows.CapitalCallWorkflow,
			input,
		)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"workflowId": we.GetID(),
			"runId":      we.GetRunID(),
			"status":     "STARTED",
			"timestamp":  time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
