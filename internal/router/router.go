package router

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/imysm/db-backup/internal/api/handler"
	"github.com/imysm/db-backup/internal/config"
	"github.com/imysm/db-backup/internal/crypto"
	"github.com/imysm/db-backup/internal/middleware"
	"github.com/imysm/db-backup/internal/ws"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源（生产环境应限制）
	},
}

// Setup 设置路由
func Setup(cfg *config.Config, db *gorm.DB, staticPath string) *gin.Engine {
	// 初始化加密器
	encryptor, err := crypto.NewEncryptor(cfg.Global.EncryptionKey != "", cfg.Global.EncryptionKey)
	if err != nil {
		panic("初始化加密器失败: " + err.Error())
	}

	r := gin.Default()

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "version": "1.0.0"})
	})

	// Prometheus 指标端点（P0功能）
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 静态文件服务
	if staticPath != "" {
		absStaticPath, _ := filepath.Abs(staticPath)
		r.NoRoute(func(c *gin.Context) {
			reqPath := filepath.Clean(filepath.Join(absStaticPath, c.Request.URL.Path))
			if !strings.HasPrefix(reqPath, absStaticPath) {
				c.JSON(403, gin.H{"error": "forbidden"})
				return
			}
			if _, err := http.Dir(absStaticPath).Open(c.Request.URL.Path); err != nil {
				// 返回 index.html
				c.File(filepath.Join(absStaticPath, "index.html"))
				return
			}
			c.File(reqPath)
		})
	}

	// API 认证中间件
	authMiddleware := middleware.APIKeyAuth(cfg.Global.APIKeys)

	// API 路由（需认证）
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
			jobs.POST("/:id/run", jobHandler.Run)
		}

		// 记录管理
		records := v1.Group("/records")
		{
			recordHandler := handler.NewRecordHandler(db)
			records.GET("", recordHandler.List)
			records.GET("/:id", recordHandler.Get)
			records.DELETE("/:id", recordHandler.Delete)
		}

		// 验证管理
		verify := v1.Group("/verify")
		{
			verifyHandler := handler.NewVerifyHandler(db)
			verify.POST("/:id", verifyHandler.VerifyBackup)
			verify.POST("/:id/restore", verifyHandler.TestRestore)
		}

		// 恢复管理
		restore := v1.Group("/restore")
		{
			restoreHandler := handler.NewRestoreHandler(db)
			restore.POST("", restoreHandler.Restore)
			restore.GET("/list", restoreHandler.List)
			restore.GET("/validate/:id", restoreHandler.Validate)
		}

		// 统计
		v1.GET("/stats", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"total_jobs":    0,
				"total_records": 0,
				"success_rate":  0,
			})
		})
	}

	return r
}

// SetupWithHub 设置路由（带 WebSocket 支持）
func SetupWithHub(cfg *config.Config, db *gorm.DB, staticPath string, hub *ws.Hub) *gin.Engine {
	r := Setup(cfg, db, staticPath)

	// WebSocket 日志端点（P0功能，通过 query param token 认证）
	wsAuth := middleware.APIKeyAuth(cfg.Global.APIKeys)
	r.GET("/ws/logs/:job_id", wsAuth, func(c *gin.Context) {
		jobID := c.Param("job_id")
		serveWebSocket(c, hub, jobID)
	})

	return r
}

// serveWebSocket 处理 WebSocket 连接
func serveWebSocket(c *gin.Context, hub *ws.Hub, taskID string) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := ws.NewClient(conn, hub, taskID)
	hub.Register(client)

	// 启动读写协程
	go client.WritePump()
	go client.ReadPump()
}
