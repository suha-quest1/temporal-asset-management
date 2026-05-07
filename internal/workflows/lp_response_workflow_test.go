package workflows

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"

	"github.com/vishworks/assetmgmt/internal/activities"
	"github.com/vishworks/assetmgmt/internal/models"
)

type LPResponseWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env *testsuite.TestWorkflowEnvironment
}

func (s *LPResponseWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.RegisterWorkflow(LPResponseWorkflow)

	var a *activities.Activities
	s.env.RegisterActivity(a)
}

func TestLPResponseWorkflowSuite(t *testing.T) {
	suite.Run(t, new(LPResponseWorkflowTestSuite))
}

// TestLPCommitsOnTime verifies that an LP who sends a commitment signal
// before the deadline is recorded as "committed".
func (s *LPResponseWorkflowTestSuite) TestLPCommitsOnTime() {
	input := models.LPResponseInput{
		CallID:        "call-1",
		LPID:          "lp-01",
		CommitmentUSD: 5_000_000,
		Email:         "lp01@test.com",
		DeadlineDays:  10,
		SecondsPerDay: 1,
	}

	// Send signal immediately
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow(SignalLPCommitment, models.LPCommitmentSignal{
			LPID:      "lp-01",
			AmountUSD: 5_000_000,
		})
	}, 0)

	s.env.ExecuteWorkflow(LPResponseWorkflow, input)

	require.True(s.T(), s.env.IsWorkflowCompleted())
	require.NoError(s.T(), s.env.GetWorkflowError())

	var result models.LPResponse
	require.NoError(s.T(), s.env.GetWorkflowResult(&result))
	require.Equal(s.T(), "lp-01", result.LPID)
	require.Equal(s.T(), "committed", result.Status)
	require.Equal(s.T(), 5_000_000.0, result.AmountUSD)
}

// TestLPCommitsAfterFirstFollowUp verifies that an LP who commits after
// the D+3 follow-up (but before D+7) is recorded as committed.
func (s *LPResponseWorkflowTestSuite) TestLPCommitsAfterFirstFollowUp() {
	input := models.LPResponseInput{
		CallID:        "call-1",
		LPID:          "lp-02",
		CommitmentUSD: 3_000_000,
		Email:         "lp02@test.com",
		DeadlineDays:  10,
		SecondsPerDay: 1, // 1 second per "day"
	}

	s.env.OnActivity("AutoFollowUp", mock.Anything, mock.MatchedBy(func(inp models.AutoFollowUpInput) bool {
		return inp.Stage == 1
	})).Return(nil)
	s.env.OnActivity("AutoFollowUp", mock.Anything, mock.Anything).Return(nil).Maybe()

	// LP commits at "day 5" (5 seconds in demo mode), which is after the D+3 follow-up
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow(SignalLPCommitment, models.LPCommitmentSignal{
			LPID:      "lp-02",
			AmountUSD: 3_000_000,
		})
	}, 5*time.Second)

	s.env.ExecuteWorkflow(LPResponseWorkflow, input)

	require.True(s.T(), s.env.IsWorkflowCompleted())
	require.NoError(s.T(), s.env.GetWorkflowError())

	var result models.LPResponse
	require.NoError(s.T(), s.env.GetWorkflowResult(&result))
	require.Equal(s.T(), "committed", result.Status)
	require.Equal(s.T(), 3_000_000.0, result.AmountUSD)
}

// TestLPDefaultsAfterAllFollowUps verifies that an LP who never responds
// goes through all 3 follow-up stages and ends as "defaulted".
func (s *LPResponseWorkflowTestSuite) TestLPDefaultsAfterAllFollowUps() {
	input := models.LPResponseInput{
		CallID:        "call-1",
		LPID:          "lp-03",
		CommitmentUSD: 2_000_000,
		Email:         "lp03@test.com",
		DeadlineDays:  15,
		SecondsPerDay: 1,
	}

	// Expect 3 follow-up calls: stage 1 (D+3), stage 2 (D+7), stage 3 (D+10)
	s.env.OnActivity("AutoFollowUp", mock.Anything, mock.MatchedBy(func(inp models.AutoFollowUpInput) bool {
		return inp.Stage == 1
	})).Return(nil).Once()
	s.env.OnActivity("AutoFollowUp", mock.Anything, mock.MatchedBy(func(inp models.AutoFollowUpInput) bool {
		return inp.Stage == 2
	})).Return(nil).Once()
	s.env.OnActivity("AutoFollowUp", mock.Anything, mock.MatchedBy(func(inp models.AutoFollowUpInput) bool {
		return inp.Stage == 3
	})).Return(nil).Once()

	// No signal sent — LP never responds
	s.env.ExecuteWorkflow(LPResponseWorkflow, input)

	require.True(s.T(), s.env.IsWorkflowCompleted())
	require.NoError(s.T(), s.env.GetWorkflowError())

	var result models.LPResponse
	require.NoError(s.T(), s.env.GetWorkflowResult(&result))
	require.Equal(s.T(), "lp-03", result.LPID)
	require.Equal(s.T(), "defaulted", result.Status)
	require.Equal(s.T(), 0.0, result.AmountUSD)
}

// TestLPCommitsLateAfterAllFollowUps verifies that an LP can still commit
// after all follow-ups but before the final deadline.
func (s *LPResponseWorkflowTestSuite) TestLPCommitsLateAfterAllFollowUps() {
	input := models.LPResponseInput{
		CallID:        "call-1",
		LPID:          "lp-04",
		CommitmentUSD: 1_000_000,
		Email:         "lp04@test.com",
		DeadlineDays:  15,
		SecondsPerDay: 1,
	}

	s.env.OnActivity("AutoFollowUp", mock.Anything, mock.Anything).Return(nil).Maybe()

	// LP commits at "day 12" — after all follow-ups (D+10) but before deadline (D+15)
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow(SignalLPCommitment, models.LPCommitmentSignal{
			LPID:      "lp-04",
			AmountUSD: 1_000_000,
		})
	}, 12*time.Second)

	s.env.ExecuteWorkflow(LPResponseWorkflow, input)

	require.True(s.T(), s.env.IsWorkflowCompleted())
	require.NoError(s.T(), s.env.GetWorkflowError())

	var result models.LPResponse
	require.NoError(s.T(), s.env.GetWorkflowResult(&result))
	require.Equal(s.T(), "committed", result.Status)
	require.Equal(s.T(), 1_000_000.0, result.AmountUSD)
}
