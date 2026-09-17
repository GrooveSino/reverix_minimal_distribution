package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type quotaAdminStub struct {
	service.AdminService
	users      []service.User
	updated    *service.User
	getByID    *service.User
	updateErr  error
	updateArgs struct {
		id        int64
		balance   float64
		operation string
	}
}

func (s *quotaAdminStub) ListUsers(context.Context, int, int, service.UserListFilters, string, string) ([]service.User, int64, error) {
	return s.users, int64(len(s.users)), nil
}

func (s *quotaAdminStub) GetUser(context.Context, int64) (*service.User, error) {
	return s.getByID, nil
}

func (s *quotaAdminStub) UpdateUserBalance(_ context.Context, userID int64, balance float64, operation string, _ string) (*service.User, error) {
	s.updateArgs.id = userID
	s.updateArgs.balance = balance
	s.updateArgs.operation = operation
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	return s.updated, nil
}

func TestQuotaHandlerRejectsNonUserTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &quotaAdminStub{getByID: &service.User{ID: 1, Role: service.RoleAdmin}}
	h := NewQuotaHandler(stub)
	r := gin.New()
	r.POST("/quota/users/:id/balance", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7})
		c.Set(string(middleware.ContextKeyUserRole), service.RoleQuota)
		h.UpdateBalance(c)
	})
	body, _ := json.Marshal(map[string]any{"balance": 10, "operation": "add"})
	req := httptest.NewRequest(http.MethodPost, "/quota/users/1/balance", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestQuotaHandlerRejectsSelf(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &quotaAdminStub{getByID: &service.User{ID: 7, Role: service.RoleUser}}
	h := NewQuotaHandler(stub)
	r := gin.New()
	r.POST("/quota/users/:id/balance", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7})
		c.Set(string(middleware.ContextKeyUserRole), service.RoleQuota)
		h.UpdateBalance(c)
	})
	body, _ := json.Marshal(map[string]any{"balance": 10, "operation": "add"})
	req := httptest.NewRequest(http.MethodPost, "/quota/users/7/balance", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}
