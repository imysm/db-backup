package executor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// EncryptionResult 加密结果
type EncryptionResult struct {
	OutputPath string // 加密后文件路径
	Key       string // 使用的密钥（hex格式）
}

// DecryptFile 解密文件
func DecryptFile(inputPath, outputPath, keyHex string) error {
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return fmt.Errorf("invalid key format: %w", err)
	}

	// 确保密钥是32字节
	if len(key) < 32 {
		// 填充密钥到32字节
		padded := make([]byte, 32)
		copy(padded, key)
		key = padded
	} else if len(key) > 32 {
		key = key[:32]
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
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return fmt.Errorf("invalid key format: %w", err)
	}

	// 确保密钥是32字节
	if len(key) < 32 {
		padded := make([]byte, 32)
		copy(padded, key)
		key = padded
	} else if len(key) > 32 {
		key = key[:32]
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
