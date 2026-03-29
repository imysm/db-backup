// Package storage 测试
package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/imysm/db-backup/internal/model"
)

func TestLocalStorage_Type(t *testing.T) {
	storage := NewLocalStorage("/tmp")
	if storage.Type() != "local" {
		t.Errorf("Type() = %s, want local", storage.Type())
	}
}

func TestLocalStorage_Save(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "storage-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建源文件
	srcFile := filepath.Join(tmpDir, "source.txt")
	if err := os.WriteFile(srcFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("创建源文件失败: %v", err)
	}

	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	// 测试保存
	err = storage.Save(ctx, srcFile, "backup/test.txt")
	if err != nil {
		t.Errorf("Save() error = %v", err)
	}

	// 验证文件存在
	destFile := filepath.Join(tmpDir, "backup", "test.txt")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Error("目标文件未创建")
	}
}

func TestLocalStorage_List(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "storage-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试文件
	testFiles := []string{"backup1.sql", "backup2.sql.gz", "backup3.dump"}
	for _, name := range testFiles {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("test"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	// 创建非备份文件（应该被忽略）
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	records, err := storage.List(ctx, "")
	if err != nil {
		t.Errorf("List() error = %v", err)
	}

	if len(records) != 3 {
		t.Errorf("List() returned %d records, want 3", len(records))
	}
}

func TestLocalStorage_List_EmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	records, err := storage.List(ctx, "")
	if err != nil {
		t.Errorf("List() error = %v", err)
	}
	if len(records) != 0 {
		t.Errorf("List() returned %d records, want 0", len(records))
	}
}

func TestLocalStorage_List_NonexistentDir(t *testing.T) {
	storage := NewLocalStorage("/nonexistent/path")
	ctx := context.Background()

	records, err := storage.List(ctx, "")
	if err != nil {
		t.Errorf("List() error = %v", err)
	}
	if len(records) != 0 {
		t.Errorf("List() returned %d records, want 0", len(records))
	}
}

func TestLocalStorage_Delete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.sql")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	// 删除文件
	err = storage.Delete(ctx, "test.sql")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// 验证文件已删除
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("文件未被删除")
	}
}

func TestLocalStorage_Delete_Nonexistent(t *testing.T) {
	storage := NewLocalStorage("/tmp")
	ctx := context.Background()

	// 删除不存在的文件应该不报错
	err := storage.Delete(ctx, "nonexistent.sql")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}
}

func TestLocalStorage_Exists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.sql")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	// 测试存在的文件
	exists, err := storage.Exists(ctx, "test.sql")
	if err != nil {
		t.Errorf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true")
	}

	// 测试不存在的文件
	exists, err = storage.Exists(ctx, "nonexistent.sql")
	if err != nil {
		t.Errorf("Exists() error = %v", err)
	}
	if exists {
		t.Error("Exists() = true, want false")
	}
}

func TestLocalStorage_Size(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.sql")
	content := []byte("test content here")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	size, err := storage.Size(ctx, "test.sql")
	if err != nil {
		t.Errorf("Size() error = %v", err)
	}
	if size != int64(len(content)) {
		t.Errorf("Size() = %d, want %d", size, len(content))
	}
}

func TestLocalStorage_Save_SourceRemoval(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	t.Run("源文件正常删除", func(t *testing.T) {
		srcFile := filepath.Join(tmpDir, "source_remove_test.sql")
		if err := os.WriteFile(srcFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
		err := storage.Save(ctx, srcFile, "backup/removed.sql")
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
			t.Error("源文件应被删除")
		}
	})

	t.Run("源文件不存在不报错", func(t *testing.T) {
		// 目标文件已存在（模拟 IsNotExist 场景）
		destFile := filepath.Join(tmpDir, "backup", "already.sql")
		if err := os.WriteFile(destFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
		// 源文件不存在，但目标已存在，localPath == destPath 路径
		err := storage.Save(ctx, destFile, "backup/already.sql")
		if err != nil {
			t.Fatalf("Save() same path error = %v", err)
		}
	})
}

func TestLocalStorage_Save_SourceRemoveError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建文件后设置父目录为只读，使 os.Remove 失败
	roDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(roDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcFile := filepath.Join(roDir, "locked.sql")
	if err := os.WriteFile(srcFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(roDir, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(roDir, 0755) //nolint:errcheck
	if os.Getuid() == 0 {
		t.Skip("以 root 运行，无法测试权限限制")
	}

	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	err = storage.Save(ctx, srcFile, "backup/readonly.sql")
	if err == nil {
		t.Error("期望 os.Remove 失败导致返回错误")
	}
}

func TestNewStorage(t *testing.T) {
	tests := []struct {
		name     string
		cfg      model.StorageConfig
		wantErr  bool
		wantType string
	}{
		{
			name: "local storage",
			cfg: model.StorageConfig{
				Type: "local",
				Path: "/tmp",
			},
			wantErr:  false,
			wantType: "local",
		},
		{
			name: "empty type defaults to local",
			cfg: model.StorageConfig{
				Path: "/tmp",
			},
			wantErr:  false,
			wantType: "local",
		},
		{
			name: "s3 storage",
			cfg: model.StorageConfig{
				Type:     "s3",
				Endpoint: "https://s3.amazonaws.com",
				Bucket:   "my-bucket",
			},
			wantErr:  false,
			wantType: "s3",
		},
		{
			name: "oss storage",
			cfg: model.StorageConfig{
				Type:     "oss",
				Endpoint: "https://oss-cn-hangzhou.aliyuncs.com",
				Bucket:   "my-bucket",
			},
			wantErr:  false,
			wantType: "oss",
		},
		{
			name: "invalid type",
			cfg: model.StorageConfig{
				Type: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewStorage(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStorage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && storage.Type() != tt.wantType {
				t.Errorf("Type() = %s, want %s", storage.Type(), tt.wantType)
			}
		})
	}
}

func TestS3Storage_NotImplemented(t *testing.T) {
	cfg := model.StorageConfig{
		Type:     "s3",
		Endpoint: "https://s3.amazonaws.com",
		Bucket:   "test",
	}
	storage, _ := NewS3Storage(cfg)
	ctx := context.Background()

	// Save - 空本地文件路径应报错
	if err := storage.Save(ctx, "", ""); err == nil {
		t.Error("Save() expected error for empty path")
	}

	// List - 会调用 S3 API，无网络连接应报错
	if _, err := storage.List(ctx, ""); err == nil {
		t.Error("List() expected error (no network)")
	}

	// Delete - 会调用 S3 API，无网络连接应报错
	if err := storage.Delete(ctx, ""); err == nil {
		t.Error("Delete() expected error (no network)")
	}

	// Exists - 空路径返回 (false, nil)
	exists, err := storage.Exists(ctx, "")
	if err != nil {
		t.Errorf("Exists() unexpected error: %v", err)
	}
	if exists {
		t.Error("Exists() = true for empty path, want false")
	}

	// Size - 空路径应报错
	if _, err := storage.Size(ctx, ""); err == nil {
		t.Error("Size() expected error for empty path")
	}
}

func TestIsBackupFile(t *testing.T) {
	tests := []struct {
		name string
		ext  string
		want bool
	}{
		{".sql", ".sql", true},
		{".gz", ".gz", true},
		{".zip", ".zip", true},
		{".dump", ".dump", true},
		{".bak", ".bak", true},
		{".dmp", ".dmp", true},
		{".txt", ".txt", false},
		{".log", ".log", false},
		{".md", ".md", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := isBackupFile("backup" + tt.ext)
			if got != tt.want {
				t.Errorf("isBackupFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFileAge(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.sql")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	age, err := GetFileAge(testFile)
	if err != nil {
		t.Errorf("GetFileAge() error = %v", err)
	}
	if age != 0 {
		t.Errorf("GetFileAge() = %d, want 0 (just created)", age)
	}

	// 测试不存在的文件
	_, err = GetFileAge("/nonexistent/file")
	if err == nil {
		t.Error("GetFileAge() expected error for nonexistent file")
	}
}

func TestLocalStorage_List_SortedByTime(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建多个文件，间隔创建时间
	files := []string{"old.sql", "new.sql", "middle.sql"}
	for i, name := range files {
		time.Sleep(10 * time.Millisecond) // 确保时间不同
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("test"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
		_ = i
	}

	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	records, err := storage.List(ctx, "")
	if err != nil {
		t.Errorf("List() error = %v", err)
	}

	// 验证按时间降序排列
	if len(records) >= 2 {
		if !records[0].CreatedAt.After(records[1].CreatedAt) {
			t.Error("List() records not sorted by time descending")
		}
	}
}
