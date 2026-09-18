package service

import (
	"context"
	"math"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrTeamBalanceExhausted = infraerrors.Forbidden("TEAM_BALANCE_EXHAUSTED", "团队公池余额不足，请联系管理员增加额度。")
	ErrTeamBalanceConflict  = infraerrors.Conflict("TEAM_BALANCE_CONFLICT", "公池额度已被修改，请刷新后重试。")
	ErrTeamBalanceInvalid   = infraerrors.BadRequest("TEAM_BALANCE_INVALID", "请输入有效的公池额度和版本。")
)

// Personal balances remain independent. Consumed is a durable all-time counter,
// not a live SUM of usage logs which can be cleaned up by an administrator.
type TeamBalance struct {
	TotalBudget float64 `json:"total_budget"`
	Consumed    float64 `json:"consumed"`
	Remaining   float64 `json:"remaining"`
	Revision    int64   `json:"revision"`
}

type TeamBalanceRepository interface {
	Get(context.Context) (*TeamBalance, error)
	Adjust(ctx context.Context, adminID int64, operation string, amount float64, revision int64) (*TeamBalance, error)
}

type TeamBalanceService struct{ repo TeamBalanceRepository }

func NewTeamBalanceService(repo TeamBalanceRepository) *TeamBalanceService {
	return &TeamBalanceService{repo: repo}
}

func (s *TeamBalanceService) Get(ctx context.Context) (*TeamBalance, error) {
	return s.repo.Get(ctx)
}

func (s *TeamBalanceService) Adjust(ctx context.Context, adminID int64, operation string, amount float64, revision int64) (*TeamBalance, error) {
	if adminID <= 0 || revision < 0 || math.IsNaN(amount) || math.IsInf(amount, 0) || amount < 0 || amount > 1e10 || (operation != "set" && operation != "add") || (operation == "add" && amount <= 0) {
		return nil, ErrTeamBalanceInvalid
	}
	amount = QuantizeUsageBillingAmount(amount)
	if operation == "add" && amount == 0 {
		return nil, ErrTeamBalanceInvalid
	}
	return s.repo.Adjust(ctx, adminID, operation, amount, revision)
}

// Read the database on each admission: no stale per-process/Redis balance cache
// may admit work after exhaustion or delay recovery after an administrator tops up.
func (s *TeamBalanceService) Check(ctx context.Context) error {
	balance, err := s.Get(ctx)
	if err != nil {
		return ErrBillingServiceUnavailable.WithCause(err)
	}
	if balance.Remaining <= 0 {
		return ErrTeamBalanceExhausted
	}
	return nil
}

func (s *APIKeyService) TeamBalanceEnabled() bool { return s.teamBalance != nil }

func (s *APIKeyService) CheckTeamBalance(ctx context.Context) error {
	if s.teamBalance == nil {
		return nil
	} // constructors used in isolated tests
	return s.teamBalance.Check(ctx)
}
