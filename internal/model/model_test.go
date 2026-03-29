// Package model 测试
package model

import (
	"testing"
	"time"
)

func TestDatabaseType(t *testing.T) {
	tests := []struct {
		name string
		dt   DatabaseType
		want string
	}{
		{"MySQL", MySQL, "mysql"},
		{"PostgreSQL", PostgreSQL, "postgres"},
		{"SQLServer", SQLServer, "sqlserver"},
		{"Oracle", Oracle, "oracle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.dt) != tt.want {
				t.Errorf("DatabaseType = %s, want %s", tt.dt, tt.want)
			}
		})
	}
}

func TestTaskStatus(t *testing.T) {
	tests := []struct {
		name string
		ts   TaskStatus
		want string
	}{
		{"Pending", TaskStatusPending, "pending"},
		{"Running", TaskStatusRunning, "running"},
		{"Success", TaskStatusSuccess, "success"},
		{"Failed", TaskStatusFailed, "failed"},
		{"Verified", TaskStatusVerified, "verified"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.ts) != tt.want {
				t.Errorf("TaskStatus = %s, want %s", tt.ts, tt.want)
			}
		})
	}
}

func TestBackupTask_TableName(t *testing.T) {
	task := BackupTask{}
	if task.TableName() != "backup_tasks" {
		t.Errorf("TableName = %s, want backup_tasks", task.TableName())
	}
}

func TestBackupRecord_TableName(t *testing.T) {
	record := BackupRecord{}
	if record.TableName() != "backup_records" {
		t.Errorf("TableName = %s, want backup_records", record.TableName())
	}
}

func TestBackupTask_Fields(t *testing.T) {
	now := time.Now()
	task := BackupTask{
		ID:        "test-001",
		Name:      "Test Task",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if task.ID != "test-001" {
		t.Errorf("ID = %s, want test-001", task.ID)
	}
	if task.Name != "Test Task" {
		t.Errorf("Name = %s, want Test Task", task.Name)
	}
	if !task.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestBackupRecord_Fields(t *testing.T) {
	now := time.Now()
	record := BackupRecord{
		ID:         1,
		TaskID:     "test-001",
		TraceID:    "trace-123",
		Status:     TaskStatusSuccess,
		StartTime:  &now,
		EndTime:    &now,
		DurationMs: 1000,
		FilePath:   "/tmp/backup.sql",
		FileSize:   1024,
		Checksum:   "abc123",
	}

	if record.ID != 1 {
		t.Errorf("ID = %d, want 1", record.ID)
	}
	if record.TaskID != "test-001" {
		t.Errorf("TaskID = %s, want test-001", record.TaskID)
	}
	if record.Status != TaskStatusSuccess {
		t.Errorf("Status = %s, want success", record.Status)
	}
}

func TestBackupResult(t *testing.T) {
	start := time.Now()
	result := BackupResult{
		TaskID:    "test-001",
		TraceID:   "trace-123",
		StartTime: start,
		EndTime:   start.Add(time.Second),
		Duration:  time.Second,
		Status:    TaskStatusSuccess,
		FilePath:  "/tmp/backup.sql",
		FileSize:  1024,
	}

	if result.Duration != time.Second {
		t.Errorf("Duration = %v, want 1s", result.Duration)
	}
}

func TestNilWriter(t *testing.T) {
	writer := &NilWriter{}

	n, err := writer.Write([]byte("test"))
	if err != nil {
		t.Errorf("Write error: %v", err)
	}
	if n != 4 {
		t.Errorf("Write returned %d, want 4", n)
	}

	n, err = writer.WriteString("test")
	if err != nil {
		t.Errorf("WriteString error: %v", err)
	}
	if n != 4 {
		t.Errorf("WriteString returned %d, want 4", n)
	}

	if err := writer.Close(); err != nil {
		t.Errorf("Close error: %v", err)
	}
}

func TestGlobalConfig(t *testing.T) {
	cfg := GlobalConfig{
		WorkDir:       "/tmp/backup",
		DefaultTZ:     "Asia/Shanghai",
		MaxConcurrent: 5,
		Timeout:       time.Hour,
	}

	if cfg.WorkDir != "/tmp/backup" {
		t.Errorf("WorkDir = %s, want /tmp/backup", cfg.WorkDir)
	}
	if cfg.MaxConcurrent != 5 {
		t.Errorf("MaxConcurrent = %d, want 5", cfg.MaxConcurrent)
	}
}

func TestDatabaseConfig(t *testing.T) {
	cfg := DatabaseConfig{
		Type:     MySQL,
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "password",
		Database: "test",
		Params: map[string]string{
			"timeout": "30s",
		},
	}

	if cfg.Type != MySQL {
		t.Errorf("Type = %s, want mysql", cfg.Type)
	}
	if cfg.Port != 3306 {
		t.Errorf("Port = %d, want 3306", cfg.Port)
	}
	if cfg.Params["timeout"] != "30s" {
		t.Errorf("Params[timeout] = %s, want 30s", cfg.Params["timeout"])
	}
}

func TestScheduleConfig(t *testing.T) {
	cfg := ScheduleConfig{
		Cron:     "0 2 * * *",
		Timezone: "Asia/Shanghai",
	}

	if cfg.Cron != "0 2 * * *" {
		t.Errorf("Cron = %s, want '0 2 * * *'", cfg.Cron)
	}
}

func TestStorageConfig(t *testing.T) {
	cfg := StorageConfig{
		Type:     "local",
		Path:     "/tmp/backup",
		Endpoint: "",
		Bucket:   "",
	}

	if cfg.Type != "local" {
		t.Errorf("Type = %s, want local", cfg.Type)
	}
}

func TestRetentionConfig(t *testing.T) {
	cfg := RetentionConfig{
		KeepLast:    7,
		KeepDays:    30,
		KeepWeekly:  4,
		KeepMonthly: 3,
	}

	if cfg.KeepLast != 7 {
		t.Errorf("KeepLast = %d, want 7", cfg.KeepLast)
	}
}

func TestCompressionConfig(t *testing.T) {
	cfg := CompressionConfig{
		Enabled: true,
		Type:    "gzip",
		Level:   6,
	}

	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}
	if cfg.Type != "gzip" {
		t.Errorf("Type = %s, want gzip", cfg.Type)
	}
}

func TestNotifyConfig(t *testing.T) {
	cfg := NotifyConfig{
		Enabled:   true,
		Type:      "dingtalk",
		Endpoint:  "https://example.com/webhook",
		Receivers: []string{"user1", "user2"},
	}

	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}
	if len(cfg.Receivers) != 2 {
		t.Errorf("Receivers length = %d, want 2", len(cfg.Receivers))
	}
}
