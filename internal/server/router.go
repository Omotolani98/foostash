package server

import (
	"net/http"

	"github.com/Omotolani98/foostash/internal/server/handlers"
	"github.com/Omotolani98/foostash/internal/server/middleware"
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/go-chi/chi/v5"
)

// Deps groups dependencies needed to construct the router.
type Deps struct {
	Auth     *service.Auth
	Invites  *service.Invites
	Projects *service.Projects
}

func NewRouter(d *Deps) http.Handler {
	auth := &handlers.AuthHandler{Auth: d.Auth}
	admin := &handlers.AdminHandler{Invites: d.Invites}
	projects := &handlers.ProjectsHandler{Projects: d.Projects}
	authMW := &middleware.Authenticator{Auth: d.Auth}

	r := chi.NewRouter()
	r.Use(middleware.Recover)
	r.Use(middleware.Logging)

	r.Route("/v1", func(r chi.Router) {
		// /v1/auth/register: signature proves possession of the private key,
		// but the key is not yet stored — body is buffered for the handler.
		r.With(middleware.BufferBody).Post("/auth/register", auth.Register)

		// /v1/auth/login: standard authenticated request.
		r.With(authMW.RequireAuth).Post("/auth/login", auth.Login)

		// /v1/admin/invites: admin-only.
		r.With(authMW.RequireAuth).Post("/admin/invites", admin.CreateInvite)

		// /v1/admin/join: cannot use RequireAuth (joining user is unknown).
		// Handler verifies the signature against the public_key in the body.
		r.With(middleware.BufferBody).Post("/admin/join", admin.Join)

		r.With(authMW.RequireAuth).Route("/projects", func(r chi.Router) {
			r.Get("/", projects.List)
			r.Post("/", projects.Create)
			r.Get("/{slug}", projects.Get)
			r.Delete("/{slug}", projects.Delete)
			r.Get("/{slug}/envs", projects.ListEnvs)
			r.Post("/{slug}/envs", projects.CreateEnv)
			r.Post("/{slug}/envs/{env}/clone", projects.CloneEnv)
			r.Delete("/{slug}/envs/{env}", projects.DeleteEnv)
		})
	})

	return r
}
