package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Omotolani98/foostash/internal/server/middleware"
	"github.com/Omotolani98/foostash/internal/server/respond"
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/go-chi/chi/v5"
)

type SecretsHandler struct {
	Secrets *service.Secrets
}

type secretDTO struct {
	Key        string `json:"key"`
	Ciphertext []byte `json:"ciphertext"`
	Nonce      []byte `json:"nonce"`
	Version    int    `json:"version"`
	UpdatedAt  string `json:"updated_at"`
	UpdatedBy  string `json:"updated_by,omitempty"`
}

type setSecretRequest struct {
	Ciphertext []byte `json:"ciphertext"`
	Nonce      []byte `json:"nonce"`
}

func (h *SecretsHandler) Set(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	body := middleware.BodyFromContext(r.Context())
	project := chi.URLParam(r, "slug")
	env := chi.URLParam(r, "env")
	key := chi.URLParam(r, "key")

	var req setSecretRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respond.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	view, err := h.Secrets.Set(r.Context(), actx, project, env, key, req.Ciphertext, req.Nonce)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	respond.WriteJSON(w, http.StatusOK, toSecretDTO(view))
}

func (h *SecretsHandler) Get(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	project := chi.URLParam(r, "slug")
	env := chi.URLParam(r, "env")
	key := chi.URLParam(r, "key")
	view, err := h.Secrets.Get(r.Context(), actx, project, env, key)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	respond.WriteJSON(w, http.StatusOK, toSecretDTO(view))
}

type listSecretsResponse struct {
	Secrets []secretDTO `json:"secrets"`
}

func (h *SecretsHandler) List(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	project := chi.URLParam(r, "slug")
	env := chi.URLParam(r, "env")
	rows, err := h.Secrets.List(r.Context(), actx, project, env)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	out := listSecretsResponse{Secrets: make([]secretDTO, 0, len(rows))}
	for i := range rows {
		out.Secrets = append(out.Secrets, toSecretDTO(&rows[i]))
	}
	respond.WriteJSON(w, http.StatusOK, out)
}

func (h *SecretsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	project := chi.URLParam(r, "slug")
	env := chi.URLParam(r, "env")
	key := chi.URLParam(r, "key")
	if err := h.Secrets.Delete(r.Context(), actx, project, env, key); err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type historyDTO struct {
	Version    int    `json:"version"`
	Ciphertext []byte `json:"ciphertext"`
	Nonce      []byte `json:"nonce"`
	SetAt      string `json:"set_at"`
	SetBy      string `json:"set_by,omitempty"`
}

type historyResponse struct {
	History []historyDTO `json:"history"`
}

func (h *SecretsHandler) History(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	project := chi.URLParam(r, "slug")
	env := chi.URLParam(r, "env")
	key := chi.URLParam(r, "key")
	rows, err := h.Secrets.History(r.Context(), actx, project, env, key)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	out := historyResponse{History: make([]historyDTO, 0, len(rows))}
	for _, h := range rows {
		dto := historyDTO{
			Version:    h.Version,
			Ciphertext: h.Ciphertext,
			Nonce:      h.Nonce,
			SetAt:      h.SetAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if h.SetBy != nil {
			dto.SetBy = h.SetBy.String()
		}
		out.History = append(out.History, dto)
	}
	respond.WriteJSON(w, http.StatusOK, out)
}

type rollbackRequest struct {
	Version int `json:"version"`
}

func (h *SecretsHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	body := middleware.BodyFromContext(r.Context())
	project := chi.URLParam(r, "slug")
	env := chi.URLParam(r, "env")
	key := chi.URLParam(r, "key")

	var req rollbackRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			respond.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}
	if req.Version == 0 {
		if v := r.URL.Query().Get("version"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				respond.WriteError(w, http.StatusBadRequest, "invalid_argument", "version must be integer")
				return
			}
			req.Version = n
		}
	}
	view, err := h.Secrets.Rollback(r.Context(), actx, project, env, key, req.Version)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	respond.WriteJSON(w, http.StatusOK, toSecretDTO(view))
}

func toSecretDTO(v *service.SecretView) secretDTO {
	dto := secretDTO{
		Key:        v.Key,
		Ciphertext: v.Ciphertext,
		Nonce:      v.Nonce,
		Version:    v.Version,
		UpdatedAt:  v.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if v.UpdatedBy != nil {
		dto.UpdatedBy = v.UpdatedBy.String()
	}
	return dto
}
