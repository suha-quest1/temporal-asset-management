# Asset Management: Intelligent Capital Call & Liquidity Orchestration

A demo of a GP (General Partner) capital call workflow built with [Temporal](https://temporal.io/) and [Go](https://go.dev/). The system notifies LPs (Limited Partners), waits for commitment signals, runs ML risk scoring, autonomously follows up overdue LPs, triggers credit line drawdowns, escalates high-risk LPs for GP decisions, settles, and emits a regulatory-grade report.

## Architecture

```
Capital call lifecycle — one workflow per call event

  issueCapitalCall → notifyLPs → startResponseTimer → awaitLPSignals
                                                            ↓
       scoreLPPortfolio ← aggregateLiquidity ← predictDefaultRisk
            ↓                    ↓                      ↓
       autoFollowUp        triggerBridge          escalateToGP
                                                        ↓
                                                  gpApprovalGate
                                                        ↓
                              settleAndReconcile ←──────┘
                                     ↓
                             emitLiquidityReport
```

**Workflows:**

| Component | Description |
|---|---|
| `CapitalCallWorkflow` | Parent orchestrator — full capital call lifecycle |
| `LPResponseWorkflow` | Child (one per LP) — waits for signal, triggers follow-ups on timeout |

**Activities:**

| Activity | Purpose |
|---|---|
| `IssueCapitalCall` | Creates call record in Postgres, computes pro-rata draws |
| `NotifyLPs` | Batch email via mock SES |
| `AutoFollowUp` | Tiered escalation: email (D+3), call (D+7), legal (D+10) |
| `PredictDefaultRisk` | ML model scores each LP's default probability |
| `AggregateLiquidity` | Computes committed vs target gap |
| `ScoreLPPortfolio` | Concentration risk (HHI) + top risky LPs |
| `TriggerBridge` | Credit line drawdown when gap > 10% |
| `EscalateToGP` | Slack-style alert to GP channel |
| `SettleAndReconcile` | Wire transfer stubs + double-entry ledger |
| `EmitLiquidityReport` | JSON + text report generation |

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/) (Docker Desktop for Mac includes both)
- No Go or Python installation required — everything runs in containers

## Database Initialization & Setup

The database schema and seed data have been consolidated into a clean structure for maintainability:
- `db/schema.sql`: Contains the final, consolidated table definitions, constraints, and indexes.
- `db/seed.sql`: Contains the canonical demo LP records (lp-01 through lp-10).

10 mock LPs are pre-seeded:

- **LP 01–07**: Commit immediately
- **LP 08**: Commits but is flagged high-risk → GP escalation → GP waives
- **LP 09–10**: Never respond → auto follow-up sequence → default

**How to Reseed Locally:**
If you need to reset the database and reseed the data, you can clear the docker volume and restart the database:
```bash
docker compose down -v
docker compose up -d
```

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/suha-quest1/temporal-asset-management.git
cd assetmgmt
```

### 2. Start all services

```bash
docker compose up --build -d
```

This starts 10 services:

| Service | Port | Description |
|---|---|---|
| `temporal` | 7233 | Temporal server |
| `temporal-ui` | [localhost:8080](http://localhost:8080) | Temporal Web UI |
| `postgres` | 5432 | Fund ledger database |
| `postgres-temporal` | — | Temporal's internal database |
| `worker` | — | Temporal worker (workflows + activities) |
| `api-server` | [localhost:8090](http://localhost:8090) | Signal endpoints |
| `mock-ses` | [localhost:8081](http://localhost:8081) | Email log viewer |
| `mock-credit-facility` | [localhost:8082](http://localhost:8082) | Credit facility stub |
| `ml-scorer` | 5001 | ML risk scoring service |
| `report-generator` | 5002 | Report generation service |

### 3. Verify services are healthy

```bash
docker compose ps
```

Wait until `temporal`, `postgres`, and `worker` show as healthy/running.

### 4. Access using the UI

```bash
cd asset-management && npm run dev
```
View the UI at: [localhost:5173](http://localhost:5173)

### To view using Mobile devices

* Ensure your laptop has the setup running 

* Laptop and mobile should be connected to the same local network


* View the npm/vite terminal to see a network URL


 * eg:   Network: http://10.57.97.159:5173/


* Open it from your mobile device

### 5. Explore

- **Temporal UI**: [http://localhost:8080](http://localhost:8080) — inspect workflow executions, see the parent/child relationship, view activity inputs/outputs
- **Email Log**: [http://localhost:8081](http://localhost:8081) — see all notifications, follow-ups, and escalations
- **Reports**: Check `./reports/` for the generated JSON reports

## Running Tests

Tests use the Temporal test suite (in-memory workflow replay) and `httptest` for activity tests. No external services needed.

```bash
# Run from host (requires Go 1.22+)
go test ./internal/... -v

# Or run inside a container
docker run --rm -v $(pwd):/app -w /app golang:1.22-alpine \
  sh -c "apk add git && go test ./internal/... -v"
```

### Test coverage

- **Workflow tests** (`internal/workflows/`):
  - `TestAllLPsCommit` — happy path, zero gap
  - `TestBridgeTriggered` — LP default causes >10% gap, bridge fires
  - `TestGPEscalation` — high-risk LP, GP waives
  - `TestGPEnforce` — high-risk LP, GP enforces default
  - `TestLPCommitsOnTime` — immediate commitment signal
  - `TestLPCommitsAfterFirstFollowUp` — commitment after D+3 follow-up
  - `TestLPDefaultsAfterAllFollowUps` — full escalation, final default
  - `TestLPCommitsLateAfterAllFollowUps` — commitment in final window
  - `TestCancelCall` — verifies early workflow abort on cancel signal
- `TestForceSettlement` — verifies uncommitted LPs default on force settlement signal


- **Activity tests** (`internal/activities/`):
  - `AggregateLiquidity` — all committed / with gap / all defaulted
  - `ScoreLPPortfolio` — HHI calculation, single LP edge case
  - `IssueCapitalCall` — pro-rata draw calculation
  - `NotifyLPs` — verifies all LPs notified
  - `PredictDefaultRisk` — ML scorer integration
  - `TriggerBridge` — credit facility integration
  - `AutoFollowUp` — SES follow-up posting
  - `EmitLiquidityReport` — report generation
  - `EscalateToGP` — GP escalation posting
  - `UpdateLiveAggregates` — projection updates without DB
  - `MarkCallCancelled` — cancel status updates without DB
  - `SendEnforcementWarning` — SES enforcement notice posting

## Project Structure

```
.
├── cmd/
│   ├── apiserver/         API server (signal endpoints)
│   ├── mockcreditfacility/ Mock credit facility
│   ├── mockses/           Mock SES email service
│   └── worker/            Temporal worker
├── db/                    schema.sql and seed.sql
├── internal/
│   ├── activities/        Temporal activity implementations + tests
│   ├── db/                Database connection helper
│   ├── models/            Shared data types
│   └── workflows/         Temporal workflow implementations + tests
├── services/
│   ├── ml-scorer/         Python Flask ML scoring service
│   └── report-generator/  Python Flask report service
├── reports/               Generated reports (mounted volume)
├── Dockerfile             Multi-stage Go build
├── docker-compose.yml     Full stack (10 services)
└── README.md
```

## Teardown

```bash
docker compose --profile demo down -v
```

This stops all containers and removes the data volumes.
