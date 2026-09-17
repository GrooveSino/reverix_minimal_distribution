package handler

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type QuotaHandler struct {
	adminService service.AdminService
}

func NewQuotaHandler(adminService service.AdminService) *QuotaHandler {
	return &QuotaHandler{adminService: adminService}
}

type quotaUserDTO struct {
	ID       int64   `json:"id"`
	Email    string  `json:"email"`
	Username string  `json:"username"`
	Balance  float64 `json:"balance"`
	Status   string  `json:"status"`
}

type quotaBalanceRequest struct {
	Balance   float64 `json:"balance" binding:"required,gt=0"`
	Operation string  `json:"operation" binding:"required,oneof=set add subtract"`
	Notes     string  `json:"notes"`
}

func (h *QuotaHandler) ListUsers(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	if runes := []rune(search); len(runes) > 100 {
		search = string(runes[:100])
	}
	filters := service.UserListFilters{
		Role:   service.RoleUser,
		Status: "active",
		Search: search,
	}
	includeSubs := false
	filters.IncludeSubscriptions = &includeSubs
	users, total, err := h.adminService.ListUsers(c.Request.Context(), page, pageSize, filters, "created_at", "desc")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]quotaUserDTO, 0, len(users))
	for i := range users {
		u := users[i]
		out = append(out, quotaUserDTO{
			ID:       u.ID,
			Email:    u.Email,
			Username: u.Username,
			Balance:  u.Balance,
			Status:   u.Status,
		})
	}
	response.Paginated(c, out, total, page, pageSize)
}

func (h *QuotaHandler) UpdateBalance(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	var req quotaBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if subject.UserID == userID {
		response.Forbidden(c, "Cannot change your own balance")
		return
	}
	target, err := h.adminService.GetUser(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if target.Role != service.RoleUser {
		response.Forbidden(c, "Can only change regular user balances")
		return
	}
	user, err := h.adminService.UpdateUserBalance(c.Request.Context(), userID, req.Balance, req.Operation, req.Notes)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, quotaUserDTO{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
		Balance:  user.Balance,
		Status:   user.Status,
	})
}
