package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/vishworks/assetmgmt/internal/activities"
	"github.com/vishworks/assetmgmt/internal/models"
)

const (
	TaskQueue          = "capital-call-queue"
	SignalLPCommitment = "lpCommitment"
	SignalGPDecision   = "gpDecision"
)

// act is a nil pointer used solely to obtain typed method references
// for workflow.ExecuteActivity. The Temporal SDK only inspects the method
// name; the receiver is never dereferenced.
var act *activities.Activities

// CapitalCallWorkflow orchestrates the full capital-call lifecycle.
func CapitalCallWorkflow(ctx workflow.Context, input models.CapitalCallInput) (*models.CapitalCallResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("CapitalCallWorkflow started", "callId", input.CallID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// ── Step 1: Issue capital call ───────────────────────────────────────
	var issueResult models.IssueCapitalCallResult
	err := workflow.ExecuteActivity(ctx, act.IssueCapitalCall, models.IssueCapitalCallInput{
		CallID:          input.CallID,
		FundID:          input.FundID,
		TargetAmountUSD: input.TargetAmountUSD,
		LPList:          input.LPList,
	}).Get(ctx, &issueResult)
	if err != nil {
		return nil, fmt.Errorf("issueCapitalCall: %w", err)
	}

	// ── Step 2: Notify LPs ──────────────────────────────────────────────
	err = workflow.ExecuteActivity(ctx, act.NotifyLPs, models.NotifyLPsInput{
		CallID: input.CallID,
		LPList: input.LPList,
	}).Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("notifyLPs: %w", err)
	}

	// ── Step 3: Start child LPResponseWorkflow per LP concurrently ──────
	childFutures := make([]workflow.Future, len(input.LPList))
	for i, lp := range input.LPList {
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID: fmt.Sprintf("lp-response-%s-%s", input.CallID, lp.LPID),
			TaskQueue:  TaskQueue,
		})
		childFutures[i] = workflow.ExecuteChildWorkflow(childCtx, LPResponseWorkflow, models.LPResponseInput{
			CallID:        input.CallID,
			LPID:          lp.LPID,
			CommitmentUSD: lp.CommitmentUSD,
			Email:         lp.Email,
			DeadlineDays:  input.DeadlineDays,
			SecondsPerDay: input.SecondsPerDay,
		})
	}

	// ── Step 4: Collect all LP responses ────────────────────────────────
	lpResponses := make([]models.LPResponse, len(input.LPList))
	for i, future := range childFutures {
		var resp models.LPResponse
		if err := future.Get(ctx, &resp); err != nil {
			logger.Warn("Child workflow failed, marking LP as defaulted",
				"lpId", input.LPList[i].LPID, "error", err)
			lpResponses[i] = models.LPResponse{
				LPID:   input.LPList[i].LPID,
				Status: "defaulted",
			}
		} else {
			lpResponses[i] = resp
		}
	}

	// ── Step 5: Aggregate liquidity ─────────────────────────────────────
	var aggResult models.AggregateLiquidityResult
	err = workflow.ExecuteActivity(ctx, act.AggregateLiquidity, models.AggregateLiquidityInput{
		CallID:          input.CallID,
		TargetAmountUSD: input.TargetAmountUSD,
		LPResponses:     lpResponses,
	}).Get(ctx, &aggResult)
	if err != nil {
		return nil, fmt.Errorf("aggregateLiquidity: %w", err)
	}

	// ── Step 6: Predict default risk ────────────────────────────────────
	var scoredResponses []models.LPResponse
	err = workflow.ExecuteActivity(ctx, act.PredictDefaultRisk, models.PredictDefaultRiskInput{
		CallID:      input.CallID,
		LPResponses: lpResponses,
	}).Get(ctx, &scoredResponses)
	if err != nil {
		return nil, fmt.Errorf("predictDefaultRisk: %w", err)
	}
	lpResponses = scoredResponses

	// ── Step 6b: Score LP portfolio ─────────────────────────────────────
	var portfolioRisk models.PortfolioRisk
	err = workflow.ExecuteActivity(ctx, act.ScoreLPPortfolio, models.ScoreLPPortfolioInput{
		CallID:      input.CallID,
		LPResponses: lpResponses,
	}).Get(ctx, &portfolioRisk)
	if err != nil {
		return nil, fmt.Errorf("scoreLPPortfolio: %w", err)
	}

	// ── Step 7: Bridge facility if gap > 10% ────────────────────────────
	var bridgeResult *models.TriggerBridgeResult
	bridgeTriggered := false
	if aggResult.GapPercent > 10.0 {
		bridgeResult = &models.TriggerBridgeResult{}
		err = workflow.ExecuteActivity(ctx, act.TriggerBridge, models.TriggerBridgeInput{
			CallID: input.CallID,
			GapUSD: aggResult.GapUSD,
		}).Get(ctx, bridgeResult)
		if err != nil {
			return nil, fmt.Errorf("triggerBridge: %w", err)
		}
		bridgeTriggered = true
		logger.Info("Bridge drawdown triggered", "amount", bridgeResult.DrawdownUSD)
	}

	// ── Step 8: Escalate high-risk LPs to GP ────────────────────────────
	// Only escalate LPs that are committed but high-risk. Defaulted LPs are
	// already resolved — there is no commitment to waive or enforce.
	highRiskIndices := []int{}
	for i, lp := range lpResponses {
		if lp.RiskScore > 0.7 && lp.Status == "committed" {
			highRiskIndices = append(highRiskIndices, i)
			// Find the original LP info for the escalation message
			var origLP models.LP
			for _, l := range input.LPList {
				if l.LPID == lp.LPID {
					origLP = l
					break
				}
			}
			err = workflow.ExecuteActivity(ctx, act.EscalateToGP, models.EscalateToGPInput{
				CallID: input.CallID,
				LP:     origLP,
				Risk:   lp.RiskScore,
			}).Get(ctx, nil)
			if err != nil {
				return nil, fmt.Errorf("escalateToGP for %s: %w", lp.LPID, err)
			}
		}
	}

	// Wait for a GP decision signal for each escalated LP.
	if len(highRiskIndices) > 0 {
		gpDecisionCh := workflow.GetSignalChannel(ctx, SignalGPDecision)
		for range highRiskIndices {
			var decision models.GPDecisionSignal
			gpDecisionCh.Receive(ctx, &decision)
			logger.Info("GP decision received", "lpId", decision.LPID, "action", decision.Action)
			for i := range lpResponses {
				if lpResponses[i].LPID == decision.LPID && decision.Action == "enforce" {
					lpResponses[i].Status = "defaulted"
					lpResponses[i].AmountUSD = 0
				}
			}
		}
	}

	// ── Step 9: Settle and reconcile ────────────────────────────────────
	err = workflow.ExecuteActivity(ctx, act.SettleAndReconcile, models.SettleAndReconcileInput{
		CallID:       input.CallID,
		LPResponses:  lpResponses,
		BridgeResult: bridgeResult,
	}).Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("settleAndReconcile: %w", err)
	}

	// ── Step 10: Emit liquidity report ──────────────────────────────────
	result := &models.CapitalCallResult{
		CallID:          input.CallID,
		TotalCommitted:  aggResult.TotalCommitted,
		TargetAmountUSD: input.TargetAmountUSD,
		GapUSD:          aggResult.GapUSD,
		GapPercent:      aggResult.GapPercent,
		BridgeTriggered: bridgeTriggered,
		BridgeResult:    bridgeResult,
		LPResponses:     lpResponses,
		PortfolioRisk:   portfolioRisk,
	}

	var reportPath string
	err = workflow.ExecuteActivity(ctx, act.EmitLiquidityReport, models.EmitReportInput{
		CallID:     input.CallID,
		FundID:     input.FundID,
		CallResult: *result,
	}).Get(ctx, &reportPath)
	if err != nil {
		return nil, fmt.Errorf("emitLiquidityReport: %w", err)
	}
	result.ReportPath = reportPath

	logger.Info("CapitalCallWorkflow completed", "callId", input.CallID)
	return result, nil
}
