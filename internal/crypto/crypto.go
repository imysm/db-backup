// Package crypto 提供 AES 加密/解密功能
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrInvalidKey 无效的加密密钥
	ErrInvalidKey = errors.New("encryption key must be 16, 24, or 32 bytes")
	// ErrInvalidCiphertext 无效的密文
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	// ErrShortCiphertext 密文太短
	ErrShortCiphertext = errors.New("ciphertext too short")
)

// Encryptor 加密器接口
type Encryptor interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
	EncryptString(plaintext string) (string, error)
	DecryptString(encoded string) (string, error)
}

// AES256Encryptor AES-256-GCM 加密器
type AES256Encryptor struct {
	key []byte
	gcm cipher.AEAD
}

// NewAES256Encryptor 创建 AES-256 加密器
// key 可以是任意长度，会通过 SHA256 派生出 32 字节密钥
func NewAES256Encryptor(key string) (*AES256Encryptor, error) {
	// 使用 SHA256 派生 32 字节密钥
	hash := sha256.Sum256([]byte(key))
	derivedKey := hash[:]

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 失败: %w", err)
	}

	return &AES256Encryptor{
		key: derivedKey,
		gcm: gcm,
	}, nil
}

// Encrypt 加密数据
func (e *AES256Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	// 生成随机 nonce
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成 nonce 失败: %w", err)
	}

	// 加密 (nonce + ciphertext)
	ciphertext := e.gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt 解密数据
func (e *AES256Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrShortCiphertext
	}

	nonce, encryptedData := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := e.gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %w", err)
	}

	return plaintext, nil
}

// EncryptString 加密字符串并返回 Base64 编码
func (e *AES256Encryptor) EncryptString(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString 解密 Base64 编码的字符串
func (e *AES256Encryptor) DecryptString(encoded string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("Base64 解码失败: %w", err)
	}

	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// NoOpEncryptor 无操作加密器（不加密）
type NoOpEncryptor struct{}

// NewNoOpEncryptor 创建无操作加密器
func NewNoOpEncryptor() *NoOpEncryptor {
	return &NoOpEncryptor{}
}

// Encrypt 直接返回原文
func (e *NoOpEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	return plaintext, nil
}

// Decrypt 直接返回密文（实际是原文）
func (e *NoOpEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	return ciphertext, nil
}

// EncryptString 直接返回原字符串
func (e *NoOpEncryptor) EncryptString(plaintext string) (string, error) {
	return plaintext, nil
}

// DecryptString 直接返回原字符串
func (e *NoOpEncryptor) DecryptString(encoded string) (string, error) {
	return encoded, nil
}

// NewEncryptor 根据配置创建加密器
func NewEncryptor(enabled bool, key string) (Encryptor, error) {
	if !enabled || key == "" {
		return NewNoOpEncryptor(), nil
	}
	return NewAES256Encryptor(key)
}
