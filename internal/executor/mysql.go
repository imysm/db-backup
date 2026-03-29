// Package executor - MySQL 执行器实现
package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/imysm/db-backup/internal/model"
	"github.com/imysm/db-backup/internal/util"
)

// MySQLExecutor MySQL 备份执行器
type MySQLExecutor struct {
	cmdRunner CommandRunner
}

// NewMySQLExecutor 创建 MySQL 执行器
func NewMySQLExecutor() *MySQLExecutor {
	return &MySQLExecutor{
		cmdRunner: &DefaultCommandRunner{},
	}
}

// Type 返回执行器类型
func (e *MySQLExecutor) Type() string {
	return string(model.MySQL)
}

// Validate 验证数据库连接
func (e *MySQLExecutor) Validate(ctx context.Context, cfg *model.DatabaseConfig) error {
	defaultsFile, err := createMySQLDefaultsExtraFile(cfg.Password)
	if err != nil {
		return fmt.Errorf("创建密码配置文件失败: %w", err)
	}
	defer os.Remove(defaultsFile)

	args := []string{
		fmt.Sprintf("--defaults-extra-file=%s", defaultsFile),
		fmt.Sprintf("-h%s", cfg.Host),
		fmt.Sprintf("-P%d", cfg.Port),
		fmt.Sprintf("-u%s", cfg.Username),
		"-e", "SELECT 1",
	}

	_, err = e.cmdRunner.Run(ctx, "mysql", args, nil)
	if err != nil {
		return fmt.Errorf("MySQL 连接失败: %w", err)
	}

	return nil
}

// Backup 执行 MySQL 备份
func (e *MySQLExecutor) Backup(ctx context.Context, task *model.BackupTask, writer model.LogWriter) (*model.BackupResult, error) {
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
	fileName := fmt.Sprintf("%s_%s.sql", dbName, timestamp)
	filePath := filepath.Join(storagePath, fileName)

	// 构建 mysqldump 参数
	args := e.buildDumpArgs(task, filePath)

	// 创建临时密码配置文件
	defaultsFile, err := createMySQLDefaultsExtraFile(task.Database.Password)
	if err != nil {
		return failResult(task.ID, traceID, startTime, fmt.Errorf("创建密码配置文件失败: %w", err)), err
	}
	defer os.Remove(defaultsFile)
	args = append([]string{fmt.Sprintf("--defaults-extra-file=%s", defaultsFile)}, args...)

	// 写入日志
	if writer != nil {
		writer.WriteString(fmt.Sprintf("[%s] 开始备份: %s\n", startTime.Format("2006-01-02 15:04:05"), task.Name))
		writer.WriteString(fmt.Sprintf("[%s] 执行命令: mysqldump %s\n", time.Now().Format("15:04:05"), strings.Join(args[:len(args)-1], " ")))
	}

	// 执行备份命令
	if writer != nil {
		err := e.cmdRunner.RunWithOutput(ctx, "mysqldump", args, nil, writer)
		if err != nil {
			return failResult(task.ID, traceID, startTime, fmt.Errorf("mysqldump 执行失败: %w", err)), err
		}
	} else {
		_, err := e.cmdRunner.Run(ctx, "mysqldump", args, nil)
		if err != nil {
			return failResult(task.ID, traceID, startTime, fmt.Errorf("mysqldump 执行失败: %w", err)), err
		}
	}

	// 处理压缩
	finalPath := filePath
	if task.Compression.Enabled {
		compressedPath, err := e.compressFile(ctx, filePath, task.Compression, writer)
		if err != nil {
			// 压缩失败不影响备份成功，只是警告
			if writer != nil {
				writer.WriteString(fmt.Sprintf("[WARN] 压缩失败: %v\n", err))
			}
		} else {
			finalPath = compressedPath
			os.Remove(filePath) // 删除未压缩文件
			if writer != nil {
				writer.WriteString(fmt.Sprintf("[%s] 压缩完成: %s\n", time.Now().Format("15:04:05"), finalPath))
			}
		}
	}

	// 获取文件信息
	fileInfo, err := os.Stat(finalPath)
	if err != nil {
		return failResult(task.ID, traceID, startTime, fmt.Errorf("获取文件信息失败: %w", err)), err
	}

	// 计算校验和
	checksum, err := util.CalculateChecksum(finalPath)
	if err != nil {
		if writer != nil {
			writer.WriteString(fmt.Sprintf("[WARN] 计算校验和失败: %v\n", err))
		}
	}

	endTime := time.Now()
	result.EndTime = endTime
	result.Duration = endTime.Sub(startTime)
	result.Status = model.TaskStatusSuccess
	result.FilePath = finalPath
	result.FileSize = fileInfo.Size()
	result.Checksum = checksum

	if writer != nil {
		writer.WriteString(fmt.Sprintf("[%s] 备份完成: %s (%s)\n",
			endTime.Format("2006-01-02 15:04:05"),
			filepath.Base(finalPath),
			util.FormatFileSize(fileInfo.Size())))
	}

	return result, nil
}

// buildDumpArgs 构建 mysqldump 参数
func (e *MySQLExecutor) buildDumpArgs(task *model.BackupTask, filePath string) []string {
	cfg := task.Database
	args := []string{
		fmt.Sprintf("-h%s", cfg.Host),
		fmt.Sprintf("-P%d", cfg.Port),
		fmt.Sprintf("-u%s", cfg.Username),
		"--single-transaction",  // InnoDB 一致性备份
		"--routines",            // 备份存储过程和函数
		"--triggers",            // 备份触发器
		"--events",              // 备份事件
		"--set-gtid-purged=OFF", // 避免 GTID 问题
		"--compress",            // 网络传输压缩（P0功能）
		fmt.Sprintf("--result-file=%s", filePath),
	}

	// 额外参数
	if cfg.Params != nil {
		if v, ok := cfg.Params["max_allowed_packet"]; ok {
			args = append(args, fmt.Sprintf("--max-allowed-packet=%s", v))
		}
		if v, ok := cfg.Params["quick"]; ok && v == "true" {
			args = append(args, "--quick")
		}
		if v, ok := cfg.Params["lock_tables"]; ok && v == "true" {
			args = append(args, "--lock-tables")
		}
		if v, ok := cfg.Params["where"]; ok {
			sanitized, err := util.SanitizeWhereClause(v)
			if err != nil {
				// 不合法的 where 参数，跳过不添加
			} else {
				args = append(args, fmt.Sprintf("--where=%s", sanitized))
			}
		}
		if v, ok := cfg.Params["ignore_table"]; ok {
			// 支持多个忽略表
			tables := strings.Split(v, ",")
			for _, t := range tables {
				args = append(args, fmt.Sprintf("--ignore-table=%s", sanitizeTableName(strings.TrimSpace(t))))
			}
		}
	}

	// 指定数据库
	if cfg.Database != "" {
		databases := strings.Split(cfg.Database, ",")
		args = append(args, databases...)
	} else {
		args = append(args, "--all-databases")
	}

	return args
}

// compressFile 压缩文件
func (e *MySQLExecutor) compressFile(ctx context.Context, filePath string, comp model.CompressionConfig, writer model.LogWriter) (string, error) {
	compressedPath := filePath + ".gz"

	level := fmt.Sprintf("-%d", comp.Level)
	if comp.Level == 0 {
		level = "-6" // 默认级别
	}

	args := []string{level, "-c", filePath}

	if writer != nil {
		writer.WriteString(fmt.Sprintf("[%s] 开始压缩 (级别 %s)...\n", time.Now().Format("15:04:05"), level))
	}

	// 使用流式管道压缩，避免大文件 OOM
	gzipCmd := exec.CommandContext(ctx, "gzip", args...)
	outFile, err := os.Create(compressedPath)
	if err != nil {
		return "", fmt.Errorf("创建压缩文件失败: %w", err)
	}
	defer outFile.Close()

	gzipCmd.Stdout = outFile
	if err := gzipCmd.Run(); err != nil {
		os.Remove(compressedPath)
		return "", fmt.Errorf("压缩失败: %w", err)
	}

	return compressedPath, nil
}

// 辅助函数

func generateTraceID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func sanitizeDBName(name string) string {
	if name == "" {
		return "all"
	}
	// 移除不安全字符
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, " ", "_")
	// 过滤路径遍历和 shell 特殊字符
	name = strings.ReplaceAll(name, "..", "_")
	name = strings.ReplaceAll(name, "&", "_")
	name = strings.ReplaceAll(name, "|", "_")
	name = strings.ReplaceAll(name, ";", "_")
	name = strings.ReplaceAll(name, "$", "_")
	name = strings.ReplaceAll(name, "`", "_")
	name = strings.ReplaceAll(name, "'", "_")
	name = strings.ReplaceAll(name, "\"", "_")
	name = strings.ReplaceAll(name, "(", "_")
	name = strings.ReplaceAll(name, ")", "_")
	name = strings.ReplaceAll(name, "{", "_")
	name = strings.ReplaceAll(name, "}", "_")
	name = strings.ReplaceAll(name, "<", "_")
	name = strings.ReplaceAll(name, ">", "_")
	name = strings.ReplaceAll(name, "\n", "_")
	name = strings.ReplaceAll(name, "\r", "_")
	return name
}

// sanitizeTableName sanitizes table names (e.g. for --ignore-table)
func sanitizeTableName(name string) string {
	return sanitizeDBName(name)
}

// buildMinimalEnv 构建最小化环境变量，只继承 PATH 和 LANG，加上额外的环境变量
func buildMinimalEnv(extraEnv []string) []string {
	env := []string{"PATH=" + os.Getenv("PATH")}
	if lang := os.Getenv("LANG"); lang != "" {
		env = append(env, "LANG="+lang)
	}
	env = append(env, extraEnv...)
	return env
}

// createMySQLDefaultsExtraFile 创建临时 MySQL 配置文件用于传递密码
// 调用者负责在用完后删除此文件
func createMySQLDefaultsExtraFile(password string) (string, error) {
	tmpFile, err := os.CreateTemp("", "mysql_defaults_*.cnf")
	if err != nil {
		return "", fmt.Errorf("创建临时配置文件失败: %w", err)
	}
	path := tmpFile.Name()

	// 设置权限为 0600，仅当前用户可读写
	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		os.Remove(path)
		return "", fmt.Errorf("设置临时配置文件权限失败: %w", err)
	}

	content := fmt.Sprintf("[client]\npassword=%s\n", password)
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(path)
		return "", fmt.Errorf("写入临时配置文件失败: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("关闭临时配置文件失败: %w", err)
	}

	return path, nil
}

// runCommand 执行命令并返回输出
func runCommand(ctx context.Context, name string, args []string, env []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = buildMinimalEnv(env)
	return cmd.CombinedOutput()
}

// runCommandWithOutput 执行命令并实时输出
func runCommandWithOutput(ctx context.Context, name string, args []string, env []string, writer model.LogWriter) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = buildMinimalEnv(env)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer stdoutPipe.Close()
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	defer stderrPipe.Close()

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if writer != nil {
			io.Copy(writer, stdoutPipe)
		} else {
			io.Copy(io.Discard, stdoutPipe)
		}
	}()
	go func() {
		defer wg.Done()
		if writer != nil {
			io.Copy(writer, stderrPipe)
		} else {
			io.Copy(io.Discard, stderrPipe)
		}
	}()

	err = cmd.Wait()
	wg.Wait()
	return err
}
