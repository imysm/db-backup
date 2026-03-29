// Package executor - PostgreSQL 执行器实现
package executor

import (
	"context"
	"fmt"
	"github.com/imysm/db-backup/internal/util"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/imysm/db-backup/internal/model"
)

// PostgresExecutor PostgreSQL 备份执行器
type PostgresExecutor struct {
	cmdRunner CommandRunner
}

// NewPostgresExecutor 创建 PostgreSQL 执行器
func NewPostgresExecutor() *PostgresExecutor {
	return &PostgresExecutor{
		cmdRunner: &DefaultCommandRunner{},
	}
}

// Type 返回执行器类型
func (e *PostgresExecutor) Type() string {
	return string(model.PostgreSQL)
}

// Validate 验证数据库连接
func (e *PostgresExecutor) Validate(ctx context.Context, cfg *model.DatabaseConfig) error {
	// 设置环境变量
	env := e.buildEnv(cfg)

	args := []string{
		"-h", cfg.Host,
		"-p", fmt.Sprintf("%d", cfg.Port),
		"-U", cfg.Username,
		"-d", cfg.Database,
		"-c", "SELECT 1",
	}

	_, err := e.cmdRunner.Run(ctx, "psql", args, env)
	if err != nil {
		return fmt.Errorf("PostgreSQL 连接失败: %w", err)
	}

	return nil
}

// Backup 执行 PostgreSQL 备份
func (e *PostgresExecutor) Backup(ctx context.Context, task *model.BackupTask, writer model.LogWriter) (*model.BackupResult, error) {
	startTime := time.Now()
	traceID := generateTraceID()

	result := &model.BackupResult{
		TaskID:    task.ID,
		TraceID:   traceID,
		StartTime: startTime,
		Status:    model.TaskStatusPending,
	}

	// 确保存储目录存在
	storagePath := task.Storage.Path
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return failResult(task.ID, traceID, startTime, fmt.Errorf("创建存储目录失败: %w", err)), err
	}

	// 生成备份文件名
	timestamp := startTime.Format("20060102_150405")
	dbName := sanitizeDBName(task.Database.Database)

	// 使用 -Fc 格式（自定义格式，内置压缩）
	fileName := fmt.Sprintf("%s_%s.dump", dbName, timestamp)
	filePath := filepath.Join(storagePath, fileName)

	// 构建 pg_dump 参数
	args := e.buildDumpArgs(task, filePath)
	env := e.buildEnv(&task.Database)

	// 写入日志
	if writer != nil {
		writer.WriteString(fmt.Sprintf("[%s] 开始备份: %s\n", startTime.Format("2006-01-02 15:04:05"), task.Name))
		writer.WriteString(fmt.Sprintf("[%s] 执行命令: pg_dump %s\n", time.Now().Format("15:04:05"), strings.Join(args, " ")))
	}

	// 执行备份命令
	if writer != nil {
		err := e.cmdRunner.RunWithOutput(ctx, "pg_dump", args, env, writer)
		if err != nil {
			return failResult(task.ID, traceID, startTime, fmt.Errorf("pg_dump 执行失败: %w", err)), err
		}
	} else {
		_, err := e.cmdRunner.Run(ctx, "pg_dump", args, env)
		if err != nil {
			return failResult(task.ID, traceID, startTime, fmt.Errorf("pg_dump 执行失败: %w", err)), err
		}
	}

	// pg_dump -Fc 格式自带压缩，不需要额外压缩

	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return failResult(task.ID, traceID, startTime, fmt.Errorf("获取文件信息失败: %w", err)), err
	}

	// 计算校验和
	checksum, err := util.CalculateChecksum(filePath)
	if err != nil {
		if writer != nil {
			writer.WriteString(fmt.Sprintf("[WARN] 计算校验和失败: %v\n", err))
		}
	}

	endTime := time.Now()
	result.EndTime = endTime
	result.Duration = endTime.Sub(startTime)
	result.Status = model.TaskStatusSuccess
	result.FilePath = filePath
	result.FileSize = fileInfo.Size()
	result.Checksum = checksum

	if writer != nil {
		writer.WriteString(fmt.Sprintf("[%s] 备份完成: %s (%s)\n",
			endTime.Format("2006-01-02 15:04:05"),
			filepath.Base(filePath),
			util.FormatFileSize(fileInfo.Size())))
	}

	return result, nil
}

// buildDumpArgs 构建 pg_dump 参数
func (e *PostgresExecutor) buildDumpArgs(task *model.BackupTask, filePath string) []string {
	cfg := task.Database

	// -Fc 使用自定义格式（内置压缩，推荐用于大型数据库）
	args := []string{
		"-h", cfg.Host,
		"-p", fmt.Sprintf("%d", cfg.Port),
		"-U", cfg.Username,
		"-d", cfg.Database,
		"-Fc", // 自定义格式，内置压缩（P0功能）
		"-f", filePath,
	}

	// 额外参数
	if cfg.Params != nil {
		if v, ok := cfg.Params["format"]; ok {
			// 允许覆盖格式（plain, custom, tar, directory）
			switch v {
			case "plain":
				args = removeArg(args, "-Fc")
				args = append(args, "-Fp")
			case "tar":
				args = removeArg(args, "-Fc")
				args = append(args, "-Ft")
			case "directory":
				args = removeArg(args, "-Fc")
				args = append(args, "-Fd")
			}
		}
		if v, ok := cfg.Params["jobs"]; ok {
			// 并行备份（仅 -Fd 格式支持）
			args = append(args, "-j", v)
		}
		if v, ok := cfg.Params["schema"]; ok {
			args = append(args, "-n", util.SanitizeParam(v))
		}
		if v, ok := cfg.Params["exclude_schema"]; ok {
			args = append(args, "-N", util.SanitizeParam(v))
		}
		if v, ok := cfg.Params["table"]; ok {
			args = append(args, "-t", util.SanitizeParam(v))
		}
		if v, ok := cfg.Params["exclude_table"]; ok {
			args = append(args, "-T", util.SanitizeParam(v))
		}
		if v, ok := cfg.Params["compress_level"]; ok {
			// 压缩级别 (0-9)
			args = append(args, fmt.Sprintf("-Z%s", v))
		}
	}

	return args
}

// buildEnv 构建环境变量
func (e *PostgresExecutor) buildEnv(cfg *model.DatabaseConfig) []string {
	env := []string{
		fmt.Sprintf("PGPASSWORD=%s", cfg.Password),
	}
	return env
}

// removeArg 从参数列表中移除指定参数
func removeArg(args []string, target string) []string {
	var result []string
	for _, arg := range args {
		if arg != target {
			result = append(result, arg)
		}
	}
	return result
}
