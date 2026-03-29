// Package config 测试
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/imysm/db-backup/internal/model"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Global.WorkDir != "/tmp/db-backup" {
		t.Errorf("WorkDir = %s, want /tmp/db-backup", cfg.Global.WorkDir)
	}
	if cfg.Global.DefaultTZ != "Asia/Shanghai" {
		t.Errorf("DefaultTZ = %s, want Asia/Shanghai", cfg.Global.DefaultTZ)
	}
	if cfg.Global.MaxConcurrent != 5 {
		t.Errorf("MaxConcurrent = %d, want 5", cfg.Global.MaxConcurrent)
	}
	if cfg.Global.Timeout != 2*time.Hour {
		t.Errorf("Timeout = %v, want 2h", cfg.Global.Timeout)
	}
	if len(cfg.Tasks) != 0 {
		t.Errorf("Tasks length = %d, want 0", len(cfg.Tasks))
	}
}

func TestLoadFromString(t *testing.T) {
	yamlContent := `
global:
  work_dir: /custom/backup
  default_tz: UTC
  max_concurrent: 10
  timeout: 1h

tasks:
  - id: test-001
    name: "Test Task"
    enabled: true
    database:
      type: mysql
      host: localhost
      port: 3306
      username: root
      password: "test123"
      database: test_db
    schedule:
      cron: "0 2 * * *"
    storage:
      type: local
      path: /backup/mysql
`

	cfg, err := LoadFromString(yamlContent)
	if err != nil {
		t.Fatalf("LoadFromString error: %v", err)
	}

	if cfg.Global.WorkDir != "/custom/backup" {
		t.Errorf("WorkDir = %s, want /custom/backup", cfg.Global.WorkDir)
	}
	if cfg.Global.MaxConcurrent != 10 {
		t.Errorf("MaxConcurrent = %d, want 10", cfg.Global.MaxConcurrent)
	}
	if len(cfg.Tasks) != 1 {
		t.Errorf("Tasks length = %d, want 1", len(cfg.Tasks))
	}
	if cfg.Tasks[0].ID != "test-001" {
		t.Errorf("Task ID = %s, want test-001", cfg.Tasks[0].ID)
	}
}

func TestLoadFromString_InvalidYAML(t *testing.T) {
	_, err := LoadFromString("invalid yaml content [")
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestLoadFromString_MissingRequired(t *testing.T) {
	yamlContent := `
tasks:
  - id: test-001
    name: ""
    database:
      type: mysql
      host: localhost
      port: 3306
`

	_, err := LoadFromString(yamlContent)
	if err == nil {
		t.Error("Expected error for missing task name")
	}
}

func TestValidate_InvalidDBType(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tasks = []model.BackupTask{
		{
			ID:   "test-001",
			Name: "Test",
			Database: model.DatabaseConfig{
				Type: "invalid",
				Host: "localhost",
				Port: 3306,
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error for invalid database type")
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tasks = []model.BackupTask{
		{
			ID:   "test-001",
			Name: "Test",
			Database: model.DatabaseConfig{
				Type: model.MySQL,
				Host: "localhost",
				Port: 99999,
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error for invalid port")
	}
}

func TestValidate_DuplicateTaskID(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tasks = []model.BackupTask{
		{
			ID:   "test-001",
			Name: "Test 1",
			Database: model.DatabaseConfig{
				Type: model.MySQL,
				Host: "localhost",
				Port: 3306,
			},
		},
		{
			ID:   "test-001",
			Name: "Test 2",
			Database: model.DatabaseConfig{
				Type: model.MySQL,
				Host: "localhost",
				Port: 3306,
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error for duplicate task ID")
	}
}

func TestValidate_InvalidMaxConcurrent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Global.MaxConcurrent = 0

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error for max_concurrent = 0")
	}
}

func TestValidate_InvalidTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Global.Timeout = 30 * time.Second

	err := Validate(cfg)
	if err == nil {
		t.Error("Expected error for timeout < 1 minute")
	}
}

func TestValidate_S3Storage(t *testing.T) {
	tests := []struct {
		name    string
		storage model.StorageConfig
		wantErr bool
	}{
		{
			name: "valid local",
			storage: model.StorageConfig{
				Type: "local",
				Path: "/tmp/backup",
			},
			wantErr: false,
		},
		{
			name: "S3 without endpoint",
			storage: model.StorageConfig{
				Type: "s3",
				Path: "/tmp/backup",
			},
			wantErr: true,
		},
		{
			name: "S3 without bucket",
			storage: model.StorageConfig{
				Type:     "s3",
				Endpoint: "https://s3.amazonaws.com",
			},
			wantErr: true,
		},
		{
			name: "valid S3",
			storage: model.StorageConfig{
				Type:     "s3",
				Endpoint: "https://s3.amazonaws.com",
				Bucket:   "my-bucket",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Tasks = []model.BackupTask{
				{
					ID:   "test-001",
					Name: "Test",
					Database: model.DatabaseConfig{
						Type: model.MySQL,
						Host: "localhost",
						Port: 3306,
					},
					Storage: tt.storage,
				},
			}

			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_Compression(t *testing.T) {
	tests := []struct {
		name    string
		comp    model.CompressionConfig
		wantErr bool
	}{
		{
			name:    "disabled",
			comp:    model.CompressionConfig{Enabled: false},
			wantErr: false,
		},
		{
			name:    "valid gzip",
			comp:    model.CompressionConfig{Enabled: true, Type: "gzip", Level: 6},
			wantErr: false,
		},
		{
			name:    "invalid type",
			comp:    model.CompressionConfig{Enabled: true, Type: "invalid", Level: 6},
			wantErr: true,
		},
		{
			name:    "invalid level",
			comp:    model.CompressionConfig{Enabled: true, Type: "gzip", Level: 10},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Tasks = []model.BackupTask{
				{
					ID:   "test-001",
					Name: "Test",
					Database: model.DatabaseConfig{
						Type: model.MySQL,
						Host: "localhost",
						Port: 3306,
					},
					Storage: model.StorageConfig{
						Type: "local",
						Path: "/tmp/backup",
					},
					Compression: tt.comp,
				},
			}

			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetTaskByID(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tasks = []model.BackupTask{
		{ID: "task-001", Name: "Task 1"},
		{ID: "task-002", Name: "Task 2"},
	}

	task, err := GetTaskByID(cfg, "task-001")
	if err != nil {
		t.Fatalf("GetTaskByID error: %v", err)
	}
	if task.Name != "Task 1" {
		t.Errorf("Task Name = %s, want Task 1", task.Name)
	}

	_, err = GetTaskByID(cfg, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent task")
	}
}

func TestGetEnabledTasks(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tasks = []model.BackupTask{
		{ID: "task-001", Name: "Task 1", Enabled: true},
		{ID: "task-002", Name: "Task 2", Enabled: false},
		{ID: "task-003", Name: "Task 3", Enabled: true},
	}

	tasks := GetEnabledTasks(cfg)
	if len(tasks) != 2 {
		t.Errorf("Enabled tasks count = %d, want 2", len(tasks))
	}
}

func TestSave(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := DefaultConfig()
	cfg.Global.WorkDir = "/test/path"

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = Save(cfg, configPath)
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// 重新加载验证
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if loaded.Global.WorkDir != "/test/path" {
		t.Errorf("WorkDir = %s, want /test/path", loaded.Global.WorkDir)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestSetDefaults(t *testing.T) {
	cfg := &model.Config{
		Tasks: []model.BackupTask{
			{
				ID:   "test-001",
				Name: "Test",
				Database: model.DatabaseConfig{
					Type: model.MySQL,
					Host: "localhost",
					Port: 3306,
				},
			},
		},
	}

	err := setDefaults(cfg)
	if err != nil {
		t.Fatalf("setDefaults error: %v", err)
	}

	// 检查全局默认值
	if cfg.Global.WorkDir != "/tmp/db-backup" {
		t.Errorf("WorkDir default = %s, want /tmp/db-backup", cfg.Global.WorkDir)
	}

	// 检查任务默认值
	if cfg.Tasks[0].Storage.Type != "local" {
		t.Errorf("Storage Type default = %s, want local", cfg.Tasks[0].Storage.Type)
	}
	if cfg.Tasks[0].Compression.Type != "gzip" {
		t.Errorf("Compression Type default = %s, want gzip", cfg.Tasks[0].Compression.Type)
	}
}
