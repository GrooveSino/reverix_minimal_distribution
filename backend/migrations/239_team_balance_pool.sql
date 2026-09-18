-- Run with gateway traffic drained during upgrade: billing and usage-log writes
-- in older binaries are separate transactions. Never mix old and new writers.
CREATE TABLE IF NOT EXISTS team_balance_pool (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    total_budget NUMERIC(20,8) NOT NULL DEFAULT 0 CHECK (total_budget >= 0),
    consumed NUMERIC(20,8) NOT NULL DEFAULT 0,
    revision BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Include every retained historical charge, even for deleted users/keys.
-- ON CONFLICT prevents re-importing history on restart or migration replay.
INSERT INTO team_balance_pool (id, consumed)
SELECT 1, COALESCE(SUM(ROUND(actual_cost, 8)), 0) FROM usage_logs
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS team_balance_adjustments (
    id BIGSERIAL PRIMARY KEY,
    admin_id BIGINT NOT NULL,
    operation TEXT NOT NULL CHECK (operation IN ('set', 'add')),
    amount NUMERIC(20,8) NOT NULL,
    total_budget NUMERIC(20,8) NOT NULL,
    revision BIGINT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
