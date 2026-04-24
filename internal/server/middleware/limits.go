package middleware

import (
	"net/http"

	"github.com/Omotolani98/foostash/internal/billing"
)

// LimitChecker provides HTTP middleware that checks quotas before passing
// to the next handler. On selfhost (nil enforcer) it short-circuits efficiently.
type LimitChecker struct {
	Enforcer *billing.Enforcer
	Counter  func(orgID string) (int, error)
	Kind     billing.ResourceKind
}

// LimitMiddleware returns middleware that checks the current count via Counter,
// compares against the plan limit for Kind, and returns 402 if exceeded.
// On a nil receiver (selfhost) it passes through without touching the DB.
func (l *LimitChecker) LimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l == nil || l.Enforcer == nil {
			next.ServeHTTP(w, r)
			return
		}
		actx := AuthFromContext(r.Context())
		if actx == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		count, err := l.Counter(actx.OrgID.String())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := l.Enforcer.Check(r.Context(), actx.OrgID, l.Kind, count); err != nil {
			if _, ok := err.(*billing.LimitExceededError); ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusPaymentRequired)
				_, _ = w.Write([]byte(`{"error":{"code":"limit_exceeded","message":"plan limit exceeded","status":402}}`))
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, r)
	})
}