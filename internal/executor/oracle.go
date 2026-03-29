// Package executor - Oracle 执行器实现
package executor

import (
	"context"
	"fmt"
	"github.com/imysm/db-backup/internal/util"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/imysm/db-backup/internal/model"
)

// OracleExecutor Oracle 备份执行器
type OracleExecutor struct {
	cmdRunner CommandRunner
}

// NewOracleExecutor 创建 Oracle 执行器
func NewOracleExecutor() *OracleExecutor {
	return &OracleExecutor{
		cmdRunner: &DefaultCommandRunner{},
	}
}

// Type 返回执行器类型
func (e *OracleExecutor) Type() string {
	return string(model.Oracle)
}

// Validate 验证数据库连接
func (e *OracleExecutor) Validate(ctx context.Context, cfg *model.DatabaseConfig) error {
	// 使用 sqlplus 验证连接
	connectString := fmt.Sprintf("%s/%s@%s:%d/%s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	args := []string{
		"-S",
		connectString,
		"SELECT 1 FROM DUAL;",
	}

	_, err := e.cmdRunner.Run(ctx, "sqlplus", args, nil)
	if err != nil {
		return fmt.Errorf("Oracle 连接失败: %w", err)
	}

	return nil
}

// Backup 执行 Oracle 备份
func (e *OracleExecutor) Backup(ctx context.Context, task *model.BackupTask, writer model.LogWriter) (*model.BackupResult, error) {
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
	schemaName := sanitizeDBName(task.Database.Username)
	fileName := fmt.Sprintf("%s_%s.dmp", schemaName, timestamp)
	filePath := filepath.Join(storagePath, fileName)
	logFile := filepath.Join(storagePath, fmt.Sprintf("%s_%s.log", schemaName, timestamp))

	// 构建 expdp (Data Pump) 命令
	// Data Pump 支持并行和压缩
	connectString := fmt.Sprintf("%s/%s@%s:%d/%s",
		task.Database.Username,
		task.Database.Password,
		task.Database.Host,
		task.Database.Port,
		task.Database.Database,
	)

	args := []string{
		connectString,
		"directory=DATA_PUMP_DIR",
		fmt.Sprintf("dumpfile=%s", filepath.Base(filePath)),
		fmt.Sprintf("logfile=%s", filepath.Base(logFile)),
		"COMPRESSION=ALL", // 启用压缩（P0功能）
		"REUSE_DUMPFILES=Y",
	}

	// 添加 schema 参数
	if task.Database.Username != "" {
		args = append(args, fmt.Sprintf("schemas=%s", task.Database.Username))
	}

	// 额外参数
	if task.Database.Params != nil {
		if v, ok := task.Database.Params["parallel"]; ok {
			args = append(args, fmt.Sprintf("parallel=%s", util.SanitizeParam(v)))
		}
		if v, ok := task.Database.Params["tables"]; ok {
			args = append(args, fmt.Sprintf("tables=%s", util.SanitizeParam(v)))
		}
		if v, ok := task.Database.Params["exclude"]; ok {
			args = append(args, fmt.Sprintf("exclude=%s", util.SanitizeParam(v)))
		}
		if v, ok := task.Database.Params["include"]; ok {
			args = append(args, fmt.Sprintf("include=%s", util.SanitizeParam(v)))
		}
	}

	// 写入日志
	if writer != nil {
		writer.WriteString(fmt.Sprintf("[%s] 开始备份: %s\n", startTime.Format("2006-01-02 15:04:05"), task.Name))
		writer.WriteString(fmt.Sprintf("[%s] 执行 expdp 命令...\n", time.Now().Format("15:04:05")))
	}

	// 执行备份命令
	cmd := exec.CommandContext(ctx, "expdp", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("ORACLE_HOME=%s", os.Getenv("ORACLE_HOME")))

	var output []byte
	var err error

	if writer != nil {
		// 使用实时输出
		pipe, err := cmd.StdoutPipe()
		if err != nil {
			return failResult(task.ID, traceID, startTime, fmt.Errorf("创建输出管道失败: %w", err)), err
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return failResult(task.ID, traceID, startTime, fmt.Errorf("创建错误管道失败: %w", err)), err
		}

		if err := cmd.Start(); err != nil {
			return failResult(task.ID, traceID, startTime, fmt.Errorf("expdp 启动失败: %w", err)), err
		}

		// 读取输出
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := pipe.Read(buf)
				if n > 0 && writer != nil {
					writer.Write(buf[:n])
				}
				if err != nil {
					break
				}
			}
		}()

		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := stderrPipe.Read(buf)
				if n > 0 && writer != nil {
					writer.Write(buf[:n])
				}
				if err != nil {
					break
				}
			}
		}()

		if err := cmd.Wait(); err != nil {
			return failResult(task.ID, traceID, startTime, fmt.Errorf("expdp 执行失败: %w", err)), err
		}
	} else {
		output, err = cmd.CombinedOutput()
		if err != nil {
			return failResult(task.ID, traceID, startTime, fmt.Errorf("expdp 执行失败: %w, output: %s", err, string(output))), err
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
