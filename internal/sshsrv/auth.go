package sshsrv

import (
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"
)

type ctxKey int

const authCtxKey ctxKey = iota

// AuthFromContext returns the AuthContext stashed on an ssh.Context by the
// public-key auth callback, or nil if none was stored.
func AuthFromContext(ctx ssh.Context) *service.AuthContext {
	v, _ := ctx.Value(authCtxKey).(*service.AuthContext)
	return v
}

// publicKeyHandler returns a Wish public-key callback that resolves the
// presented key's SHA256 fingerprint to an AuthContext via auth.ResolveSSHKey,
// then stashes it on the ssh.Context for the session handler.
func publicKeyHandler(auth *service.Auth) ssh.PublicKeyHandler {
	return func(ctx ssh.Context, key ssh.PublicKey) bool {
		fingerprint := gossh.FingerprintSHA256(key)
		actx, err := auth.ResolveSSHKey(ctx, fingerprint)
		if err != nil || actx == nil {
			return false
		}
		ctx.SetValue(authCtxKey, actx)
		return true
	}
}
