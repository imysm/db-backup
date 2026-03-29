// Package service 提供业务逻辑服务
package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	apimodel "github.com/imysm/db-backup/internal/api/model"
	"github.com/imysm/db-backup/internal/crypto"
	"github.com/imysm/db-backup/internal/executor"
	internalmodel "github.com/imysm/db-backup/internal/model"
	"github.com/imysm/db-backup/internal/util"
	"gorm.io/gorm"
)

// BackupService 备份服务
type BackupService struct {
	db        *gorm.DB
	encryptor crypto.Encryptor
	running   sync.Map // jobID -> struct{}: 正在执行的备份任务
}

// NewBackupService 创建备份服务
func NewBackupService(db *gorm.DB, encryptor crypto.Encryptor) *BackupService {
	return &BackupService{db: db, encryptor: encryptor}
}

// RunNow 立即执行备份任务
func (s *BackupService) RunNow(ctx context.Context, jobID uint) (*apimodel.BackupRecord, error) {
	// 检查是否已在执行
	if _, loaded := s.running.LoadOrStore(jobID, struct{}{}); loaded {
		return nil, fmt.Errorf("任务 %d 正在执行中，请稍后再试", jobID)
	}

	// 1. 从数据库查询任务
	var job apimodel.BackupJob
	if err := s.db.First(&job, jobID).Error; err != nil {
		s.running.Delete(jobID)
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("任务不存在")
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}

	// 2. 创建备份记录
	record := &apimodel.BackupRecord{
		JobID:     job.ID,
		JobName:   job.Name,
		StartedAt: time.Now(),
		Status:    apimodel.BackupStatusRunning,
	}
	if err := s.db.Create(record).Error; err != nil {
		s.running.Delete(jobID)
		return nil, fmt.Errorf("创建备份记录失败: %w", err)
	}

	// 3. 异步执行备份
	go func() {
		defer s.running.Delete(jobID)
		defer func() {
			if r := recover(); r != nil {
				finishedAt := time.Now()
				s.db.Model(record).Updates(map[string]interface{}{
					"finished_at":   finishedAt,
					"duration":      int(finishedAt.Sub(record.StartedAt).Seconds()),
					"status":        apimodel.BackupStatusFailed,
					"error_message": fmt.Sprintf("备份 goroutine panic: %v", r),
				})
			}
		}()
		s.executeBackup(job, record)
	}()

	return record, nil
}

// executeBackup 执行备份
func (s *BackupService) executeBackup(job apimodel.BackupJob, record *apimodel.BackupRecord) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	// 更新状态为运行中
	s.db.Model(record).Update("status", apimodel.BackupStatusRunning)

	// 转换为内部模型
	task := s.convertJobToTask(job)

	// 解密密码（兼容未加密旧数据：解密失败则使用原文）
	if task.Database.Password != "" {
		decrypted, err := s.encryptor.DecryptString(task.Database.Password)
		if err == nil {
			task.Database.Password = decrypted
		}
		// 解密失败说明是旧数据未加密，直接使用原文
	}

	// 创建执行器
	executorInst, err := executor.NewExecutor(internalmodel.DatabaseType(job.DatabaseType))
	if err != nil {
		s.recordFailure(record, err)
		return
	}

	// 执行备份（带重试机制）
	var result *internalmodel.BackupResult
	var backupErr error
	const maxRetries = 3
	retryDelays := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, backupErr = executorInst.Backup(ctx, task, &nilWriter{})
		if backupErr == nil {
			break
		}
		if attempt < maxRetries && isRetryableError(backupErr) {
			delay := retryDelays[attempt]
			fmt.Printf("[重试] 备份失败 (第%d次)，%v后重试: %v\n", attempt+1, delay, backupErr)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				s.recordFailure(record, ctx.Err())
				return
			}
			continue
		}
		break
	}
	if backupErr != nil {
		s.recordFailure(record, backupErr)
		return
	}

	// 更新备份记录
	finishedAt := time.Now()
	updates := map[string]interface{}{
		"finished_at": finishedAt,
		"duration":    int(finishedAt.Sub(record.StartedAt).Seconds()),
		"file_path":   result.FilePath,
		"file_size":   result.FileSize,
		"checksum":    result.Checksum,
		"status":      apimodel.BackupStatusSuccess,
	}

	// 如果有文件路径，计算校验和
	if result.FilePath != "" {
		checksum, err := util.CalculateChecksum(result.FilePath)
		if err == nil {
			updates["checksum"] = checksum
		}

		// 获取文件大小
		if info, err := os.Stat(result.FilePath); err == nil {
			updates["file_size"] = info.Size()
		}
	}

	s.db.Model(record).Updates(updates)

	// 更新任务的最后运行时间
	s.db.Model(&job).Update("last_run", finishedAt)
}

// recordFailure 记录失败
func (s *BackupService) recordFailure(record *apimodel.BackupRecord, err error) {
	finishedAt := time.Now()
	s.db.Model(record).Updates(map[string]interface{}{
		"finished_at":   finishedAt,
		"duration":      int(finishedAt.Sub(record.StartedAt).Seconds()),
		"status":        apimodel.BackupStatusFailed,
		"error_message": err.Error(),
	})
}

// convertJobToTask 将 API 模型的 Job 转换为内部模型的 Task
func (s *BackupService) convertJobToTask(job apimodel.BackupJob) *internalmodel.BackupTask {
	return &internalmodel.BackupTask{
		ID:      fmt.Sprintf("%d", job.ID),
		Name:    job.Name,
		Enabled: job.Enabled,
		Database: internalmodel.DatabaseConfig{
			Type:     internalmodel.DatabaseType(job.DatabaseType),
			Host:     job.Host,
			Port:     job.Port,
			Username: job.Username,
			Password: job.Password,
			Database: job.Database,
		},
		Schedule: internalmodel.ScheduleConfig{
			Cron: job.Schedule,
		},
		Storage: internalmodel.StorageConfig{
			Type: string(job.StorageType),
			Path: "/var/lib/db-backup",
		},
		Compression: internalmodel.CompressionConfig{
			Enabled: job.Compress,
			Type:    "gzip",
		},
	}
}

// GetStoragePath 获取存储路径
func (s *BackupService) GetStoragePath(job *apimodel.BackupJob) string {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.sql.gz", job.Database, timestamp)
	return filepath.Join("/var/lib/db-backup", fmt.Sprintf("%d", job.ID), filename)
}

// nilWriter 空日志写入器
type nilWriter struct{}

func (w *nilWriter) Write(p []byte) (n int, err error)       { return len(p), nil }
func (w *nilWriter) WriteString(s string) (n int, err error) { return len(s), nil }
func (w *nilWriter) Close() error                            { return nil }

// isRetryableError 判断错误是否可重试（网络超时、连接失败等，不重试配置错误）
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// 配置错误（如 unknown database、access denied）不应重试
		nonRetryable := []string{
			"unknown database", "access denied", "authentication failed",
			"unknown option", "syntax error", "usage: ",
		}
		stderr := strings.ToLower(string(exitErr.Stderr))
		for _, kw := range nonRetryable {
			if strings.Contains(stderr, kw) {
				return false
			}
		}
		// 非 0 exit code 但无明确错误信息时，保守不重试
		return false
	}
	errMsg := err.Error()
	retryableKeywords := []string{
		"connection refused", "timeout", "deadline exceeded",
		"temporary failure", "network", "etimedout", "econnrefused",
	}
	for _, kw := range retryableKeywords {
		if strings.Contains(strings.ToLower(errMsg), kw) {
			return true
		}
	}
	return false
}
