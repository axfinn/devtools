package utils

import (
	"strings"
	"testing"
)

func TestEncryptionService(t *testing.T) {
	// 创建加密服务
	service, err := NewEncryptionService("test-master-key-32-bytes-long!")
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	t.Run("Encrypt and Decrypt", func(t *testing.T) {
		plaintext := "my-secret-password"

		// 加密
		ciphertext, err := service.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encryption failed: %v", err)
		}

		// 密文不应该等于明文
		if ciphertext == plaintext {
			t.Error("Ciphertext should not equal plaintext")
		}

		// 解密
		decrypted, err := service.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("Decryption failed: %v", err)
		}

		// 解密后应该等于原文
		if decrypted != plaintext {
			t.Errorf("Decrypted text doesn't match. Got: %s, Want: %s", decrypted, plaintext)
		}
	})

	t.Run("Empty String", func(t *testing.T) {
		// 加密空字符串
		ciphertext, err := service.Encrypt("")
		if err != nil {
			t.Fatalf("Encryption failed: %v", err)
		}

		if ciphertext != "" {
			t.Error("Empty string should encrypt to empty string")
		}

		// 解密空字符串
		decrypted, err := service.Decrypt("")
		if err != nil {
			t.Fatalf("Decryption failed: %v", err)
		}

		if decrypted != "" {
			t.Error("Empty ciphertext should decrypt to empty string")
		}
	})

	t.Run("Long Text", func(t *testing.T) {
		// 测试长文本（SSH私钥）
		plaintext := strings.Repeat("This is a very long SSH private key content. ", 100)

		ciphertext, err := service.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encryption failed: %v", err)
		}

		decrypted, err := service.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("Decryption failed: %v", err)
		}

		if decrypted != plaintext {
			t.Error("Long text decryption failed")
		}
	})

	t.Run("Different Nonce Each Time", func(t *testing.T) {
		plaintext := "same-text"

		// 加密两次
		ciphertext1, _ := service.Encrypt(plaintext)
		ciphertext2, _ := service.Encrypt(plaintext)

		// 由于每次使用不同的 nonce，密文应该不同
		if ciphertext1 == ciphertext2 {
			t.Error("Same plaintext should produce different ciphertexts due to random nonce")
		}

		// 但都应该能正确解密
		decrypted1, _ := service.Decrypt(ciphertext1)
		decrypted2, _ := service.Decrypt(ciphertext2)

		if decrypted1 != plaintext || decrypted2 != plaintext {
			t.Error("Both ciphertexts should decrypt to the same plaintext")
		}
	})

	t.Run("Invalid Ciphertext", func(t *testing.T) {
		// 尝试解密无效的密文
		_, err := service.Decrypt("invalid-base64-!@#$")
		if err == nil {
			t.Error("Should fail to decrypt invalid base64")
		}

		// 尝试解密被篡改的密文
		validCiphertext, _ := service.Encrypt("test")
		tamperedCiphertext := validCiphertext + "extra"

		_, err = service.Decrypt(tamperedCiphertext)
		if err == nil {
			t.Error("Should fail to decrypt tampered ciphertext")
		}
	})

	t.Run("EncryptIfNotEmpty", func(t *testing.T) {
		// 非空字符串应该加密
		result, err := service.EncryptIfNotEmpty("password")
		if err != nil || result == "" || result == "password" {
			t.Error("EncryptIfNotEmpty should encrypt non-empty string")
		}

		// 空字符串应该返回空
		result, err = service.EncryptIfNotEmpty("")
		if err != nil || result != "" {
			t.Error("EncryptIfNotEmpty should return empty for empty string")
		}
	})

	t.Run("DecryptIfNotEmpty", func(t *testing.T) {
		// 先加密
		encrypted, _ := service.Encrypt("password")

		// 非空密文应该解密
		result, err := service.DecryptIfNotEmpty(encrypted)
		if err != nil || result != "password" {
			t.Error("DecryptIfNotEmpty should decrypt non-empty ciphertext")
		}

		// 空密文应该返回空
		result, err = service.DecryptIfNotEmpty("")
		if err != nil || result != "" {
			t.Error("DecryptIfNotEmpty should return empty for empty string")
		}
	})
}

func TestNewEncryptionService(t *testing.T) {
	t.Run("Valid Key", func(t *testing.T) {
		service, err := NewEncryptionService("any-length-key")
		if err != nil {
			t.Fatalf("Should accept any length key: %v", err)
		}
		if len(service.masterKey) != 32 {
			t.Errorf("Master key should be 32 bytes, got: %d", len(service.masterKey))
		}
	})

	t.Run("Empty Key", func(t *testing.T) {
		_, err := NewEncryptionService("")
		if err == nil {
			t.Error("Should reject empty key")
		}
	})

	t.Run("Different Keys Produce Different Results", func(t *testing.T) {
		service1, _ := NewEncryptionService("key1")
		service2, _ := NewEncryptionService("key2")

		plaintext := "test"
		encrypted1, _ := service1.Encrypt(plaintext)

		// 用不同的密钥解密应该失败
		_, err := service2.Decrypt(encrypted1)
		if err == nil {
			t.Error("Decryption with different key should fail")
		}
	})
}

func TestGenerateRandomKey(t *testing.T) {
	key1, err := GenerateRandomKey()
	if err != nil {
		t.Fatalf("Failed to generate random key: %v", err)
	}

	key2, err := GenerateRandomKey()
	if err != nil {
		t.Fatalf("Failed to generate random key: %v", err)
	}

	// 两个随机密钥应该不同
	if key1 == key2 {
		t.Error("Random keys should be different")
	}

	// 应该能成功创建加密服务
	service, err := NewEncryptionService(key1)
	if err != nil {
		t.Errorf("Should be able to create service with generated key: %v", err)
	}

	// 应该能正常加密解密
	plaintext := "test"
	ciphertext, _ := service.Encrypt(plaintext)
	decrypted, _ := service.Decrypt(ciphertext)

	if decrypted != plaintext {
		t.Error("Generated key should work for encryption/decryption")
	}
}
