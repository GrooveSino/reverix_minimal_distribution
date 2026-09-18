package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type teamBalanceRepository struct{ db *sql.DB }

func NewTeamBalanceRepository(db *sql.DB) service.TeamBalanceRepository {
	return &teamBalanceRepository{db: db}
}

func (r *teamBalanceRepository) Get(ctx context.Context) (*service.TeamBalance, error) {
	var b service.TeamBalance
	err := r.db.QueryRowContext(ctx, `SELECT total_budget, consumed, total_budget - consumed, revision FROM team_balance_pool WHERE id = 1`).Scan(&b.TotalBudget, &b.Consumed, &b.Remaining, &b.Revision)
	return &b, err
}

func (r *teamBalanceRepository) Adjust(ctx context.Context, adminID int64, operation string, amount float64, revision int64) (*service.TeamBalance, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var b service.TeamBalance
	// Revision changes only on administrative edits. Concurrent consumption never
	// gets overwritten; duplicate submissions with the same revision cannot add twice.
	err = tx.QueryRowContext(ctx, `UPDATE team_balance_pool
        SET total_budget = CASE WHEN $1 = 'add' THEN total_budget + $2 ELSE $2 END,
            revision = revision + 1, updated_at = NOW()
        WHERE id = 1 AND revision = $3
        RETURNING total_budget, consumed, total_budget - consumed, revision`, operation, amount, revision).Scan(&b.TotalBudget, &b.Consumed, &b.Remaining, &b.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrTeamBalanceConflict
	}
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO team_balance_adjustments (admin_id, operation, amount, total_budget, revision) VALUES ($1, $2, $3, $4, $5)`, adminID, operation, amount, b.TotalBudget, b.Revision)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &b, nil
}

// Called inside the personal billing transaction AFTER claiming the request's
// idempotency key. Allow overdraft for already-admitted requests; never clamp it.
func consumeTeamBalance(ctx context.Context, tx *sql.Tx, amount float64) error {
	if amount <= 0 {
		return nil
	}
	result, err := tx.ExecContext(ctx, `UPDATE team_balance_pool SET consumed = consumed + $1, updated_at = NOW() WHERE id = 1`, amount)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("team balance pool is not initialized")
	}
	return nil
}
