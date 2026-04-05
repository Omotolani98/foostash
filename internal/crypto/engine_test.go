package crypto_test

import (
	"bytes"
	"testing"

	"github.com/Omotolani98/foostash/internal/crypto"
)

func newTestEngine(t *testing.T) *crypto.Engine {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	e, err := crypto.NewEngine(key)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	return e
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	e := newTestEngine(t)
	plaintext := []byte("sk_live_abc123xyz")

	ct, nonce, err := e.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	got, err := e.Decrypt(ct, nonce)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("roundtrip mismatch: got %q, want %q", got, plaintext)
	}
}

func TestNonceIsUniquePerEncrypt(t *testing.T) {
	e := newTestEngine(t)
	plaintext := []byte("same-value")

	_, nonce1, err := e.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt 1: %v", err)
	}
	_, nonce2, err := e.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt 2: %v", err)
	}
	if bytes.Equal(nonce1, nonce2) {
		t.Fatal("expected unique nonces for successive encryptions")
	}
}

func TestCiphertextDiffersForSamePlaintext(t *testing.T) {
	e := newTestEngine(t)
	plaintext := []byte("same-value")

	ct1, _, _ := e.Encrypt(plaintext)
	ct2, _, _ := e.Encrypt(plaintext)
	if bytes.Equal(ct1, ct2) {
		t.Fatal("expected different ciphertexts for same plaintext")
	}
}

func TestTamperedCiphertextFails(t *testing.T) {
	e := newTestEngine(t)
	ct, nonce, err := e.Encrypt([]byte("hello"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	ct[0] ^= 0xFF
	if _, err := e.Decrypt(ct, nonce); err == nil {
		t.Fatal("expected decrypt to fail on tampered ciphertext")
	}
}

func TestInvalidKeyLength(t *testing.T) {
	_, err := crypto.NewEngine("c2hvcnQ=")
	if err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestInvalidBase64(t *testing.T) {
	_, err := crypto.NewEngine("not base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}
