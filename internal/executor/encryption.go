package executor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// sha256HashHex 计算 SHA256 校验和 (hex 编码)
func sha256HashHex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// getUint64LE 从小端字节序读取 uint64
func getUint64LE(data []byte) uint64 {
	return binary.LittleEndian.Uint64(data)
}

// getUint32LE 从小端字节序读取 uint32
func getUint32LE(data []byte) uint32 {
	return binary.LittleEndian.Uint32(data)
}

// EncryptionResult 加密结果
type EncryptionResult struct {
	OutputPath string // 加密后文件路径
	Key       string // 使用的密钥（hex格式）
}

// DecryptFile 解密文件
func DecryptFile(inputPath, outputPath, keyHex string) error {
	// 验证密钥（必须是 32 字节，不允许填充/截断）
	if err := ValidateEncryptionKey(keyHex); err != nil {
		return err
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return fmt.Errorf("密钥格式错误: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input: %w", err)
	}
	defer inputFile.Close()

	// 读取 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(inputFile, nonce); err != nil {
		return fmt.Errorf("failed to read nonce: %w", err)
	}

	// 读取加密数据
	ciphertext, err := io.ReadAll(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read ciphertext: %w", err)
	}

	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	// 写入输出
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output: %w", err)
	}
	defer outputFile.Close()

	if _, err := outputFile.Write(plaintext); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	return nil
}

// EncryptFile 加密文件
func EncryptFile(inputPath, outputPath, keyHex string) error {
	// 验证密钥（必须是 32 字节，不允许填充/截断）
	if err := ValidateEncryptionKey(keyHex); err != nil {
		return err
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return fmt.Errorf("密钥格式错误: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input: %w", err)
	}
	defer inputFile.Close()

	plaintext, err := io.ReadAll(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	// 加密
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output: %w", err)
	}
	defer outputFile.Close()

	if _, err := outputFile.Write(ciphertext); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	return nil
}

// CalculateChecksum 计算文件 SHA256 校验和
func CalculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := make([]byte, 32)
	reader := io.LimitReader(file, 10*1024*1024*1024) // 限制10GB

	n, err := reader.Read(hash)
	if err != nil && err != io.EOF {
		return "", err
	}

	return hex.EncodeToString(hash[:n]), nil
}

// EncryptBackup 加密备份文件
func EncryptBackup(inputPath, outputDir, keyHex string) (*EncryptionResult, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	filename := filepath.Base(inputPath)
	encryptedPath := filepath.Join(outputDir, filename+".enc")

	if err := EncryptFile(inputPath, encryptedPath, keyHex); err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	return &EncryptionResult{
		OutputPath: encryptedPath,
		Key:        keyHex,
	}, nil
}

// DecryptBackup 解密备份文件
func DecryptBackup(inputPath, outputDir, keyHex string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output dir: %w", err)
	}

	filename := filepath.Base(inputPath)
	decryptedPath := filepath.Join(outputDir, filename+".dec")
	decryptedPath = decryptedPath[:len(decryptedPath)-4] // 去掉 .enc

	if err := DecryptFile(inputPath, decryptedPath, keyHex); err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return decryptedPath, nil
}

// VerifyEncryptedFile 验证加密文件完整性
// keyHex: 解密密钥（hex格式，用于验证是否能正确解密文件头部）
// expectedKeyHash: 期望的密钥 SHA256 哈希（hex），可选，若为空则只验证文件格式
func VerifyEncryptedFile(filePath string, keyHex string, expectedKeyHash string) (bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("读取文件失败: %w", err)
	}

	if len(data) < 15 {
		return false, fmt.Errorf("文件太小，不是有效的加密文件")
	}

	if string(data[:4]) != "DBEN" {
		return false, fmt.Errorf("无效的 magic bytes")
	}

	version := data[4]
	if version != 0x03 {
		return false, fmt.Errorf("不支持的加密版本: %d", version)
	}

	// 验证密钥（必须是 32 字节，不允许填充/截断）
	if err := ValidateEncryptionKey(keyHex); err != nil {
		return false, err
	}

	// 解析密钥（已验证为 32 字节）
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return false, fmt.Errorf("密钥格式错误: %w", err)
	}

	// 直接使用 32 字节密钥
	key32 := key

	// 如果提供了期望的密钥哈希，进行密钥验证
	if expectedKeyHash != "" {
		actualHash := sha256HashHex(key32)
		if actualHash != expectedKeyHash {
			return false, fmt.Errorf("密钥不匹配：提供的密钥哈希为 %s，与期望的 %s 不符", actualHash, expectedKeyHash)
		}
	}

	// 尝试验证密钥是否能解密第一个块（如果文件非空）
	nonceSize := int(data[5])
	originalSize := getUint64LE(data[7:])

	if originalSize > 0 && int(originalSize) < 1024*1024*1024 {
		// 文件较小，尝试解密验证密钥正确性
		blockCipher, err := aes.NewCipher(key32)
		if err != nil {
			return false, fmt.Errorf("密钥格式错误: %w", err)
		}
		gcm, err := cipher.NewGCM(blockCipher)
		if err != nil {
			return false, fmt.Errorf("创建 GCM 失败: %w", err)
		}

		// 跳过 header，尝试读取第一个加密块
		offset := 15
		if len(data) > offset+nonceSize+4 {
			nonce := data[offset : offset+nonceSize]
			blockLen := getUint32LE(data[offset+nonceSize : offset+nonceSize+4])
			if blockLen > 0 && blockLen < 10*1024*1024 && int(blockLen) <= len(data)-offset-nonceSize-4 {
				ciphertext := data[offset+nonceSize+4 : offset+nonceSize+4+int(blockLen)]
				_, err := gcm.Open(nil, nonce, ciphertext, nil)
				if err != nil {
					return false, fmt.Errorf("密钥验证失败：无法用提供的密钥解密文件: %w", err)
				}
			}
		}
	}

	return true, nil
}

// ValidateEncryptionKey 验证加密密钥是否有效
// 要求：非空，且长度必须为 32 字节（AES-256 标准）
// 注意：hex 编码后的字符串长度是原始字节数的 2 倍，所以 hex 字符串长度应为 64
func ValidateEncryptionKey(key string) error {
	if key == "" {
		return fmt.Errorf("加密密钥不能为空，请检查任务配置中的 EncryptKey 字段")
	}
	// hex 解码后应该是 32 字节
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return fmt.Errorf("加密密钥格式错误：必须是有效的 hex 字符串")
	}
	if len(keyBytes) != 32 {
		return fmt.Errorf("加密密钥长度必须为 32 字节（AES-256），当前为 %d 字节", len(keyBytes))
	}
	return nil
}
