// Package executor - SQL Server 执行器实现
package executor

import (
	"context"
	"fmt"
	"github.com/imysm/db-backup/internal/util"
	"os"
	"path/filepath"
	"time"

	"github.com/imysm/db-backup/internal/model"
)

// SQLServerExecutor SQL Server 备份执行器
type SQLServerExecutor struct {
	cmdRunner CommandRunner
}

// NewSQLServerExecutor 创建 SQL Server 执行器
func NewSQLServerExecutor() *SQLServerExecutor {
	return &SQLServerExecutor{
		cmdRunner: &DefaultCommandRunner{},
	}
}

// Type 返回执行器类型
func (e *SQLServerExecutor) Type() string {
	return string(model.SQLServer)
}

// Validate 验证数据库连接
func (e *SQLServerExecutor) Validate(ctx context.Context, cfg *model.DatabaseConfig) error {
	// 使用 sqlcmd 验证连接
	args := []string{
		"-S", fmt.Sprintf("%s,%d", cfg.Host, cfg.Port),
		"-U", cfg.Username,
		"-d", cfg.Database,
		"-Q", "SELECT 1",
	}

	env := []string{fmt.Sprintf("SQLCMDPASSWORD=%s", cfg.Password)}

	_, err := e.cmdRunner.Run(ctx, "sqlcmd", args, env)
	if err != nil {
		return fmt.Errorf("SQL Server 连接失败: %w", err)
	}

	return nil
}

// Backup 执行 SQL Server 备份
func (e *SQLServerExecutor) Backup(ctx context.Context, task *model.BackupTask, writer model.LogWriter) (*model.BackupResult, error) {
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
	fileName := fmt.Sprintf("%s_%s.bak", dbName, timestamp)
	filePath := filepath.Join(storagePath, fileName)

	// 安全校验：验证数据库名和文件路径
	if err := util.ValidateDatabaseName(task.Database.Database); err != nil {
		return failResult(task.ID, traceID, startTime, fmt.Errorf("数据库名校验失败: %w", err)), err
	}
	if err := util.ValidateBackupPath(filePath); err != nil {
		return failResult(task.ID, traceID, startTime, fmt.Errorf("备份路径校验失败: %w", err)), err
	}

	// 构建备份命令
	backupSQL := fmt.Sprintf(
		"BACKUP DATABASE [%s] TO DISK='%s' WITH COMPRESSION, STATS=10",
		task.Database.Database,
		filePath,
	)

	args := []string{
		"-S", fmt.Sprintf("%s,%d", task.Database.Host, task.Database.Port),
		"-U", task.Database.Username,
		"-d", task.Database.Database,
		"-Q", backupSQL,
	}

	env := []string{fmt.Sprintf("SQLCMDPASSWORD=%s", task.Database.Password)}

	// 写入日志
	if writer != nil {
		writer.WriteString(fmt.Sprintf("[%s] 开始备份: %s\n", startTime.Format("2006-01-02 15:04:05"), task.Name))
		writer.WriteString(fmt.Sprintf("[%s] 执行备份命令...\n", time.Now().Format("15:04:05")))
	}

	// 执行备份命令
	if writer != nil {
		err := e.cmdRunner.RunWithOutput(ctx, "sqlcmd", args, env, writer)
		if err != nil {
			return failResult(task.ID, traceID, startTime, fmt.Errorf("SQL Server 备份失败: %w", err)), err
		}
	} else {
		_, err := e.cmdRunner.Run(ctx, "sqlcmd", args, env)
		if err != nil {
			return failResult(task.ID, traceID, startTime, fmt.Errorf("SQL Server 备份失败: %w", err)), err
		}
	}

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

	// 处理加密
	if task.Storage.Encryption.Enabled {
		encryptedPath, err := EncryptBackupFile(filePath, task.Storage.Encryption)
		if err != nil {
			return failResult(task.ID, traceID, startTime, fmt.Errorf("加密备份文件失败: %w", err)), err
		}
		filePath = encryptedPath
		if writer != nil {
			writer.WriteString(fmt.Sprintf("[%s] 加密完成: %s\n", time.Now().Format("15:04:05"), filePath))
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
