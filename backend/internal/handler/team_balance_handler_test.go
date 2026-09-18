package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type teamBalanceHandlerRepo struct{ adjustments int }

func (*teamBalanceHandlerRepo) Get(context.Context) (*service.TeamBalance, error) {
	return &service.TeamBalance{TotalBudget: 10000, Consumed: 1000, Remaining: 9000}, nil
}
func (r *teamBalanceHandlerRepo) Adjust(context.Context, int64, string, float64, int64) (*service.TeamBalance, error) {
	r.adjustments++
	return r.Get(context.Background())
}

func TestTeamBalanceHandlerRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, role := range []string{service.RoleUser, "quota", service.RoleAdmin} {
		t.Run(role, func(t *testing.T) {
			repo := &teamBalanceHandlerRepo{}
			h := NewTeamBalanceHandler(service.NewTeamBalanceService(repo))
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(string(middleware.ContextKeyUserRole), role)
				c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})
			})
			router.GET("/balance", h.Get)
			router.POST("/balance", h.Adjust)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/balance", nil))
			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Body.String(), `"remaining":9000`)
			require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			w = httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/balance", strings.NewReader(`{"operation":"set","amount":0,"revision":0}`))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			if role == service.RoleAdmin {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, 1, repo.adjustments)
			} else {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, repo.adjustments)
			}
		})
	}
}
