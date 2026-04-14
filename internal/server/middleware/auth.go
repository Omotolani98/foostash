// Package middleware implements HTTP middleware for the foostash server.
package middleware

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/Omotolani98/foostash/internal/service"
	"github.com/Omotolani98/foostash/internal/sshauth"
)

type ctxKey int

const (
	authCtxKey ctxKey = iota
	bodyCtxKey
)

// WithAuthContext stores an AuthContext on a request context.
func WithAuthContext(ctx context.Context, a *service.AuthContext) context.Context {
	return context.WithValue(ctx, authCtxKey, a)
}

// AuthFromContext retrieves the AuthContext from a request context; nil if unset.
func AuthFromContext(ctx context.Context) *service.AuthContext {
	v, _ := ctx.Value(authCtxKey).(*service.AuthContext)
	return v
}

// BodyFromContext returns the raw request body bytes captured by VerifySignature.
// Handlers should use this instead of re-reading r.Body (which is already consumed).
func BodyFromContext(ctx context.Context) []byte {
	v, _ := ctx.Value(bodyCtxKey).([]byte)
	return v
}

func withBody(ctx context.Context, body []byte) context.Context {
	return context.WithValue(ctx, bodyCtxKey, body)
}

// Authenticator verifies the signature against a key known to the server and
// resolves an AuthContext. It is split from the resolver for reuse on the
// /join handler, which must verify against a key that is not yet in the DB.
type Authenticator struct {
	Auth *service.Auth
}

// RequireAuth is the standard middleware: reads + buffers the body, verifies
// the signature against the public key stored for the presented fingerprint,
// and injects the AuthContext.
func (a *Authenticator) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := readAndBuffer(r)
		if err != nil {
			writeAuthError(w, "invalid_body", err.Error(), http.StatusBadRequest)
			return
		}
		fingerprint := r.Header.Get(sshauth.HeaderFingerprint)
		timestamp := r.Header.Get(sshauth.HeaderTimestamp)
		sig := r.Header.Get(sshauth.HeaderSignature)
		if fingerprint == "" || timestamp == "" || sig == "" {
			writeAuthError(w, "missing_signature", "signature headers missing", http.StatusUnauthorized)
			return
		}

		actx, err := a.Auth.ResolveSSHKey(r.Context(), fingerprint)
		if err != nil {
			if errors.Is(err, service.ErrUserRevoked) {
				writeAuthError(w, "user_revoked", "user revoked", http.StatusUnauthorized)
				return
			}
			writeAuthError(w, "unknown_key", "unknown ssh key", http.StatusUnauthorized)
			return
		}
		owner, err := a.Auth.PublicKeyFor(r.Context(), fingerprint)
		if err != nil {
			writeAuthError(w, "unknown_key", "unknown ssh key", http.StatusUnauthorized)
			return
		}
		if err := sshauth.Verify(owner, r.Method, r.URL.Path, timestamp, sig, body); err != nil {
			writeAuthError(w, "bad_signature", err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := WithAuthContext(r.Context(), actx)
		ctx = withBody(ctx, body)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// BufferBody reads the body once and stashes it on the context for handlers
// that don't need auth but still want access (e.g. /join).
func BufferBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := readAndBuffer(r)
		if err != nil {
			writeAuthError(w, "invalid_body", err.Error(), http.StatusBadRequest)
			return
		}
		ctx := withBody(r.Context(), body)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func readAndBuffer(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	_ = r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func writeAuthError(w http.ResponseWriter, code, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":{"code":"` + code + `","message":"` + msg + `","status":` + itoa(status) + `}}`))
}

func itoa(i int) string {
	// avoid strconv import here for a tiny helper
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [12]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
