package verify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Verifier 备份验证器
type Verifier struct {
	storageType string
	config      map[string]interface{}
}

// VerificationResult 验证结果
type VerificationResult struct {
	Passed     bool   // 是否通过
	FileExists bool   // 文件是否存在
	ChecksumOK bool   // 校验和是否正确
	RestoreOK  bool   // 恢复测试是否成功
	SizeMatch  bool   // 大小是否匹配
	Error      string // 错误信息
}

// NewVerifier 创建验证器
func NewVerifier(storageType string, config map[string]interface{}) *Verifier {
	return &Verifier{
		storageType: storageType,
		config:      config,
	}
}

// VerifyFile 验证备份文件
func (v *Verifier) VerifyFile(ctx context.Context, filePath string, expectedChecksum string) (*VerificationResult, error) {
	result := &VerificationResult{}

	// 1. 检查文件是否存在
	exists, err := v.checkFileExists(ctx, filePath)
	if err != nil {
		return result, err
	}
	result.FileExists = exists
	if !exists {
		result.Error = "备份文件不存在"
		return result, nil
	}

	// 2. 验证校验和
	if expectedChecksum != "" {
		checksum, err := v.calculateChecksum(ctx, filePath)
		if err != nil {
			result.Error = fmt.Sprintf("计算校验和失败: %v", err)
			return result, nil
		}
		result.ChecksumOK = (checksum == expectedChecksum)
		if !result.ChecksumOK {
			result.Error = fmt.Sprintf("校验和不匹配: 期望 %s, 实际 %s", expectedChecksum, checksum)
			return result, nil
		}
	}

	result.Passed = true
	return result, nil
}

// checkFileExists 检查文件是否存在
func (v *Verifier) checkFileExists(ctx context.Context, filePath string) (bool, error) {
	switch v.storageType {
	case "local":
		_, err := os.Stat(filePath)
		if os.IsNotExist(err) {
			return false, nil
		}
		return err == nil, err
	default:
		// 云存储需要调用 API 检查
		return true, nil
	}
}

// calculateChecksum 计算文件校验和
func (v *Verifier) calculateChecksum(ctx context.Context, filePath string) (string, error) {
	var reader io.Reader

	switch v.storageType {
	case "local":
		f, err := os.Open(filePath)
		if err != nil {
			return "", err
		}
		defer f.Close()
		reader = f
	default:
		return "", fmt.Errorf("不支持的存储类型: %s", v.storageType)
	}

	hasher := sha256.New()
	_, err := io.Copy(hasher, reader)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// TestRestore 测试恢复
func (v *Verifier) TestRestore(ctx context.Context, filePath string, dbType string) error {
	// 简化实现：只检查文件是否存在且可读
	switch v.storageType {
	case "local":
		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("打开备份文件失败: %w", err)
		}
		defer f.Close()

		// 检查文件是否为空
		info, err := f.Stat()
		if err != nil {
			return fmt.Errorf("获取文件信息失败: %w", err)
		}
		if info.Size() == 0 {
			return fmt.Errorf("备份文件为空")
		}

		// 检查文件扩展名
		ext := filepath.Ext(filePath)
		if ext != ".sql" && ext != ".dump" && ext != ".tar.gz" && ext != ".gz" {
			return fmt.Errorf("未知的备份文件格式: %s", ext)
		}

		return nil
	default:
		return fmt.Errorf("只支持本地存储的恢复测试")
	}
}


