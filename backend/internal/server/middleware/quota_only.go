package middleware

import (
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// QuotaOnly allows quota operators (and full admins) to manage regular-user balances.
func QuotaOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := GetUserRoleFromContext(c)
		if !ok {
			AbortWithError(c, 401, "UNAUTHORIZED", "User not found in context")
			return
		}
		if role != service.RoleQuota && role != service.RoleAdmin {
			AbortWithError(c, 403, "FORBIDDEN", "Quota admin access required")
			return
		}
		c.Next()
	}
}
