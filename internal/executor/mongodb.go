// Package executor - MongoDB 执行器实现
package executor

import (
	"context"
	"fmt"
	"github.com/imysm/db-backup/internal/util"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/imysm/db-backup/internal/model"
)

// MongoDBExecutor MongoDB 备份执行器
type MongoDBExecutor struct {
	cmdRunner CommandRunner
}

// NewMongoDBExecutor 创建 MongoDB 执行器
func NewMongoDBExecutor() *MongoDBExecutor {
	return &MongoDBExecutor{
		cmdRunner: &DefaultCommandRunner{},
	}
}

// Type 返回执行器类型
func (e *MongoDBExecutor) Type() string {
	return string(model.MongoDB)
}

// Validate 验证数据库连接
func (e *MongoDBExecutor) Validate(ctx context.Context, cfg *model.DatabaseConfig) error {
	args := []string{
		"--host", cfg.Host,
		"--port", fmt.Sprintf("%d", cfg.Port),
	}

	if cfg.Username != "" {
		args = append(args, "--username", cfg.Username)
	}
	if cfg.Database != "" {
		args = append(args, "--db", cfg.Database)
	}

	args = append(args, "--eval", "db.runCommand({ping:1})")

	var env []string
	if cfg.Password != "" {
		env = append(env, fmt.Sprintf("MONGOPASSWORD=%s", cfg.Password))
	}

	_, err := e.cmdRunner.Run(ctx, "mongosh", args, env)
	if err != nil {
		// 尝试使用旧版 mongo 客户端
		_, err = e.cmdRunner.Run(ctx, "mongo", args, env)
		if err != nil {
			return fmt.Errorf("MongoDB 连接失败: %w", err)
		}
	}

	return nil
}

// Backup 执行 MongoDB 备份
func (e *MongoDBExecutor) Backup(ctx context.Context, task *model.BackupTask, writer model.LogWriter) (*model.BackupResult, error) {
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

	// 生成备份目录名（mongodump 输出到目录）
	timestamp := startTime.Format("20060102_150405")
	dbName := sanitizeDBName(task.Database.Database)
	if dbName == "" || dbName == "all" {
		dbName = "all"
	}

	backupDirName := fmt.Sprintf("%s_%s", dbName, timestamp)
	backupDir := filepath.Join(storagePath, backupDirName)

	// 构建 mongodump 参数
	args, env := e.buildDumpArgs(task, backupDir)

	// 写入日志
	if writer != nil {
		writer.WriteString(fmt.Sprintf("[%s] 开始备份: %s\n", startTime.Format("2006-01-02 15:04:05"), task.Name))
		writer.WriteString(fmt.Sprintf("[%s] 执行命令: mongodump %s\n", time.Now().Format("15:04:05"), strings.Join(args, " ")))
	}

	// 执行备份命令
	if writer != nil {
		err := e.cmdRunner.RunWithOutput(ctx, "mongodump", args, env, writer)
		if err != nil {
			return failResult(task.ID, traceID, startTime, fmt.Errorf("mongodump 执行失败: %w", err)), err
		}
	} else {
		_, err := e.cmdRunner.Run(ctx, "mongodump", args, env)
		if err != nil {
			return failResult(task.ID, traceID, startTime, fmt.Errorf("mongodump 执行失败: %w", err)), err
		}
	}

	// 压缩备份目录为单个文件（如果启用了压缩）
	finalPath := backupDir
	if task.Compression.Enabled {
		compressedPath, err := e.compressDir(ctx, backupDir, task.Compression, writer)
		if err != nil {
			if writer != nil {
				writer.WriteString(fmt.Sprintf("[WARN] 压缩失败: %v\n", err))
			}
		} else {
			finalPath = compressedPath
			// 删除未压缩目录
			os.RemoveAll(backupDir)
			if writer != nil {
				writer.WriteString(fmt.Sprintf("[%s] 压缩完成: %s\n", time.Now().Format("15:04:05"), finalPath))
			}
		}
	}

	// 处理加密
	if task.Storage.Encryption.Enabled {
		encryptedPath, err := EncryptBackupFile(finalPath, task.Storage.Encryption)
		if err != nil {
			return failResult(task.ID, traceID, startTime, fmt.Errorf("加密备份文件失败: %w", err)), err
		}
		finalPath = encryptedPath
		if writer != nil {
			writer.WriteString(fmt.Sprintf("[%s] 加密完成: %s\n", time.Now().Format("15:04:05"), finalPath))
		}
	}

	// 获取文件/目录信息
	var size int64
	var checkPath string

	fileInfo, err := os.Stat(finalPath)
	if err != nil {
		return failResult(task.ID, traceID, startTime, fmt.Errorf("获取文件信息失败: %w", err)), err
	}

	if fileInfo.IsDir() {
		// 计算目录大小
		size, err = e.dirSize(finalPath)
		if err != nil {
			if writer != nil {
				writer.WriteString(fmt.Sprintf("[WARN] 计算目录大小失败: %v\n", err))
			}
		}
		checkPath = filepath.Join(finalPath, "metadata.json")
		if _, err := os.Stat(checkPath); os.IsNotExist(err) {
			// 如果没有 metadata.json，使用第一个 bson 文件
			checkPath = ""
		}
	} else {
		size = fileInfo.Size()
		checkPath = finalPath
	}

	// 计算校验和
	var checksum string
	if checkPath != "" {
		checksum, err = util.CalculateChecksum(checkPath)
		if err != nil {
			if writer != nil {
				writer.WriteString(fmt.Sprintf("[WARN] 计算校验和失败: %v\n", err))
			}
		}
	}

	endTime := time.Now()
	result.EndTime = endTime
	result.Duration = endTime.Sub(startTime)
	result.Status = model.TaskStatusSuccess
	result.FilePath = finalPath
	result.FileSize = size
	result.Checksum = checksum

	if writer != nil {
		writer.WriteString(fmt.Sprintf("[%s] 备份完成: %s (%s)\n",
			endTime.Format("2006-01-02 15:04:05"),
			filepath.Base(finalPath),
			util.FormatFileSize(size)))
	}

	return result, nil
}

// buildDumpArgs 构建 mongodump 参数
func (e *MongoDBExecutor) buildDumpArgs(task *model.BackupTask, outputDir string) ([]string, []string) {
	cfg := task.Database

	args := []string{
		"--host", cfg.Host,
		"--port", fmt.Sprintf("%d", cfg.Port),
		"--gzip", // 使用 gzip 压缩（P0功能）
		"--out", outputDir,
	}

	if cfg.Username != "" {
		args = append(args, "--username", cfg.Username)
	}
	if cfg.Database != "" && cfg.Database != "all" {
		args = append(args, "--db", cfg.Database)
	}

	var env []string
	if cfg.Password != "" {
		env = append(env, fmt.Sprintf("MONGOPASSWORD=%s", cfg.Password))
	}

	// 额外参数
	if cfg.Params != nil {
		if v, ok := cfg.Params["authenticationDatabase"]; ok {
			args = append(args, "--authenticationDatabase", util.SanitizeParam(v))
		}
		if v, ok := cfg.Params["uri"]; ok {
			// 使用 URI 连接字符串（允许特殊字符如 @, :, / 用于连接字符串格式）
			args = append(args, "--uri", util.SanitizeParam(v))
		}
		if v, ok := cfg.Params["collection"]; ok {
			args = append(args, "--collection", util.SanitizeParam(v))
		}
		if v, ok := cfg.Params["query"]; ok {
			args = append(args, "--query", util.SanitizeParam(v))
		}
		if v, ok := cfg.Params["readPreference"]; ok {
			args = append(args, "--readPreference", util.SanitizeParam(v))
		}
	}

	return args, env
}

// compressDir 压缩目录
func (e *MongoDBExecutor) compressDir(ctx context.Context, dirPath string, comp model.CompressionConfig, writer model.LogWriter) (string, error) {
	compressedPath := dirPath + ".tar.gz"

	// 使用 tar + gzip 压缩（流式写入，避免大目录 OOM）
	outFile, err := os.Create(compressedPath)
	if err != nil {
		return "", fmt.Errorf("创建压缩文件失败: %w", err)
	}
	defer outFile.Close()

	cmd := exec.CommandContext(ctx, "tar", "-czf", "-", "-C", filepath.Dir(dirPath), filepath.Base(dirPath))
	cmd.Stdout = outFile

	if writer != nil {
		writer.WriteString(fmt.Sprintf("[%s] 开始压缩目录...\n", time.Now().Format("15:04:05")))
	}

	if err := cmd.Run(); err != nil {
		os.Remove(compressedPath)
		return "", fmt.Errorf("压缩失败: %w", err)
	}

	return compressedPath, nil
}

// dirSize 计算目录大小
func (e *MongoDBExecutor) dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
