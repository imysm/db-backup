package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/imysm/db-backup/internal/api/router"
	"github.com/imysm/db-backup/internal/config"
	"github.com/imysm/db-backup/internal/logger"
	"github.com/imysm/db-backup/internal/scheduler"
	"gorm.io/gorm"
)

// startWebServer 启动 Web 服务模式
func startWebServer(cfg *config.Config, staticPath string, port int) {
	fmt.Printf("启动 Web 服务模式...\n")
	fmt.Printf("  端口: %d\n", port)
	fmt.Printf("  静态文件: %s\n", staticPath)

	// 初始化数据库
	db, err := config.InitDB(cfg.Database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化数据库失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  数据库: OK")

	// 创建调度器
	sched, err := scheduler.NewScheduler(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建调度器失败: %v\n", err)
		os.Exit(1)
	}

	// 设置路由
	r := router.Setup(cfg, db, staticPath)

	// 启动 HTTP 服务器
	addr := ":" + strconv.Itoa(port)
	fmt.Printf("\n🚀 Web 控制台: http://localhost%s\n", addr)

	// 优雅退出
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		fmt.Println("\n正在停止 Web 服务...")
		sched.Stop()
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		logger.Info.Println("Web 服务已停止")
		os.Exit(0)
	}()

	if err := r.Run(addr); err != nil {
		fmt.Fprintf(os.Stderr, "启动 HTTP 服务失败: %v\n", err)
		os.Exit(1)
	}
}

// ensureDBClosed 确保数据库关闭
func ensureDBClosed(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}
}
