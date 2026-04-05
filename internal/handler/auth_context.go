package handler

import "context"

type ctxKey int

const authCtxKey ctxKey = 1

type AuthContext struct {
	UserID string
	OrgID  string
	Scopes []string
	ViaJWT bool
}

func (a *AuthContext) HasScope(scope string) bool {
	if a.ViaJWT {
		return true
	}
	for _, s := range a.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

func WithAuth(ctx context.Context, auth *AuthContext) context.Context {
	return context.WithValue(ctx, authCtxKey, auth)
}

func AuthFromContext(ctx context.Context) *AuthContext {
	a, _ := ctx.Value(authCtxKey).(*AuthContext)
	return a
}
