package auth

import (
	"github.com/gin-gonic/gin"
)

// AuthContext 认证上下文
type AuthContext struct {
	UserID      uint
	Username    string
	IsSuper     bool
	TenantID    *uint
	Roles       []string
	Permissions map[string]bool
}

// GetAuthContext 获取认证上下文
func GetAuthContext(c *gin.Context) *AuthContext {
	// 从 gin context 获取用户信息
	// 这个简化版本从 header 或 context 中获取
	if userID, exists := c.Get("userID"); exists {
		return &AuthContext{
			UserID:   userID.(uint),
			Username: "admin",
			IsSuper:  true,
		}
	}
	return nil
}

// HasPermission 检查权限
func (a *AuthContext) HasPermission(perm string) bool {
	if a.IsSuper {
		return true
	}
	return a.Permissions[perm]
}
