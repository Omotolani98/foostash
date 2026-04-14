package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/Omotolani98/foostash/internal/repo"
	"github.com/Omotolani98/foostash/internal/service"
)

// Auditor records request outcomes for mutating routes.
type Auditor struct {
	Service *service.Audit
}

// Record returns middleware that inserts an audit row after the wrapped
// handler completes. Only runs for authenticated routes (needs AuthContext).
// action is a stable label like "project.create".
func (a *Auditor) Record(action, resourceType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			actx := AuthFromContext(r.Context())
			if actx == nil {
				return
			}
			uid := actx.UserID
			in := repo.AuditInsert{
				OrgID:        actx.OrgID,
				UserID:       &uid,
				Action:       action,
				ResourceType: resourceType,
				ResourceID:   resourceIDFromRequest(r),
				Status:       rec.status,
				RequestIP:    clientIP(r),
			}
			// Use a detached context so a client disconnect doesn't abort the write.
			go func() {
				if err := a.Service.Record(context.Background(), in); err != nil {
					slog.Error("audit record", "err", err, "action", action)
				}
			}()
		})
	}
}

func resourceIDFromRequest(r *http.Request) string {
	// Best-effort: last path segment for resource-scoped routes. Good enough
	// for projects/{slug}, users/{id}, invites, etc.
	p := r.URL.Path
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			if i == len(p)-1 {
				continue
			}
			return p[i+1:]
		}
	}
	return p
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
