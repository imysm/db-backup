package model

import (
	"testing"
	"time"
)

func TestBackupJobTableName(t *testing.T) {
	job := BackupJob{}
	if job.TableName() != "backup_jobs" {
		t.Errorf("期望表名 backup_jobs, 实际 %s", job.TableName())
	}
}

func TestBackupRecordTableName(t *testing.T) {
	record := BackupRecord{}
	if record.TableName() != "backup_records" {
		t.Errorf("期望表名 backup_records, 实际 %s", record.TableName())
	}
}

func TestBackupHistoryTableName(t *testing.T) {
	history := BackupHistory{}
	if history.TableName() != "backup_history" {
		t.Errorf("期望表名 backup_history, 实际 %s", history.TableName())
	}
}

func TestBackupJobFields(t *testing.T) {
	job := BackupJob{
		Name:         "test",
		DatabaseType: DatabaseTypePostgres,
		Host:         "localhost",
		Port:         5432,
		Database:     "testdb",
		Schedule:     "0 0 * * *",
		StorageType:  StorageTypeLocal,
		Compress:     true,
		Encrypt:      false,
		NotifyOnFail: true,
		Enabled:      true,
	}

	// 验证字段
	if job.Name != "test" {
		t.Errorf("Name 字段设置失败")
	}
	if job.DatabaseType != DatabaseTypePostgres {
		t.Errorf("DatabaseType 字段设置失败")
	}
	if job.Compress != true {
		t.Errorf("Compress 应该默认启用")
	}
}

func TestBackupRecordStatus(t *testing.T) {
	record := BackupRecord{
		StartedAt: time.Now(),
	}

	record.Status = BackupStatusPending
	if record.Status != BackupStatusPending {
		t.Errorf("状态设置失败")
	}

	record.Status = BackupStatusRunning
	if record.Status != BackupStatusRunning {
		t.Errorf("状态设置失败")
	}

	record.Status = BackupStatusSuccess
	if record.Status != BackupStatusSuccess {
		t.Errorf("状态设置失败")
	}

	record.Status = BackupStatusFailed
	if record.Status != BackupStatusFailed {
		t.Errorf("状态设置失败")
	}
}

func TestDatabaseType(t *testing.T) {
	tests := []struct {
		input    DatabaseType
		expected string
	}{
		{DatabaseTypePostgres, "postgres"},
		{DatabaseTypeMySQL, "mysql"},
		{DatabaseTypeMongoDB, "mongodb"},
	}

	for _, test := range tests {
		if string(test.input) != test.expected {
			t.Errorf("期望 %s, 实际 %s", test.expected, test.input)
		}
	}
}

func TestStorageType(t *testing.T) {
	tests := []struct {
		input    StorageType
		expected string
	}{
		{StorageTypeLocal, "local"},
		{StorageTypeS3, "s3"},
		{StorageTypeOSS, "oss"},
		{StorageTypeCOS, "cos"},
	}

	for _, test := range tests {
		if string(test.input) != test.expected {
			t.Errorf("期望 %s, 实际 %s", test.expected, test.input)
		}
	}
}

func TestBackupType(t *testing.T) {
	tests := []struct {
		input    BackupType
		expected string
	}{
		{BackupTypeFull, "full"},
		{BackupTypeIncremental, "incremental"},
	}

	for _, test := range tests {
		if string(test.input) != test.expected {
			t.Errorf("期望 %s, 实际 %s", test.expected, test.input)
		}
	}
}
