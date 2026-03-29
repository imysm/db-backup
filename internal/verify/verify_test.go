package verify

import (
	"context"
	"os"
	"testing"

	"github.com/imysm/db-backup/internal/util"
)

func TestNewVerifier(t *testing.T) {
	v := NewVerifier("local", nil)
	if v == nil {
		t.Error("NewVerifier() returned nil")
	}
}

func TestVerifier_VerifyFile_NonExistent(t *testing.T) {
	v := NewVerifier("local", nil)
	ctx := context.Background()

	result, err := v.VerifyFile(ctx, "/nonexistent/backup.sql", "")
	if err != nil {
		t.Errorf("VerifyFile should not return error: %v", err)
	}
	if result.FileExists {
		t.Error("FileExists should be false for nonexistent file")
	}
	if result.Passed {
		t.Error("Passed should be false for nonexistent file")
	}
}

func TestVerifier_VerifyFile_Exists(t *testing.T) {
	v := NewVerifier("local", nil)
	ctx := context.Background()

	// 创建临时测试文件
	tmpFile := "/tmp/test_verify_backup.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("test content")
	f.Close()
	defer os.Remove(tmpFile)

	result, err := v.VerifyFile(ctx, tmpFile, "")
	if err != nil {
		t.Errorf("VerifyFile should not return error: %v", err)
	}
	if !result.FileExists {
		t.Error("FileExists should be true for existing file")
	}
	if !result.Passed {
		t.Error("Passed should be true when no checksum to verify")
	}
}

func TestVerifier_VerifyFile_WithChecksum(t *testing.T) {
	v := NewVerifier("local", nil)
	ctx := context.Background()

	// 创建临时测试文件
	tmpFile := "/tmp/test_verify_checksum.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("test content")
	f.Close()
	defer os.Remove(tmpFile)

	// 计算正确的校验和
	correctChecksum, err := util.CalculateChecksum(tmpFile)
	if err != nil {
		t.Fatalf("计算校验和失败: %v", err)
	}

	// 测试正确的校验和
	result, err := v.VerifyFile(ctx, tmpFile, correctChecksum)
	if err != nil {
		t.Errorf("VerifyFile should not return error: %v", err)
	}
	if !result.ChecksumOK {
		t.Error("ChecksumOK should be true for correct checksum")
	}
	if !result.Passed {
		t.Error("Passed should be true for correct checksum")
	}

	// 测试错误的校验和
	result, err = v.VerifyFile(ctx, tmpFile, "wrongchecksum")
	if err != nil {
		t.Errorf("VerifyFile should not return error: %v", err)
	}
	if result.ChecksumOK {
		t.Error("ChecksumOK should be false for wrong checksum")
	}
	if result.Passed {
		t.Error("Passed should be false for wrong checksum")
	}
}

func TestVerifier_TestRestore(t *testing.T) {
	v := NewVerifier("local", nil)
	ctx := context.Background()

	// 测试不存在的文件
	err := v.TestRestore(ctx, "/nonexistent/backup.sql", "postgres")
	if err == nil {
		t.Error("TestRestore for nonexistent file should return error")
	}
}

func TestVerifier_TestRestore_ValidFile(t *testing.T) {
	v := NewVerifier("local", nil)
	ctx := context.Background()

	// 创建有效的测试文件
	tmpFile := "/tmp/test_restore_valid.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("CREATE TABLE test (id INT);")
	f.Close()
	defer os.Remove(tmpFile)

	err = v.TestRestore(ctx, tmpFile, "postgres")
	if err != nil {
		t.Errorf("TestRestore for valid file should not return error: %v", err)
	}
}

func TestVerifier_TestRestore_EmptyFile(t *testing.T) {
	v := NewVerifier("local", nil)
	ctx := context.Background()

	// 创建空文件
	tmpFile := "/tmp/test_restore_empty.sql"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.Close()
	defer os.Remove(tmpFile)

	err = v.TestRestore(ctx, tmpFile, "postgres")
	if err == nil {
		t.Error("TestRestore for empty file should return error")
	}
}

func TestVerifier_TestRestore_InvalidExtension(t *testing.T) {
	v := NewVerifier("local", nil)
	ctx := context.Background()

	// 创建无效扩展名的文件
	tmpFile := "/tmp/test_restore_invalid.xyz"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("test content")
	f.Close()
	defer os.Remove(tmpFile)

	err = v.TestRestore(ctx, tmpFile, "postgres")
	if err == nil {
		t.Error("TestRestore for invalid extension should return error")
	}
}

func TestCalculateChecksum(t *testing.T) {
	// 创建临时文件
	tmpFile := "/tmp/test_checksum.txt"
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	f.WriteString("hello world")
	f.Close()
	defer os.Remove(tmpFile)

	// 计算校验和
	checksum, err := util.CalculateChecksum(tmpFile)
	if err != nil {
		t.Errorf("CalculateChecksum should not return error: %v", err)
	}
	if checksum == "" {
		t.Error("Checksum should not be empty")
	}

	// 相同内容应该产生相同校验和
	checksum2, err := util.CalculateChecksum(tmpFile)
	if err != nil {
		t.Errorf("CalculateChecksum should not return error: %v", err)
	}
	if checksum != checksum2 {
		t.Error("Same file should produce same checksum")
	}
}

func TestCalculateChecksum_NonExistent(t *testing.T) {
	_, err := util.CalculateChecksum("/nonexistent/file")
	if err == nil {
		t.Error("CalculateChecksum for nonexistent file should return error")
	}
}

func TestVerificationResult(t *testing.T) {
	result := &VerificationResult{
		Passed:     true,
		FileExists: true,
		ChecksumOK: true,
		RestoreOK:  true,
		SizeMatch:  true,
		Error:      "",
	}

	if !result.Passed {
		t.Error("Passed should be true")
	}
	if !result.FileExists {
		t.Error("FileExists should be true")
	}
	if !result.ChecksumOK {
		t.Error("ChecksumOK should be true")
	}
}

func TestVerifier_UnsupportedStorage(t *testing.T) {
	v := NewVerifier("s3", nil)
	ctx := context.Background()

	// S3 存储类型的文件存在检查应该返回 true（简化实现）
	exists, err := v.checkFileExists(ctx, "some/key")
	if err != nil {
		t.Errorf("checkFileExists for s3 should not return error: %v", err)
	}
	if !exists {
		t.Error("checkFileExists for s3 should return true")
	}

	// S3 存储类型的校验和计算应该返回错误
	_, err = v.calculateChecksum(ctx, "some/key")
	if err == nil {
		t.Error("calculateChecksum for s3 should return error")
	}
}
