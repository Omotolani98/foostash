package handler

import (
	"net/http"
	"time"

	"github.com/Omotolani98/foostash/internal/service"
	"github.com/go-chi/chi/v5"
)

type SecretsHandler struct {
	svc *service.SecretsService
}

func NewSecretsHandler(s *service.SecretsService) *SecretsHandler {
	return &SecretsHandler{svc: s}
}

type pullResp struct {
	Project     string            `json:"project"`
	Environment string            `json:"environment"`
	Secrets     map[string]string `json:"secrets"`
	Version     int               `json:"version"`
	PulledAt    time.Time         `json:"pulled_at"`
}

func (h *SecretsHandler) Pull(w http.ResponseWriter, r *http.Request) {
	auth := AuthFromContext(r.Context())
	if !auth.HasScope("secrets:read") {
		WriteError(w, service.ErrForbidden)
		return
	}
	slug := chi.URLParam(r, "slug")
	env := chi.URLParam(r, "env")

	res, err := h.svc.Pull(r.Context(), auth.OrgID, slug, env, auth.UserID, r.RemoteAddr, r.UserAgent())
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, pullResp{
		Project: slug, Environment: env, Secrets: res.Secrets, Version: res.Version,
		PulledAt: time.Now().UTC(),
	})
}

type setReq struct {
	Secrets map[string]string `json:"secrets"`
}

type setResp struct {
	Project     string   `json:"project"`
	Environment string   `json:"environment"`
	Set         []string `json:"set"`
	Version     int      `json:"version"`
}

func (h *SecretsHandler) Set(w http.ResponseWriter, r *http.Request) {
	auth := AuthFromContext(r.Context())
	if !auth.HasScope("secrets:write") {
		WriteError(w, service.ErrForbidden)
		return
	}
	slug := chi.URLParam(r, "slug")
	env := chi.URLParam(r, "env")

	var req setReq
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err)
		return
	}

	keys, version, err := h.svc.Set(r.Context(), auth.OrgID, slug, env, req.Secrets, auth.UserID, r.RemoteAddr, r.UserAgent())
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, setResp{
		Project: slug, Environment: env, Set: keys, Version: version,
	})
}

func (h *SecretsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	auth := AuthFromContext(r.Context())
	if !auth.HasScope("secrets:write") {
		WriteError(w, service.ErrForbidden)
		return
	}
	slug := chi.URLParam(r, "slug")
	env := chi.URLParam(r, "env")
	key := chi.URLParam(r, "key")

	if err := h.svc.Delete(r.Context(), auth.OrgID, slug, env, key, auth.UserID, r.RemoteAddr, r.UserAgent()); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
