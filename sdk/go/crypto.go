package foostash

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
)

// ErrInvalidKeyLength is returned when MasterKey does not decode to 32 bytes.
var ErrInvalidKeyLength = errors.New("foostash: master key must be 32 bytes (base64-encoded)")

// engine wraps an AES-256-GCM AEAD. Read-only — SDK decrypts server blobs,
// never encrypts. Mirrors internal/crypto/engine.go.
type engine struct {
	aead cipher.AEAD
}

func newEngine(masterKeyBase64 string) (*engine, error) {
	key, err := base64.StdEncoding.DecodeString(masterKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode master key: %w", err)
	}
	if len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	return &engine{aead: aead}, nil
}

func (e *engine) Decrypt(ciphertext, nonce []byte) ([]byte, error) {
	plaintext, err := e.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}
