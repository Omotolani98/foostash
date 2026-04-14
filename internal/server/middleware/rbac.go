package middleware

import (
	"net/http"

	"github.com/Omotolani98/foostash/internal/server/respond"
)

// RequireRole rejects requests whose authenticated caller does not hold one of
// the allowed roles. Must be chained after Authenticator.RequireAuth so the
// AuthContext is populated.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actx := AuthFromContext(r.Context())
			if actx == nil {
				respond.WriteError(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
				return
			}
			if _, ok := allowed[actx.Role]; !ok {
				respond.WriteError(w, http.StatusForbidden, "forbidden", "role not permitted")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
