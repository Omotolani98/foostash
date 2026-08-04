package handlers

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Omotolani98/foostash/internal/server/middleware"
	"github.com/Omotolani98/foostash/internal/sshauth"
	"golang.org/x/crypto/ssh"
)

// registerHandler wires Register behind BufferBody exactly as the router does.
// The Auth service is nil: every case here must be rejected before the
// service is called, so a call would panic and fail the test.
func registerHandler() http.Handler {
	h := &AuthHandler{}
	return middleware.BufferBody(http.HandlerFunc(h.Register))
}

func newRegisterSigner(t *testing.T) (ssh.Signer, string) {
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
	return signer, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
}

func doRegister(t *testing.T, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	registerHandler().ServeHTTP(rec, req)
	return rec
}

func TestRegisterRejectsInvalidJSON(t *testing.T) {
	rec := doRegister(t, "{not json", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestRegisterRejectsMissingPublicKey(t *testing.T) {
	rec := doRegister(t, `{"email":"a@b.c","org_name":"Acme"}`, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestRegisterRejectsBadPublicKey(t *testing.T) {
	body := `{"email":"a@b.c","org_name":"Acme","public_key":"not-a-key"}`
	rec := doRegister(t, body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestRegisterRejectsMissingSignatureHeaders(t *testing.T) {
	_, pubKey := newRegisterSigner(t)
	body := fmt.Sprintf(`{"email":"a@b.c","org_name":"Acme","public_key":%q}`, pubKey)
	rec := doRegister(t, body, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestRegisterRejectsWrongKeySignature(t *testing.T) {
	_, pubKey := newRegisterSigner(t)
	other, _ := newRegisterSigner(t)
	body := fmt.Sprintf(`{"email":"a@b.c","org_name":"Acme","public_key":%q}`, pubKey)

	// Sign with a different key than the one in the body.
	sig, ts, err := sshauth.Sign(other, http.MethodPost, "/v1/auth/register", []byte(body))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	rec := doRegister(t, body, map[string]string{
		sshauth.HeaderTimestamp: ts,
		sshauth.HeaderSignature: sig,
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestRegisterRejectsStaleTimestamp(t *testing.T) {
	signer, pubKey := newRegisterSigner(t)
	body := fmt.Sprintf(`{"email":"a@b.c","org_name":"Acme","public_key":%q}`, pubKey)

	stale := time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339)
	payload := sshauth.CanonicalPayload(http.MethodPost, "/v1/auth/register", stale, []byte(body))
	rawSig, err := signer.Sign(rand.Reader, payload)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	rec := doRegister(t, body, map[string]string{
		sshauth.HeaderTimestamp: stale,
		sshauth.HeaderSignature: encodeSig(rawSig),
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func encodeSig(sig *ssh.Signature) string {
	return base64.StdEncoding.EncodeToString(ssh.Marshal(sig))
}
