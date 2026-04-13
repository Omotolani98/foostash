package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Omotolani98/foostash/internal/server/middleware"
	"github.com/Omotolani98/foostash/internal/server/respond"
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/go-chi/chi/v5"
)

type ProjectsHandler struct {
	Projects *service.Projects
}

type createProjectRequest struct {
	Name string `json:"name"`
}

type projectSummary struct {
	ID        string `json:"id,omitempty"`
	Slug      string `json:"slug"`
	Name      string `json:"name,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

func (h *ProjectsHandler) Create(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	body := middleware.BodyFromContext(r.Context())
	var req createProjectRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respond.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	proj, err := h.Projects.Create(r.Context(), actx, req.Name)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	respond.WriteJSON(w, http.StatusCreated, projectSummary{
		ID:        proj.ID.String(),
		Slug:      proj.Slug,
		Name:      proj.Name,
		CreatedAt: proj.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

type projectListEntryDTO struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	EnvCount  int    `json:"env_count"`
	UpdatedAt string `json:"updated_at"`
}

type projectListResponse struct {
	Projects []projectListEntryDTO `json:"projects"`
}

func (h *ProjectsHandler) List(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	items, err := h.Projects.List(r.Context(), actx)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	out := projectListResponse{Projects: make([]projectListEntryDTO, 0, len(items))}
	for _, it := range items {
		out.Projects = append(out.Projects, projectListEntryDTO{
			Slug:      it.Slug,
			Name:      it.Name,
			EnvCount:  it.EnvCount,
			UpdatedAt: it.UpdatedAt,
		})
	}
	respond.WriteJSON(w, http.StatusOK, out)
}

type projectDetailResponse struct {
	ID           string    `json:"id"`
	Slug         string    `json:"slug"`
	Name         string    `json:"name"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`
	Environments []envDTO  `json:"environments"`
}

type envDTO struct {
	Slug      string `json:"slug"`
	CreatedAt string `json:"created_at"`
}

func (h *ProjectsHandler) Get(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	slug := chi.URLParam(r, "slug")
	view, err := h.Projects.Get(r.Context(), actx, slug)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	resp := projectDetailResponse{
		ID:        view.ID.String(),
		Slug:      view.Slug,
		Name:      view.Name,
		CreatedAt: view.CreatedAt,
		UpdatedAt: view.UpdatedAt,
	}
	for _, e := range view.Environments {
		resp.Environments = append(resp.Environments, envDTO{Slug: e.Slug, CreatedAt: e.CreatedAt})
	}
	respond.WriteJSON(w, http.StatusOK, resp)
}

func (h *ProjectsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	slug := chi.URLParam(r, "slug")
	if err := h.Projects.Delete(r.Context(), actx, slug); err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Environments ---

type envListResponse struct {
	Environments []envDTO `json:"environments"`
}

func (h *ProjectsHandler) ListEnvs(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	slug := chi.URLParam(r, "slug")
	envs, err := h.Projects.Envs().List(r.Context(), actx, slug)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	out := envListResponse{Environments: make([]envDTO, 0, len(envs))}
	for _, e := range envs {
		out.Environments = append(out.Environments, envDTO{Slug: e.Slug, CreatedAt: e.CreatedAt})
	}
	respond.WriteJSON(w, http.StatusOK, out)
}

type createEnvRequest struct {
	Slug string `json:"slug"`
}

func (h *ProjectsHandler) CreateEnv(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	body := middleware.BodyFromContext(r.Context())
	projectSlug := chi.URLParam(r, "slug")
	var req createEnvRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respond.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	env, err := h.Projects.Envs().Create(r.Context(), actx, projectSlug, req.Slug)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	respond.WriteJSON(w, http.StatusCreated, envDTO{
		Slug:      env.Slug,
		CreatedAt: env.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

type cloneEnvRequest struct {
	Dest string `json:"dest"`
}

func (h *ProjectsHandler) CloneEnv(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	body := middleware.BodyFromContext(r.Context())
	projectSlug := chi.URLParam(r, "slug")
	src := chi.URLParam(r, "env")
	var req cloneEnvRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respond.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	env, err := h.Projects.Envs().Clone(r.Context(), actx, projectSlug, src, req.Dest)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	respond.WriteJSON(w, http.StatusCreated, envDTO{
		Slug:      env.Slug,
		CreatedAt: env.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

func (h *ProjectsHandler) DeleteEnv(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	projectSlug := chi.URLParam(r, "slug")
	envSlug := chi.URLParam(r, "env")
	if err := h.Projects.Envs().Delete(r.Context(), actx, projectSlug, envSlug); err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
