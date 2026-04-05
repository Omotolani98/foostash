package handler

import (
	"net/http"

	"github.com/Omotolani98/foostash/internal/service"
	"github.com/go-chi/chi/v5"
)

type ProjectsHandler struct {
	svc *service.ProjectsService
}

func NewProjectsHandler(s *service.ProjectsService) *ProjectsHandler {
	return &ProjectsHandler{svc: s}
}

type projectResp struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type createProjectReq struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

func (h *ProjectsHandler) List(w http.ResponseWriter, r *http.Request) {
	auth := AuthFromContext(r.Context())
	rows, err := h.svc.List(r.Context(), auth.OrgID)
	if err != nil {
		WriteError(w, err)
		return
	}
	out := make([]projectResp, 0, len(rows))
	for _, p := range rows {
		out = append(out, projectResp{ID: p.ID, Slug: p.Slug, Name: p.Name})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"projects": out})
}

func (h *ProjectsHandler) Create(w http.ResponseWriter, r *http.Request) {
	auth := AuthFromContext(r.Context())
	var req createProjectReq
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err)
		return
	}
	p, err := h.svc.Create(r.Context(), auth.OrgID, req.Slug, req.Name)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, projectResp{ID: p.ID, Slug: p.Slug, Name: p.Name})
}

func (h *ProjectsHandler) Get(w http.ResponseWriter, r *http.Request) {
	auth := AuthFromContext(r.Context())
	slug := chi.URLParam(r, "slug")
	p, err := h.svc.Get(r.Context(), auth.OrgID, slug)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, projectResp{ID: p.ID, Slug: p.Slug, Name: p.Name})
}

func (h *ProjectsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	auth := AuthFromContext(r.Context())
	slug := chi.URLParam(r, "slug")
	if err := h.svc.Delete(r.Context(), auth.OrgID, slug); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type envResp struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

type createEnvReq struct {
	Slug string `json:"slug"`
}

func (h *ProjectsHandler) ListEnvs(w http.ResponseWriter, r *http.Request) {
	auth := AuthFromContext(r.Context())
	slug := chi.URLParam(r, "slug")
	rows, err := h.svc.ListEnvs(r.Context(), auth.OrgID, slug)
	if err != nil {
		WriteError(w, err)
		return
	}
	out := make([]envResp, 0, len(rows))
	for _, e := range rows {
		out = append(out, envResp{ID: e.ID, Slug: e.Slug})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"environments": out})
}

func (h *ProjectsHandler) CreateEnv(w http.ResponseWriter, r *http.Request) {
	auth := AuthFromContext(r.Context())
	slug := chi.URLParam(r, "slug")
	var req createEnvReq
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err)
		return
	}
	e, err := h.svc.CreateEnv(r.Context(), auth.OrgID, slug, req.Slug)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, envResp{ID: e.ID, Slug: e.Slug})
}

func (h *ProjectsHandler) DeleteEnv(w http.ResponseWriter, r *http.Request) {
	auth := AuthFromContext(r.Context())
	slug := chi.URLParam(r, "slug")
	env := chi.URLParam(r, "env")
	if err := h.svc.DeleteEnv(r.Context(), auth.OrgID, slug, env); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
