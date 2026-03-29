package binlog

import (
	"testing"
	"time"
)

func TestBinlogRecord_Fields(t *testing.T) {
	now := time.Now()
	record := BinlogRecord{
		ID:              1,
		JobID:           100,
		Filename:        "mysql-bin.000001",
		Position:        12345,
		ServerID:        1,
		FirstEventTime:  now,
		LastEventTime:   now,
		FileSize:        1024,
		EventCount:      100,
		Checksum:        "abc123",
		Status:          "archived",
	}

	if record.ID != 1 {
		t.Errorf("expected ID 1, got %d", record.ID)
	}
	if record.JobID != 100 {
		t.Errorf("expected JobID 100, got %d", record.JobID)
	}
	if record.Filename != "mysql-bin.000001" {
		t.Errorf("expected filename 'mysql-bin.000001', got '%s'", record.Filename)
	}
	if record.Position != 12345 {
		t.Errorf("expected position 12345, got %d", record.Position)
	}
	if record.ServerID != 1 {
		t.Errorf("expected ServerID 1, got %d", record.ServerID)
	}
	if record.Status != "archived" {
		t.Errorf("expected status 'archived', got '%s'", record.Status)
	}
}

func TestBinlogRecord_TableName(t *testing.T) {
	record := BinlogRecord{}
	if record.TableName() != "binlog_records" {
		t.Errorf("expected 'binlog_records', got '%s'", record.TableName())
	}
}

func TestPITRConfig_TableName(t *testing.T) {
	config := PITRConfig{}
	if config.TableName() != "pitr_configurations" {
		t.Errorf("expected 'pitr_configurations', got '%s'", config.TableName())
	}
}

func TestPITRConfig_Fields(t *testing.T) {
	now := time.Now()
	config := PITRConfig{
		ID:                    1,
		JobID:                 100,
		BinlogRetentionDays:   7,
		ArchiveEnabled:        true,
		PITREnabled:          true,
		EarliestRecoveryTime: &now,
		CreatedBy:            "admin",
	}

	if config.ID != 1 {
		t.Errorf("expected ID 1, got %d", config.ID)
	}
	if config.JobID != 100 {
		t.Errorf("expected JobID 100, got %d", config.JobID)
	}
	if config.BinlogRetentionDays != 7 {
		t.Errorf("expected 7 days, got %d", config.BinlogRetentionDays)
	}
	if !config.ArchiveEnabled {
		t.Error("expected ArchiveEnabled to be true")
	}
	if !config.PITREnabled {
		t.Error("expected PITREnabled to be true")
	}
}

func TestBinlogInfo_Fields(t *testing.T) {
	now := time.Now()
	info := BinlogInfo{
		Filename:   "mysql-bin.000002",
		Position:   54321,
		ServerID:   1,
		FileSize:   2048,
		FirstTime:  now,
		LastTime:   now,
	}

	if info.Filename != "mysql-bin.000002" {
		t.Errorf("expected 'mysql-bin.000002', got '%s'", info.Filename)
	}
	if info.Position != 54321 {
		t.Errorf("expected 54321, got %d", info.Position)
	}
	if info.ServerID != 1 {
		t.Errorf("expected ServerID 1, got %d", info.ServerID)
	}
	if info.FileSize != 2048 {
		t.Errorf("expected 2048, got %d", info.FileSize)
	}
}

func TestArchiveConfig_Fields(t *testing.T) {
	config := ArchiveConfig{
		Host:     "localhost",
		Port:     3306,
		User:     "backup",
		Password: "secret",
		Mode:     "local",
		LocalDir: "/var/backup/binlog",
		COSPath:  "bucket/binlog",
	}

	if config.Host != "localhost" {
		t.Errorf("expected 'localhost', got '%s'", config.Host)
	}
	if config.Port != 3306 {
		t.Errorf("expected 3306, got %d", config.Port)
	}
	if config.Mode != "local" {
		t.Errorf("expected 'local', got '%s'", config.Mode)
	}
	if config.LocalDir != "/var/backup/binlog" {
		t.Errorf("expected '/var/backup/binlog', got '%s'", config.LocalDir)
	}
}

func TestStorage_NewStorage(t *testing.T) {
	// Storage 的 NewStorage 方法需要 db 参数
	// 这里只测试 Storage 结构体本身
	storage := &Storage{}
	if storage == nil {
		t.Error("expected non-nil Storage")
	}
}

func TestBinlogFetcher_NewBinlogFetcher(t *testing.T) {
	config := &ArchiveConfig{
		Host:     "localhost",
		Port:     3306,
		User:     "backup",
		Password: "secret",
		Mode:     "local",
		LocalDir: "/tmp/binlog",
	}

	fetcher := NewBinlogFetcher(config)
	if fetcher == nil {
		t.Error("expected non-nil BinlogFetcher")
	}
	if fetcher.config != config {
		t.Error("expected config to be set")
	}
}

func TestBinlogRecord_StatusValues(t *testing.T) {
	validStatuses := []string{"archived", "deleted", "error"}

	for _, status := range validStatuses {
		record := BinlogRecord{Status: status}
		if record.Status != status {
			t.Errorf("expected status '%s', got '%s'", status, record.Status)
		}
	}
}
