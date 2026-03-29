// Package config - 更多测试
package config

import (
	"testing"
	"time"

	"github.com/imysm/db-backup/internal/model"
)

func TestSetDefaults_EmptyConfig(t *testing.T) {
	cfg := &model.Config{}
	err := setDefaults(cfg)
	if err != nil {
		t.Errorf("setDefaults() error = %v", err)
	}

	if cfg.Global.WorkDir != "/tmp/db-backup" {
		t.Errorf("WorkDir = %s, want /tmp/db-backup", cfg.Global.WorkDir)
	}
	if cfg.Global.Timeout != 2*time.Hour {
		t.Errorf("Timeout = %v, want 2h", cfg.Global.Timeout)
	}
}

func TestSetDefaults_TaskDefaults(t *testing.T) {
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
		t.Errorf("setDefaults() error = %v", err)
	}

	task := cfg.Tasks[0]
	if task.Storage.Type != "local" {
		t.Errorf("Storage.Type = %s, want local", task.Storage.Type)
	}
	if task.Compression.Type != "gzip" {
		t.Errorf("Compression.Type = %s, want gzip", task.Compression.Type)
	}
	if task.Compression.Level != 6 {
		t.Errorf("Compression.Level = %d, want 6", task.Compression.Level)
	}
}

func TestValidate_EmptyTaskID(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tasks = []model.BackupTask{
		{
			Name: "Test",
			Database: model.DatabaseConfig{
				Type: model.MySQL,
				Host: "localhost",
				Port: 3306,
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Validate() expected error for empty task ID")
	}
}

func TestValidate_EmptyTaskName(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tasks = []model.BackupTask{
		{
			ID: "test-001",
			Database: model.DatabaseConfig{
				Type: model.MySQL,
				Host: "localhost",
				Port: 3306,
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Validate() expected error for empty task name")
	}
}

func TestValidate_EmptyHost(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tasks = []model.BackupTask{
		{
			ID:   "test-001",
			Name: "Test",
			Database: model.DatabaseConfig{
				Type: model.MySQL,
				Port: 3306,
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Validate() expected error for empty host")
	}
}

func TestValidate_ZeroPort(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tasks = []model.BackupTask{
		{
			ID:   "test-001",
			Name: "Test",
			Database: model.DatabaseConfig{
				Type: model.MySQL,
				Host: "localhost",
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Validate() expected error for zero port")
	}
}

func TestValidate_InvalidStorageType(t *testing.T) {
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
				Type: "invalid",
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Validate() expected error for invalid storage type")
	}
}

func TestLoadFromString_ComplexConfig(t *testing.T) {
	yamlContent := `
global:
  work_dir: /data/backup
  default_tz: UTC
  max_concurrent: 10
  timeout: 4h

log:
  level: debug
  format: json

tasks:
  - id: mysql-prod
    name: "MySQL Production"
    enabled: true
    database:
      type: mysql
      host: db.example.com
      port: 3306
      username: backup
      password: "secret"
      database: "production"
      params:
        max_allowed_packet: "512M"
        quick: "true"
    schedule:
      cron: "0 0 3 * * *"
      timezone: America/New_York
    storage:
      type: local
      path: /data/backup/mysql
    retention:
      keep_last: 14
      keep_days: 60
      keep_weekly: 12
      keep_monthly: 12
    compression:
      enabled: true
      type: gzip
      level: 9
    notify:
      enabled: true
      type: dingtalk
      endpoint: "https://oapi.dingtalk.com/robot/send?access_token=xxx"
`

	cfg, err := LoadFromString(yamlContent)
	if err != nil {
		t.Fatalf("LoadFromString() error = %v", err)
	}

	if cfg.Global.WorkDir != "/data/backup" {
		t.Errorf("WorkDir = %s, want /data/backup", cfg.Global.WorkDir)
	}
	if cfg.Global.MaxConcurrent != 10 {
		t.Errorf("MaxConcurrent = %d, want 10", cfg.Global.MaxConcurrent)
	}
	if len(cfg.Tasks) != 1 {
		t.Errorf("Tasks length = %d, want 1", len(cfg.Tasks))
	}
	if cfg.Tasks[0].Database.Host != "db.example.com" {
		t.Errorf("Database.Host = %s, want db.example.com", cfg.Tasks[0].Database.Host)
	}
	if cfg.Tasks[0].Retention.KeepLast != 14 {
		t.Errorf("Retention.KeepLast = %d, want 14", cfg.Tasks[0].Retention.KeepLast)
	}
}
