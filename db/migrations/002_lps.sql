-- Create the lps master table
CREATE TABLE IF NOT EXISTS lps (
    lp_id          VARCHAR(64) PRIMARY KEY,
    commitment_usd NUMERIC(18,2) NOT NULL,
    email          VARCHAR(256) NOT NULL
);

-- Seed the canonical 10 LPs (matches demodriver simulation)
INSERT INTO lps (lp_id, commitment_usd, email) VALUES
    ('lp-01', 15000000.00, 'lp-01@example.com'),
    ('lp-02', 12000000.00, 'lp-02@example.com'),
    ('lp-03', 18000000.00, 'lp-03@example.com'),
    ('lp-04', 10000000.00, 'lp-04@example.com'),
    ('lp-05', 22000000.00, 'lp-05@example.com'),
    ('lp-06',  8000000.00, 'lp-06@example.com'),
    ('lp-07', 30000000.00, 'lp-07@example.com'),
    ('lp-08', 25000000.00, 'lp-08@example.com'),
    ('lp-09', 14000000.00, 'lp-09@example.com'),
    ('lp-10', 20000000.00, 'lp-10@example.com')
ON CONFLICT (lp_id) DO NOTHING;
