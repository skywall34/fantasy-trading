package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var encryptionKey []byte

// InitEncryption initializes the encryption key from environment or generates a new one
func InitEncryption() error {
	keyStr := os.Getenv("ENCRYPTION_KEY")
	if keyStr == "" {
		// Generate a random 32-byte key for AES-256
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return fmt.Errorf("failed to generate encryption key: %w", err)
		}
		encryptionKey = key
		keyBase64 := base64.StdEncoding.EncodeToString(key)

		// Save the key to .env file
		if err := saveKeyToEnvFile(keyBase64); err != nil {
			fmt.Printf("WARNING: Failed to save encryption key to .env: %v\n", err)
			fmt.Printf("Please manually set ENCRYPTION_KEY env var to: %s\n", keyBase64)
		} else {
			fmt.Printf("Generated new encryption key and saved to .env file\n")
		}
	} else {
		key, err := base64.StdEncoding.DecodeString(keyStr)
		if err != nil {
			return fmt.Errorf("failed to decode encryption key: %w", err)
		}
		if len(key) != 32 {
			return fmt.Errorf("encryption key must be 32 bytes (256 bits), got %d bytes", len(key))
		}
		encryptionKey = key
		fmt.Printf("Loaded encryption key from environment\n")
	}
	return nil
}

// saveKeyToEnvFile saves the encryption key to .env file
func saveKeyToEnvFile(keyBase64 string) error {
	envPath := ".env"

	// Read existing .env file if it exists
	var existingContent string
	if data, err := os.ReadFile(envPath); err == nil {
		existingContent = string(data)
		// Check if ENCRYPTION_KEY already exists (shouldn't happen, but just in case)
		if strings.Contains(existingContent, "ENCRYPTION_KEY=") {
			return nil // Key already exists, don't overwrite
		}
	}

	// Ensure existing content ends with newline if it exists
	if existingContent != "" && !strings.HasSuffix(existingContent, "\n") {
		existingContent += "\n"
	}

	// Append the encryption key
	newContent := existingContent + fmt.Sprintf("ENCRYPTION_KEY=%s\n", keyBase64)

	// Ensure directory exists
	dir := filepath.Dir(envPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Write the file with appropriate permissions
	if err := os.WriteFile(envPath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	return nil
}

// Encrypt encrypts plaintext using AES-256-GCM
func Encrypt(plaintext string) (string, error) {
	if len(encryptionKey) == 0 {
		return "", errors.New("encryption key not initialized")
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func Decrypt(ciphertext string) (string, error) {
	if len(encryptionKey) == 0 {
		return "", errors.New("encryption key not initialized")
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// DecryptAPIKeys is a helper function to decrypt API key and secret
func DecryptAPIKeys(encryptedKey, encryptedSecret string) (string, string, error) {
	apiKey, err := Decrypt(encryptedKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	apiSecret, err := Decrypt(encryptedSecret)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt API secret: %w", err)
	}

	return apiKey, apiSecret, nil
}
