package workflows

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"

	"github.com/vishworks/assetmgmt/internal/activities"
	"github.com/vishworks/assetmgmt/internal/models"
)

// CapitalCallWorkflowTestSuite ─────────────────────────────────────────────
type CapitalCallWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env *testsuite.TestWorkflowEnvironment
}

func (s *CapitalCallWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.RegisterWorkflow(CapitalCallWorkflow)
	s.env.RegisterWorkflow(LPResponseWorkflow)

	var a *activities.Activities
	s.env.RegisterActivity(a)
}

func TestCapitalCallWorkflowSuite(t *testing.T) {
	suite.Run(t, new(CapitalCallWorkflowTestSuite))
}

// helper builds a standard 3-LP input.
func testInput() models.CapitalCallInput {
	return models.CapitalCallInput{
		CallID:          "test-call-1",
		FundID:          "fund-1",
		TargetAmountUSD: 10_000_000,
		LPList: []models.LP{
			{LPID: "lp-01", CommitmentUSD: 4_000_000, Email: "lp01@test.com"},
			{LPID: "lp-02", CommitmentUSD: 3_000_000, Email: "lp02@test.com"},
			{LPID: "lp-03", CommitmentUSD: 3_000_000, Email: "lp03@test.com"},
		},
		DeadlineDays:  10,
		SecondsPerDay: 1, // fast timers for tests
	}
}

// TestAllLPsCommit verifies the happy path: every LP commits, no gap,
// no bridge, no GP escalation.
func (s *CapitalCallWorkflowTestSuite) TestAllLPsCommit() {
	input := testInput()

	// Mock activities
	s.env.OnActivity("IssueCapitalCall", mock.Anything, mock.Anything).Return(
		&models.IssueCapitalCallResult{
			CallID: "test-call-1",
			DrawAmounts: map[string]float64{
				"lp-01": 4_000_000,
				"lp-02": 3_000_000,
				"lp-03": 3_000_000,
			},
		}, nil,
	)
	s.env.OnActivity("NotifyLPs", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("AutoFollowUp", mock.Anything, mock.Anything).Return(nil).Maybe()
	// UpdateLiveAggregates is called by child workflows on commit/default.
	s.env.OnActivity("UpdateLiveAggregates", mock.Anything, mock.Anything).Return(nil).Maybe()

	s.env.OnActivity("AggregateLiquidity", mock.Anything, mock.Anything).Return(
		&models.AggregateLiquidityResult{
			TotalCommitted: 10_000_000,
			GapUSD:         0,
			GapPercent:     0,
		}, nil,
	)

	s.env.OnActivity("PredictDefaultRisk", mock.Anything, mock.Anything).Return(
		[]models.LPResponse{
			{LPID: "lp-01", Status: "committed", AmountUSD: 4_000_000, RiskScore: 0.1},
			{LPID: "lp-02", Status: "committed", AmountUSD: 3_000_000, RiskScore: 0.2},
			{LPID: "lp-03", Status: "committed", AmountUSD: 3_000_000, RiskScore: 0.15},
		}, nil,
	)

	s.env.OnActivity("ScoreLPPortfolio", mock.Anything, mock.Anything).Return(
		&models.PortfolioRisk{ConcentrationScore: 0.34, TopRiskyLPs: []string{}}, nil,
	)

	s.env.OnActivity("SettleAndReconcile", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("EmitLiquidityReport", mock.Anything, mock.Anything).Return(
		"/reports/test-call-1.json", nil,
	)

	// All child workflows get commitment signals immediately
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflowByID("lp-response-test-call-1-lp-01", SignalLPCommitment,
			models.LPCommitmentSignal{LPID: "lp-01", AmountUSD: 4_000_000})
		s.env.SignalWorkflowByID("lp-response-test-call-1-lp-02", SignalLPCommitment,
			models.LPCommitmentSignal{LPID: "lp-02", AmountUSD: 3_000_000})
		s.env.SignalWorkflowByID("lp-response-test-call-1-lp-03", SignalLPCommitment,
			models.LPCommitmentSignal{LPID: "lp-03", AmountUSD: 3_000_000})
	}, 0)

	s.env.ExecuteWorkflow(CapitalCallWorkflow, input)

	require.True(s.T(), s.env.IsWorkflowCompleted())
	require.NoError(s.T(), s.env.GetWorkflowError())

	var result models.CapitalCallResult
	require.NoError(s.T(), s.env.GetWorkflowResult(&result))
	require.Equal(s.T(), "test-call-1", result.CallID)
	require.Equal(s.T(), 10_000_000.0, result.TotalCommitted)
	require.Equal(s.T(), 0.0, result.GapUSD)
	require.False(s.T(), result.BridgeTriggered)
	require.Equal(s.T(), "/reports/test-call-1.json", result.ReportPath)
}

// TestBridgeTriggered verifies that a >10% liquidity gap triggers the bridge.
func (s *CapitalCallWorkflowTestSuite) TestBridgeTriggered() {
	input := testInput()

	s.env.OnActivity("IssueCapitalCall", mock.Anything, mock.Anything).Return(
		&models.IssueCapitalCallResult{CallID: "test-call-1", DrawAmounts: map[string]float64{}}, nil,
	)
	s.env.OnActivity("NotifyLPs", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("AutoFollowUp", mock.Anything, mock.Anything).Return(nil).Maybe()
	// UpdateLiveAggregates is called by child workflows on commit/default.
	s.env.OnActivity("UpdateLiveAggregates", mock.Anything, mock.Anything).Return(nil).Maybe()

	// LP-03 defaults → 30% gap
	s.env.OnActivity("AggregateLiquidity", mock.Anything, mock.Anything).Return(
		&models.AggregateLiquidityResult{
			TotalCommitted: 7_000_000,
			GapUSD:         3_000_000,
			GapPercent:     30.0,
		}, nil,
	)

	s.env.OnActivity("PredictDefaultRisk", mock.Anything, mock.Anything).Return(
		[]models.LPResponse{
			{LPID: "lp-01", Status: "committed", AmountUSD: 4_000_000, RiskScore: 0.1},
			{LPID: "lp-02", Status: "committed", AmountUSD: 3_000_000, RiskScore: 0.2},
			{LPID: "lp-03", Status: "defaulted", AmountUSD: 0, RiskScore: 0.6},
		}, nil,
	)

	s.env.OnActivity("ScoreLPPortfolio", mock.Anything, mock.Anything).Return(
		&models.PortfolioRisk{ConcentrationScore: 0.40}, nil,
	)

	// Bridge should be triggered
	s.env.OnActivity("TriggerBridge", mock.Anything, mock.Anything).Return(
		&models.TriggerBridgeResult{
			ConfirmationID: "BRIDGE-123",
			DrawdownUSD:    3_000_000,
			FeeUSD:         45_000,
		}, nil,
	)

	s.env.OnActivity("SettleAndReconcile", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("EmitLiquidityReport", mock.Anything, mock.Anything).Return("/reports/test-call-1.json", nil)

	// Send signals for LP-01 and LP-02; LP-03 will time out
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflowByID("lp-response-test-call-1-lp-01", SignalLPCommitment,
			models.LPCommitmentSignal{LPID: "lp-01", AmountUSD: 4_000_000})
		s.env.SignalWorkflowByID("lp-response-test-call-1-lp-02", SignalLPCommitment,
			models.LPCommitmentSignal{LPID: "lp-02", AmountUSD: 3_000_000})
	}, 0)

	s.env.ExecuteWorkflow(CapitalCallWorkflow, input)

	require.True(s.T(), s.env.IsWorkflowCompleted())
	require.NoError(s.T(), s.env.GetWorkflowError())

	var result models.CapitalCallResult
	require.NoError(s.T(), s.env.GetWorkflowResult(&result))
	require.True(s.T(), result.BridgeTriggered)
	require.NotNil(s.T(), result.BridgeResult)
	require.Equal(s.T(), 3_000_000.0, result.BridgeResult.DrawdownUSD)
}

// TestGPEscalation verifies that a high-risk LP triggers GP escalation
// and the workflow waits for the GP decision signal.
func (s *CapitalCallWorkflowTestSuite) TestGPEscalation() {
	input := testInput()

	s.env.OnActivity("IssueCapitalCall", mock.Anything, mock.Anything).Return(
		&models.IssueCapitalCallResult{CallID: "test-call-1", DrawAmounts: map[string]float64{}}, nil,
	)
	s.env.OnActivity("NotifyLPs", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("AutoFollowUp", mock.Anything, mock.Anything).Return(nil).Maybe()
	// UpdateLiveAggregates is called by child workflows on commit/default.
	s.env.OnActivity("UpdateLiveAggregates", mock.Anything, mock.Anything).Return(nil).Maybe()

	s.env.OnActivity("AggregateLiquidity", mock.Anything, mock.Anything).Return(
		&models.AggregateLiquidityResult{TotalCommitted: 10_000_000, GapUSD: 0, GapPercent: 0}, nil,
	)

	// LP-02 is high risk (0.85)
	s.env.OnActivity("PredictDefaultRisk", mock.Anything, mock.Anything).Return(
		[]models.LPResponse{
			{LPID: "lp-01", Status: "committed", AmountUSD: 4_000_000, RiskScore: 0.1},
			{LPID: "lp-02", Status: "committed", AmountUSD: 3_000_000, RiskScore: 0.85},
			{LPID: "lp-03", Status: "committed", AmountUSD: 3_000_000, RiskScore: 0.15},
		}, nil,
	)

	s.env.OnActivity("ScoreLPPortfolio", mock.Anything, mock.Anything).Return(
		&models.PortfolioRisk{ConcentrationScore: 0.34, TopRiskyLPs: []string{"lp-02"}}, nil,
	)

	s.env.OnActivity("EscalateToGP", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("SettleAndReconcile", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("EmitLiquidityReport", mock.Anything, mock.Anything).Return("/reports/test-call-1.json", nil)

	// All LPs commit immediately
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflowByID("lp-response-test-call-1-lp-01", SignalLPCommitment,
			models.LPCommitmentSignal{LPID: "lp-01", AmountUSD: 4_000_000})
		s.env.SignalWorkflowByID("lp-response-test-call-1-lp-02", SignalLPCommitment,
			models.LPCommitmentSignal{LPID: "lp-02", AmountUSD: 3_000_000})
		s.env.SignalWorkflowByID("lp-response-test-call-1-lp-03", SignalLPCommitment,
			models.LPCommitmentSignal{LPID: "lp-03", AmountUSD: 3_000_000})
	}, 0)

	// GP waives the high-risk LP after escalation
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow(SignalGPDecision, models.GPDecisionSignal{
			LPID: "lp-02", Action: "waive", GPName: "TestGP",
		})
	}, 0)

	s.env.ExecuteWorkflow(CapitalCallWorkflow, input)

	require.True(s.T(), s.env.IsWorkflowCompleted())
	require.NoError(s.T(), s.env.GetWorkflowError())

	var result models.CapitalCallResult
	require.NoError(s.T(), s.env.GetWorkflowResult(&result))

	// LP-02 was waived, so should still be committed
	for _, lp := range result.LPResponses {
		if lp.LPID == "lp-02" {
			require.Equal(s.T(), "committed", lp.Status)
		}
	}
}

// TestDefaultedHighRiskLPNotEscalated verifies that a defaulted LP whose
// ML risk score exceeds 0.7 is NOT escalated to the GP. A defaulted LP has
// no commitment to waive or enforce, so waiting for a GP decision would
// deadlock the workflow.
func (s *CapitalCallWorkflowTestSuite) TestDefaultedHighRiskLPNotEscalated() {
	input := testInput()

	s.env.OnActivity("IssueCapitalCall", mock.Anything, mock.Anything).Return(
		&models.IssueCapitalCallResult{CallID: "test-call-1", DrawAmounts: map[string]float64{}}, nil,
	)
	s.env.OnActivity("NotifyLPs", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("AutoFollowUp", mock.Anything, mock.Anything).Return(nil).Maybe()
	// UpdateLiveAggregates is called by child workflows on commit/default.
	s.env.OnActivity("UpdateLiveAggregates", mock.Anything, mock.Anything).Return(nil).Maybe()

	s.env.OnActivity("AggregateLiquidity", mock.Anything, mock.Anything).Return(
		&models.AggregateLiquidityResult{TotalCommitted: 7_000_000, GapUSD: 3_000_000, GapPercent: 30.0}, nil,
	)

	// LP-03 defaulted with a high risk score — must NOT trigger GP escalation
	s.env.OnActivity("PredictDefaultRisk", mock.Anything, mock.Anything).Return(
		[]models.LPResponse{
			{LPID: "lp-01", Status: "committed", AmountUSD: 4_000_000, RiskScore: 0.1},
			{LPID: "lp-02", Status: "committed", AmountUSD: 3_000_000, RiskScore: 0.2},
			{LPID: "lp-03", Status: "defaulted", AmountUSD: 0, RiskScore: 0.85},
		}, nil,
	)

	s.env.OnActivity("ScoreLPPortfolio", mock.Anything, mock.Anything).Return(
		&models.PortfolioRisk{ConcentrationScore: 0.40}, nil,
	)
	s.env.OnActivity("TriggerBridge", mock.Anything, mock.Anything).Return(
		&models.TriggerBridgeResult{ConfirmationID: "BRIDGE-001", DrawdownUSD: 3_000_000, FeeUSD: 45_000}, nil,
	)
	// EscalateToGP must NOT be called — if it were, the workflow would hang
	// waiting for a GP decision that this test never sends.
	s.env.OnActivity("EscalateToGP", mock.Anything, mock.Anything).Return(nil).Maybe()
	s.env.OnActivity("SettleAndReconcile", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("EmitLiquidityReport", mock.Anything, mock.Anything).Return("/reports/test-call-1.json", nil)

	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflowByID("lp-response-test-call-1-lp-01", SignalLPCommitment,
			models.LPCommitmentSignal{LPID: "lp-01", AmountUSD: 4_000_000})
		s.env.SignalWorkflowByID("lp-response-test-call-1-lp-02", SignalLPCommitment,
			models.LPCommitmentSignal{LPID: "lp-02", AmountUSD: 3_000_000})
		// lp-03 does not commit — times out and becomes defaulted
	}, 0)

	s.env.ExecuteWorkflow(CapitalCallWorkflow, input)

	require.True(s.T(), s.env.IsWorkflowCompleted())
	require.NoError(s.T(), s.env.GetWorkflowError())

	var result models.CapitalCallResult
	require.NoError(s.T(), s.env.GetWorkflowResult(&result))
	require.True(s.T(), result.BridgeTriggered)

	for _, lp := range result.LPResponses {
		if lp.LPID == "lp-03" {
			require.Equal(s.T(), "defaulted", lp.Status)
		}
	}
}

// TestGPEnforce verifies the correct enforce semantics:
// GP enforcement is a governance/compliance warning action only.
// The LP's status remains "committed", the contribution amount is unchanged,
// and a warning email is sent via SendEnforcementWarning.
// Enforce does NOT remove the LP from settlement or reduce aggregate liquidity.
func (s *CapitalCallWorkflowTestSuite) TestGPEnforce() {
	input := testInput()

	s.env.OnActivity("IssueCapitalCall", mock.Anything, mock.Anything).Return(
		&models.IssueCapitalCallResult{CallID: "test-call-1", DrawAmounts: map[string]float64{}}, nil,
	)
	s.env.OnActivity("NotifyLPs", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("AutoFollowUp", mock.Anything, mock.Anything).Return(nil).Maybe()
	// UpdateLiveAggregates is called by child workflows on commit/default.
	s.env.OnActivity("UpdateLiveAggregates", mock.Anything, mock.Anything).Return(nil).Maybe()

	s.env.OnActivity("AggregateLiquidity", mock.Anything, mock.Anything).Return(
		&models.AggregateLiquidityResult{TotalCommitted: 10_000_000, GapUSD: 0, GapPercent: 0}, nil,
	)

	s.env.OnActivity("PredictDefaultRisk", mock.Anything, mock.Anything).Return(
		[]models.LPResponse{
			{LPID: "lp-01", Status: "committed", AmountUSD: 4_000_000, RiskScore: 0.1},
			{LPID: "lp-02", Status: "committed", AmountUSD: 3_000_000, RiskScore: 0.85},
			{LPID: "lp-03", Status: "committed", AmountUSD: 3_000_000, RiskScore: 0.15},
		}, nil,
	)

	s.env.OnActivity("ScoreLPPortfolio", mock.Anything, mock.Anything).Return(
		&models.PortfolioRisk{ConcentrationScore: 0.34}, nil,
	)

	s.env.OnActivity("EscalateToGP", mock.Anything, mock.Anything).Return(nil)
	// SendEnforcementWarning sends the compliance warning email; it must be called
	// when GP chooses "enforce". It does NOT alter the LP state.
	s.env.OnActivity("SendEnforcementWarning", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("SettleAndReconcile", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("EmitLiquidityReport", mock.Anything, mock.Anything).Return("/reports/test-call-1.json", nil)

	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflowByID("lp-response-test-call-1-lp-01", SignalLPCommitment,
			models.LPCommitmentSignal{LPID: "lp-01", AmountUSD: 4_000_000})
		s.env.SignalWorkflowByID("lp-response-test-call-1-lp-02", SignalLPCommitment,
			models.LPCommitmentSignal{LPID: "lp-02", AmountUSD: 3_000_000})
		s.env.SignalWorkflowByID("lp-response-test-call-1-lp-03", SignalLPCommitment,
			models.LPCommitmentSignal{LPID: "lp-03", AmountUSD: 3_000_000})
	}, 0)

	// GP enforces → compliance warning email sent; LP-02 stays committed
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow(SignalGPDecision, models.GPDecisionSignal{
			LPID: "lp-02", Action: "enforce", GPName: "TestGP",
		})
	}, 0)

	s.env.ExecuteWorkflow(CapitalCallWorkflow, input)

	require.True(s.T(), s.env.IsWorkflowCompleted())
	require.NoError(s.T(), s.env.GetWorkflowError())

	var result models.CapitalCallResult
	require.NoError(s.T(), s.env.GetWorkflowResult(&result))

	// Enforce is a governance-only action: LP-02 remains committed with its
	// original contribution intact. Only true non-responders become defaulted.
	for _, lp := range result.LPResponses {
		if lp.LPID == "lp-02" {
			require.Equal(s.T(), "committed", lp.Status,
				"enforce should keep LP status as committed")
			require.Equal(s.T(), 3_000_000.0, lp.AmountUSD,
				"enforce should not zero out the LP contribution")
		}
	}
}
