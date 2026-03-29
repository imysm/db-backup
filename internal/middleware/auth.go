// Package middleware 提供 HTTP 中间件
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth API Key 认证中间件
func APIKeyAuth(apiKeys []string) gin.HandlerFunc {
	keys := make(map[string]bool, len(apiKeys))
	for _, k := range apiKeys {
		keys[k] = true
	}

	return func(c *gin.Context) {
		var token string

		// 优先从 Authorization header 获取
		auth := c.GetHeader("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}

		// 其次从 X-API-Key header 获取
		if token == "" {
			token = c.GetHeader("X-API-Key")
		}

		// 最后从 query param 获取（WebSocket 使用）
		if token == "" {
			token = c.Query("token")
		}

		if token == "" || !keys[token] {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "未授权访问，请提供有效的 API Key",
			})
			return
		}

		c.Next()
	}
}
