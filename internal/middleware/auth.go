package middleware

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuthContext 认证上下文
type AuthContext struct {
	UserID      uint
	Username    string
	IsSuper     bool
	IsAPIKey    bool
	TenantID    *uint
	Roles       []string
	Permissions map[string]bool
}

// ContextKey 上下文键
const (
	UserContextKey    = "auth_user_id"
	UsernameContextKey = "auth_username"
	IsSuperContextKey  = "auth_is_super"
	IsAPIKeyContextKey = "auth_is_api_key"
	TenantContextKey   = "auth_tenant_id"
	RolesContextKey    = "auth_roles"
	PermsContextKey    = "auth_permissions"
)

// APIKeyAuth API Key 认证中间件（保持向后兼容）
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

		// 设置认证上下文（API Key 模式）
		c.Set(UserContextKey, uint(0))
		c.Set(UsernameContextKey, "api_key_user")
		c.Set(IsAPIKeyContextKey, true)
		c.Set(IsSuperContextKey, true) // API Key 用户默认有超级权限

		c.Next()
	}
}

// UnifiedAuth 统一认证中间件（支持 API Key / Basic / JWT）
func UnifiedAuth(apiKeys []string, db *gorm.DB, jwtSecret string) gin.HandlerFunc {
	keys := make(map[string]bool, len(apiKeys))
	for _, k := range apiKeys {
		keys[k] = true
	}

	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")

		// 1. Bearer Token 认证
		if strings.HasPrefix(auth, "Bearer ") {
			token := strings.TrimPrefix(auth, "Bearer ")
			
			// 1.1 尝试 JWT 认证
			if jwtSecret != "" {
				claims, err := parseJWT(token, jwtSecret)
				if err == nil && claims != nil {
					setJWTAuthContext(c, claims)
					c.Next()
					return
				}
			}

			// 1.2 尝试 API Key 认证
			if keys[token] {
				setAPIKeyAuthContext(c)
				c.Next()
				return
			}

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "无效的 Token",
			})
			return
		}

		// 2. Basic 认证
		if strings.HasPrefix(auth, "Basic ") {
			token := strings.TrimPrefix(auth, "Basic ")
			username, password, ok := parseBasicAuth(token)
			if !ok {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"code":    401,
					"message": "无效的 Basic 认证",
				})
				return
			}

			// 验证用户
			if db != nil {
				user, err := validateUser(db, username, password)
				if err == nil && user != nil {
					setUserAuthContext(c)
					c.Next()
					return
				}
			}

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "用户名或密码错误",
			})
			return
		}

		// 3. X-API-Key Header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" && keys[apiKey] {
			setAPIKeyAuthContext(c)
			c.Next()
			return
		}

		// 4. Query Token（WebSocket）
		token := c.Query("token")
		if token != "" && keys[token] {
			setAPIKeyAuthContext(c)
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权访问，请提供有效的认证信息",
		})
	}
}

// --- 辅助函数 ---

func parseBasicAuth(auth string) (username, password string, ok bool) {
	decoded, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		return "", "", false
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func setAPIKeyAuthContext(c *gin.Context) {
	c.Set(UserContextKey, uint(0))
	c.Set(UsernameContextKey, "api_key_user")
	c.Set(IsAPIKeyContextKey, true)
	c.Set(IsSuperContextKey, true)
}

func setUserAuthContext(c *gin.Context) {
	c.Set(UserContextKey, uint(1))
	c.Set(UsernameContextKey, "user")
	c.Set(IsAPIKeyContextKey, false)
	c.Set(IsSuperContextKey, false)
}

func setJWTAuthContext(c *gin.Context, claims *JWTClaims) {
	c.Set(UserContextKey, claims.UserID)
	c.Set(UsernameContextKey, claims.Username)
	c.Set(IsAPIKeyContextKey, false)
	c.Set(IsSuperContextKey, claims.IsSuper)
	if claims.TenantID > 0 {
		tid := claims.TenantID
		c.Set(TenantContextKey, &tid)
	}
}

// User 用户模型
type User struct {
	ID           uint
	Username     string
	Password     string
	Status      bool
	TenantID     *uint
	IsSuper     bool
}

// validateUser 验证用户（需要数据库支持）
func validateUser(db *gorm.DB, username, password string) (*User, error) {
	var user User
	err := db.Where("username = ? AND status = ?", username, true).First(&user).Error
	if err != nil {
		return nil, err
	}

	// 验证密码
	if !verifyPassword(password, user.Password) {
		return nil, errors.New("password mismatch")
	}

	return &user, nil
}

// verifyPassword 验证密码（简化版）
func verifyPassword(input, hashed string) bool {
	// 简化实现：实际应使用 bcrypt
	return subtle.ConstantTimeCompare([]byte(input), []byte(hashed)) == 1
}

// JWT 相关

// JWTClaims JWT Claims
type JWTClaims struct {
	UserID   uint
	Username string
	TenantID uint
	IsSuper  bool
	Exp      int64
}

// parseJWT 解析 JWT
func parseJWT(token, secret string) (*JWTClaims, error) {
	// 简化实现：实际应使用 jwt-go 库
	// 这里返回 nil 表示暂不支持
	return nil, errors.New("jwt not implemented")
}

// GenerateJWT 生成 JWT（示例）
func GenerateJWT(userID uint, username string, secret string, expDuration time.Duration) (string, error) {
	// 简化实现
	return "", errors.New("jwt not implemented")
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
	User    *User  `json:"user,omitempty"`
}

// GetAuthContext 获取认证上下文
func GetAuthContext(c *gin.Context) *AuthContext {
	userID, _ := c.Get(UserContextKey)
	username, _ := c.Get(UsernameContextKey)
	isSuper, _ := c.Get(IsSuperContextKey)
	isAPIKey, _ := c.Get(IsAPIKeyContextKey)
	tenantID, _ := c.Get(TenantContextKey)

	ctx := &AuthContext{
		IsAPIKey: false,
	}

	if uid, ok := userID.(uint); ok {
		ctx.UserID = uid
	}
	if uname, ok := username.(string); ok {
		ctx.Username = uname
	}
	if super, ok := isSuper.(bool); ok {
		ctx.IsSuper = super
	}
	if apiKey, ok := isAPIKey.(bool); ok {
		ctx.IsAPIKey = apiKey
	}
	if tid, ok := tenantID.(*uint); ok && tid != nil {
		ctx.TenantID = tid
	}

	return ctx
}

// HasPermission 检查权限
func (a *AuthContext) HasPermission(perm string) bool {
	if a.IsSuper || a.IsAPIKey {
		return true
	}
	return a.Permissions[perm]
}

// RequirePermission 权限检查中间件
func RequirePermission(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := GetAuthContext(c)
		if authCtx == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			return
		}

		if !authCtx.HasPermission(perm) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "权限不足"})
			return
		}

		c.Next()
	}
}

// RequireSuperAdmin 超级管理员检查
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := GetAuthContext(c)
		if authCtx == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			return
		}

		if !authCtx.IsSuper {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "需要超级管理员权限"})
			return
		}

		c.Next()
	}
}

// RateLimit 简化版速率限制
func RateLimit(limit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 简化实现：实际应使用令牌桶
		c.Next()
	}
}
