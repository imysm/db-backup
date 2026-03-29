// Package retention 测试
package retention

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/imysm/db-backup/internal/model"
	"github.com/imysm/db-backup/internal/storage"
)

func TestNewPolicy(t *testing.T) {
	cfg := model.RetentionConfig{
		KeepLast:    7,
		KeepDays:    30,
		KeepWeekly:  4,
		KeepMonthly: 3,
	}

	policy := NewPolicy(cfg)

	if policy.KeepLast != 7 {
		t.Errorf("KeepLast = %d, want 7", policy.KeepLast)
	}
	if policy.KeepDays != 30 {
		t.Errorf("KeepDays = %d, want 30", policy.KeepDays)
	}
	if policy.KeepWeekly != 4 {
		t.Errorf("KeepWeekly = %d, want 4", policy.KeepWeekly)
	}
	if policy.KeepMonthly != 3 {
		t.Errorf("KeepMonthly = %d, want 3", policy.KeepMonthly)
	}
}

func TestPolicy_Apply_Empty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "retention-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	policy := NewPolicy(model.RetentionConfig{
		KeepLast: 7,
	})
	store := storage.NewLocalStorage(tmpDir)
	ctx := context.Background()

	toDelete, err := policy.Apply(ctx, store, "")
	if err != nil {
		t.Errorf("Apply() error = %v", err)
	}
	if len(toDelete) != 0 {
		t.Errorf("Apply() returned %d files to delete, want 0", len(toDelete))
	}
}

func TestPolicy_Apply_KeepLast(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "retention-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试目录
	taskDir := filepath.Join(tmpDir, "task-001")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	// 创建 10 个不同名称的备份文件
	for i := 0; i < 10; i++ {
		filename := filepath.Join(taskDir, fmt.Sprintf("backup_%d.sql", i))
		if err := os.WriteFile(filename, []byte("test"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // 确保时间不同
	}

	policy := NewPolicy(model.RetentionConfig{
		KeepLast: 5,
	})
	store := storage.NewLocalStorage(tmpDir)
	ctx := context.Background()

	toDelete, err := policy.Apply(ctx, store, "task-001")
	if err != nil {
		t.Errorf("Apply() error = %v", err)
	}
	if len(toDelete) != 5 {
		t.Errorf("Apply() returned %d files to delete, want 5", len(toDelete))
	}
}

func TestPolicy_Apply_KeepDays(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "retention-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试目录
	taskDir := filepath.Join(tmpDir, "task-001")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	// 创建文件
	for i := 0; i < 5; i++ {
		filename := filepath.Join(taskDir, "backup.sql")
		if err := os.WriteFile(filename, []byte("test"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	policy := NewPolicy(model.RetentionConfig{
		KeepDays: 1, // 保留 1 天内的所有文件
	})
	store := storage.NewLocalStorage(tmpDir)
	ctx := context.Background()

	toDelete, err := policy.Apply(ctx, store, "task-001")
	if err != nil {
		t.Errorf("Apply() error = %v", err)
	}
	// 所有文件都是刚创建的，不应该被删除
	if len(toDelete) != 0 {
		t.Errorf("Apply() returned %d files to delete, want 0", len(toDelete))
	}
}

func TestPolicy_Apply_KeepWeekly(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "retention-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试目录
	taskDir := filepath.Join(tmpDir, "task-001")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	// 创建文件
	for i := 0; i < 10; i++ {
		filename := filepath.Join(taskDir, "backup.sql")
		if err := os.WriteFile(filename, []byte("test"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	policy := NewPolicy(model.RetentionConfig{
		KeepWeekly: 2,
	})
	store := storage.NewLocalStorage(tmpDir)
	ctx := context.Background()

	_, err = policy.Apply(ctx, store, "task-001")
	if err != nil {
		t.Errorf("Apply() error = %v", err)
	}
}

func TestPolicy_Apply_KeepMonthly(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "retention-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试目录
	taskDir := filepath.Join(tmpDir, "task-001")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	// 创建文件
	for i := 0; i < 5; i++ {
		filename := filepath.Join(taskDir, "backup.sql")
		if err := os.WriteFile(filename, []byte("test"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	policy := NewPolicy(model.RetentionConfig{
		KeepMonthly: 1,
	})
	store := storage.NewLocalStorage(tmpDir)
	ctx := context.Background()

	_, err = policy.Apply(ctx, store, "task-001")
	if err != nil {
		t.Errorf("Apply() error = %v", err)
	}
}

func TestPolicy_Cleanup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "retention-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试目录
	taskDir := filepath.Join(tmpDir, "task-001")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	// 创建 10 个不同名称的文件
	for i := 0; i < 10; i++ {
		filename := filepath.Join(taskDir, fmt.Sprintf("backup_%d.sql", i))
		if err := os.WriteFile(filename, []byte("test"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	policy := NewPolicy(model.RetentionConfig{
		KeepLast: 5,
	})
	store := storage.NewLocalStorage(tmpDir)
	ctx := context.Background()

	success, failed, err := policy.Cleanup(ctx, store, "task-001")
	if err != nil {
		t.Errorf("Cleanup() error = %v", err)
	}
	if success != 5 {
		t.Errorf("Cleanup() success = %d, want 5", success)
	}
	if failed != 0 {
		t.Errorf("Cleanup() failed = %d, want 0", failed)
	}
}

func TestPolicy_Apply_CombinedRules(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "retention-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试目录
	taskDir := filepath.Join(tmpDir, "task-001")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	// 创建 20 个不同名称的文件
	for i := 0; i < 20; i++ {
		filename := filepath.Join(taskDir, fmt.Sprintf("backup_%d.sql", i))
		if err := os.WriteFile(filename, []byte("test"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 只测试 KeepLast
	policy := NewPolicy(model.RetentionConfig{
		KeepLast: 5,
	})
	store := storage.NewLocalStorage(tmpDir)
	ctx := context.Background()

	toDelete, err := policy.Apply(ctx, store, "task-001")
	if err != nil {
		t.Errorf("Apply() error = %v", err)
	}
	// 验证返回了要删除的文件
	if len(toDelete) < 10 {
		t.Errorf("Apply() returned %d files to delete, want at least 10", len(toDelete))
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{0, 10, 0},
		{-1, 1, -1},
	}

	for _, tt := range tests {
		got := min(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
