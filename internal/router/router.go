package router

import (
	"net/http"
	"time"

	"github.com/Omotolani98/foostash/internal/handler"
	appmw "github.com/Omotolani98/foostash/internal/middleware"
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
)

type Dependencies struct {
	Auth     *handler.AuthHandler
	Projects *handler.ProjectsHandler
	Secrets  *handler.SecretsHandler
	Health   *handler.HealthHandler
	AuthSvc  *service.AuthService
	Redis    *redis.Client
}

func New(deps *Dependencies) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
			next.ServeHTTP(w, r)
		})
	})

	r.Use(appmw.RateLimit(deps.Redis, appmw.RateLimitConfig{
		Requests: 300,
		Window:   time.Minute,
	}))

	r.Get("/healthz", deps.Health.Check)

	r.Post("/v1/auth/register", deps.Auth.Register)
	r.Post("/v1/auth/login", deps.Auth.Login)

	r.Group(func(r chi.Router) {
		r.Use(appmw.Auth(deps.AuthSvc))

		r.Post("/v1/auth/api-keys", deps.Auth.CreateAPIKey)
		r.Get("/v1/auth/api-keys", deps.Auth.ListAPIKeys)
		r.Delete("/v1/auth/api-keys/{keyID}", deps.Auth.RevokeAPIKey)

		r.Get("/v1/projects", deps.Projects.List)
		r.Post("/v1/projects", deps.Projects.Create)
		r.Get("/v1/projects/{slug}", deps.Projects.Get)
		r.Delete("/v1/projects/{slug}", deps.Projects.Delete)

		r.Get("/v1/projects/{slug}/envs", deps.Projects.ListEnvs)
		r.Post("/v1/projects/{slug}/envs", deps.Projects.CreateEnv)
		r.Delete("/v1/projects/{slug}/envs/{env}", deps.Projects.DeleteEnv)

		r.Get("/v1/projects/{slug}/envs/{env}/secrets", deps.Secrets.Pull)
		r.Post("/v1/projects/{slug}/envs/{env}/secrets", deps.Secrets.Set)
		r.Delete("/v1/projects/{slug}/envs/{env}/secrets/{key}", deps.Secrets.Delete)
		r.Get("/v1/projects/{slug}/envs/{env}/secrets/{key}/versions", deps.Secrets.Versions)
		r.Get("/v1/projects/{slug}/envs/{env}/diff/{otherEnv}", deps.Secrets.Diff)
	})

	return r
}
