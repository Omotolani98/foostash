package server

import (
	"net/http"

	"github.com/Omotolani98/foostash/internal/server/handlers"
	"github.com/Omotolani98/foostash/internal/server/middleware"
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps groups dependencies needed to construct the router.
type Deps struct {
	Auth     *service.Auth
	Invites  *service.Invites
	Projects *service.Projects
	Users    *service.Users
	Audit    *service.Audit
	Pool     *pgxpool.Pool
	Version  string
}

func NewRouter(d *Deps) http.Handler {
	auth := &handlers.AuthHandler{Auth: d.Auth}
	admin := &handlers.AdminHandler{Invites: d.Invites}
	projects := &handlers.ProjectsHandler{Projects: d.Projects}
	users := &handlers.UsersHandler{Users: d.Users}
	audit := &handlers.AuditHandler{Audit: d.Audit}
	health := &handlers.HealthHandler{Pool: d.Pool, Version: d.Version}
	authMW := &middleware.Authenticator{Auth: d.Auth}
	auditor := &middleware.Auditor{Service: d.Audit}

	r := chi.NewRouter()
	r.Use(middleware.Recover)
	r.Use(middleware.Logging)

	r.Route("/v1", func(r chi.Router) {
		// /v1/health: public, pings the DB so orchestrators can tell
		// "process up" apart from "DB reachable".
		r.Get("/health", health.Get)

		// /v1/auth/register: signature proves possession of the private key,
		// but the key is not yet stored — body is buffered for the handler.
		r.With(middleware.BufferBody).Post("/auth/register", auth.Register)

		// /v1/auth/login: standard authenticated request.
		r.With(authMW.RequireAuth).Post("/auth/login", auth.Login)

		// /v1/admin/*: admin-only (roles enforced via middleware in addition
		// to service-layer checks for defence in depth).
		r.With(authMW.RequireAuth, middleware.RequireRole("admin")).Route("/admin", func(r chi.Router) {
			r.With(auditor.Record("invite.create", "invite")).Post("/invites", admin.CreateInvite)
			r.Get("/users", users.List)
			r.With(auditor.Record("user.change_role", "user")).Patch("/users/{id}/role", users.ChangeRole)
			r.With(auditor.Record("user.revoke", "user")).Delete("/users/{id}", users.Revoke)
			r.Get("/audit", audit.List)
		})

		// /v1/admin/join: cannot use RequireAuth (joining user is unknown).
		// Handler verifies the signature against the public_key in the body.
		r.With(middleware.BufferBody).Post("/admin/join", admin.Join)

		r.With(authMW.RequireAuth).Route("/projects", func(r chi.Router) {
			r.Get("/", projects.List)
			r.Get("/{slug}", projects.Get)
			r.Get("/{slug}/envs", projects.ListEnvs)

			// Mutating routes require admin.
			r.With(middleware.RequireRole("admin"), auditor.Record("project.create", "project")).
				Post("/", projects.Create)
			r.With(middleware.RequireRole("admin"), auditor.Record("project.delete", "project")).
				Delete("/{slug}", projects.Delete)
			r.With(middleware.RequireRole("admin"), auditor.Record("env.create", "environment")).
				Post("/{slug}/envs", projects.CreateEnv)
			r.With(middleware.RequireRole("admin"), auditor.Record("env.clone", "environment")).
				Post("/{slug}/envs/{env}/clone", projects.CloneEnv)
			r.With(middleware.RequireRole("admin"), auditor.Record("env.delete", "environment")).
				Delete("/{slug}/envs/{env}", projects.DeleteEnv)
		})
	})

	return r
}
