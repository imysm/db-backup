package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/imysm/db-backup/internal/api/handler"
	"github.com/imysm/db-backup/internal/config"
	"github.com/imysm/db-backup/internal/crypto"
	"github.com/imysm/db-backup/internal/middleware"
	"gorm.io/gorm"
)

// Setup 设置路由
func Setup(cfg *config.Config, db *gorm.DB, staticPath string) *gin.Engine {
	encryptor, _ := crypto.NewEncryptor(cfg.Global.EncryptionKey != "", cfg.Global.EncryptionKey)
	r := gin.Default()

	// 全局速率限制中间件（100 req/s per IP）
	r.Use(RateLimitMiddleware(100, 100, 10000))

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "version": "1.0.0"})
	})

	// 静态文件服务
	if staticPath != "" {
		absStaticPath, _ := filepath.Abs(staticPath)
		// Ensure trailing slash for consistent prefix matching
		if !strings.HasSuffix(absStaticPath, "/") {
			absStaticPath += "/"
		}
		r.NoRoute(safeStaticHandler(absStaticPath))
	}

	// API 路由（需认证）
	authMiddleware := middleware.APIKeyAuth(cfg.Global.APIKeys)
	v1 := r.Group("/api/v1", authMiddleware)
	{
		// 任务管理
		jobs := v1.Group("/jobs")
		{
			jobHandler := handler.NewJobHandler(db, encryptor)
			jobs.GET("", jobHandler.List)
			jobs.POST("", jobHandler.Create)
			jobs.GET("/:id", jobHandler.Get)
			jobs.PUT("/:id", jobHandler.Update)
			jobs.DELETE("/:id", jobHandler.Delete)
			// 高成本操作：更严格的速率限制（10 req/s per IP）
			jobs.POST("/:id/run", RateLimitMiddleware(10, 10, 10000), jobHandler.Run)
			jobs.POST("/:id/test-connection", jobHandler.TestConnection)
			jobs.GET("/:id/next-runs", jobHandler.NextRuns)
		}

		// 健康报告
		healthHandler := handler.NewHealthCheckHandler(db)
		v1.GET("/health/report", healthHandler.GetHealthReport)

		// 存储管理
		storageHandler := handler.NewStorageHandler(db)
		v1.GET("/storage/stats", storageHandler.GetStats)
		v1.GET("/storage/objects", storageHandler.ListObjects)

		// 告警管理
		alertHandler := handler.NewAlertHandler(db)
		v1.GET("/alert/channels", alertHandler.ListChannels)
		v1.POST("/alert/channels", alertHandler.CreateChannel)
		v1.PUT("/alert/channels/:id", alertHandler.UpdateChannel)
		v1.DELETE("/alert/channels/:id", alertHandler.DeleteChannel)
		v1.GET("/alert/rules", alertHandler.ListRules)
		v1.POST("/alert/rules", alertHandler.CreateRule)
		v1.PUT("/alert/rules/:id", alertHandler.UpdateRule)
		v1.DELETE("/alert/rules/:id", alertHandler.DeleteRule)

		// 记录管理
		records := v1.Group("/records")
		{
			recordHandler := handler.NewRecordHandler(db)
			records.GET("", recordHandler.List)
			records.GET("/:id", recordHandler.Get)
			records.GET("/:id/download", recordHandler.Download)
			records.DELETE("/:id", recordHandler.Delete)
		}

		// 验证管理
		verify := v1.Group("/verify")
		{
			verifyHandler := handler.NewVerifyHandler(db)
			verify.POST("/:id", verifyHandler.VerifyBackup)
			verify.POST("/:id/restore", verifyHandler.TestRestore)
			verify.POST("/batch", verifyHandler.BatchVerify)
		}

		// 恢复管理
		restore := v1.Group("/restore")
		{
			restoreHandler := handler.NewRestoreHandler(db)
			// 高成本操作：更严格的速率限制（10 req/s per IP）
			restore.POST("", RateLimitMiddleware(10, 10, 10000), restoreHandler.Restore)
			restore.GET("/list", restoreHandler.List)
			restore.GET("/validate/:id", restoreHandler.Validate)
			restore.POST("/validate/:id", restoreHandler.ValidatePOST)
			restore.GET("/:id", restoreHandler.GetDetail)
		}

		// 统计
		statsHandler := handler.NewStatsHandler(db)
		v1.GET("/stats", statsHandler.GetStats)
	}

	return r
}

// safeStaticHandler returns a NoRoute handler with hardened path traversal protection.
func safeStaticHandler(absStaticPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqPath := c.Request.URL.Path

		// Reject paths with null bytes
		if strings.Contains(reqPath, "\x00") {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		// Reject hidden files (segments starting with .)
		segments := strings.Split(reqPath, "/")
		for _, seg := range segments {
			if strings.HasPrefix(seg, ".") {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
		}

		// Clean and join with static path
		cleaned := filepath.Clean("/" + reqPath)
		fullPath := filepath.Join(absStaticPath, cleaned)

		// Ensure the resolved path is under the static directory
		// Use filepath.Rel to detect escape attempts reliably
		rel, err := filepath.Rel(absStaticPath, fullPath)
		if err != nil || strings.HasPrefix(rel, "..") {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		// Check that the fullPath is not a symlink pointing outside
		fi, err := os.Lstat(fullPath)
		if err == nil && fi.Mode()&os.ModeSymlink != 0 {
			real, err := filepath.EvalSymlinks(fullPath)
			if err != nil || !strings.HasPrefix(real, absStaticPath) {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
		}

		// Serve the file or fallback to index.html
		if _, err := http.Dir(strings.TrimRight(absStaticPath, "/")).Open(cleaned); err != nil {
			c.File(filepath.Join(absStaticPath, "index.html"))
			return
		}
		c.File(fullPath)
	}
}
