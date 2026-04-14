package sshauth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func newTestSigner(t *testing.T) (ssh.Signer, ssh.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("pub: %v", err)
	}
	return signer, sshPub
}

func TestSignVerifyRoundTrip(t *testing.T) {
	signer, pub := newTestSigner(t)
	body := []byte(`{"hello":"world"}`)
	sig, ts, err := Sign(signer, "POST", "/v1/auth/register", body)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if err := Verify(pub, "POST", "/v1/auth/register", ts, sig, body); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestVerifyRejectsTamperedBody(t *testing.T) {
	signer, pub := newTestSigner(t)
	sig, ts, err := Sign(signer, "POST", "/v1/x", []byte("original"))
	if err != nil {
		t.Fatal(err)
	}
	err = Verify(pub, "POST", "/v1/x", ts, sig, []byte("tampered"))
	if !errors.Is(err, ErrBadSignature) {
		t.Fatalf("expected ErrBadSignature, got %v", err)
	}
}

func TestVerifyRejectsStaleTimestamp(t *testing.T) {
	signer, pub := newTestSigner(t)
	body := []byte("body")
	stale := time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339)
	payload := CanonicalPayload("GET", "/v1/x", stale, body)
	rawSig, err := signer.Sign(rand.Reader, payload)
	if err != nil {
		t.Fatal(err)
	}
	sigB64 := base64.StdEncoding.EncodeToString(ssh.Marshal(rawSig))
	if err := Verify(pub, "GET", "/v1/x", stale, sigB64, body); !errors.Is(err, ErrExpiredRequest) {
		t.Fatalf("expected ErrExpiredRequest, got %v", err)
	}
}

func TestVerifyRejectsBadTimestamp(t *testing.T) {
	_, pub := newTestSigner(t)
	err := Verify(pub, "GET", "/v1/x", "not-a-date", "ignored", nil)
	if !errors.Is(err, ErrBadTimestamp) {
		t.Fatalf("expected ErrBadTimestamp, got %v", err)
	}
}

func TestFingerprintMatchesAuthorizedKey(t *testing.T) {
	_, pub := newTestSigner(t)
	fp := Fingerprint(pub)
	if len(fp) < 10 || fp[:7] != "SHA256:" {
		t.Fatalf("unexpected fingerprint format: %q", fp)
	}
	enc := MarshalAuthorizedKey(pub)
	parsed, err := ParseAuthorizedKey(enc)
	if err != nil {
		t.Fatalf("round-trip parse: %v", err)
	}
	if Fingerprint(parsed) != fp {
		t.Fatalf("fingerprint mismatch after round-trip")
	}
}
