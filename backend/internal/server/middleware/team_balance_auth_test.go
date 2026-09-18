//go:build unit

package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type authTeamPool struct {
	remaining float64
	err       error
}

func (p *authTeamPool) Get(context.Context) (*service.TeamBalance, error) {
	return &service.TeamBalance{Remaining: p.remaining}, p.err
}
func (*authTeamPool) Adjust(context.Context, int64, string, float64, int64) (*service.TeamBalance, error) {
	panic("unexpected mutation")
}

func TestTeamBalanceAuthProtocols(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, google := range []bool{false, true} {
		t.Run(map[bool]string{false: "openai-anthropic", true: "google"}[google], func(t *testing.T) {
			pool := &authTeamPool{}
			cfg := &config.Config{RunMode: config.RunModeStandard}
			group := &service.Group{ID: 1, Status: service.StatusActive, Hydrated: true}
			key := &service.APIKey{ID: 1, UserID: 1, Key: "test-key", Status: service.StatusActive, GroupID: &group.ID, Group: group, User: &service.User{ID: 1, Role: service.RoleUser, Status: service.StatusActive, Balance: 100}}
			keys := &stubApiKeyRepo{getByKey: func(context.Context, string) (*service.APIKey, error) { return key, nil }}
			billing := service.ProvideBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil, service.NewTeamBalanceService(pool))
			t.Cleanup(billing.Stop)
			api := service.ProvideAPIKeyService(keys, nil, nil, nil, nil, nil, cfg, billing, nil)
			router := gin.New()
			if google {
				router.Use(APIKeyAuthWithSubscriptionGoogle(api, nil, cfg))
			} else {
				router.Use(gin.HandlerFunc(NewAPIKeyAuthMiddleware(api, nil, cfg)))
			}
			router.POST("/invoke", func(c *gin.Context) { c.Status(200) })
			invoke := func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("POST", "/invoke", nil)
				req.Header.Set("Authorization", "Bearer test-key")
				router.ServeHTTP(w, req)
				return w
			}
			require.Equal(t, http.StatusForbidden, invoke().Code)
			pool.remaining = 100
			require.Equal(t, http.StatusOK, invoke().Code)
			pool.remaining = -1
			require.Equal(t, http.StatusForbidden, invoke().Code)
			pool.err = errors.New("private database details")
			w := invoke()
			require.Equal(t, http.StatusServiceUnavailable, w.Code)
			require.NotContains(t, w.Body.String(), "private database details")
		})
	}
}
