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

type VaultHandler struct {
	Vault *service.Vault
}

type vaultDTO struct {
	Key        string `json:"key"`
	Ciphertext []byte `json:"ciphertext"`
	Nonce      []byte `json:"nonce"`
	Version    int    `json:"version"`
	UpdatedAt  string `json:"updated_at"`
	UpdatedBy  string `json:"updated_by,omitempty"`
}

type setVaultRequest struct {
	Ciphertext []byte `json:"ciphertext"`
	Nonce      []byte `json:"nonce"`
}

func (h *VaultHandler) Set(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	body := middleware.BodyFromContext(r.Context())
	key := chi.URLParam(r, "key")

	var req setVaultRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respond.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	view, err := h.Vault.Set(r.Context(), actx, key, req.Ciphertext, req.Nonce)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	respond.WriteJSON(w, http.StatusOK, toVaultDTO(view))
}

func (h *VaultHandler) Get(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	key := chi.URLParam(r, "key")
	view, err := h.Vault.Get(r.Context(), actx, key)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	respond.WriteJSON(w, http.StatusOK, toVaultDTO(view))
}

type listVaultResponse struct {
	Secrets []vaultDTO `json:"secrets"`
}

func (h *VaultHandler) List(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	rows, err := h.Vault.List(r.Context(), actx)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	out := listVaultResponse{Secrets: make([]vaultDTO, 0, len(rows))}
	for i := range rows {
		out.Secrets = append(out.Secrets, toVaultDTO(&rows[i]))
	}
	respond.WriteJSON(w, http.StatusOK, out)
}

func (h *VaultHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	key := chi.URLParam(r, "key")
	if err := h.Vault.Delete(r.Context(), actx, key); err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type vaultHistoryResponse struct {
	History []historyDTO `json:"history"`
}

func (h *VaultHandler) History(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	key := chi.URLParam(r, "key")
	rows, err := h.Vault.History(r.Context(), actx, key)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	out := vaultHistoryResponse{History: make([]historyDTO, 0, len(rows))}
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

func (h *VaultHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	body := middleware.BodyFromContext(r.Context())
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
	view, err := h.Vault.Rollback(r.Context(), actx, key, req.Version)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	respond.WriteJSON(w, http.StatusOK, toVaultDTO(view))
}

func toVaultDTO(v *service.VaultView) vaultDTO {
	dto := vaultDTO{
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
