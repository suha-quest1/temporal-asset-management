package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/activity"

	"github.com/vishworks/assetmgmt/internal/models"
)

// Activities holds shared dependencies for all activities.
type Activities struct {
	DB             *pgxpool.Pool
	SESURL         string
	MLScorerURL    string
	CreditFacURL   string
	ReportGenURL   string
}

// IssueCapitalCall creates the call record in the fund admin system and
// computes each LP's pro-rata draw amount.
func (a *Activities) IssueCapitalCall(ctx context.Context, input models.IssueCapitalCallInput) (*models.IssueCapitalCallResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Issuing capital call", "callId", input.CallID, "fundId", input.FundID)

	// Compute total commitments for pro-rata calculation
	totalCommitment := 0.0
	for _, lp := range input.LPList {
		totalCommitment += lp.CommitmentUSD
	}

	drawAmounts := make(map[string]float64, len(input.LPList))
	for _, lp := range input.LPList {
		ratio := lp.CommitmentUSD / totalCommitment
		drawAmounts[lp.LPID] = math.Round(input.TargetAmountUSD*ratio*100) / 100
	}

	if a.DB != nil {
		// Insert capital call record
		_, err := a.DB.Exec(ctx,
			`INSERT INTO capital_calls (call_id, fund_id, target_amount_usd, status)
			 VALUES ($1, $2, $3, 'issued')
			 ON CONFLICT (call_id) DO NOTHING`,
			input.CallID, input.FundID, input.TargetAmountUSD,
		)
		if err != nil {
			return nil, fmt.Errorf("insert capital_calls: %w", err)
		}

		// Insert per-LP draw amounts
		for _, lp := range input.LPList {
			_, err := a.DB.Exec(ctx,
				`INSERT INTO capital_call_lps (call_id, lp_id, commitment_usd, draw_amount_usd, status)
				 VALUES ($1, $2, $3, $4, 'pending')
				 ON CONFLICT (call_id, lp_id) DO NOTHING`,
				input.CallID, lp.LPID, lp.CommitmentUSD, drawAmounts[lp.LPID],
			)
			if err != nil {
				return nil, fmt.Errorf("insert capital_call_lps: %w", err)
			}
		}
	}

	logger.Info("Capital call issued", "callId", input.CallID, "lpCount", len(input.LPList))
	return &models.IssueCapitalCallResult{
		CallID:      input.CallID,
		DrawAmounts: drawAmounts,
	}, nil
}

// NotifyLPs sends email notifications to all LPs via the SES service.
func (a *Activities) NotifyLPs(ctx context.Context, input models.NotifyLPsInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Notifying LPs", "callId", input.CallID, "count", len(input.LPList))

	for _, lp := range input.LPList {
		payload := map[string]interface{}{
			"to":      lp.Email,
			"subject": fmt.Sprintf("Capital Call %s — Action Required", input.CallID),
			"body":    fmt.Sprintf("Dear LP %s, a capital call for $%.2f has been issued. Please respond via the investor portal.", lp.LPID, lp.CommitmentUSD),
			"callId":  input.CallID,
			"lpId":    lp.LPID,
		}
		if err := postJSON(ctx, a.SESURL+"/send", payload); err != nil {
			logger.Warn("Failed to notify LP", "lpId", lp.LPID, "error", err)
			// Continue notifying others; don't fail the whole batch.
		}
	}
	return nil
}

// AutoFollowUp sends a tiered escalation communication to a non-responsive LP.
func (a *Activities) AutoFollowUp(ctx context.Context, input models.AutoFollowUpInput) error {
	logger := activity.GetLogger(ctx)

	stageNames := map[int]string{1: "email", 2: "phone call", 3: "legal notice"}
	stageName := stageNames[input.Stage]
	logger.Info("Auto follow-up", "callId", input.CallID, "lpId", input.LPID, "stage", stageName)

	payload := map[string]interface{}{
		"to":      input.Email,
		"subject": fmt.Sprintf("Capital Call %s — Follow-up (%s)", input.CallID, stageName),
		"body":    fmt.Sprintf("Follow-up stage %d (%s) for LP %s on capital call %s.", input.Stage, stageName, input.LPID, input.CallID),
		"callId":  input.CallID,
		"lpId":    input.LPID,
		"stage":   input.Stage,
		"type":    "followup",
	}
	if err := postJSON(ctx, a.SESURL+"/send", payload); err != nil {
		logger.Warn("Failed to send follow-up", "lpId", input.LPID, "error", err)
	}
	return nil
}

// PredictDefaultRisk calls the ML scorer service to score each LP's default
// probability and returns updated LP responses with risk scores.
func (a *Activities) PredictDefaultRisk(ctx context.Context, input models.PredictDefaultRiskInput) ([]models.LPResponse, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Predicting default risk", "callId", input.CallID, "lpCount", len(input.LPResponses))

	reqBody, err := json.Marshal(map[string]interface{}{
		"callId":      input.CallID,
		"lpResponses": input.LPResponses,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := http.Post(a.MLScorerURL+"/score", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("call ml-scorer: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read ml-scorer response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ml-scorer returned %d: %s", resp.StatusCode, string(body))
	}

	var scores []struct {
		LPID      string  `json:"lpId"`
		RiskScore float64 `json:"riskScore"`
	}
	if err := json.Unmarshal(body, &scores); err != nil {
		return nil, fmt.Errorf("unmarshal ml-scorer response: %w", err)
	}

	scoreMap := make(map[string]float64, len(scores))
	for _, s := range scores {
		scoreMap[s.LPID] = s.RiskScore
	}

	result := make([]models.LPResponse, len(input.LPResponses))
	copy(result, input.LPResponses)
	for i := range result {
		if score, ok := scoreMap[result[i].LPID]; ok {
			result[i].RiskScore = score
		}
	}

	return result, nil
}

// AggregateLiquidity computes the total committed amount and the gap vs target.
func (a *Activities) AggregateLiquidity(ctx context.Context, input models.AggregateLiquidityInput) (*models.AggregateLiquidityResult, error) {
	logger := activity.GetLogger(ctx)

	totalCommitted := 0.0
	for _, lp := range input.LPResponses {
		if lp.Status == "committed" {
			totalCommitted += lp.AmountUSD
		}
	}

	gap := input.TargetAmountUSD - totalCommitted
	if gap < 0 {
		gap = 0
	}
	gapPercent := 0.0
	if input.TargetAmountUSD > 0 {
		gapPercent = (gap / input.TargetAmountUSD) * 100
	}

	logger.Info("Liquidity aggregated",
		"callId", input.CallID,
		"committed", totalCommitted,
		"target", input.TargetAmountUSD,
		"gapPercent", fmt.Sprintf("%.1f%%", gapPercent),
	)

	return &models.AggregateLiquidityResult{
		TotalCommitted: totalCommitted,
		GapUSD:         gap,
		GapPercent:     gapPercent,
	}, nil
}

// ScoreLPPortfolio computes concentration risk using a Herfindahl-Hirschman
// Index and identifies the riskiest LPs.
func (a *Activities) ScoreLPPortfolio(ctx context.Context, input models.ScoreLPPortfolioInput) (*models.PortfolioRisk, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Scoring LP portfolio", "callId", input.CallID)

	totalAmount := 0.0
	for _, lp := range input.LPResponses {
		totalAmount += lp.AmountUSD
	}

	// Herfindahl-Hirschman Index: sum of squared market shares
	hhi := 0.0
	if totalAmount > 0 {
		for _, lp := range input.LPResponses {
			share := lp.AmountUSD / totalAmount
			hhi += share * share
		}
	}

	// Identify top risky LPs (by risk score, descending)
	sorted := make([]models.LPResponse, len(input.LPResponses))
	copy(sorted, input.LPResponses)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RiskScore > sorted[j].RiskScore
	})

	topRisky := make([]string, 0)
	for i := 0; i < len(sorted) && i < 3; i++ {
		if sorted[i].RiskScore > 0.5 {
			topRisky = append(topRisky, sorted[i].LPID)
		}
	}

	return &models.PortfolioRisk{
		ConcentrationScore: math.Round(hhi*10000) / 10000,
		TopRiskyLPs:        topRisky,
	}, nil
}

// TriggerBridge posts a drawdown request to the credit facility when the
// liquidity gap exceeds the threshold.
func (a *Activities) TriggerBridge(ctx context.Context, input models.TriggerBridgeInput) (*models.TriggerBridgeResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Triggering bridge facility", "callId", input.CallID, "gapUSD", input.GapUSD)

	reqBody, err := json.Marshal(map[string]interface{}{
		"callId": input.CallID,
		"amount": input.GapUSD,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := http.Post(a.CreditFacURL+"/drawdown", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("call credit facility: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read credit facility response: %w", err)
	}

	var result models.TriggerBridgeResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal credit facility response: %w", err)
	}

	logger.Info("Bridge drawdown confirmed", "confirmationId", result.ConfirmationID)
	return &result, nil
}

// EscalateToGP sends a Slack-style alert to the GP channel with LP summary.
func (a *Activities) EscalateToGP(ctx context.Context, input models.EscalateToGPInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Escalating to GP",
		"callId", input.CallID,
		"lpId", input.LP.LPID,
		"riskScore", input.Risk,
	)

	// In production this would POST to Slack. Here we log and optionally
	// POST to the mock-SES service for visibility.
	payload := map[string]interface{}{
		"to":      "gp-channel",
		"subject": fmt.Sprintf("[ESCALATION] High-risk LP %s on call %s", input.LP.LPID, input.CallID),
		"body":    fmt.Sprintf("LP %s (commitment $%.2f) has a risk score of %.2f. GP decision required: waive or enforce.", input.LP.LPID, input.LP.CommitmentUSD, input.Risk),
		"callId":  input.CallID,
		"lpId":    input.LP.LPID,
		"type":    "escalation",
	}
	_ = postJSON(ctx, a.SESURL+"/send", payload)
	return nil
}

// SettleAndReconcile creates wire transfer stubs and double-entry ledger records.
func (a *Activities) SettleAndReconcile(ctx context.Context, input models.SettleAndReconcileInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Settling and reconciling", "callId", input.CallID)

	if a.DB != nil {
		for _, lp := range input.LPResponses {
			// Update each LP's final status in the fund ledger table.
			_, err := a.DB.Exec(ctx,
				`UPDATE capital_call_lps SET status = $1 WHERE call_id = $2 AND lp_id = $3`,
				lp.Status, input.CallID, lp.LPID,
			)
			if err != nil {
				return fmt.Errorf("update capital_call_lps status for LP %s: %w", lp.LPID, err)
			}

			if lp.Status != "committed" {
				continue
			}
			// Debit: LP capital account → Credit: Fund cash account
			_, err = a.DB.Exec(ctx,
				`INSERT INTO ledger_entries (call_id, lp_id, debit_account, credit_account, amount_usd, entry_type)
				 VALUES ($1, $2, $3, $4, $5, 'capital_call')`,
				input.CallID, lp.LPID,
				fmt.Sprintf("lp/%s/capital", lp.LPID),
				"fund/cash",
				lp.AmountUSD,
			)
			if err != nil {
				return fmt.Errorf("insert ledger entry for LP %s: %w", lp.LPID, err)
			}
		}

		// If bridge was used, record the credit facility ledger entry
		if input.BridgeResult != nil {
			_, err := a.DB.Exec(ctx,
				`INSERT INTO ledger_entries (call_id, lp_id, debit_account, credit_account, amount_usd, entry_type)
				 VALUES ($1, NULL, 'credit_facility/drawdown', 'fund/cash', $2, 'bridge_drawdown')`,
				input.CallID, input.BridgeResult.DrawdownUSD,
			)
			if err != nil {
				return fmt.Errorf("insert bridge ledger entry: %w", err)
			}
		}

		// Update call status
		_, err := a.DB.Exec(ctx,
			`UPDATE capital_calls SET status = 'settled' WHERE call_id = $1`,
			input.CallID,
		)
		if err != nil {
			return fmt.Errorf("update call status: %w", err)
		}
	}

	logger.Info("Settlement complete", "callId", input.CallID)
	return nil
}

// EmitLiquidityReport generates a PDF + JSON report via the report-generator service.
func (a *Activities) EmitLiquidityReport(ctx context.Context, input models.EmitReportInput) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Emitting liquidity report", "callId", input.CallID)

	reqBody, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("marshal report request: %w", err)
	}

	resp, err := http.Post(a.ReportGenURL+"/generate", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("call report generator: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read report generator response: %w", err)
	}

	var result struct {
		ReportPath string `json:"reportPath"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("unmarshal report response: %w", err)
	}

	logger.Info("Report generated", "path", result.ReportPath)
	return result.ReportPath, nil
}

// postJSON is a helper that POSTs a JSON payload to the given URL.
func postJSON(_ context.Context, url string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// NewActivities creates a new Activities instance with the given configuration.
func NewActivities(pool *pgxpool.Pool, sesURL, mlScorerURL, creditFacURL, reportGenURL string) *Activities {
	return &Activities{
		DB:           pool,
		SESURL:       sesURL,
		MLScorerURL:  mlScorerURL,
		CreditFacURL: creditFacURL,
		ReportGenURL: reportGenURL,
	}
}

// GenerateID returns a new unique identifier for use in activity results.
func GenerateID() string {
	return uuid.New().String()
}
