package middleware

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PersonalUserCutGuard hides SaaS self-service endpoints from non-admin users.
// Admins keep the original console. Must run after JWT auth.
func PersonalUserCutGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := GetUserRoleFromContext(c)
		if role == service.RoleAdmin {
			c.Next()
			return
		}
		if personalUserFeatureBlocked(c.Request.URL.Path, c.Request.Method) {
			response.Forbidden(c, "This feature is disabled for personal users")
			c.Abort()
			return
		}
		c.Next()
	}
}

func personalUserFeatureBlocked(path, method string) bool {
	path = strings.ToLower(strings.TrimSpace(path))
	method = strings.ToUpper(strings.TrimSpace(method))

	blockedPrefixes := []string{
		"/api/v1/user/aff",
		"/api/v1/user/account-bindings",
		"/api/v1/user/auth-identities",
		"/api/v1/user/totp",
		"/api/v1/user/passkeys",
		"/api/v1/user/notify-email",
		"/api/v1/redeem",
		"/api/v1/subscriptions",
		"/api/v1/channels/available",
		"/api/v1/payment",
	}
	for _, prefix := range blockedPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}

	if method == "PUT" && (path == "/api/v1/user" || path == "/api/v1/user/") {
		return true
	}
	return false
}
