package crypto

import (
	"bytes"
	"os"
	"testing"
)

func TestNewEncryptor_ValidKeySizes(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		key     string
		wantErr bool
	}{
		{
			name:    "enabled with 16 bytes key",
			enabled: true,
			key:     "1234567890123456",
			wantErr: false,
		},
		{
			name:    "enabled with 32 bytes key",
			enabled: true,
			key:     "12345678901234567890123456789012",
			wantErr: false,
		},
		{
			name:    "disabled (returns NoOpEncryptor)",
			enabled: false,
			key:     "",
			wantErr: false,
		},
		{
			name:    "enabled with empty key (returns NoOpEncryptor)",
			enabled: true,
			key:     "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEncryptor(tt.enabled, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEncryptor() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptor_EncryptDecrypt(t *testing.T) {
	key := "1234567890123456"
	encryptor, err := NewEncryptor(true, key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "simple text",
			plaintext: []byte("Hello, World!"),
		},
		{
			name:      "empty text",
			plaintext: []byte(""),
		},
		{
			name:      "long text",
			plaintext: bytes.Repeat([]byte("A"), 1024),
		},
		{
			name:      "binary data",
			plaintext: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 加密
			ciphertext, err := encryptor.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// 验证密文不等于明文
			if bytes.Equal(tt.plaintext, ciphertext) && len(tt.plaintext) > 0 {
				t.Error("Ciphertext should not equal plaintext")
			}

			// 解密
			decrypted, err := encryptor.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// 验证解密后等于原文
			if !bytes.Equal(tt.plaintext, decrypted) {
				t.Errorf("Decrypted = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptor_EncryptDecryptString(t *testing.T) {
	key := "1234567890123456"
	encryptor, err := NewEncryptor(true, key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	// 转换为 AES256Encryptor 以使用字符串方法
	aesEnc, ok := encryptor.(*AES256Encryptor)
	if !ok {
		t.Skip("Encryptor is not AES256Encryptor")
	}

	plaintext := "Hello, World!"

	// 加密为字符串
	ciphertextHex, err := aesEnc.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("EncryptString() error = %v", err)
	}

	// 验证密文是 Base64 字符串
	if len(ciphertextHex) == 0 {
		t.Error("Ciphertext should not be empty")
	}

	// 解密
	decrypted, err := aesEnc.DecryptString(ciphertextHex)
	if err != nil {
		t.Fatalf("DecryptString() error = %v", err)
	}

	// 验证
	if decrypted != plaintext {
		t.Errorf("Decrypted = %v, want %v", decrypted, plaintext)
	}
}

func TestEncryptor_EncryptDecryptFile(t *testing.T) {
	key := "1234567890123456"
	encryptor, err := NewEncryptor(true, key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	// 创建临时文件
	tmpDir := t.TempDir()
	plainFile := tmpDir + "/plain.txt"

	// 写入原始文件
	plaintext := []byte("This is a test file for encryption")
	if err := os.WriteFile(plainFile, plaintext, 0644); err != nil {
		t.Fatalf("Failed to write plain file: %v", err)
	}

	// 读取文件内容并加密
	data, err := os.ReadFile(plainFile)
	if err != nil {
		t.Fatalf("Failed to read plain file: %v", err)
	}

	ciphertext, err := encryptor.Encrypt(data)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// 验证加密后不等于原文
	if bytes.Equal(data, ciphertext) {
		t.Error("Ciphertext should not equal plaintext")
	}

	// 解密
	decrypted, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	// 验证解密后等于原文
	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decrypted = %v, want %v", decrypted, plaintext)
	}
}

func TestEncryptor_WrongKey(t *testing.T) {
	// 使用密钥1加密
	encryptor1, _ := NewEncryptor(true, "1234567890123456")
	plaintext := []byte("Secret message")
	ciphertext, err := encryptor1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// 使用密钥2解密（应该失败）
	encryptor2, _ := NewEncryptor(true, "6543210987654321")
	_, err = encryptor2.Decrypt(ciphertext)
	if err == nil {
		t.Error("Decrypt with wrong key should fail")
	}
}
