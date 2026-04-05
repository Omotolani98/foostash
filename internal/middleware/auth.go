package middleware

import (
	"net/http"
	"strings"

	"github.com/Omotolani98/foostash/internal/handler"
	"github.com/Omotolani98/foostash/internal/service"
)

func Auth(authSvc *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				handler.WriteError(w, service.ErrUnauthorized)
				return
			}

			var auth *handler.AuthContext
			if strings.HasPrefix(token, service.APIKeyPrefix) {
				row, err := authSvc.ResolveAPIKey(r.Context(), token)
				if err != nil {
					handler.WriteError(w, service.ErrUnauthorized)
					return
				}
				auth = &handler.AuthContext{
					UserID: row.UserID,
					OrgID:  row.OrgID,
					Scopes: row.Scopes,
				}
			} else {
				claims, err := authSvc.VerifyJWT(token)
				if err != nil {
					handler.WriteError(w, service.ErrUnauthorized)
					return
				}
				auth = &handler.AuthContext{
					UserID: claims.UserID,
					OrgID:  claims.OrgID,
					ViaJWT: true,
				}
			}

			ctx := handler.WithAuth(r.Context(), auth)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
