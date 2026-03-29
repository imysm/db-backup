// Package executor - 更多测试
package executor

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/imysm/db-backup/internal/model"
	"github.com/imysm/db-backup/internal/util"
)

// TestLogWriter 测试日志写入器
type TestLogWriter struct {
	buf bytes.Buffer
}

func (w *TestLogWriter) Write(p []byte) (n int, err error)       { return w.buf.Write(p) }
func (w *TestLogWriter) WriteString(s string) (n int, err error) { return w.buf.WriteString(s) }
func (w *TestLogWriter) Close() error                            { return nil }
func (w *TestLogWriter) String() string                          { return w.buf.String() }

func TestMySQLExecutor_BuildDumpArgs(t *testing.T) {
	exec := NewMySQLExecutor()

	tests := []struct {
		name     string
		task     *model.BackupTask
		contains []string
	}{
		{
			name: "basic args",
			task: &model.BackupTask{
				Database: model.DatabaseConfig{
					Host:     "localhost",
					Port:     3306,
					Username: "root",
					Database: "test_db",
				},
			},
			contains: []string{"-hlocalhost", "-P3306", "-uroot", "--single-transaction", "test_db"},
		},
		{
			name: "with params",
			task: &model.BackupTask{
				Database: model.DatabaseConfig{
					Host:     "localhost",
					Port:     3306,
					Username: "root",
					Database: "test_db",
					Params: map[string]string{
						"max_allowed_packet": "1G",
						"quick":              "true",
					},
				},
			},
			contains: []string{"--max-allowed-packet=1G", "--quick"},
		},
		{
			name: "all databases",
			task: &model.BackupTask{
				Database: model.DatabaseConfig{
					Host:     "localhost",
					Port:     3306,
					Username: "root",
					Database: "",
				},
			},
			contains: []string{"--all-databases"},
		},
		{
			name: "ignore tables",
			task: &model.BackupTask{
				Database: model.DatabaseConfig{
					Host:     "localhost",
					Port:     3306,
					Username: "root",
					Database: "test_db",
					Params: map[string]string{
						"ignore_table": "test_db.logs,test_db.temp",
					},
				},
			},
			contains: []string{"--ignore-table=test_db.logs", "--ignore-table=test_db.temp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := exec.buildDumpArgs(tt.task, "/tmp/backup.sql")

			for _, want := range tt.contains {
				found := false
				for _, arg := range args {
					if arg == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("buildDumpArgs() missing %s in %v", want, args)
				}
			}
		})
	}
}

func TestMySQLExecutor_CompressFile(t *testing.T) {
	// 创建临时文件
	tmpDir, err := os.MkdirTemp("", "compress-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.sql")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	exec := NewMySQLExecutor()
	writer := &TestLogWriter{}

	compressedPath, err := exec.compressFile(context.Background(), testFile, model.CompressionConfig{
		Enabled: true,
		Type:    "gzip",
		Level:   6,
	}, writer)

	if err != nil {
		t.Errorf("compressFile() error = %v", err)
	}

	// 验证压缩文件存在
	if _, err := os.Stat(compressedPath); os.IsNotExist(err) {
		t.Error("压缩文件未创建")
	}

	// 验证扩展名
	if filepath.Ext(compressedPath) != ".gz" {
		t.Errorf("压缩文件扩展名 = %s, want .gz", filepath.Ext(compressedPath))
	}
}

func TestMySQLExecutor_CalculateChecksum(t *testing.T) {
	// 创建临时文件
	tmpDir, err := os.MkdirTemp("", "checksum-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.sql")
	content := []byte("test content for checksum")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	checksum, err := util.CalculateChecksum(testFile)

	if err != nil {
		t.Errorf("calculateChecksum() error = %v", err)
	}

	if checksum == "" {
		t.Error("checksum should not be empty")
	}

	// 验证两次计算结果相同
	checksum2, _ := util.CalculateChecksum(testFile)
	if checksum != checksum2 {
		t.Errorf("checksums differ: %s vs %s", checksum, checksum2)
	}
}

func TestMySQLExecutor_CalculateChecksum_FileNotFound(t *testing.T) {
	_, err := util.CalculateChecksum("/nonexistent/file")

	if err == nil {
		t.Error("calculateChecksum() expected error for nonexistent file")
	}
}

func TestRunCommand(t *testing.T) {
	ctx := context.Background()

	output, err := runCommand(ctx, "echo", []string{"hello"}, nil)
	if err != nil {
		t.Errorf("runCommand() error = %v", err)
	}
	if string(output) != "hello\n" {
		t.Errorf("runCommand() output = %s, want 'hello\\n'", string(output))
	}
}

func TestRunCommand_WithEnv(t *testing.T) {
	ctx := context.Background()

	env := []string{"TEST_VAR=test_value"}
	output, err := runCommand(ctx, "sh", []string{"-c", "echo $TEST_VAR"}, env)
	if err != nil {
		t.Errorf("runCommand() error = %v", err)
	}
	if string(output) != "test_value\n" {
		t.Errorf("runCommand() output = %s, want 'test_value\\n'", string(output))
	}
}

func TestRunCommandWithOutput(t *testing.T) {
	ctx := context.Background()
	writer := &TestLogWriter{}

	err := runCommandWithOutput(ctx, "echo", []string{"test"}, nil, writer)
	if err != nil {
		t.Errorf("runCommandWithOutput() error = %v", err)
	}

	// 输出可能包含换行符，验证包含预期内容
	if writer.String() == "" {
		t.Error("runCommandWithOutput() wrote nothing")
	}
}

func TestRunCommand_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := runCommand(ctx, "sleep", []string{"10"}, nil)
	if err == nil {
		t.Error("runCommand() expected timeout error")
	}
}

func TestSanitizeDBName_AllDatabases(t *testing.T) {
	// 空数据库名应该返回 "all"
	result := sanitizeDBName("")
	if result != "all" {
		t.Errorf("sanitizeDBName('') = %s, want 'all'", result)
	}
}

func TestNewExecutorWithDeps(t *testing.T) {
	mockRunner := &MockCommandRunner{}

	exec, err := NewExecutorWithDeps(model.MySQL, mockRunner)
	if err != nil {
		t.Errorf("NewExecutorWithDeps() error = %v", err)
	}
	if exec == nil {
		t.Error("NewExecutorWithDeps() returned nil")
	}

	// 测试不支持的类型
	_, err = NewExecutorWithDeps("invalid", mockRunner)
	if err == nil {
		t.Error("NewExecutorWithDeps() expected error for invalid type")
	}
}

func TestPostgresExecutor_WithMock(t *testing.T) {
	mockRunner := &MockCommandRunner{}
	exec := &PostgresExecutor{cmdRunner: mockRunner}

	if exec.Type() != "postgres" {
		t.Errorf("Type() = %s, want postgres", exec.Type())
	}

	// Validate 已实现，使用 mock runner 应返回 nil
	err := exec.Validate(context.Background(), &model.DatabaseConfig{})
	if err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestSQLServerExecutor_WithMock(t *testing.T) {
	mockRunner := &MockCommandRunner{}
	exec := &SQLServerExecutor{cmdRunner: mockRunner}

	if exec.Type() != "sqlserver" {
		t.Errorf("Type() = %s, want sqlserver", exec.Type())
	}

	// Validate 已实现，使用 mock runner 应返回 nil
	err := exec.Validate(context.Background(), &model.DatabaseConfig{})
	if err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestOracleExecutor_WithMock(t *testing.T) {
	mockRunner := &MockCommandRunner{}
	exec := &OracleExecutor{cmdRunner: mockRunner}

	if exec.Type() != "oracle" {
		t.Errorf("Type() = %s, want oracle", exec.Type())
	}

	// Validate 已实现，使用 mock runner 应返回 nil
	err := exec.Validate(context.Background(), &model.DatabaseConfig{})
	if err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}
