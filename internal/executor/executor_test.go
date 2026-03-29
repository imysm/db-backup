// Package executor 测试
package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/imysm/db-backup/internal/model"
	"github.com/imysm/db-backup/internal/util"
)

// MockCommandRunner Mock 命令执行器
type MockCommandRunner struct {
	RunFunc           func(ctx context.Context, name string, args []string, env []string) ([]byte, error)
	RunWithOutputFunc func(ctx context.Context, name string, args []string, env []string, writer model.LogWriter) error
}

func (m *MockCommandRunner) Run(ctx context.Context, name string, args []string, env []string) ([]byte, error) {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, name, args, env)
	}
	return []byte("OK"), nil
}

func (m *MockCommandRunner) RunWithOutput(ctx context.Context, name string, args []string, env []string, writer model.LogWriter) error {
	if m.RunWithOutputFunc != nil {
		return m.RunWithOutputFunc(ctx, name, args, env, writer)
	}
	return nil
}

func TestNewExecutor(t *testing.T) {
	tests := []struct {
		name    string
		dbType  model.DatabaseType
		wantErr bool
	}{
		{"MySQL", model.MySQL, false},
		{"PostgreSQL", model.PostgreSQL, false},
		{"SQLServer", model.SQLServer, false},
		{"Oracle", model.Oracle, false},
		{"Invalid", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec, err := NewExecutor(tt.dbType)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewExecutor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && exec == nil {
				t.Error("NewExecutor() returned nil executor")
			}
		})
	}
}

func TestMySQLExecutor_Type(t *testing.T) {
	exec := NewMySQLExecutor()
	if exec.Type() != "mysql" {
		t.Errorf("Type() = %s, want mysql", exec.Type())
	}
}

func TestMySQLExecutor_Validate_Success(t *testing.T) {
	exec := &MySQLExecutor{
		cmdRunner: &MockCommandRunner{
			RunFunc: func(ctx context.Context, name string, args []string, env []string) ([]byte, error) {
				return []byte("1"), nil
			},
		},
	}

	cfg := &model.DatabaseConfig{
		Type:     model.MySQL,
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "password",
		Database: "test",
	}

	err := exec.Validate(context.Background(), cfg)
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestMySQLExecutor_Validate_Failure(t *testing.T) {
	exec := &MySQLExecutor{
		cmdRunner: &MockCommandRunner{
			RunFunc: func(ctx context.Context, name string, args []string, env []string) ([]byte, error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	cfg := &model.DatabaseConfig{
		Type:     model.MySQL,
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "password",
	}

	err := exec.Validate(context.Background(), cfg)
	if err == nil {
		t.Error("Validate() expected error, got nil")
	}
}

func TestMySQLExecutor_Backup_Success(t *testing.T) {
	exec := &MySQLExecutor{
		cmdRunner: &MockCommandRunner{
			RunWithOutputFunc: func(ctx context.Context, name string, args []string, env []string, writer model.LogWriter) error {
				// 模拟创建备份文件
				return nil
			},
		},
	}

	task := &model.BackupTask{
		ID:   "test-001",
		Name: "Test Backup",
		Database: model.DatabaseConfig{
			Type:     model.MySQL,
			Host:     "localhost",
			Port:     3306,
			Username: "root",
			Password: "password",
			Database: "test_db",
		},
		Storage: model.StorageConfig{
			Type: "local",
			Path: "/tmp/db-backup-test",
		},
		Compression: model.CompressionConfig{
			Enabled: false,
		},
	}

	// 由于没有真实的 mysqldump，备份会失败
	// 这个测试主要验证函数不会 panic
	_, _ = exec.Backup(context.Background(), task, nil)
}

func TestPostgresExecutor_Type(t *testing.T) {
	exec := NewPostgresExecutor()
	if exec.Type() != "postgres" {
		t.Errorf("Type() = %s, want postgres", exec.Type())
	}
}

func TestPostgresExecutor_Validate(t *testing.T) {
	exec := NewPostgresExecutor()
	cfg := &model.DatabaseConfig{
		Type:     model.PostgreSQL,
		Host:     "localhost",
		Port:     5432,
		Username: "postgres",
		Password: "password",
	}

	err := exec.Validate(context.Background(), cfg)
	if err == nil {
		t.Error("Validate() expected error (not implemented)")
	}
}

func TestSQLServerExecutor_Type(t *testing.T) {
	exec := NewSQLServerExecutor()
	if exec.Type() != "sqlserver" {
		t.Errorf("Type() = %s, want sqlserver", exec.Type())
	}
}

func TestSQLServerExecutor_Validate(t *testing.T) {
	exec := NewSQLServerExecutor()
	cfg := &model.DatabaseConfig{
		Type:     model.SQLServer,
		Host:     "localhost",
		Port:     1433,
		Username: "sa",
		Password: "password",
	}

	err := exec.Validate(context.Background(), cfg)
	if err == nil {
		t.Error("Validate() expected error (not implemented)")
	}
}

func TestOracleExecutor_Type(t *testing.T) {
	exec := NewOracleExecutor()
	if exec.Type() != "oracle" {
		t.Errorf("Type() = %s, want oracle", exec.Type())
	}
}

func TestOracleExecutor_Validate(t *testing.T) {
	exec := NewOracleExecutor()
	cfg := &model.DatabaseConfig{
		Type:     model.Oracle,
		Host:     "localhost",
		Port:     1521,
		Username: "system",
		Password: "password",
	}

	err := exec.Validate(context.Background(), cfg)
	if err == nil {
		t.Error("Validate() expected error (not implemented)")
	}
}

func TestDefaultCommandRunner_Run(t *testing.T) {
	runner := &DefaultCommandRunner{}

	// 测试执行 echo 命令
	output, err := runner.Run(context.Background(), "echo", []string{"hello"}, nil)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
	if string(output) != "hello\n" {
		t.Errorf("Run() output = %s, want 'hello\\n'", string(output))
	}
}

func TestDefaultCommandRunner_Run_Timeout(t *testing.T) {
	runner := &DefaultCommandRunner{}

	// 创建带超时的 context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 执行一个长时间运行的命令
	_, err := runner.Run(ctx, "sleep", []string{"10"}, nil)
	if err == nil {
		t.Error("Run() expected timeout error")
	}
}

func TestFailResult(t *testing.T) {
	startTime := time.Now()
	testErr := errors.New("test error")

	result := failResult("task-001", "trace-123", startTime, testErr)

	if result.TaskID != "task-001" {
		t.Errorf("TaskID = %s, want task-001", result.TaskID)
	}
	if result.TraceID != "trace-123" {
		t.Errorf("TraceID = %s, want trace-123", result.TraceID)
	}
	if result.Status != model.TaskStatusFailed {
		t.Errorf("Status = %s, want failed", result.Status)
	}
	if result.Error != "test error" {
		t.Errorf("Error = %s, want 'test error'", result.Error)
	}
	if result.StartTime.IsZero() {
		t.Error("StartTime should not be zero")
	}
	if result.EndTime.IsZero() {
		t.Error("EndTime should not be zero")
	}
}

func TestGenerateTraceID(t *testing.T) {
	id1 := generateTraceID()
	id2 := generateTraceID()

	if id1 == "" {
		t.Error("TraceID should not be empty")
	}
	if id1 == id2 {
		t.Error("TraceIDs should be unique")
	}
}

func TestSanitizeDBName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"test_db", "test_db"},
		{"test/db", "test_db"},
		{"test\\db", "test_db"},
		{"test db", "test_db"},
		{"", "all"},
		{"../etc/passwd", "__etc_passwd"},
		{"db; DROP TABLE", "db__DROP_TABLE"},
		{"db`table", "db_table"},
		{"db&cmd", "db_cmd"},
		{"db|cmd", "db_cmd"},
		{"db$var", "db_var"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeDBName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeDBName() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{500, "500 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1572864, "1.50 MB"},
		{1073741824, "1.00 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := util.FormatFileSize(tt.size)
			if got != tt.want {
				t.Errorf("util.FormatFileSize() = %s, want %s", got, tt.want)
			}
		})
	}
}
