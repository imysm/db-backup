package service

import (
	"os/exec"
	"strings"
	"testing"

	apimodel "github.com/imysm/db-backup/internal/api/model"
	internalmodel "github.com/imysm/db-backup/internal/model"
)

func TestIsRetryableError_ExitError(t *testing.T) {
	tests := []struct {
		name    string
		stderr  string
		want    bool
	}{
		{"unknown database", "ERROR 1049 (42000): Unknown database 'testdb'", false},
		{"access denied", "ERROR 1045 (28000): Access denied for user", false},
		{"authentication failed", "authentication failed for user 'root'", false},
		{"unknown option", "unknown option '--bad-option'", false},
		{"syntax error", "ERROR 1064: You have an error in your SQL syntax", false},
		{"usage error", "usage: mysqldump [OPTIONS]", false},
		{"generic exit error", "some random error output", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitErr := &exec.ExitError{Stderr: []byte(tt.stderr)}
			got := isRetryableError(exitErr)
			if got != tt.want {
				t.Errorf("isRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRetryableError_Keywords(t *testing.T) {
	tests := []struct {
		name string
		err  string
		want bool
	}{
		{"connection refused", "dial tcp 127.0.0.1:3306: connection refused", true},
		{"timeout", "context timeout", true},
		{"deadline exceeded", "context deadline exceeded", true},
		{"temporary failure", "temporary failure in name resolution", true},
		{"network unreachable", "network is unreachable", true},
		{"ETIMEDOUT", "dial tcp: i/o timeout (ETIMEDOUT)", true},
		{"ECONNREFUSED", "dial tcp: connection refused (ECONNREFUSED)", true},
		{"normal error", "invalid configuration", false},
		{"empty error", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryableError(&testError{msg: tt.err})
			if got != tt.want {
				t.Errorf("isRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

func TestNilWriter(t *testing.T) {
	var w nilWriter

	n, err := w.Write([]byte("hello"))
	if err != nil || n != 5 {
		t.Errorf("Write() = (%d, %v), want (5, nil)", n, err)
	}

	n, err = w.WriteString("world")
	if err != nil || n != 5 {
		t.Errorf("WriteString() = (%d, %v), want (5, nil)", n, err)
	}

	if err := w.Close(); err != nil {
		t.Errorf("Close() = %v, want nil", err)
	}
}

func TestConvertJobToTask(t *testing.T) {
	svc := NewBackupService(nil, nil)
	job := &apimodel.BackupJob{
		Name:         "test-job",
		DatabaseType: apimodel.DatabaseTypeMySQL,
		Host:         "localhost",
		Port:         3306,
		Database:     "testdb",
		Username:     "root",
		Password:     "secret",
		Schedule:     "0 2 * * *",
		Compress:     true,
	}

	task := svc.convertJobToTask(*job)

	if task.Name != job.Name {
		t.Errorf("Name = %q, want %q", task.Name, job.Name)
	}
	if task.Database.Type != internalmodel.MySQL {
		t.Errorf("Database.Type = %q, want mysql", task.Database.Type)
	}
	if task.Database.Host != "localhost" {
		t.Errorf("Host = %q, want localhost", task.Database.Host)
	}
	if task.Database.Port != 3306 {
		t.Errorf("Port = %d, want 3306", task.Database.Port)
	}
	if task.Database.Password != "secret" {
		t.Errorf("Password = %q, want secret", task.Database.Password)
	}
	if task.Schedule.Cron != "0 2 * * *" {
		t.Errorf("Cron = %q, want '0 2 * * *'", task.Schedule.Cron)
	}
	if !task.Compression.Enabled {
		t.Error("Compression.Enabled = false, want true")
	}
	if task.Storage.Path != "/var/lib/db-backup" {
		t.Errorf("Storage.Path = %q, want /var/lib/db-backup", task.Storage.Path)
	}
}

func TestGetStoragePath(t *testing.T) {
	svc := NewBackupService(nil, nil)
	job := &apimodel.BackupJob{
		ID:       42,
		Database: "mydb",
	}

	path := svc.GetStoragePath(job)
	if !strings.Contains(path, "42") {
		t.Errorf("path should contain job ID 42: %s", path)
	}
	if !strings.Contains(path, "mydb") {
		t.Errorf("path should contain database name: %s", path)
	}
	if !strings.HasSuffix(path, ".sql.gz") {
		t.Errorf("path should end with .sql.gz: %s", path)
	}
}

func TestNewBackupService(t *testing.T) {
	svc := NewBackupService(nil, nil)
	if svc == nil {
		t.Error("NewBackupService() returned nil")
	}
}

func TestRecordFailure(t *testing.T) {
	// recordFailure requires a non-nil DB; skip since we can't easily test nil DB
	// (it will panic, which is expected behavior)
	t.Skip("recordFailure requires valid DB")
}
