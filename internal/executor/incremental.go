package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/imysm/db-backup/internal/model"
)

// IncrementalBackupResult 增量备份结果
type IncrementalBackupResult struct {
	*model.BackupResult
	BinlogFile string `json:"binlog_file"`
	BinlogPos  uint32 `json:"binlog_pos"`
}

// BinlogPosition Binlog 位置信息
type BinlogPosition struct {
	File     string `json:"file"`
	Position uint32 `json:"position"`
}

// IncrementalBackupExecute 执行增量备份
func (e *MySQLExecutor) IncrementalBackupExecute(ctx context.Context, task *model.BackupTask, writer model.LogWriter) (*IncrementalBackupResult, error) {
	startTime := time.Now()
	traceID := generateTraceID()

	// 获取当前 binlog 位置
	binlogPos, err := e.getBinlogPosition(ctx, task)
	if err != nil {
		writer.WriteString(fmt.Sprintf("获取 binlog 位置失败: %v", err))
		return nil, fmt.Errorf("failed to get binlog position: %w", err)
	}

	writer.WriteString(fmt.Sprintf("当前 Binlog 位置: %s:%d", binlogPos.File, binlogPos.Position))

	// 执行备份
	result, err := e.Backup(ctx, task, writer)
	if err != nil {
		return &IncrementalBackupResult{
			BackupResult: failResult(task.ID, traceID, startTime, err),
			BinlogFile:  binlogPos.File,
			BinlogPos:   binlogPos.Position,
		}, err
	}

	result.Status = model.TaskStatusSuccess

	return &IncrementalBackupResult{
		BackupResult: result,
		BinlogFile:  binlogPos.File,
		BinlogPos:   binlogPos.Position,
	}, nil
}

// getBinlogPosition 获取 MySQL 当前 binlog 位置
func (e *MySQLExecutor) getBinlogPosition(ctx context.Context, task *model.BackupTask) (*BinlogPosition, error) {
	query := "SHOW MASTER STATUS"
	
	output, err := e.runQuery(ctx, task, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// 解析输出
	pos, err := parseBinlogPosition(output)
	if err != nil {
		return nil, err
	}

	return pos, nil
}

// parseBinlogPosition 解析 binlog 位置
func parseBinlogPosition(output string) (*BinlogPosition, error) {
	// 简单解析 - 格式: File\tPosition
	var pos BinlogPosition
	_, err := fmt.Sscanf(output, "%s %d", &pos.File, &pos.Position)
	if err != nil {
		return nil, fmt.Errorf("failed to parse binlog position: %w", err)
	}
	return &pos, nil
}

// runQuery 执行 SQL 查询
func (e *MySQLExecutor) runQuery(ctx context.Context, task *model.BackupTask, query string) (string, error) {
	args := []string{
		"-h" + task.Database.Host,
		"-P" + fmt.Sprintf("%d", task.Database.Port),
		"-u" + task.Database.Username,
		"--password=" + task.Database.Password,
		"-e", query,
	}

	output, err := e.cmdRunner.Run(ctx, "mysql", args, nil)
	if err != nil {
		return "", fmt.Errorf("mysql query failed: %w", err)
	}

	return string(output), nil
}

// GetBinlogInfo 获取 binlog 信息
func GetBinlogInfo(ctx context.Context, host string, port int, username, password string) (*BinlogPosition, error) {
	args := []string{
		"-h" + host,
		"-P" + fmt.Sprintf("%d", port),
		"-u" + username,
		"--password=" + password,
		"-e", "SHOW MASTER STATUS",
	}

	output, err := runCommand(ctx, "mysql", args, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get binlog info: %w", err)
	}

	return parseBinlogPosition(string(output))
}
