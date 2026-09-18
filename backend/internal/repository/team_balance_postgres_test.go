//go:build !integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/migrations"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// Uses a dedicated temporary schema and real PostgreSQL transactions. Run with
// TEAM_BALANCE_TEST_DSN=postgres://... go test ./internal/repository -run TestTeamBalancePostgres
func TestTeamBalancePostgres(t *testing.T) {
	dsn := os.Getenv("TEAM_BALANCE_TEST_DSN")
	if dsn == "" {
		t.Skip("set TEAM_BALANCE_TEST_DSN to run PostgreSQL transaction tests")
	}
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	schema := fmt.Sprintf("team_balance_test_%d", time.Now().UnixNano())
	_, err = db.ExecContext(ctx, "CREATE SCHEMA "+schema)
	require.NoError(t, err)
	defer func() { _, _ = db.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE") }()
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	isolated, err := sql.Open("postgres", u.String())
	require.NoError(t, err)
	defer func() { _ = isolated.Close() }()
	_, err = isolated.ExecContext(ctx, `
        CREATE TABLE usage_logs (actual_cost NUMERIC(20,10));
        INSERT INTO usage_logs VALUES (1000);
        CREATE TABLE users (id BIGINT PRIMARY KEY, balance NUMERIC(20,8), frozen_balance NUMERIC(20,8) DEFAULT 0, updated_at TIMESTAMPTZ, deleted_at TIMESTAMPTZ);
        INSERT INTO users (id, balance) SELECT id, 1000 FROM generate_series(1,10) AS id;
        CREATE TABLE usage_billing_dedup (id BIGSERIAL, request_id TEXT, api_key_id BIGINT, request_fingerprint TEXT, UNIQUE(request_id, api_key_id));
        CREATE TABLE usage_billing_dedup_archive (request_id TEXT, api_key_id BIGINT, request_fingerprint TEXT);
    `)
	require.NoError(t, err)
	migration, err := migrations.FS.ReadFile("239_team_balance_pool.sql")
	require.NoError(t, err)
	_, err = isolated.ExecContext(ctx, string(migration))
	require.NoError(t, err)
	pool := NewTeamBalanceRepository(isolated)
	gate := service.NewTeamBalanceService(pool)
	b, err := pool.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, 1000.0, b.Consumed)
	require.ErrorIs(t, gate.Check(ctx), service.ErrTeamBalanceExhausted)
	b, err = gate.Adjust(ctx, 99, "set", 10000, 0)
	require.NoError(t, err)
	require.Equal(t, 9000.0, b.Remaining)

	billing := NewUsageBillingRepository(nil, isolated)
	// Ten users, each charged $100. Concurrent retries must charge exactly once.
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for userID := int64(1); userID <= 10; userID++ {
		for retry := 0; retry < 2; retry++ {
			wg.Add(1)
			go func(id int64) {
				defer wg.Done()
				_, e := billing.Apply(ctx, &service.UsageBillingCommand{RequestID: fmt.Sprintf("req-%d", id), APIKeyID: id, UserID: id, BalanceCost: 100})
				errs <- e
			}(userID)
		}
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		require.NoError(t, e)
	}
	b, err = pool.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, 8000.0, b.Remaining)
	var count int
	require.NoError(t, isolated.QueryRow(`SELECT COUNT(*) FROM users WHERE balance = 900`).Scan(&count))
	require.Equal(t, 10, count)

	// Failure of either debit rolls back both the personal debit and dedup key.
	_, err = isolated.Exec(`ALTER TABLE team_balance_pool ADD CONSTRAINT test_fail CHECK (consumed <= 2000)`)
	require.NoError(t, err)
	cmd := &service.UsageBillingCommand{RequestID: "failure", APIKeyID: 1, UserID: 1, BalanceCost: 100}
	_, err = billing.Apply(ctx, cmd)
	require.Error(t, err)
	var personal float64
	require.NoError(t, isolated.QueryRow(`SELECT balance FROM users WHERE id = 1`).Scan(&personal))
	require.Equal(t, 900.0, personal)
	_, err = isolated.Exec(`ALTER TABLE team_balance_pool DROP CONSTRAINT test_fail`)
	require.NoError(t, err)
	result, err := billing.Apply(ctx, cmd)
	require.NoError(t, err)
	require.True(t, result.Applied)

	// Exhaustion, then an already-admitted request settles into overdraft.
	b, err = gate.Adjust(ctx, 99, "set", 2100, b.Revision)
	require.NoError(t, err)
	require.Zero(t, b.Remaining)
	require.ErrorIs(t, gate.Check(ctx), service.ErrTeamBalanceExhausted)
	_, err = billing.Apply(ctx, &service.UsageBillingCommand{RequestID: "inflight", APIKeyID: 2, UserID: 2, BalanceCost: 100})
	require.NoError(t, err)
	b, err = pool.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, -100.0, b.Remaining)
	oldRevision := b.Revision
	b, err = gate.Adjust(ctx, 99, "add", 500, oldRevision)
	require.NoError(t, err)
	require.Equal(t, 400.0, b.Remaining)
	require.NoError(t, gate.Check(ctx))
	_, err = gate.Adjust(ctx, 99, "add", 500, oldRevision)
	require.ErrorIs(t, err, service.ErrTeamBalanceConflict)

	// Cleanup/replaying the migration must never restore spent money.
	_, err = isolated.Exec(`DELETE FROM usage_logs`)
	require.NoError(t, err)
	_, err = isolated.Exec(string(migration))
	require.NoError(t, err)
	after, err := pool.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, b, after)
}
