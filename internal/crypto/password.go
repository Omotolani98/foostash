package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

var ErrDecryptFailed = errors.New("decryption failed: wrong password or corrupted data")

const (
	argonTime    = 2
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	saltSize     = 16
)

func deriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}

func EncryptWithPassword(data []byte, password string) ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	key := deriveKey(password, salt)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := aead.Seal(nonce, nonce, data, nil)

	result := make([]byte, saltSize+len(ciphertext))
	copy(result[:saltSize], salt)
	copy(result[saltSize:], ciphertext)
	return result, nil
}

func DecryptWithPassword(data []byte, password string) ([]byte, error) {
	if len(data) < saltSize {
		return nil, ErrDecryptFailed
	}

	salt := data[:saltSize]
	ciphertext := data[saltSize:]

	key := deriveKey(password, salt)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, ErrDecryptFailed
	}

	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrDecryptFailed
	}

	nonce := ciphertext[:nonceSize]
	actualCiphertext := ciphertext[nonceSize:]

	plaintext, err := aead.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, ErrDecryptFailed
	}

	return plaintext, nil
}

func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

func Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
