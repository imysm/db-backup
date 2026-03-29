package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/imysm/db-backup/internal/model"
)

func TestLocalStorage_Download(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tmpDir, "subdir", "test.sql")
	os.MkdirAll(filepath.Dir(srcPath), 0755)
	os.WriteFile(srcPath, []byte("backup data"), 0644)

	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	// Download to a new location
	dstPath := filepath.Join(tmpDir, "downloaded", "test.sql")
	err := storage.Download(ctx, "subdir/test.sql", dstPath)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "backup data" {
		t.Errorf("Download() content = %q, want %q", string(data), "backup data")
	}
}

func TestLocalStorage_Download_SourceNotExist(t *testing.T) {
	storage := NewLocalStorage("/nonexistent")
	ctx := context.Background()

	err := storage.Download(ctx, "missing.sql", t.TempDir()+"/out.sql")
	if err == nil {
		t.Error("Download() expected error for missing source file")
	}
}

func TestLocalStorage_GetUsage(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "backup")
	os.MkdirAll(subDir, 0755)

	os.WriteFile(filepath.Join(subDir, "a.sql"), []byte("12345"), 0644)
	os.WriteFile(filepath.Join(subDir, "b.sql"), []byte("67890"), 0644)
	os.MkdirAll(filepath.Join(subDir, "nested"), 0755)
	os.WriteFile(filepath.Join(subDir, "nested", "c.sql"), []byte("abc"), 0644)

	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	total, err := storage.GetUsage(ctx, "backup")
	if err != nil {
		t.Fatalf("GetUsage() error = %v", err)
	}
	// 5 + 5 + 3 = 13
	if total != 13 {
		t.Errorf("GetUsage() = %d, want 13", total)
	}
}

func TestLocalStorage_Size_NotExist(t *testing.T) {
	storage := NewLocalStorage("/nonexistent")
	ctx := context.Background()

	_, err := storage.Size(ctx, "missing.sql")
	if err == nil {
		t.Error("Size() expected error for missing file")
	}
}

func TestLocalStorage_List_SubDir(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "backup", "1")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "file.sql"), []byte("data"), 0644)

	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	records, err := storage.List(ctx, "backup/1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 1 {
		t.Errorf("List() = %d records, want 1", len(records))
	}
}

func TestLocalStorage_List_IgnoresDirs(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "file.sql"), []byte("data"), 0644)

	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	records, err := storage.List(ctx, "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 1 {
		t.Errorf("List() = %d records, want 1 (dirs should be ignored)", len(records))
	}
}

func TestLocalStorage_Save_NonexistentSource(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	err := storage.Save(ctx, "/nonexistent/file.sql", "backup/out.sql")
	if err == nil {
		t.Error("Save() expected error for nonexistent source")
	}
}

func TestLocalStorage_Save_DeepPath(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "src.sql")
	os.WriteFile(srcFile, []byte("deep"), 0644)

	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	err := storage.Save(ctx, srcFile, "a/b/c/d/file.sql")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "a/b/c/d/file.sql")); os.IsNotExist(err) {
		t.Error("deep path not created")
	}
}

func TestDerefHelpers(t *testing.T) {
	// derefInt64
	if got := derefInt64(nil); got != 0 {
		t.Errorf("derefInt64(nil) = %d, want 0", got)
	}
	v := int64(42)
	if got := derefInt64(&v); got != 42 {
		t.Errorf("derefInt64(&42) = %d, want 42", got)
	}

	// derefTime
	if got := derefTime(nil); !got.IsZero() {
		t.Error("derefTime(nil) should be zero")
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		input string
		zero  bool
	}{
		{"2024-01-01T00:00:00Z", false},
		{"2024-01-01T00:00:00.000Z", false},
		{"invalid", true},
		{"", true},
	}
	for _, tt := range tests {
		got := parseTime(tt.input)
		if got.IsZero() != tt.zero {
			t.Errorf("parseTime(%q) zero=%v, want zero=%v", tt.input, got.IsZero(), tt.zero)
		}
	}
}

func TestIsBackupFile_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"FILE.SQL", true},
		{"file.GZ", true},
		{"file.Zip", true},
		{"file.tar", true},
		{"file.ARCHIVE", true},
		{"file.exe", false},
		{"file.pdf", false},
	}
	for _, tt := range tests {
		got := isBackupFile(tt.name)
		if got != tt.want {
			t.Errorf("isBackupFile(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestNewStorage_COS(t *testing.T) {
	cfg := model.StorageConfig{
		Type:     "cos",
		COSBucket: "test-bucket",
		COSRegion: "ap-guangzhou",
		AccessKey: "ak",
		SecretKey: "sk",
	}
	s, err := NewStorage(cfg)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	if s.Type() != "cos" {
		t.Errorf("Type() = %q, want cos", s.Type())
	}
}
