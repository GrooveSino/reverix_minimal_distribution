package service

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type teamBalanceStub struct {
	balance     TeamBalance
	err         error
	adjustments int
}

func (r *teamBalanceStub) Get(context.Context) (*TeamBalance, error) { return &r.balance, r.err }
func (r *teamBalanceStub) Adjust(_ context.Context, _ int64, _ string, _ float64, _ int64) (*TeamBalance, error) {
	r.adjustments++
	return &r.balance, r.err
}

func TestTeamBalanceAdmissionAndRecovery(t *testing.T) {
	repo := &teamBalanceStub{}
	svc := NewTeamBalanceService(repo)
	for _, remaining := range []float64{0, -0.001, -100} {
		repo.balance.Remaining = remaining
		require.ErrorIs(t, svc.Check(context.Background()), ErrTeamBalanceExhausted)
	}
	repo.balance.Remaining = 0.00000001
	require.NoError(t, svc.Check(context.Background()))
	repo.err = errors.New("database unavailable")
	require.ErrorIs(t, svc.Check(context.Background()), ErrBillingServiceUnavailable)
}

func TestTeamBalanceRunsBeforePersonalOrSubscriptionChecks(t *testing.T) {
	pool := NewTeamBalanceService(&teamBalanceStub{})
	for _, mode := range []string{config.RunModeSimple, config.RunModeStandard} {
		svc := &BillingCacheService{cfg: &config.Config{RunMode: mode}, teamBalance: pool}
		require.ErrorIs(t, svc.CheckBillingEligibility(context.Background(), &User{ID: 1, Balance: 100}, nil, nil, nil, ""), ErrTeamBalanceExhausted)
	}
}

func TestTeamBalanceRejectsInvalidAdjustments(t *testing.T) {
	repo := &teamBalanceStub{}
	svc := NewTeamBalanceService(repo)
	for _, amount := range []float64{-1, math.NaN(), math.Inf(1), 1e11} {
		_, err := svc.Adjust(context.Background(), 1, "set", amount, 0)
		require.ErrorIs(t, err, ErrTeamBalanceInvalid)
	}
	for _, op := range []string{"subtract", "", "ADD"} {
		_, err := svc.Adjust(context.Background(), 1, op, 10, 0)
		require.ErrorIs(t, err, ErrTeamBalanceInvalid)
	}
	_, err := svc.Adjust(context.Background(), 1, "add", 0.000000001, 0)
	require.ErrorIs(t, err, ErrTeamBalanceInvalid)
	require.Zero(t, repo.adjustments)
	_, err = svc.Adjust(context.Background(), 1, "set", 0, 0)
	require.NoError(t, err)
	require.Equal(t, 1, repo.adjustments)
}

type teamBillingCommandRecorder struct {
	UsageBillingRepository
	command *UsageBillingCommand
}

func (r *teamBillingCommandRecorder) Apply(_ context.Context, cmd *UsageBillingCommand) (*UsageBillingApplyResult, error) {
	r.command = cmd
	return nil, errors.New("stop after capturing command")
}

func TestTeamBalanceSubscriptionStillDebitsPersonalWallet(t *testing.T) {
	p := &postUsageBillingParams{
		Cost: &CostBreakdown{TotalCost: 100, ActualCost: 50},
		User: &User{ID: 1}, APIKey: &APIKey{ID: 2}, Account: &Account{ID: 3},
		Subscription: &UserSubscription{ID: 4}, IsSubscriptionBill: true,
	}
	repo := &teamBillingCommandRecorder{}
	deps := &billingDeps{billingCacheService: &BillingCacheService{teamBalance: NewTeamBalanceService(&teamBalanceStub{})}}
	_, err := applyUsageBilling(context.Background(), "subscription", nil, p, deps, repo)
	require.Error(t, err)
	require.NotNil(t, repo.command)
	require.Equal(t, 50.0, repo.command.BalanceCost)
	require.Equal(t, 50.0, repo.command.SubscriptionCost)
	require.NotEmpty(t, repo.command.RequestFingerprint)

	_, err = applyUsageBilling(context.Background(), "subscription", nil, p, deps, nil)
	require.ErrorIs(t, err, ErrBillingServiceUnavailable, "must not fall back to personal-only billing")
}
