package multiTenant

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// TenantContextKey 租户上下文 key
const TenantContextKey = "tenant_id"

// TenantContext 租户上下文
type TenantContext struct {
	TenantID uint
}

// GetTenantFromContext 从上下文获取租户上下文
func GetTenantFromContext(c *gin.Context) *TenantContext {
	if tenantID, exists := c.Get(TenantContextKey); exists {
		return &TenantContext{TenantID: tenantID.(uint)}
	}
	return nil
}

// RequireTenant 租户检查中间件
func RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantIDStr := c.GetHeader("X-Tenant-ID")
		if tenantIDStr == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "需要租户ID"})
			return
		}

		tenantID, err := strconv.ParseUint(tenantIDStr, 10, 32)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "无效的租户ID"})
			return
		}

		c.Set(TenantContextKey, uint(tenantID))
		c.Next()
	}
}
