CREATE TABLE IF NOT EXISTS capital_calls (
    call_id     VARCHAR(64) PRIMARY KEY,
    fund_id     VARCHAR(64) NOT NULL,
    target_amount_usd NUMERIC(18,2) NOT NULL,
    status      VARCHAR(32) NOT NULL DEFAULT 'issued',
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS capital_call_lps (
    call_id         VARCHAR(64) NOT NULL REFERENCES capital_calls(call_id),
    lp_id           VARCHAR(64) NOT NULL,
    commitment_usd  NUMERIC(18,2) NOT NULL,
    draw_amount_usd NUMERIC(18,2),
    status          VARCHAR(32) NOT NULL DEFAULT 'pending',
    PRIMARY KEY (call_id, lp_id)
);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id              SERIAL PRIMARY KEY,
    call_id         VARCHAR(64) NOT NULL,
    lp_id           VARCHAR(64),
    debit_account   VARCHAR(128),
    credit_account  VARCHAR(128),
    amount_usd      NUMERIC(18,2) NOT NULL,
    entry_type      VARCHAR(64),
    created_at      TIMESTAMP NOT NULL DEFAULT NOW()
);
