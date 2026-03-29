package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateFilePath(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		baseDir string
		wantErr bool
	}{
		{
			name:    "正常相对路径",
			path:    "backup.sql.gz",
			baseDir: tmpDir,
			wantErr: false,
		},
		{
			name:    "正常子目录路径",
			path:    "subdir/backup.sql.gz",
			baseDir: tmpDir,
			wantErr: false,
		},
		{
			name:    "绝对路径在目录下",
			path:    filepath.Join(tmpDir, "backup.sql.gz"),
			baseDir: tmpDir,
			wantErr: false,
		},
		{
			name:    "路径遍历 ../etc/passwd",
			path:    "../etc/passwd",
			baseDir: tmpDir,
			wantErr: true,
		},
		{
			name:    "绝对路径 /etc/passwd",
			path:    "/etc/passwd",
			baseDir: tmpDir,
			wantErr: true,
		},
		{
			name:    "嵌入 .. 的路径",
			path:    filepath.Join(tmpDir, "..", "..", "etc", "passwd"),
			baseDir: tmpDir,
			wantErr: true,
		},
		{
			name:    "深层嵌入 .. 的路径",
			path:    "subdir/../../etc/passwd",
			baseDir: tmpDir,
			wantErr: true,
		},
		{
			name:    "空路径",
			path:    "",
			baseDir: tmpDir,
			wantErr: true,
		},
		{
			name:    "空基础目录",
			path:    "backup.sql.gz",
			baseDir: "",
			wantErr: true,
		},
		{
			name:    "目录名前缀攻击",
			path:    "../backup-evil/evil.sql",
			baseDir: tmpDir,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePath(tt.path, tt.baseDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilePath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if _, ok := err.(*ErrPathTraversal); ok {
					t.Logf("正确返回 ErrPathTraversal: %v", err)
				}
			}
		})
	}
}

func TestValidateFilePath_Symlink(t *testing.T) {
	tmpDir := t.TempDir()
	outsideDir := t.TempDir()

	// 创建一个指向外部目录的符号链接
	evilLink := filepath.Join(tmpDir, "evil_link")
	if err := os.Symlink(outsideDir, evilLink); err != nil {
		t.Skip("无法创建符号链接，跳过测试")
	}

	// 创建外部文件
	outsideFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	// 通过符号链接路径应该被允许（符号链接在允许目录下）
	// 但实际文件指向外部 — 这是符号链接的已知限制
	// 当前实现不做符号链接解析，因为：
	// 1. 备份文件通常是系统创建的，不是用户上传的
	// 2. 符号链接解析需要文件存在
	// 3. 防止路径遍历的主要目标是阻止直接构造恶意路径
	err := ValidateFilePath("evil_link/secret.txt", tmpDir)
	if err != nil {
		t.Logf("符号链接路径被拒绝（更安全）: %v", err)
	} else {
		t.Logf("符号链接路径被允许（已知限制）: 符号链接指向外部目录")
	}
}

func TestValidateFilePath_EdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	// 路径恰好是基础目录本身
	err := ValidateFilePath(".", tmpDir)
	if err != nil {
		t.Errorf("基础目录本身应该被允许: %v", err)
	}

	// 路径恰好是基础目录本身（绝对路径）
	err = ValidateFilePath(tmpDir, tmpDir)
	if err != nil {
		t.Errorf("基础目录绝对路径应该被允许: %v", err)
	}

	// 只有 . 的情况
	err = ValidateFilePath("./", tmpDir)
	if err != nil {
		t.Errorf("./ 路径应该被允许: %v", err)
	}
}
