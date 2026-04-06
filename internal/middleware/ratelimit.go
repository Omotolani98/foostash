package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Omotolani98/foostash/internal/handler"
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/redis/go-redis/v9"
)

type RateLimitConfig struct {
	Requests int
	Window   time.Duration
}

func RateLimit(rdb *redis.Client, cfg RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rdb == nil {
				next.ServeHTTP(w, r)
				return
			}
			key := rateKey(r)
			ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
			defer cancel()

			n, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if n == 1 {
				rdb.Expire(ctx, key, cfg.Window)
			}
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.Requests))
			remaining := max(cfg.Requests-int(n), 0)
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			if n > int64(cfg.Requests) {
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", cfg.Window.Seconds()))
				handler.WriteError(w, service.ErrRateLimited)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func rateKey(r *http.Request) string {
	auth := handler.AuthFromContext(r.Context())
	if auth != nil && auth.UserID != "" {
		return "ratelimit:user:" + auth.UserID
	}
	return "ratelimit:ip:" + clientIP(r)
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
