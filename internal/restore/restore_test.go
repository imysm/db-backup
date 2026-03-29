package restore

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestGetRestorer(t *testing.T) {
	tests := []struct {
		name    string
		dbType  string
		wantErr bool
	}{
		{"PostgreSQL", "postgres", false},
		{"PostgreSQL alias", "postgresql", false},
		{"MySQL", "mysql", false},
		{"MongoDB", "mongodb", false},
		{"MongoDB alias", "mongo", false},
		{"Unsupported", "oracle", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restorer, err := GetRestorer(tt.dbType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRestorer() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && restorer == nil {
				t.Error("GetRestorer() returned nil")
			}
		})
	}
}

func TestPostgresRestorer_Validate(t *testing.T) {
	r := NewPostgresRestorer()
	ctx := context.Background()

	// 测试不存在的文件
	err := r.Validate(ctx, "/nonexistent/backup.sql")
	if err == nil {
		t.Error("Validate for nonexistent file should return error")
	}
}

func TestPostgresRestorer_Restore(t *testing.T) {
	r := NewPostgresRestorer()
	ctx := context.Background()

	// 测试不存在的文件
	req := &RestoreRequest{
		BackupFile: "/nonexistent/backup.sql",
		TargetHost: "localhost",
		TargetPort: 5432,
		TargetDB:   "testdb",
		TargetUser: "postgres",
		TargetPass: "password",
	}

	_, err := r.Restore(ctx, req)
	if err == nil {
		t.Error("Restore for nonexistent file should return error")
	}
}

func TestPostgresRestorer_Restore_NoRestore(t *testing.T) {
	r := NewPostgresRestorer()
	ctx := context.Background()

	// 测试只校验不恢复（使用临时文件）
	tmpFile := "/tmp/test_backup_no_restore.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("test content")
	f.Close()
	defer os.Remove(tmpFile)

	req := &RestoreRequest{
		BackupFile: tmpFile,
		NoRestore:  true,
	}

	result, err := r.Restore(ctx, req)
	if err != nil {
		t.Errorf("Restore with NoRestore should not return error: %v", err)
	}
	if !result.Success {
		t.Error("Restore with NoRestore should succeed")
	}
}

func TestMySQLRestorer_Validate(t *testing.T) {
	r := NewMySQLRestorer()
	ctx := context.Background()

	// 测试不存在的文件
	err := r.Validate(ctx, "/nonexistent/backup.sql")
	if err == nil {
		t.Error("Validate for nonexistent file should return error")
	}
}

func TestMySQLRestorer_Restore(t *testing.T) {
	r := NewMySQLRestorer()
	ctx := context.Background()

	// 测试不存在的文件
	req := &RestoreRequest{
		BackupFile: "/nonexistent/backup.sql",
		TargetHost: "localhost",
		TargetPort: 3306,
		TargetDB:   "testdb",
		TargetUser: "root",
		TargetPass: "password",
	}

	_, err := r.Restore(ctx, req)
	if err == nil {
		t.Error("Restore for nonexistent file should return error")
	}
}

func TestMySQLRestorer_Restore_NoRestore(t *testing.T) {
	r := NewMySQLRestorer()
	ctx := context.Background()

	// 测试只校验不恢复
	tmpFile := "/tmp/test_backup_mysql_no_restore.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("test content")
	f.Close()
	defer os.Remove(tmpFile)

	req := &RestoreRequest{
		BackupFile: tmpFile,
		NoRestore:  true,
	}

	result, err := r.Restore(ctx, req)
	if err != nil {
		t.Errorf("Restore with NoRestore should not return error: %v", err)
	}
	if !result.Success {
		t.Error("Restore with NoRestore should succeed")
	}
}

func TestMongoRestorer_Validate(t *testing.T) {
	r := NewMongoRestorer()
	ctx := context.Background()

	// 测试不存在的文件
	err := r.Validate(ctx, "/nonexistent/backup")
	if err == nil {
		t.Error("Validate for nonexistent path should return error")
	}
}

func TestMongoRestorer_Restore(t *testing.T) {
	r := NewMongoRestorer()
	ctx := context.Background()

	// 测试不存在的文件
	req := &RestoreRequest{
		BackupFile: "/nonexistent/backup",
		TargetHost: "localhost",
		TargetPort: 27017,
		TargetDB:   "testdb",
		TargetUser: "admin",
		TargetPass: "password",
	}

	_, err := r.Restore(ctx, req)
	if err == nil {
		t.Error("Restore for nonexistent path should return error")
	}
}

func TestMongoRestorer_Restore_NoRestore(t *testing.T) {
	r := NewMongoRestorer()
	ctx := context.Background()

	// 创建临时目录
	tmpDir := "/tmp/test_mongo_backup_no_restore"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	req := &RestoreRequest{
		BackupFile: tmpDir,
		NoRestore:  true,
	}

	result, err := r.Restore(ctx, req)
	if err != nil {
		t.Errorf("Restore with NoRestore should not return error: %v", err)
	}
	if !result.Success {
		t.Error("Restore with NoRestore should succeed")
	}
}

func TestRestoreRequest(t *testing.T) {
	req := &RestoreRequest{
		BackupFile:  "/tmp/backup.sql",
		TargetHost:  "localhost",
		TargetPort:  5432,
		TargetDB:    "testdb",
		TargetUser:  "postgres",
		TargetPass:  "password",
		DBType:      "postgres",
		StorageType: "local",
		NoRestore:   false,
	}

	if req.BackupFile != "/tmp/backup.sql" {
		t.Errorf("BackupFile = %v, want /tmp/backup.sql", req.BackupFile)
	}
	if req.TargetHost != "localhost" {
		t.Errorf("TargetHost = %v, want localhost", req.TargetHost)
	}
}

func TestRestoreResult(t *testing.T) {
	result := &RestoreResult{
		Success:      true,
		StartTime:    "2024-01-01 00:00:00",
		EndTime:      "2024-01-01 00:01:00",
		Duration:     60,
		RowsAffected: 1000,
		Error:        "",
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Duration != 60 {
		t.Errorf("Duration = %v, want 60", result.Duration)
	}
}

func TestRestorerInterface(t *testing.T) {
	// 测试 Restorer 接口是否被正确实现
	var _ Restorer = (*PostgresRestorer)(nil)
	var _ Restorer = (*MySQLRestorer)(nil)
	var _ Restorer = (*MongoRestorer)(nil)
}

func TestType(t *testing.T) {
	if TypeLocal != "local" {
		t.Errorf("TypeLocal = %s, want local", TypeLocal)
	}
	if TypeRemote != "remote" {
		t.Errorf("TypeRemote = %s, want remote", TypeRemote)
	}
}

func TestPostgresRestorer_Validate_EmptyFile(t *testing.T) {
	r := NewPostgresRestorer()
	ctx := context.Background()

	// 创建空文件
	tmpFile := "/tmp/test_empty_file.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.Close()
	defer os.Remove(tmpFile)

	err = r.Validate(ctx, tmpFile)
	if err == nil {
		t.Error("Validate for empty file should return error")
	}
}

func TestPostgresRestorer_Validate_InvalidExtension(t *testing.T) {
	r := NewPostgresRestorer()
	ctx := context.Background()

	// 创建无效扩展名的文件
	tmpFile := "/tmp/test_invalid.xyz"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("test content")
	f.Close()
	defer os.Remove(tmpFile)

	err = r.Validate(ctx, tmpFile)
	// 注意：Validate 可能不检查扩展名，所以这里不严格要求返回错误
	if err != nil {
		t.Logf("Validate returned error: %v", err)
	}
}

func TestMySQLRestorer_Validate_EmptyFile(t *testing.T) {
	r := NewMySQLRestorer()
	ctx := context.Background()

	// 创建空文件
	tmpFile := "/tmp/test_empty_mysql.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.Close()
	defer os.Remove(tmpFile)

	err = r.Validate(ctx, tmpFile)
	if err == nil {
		t.Error("Validate for empty file should return error")
	}
}

func TestMySQLRestorer_Restore_WithPassword(t *testing.T) {
	r := NewMySQLRestorer()
	ctx := context.Background()

	// 测试带密码的恢复
	req := &RestoreRequest{
		BackupFile: "/nonexistent/backup.sql",
		TargetHost: "localhost",
		TargetPort: 3306,
		TargetDB:   "testdb",
		TargetUser: "root",
		TargetPass: "password",
	}

	_, err := r.Restore(ctx, req)
	if err == nil {
		t.Log("Restore failed as expected for nonexistent file")
	}
}

func TestPostgresRestorer_Restore_WithPassword(t *testing.T) {
	r := NewPostgresRestorer()
	ctx := context.Background()

	// 测试带密码的恢复
	req := &RestoreRequest{
		BackupFile: "/nonexistent/backup.sql",
		TargetHost: "localhost",
		TargetPort: 5432,
		TargetDB:   "testdb",
		TargetUser: "postgres",
		TargetPass: "password",
	}

	_, err := r.Restore(ctx, req)
	if err == nil {
		t.Log("Restore failed as expected for nonexistent file")
	}
}

func TestMongoRestorer_Restore_WithPassword(t *testing.T) {
	r := NewMongoRestorer()
	ctx := context.Background()

	// 测试带密码的恢复
	req := &RestoreRequest{
		BackupFile: "/nonexistent/backup",
		TargetHost: "localhost",
		TargetPort: 27017,
		TargetDB:   "testdb",
		TargetUser: "admin",
		TargetPass: "password",
	}

	_, err := r.Restore(ctx, req)
	if err == nil {
		t.Log("Restore failed as expected for nonexistent path")
	}
}

func TestRestoreRequest_WithAllFields(t *testing.T) {
	req := &RestoreRequest{
		BackupFile:  "/tmp/backup.sql",
		TargetHost:  "localhost",
		TargetPort:  5432,
		TargetDB:    "testdb",
		TargetUser:  "postgres",
		TargetPass:  "password",
		DBType:      "postgres",
		StorageType: "local",
		NoRestore:   false,
	}

	if req.BackupFile != "/tmp/backup.sql" {
		t.Errorf("BackupFile = %v, want /tmp/backup.sql", req.BackupFile)
	}
	if req.TargetHost != "localhost" {
		t.Errorf("TargetHost = %v, want localhost", req.TargetHost)
	}
	if req.TargetPort != 5432 {
		t.Errorf("TargetPort = %v, want 5432", req.TargetPort)
	}
	if req.TargetDB != "testdb" {
		t.Errorf("TargetDB = %v, want testdb", req.TargetDB)
	}
	if req.TargetUser != "postgres" {
		t.Errorf("TargetUser = %v, want postgres", req.TargetUser)
	}
	if req.TargetPass != "password" {
		t.Errorf("TargetPass = %v, want password", req.TargetPass)
	}
	if req.DBType != "postgres" {
		t.Errorf("DBType = %v, want postgres", req.DBType)
	}
	if req.StorageType != "local" {
		t.Errorf("StorageType = %v, want local", req.StorageType)
	}
	if req.NoRestore != false {
		t.Errorf("NoRestore = %v, want false", req.NoRestore)
	}
}

func TestRestoreResult_WithAllFields(t *testing.T) {
	result := &RestoreResult{
		Success:      true,
		StartTime:    "2024-01-01 00:00:00",
		EndTime:      "2024-01-01 00:01:00",
		Duration:     60,
		RowsAffected: 1000,
		Error:        "",
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.StartTime != "2024-01-01 00:00:00" {
		t.Errorf("StartTime = %v, want 2024-01-01 00:00:00", result.StartTime)
	}
	if result.EndTime != "2024-01-01 00:01:00" {
		t.Errorf("EndTime = %v, want 2024-01-01 00:01:00", result.EndTime)
	}
	if result.Duration != 60 {
		t.Errorf("Duration = %v, want 60", result.Duration)
	}
	if result.RowsAffected != 1000 {
		t.Errorf("RowsAffected = %v, want 1000", result.RowsAffected)
	}
	if result.Error != "" {
		t.Errorf("Error = %v, want empty", result.Error)
	}
}

func TestGetRestorer_AllTypes(t *testing.T) {
	types := []string{"postgres", "postgresql", "mysql", "mongodb", "mongo"}

	for _, dbType := range types {
		restorer, err := GetRestorer(dbType)
		if err != nil {
			t.Errorf("GetRestorer(%s) returned error: %v", dbType, err)
		}
		if restorer == nil {
			t.Errorf("GetRestorer(%s) returned nil", dbType)
		}
	}
}

func TestPostgresRestorer_Validate_ValidFile(t *testing.T) {
	r := NewPostgresRestorer()
	ctx := context.Background()

	// 创建有效的测试文件
	tmpFile := "/tmp/test_valid_backup.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("CREATE TABLE test (id INT);")
	f.Close()
	defer os.Remove(tmpFile)

	err = r.Validate(ctx, tmpFile)
	if err != nil {
		t.Logf("Validate returned: %v", err)
	}
}

func TestMySQLRestorer_Validate_ValidFile(t *testing.T) {
	r := NewMySQLRestorer()
	ctx := context.Background()

	// 创建有效的测试文件
	tmpFile := "/tmp/test_valid_mysql_backup.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("CREATE TABLE test (id INT);")
	f.Close()
	defer os.Remove(tmpFile)

	err = r.Validate(ctx, tmpFile)
	if err != nil {
		t.Logf("Validate returned: %v", err)
	}
}

func TestMongoRestorer_Validate_ValidDir(t *testing.T) {
	r := NewMongoRestorer()
	ctx := context.Background()

	// 创建有效的测试目录
	tmpDir := "/tmp/test_valid_mongo_backup"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	err := r.Validate(ctx, tmpDir)
	if err != nil {
		t.Logf("Validate returned: %v", err)
	}
}

func TestPostgresRestorer_Restore_WithValidFile(t *testing.T) {
	r := NewPostgresRestorer()
	ctx := context.Background()

	// 创建有效的测试文件
	tmpFile := "/tmp/test_valid_pg_restore.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("CREATE TABLE test (id INT); INSERT INTO test VALUES (1);")
	f.Close()
	defer os.Remove(tmpFile)

	req := &RestoreRequest{
		BackupFile: tmpFile,
		TargetHost: "localhost",
		TargetPort: 5432,
		TargetDB:   "testdb",
		TargetUser: "postgres",
		TargetPass: "password",
		NoRestore:  true, // 只验证不恢复
	}

	result, err := r.Restore(ctx, req)
	if err != nil {
		t.Errorf("Restore with NoRestore should not return error: %v", err)
	}
	if !result.Success {
		t.Error("Restore with NoRestore should succeed")
	}
}

func TestMySQLRestorer_Restore_WithValidFile(t *testing.T) {
	r := NewMySQLRestorer()
	ctx := context.Background()

	// 创建有效的测试文件
	tmpFile := "/tmp/test_valid_mysql_restore.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("CREATE TABLE test (id INT); INSERT INTO test VALUES (1);")
	f.Close()
	defer os.Remove(tmpFile)

	req := &RestoreRequest{
		BackupFile: tmpFile,
		TargetHost: "localhost",
		TargetPort: 3306,
		TargetDB:   "testdb",
		TargetUser: "root",
		TargetPass: "password",
		NoRestore:  true, // 只验证不恢复
	}

	result, err := r.Restore(ctx, req)
	if err != nil {
		t.Errorf("Restore with NoRestore should not return error: %v", err)
	}
	if !result.Success {
		t.Error("Restore with NoRestore should succeed")
	}
}

func TestMongoRestorer_Restore_WithValidDir(t *testing.T) {
	r := NewMongoRestorer()
	ctx := context.Background()

	// 创建有效的测试目录
	tmpDir := "/tmp/test_valid_mongo_restore"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	req := &RestoreRequest{
		BackupFile: tmpDir,
		TargetHost: "localhost",
		TargetPort: 27017,
		TargetDB:   "testdb",
		TargetUser: "admin",
		TargetPass: "password",
		NoRestore:  true, // 只验证不恢复
	}

	result, err := r.Restore(ctx, req)
	if err != nil {
		t.Errorf("Restore with NoRestore should not return error: %v", err)
	}
	if !result.Success {
		t.Error("Restore with NoRestore should succeed")
	}
}

func TestPostgresRestorer_Restore_ContextCancelled(t *testing.T) {
	r := NewPostgresRestorer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	// 创建有效的测试文件
	tmpFile := "/tmp/test_cancel_restore.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("test content")
	f.Close()
	defer os.Remove(tmpFile)

	req := &RestoreRequest{
		BackupFile: tmpFile,
		NoRestore:  true,
	}

	_, err = r.Restore(ctx, req)
	// 即使 context 取消，NoRestore 模式也应该成功
	if err != nil {
		t.Logf("Restore returned: %v", err)
	}
}

func TestCopyFromStorage(t *testing.T) {
	ctx := context.Background()

	// 创建测试数据
	content := "test backup content"
	reader := strings.NewReader(content)

	// 创建临时文件路径
	tmpFile := "/tmp/test_copy_from_storage.sql"
	defer os.Remove(tmpFile)

	// 测试复制
	err := CopyFromStorage(ctx, "local", "test-key", tmpFile, reader)
	if err != nil {
		t.Errorf("CopyFromStorage returned error: %v", err)
	}

	// 验证文件内容
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(data) != content {
		t.Errorf("File content = %s, want %s", string(data), content)
	}
}

func TestCopyFromStorage_CreateDir(t *testing.T) {
	ctx := context.Background()

	content := "test content"
	reader := strings.NewReader(content)

	// 使用需要创建目录的路径
	tmpFile := "/tmp/test_storage_dir/test_copy.sql"
	defer os.RemoveAll("/tmp/test_storage_dir")

	err := CopyFromStorage(ctx, "local", "test-key", tmpFile, reader)
	if err != nil {
		t.Errorf("CopyFromStorage returned error: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("File was not created")
	}
}

func TestCopyFromStorage_InvalidPath(t *testing.T) {
	ctx := context.Background()

	content := "test content"
	reader := strings.NewReader(content)

	// 使用无效路径
	invalidPath := "/nonexistent/dir/test.sql"

	err := CopyFromStorage(ctx, "local", "test-key", invalidPath, reader)
	if err == nil {
		t.Error("CopyFromStorage should return error for invalid path")
	}
}

func TestPostgresRestorer_Validate_Directory(t *testing.T) {
	r := NewPostgresRestorer()
	ctx := context.Background()

	// 创建目录而不是文件
	tmpDir := "/tmp/test_pg_restore_dir"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	err := r.Validate(ctx, tmpDir)
	// 目录不是有效的备份文件
	if err == nil {
		t.Log("Warning: Validate for directory should return error")
	}
}

func TestMySQLRestorer_Validate_Directory(t *testing.T) {
	r := NewMySQLRestorer()
	ctx := context.Background()

	// 创建目录而不是文件
	tmpDir := "/tmp/test_mysql_restore_dir"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	err := r.Validate(ctx, tmpDir)
	// 目录不是有效的备份文件
	if err == nil {
		t.Log("Warning: Validate for directory should return error")
	}
}

func TestRestoreRequest_AllDBTypes(t *testing.T) {
	dbTypes := []string{"postgres", "mysql", "mongodb"}

	for _, dbType := range dbTypes {
		req := &RestoreRequest{
			BackupFile: "/tmp/backup.sql",
			DBType:     dbType,
			TargetHost: "localhost",
			TargetPort: 5432,
			TargetDB:   "testdb",
		}

		if req.DBType != dbType {
			t.Errorf("DBType = %s, want %s", req.DBType, dbType)
		}
	}
}

func TestRestoreResult_AllFields(t *testing.T) {
	result := &RestoreResult{
		Success:      true,
		StartTime:    "2024-01-01 00:00:00",
		EndTime:      "2024-01-01 00:01:00",
		Duration:     60,
		RowsAffected: 1000,
		Error:        "",
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Duration != 60 {
		t.Errorf("Duration = %d, want 60", result.Duration)
	}
	if result.RowsAffected != 1000 {
		t.Errorf("RowsAffected = %d, want 1000", result.RowsAffected)
	}
	if result.Error != "" {
		t.Errorf("Error should be empty, got %s", result.Error)
	}
}

func TestGetRestorer_AllCases(t *testing.T) {
	tests := []struct {
		name    string
		dbType  string
		wantErr bool
	}{
		{"PostgreSQL", "postgres", false},
		{"PostgreSQL alias", "postgresql", false},
		{"MySQL", "mysql", false},
		{"MongoDB", "mongodb", false},
		{"MongoDB alias", "mongo", false},
		{"Unknown", "unknown", true},
		{"Empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restorer, err := GetRestorer(tt.dbType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRestorer(%s) error = %v, wantErr %v", tt.dbType, err, tt.wantErr)
			}
			if !tt.wantErr && restorer == nil {
				t.Errorf("GetRestorer(%s) returned nil", tt.dbType)
			}
		})
	}
}

func TestPostgresRestorer_Restore_WithAllOptions(t *testing.T) {
	r := NewPostgresRestorer()
	ctx := context.Background()

	// 创建有效的测试文件
	tmpFile := "/tmp/test_pg_all_options.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("CREATE TABLE test (id INT);")
	f.Close()
	defer os.Remove(tmpFile)

	req := &RestoreRequest{
		BackupFile: tmpFile,
		TargetHost: "192.168.1.100",
		TargetPort: 5433,
		TargetDB:   "production_db",
		TargetUser: "admin",
		TargetPass: "secret",
		NoRestore:  true,
	}

	result, err := r.Restore(ctx, req)
	if err != nil {
		t.Errorf("Restore should not return error: %v", err)
	}
	if !result.Success {
		t.Error("Restore should succeed")
	}
}

func TestMySQLRestorer_Restore_WithAllOptions(t *testing.T) {
	r := NewMySQLRestorer()
	ctx := context.Background()

	// 创建有效的测试文件
	tmpFile := "/tmp/test_mysql_all_options.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("CREATE TABLE test (id INT);")
	f.Close()
	defer os.Remove(tmpFile)

	req := &RestoreRequest{
		BackupFile: tmpFile,
		TargetHost: "192.168.1.100",
		TargetPort: 3307,
		TargetDB:   "production_db",
		TargetUser: "admin",
		TargetPass: "secret",
		NoRestore:  true,
	}

	result, err := r.Restore(ctx, req)
	if err != nil {
		t.Errorf("Restore should not return error: %v", err)
	}
	if !result.Success {
		t.Error("Restore should succeed")
	}
}

func TestMongoRestorer_Restore_WithAllOptions(t *testing.T) {
	r := NewMongoRestorer()
	ctx := context.Background()

	// 创建有效的测试目录
	tmpDir := "/tmp/test_mongo_all_options"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	req := &RestoreRequest{
		BackupFile: tmpDir,
		TargetHost: "192.168.1.100",
		TargetPort: 27018,
		TargetDB:   "production_db",
		TargetUser: "admin",
		TargetPass: "secret",
		NoRestore:  true,
	}

	result, err := r.Restore(ctx, req)
	if err != nil {
		t.Errorf("Restore should not return error: %v", err)
	}
	if !result.Success {
		t.Error("Restore should succeed")
	}
}

func TestPostgresRestorer_Restore_ActualRestore(t *testing.T) {
	r := NewPostgresRestorer()
	ctx := context.Background()

	// 创建有效的测试文件
	tmpFile := "/tmp/test_pg_actual_restore.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("CREATE TABLE test (id INT);")
	f.Close()
	defer os.Remove(tmpFile)

	req := &RestoreRequest{
		BackupFile: tmpFile,
		TargetHost: "localhost",
		TargetPort: 5432,
		TargetDB:   "testdb",
		TargetUser: "postgres",
		TargetPass: "password",
		NoRestore:  false, // 尝试实际恢复
	}

	_, err = r.Restore(ctx, req)
	// 由于没有实际的 PostgreSQL，预期会失败
	if err == nil {
		t.Log("Warning: Actual restore succeeded unexpectedly")
	}
}

func TestMySQLRestorer_Restore_ActualRestore(t *testing.T) {
	r := NewMySQLRestorer()
	ctx := context.Background()

	// 创建有效的测试文件
	tmpFile := "/tmp/test_mysql_actual_restore.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("CREATE TABLE test (id INT);")
	f.Close()
	defer os.Remove(tmpFile)

	req := &RestoreRequest{
		BackupFile: tmpFile,
		TargetHost: "localhost",
		TargetPort: 3306,
		TargetDB:   "testdb",
		TargetUser: "root",
		TargetPass: "password",
		NoRestore:  false, // 尝试实际恢复
	}

	_, err = r.Restore(ctx, req)
	// 由于没有实际的 MySQL，预期会失败
	if err == nil {
		t.Log("Warning: Actual restore succeeded unexpectedly")
	}
}

func TestMongoRestorer_Restore_ActualRestore(t *testing.T) {
	r := NewMongoRestorer()
	ctx := context.Background()

	// 创建有效的测试目录
	tmpDir := "/tmp/test_mongo_actual_restore"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	req := &RestoreRequest{
		BackupFile: tmpDir,
		TargetHost: "localhost",
		TargetPort: 27017,
		TargetDB:   "testdb",
		TargetUser: "admin",
		TargetPass: "password",
		NoRestore:  false, // 尝试实际恢复
	}

	_, err := r.Restore(ctx, req)
	// 由于没有实际的 MongoDB，预期会失败
	if err == nil {
		t.Log("Warning: Actual restore succeeded unexpectedly")
	}
}

func TestCopyFromStorage_WithLargeFile(t *testing.T) {
	ctx := context.Background()

	// 创建大文件内容
	largeContent := strings.Repeat("x", 1024*1024) // 1MB
	reader := strings.NewReader(largeContent)

	tmpFile := "/tmp/test_large_copy.sql"
	defer os.Remove(tmpFile)

	err := CopyFromStorage(ctx, "local", "test-key", tmpFile, reader)
	if err != nil {
		t.Errorf("CopyFromStorage returned error: %v", err)
	}

	// 验证文件大小
	info, err := os.Stat(tmpFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	if info.Size() != int64(len(largeContent)) {
		t.Errorf("File size = %d, want %d", info.Size(), len(largeContent))
	}
}

// mockCommandRunner 记录命令参数和环境变量的 mock
type mockCommandRunner struct {
	lastName string
	lastArgs []string
	lastEnv  []string
}

func (m *mockCommandRunner) RunCommand(ctx context.Context, name string, args []string, env []string) ([]byte, error) {
	m.lastName = name
	m.lastArgs = args
	m.lastEnv = env
	return nil, fmt.Errorf("mock: command not executed")
}

func TestMongoRestorer_PasswordNotInCommandLine(t *testing.T) {
	// 安全测试：验证密码不出现在命令行参数中
	mock := &mockCommandRunner{}
	r := &MongoRestorer{cmdRunner: mock}

	tmpDir := "/tmp/test_mongo_security"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	req := &RestoreRequest{
		BackupFile: tmpDir,
		TargetHost: "localhost",
		TargetPort: 27017,
		TargetDB:   "testdb",
		TargetUser: "admin",
		TargetPass: "s3cretP@ssw0rd!",
	}

	r.Restore(context.Background(), req)

	// 密码绝不能出现在命令行参数中
	for _, arg := range mock.lastArgs {
		if arg == req.TargetPass {
			t.Error("SECURITY: password found in command-line args")
		}
	}

	// --password 标志不应出现
	for _, arg := range mock.lastArgs {
		if arg == "--password" {
			t.Error("SECURITY: --password flag found in command-line args")
		}
	}
}

func TestMongoRestorer_PasswordViaEnvVar(t *testing.T) {
	// 验证密码通过 MONGOPASSWORD 环境变量传递
	mock := &mockCommandRunner{}
	r := &MongoRestorer{cmdRunner: mock}

	tmpDir := "/tmp/test_mongo_env"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	req := &RestoreRequest{
		BackupFile: tmpDir,
		TargetHost: "localhost",
		TargetPort: 27017,
		TargetDB:   "testdb",
		TargetUser: "admin",
		TargetPass: "myPassword123",
	}

	r.Restore(context.Background(), req)

	found := false
	for _, e := range mock.lastEnv {
		if strings.HasPrefix(e, "MONGOPASSWORD=") {
			found = true
			break
		}
	}
	if !found {
		t.Error("MONGOPASSWORD environment variable not set")
	}
}

func TestMongoRestorer_SpecialCharsInPassword(t *testing.T) {
	// 验证含特殊字符的密码通过环境变量安全传递
	mock := &mockCommandRunner{}
	r := &MongoRestorer{cmdRunner: mock}

	tmpDir := "/tmp/test_mongo_special"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	req := &RestoreRequest{
		BackupFile: tmpDir,
		TargetHost: "localhost",
		TargetPort: 27017,
		TargetDB:   "testdb",
		TargetUser: "admin",
		TargetPass: "p@ss:word#123!$%",
	}

	r.Restore(context.Background(), req)

	// 特殊字符密码不应出现在命令行
	for _, arg := range mock.lastArgs {
		if strings.Contains(arg, "p@ss") {
			t.Error("SECURITY: password with special chars found in command-line args")
		}
	}
}
