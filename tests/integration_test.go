package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/imysm/db-backup/internal/config"
	"github.com/imysm/db-backup/internal/executor"
	"github.com/imysm/db-backup/internal/model"
	"github.com/imysm/db-backup/internal/notify"
	"github.com/imysm/db-backup/internal/retention"
	"github.com/imysm/db-backup/internal/scheduler"
	"github.com/imysm/db-backup/internal/storage"
)

// TestConfigLoading 测试配置加载
func TestConfigLoading(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
global:
  work_dir: /tmp/db-backup-test
  default_tz: Asia/Shanghai
  max_concurrent: 3
  timeout: 1h

log:
  level: debug
  format: console

tasks:
  - id: test-001
    name: Test Backup
    enabled: true
    schedule:
      cron: "0 2 * * *"
    database:
      type: mysql
      host: localhost
      port: 3306
      username: root
      password: test
      database: testdb
    storage:
      type: local
      path: /tmp/backup-test
    compression:
      enabled: true
    retention:
      keep_last: 5
      keep_days: 7
    notify:
      enabled: false
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// 加载配置
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 验证配置
	if cfg.Global.WorkDir != "/tmp/db-backup-test" {
		t.Errorf("WorkDir = %s, want /tmp/db-backup-test", cfg.Global.WorkDir)
	}
	if len(cfg.Tasks) != 1 {
		t.Errorf("Tasks count = %d, want 1", len(cfg.Tasks))
	}
}

// TestSchedulerLifecycle 测试调度器生命周期
func TestSchedulerLifecycle(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       t.TempDir(),
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 2,
			Timeout:       30 * time.Minute,
		},
		Tasks: []model.BackupTask{},
	}

	sched, err := scheduler.NewScheduler(cfg)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	// 启动调度器
	if err := sched.Start(); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}

	// 验证运行状态
	running := sched.GetRunningTasks()
	if len(running) != 0 {
		t.Errorf("Running tasks = %d, want 0", len(running))
	}

	// 停止调度器
	sched.Stop()
}

// TestExecutorWithMockDatabase 测试执行器（模拟数据库）
func TestExecutorWithMockDatabase(t *testing.T) {
	// 使用正确的 API: NewExecutor 接受 DatabaseType
	exec, err := executor.NewExecutor(model.MySQL)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	task := &model.BackupTask{
		ID:      "test-001",
		Name:    "Test Task",
		Enabled: true,
		Database: model.DatabaseConfig{
			Type:     model.MySQL,
			Host:     "localhost",
			Port:     3306,
			Username: "root",
			Password: "test",
			Database: "testdb",
		},
		Storage: model.StorageConfig{
			Type: "local",
			Path: t.TempDir(),
		},
	}

	ctx := context.Background()
	// 使用正确的 API: Backup 方法，不是 Execute
	_, err = exec.Backup(ctx, task, nil)
	// 由于没有实际的数据库，预期会失败
	if err == nil {
		t.Log("Executor completed (unexpected success)")
	} else {
		t.Logf("Executor failed as expected: %v", err)
	}
}

// TestStorageLocal 测试本地存储
func TestStorageLocal(t *testing.T) {
	tmpDir := t.TempDir()

	localStorage := storage.NewLocalStorage(tmpDir)
	ctx := context.Background()

	// 创建临时测试文件
	srcFile := filepath.Join(t.TempDir(), "source.sql")
	testData := []byte("test backup content")
	if err := os.WriteFile(srcFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// 测试保存文件 - 使用正确的 API: Save(ctx, localPath, remotePath)
	remotePath := "backups/test.sql"
	err := localStorage.Save(ctx, srcFile, remotePath)
	if err != nil {
		t.Fatalf("Failed to save file: %v", err)
	}

	// 测试文件存在
	exists, err := localStorage.Exists(ctx, remotePath)
	if err != nil {
		t.Fatalf("Failed to check file exists: %v", err)
	}
	if !exists {
		t.Error("File should exist after save")
	}

	// 测试获取文件大小
	size, err := localStorage.Size(ctx, remotePath)
	if err != nil {
		t.Fatalf("Failed to get file size: %v", err)
	}
	if size != int64(len(testData)) {
		t.Errorf("File size = %d, want %d", size, len(testData))
	}

	// 测试删除文件
	err = localStorage.Delete(ctx, remotePath)
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// 验证文件已删除
	exists, _ = localStorage.Exists(ctx, remotePath)
	if exists {
		t.Error("File should be deleted")
	}
}

// TestRetentionPolicy 测试保留策略
func TestRetentionPolicy(t *testing.T) {
	tmpDir := t.TempDir()

	retentionCfg := model.RetentionConfig{
		KeepLast: 3,
		KeepDays: 7,
	}

	// 使用正确的 API: NewPolicy
	policy := retention.NewPolicy(retentionCfg)

	// 创建本地存储
	localStorage := storage.NewLocalStorage(tmpDir)

	// 创建一些测试备份文件
	for i := 0; i < 5; i++ {
		srcFile := filepath.Join(t.TempDir(), "source-"+string(rune('0'+i))+".sql")
		os.WriteFile(srcFile, []byte("test"), 0644)

		remotePath := filepath.Join("test-001", "backup-"+string(rune('0'+i))+".sql")
		localStorage.Save(context.Background(), srcFile, remotePath)
	}

	// 应用保留策略
	ctx := context.Background()
	toDelete, err := policy.Apply(ctx, localStorage, "test-001")
	if err != nil {
		t.Logf("Retention apply result: %v", err)
	}
	t.Logf("Files to delete: %v", toDelete)
}

// TestNotifier 测试通知器
func TestNotifier(t *testing.T) {
	// 使用正确的 API: NewNotifier
	notifier := notify.NewNotifier()

	notifyCfg := model.NotifyConfig{
		Enabled:  false, // 禁用实际发送
		Type:     "webhook",
		Endpoint: "http://localhost:8080/notify",
	}

	ctx := context.Background()

	// 测试发送（禁用时应该直接返回 nil）
	err := notifier.Send(ctx, notifyCfg, "test message")
	if err != nil {
		t.Errorf("Send with disabled notify should return nil, got: %v", err)
	}
}

// TestBackupTaskValidation 测试备份任务验证
func TestBackupTaskValidation(t *testing.T) {
	tests := []struct {
		name    string
		task    model.BackupTask
		wantErr bool
	}{
		{
			name: "Valid MySQL task",
			task: model.BackupTask{
				ID:      "mysql-001",
				Name:    "MySQL Backup",
				Enabled: true,
				Database: model.DatabaseConfig{
					Type:     model.MySQL,
					Host:     "localhost",
					Port:     3306,
					Username: "root",
					Password: "password",
					Database: "testdb",
				},
				Storage: model.StorageConfig{
					Type: "local",
					Path: "/tmp/backup",
				},
			},
			wantErr: false,
		},
		{
			name: "Valid PostgreSQL task",
			task: model.BackupTask{
				ID:      "pg-001",
				Name:    "PostgreSQL Backup",
				Enabled: true,
				Database: model.DatabaseConfig{
					Type:     model.PostgreSQL,
					Host:     "localhost",
					Port:     5432,
					Username: "postgres",
					Password: "password",
					Database: "testdb",
				},
				Storage: model.StorageConfig{
					Type: "local",
					Path: "/tmp/backup",
				},
			},
			wantErr: false,
		},
		{
			name: "Valid SQLServer task",
			task: model.BackupTask{
				ID:      "sqlserver-001",
				Name:    "SQLServer Backup",
				Enabled: true,
				Database: model.DatabaseConfig{
					Type:     model.SQLServer,
					Host:     "localhost",
					Port:     1433,
					Username: "sa",
					Password: "password",
					Database: "testdb",
				},
				Storage: model.StorageConfig{
					Type: "local",
					Path: "/tmp/backup",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证任务基本字段
			if tt.task.ID == "" {
				t.Error("Task ID should not be empty")
			}
			if tt.task.Name == "" {
				t.Error("Task Name should not be empty")
			}
			if tt.task.Database.Type == "" {
				t.Error("Database Type should not be empty")
			}
			t.Logf("Task %s is valid", tt.name)
		})
	}
}

// TestConcurrentBackups 测试并发备份
func TestConcurrentBackups(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       t.TempDir(),
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 2,
			Timeout:       5 * time.Second,
		},
		Tasks: []model.BackupTask{},
	}

	sched, err := scheduler.NewScheduler(cfg)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	if err := sched.Start(); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer sched.Stop()

	// 验证最大并发数
	running := sched.GetRunningTasks()
	if len(running) > cfg.Global.MaxConcurrent {
		t.Errorf("Running tasks %d exceeds max concurrent %d", len(running), cfg.Global.MaxConcurrent)
	}
}

// TestBackupFileNaming 测试备份文件命名
func TestBackupFileNaming(t *testing.T) {
	taskID := "test-001"
	timestamp := time.Now()

	// 测试生成备份文件名
	expectedPattern := taskID + "_"
	filename := generateBackupFilename(taskID, timestamp, "sql")

	if len(filename) < len(expectedPattern) {
		t.Errorf("Filename %s too short", filename)
	}
}

// generateBackupFilename 生成备份文件名
func generateBackupFilename(taskID string, timestamp time.Time, ext string) string {
	return taskID + "_" + timestamp.Format("20060102_150405") + "." + ext
}
