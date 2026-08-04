package service

import (
	"context"
	"errors"
	"testing"
)

func TestSecretsBulkSetValidatesInputBeforeRepo(t *testing.T) {
	svc := NewSecrets(nil)
	valid := SecretUpsert{Key: "A", Ciphertext: []byte("ciphertext"), Nonce: []byte("nonce")}

	cases := []struct {
		name  string
		items []SecretUpsert
	}{
		{name: "empty", items: nil},
		{name: "empty key", items: []SecretUpsert{{Key: "", Ciphertext: []byte("ciphertext"), Nonce: []byte("nonce")}}},
		{name: "empty ciphertext", items: []SecretUpsert{{Key: "A", Nonce: []byte("nonce")}}},
		{name: "empty nonce", items: []SecretUpsert{{Key: "A", Ciphertext: []byte("ciphertext")}}},
		{name: "duplicate key", items: []SecretUpsert{valid, valid}},
		{name: "oversized ciphertext", items: []SecretUpsert{{Key: "A", Ciphertext: make([]byte, maxSecretCiphertextLen+1), Nonce: []byte("nonce")}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.BulkSet(context.Background(), nil, "project", "dev", tc.items); !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("err = %v, want ErrInvalidArgument", err)
			}
		})
	}
}

func TestSecretsBulkSetRejectsTooManyItems(t *testing.T) {
	svc := NewSecrets(nil)
	items := make([]SecretUpsert, maxBulkSecrets+1)
	for i := range items {
		items[i] = SecretUpsert{Key: string(rune('a' + i%26)), Ciphertext: []byte("ciphertext"), Nonce: []byte("nonce")}
	}
	if _, err := svc.BulkSet(context.Background(), nil, "project", "dev", items); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("err = %v, want ErrInvalidArgument", err)
	}
}
