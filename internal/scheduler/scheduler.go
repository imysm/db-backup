// Package scheduler 提供任务调度功能
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/imysm/db-backup/internal/executor"
	"github.com/imysm/db-backup/internal/logger"
	"github.com/imysm/db-backup/internal/metrics"
	"github.com/imysm/db-backup/internal/model"
	"github.com/imysm/db-backup/internal/notify"
	"github.com/imysm/db-backup/internal/retention"
	"github.com/imysm/db-backup/internal/storage"
	"github.com/imysm/db-backup/internal/ws"
	"github.com/robfig/cron/v3"
)

// Scheduler 任务调度器
type Scheduler struct {
	cfg       *model.Config
	cron      *cron.Cron
	executors map[string]executor.Executor
	storages  map[string]storage.Storage
	notifier  *notify.Notifier
	mu        sync.Mutex
	running   map[string]bool // 正在运行的任务
	sem       chan struct{}   // 并发限制信号量
	entryMap  map[string]cron.EntryID
	hub       *ws.Hub // WebSocket Hub（P0功能）
}

// NewScheduler 创建调度器
func NewScheduler(cfg *model.Config) (*Scheduler, error) {
	// 初始化 cron（支持秒级）
	location, err := time.LoadLocation(cfg.Global.DefaultTZ)
	if err != nil {
		location = time.FixedZone("CST", 8*3600)
	}

	c := cron.New(
		cron.WithSeconds(),
		cron.WithLocation(location),
		cron.WithChain(
			cron.Recover(cron.DefaultLogger),
		),
	)

	s := &Scheduler{
		cfg:       cfg,
		cron:      c,
		executors: make(map[string]executor.Executor),
		storages:  make(map[string]storage.Storage),
		running:   make(map[string]bool),
		sem:       make(chan struct{}, cfg.Global.MaxConcurrent),
		entryMap:  make(map[string]cron.EntryID),
		notifier:  notify.NewNotifier(),
		hub:       ws.NewHub(),
	}

	// 启动 WebSocket Hub
	go s.hub.Run()

	// 初始化执行器和存储
	for _, task := range cfg.Tasks {
		if !task.Enabled {
			continue
		}

		exec, err := executor.NewExecutor(task.Database.Type)
		if err != nil {
			return nil, fmt.Errorf("创建执行器失败 [%s]: %w", task.ID, err)
		}
		s.executors[task.ID] = exec

		store, err := storage.NewStorage(task.Storage)
		if err != nil {
			return nil, fmt.Errorf("创建存储失败 [%s]: %w", task.ID, err)
		}
		s.storages[task.ID] = store
	}

	return s, nil
}

// GetHub 获取 WebSocket Hub
func (s *Scheduler) GetHub() *ws.Hub {
	return s.hub
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	// 注册定时任务
	for _, task := range s.cfg.Tasks {
		if !task.Enabled || task.Schedule.Cron == "" {
			continue
		}

		taskCopy := task
		taskID := task.ID

		entryID, err := s.cron.AddFunc(task.Schedule.Cron, func() {
			s.runTask(taskID, &taskCopy, nil)
		})
		if err != nil {
			return fmt.Errorf("添加定时任务失败 [%s]: %w", task.ID, err)
		}

		s.entryMap[task.ID] = entryID
		logger.Info.Printf("已调度任务: %s (%s) cron=%s\n", task.ID, task.Name, task.Schedule.Cron)
	}

	s.cron.Start()
	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.cron.Stop()
	s.hub.Stop()
}

// ReloadTask 重载任务调度
func (s *Scheduler) ReloadTask(task *model.BackupTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 移除旧任务
	if entryID, exists := s.entryMap[task.ID]; exists {
		s.cron.Remove(entryID)
		delete(s.entryMap, task.ID)
	}

	// 如果任务已禁用或没有 cron 表达式，不重新添加
	if !task.Enabled || task.Schedule.Cron == "" {
		return nil
	}

	// 添加新任务
	taskCopy := *task
	taskID := task.ID

	entryID, err := s.cron.AddFunc(task.Schedule.Cron, func() {
		s.runTask(taskID, &taskCopy, nil)
	})
	if err != nil {
		return fmt.Errorf("重新调度任务失败: %w", err)
	}

	s.entryMap[task.ID] = entryID
	logger.Info.Printf("已重载任务: %s (%s) cron=%s\n", task.ID, task.Name, task.Schedule.Cron)

	return nil
}

// RunNow 立即执行指定任务
// RunNow 立即执行任务，返回一个 channel 在任务完成时关闭
func (s *Scheduler) RunNow(taskID string, writer model.LogWriter) (<-chan struct{}, error) {
	// 查找任务配置
	var task *model.BackupTask
	for i := range s.cfg.Tasks {
		if s.cfg.Tasks[i].ID == taskID {
			task = &s.cfg.Tasks[i]
			break
		}
	}

	if task == nil {
		return nil, fmt.Errorf("任务不存在: %s", taskID)
	}

	done := make(chan struct{})
	// 异步执行，确保 panic 时 done channel 仍被关闭
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error.Printf("RunNow task %s panic: %v\n", taskID, r)
			}
			close(done)
		}()
		s.runTask(taskID, task, writer)
	}()

	return done, nil
}

// runTask 执行备份任务
func (s *Scheduler) runTask(taskID string, task *model.BackupTask, writer model.LogWriter) {
	s.mu.Lock()
	if s.running[taskID] {
		s.mu.Unlock()
		if writer != nil {
			writer.WriteString("[WARN] 任务正在执行中，跳过本次调度\n")
		}
		return
	}
	s.running[taskID] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.running, taskID)
		s.mu.Unlock()
		<-s.sem // 释放信号量
	}()

	// 获取并发信号量
	select {
	case s.sem <- struct{}{}:
	default:
		if writer != nil {
			writer.WriteString("[WARN] 已达到最大并发数，跳过本次调度\n")
		}
		return
	}

	// 更新活跃备份数指标
	metrics.ActiveJobs.Inc()
	defer metrics.ActiveJobs.Dec()

	// 创建 WebSocket 日志写入器
	wsWriter := ws.NewWebSocketLogWriter(s.hub, taskID)

	// 如果提供了额外的 writer，使用多路写入器
	var logWriter model.LogWriter = wsWriter
	if writer != nil {
		logWriter = ws.NewMultiLogWriter(wsWriter, writer)
	}

	if logWriter != nil {
		logWriter.WriteString(fmt.Sprintf("=== 开始备份任务: %s (%s) ===\n", taskID, task.Name))
	}

	// 创建带超时的 context
	timeout := s.cfg.Global.Timeout
	if timeout == 0 {
		timeout = 2 * time.Hour
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 获取执行器
	exec, ok := s.executors[taskID]
	if !ok {
		err := fmt.Errorf("执行器不存在")
		s.notifyFailure(task, err, logWriter)
		metrics.RecordBackup(taskID, string(task.Database.Type), "failure", 0, 0)
		return
	}

	// 执行备份
	result, err := exec.Backup(ctx, task, logWriter)
	if err != nil {
		if logWriter != nil {
			logWriter.WriteString(fmt.Sprintf("=== 备份失败: %v ===\n", err))
		}
		s.notifyFailure(task, err, logWriter)
		metrics.RecordBackup(taskID, string(task.Database.Type), "failure", 0, 0)
		return
	}

	if logWriter != nil {
		logWriter.WriteString(fmt.Sprintf("=== 备份成功: %s (%s) ===\n", result.FilePath, formatDuration(result.Duration)))
	}

	// 记录指标（P0功能）
	metrics.RecordBackup(taskID, string(task.Database.Type), "success", result.Duration.Seconds(), result.FileSize)

	// 应用保留策略
	if err := s.applyRetentionPolicy(ctx, taskID, task, logWriter); err != nil {
		if logWriter != nil {
			logWriter.WriteString(fmt.Sprintf("[WARN] 保留策略执行失败: %v\n", err))
		}
	}

	// 更新存储使用量指标
	if store, ok := s.storages[taskID]; ok {
		if usage, err := store.GetUsage(ctx, taskID); err == nil {
			metrics.UpdateStorageUsage(task.Storage.Type, usage)
		}
	}

	// 发送成功通知
	s.notifySuccess(task, result, logWriter)
}

// applyRetentionPolicy 应用保留策略
func (s *Scheduler) applyRetentionPolicy(ctx context.Context, taskID string, task *model.BackupTask, writer model.LogWriter) error {
	if task.Retention.KeepLast == 0 && task.Retention.KeepDays == 0 &&
		task.Retention.KeepWeekly == 0 && task.Retention.KeepMonthly == 0 {
		return nil
	}

	store, ok := s.storages[taskID]
	if !ok {
		return fmt.Errorf("存储不存在")
	}

	policy := retention.NewPolicy(task.Retention)

	success, failed, err := policy.Cleanup(ctx, store, taskID)
	if err != nil {
		return err
	}

	if writer != nil && (success > 0 || failed > 0) {
		writer.WriteString(fmt.Sprintf("[清理] 删除 %d 个过期备份，失败 %d 个\n", success, failed))
	}

	return nil
}

// notifySuccess 发送成功通知
func (s *Scheduler) notifySuccess(task *model.BackupTask, result *model.BackupResult, writer model.LogWriter) {
	if !task.Notify.Enabled {
		return
	}

	// 使用 NotifyConfig 而不是直接用 NotifyConfig
	cfg := model.NotifyConfig{
		Enabled:  task.Notify.Enabled,
		Type:     task.Notify.Type,
		Endpoint: task.Notify.Endpoint,
	}

	if err := s.notifier.NotifySuccess(context.Background(), cfg, task.Name, result); err != nil {
		if writer != nil {
			writer.WriteString(fmt.Sprintf("[WARN] 发送通知失败: %v\n", err))
		}
	}
}

// notifyFailure 发送失败通知
func (s *Scheduler) notifyFailure(task *model.BackupTask, err error, writer model.LogWriter) {
	if !task.Notify.Enabled {
		return
	}

	cfg := model.NotifyConfig{
		Enabled:  task.Notify.Enabled,
		Type:     task.Notify.Type,
		Endpoint: task.Notify.Endpoint,
	}

	if notifyErr := s.notifier.NotifyFailure(context.Background(), cfg, task.Name, err); notifyErr != nil {
		if writer != nil {
			writer.WriteString(fmt.Sprintf("[WARN] 发送通知失败: %v\n", notifyErr))
		}
	}
}

// ValidateAll 验证所有任务的数据库连接
func (s *Scheduler) ValidateAll(ctx context.Context) map[string]error {
	results := make(map[string]error)
	for _, task := range s.cfg.Tasks {
		exec, ok := s.executors[task.ID]
		if !ok {
			results[task.ID] = fmt.Errorf("执行器不存在")
			continue
		}
		results[task.ID] = exec.Validate(ctx, &task.Database)
	}
	return results
}

// IsRunning 检查任务是否正在运行
func (s *Scheduler) IsRunning(taskID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running[taskID]
}

// GetRunningTasks 获取正在运行的任务列表
func (s *Scheduler) GetRunningTasks() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var tasks []string
	for taskID := range s.running {
		tasks = append(tasks, taskID)
	}
	return tasks
}

// 辅助函数

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1f秒", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1f分钟", d.Minutes())
	}
	return fmt.Sprintf("%.1f小时", d.Hours())
}
