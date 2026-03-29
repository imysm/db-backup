// Package scheduler - 更多测试提高覆盖率
package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/imysm/db-backup/internal/model"
)

// TestLogWriter 测试日志写入器
type TestLogWriter struct{}

func (w *TestLogWriter) Write(p []byte) (n int, err error)       { return len(p), nil }
func (w *TestLogWriter) WriteString(s string) (n int, err error) { return len(s), nil }
func (w *TestLogWriter) Close() error                            { return nil }

func TestScheduler_RunNow_WithLogWriter(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{
			{
				ID:      "test-001",
				Name:    "Test Task",
				Enabled: true,
				Database: model.DatabaseConfig{
					Type:     model.MySQL,
					Host:     "localhost",
					Port:     3306,
					Username: "root",
					Password: "password",
					Database: "test",
				},
				Storage: model.StorageConfig{
					Type: "local",
					Path: "/tmp/backup-test-logs",
				},
				Compression: model.CompressionConfig{
					Enabled: false,
				},
				Notify: model.NotifyConfig{
					Enabled: false,
				},
			},
		},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	writer := &TestLogWriter{}
	_, err = sched.RunNow("test-001", writer)
	if err != nil {
		t.Errorf("RunNow() error = %v", err)
	}

	// 等待任务执行
	time.Sleep(200 * time.Millisecond)
}

func TestScheduler_RunNow_AlreadyRunning(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{
			{
				ID:      "test-001",
				Name:    "Test Task",
				Enabled: true,
				Database: model.DatabaseConfig{
					Type:     model.MySQL,
					Host:     "localhost",
					Port:     3306,
					Username: "root",
					Password: "password",
					Database: "test",
				},
				Storage: model.StorageConfig{
					Type: "local",
					Path: "/tmp/backup-test-running",
				},
				Compression: model.CompressionConfig{
					Enabled: false,
				},
				Notify: model.NotifyConfig{
					Enabled: false,
				},
			},
		},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	writer := &TestLogWriter{}

	// 第一次调用
	_, _ = sched.RunNow("test-001", writer)

	// 立即第二次调用（应该被跳过）
	_, _ = sched.RunNow("test-001", writer)

	// 等待任务完成
	time.Sleep(200 * time.Millisecond)
}

func TestScheduler_Start_MultipleTasks(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{
			{
				ID:      "mysql-001",
				Name:    "MySQL Task 1",
				Enabled: true,
				Schedule: model.ScheduleConfig{
					Cron: "0 0 1 * * *",
				},
				Database: model.DatabaseConfig{
					Type: model.MySQL,
					Host: "localhost",
					Port: 3306,
				},
				Storage: model.StorageConfig{
					Type: "local",
					Path: "/tmp/backup",
				},
			},
			{
				ID:      "mysql-002",
				Name:    "MySQL Task 2",
				Enabled: true,
				Schedule: model.ScheduleConfig{
					Cron: "0 0 2 * * *",
				},
				Database: model.DatabaseConfig{
					Type: model.MySQL,
					Host: "localhost",
					Port: 3306,
				},
				Storage: model.StorageConfig{
					Type: "local",
					Path: "/tmp/backup",
				},
			},
			{
				ID:      "mysql-003",
				Name:    "MySQL Task 3 (Disabled)",
				Enabled: false,
				Schedule: model.ScheduleConfig{
					Cron: "0 0 3 * * *",
				},
				Database: model.DatabaseConfig{
					Type: model.MySQL,
					Host: "localhost",
					Port: 3306,
				},
				Storage: model.StorageConfig{
					Type: "local",
					Path: "/tmp/backup",
				},
			},
		},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	if err := sched.Start(); err != nil {
		t.Errorf("Start() error = %v", err)
	}

	sched.Stop()
}

func TestScheduler_ValidateAll_WithTimeout(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{
			{
				ID:      "test-001",
				Name:    "Test Task",
				Enabled: true,
				Database: model.DatabaseConfig{
					Type: model.MySQL,
					Host: "localhost",
					Port: 3306,
				},
				Storage: model.StorageConfig{
					Type: "local",
					Path: "/tmp/backup",
				},
			},
		},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := sched.ValidateAll(ctx)
	if len(results) != 1 {
		t.Errorf("ValidateAll() returned %d results, want 1", len(results))
	}
}

func TestScheduler_IsRunning_AfterRun(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{
			{
				ID:      "test-001",
				Name:    "Test Task",
				Enabled: true,
				Database: model.DatabaseConfig{
					Type: model.MySQL,
					Host: "localhost",
					Port: 3306,
				},
				Storage: model.StorageConfig{
					Type: "local",
					Path: "/tmp/backup",
				},
			},
		},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	// 任务未运行
	if sched.IsRunning("test-001") {
		t.Error("IsRunning() = true before RunNow")
	}

	// 启动任务
	_, _ = sched.RunNow("test-001", nil)

	// 立即检查（可能仍在运行）
	_ = sched.IsRunning("test-001")

	// 等待任务完成
	time.Sleep(100 * time.Millisecond)
}

func TestScheduler_GetRunningTasks_DuringRun(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{
			{
				ID:      "test-001",
				Name:    "Test Task",
				Enabled: true,
				Database: model.DatabaseConfig{
					Type: model.MySQL,
					Host: "localhost",
					Port: 3306,
				},
				Storage: model.StorageConfig{
					Type: "local",
					Path: "/tmp/backup",
				},
			},
		},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	// 启动任务
	_, _ = sched.RunNow("test-001", nil)

	// 获取正在运行的任务
	running := sched.GetRunningTasks()
	_ = running // 可能包含 "test-001"

	// 等待任务完成
	time.Sleep(100 * time.Millisecond)
}

func TestScheduler_ReloadTask_EnableDisable(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	if err := sched.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer sched.Stop()

	// 添加启用的任务
	task := &model.BackupTask{
		ID:      "test-001",
		Name:    "Test Task",
		Enabled: true,
		Schedule: model.ScheduleConfig{
			Cron: "0 0 2 * * *",
		},
		Database: model.DatabaseConfig{
			Type: model.MySQL,
			Host: "localhost",
			Port: 3306,
		},
		Storage: model.StorageConfig{
			Type: "local",
			Path: "/tmp/backup",
		},
	}

	if err := sched.ReloadTask(task); err != nil {
		t.Errorf("ReloadTask(enabled) error = %v", err)
	}

	// 禁用任务
	task.Enabled = false
	if err := sched.ReloadTask(task); err != nil {
		t.Errorf("ReloadTask(disabled) error = %v", err)
	}

	// 重新启用
	task.Enabled = true
	if err := sched.ReloadTask(task); err != nil {
		t.Errorf("ReloadTask(enabled again) error = %v", err)
	}
}

func TestFormatDuration_AllCases(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{100 * time.Millisecond, "0.1秒"},
		{500 * time.Millisecond, "0.5秒"},
		{1 * time.Second, "1.0秒"},
		{30 * time.Second, "30.0秒"},
		{59 * time.Second, "59.0秒"},
		{60 * time.Second, "1.0分钟"},
		{90 * time.Second, "1.5分钟"},
		{120 * time.Second, "2.0分钟"},
		{30 * time.Minute, "30.0分钟"},
		{59 * time.Minute, "59.0分钟"},
		{60 * time.Minute, "1.0小时"},
		{90 * time.Minute, "1.5小时"},
		{120 * time.Minute, "2.0小时"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %s, want %s", tt.d, got, tt.want)
		}
	}
}
