// Package scheduler 测试
package scheduler

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/imysm/db-backup/internal/model"
)

func TestNewScheduler(t *testing.T) {
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
					Path: "/tmp/backup",
				},
			},
		},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}
	if sched == nil {
		t.Fatal("NewScheduler() returned nil")
	}
}

func TestNewScheduler_InvalidDBType(t *testing.T) {
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
					Type: "invalid",
				},
			},
		},
	}

	_, err := NewScheduler(cfg)
	if err == nil {
		t.Error("NewScheduler() expected error for invalid database type")
	}
}

func TestScheduler_StartStop(t *testing.T) {
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

	// 启动
	if err := sched.Start(); err != nil {
		t.Errorf("Start() error = %v", err)
	}

	// 停止
	sched.Stop()
}

func TestScheduler_RunNow_TaskNotFound(t *testing.T) {
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

	_, err = sched.RunNow("nonexistent", nil)
	if err == nil {
		t.Error("RunNow() expected error for nonexistent task")
	}
}

func TestScheduler_IsRunning(t *testing.T) {
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

	if sched.IsRunning("test-001") {
		t.Error("IsRunning() = true, want false")
	}
}

func TestScheduler_GetRunningTasks(t *testing.T) {
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

	tasks := sched.GetRunningTasks()
	if len(tasks) != 0 {
		t.Errorf("GetRunningTasks() returned %d tasks, want 0", len(tasks))
	}
}

func TestScheduler_ValidateAll(t *testing.T) {
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
					Path: "/tmp/backup",
				},
			},
		},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	ctx := context.Background()
	results := sched.ValidateAll(ctx)

	if len(results) != 1 {
		t.Errorf("ValidateAll() returned %d results, want 1", len(results))
	}
}

func TestScheduler_ReloadTask(t *testing.T) {
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

	task := &model.BackupTask{
		ID:      "test-001",
		Name:    "Test Task",
		Enabled: false, // 禁用的任务不会添加到调度
		Database: model.DatabaseConfig{
			Type: model.MySQL,
			Host: "localhost",
			Port: 3306,
		},
	}

	err = sched.ReloadTask(task)
	if err != nil {
		t.Errorf("ReloadTask() error = %v", err)
	}
}

func TestScheduler_Start_WithTasks(t *testing.T) {
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
				Schedule: model.ScheduleConfig{
					Cron: "0 0 2 * * *",
				},
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

func TestScheduler_RunNow_Success(t *testing.T) {
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
					Path: "/tmp/backup-test",
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

	// RunNow 是异步执行的
	_, err = sched.RunNow("test-001", nil)
	if err != nil {
		t.Errorf("RunNow() error = %v", err)
	}

	// 等待任务执行
	time.Sleep(100 * time.Millisecond)
}

func TestScheduler_ReloadTask_RemoveOld(t *testing.T) {
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

	// 先添加一个任务
	task1 := &model.BackupTask{
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
	sched.ReloadTask(task1)

	// 修改任务并重新加载
	task1.Schedule.Cron = "0 0 3 * * *"
	err = sched.ReloadTask(task1)
	if err != nil {
		t.Errorf("ReloadTask() error = %v", err)
	}
}

func TestScheduler_ValidateAll_MultipleTasks(t *testing.T) {
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
				Name:    "MySQL Task",
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
			{
				ID:      "pg-001",
				Name:    "PostgreSQL Task",
				Enabled: true,
				Database: model.DatabaseConfig{
					Type: model.PostgreSQL,
					Host: "localhost",
					Port: 5432,
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

	ctx := context.Background()
	results := sched.ValidateAll(ctx)

	if len(results) != 2 {
		t.Errorf("ValidateAll() returned %d results, want 2", len(results))
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{500 * time.Millisecond, "0.5秒"},
		{30 * time.Second, "30.0秒"},
		{90 * time.Second, "1.5分钟"},
		{2 * time.Minute, "2.0分钟"},
		{90 * time.Minute, "1.5小时"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %s, want %s", tt.d, got, tt.want)
		}
	}
}

func TestNewScheduler_EmptyTasks(t *testing.T) {
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
	if sched == nil {
		t.Fatal("NewScheduler() returned nil")
	}
}

func TestNewScheduler_DisabledTask(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{
			{
				ID:      "disabled-001",
				Name:    "Disabled Task",
				Enabled: false,
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
	if sched == nil {
		t.Fatal("NewScheduler() returned nil")
	}
}

func TestScheduler_Start_EmptyCron(t *testing.T) {
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
				Schedule: model.ScheduleConfig{
					Cron: "", // Empty cron
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

func TestScheduler_ReloadTask_DisableTask(t *testing.T) {
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

	// Add and then disable task
	task := &model.BackupTask{
		ID:      "test-001",
		Name:    "Test Task",
		Enabled: false,
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
	sched.ReloadTask(task)
}

func TestScheduler_GetRunningTasks_AfterRun(t *testing.T) {
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
					Path: "/tmp/backup-test",
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

	// Start the task
	_, _ = sched.RunNow("test-001", nil)

	// Check running tasks
	time.Sleep(10 * time.Millisecond)
	tasks := sched.GetRunningTasks()
	// Task might be running or already finished
	t.Logf("Running tasks: %d", len(tasks))
}

func TestNewScheduler_InvalidTimezone(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Invalid/Timezone",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}
	// Should fallback to CST when timezone is invalid
	if sched == nil {
		t.Fatal("NewScheduler() returned nil")
	}
}

func TestScheduler_ValidateAll_EmptyTasks(t *testing.T) {
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

	ctx := context.Background()
	results := sched.ValidateAll(ctx)

	if len(results) != 0 {
		t.Errorf("ValidateAll() returned %d results, want 0", len(results))
	}
}

func TestScheduler_RunNow_WithNotify(t *testing.T) {
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
					Path: "/tmp/backup-test",
				},
				Compression: model.CompressionConfig{
					Enabled: false,
				},
				Notify: model.NotifyConfig{
					Enabled:  true,
					Type:     "webhook",
					Endpoint: "http://localhost:8080/notify",
				},
			},
		},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	// RunNow will execute but likely fail since no real MySQL
	_, err = sched.RunNow("test-001", nil)
	if err != nil {
		t.Logf("RunNow returned: %v", err)
	}
}

func TestScheduler_RunNow_WithRetention(t *testing.T) {
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
					Path: "/tmp/backup-test",
				},
				Compression: model.CompressionConfig{
					Enabled: false,
				},
				Retention: model.RetentionConfig{
					KeepLast: 5,
					KeepDays: 30,
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

	_, err = sched.RunNow("test-001", nil)
	if err != nil {
		t.Logf("RunNow returned: %v", err)
	}
}

func TestScheduler_RunNow_CompressEnabled(t *testing.T) {
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
					Path: "/tmp/backup-test",
				},
				Compression: model.CompressionConfig{
					Enabled: true,
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

	_, err = sched.RunNow("test-001", nil)
	if err != nil {
		t.Logf("RunNow returned: %v", err)
	}
}

func TestScheduler_RunNow_PostgreSQL(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{
			{
				ID:      "pg-001",
				Name:    "PostgreSQL Task",
				Enabled: true,
				Database: model.DatabaseConfig{
					Type:     model.PostgreSQL,
					Host:     "localhost",
					Port:     5432,
					Username: "postgres",
					Password: "password",
					Database: "test",
				},
				Storage: model.StorageConfig{
					Type: "local",
					Path: "/tmp/backup-test",
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

	_, err = sched.RunNow("pg-001", nil)
	if err != nil {
		t.Logf("RunNow returned: %v", err)
	}
}

func TestScheduler_RunNow_InvalidDBType(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{
			{
				ID:      "invalid-001",
				Name:    "Invalid Task",
				Enabled: true,
				Database: model.DatabaseConfig{
					Type: "invalid_db_type",
					Host: "localhost",
					Port: 1234,
				},
				Storage: model.StorageConfig{
					Type: "local",
					Path: "/tmp/backup-test",
				},
				Notify: model.NotifyConfig{
					Enabled: false,
				},
			},
		},
	}

	_, err := NewScheduler(cfg)
	if err == nil {
		t.Error("NewScheduler should return error for invalid database type")
	}
}

func TestScheduler_RunNow_WithNotifySuccess(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{
			{
				ID:      "notify-001",
				Name:    "Notify Test Task",
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
					Path: "/tmp/backup-test",
				},
				Notify: model.NotifyConfig{
					Enabled:  true,
					Type:     "webhook",
					Endpoint: "http://localhost:8080/notify",
				},
				Retention: model.RetentionConfig{
					KeepLast: 5,
					KeepDays: 30,
				},
			},
		},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	// RunNow will trigger notifySuccess on completion
	_, err = sched.RunNow("notify-001", nil)
	if err != nil {
		t.Logf("RunNow returned: %v", err)
	}
}

func TestScheduler_RunNow_WithRetentionCleanup(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{
			{
				ID:      "retention-001",
				Name:    "Retention Test Task",
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
					Path: "/tmp/backup-test",
				},
				Notify: model.NotifyConfig{
					Enabled: false,
				},
				Retention: model.RetentionConfig{
					KeepLast:    3,
					KeepDays:    7,
					KeepWeekly:  2,
					KeepMonthly: 1,
				},
			},
		},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	// RunNow will trigger applyRetentionPolicy on completion
	_, err = sched.RunNow("retention-001", nil)
	if err != nil {
		t.Logf("RunNow returned: %v", err)
	}
}

// testLogWriter is a mock log writer for testing
type testLogWriter struct {
	buf *bytes.Buffer
}

func (w *testLogWriter) Write(p []byte) (n int, err error) {
	return w.buf.Write(p)
}

func (w *testLogWriter) WriteString(s string) (n int, err error) {
	return w.buf.WriteString(s)
}

func (w *testLogWriter) Close() error {
	return nil
}

func TestScheduler_RunNow_WithFailure(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{
			{
				ID:      "fail-001",
				Name:    "Fail Test Task",
				Enabled: true,
				Database: model.DatabaseConfig{
					Type:     model.MySQL,
					Host:     "nonexistent-host",
					Port:     3306,
					Username: "root",
					Password: "password",
					Database: "test",
				},
				Storage: model.StorageConfig{
					Type: "local",
					Path: "/tmp/backup-test",
				},
				Notify: model.NotifyConfig{
					Enabled:  true,
					Type:     "webhook",
					Endpoint: "http://localhost:8080/notify",
				},
			},
		},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	// RunNow will fail because host doesn't exist
	_, err = sched.RunNow("fail-001", nil)
	if err != nil {
		t.Logf("RunNow failed as expected: %v", err)
	}
}

func TestScheduler_ValidateAll_WithTasks(t *testing.T) {
	cfg := &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       time.Hour,
		},
		Tasks: []model.BackupTask{
			{
				ID:      "validate-001",
				Name:    "Validate Test Task",
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
					Path: "/tmp/backup-test",
				},
			},
		},
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	ctx := context.Background()
	results := sched.ValidateAll(ctx)

	if len(results) != 1 {
		t.Errorf("ValidateAll returned %d results, want 1", len(results))
	}
}
