package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateChecksum(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	sum, err := CalculateChecksum(tmpFile)
	if err != nil {
		t.Fatalf("CalculateChecksum() error = %v", err)
	}
	if sum == "" {
		t.Error("CalculateChecksum() returned empty string")
	}

	// 相同内容应返回相同校验和
	sum2, err := CalculateChecksum(tmpFile)
	if err != nil {
		t.Fatalf("CalculateChecksum() error = %v", err)
	}
	if sum != sum2 {
		t.Errorf("same file produced different checksums: %s != %s", sum, sum2)
	}

	// 不存在的文件
	_, err = CalculateChecksum("/nonexistent/file")
	if err == nil {
		t.Error("CalculateChecksum() expected error for nonexistent file")
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
	}

	for _, tt := range tests {
		got := FormatFileSize(tt.size)
		if got != tt.want {
			t.Errorf("FormatFileSize(%d) = %q, want %q", tt.size, got, tt.want)
		}
	}
}
