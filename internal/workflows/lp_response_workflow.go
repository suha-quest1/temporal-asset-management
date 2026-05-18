package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/vishworks/assetmgmt/internal/models"
)

// LPResponseWorkflow handles a single LP's response lifecycle. It waits for
// a commitment signal, and if the LP does not respond within the deadline,
// it triggers an escalation sequence (email → phone call → legal notice).
func LPResponseWorkflow(ctx workflow.Context, input models.LPResponseInput) (*models.LPResponse, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("LPResponseWorkflow started", "callId", input.CallID, "lpId", input.LPID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    10 * time.Second,
			MaximumAttempts:    3,
		},
	}
	actCtx := workflow.WithActivityOptions(ctx, ao)

	signalCh := workflow.GetSignalChannel(ctx, SignalLPCommitment)

	// Determine time unit (real days vs demo seconds)
	dayDuration := 24 * time.Hour
	if input.SecondsPerDay > 0 {
		dayDuration = time.Duration(input.SecondsPerDay) * time.Second
	}

	// Follow-up stages at D+3, D+7, D+10
	followUpDays := []int{3, 7, 10}
	elapsedDays := 0

	for stage, day := range followUpDays {
		waitDays := day - elapsedDays
		timer := workflow.NewTimer(ctx, time.Duration(waitDays)*dayDuration)

		selector := workflow.NewSelector(ctx)
		committed := false
		var commitment models.LPCommitmentSignal

		selector.AddReceive(signalCh, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, &commitment)
			committed = true
		})
		selector.AddFuture(timer, func(f workflow.Future) {
			// Timer fired — LP did not respond in this window.
		})
		selector.Select(ctx)

		if committed {
			logger.Info("LP committed", "lpId", input.LPID, "amount", commitment.AmountUSD)
			
			_ = workflow.ExecuteActivity(actCtx, act.UpdateLiveAggregates, models.UpdateLiveAggregatesInput{
				CallID:    input.CallID,
				LPID:      input.LPID,
				Status:    "committed",
				AmountUSD: commitment.AmountUSD,
			}).Get(ctx, nil)
			
			return &models.LPResponse{
				LPID:      input.LPID,
				Status:    "committed",
				AmountUSD: commitment.AmountUSD,
			}, nil
		}

		// Timer expired — send follow-up at this stage
		logger.Info("LP follow-up triggered", "lpId", input.LPID, "stage", stage+1)
		_ = workflow.ExecuteActivity(actCtx, act.AutoFollowUp, models.AutoFollowUpInput{
			CallID: input.CallID,
			LPID:   input.LPID,
			Email:  input.Email,
			Stage:  stage + 1,
		}).Get(ctx, nil)

		elapsedDays = day
	}

	// Final wait: from last follow-up (D+10) to deadline
	remainingDays := input.DeadlineDays - elapsedDays
	if remainingDays > 0 {
		timer := workflow.NewTimer(ctx, time.Duration(remainingDays)*dayDuration)
		selector := workflow.NewSelector(ctx)
		committed := false
		var commitment models.LPCommitmentSignal

		selector.AddReceive(signalCh, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, &commitment)
			committed = true
		})
		selector.AddFuture(timer, func(f workflow.Future) {})
		selector.Select(ctx)

		if committed {
			logger.Info("LP committed (late)", "lpId", input.LPID, "amount", commitment.AmountUSD)
			
			_ = workflow.ExecuteActivity(actCtx, act.UpdateLiveAggregates, models.UpdateLiveAggregatesInput{
				CallID:    input.CallID,
				LPID:      input.LPID,
				Status:    "committed",
				AmountUSD: commitment.AmountUSD,
			}).Get(ctx, nil)
			
			return &models.LPResponse{
				LPID:      input.LPID,
				Status:    "committed",
				AmountUSD: commitment.AmountUSD,
			}, nil
		}
	}

	// LP never responded — defaulted
	logger.Info("LP defaulted", "lpId", input.LPID)
	
	_ = workflow.ExecuteActivity(actCtx, act.UpdateLiveAggregates, models.UpdateLiveAggregatesInput{
		CallID:    input.CallID,
		LPID:      input.LPID,
		Status:    "defaulted",
		AmountUSD: 0,
	}).Get(ctx, nil)
	
	return &models.LPResponse{
		LPID:   input.LPID,
		Status: "defaulted",
	}, nil
}
