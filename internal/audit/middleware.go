package audit

import (
	"github.com/imysm/db-backup/internal/auth"
	"github.com/imysm/db-backup/internal/multiTenant"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const RequestIDKey = "X-Request-ID"

// AuditMiddleware 审计日志中间件
func AuditMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成请求ID
		requestID := uuid.New().String()
		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)

		// 记录开始时间
		start := time.Now()

		// 处理请求
		c.Next()

		// 请求结束后记录审计日志
		go func() {
			// 只记录写操作
			if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
				return
			}

			// 获取用户信息
			var userID *uint
			var username string
			var tenantID *uint

			if authCtx := auth.GetAuthContext(c); authCtx != nil {
				userID = &authCtx.UserID
				username = authCtx.Username
				tenantID = authCtx.TenantID
			}

			// 获取租户信息
			if tenantID == nil {
				if tc := multiTenant.GetTenantFromContext(c); tc != nil {
					tenantID = &tc.TenantID
				}
			}

			// 获取资源信息
			resource, resourceID := parseResource(c.FullPath(), c.Params)

			// 获取结果
			result := ResultSuccess
			if c.Writer.Status() >= 400 {
				result = ResultFailure
			} else if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
				result = ResultSuccess
			}

			// 计算耗时
			duration := int(time.Since(start).Milliseconds())

			// 记录日志
			auditLog := &AuditLog{
				UserID:       userID,
				Username:     username,
				TenantID:     tenantID,
				Action:       c.Request.Method,
				Resource:     resource,
				ResourceID:   resourceID,
				ResourceName: c.FullPath(),
				Result:       result,
				IPAddress:    c.ClientIP(),
				UserAgent:    c.Request.UserAgent(),
				RequestID:    requestID,
				Duration:     duration,
			}

			if err := db.Create(auditLog).Error; err != nil {
				log.Printf("[AUDIT] Failed to create audit log: %v", err)
			}
		}()
	}
}

// AuditLogMiddleware 手动审计日志中间件（用于需要详细记录的场景）
func AuditLogMiddleware(db *gorm.DB, action, resource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID, _ := c.Get(RequestIDKey)
		reqID, _ := requestID.(string)

		start := time.Now()
		c.Next()

		var userID *uint
		var username string
		var tenantID *uint

		if authCtx := auth.GetAuthContext(c); authCtx != nil {
			userID = &authCtx.UserID
			username = authCtx.Username
			tenantID = authCtx.TenantID
		}

		if tenantID == nil {
			if tc := multiTenant.GetTenantFromContext(c); tc != nil {
				tenantID = &tc.TenantID
			}
		}

		result := ResultSuccess
		if c.Writer.Status() >= 400 {
			result = ResultFailure
		}

		duration := int(time.Since(start).Milliseconds())

		auditLog := &AuditLog{
			UserID:     userID,
			Username:   username,
			TenantID:   tenantID,
			Action:     action,
			Resource:   resource,
			Result:     result,
			IPAddress:  c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			RequestID:  reqID,
			Duration:   duration,
		}

		if err := db.Create(auditLog).Error; err != nil {
			log.Printf("[AUDIT] Failed to create audit log: %v", err)
		}
	}
}

// parseResource 从路径和参数中解析资源和ID
func parseResource(path string, params gin.Params) (resource string, resourceID *uint) {
	// 从路径中提取资源
	// /api/v1/jobs/123 -> job, 123
	resource = "unknown"

	// 移除 /api/v1/ 前缀
	path = removeAPIv1Prefix(path)

	parts := splitPath(path)
	if len(parts) == 0 {
		return
	}

	resource = normalizeResourceName(parts[0])

	// 查找 ID 参数
	for _, part := range parts[1:] {
		if id, err := strconv.ParseUint(part, 10, 32); err == nil {
			rid := uint(id)
			resourceID = &rid
			break
		}
	}

	return
}

func removeAPIv1Prefix(path string) string {
	prefix := "/api/v1/"
	if len(path) > len(prefix) && path[:len(prefix)] == prefix {
		return path[len(prefix):]
	}
	return path
}

func splitPath(path string) []string {
	if path == "" || path == "/" {
		return nil
	}

	// 移除首尾斜杠
	path = trimSlashes(path)

	parts := make([]string, 0)
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if start < i {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}

func trimSlashes(s string) string {
	for len(s) > 0 && s[0] == '/' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func normalizeResourceName(name string) string {
	// 资源名称映射
	resourceMap := map[string]string{
		"jobs":      "job",
		"records":  "record",
		"templates": "template",
		"verify":   "verify",
		"restore":  "restore",
		"merge":    "merge",
		"stats":    "stats",
		"users":    "user",
		"roles":    "role",
		"tenants":   "tenant",
		"settings":  "setting",
	}

	if normalized, ok := resourceMap[name]; ok {
		return normalized
	}
	return name
}
