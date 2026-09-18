package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type TeamBalanceHandler struct{ service *service.TeamBalanceService }

func NewTeamBalanceHandler(s *service.TeamBalanceService) *TeamBalanceHandler {
	return &TeamBalanceHandler{service: s}
}

func (h *TeamBalanceHandler) Get(c *gin.Context) {
	b, err := h.service.Get(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, b)
}

func (h *TeamBalanceHandler) Adjust(c *gin.Context) {
	// Defense in depth: quota administrators must not change the team budget.
	role, _ := middleware.GetUserRoleFromContext(c)
	if role != service.RoleAdmin {
		response.Forbidden(c, "仅管理员可以修改团队公池额度")
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req struct {
		Operation string   `json:"operation" binding:"required,oneof=set add"`
		Amount    *float64 `json:"amount" binding:"required"`
		Revision  *int64   `json:"revision" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请输入有效的操作、金额和版本")
		return
	}
	b, err := h.service.Adjust(c.Request.Context(), subject.UserID, req.Operation, *req.Amount, *req.Revision)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, b)
}
