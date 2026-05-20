package models

// LP represents a Limited Partner in the fund.
type LP struct {
	LPID          string  `json:"lpId"`
	CommitmentUSD float64 `json:"commitmentUSD"`
	Email         string  `json:"email"`
}

//!!!- why is there no GP struct?

// CapitalCallInput is the input to the parent CapitalCallWorkflow.
type CapitalCallInput struct {
	CallID          string  `json:"callId"`
	FundID          string  `json:"fundId"`
	TargetAmountUSD float64 `json:"targetAmountUSD"`
	LPList          []LP    `json:"lpList"`
	DeadlineDays    int     `json:"deadlineDays"`
	// SecondsPerDay overrides the day duration for demo mode.
	// If 0, defaults to 86400 (real days).
	SecondsPerDay int `json:"secondsPerDay,omitempty"`
}

// CapitalCallResult is the final output of CapitalCallWorkflow.
type CapitalCallResult struct {
	CallID          string               `json:"callId"`
	TotalCommitted  float64              `json:"totalCommitted"`
	TargetAmountUSD float64              `json:"targetAmountUSD"`
	GapUSD          float64              `json:"gapUSD"`
	GapPercent      float64              `json:"gapPercent"`
	BridgeTriggered bool                 `json:"bridgeTriggered"`
	BridgeResult    *TriggerBridgeResult `json:"bridgeResult,omitempty"`
	LPResponses     []LPResponse         `json:"lpResponses"`
	PortfolioRisk   PortfolioRisk        `json:"portfolioRisk"`
	ReportPath      string               `json:"reportPath"`
}

// LPResponse represents the outcome of a single LP's participation.
type LPResponse struct {
	LPID      string  `json:"lpId"`
	Status    string  `json:"status"` // "committed" | "defaulted"
	AmountUSD float64 `json:"amountUSD"`
	RiskScore float64 `json:"riskScore"`
}

// LPResponseInput is the input to the child LPResponseWorkflow.
type LPResponseInput struct {
	CallID        string  `json:"callId"`
	LPID          string  `json:"lpId"`
	CommitmentUSD float64 `json:"commitmentUSD"`
	Email         string  `json:"email"`
	DeadlineDays  int     `json:"deadlineDays"`
	// SecondsPerDay overrides the day duration for demo mode.
	SecondsPerDay int `json:"secondsPerDay,omitempty"`
}

// LPCommitmentSignal is sent by an LP to confirm their commitment.
type LPCommitmentSignal struct {
	LPID      string  `json:"lpId"`
	AmountUSD float64 `json:"amountUSD"`
}

// GPDecisionSignal is sent by a GP to decide on a high-risk LP.
type GPDecisionSignal struct {
	LPID   string `json:"lpId"`
	Action string `json:"action"` // "waive" | "enforce"
	GPName string `json:"gpName"`
}

// --- Activity inputs and outputs ---

type IssueCapitalCallInput struct {
	CallID          string  `json:"callId"`
	FundID          string  `json:"fundId"`
	TargetAmountUSD float64 `json:"targetAmountUSD"`
	LPList          []LP    `json:"lpList"`
	DeadlineDays    int     `json:"deadlineDays"`
}

type IssueCapitalCallResult struct {
	CallID      string             `json:"callId"`
	DrawAmounts map[string]float64 `json:"drawAmounts"`
}

type NotifyLPsInput struct {
	CallID string `json:"callId"`
	LPList []LP   `json:"lpList"`
}

type AutoFollowUpInput struct {
	CallID string `json:"callId"`
	LPID   string `json:"lpId"`
	Email  string `json:"email"`
	Stage  int    `json:"stage"` // 1=email D+3, 2=call D+7, 3=legal D+10
}

type PredictDefaultRiskInput struct {
	CallID      string       `json:"callId"`
	LPResponses []LPResponse `json:"lpResponses"`
}

type AggregateLiquidityInput struct {
	CallID          string       `json:"callId"`
	TargetAmountUSD float64      `json:"targetAmountUSD"`
	LPResponses     []LPResponse `json:"lpResponses"`
}

type AggregateLiquidityResult struct {
	TotalCommitted float64 `json:"totalCommitted"`
	GapUSD         float64 `json:"gapUSD"`
	GapPercent     float64 `json:"gapPercent"`
}

type ScoreLPPortfolioInput struct {
	CallID      string       `json:"callId"`
	LPResponses []LPResponse `json:"lpResponses"`
}

type PortfolioRisk struct {
	ConcentrationScore float64  `json:"concentrationScore"`
	TopRiskyLPs        []string `json:"topRiskyLPs"`
}

type TriggerBridgeInput struct {
	CallID string  `json:"callId"`
	GapUSD float64 `json:"gapUSD"`
}

type TriggerBridgeResult struct {
	ConfirmationID string  `json:"confirmationId"`
	DrawdownUSD    float64 `json:"drawdownUSD"`
	FeeUSD         float64 `json:"feeUSD"`
}

type EscalateToGPInput struct {
	CallID string  `json:"callId"`
	LP     LP      `json:"lp"`
	Risk   float64 `json:"risk"`
}
type EnforcementWarningInput struct {
	CallID string `json:"callId"`
	LP     LP     `json:"lp"`
	GPName string `json:"gpName"`
}

type SettleAndReconcileInput struct {
	CallID       string               `json:"callId"`
	LPResponses  []LPResponse         `json:"lpResponses"`
	BridgeResult *TriggerBridgeResult `json:"bridgeResult,omitempty"`
}

type EmitReportInput struct {
	CallID     string            `json:"callId"`
	FundID     string            `json:"fundId"`
	CallResult CapitalCallResult `json:"callResult"`
}

type UpdateLiveAggregatesInput struct {
	CallID    string  `json:"callId"`
	LPID      string  `json:"lpId"`
	Status    string  `json:"status"`
	AmountUSD float64 `json:"amountUSD"`
}
