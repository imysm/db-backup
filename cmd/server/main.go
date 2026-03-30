// Package main 数据库备份系统入口
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/imysm/db-backup/internal/config"
	"github.com/imysm/db-backup/internal/logger"
	"github.com/imysm/db-backup/internal/scheduler"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	// 命令行参数
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	validate := flag.Bool("validate", false, "仅验证数据库连接")
	runTask := flag.String("run", "", "立即执行指定任务ID")
	showVersion := flag.Bool("version", false, "显示版本信息")

	// Web 服务模式
	webMode := flag.Bool("web", false, "启动 Web 服务模式")
	staticPath := flag.String("static", "", "静态文件目录 (用于 Web 模式)")
	port := flag.Int("port", 8080, "HTTP 端口 (用于 Web 模式)")

	flag.Parse()

	// 显示版本
	if *showVersion {
		fmt.Printf("db-backup version %s (built at %s)\n", version, buildTime)
		return
	}

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("数据库备份系统 v%s\n", version)
	fmt.Printf("配置文件: %s\n", *configPath)
	fmt.Printf("任务数量: %d\n", len(cfg.Tasks))

	// 初始化日志
	if err := logger.Init(cfg.Log.File, cfg.Log.Level); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}

	// Web 服务模式
	if *webMode {
		startWebServer(cfg, *staticPath, *port)
		return
	}

	// CLI 模式：创建调度器
	sched, err := scheduler.NewScheduler(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建调度器失败: %v\n", err)
		os.Exit(1)
	}

	// 验证模式
	if *validate {
		fmt.Println("\n验证数据库连接...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		results := sched.ValidateAll(ctx)
		for taskID, err := range results {
			if err != nil {
				fmt.Printf("  ❌ %s: %v\n", taskID, err)
			} else {
				fmt.Printf("  ✅ %s: OK\n", taskID)
			}
		}
		return
	}

	// 立即执行模式
	if *runTask != "" {
		fmt.Printf("\n立即执行任务: %s\n", *runTask)
		done, err := sched.RunNow(*runTask, &stdoutWriter{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "执行任务失败: %v\n", err)
			os.Exit(1)
		}
		// 等待任务完成
		<-done
		return
	}

	// 启动调度器
	if err := sched.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "启动调度器失败: %v\n", err)
		os.Exit(1)
	}
	defer sched.Stop()

	fmt.Println("\n备份系统已启动，按 Ctrl+C 退出...")

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n备份系统已停止")
}

// stdoutWriter 标准输出写入器
type stdoutWriter struct{}

func (w *stdoutWriter) Write(p []byte) (n int, err error) {
	return os.Stdout.Write(p)
}

func (w *stdoutWriter) WriteString(s string) (n int, err error) {
	return fmt.Print(s)
}

func (w *stdoutWriter) Close() error {
	return nil
}
