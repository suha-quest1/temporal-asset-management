package activities

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"

	"github.com/vishworks/assetmgmt/internal/models"
)

// newTestEnv creates a TestActivityEnvironment with the given Activities
// struct registered, so that activity.GetLogger works correctly.
func newTestEnv(t *testing.T, a *Activities) *testsuite.TestActivityEnvironment {
	t.Helper()
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(a)
	return env
}

// ── AggregateLiquidity ─────────────────────────────────────────────────────

func TestAggregateLiquidity_AllCommitted(t *testing.T) {
	a := &Activities{}
	env := newTestEnv(t, a)

	val, err := env.ExecuteActivity(a.AggregateLiquidity, models.AggregateLiquidityInput{
		CallID:          "call-1",
		TargetAmountUSD: 10_000_000,
		LPResponses: []models.LPResponse{
			{LPID: "lp-01", Status: "committed", AmountUSD: 6_000_000},
			{LPID: "lp-02", Status: "committed", AmountUSD: 4_000_000},
		},
	})
	require.NoError(t, err)

	var result models.AggregateLiquidityResult
	require.NoError(t, val.Get(&result))
	assert.Equal(t, 10_000_000.0, result.TotalCommitted)
	assert.Equal(t, 0.0, result.GapUSD)
	assert.Equal(t, 0.0, result.GapPercent)
}

func TestAggregateLiquidity_WithGap(t *testing.T) {
	a := &Activities{}
	env := newTestEnv(t, a)

	val, err := env.ExecuteActivity(a.AggregateLiquidity, models.AggregateLiquidityInput{
		CallID:          "call-1",
		TargetAmountUSD: 10_000_000,
		LPResponses: []models.LPResponse{
			{LPID: "lp-01", Status: "committed", AmountUSD: 4_000_000},
			{LPID: "lp-02", Status: "defaulted", AmountUSD: 0},
			{LPID: "lp-03", Status: "committed", AmountUSD: 3_000_000},
		},
	})
	require.NoError(t, err)

	var result models.AggregateLiquidityResult
	require.NoError(t, val.Get(&result))
	assert.Equal(t, 7_000_000.0, result.TotalCommitted)
	assert.Equal(t, 3_000_000.0, result.GapUSD)
	assert.Equal(t, 30.0, result.GapPercent)
}

func TestAggregateLiquidity_AllDefaulted(t *testing.T) {
	a := &Activities{}
	env := newTestEnv(t, a)

	val, err := env.ExecuteActivity(a.AggregateLiquidity, models.AggregateLiquidityInput{
		CallID:          "call-1",
		TargetAmountUSD: 5_000_000,
		LPResponses: []models.LPResponse{
			{LPID: "lp-01", Status: "defaulted", AmountUSD: 0},
		},
	})
	require.NoError(t, err)

	var result models.AggregateLiquidityResult
	require.NoError(t, val.Get(&result))
	assert.Equal(t, 0.0, result.TotalCommitted)
	assert.Equal(t, 5_000_000.0, result.GapUSD)
	assert.Equal(t, 100.0, result.GapPercent)
}

// ── ScoreLPPortfolio ───────────────────────────────────────────────────────

func TestScoreLPPortfolio_Concentration(t *testing.T) {
	a := &Activities{}
	env := newTestEnv(t, a)

	val, err := env.ExecuteActivity(a.ScoreLPPortfolio, models.ScoreLPPortfolioInput{
		CallID: "call-1",
		LPResponses: []models.LPResponse{
			{LPID: "lp-01", AmountUSD: 5_000_000, RiskScore: 0.8},
			{LPID: "lp-02", AmountUSD: 5_000_000, RiskScore: 0.3},
		},
	})
	require.NoError(t, err)

	var result models.PortfolioRisk
	require.NoError(t, val.Get(&result))
	// HHI for two equal shares: 0.5^2 + 0.5^2 = 0.5
	assert.Equal(t, 0.5, result.ConcentrationScore)
	assert.Contains(t, result.TopRiskyLPs, "lp-01")
}

func TestScoreLPPortfolio_SingleLP(t *testing.T) {
	a := &Activities{}
	env := newTestEnv(t, a)

	val, err := env.ExecuteActivity(a.ScoreLPPortfolio, models.ScoreLPPortfolioInput{
		CallID: "call-1",
		LPResponses: []models.LPResponse{
			{LPID: "lp-01", AmountUSD: 10_000_000, RiskScore: 0.1},
		},
	})
	require.NoError(t, err)

	var result models.PortfolioRisk
	require.NoError(t, val.Get(&result))
	// HHI for single LP: 1.0 (maximum concentration)
	assert.Equal(t, 1.0, result.ConcentrationScore)
}

// ── IssueCapitalCall ───────────────────────────────────────────────────────

func TestIssueCapitalCall_ProRataCalculation(t *testing.T) {
	a := &Activities{} // No DB — will skip Postgres insert
	env := newTestEnv(t, a)

	val, err := env.ExecuteActivity(a.IssueCapitalCall, models.IssueCapitalCallInput{
		CallID:          "call-1",
		FundID:          "fund-1",
		TargetAmountUSD: 10_000_000,
		LPList: []models.LP{
			{LPID: "lp-01", CommitmentUSD: 6_000_000},
			{LPID: "lp-02", CommitmentUSD: 4_000_000},
		},
	})
	require.NoError(t, err)

	var result models.IssueCapitalCallResult
	require.NoError(t, val.Get(&result))
	assert.Equal(t, "call-1", result.CallID)
	assert.Equal(t, 6_000_000.0, result.DrawAmounts["lp-01"])
	assert.Equal(t, 4_000_000.0, result.DrawAmounts["lp-02"])
}

func TestIssueCapitalCall_EqualCommitments(t *testing.T) {
	a := &Activities{}
	env := newTestEnv(t, a)

	val, err := env.ExecuteActivity(a.IssueCapitalCall, models.IssueCapitalCallInput{
		CallID:          "call-2",
		FundID:          "fund-1",
		TargetAmountUSD: 9_000_000,
		LPList: []models.LP{
			{LPID: "lp-01", CommitmentUSD: 3_000_000},
			{LPID: "lp-02", CommitmentUSD: 3_000_000},
			{LPID: "lp-03", CommitmentUSD: 3_000_000},
		},
	})
	require.NoError(t, err)

	var result models.IssueCapitalCallResult
	require.NoError(t, val.Get(&result))
	for _, lpID := range []string{"lp-01", "lp-02", "lp-03"} {
		assert.Equal(t, 3_000_000.0, result.DrawAmounts[lpID])
	}
}

// ── NotifyLPs ──────────────────────────────────────────────────────────────

func TestNotifyLPs_SendsToAllLPs(t *testing.T) {
	received := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	}))
	defer srv.Close()

	a := &Activities{SESURL: srv.URL}
	env := newTestEnv(t, a)

	_, err := env.ExecuteActivity(a.NotifyLPs, models.NotifyLPsInput{
		CallID: "call-1",
		LPList: []models.LP{
			{LPID: "lp-01", Email: "a@test.com", CommitmentUSD: 1_000_000},
			{LPID: "lp-02", Email: "b@test.com", CommitmentUSD: 2_000_000},
			{LPID: "lp-03", Email: "c@test.com", CommitmentUSD: 3_000_000},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 3, received)
}

// ── PredictDefaultRisk ─────────────────────────────────────────────────────

func TestPredictDefaultRisk_ReturnsScores(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scores := []map[string]interface{}{
			{"lpId": "lp-01", "riskScore": 0.15},
			{"lpId": "lp-02", "riskScore": 0.82},
		}
		json.NewEncoder(w).Encode(scores)
	}))
	defer srv.Close()

	a := &Activities{MLScorerURL: srv.URL}
	env := newTestEnv(t, a)

	val, err := env.ExecuteActivity(a.PredictDefaultRisk, models.PredictDefaultRiskInput{
		CallID: "call-1",
		LPResponses: []models.LPResponse{
			{LPID: "lp-01", Status: "committed", AmountUSD: 5_000_000},
			{LPID: "lp-02", Status: "committed", AmountUSD: 3_000_000},
		},
	})
	require.NoError(t, err)

	var result []models.LPResponse
	require.NoError(t, val.Get(&result))
	require.Len(t, result, 2)
	assert.Equal(t, 0.15, result[0].RiskScore)
	assert.Equal(t, 0.82, result[1].RiskScore)
	// Original fields preserved
	assert.Equal(t, "committed", result[0].Status)
	assert.Equal(t, 5_000_000.0, result[0].AmountUSD)
}

// ── TriggerBridge ──────────────────────────────────────────────────────────

func TestTriggerBridge_ReturnsConfirmation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"confirmationId": "BRIDGE-123",
			"drawdownUSD":    req["amount"],
			"feeUSD":         req["amount"].(float64) * 0.015,
		})
	}))
	defer srv.Close()

	a := &Activities{CreditFacURL: srv.URL}
	env := newTestEnv(t, a)

	val, err := env.ExecuteActivity(a.TriggerBridge, models.TriggerBridgeInput{
		CallID: "call-1",
		GapUSD: 2_000_000,
	})
	require.NoError(t, err)

	var result models.TriggerBridgeResult
	require.NoError(t, val.Get(&result))
	assert.Equal(t, "BRIDGE-123", result.ConfirmationID)
	assert.Equal(t, 2_000_000.0, result.DrawdownUSD)
	assert.Equal(t, 30_000.0, result.FeeUSD)
}

// ── AutoFollowUp ───────────────────────────────────────────────────────────

func TestAutoFollowUp_PostsToSES(t *testing.T) {
	var receivedPayload map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := &Activities{SESURL: srv.URL}
	env := newTestEnv(t, a)

	_, err := env.ExecuteActivity(a.AutoFollowUp, models.AutoFollowUpInput{
		CallID: "call-1",
		LPID:   "lp-05",
		Email:  "lp05@test.com",
		Stage:  2,
	})
	require.NoError(t, err)
	assert.Equal(t, "lp-05", receivedPayload["lpId"])
	assert.Equal(t, float64(2), receivedPayload["stage"])
	assert.Equal(t, "followup", receivedPayload["type"])
}

// ── EmitLiquidityReport ────────────────────────────────────────────────────

func TestEmitLiquidityReport_ReturnsPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"reportPath": "/reports/call-1.json",
		})
	}))
	defer srv.Close()

	a := &Activities{ReportGenURL: srv.URL}
	env := newTestEnv(t, a)

	val, err := env.ExecuteActivity(a.EmitLiquidityReport, models.EmitReportInput{
		CallID: "call-1",
		FundID: "fund-1",
		CallResult: models.CapitalCallResult{
			CallID:          "call-1",
			TotalCommitted:  10_000_000,
			TargetAmountUSD: 10_000_000,
		},
	})
	require.NoError(t, err)

	var path string
	require.NoError(t, val.Get(&path))
	assert.Equal(t, "/reports/call-1.json", path)
}

// ── EscalateToGP ───────────────────────────────────────────────────────────

func TestEscalateToGP_PostsToSES(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := &Activities{SESURL: srv.URL}
	env := newTestEnv(t, a)

	_, err := env.ExecuteActivity(a.EscalateToGP, models.EscalateToGPInput{
		CallID: "call-1",
		LP:     models.LP{LPID: "lp-08", CommitmentUSD: 8_000_000, Email: "lp08@test.com"},
		Risk:   0.85,
	})
	require.NoError(t, err)
	assert.True(t, called)
}

// ── SettleAndReconcile (no DB) ─────────────────────────────────────────────

func TestSettleAndReconcile_NoDB(t *testing.T) {
	a := &Activities{} // No DB — skips Postgres operations
	env := newTestEnv(t, a)

	_, err := env.ExecuteActivity(a.SettleAndReconcile, models.SettleAndReconcileInput{
		CallID: "call-1",
		LPResponses: []models.LPResponse{
			{LPID: "lp-01", Status: "committed", AmountUSD: 5_000_000},
		},
	})
	require.NoError(t, err)
}

// ── UpdateLiveAggregates (no DB) ───────────────────────────────────────────

func TestUpdateLiveAggregates_NoDB(t *testing.T) {
	a := &Activities{} // No DB — skips Postgres operations
	env := newTestEnv(t, a)

	_, err := env.ExecuteActivity(a.UpdateLiveAggregates, models.UpdateLiveAggregatesInput{
		CallID:    "call-1",
		LPID:      "lp-01",
		Status:    "committed",
		AmountUSD: 5_000_000,
	})
	require.NoError(t, err)
}

// ── MarkCallCancelled (no DB) ──────────────────────────────────────────────

func TestMarkCallCancelled_NoDB(t *testing.T) {
	a := &Activities{} // No DB — skips Postgres operations
	env := newTestEnv(t, a)

	_, err := env.ExecuteActivity(a.MarkCallCancelled, "call-1")
	require.NoError(t, err)
}

// ── SendEnforcementWarning ─────────────────────────────────────────────────

func TestSendEnforcementWarning_PostsToSES(t *testing.T) {
	called := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := &Activities{SESURL: srv.URL}
	env := newTestEnv(t, a)

	_, err := env.ExecuteActivity(a.SendEnforcementWarning, models.EnforcementWarningInput{
		CallID: "call-1",
		LP:     models.LP{LPID: "lp-08", CommitmentUSD: 8_000_000, Email: "lp08@test.com"},
		GPName: "TestGP",
	})
	require.NoError(t, err)
	// Expect 2 calls: one to LP, one to GP channel
	assert.Equal(t, 2, called)
}

// ── Edge case: unused context import guard ─────────────────────────────────

var _ = context.Background // ensure context import is used
