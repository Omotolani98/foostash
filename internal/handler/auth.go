package handler

import (
	"net/http"

	"github.com/Omotolani98/foostash/internal/service"
	"github.com/go-chi/chi/v5"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(s *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: s}
}

type registerReq struct {
	OrgName  string `json:"org_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResp struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
	Email  string `json:"email"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err)
		return
	}
	res, err := h.svc.Register(r.Context(), service.RegisterInput{
		OrgName: req.OrgName, Email: req.Email, Password: req.Password,
	})
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, authResp{
		Token: res.Token, UserID: res.User.ID, OrgID: res.Org.ID, Email: res.User.Email,
	})
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err)
		return
	}
	res, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, authResp{
		Token: res.Token, UserID: res.User.ID, OrgID: res.Org.ID, Email: res.User.Email,
	})
}

type createAPIKeyReq struct {
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

type createAPIKeyResp struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Prefix string   `json:"prefix"`
	Scopes []string `json:"scopes"`
	Key    string   `json:"key"`
}

func (h *AuthHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	auth := AuthFromContext(r.Context())
	if auth == nil {
		WriteError(w, service.ErrUnauthorized)
		return
	}
	var req createAPIKeyReq
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err)
		return
	}
	created, err := h.svc.CreateAPIKey(r.Context(), auth.UserID, auth.OrgID, req.Name, req.Scopes)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, createAPIKeyResp{
		ID: created.Row.ID, Name: req.Name, Prefix: created.Row.KeyPrefix,
		Scopes: created.Row.Scopes, Key: created.RawKey,
	})
}

type apiKeyResp struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Prefix string   `json:"prefix"`
	Scopes []string `json:"scopes"`
}

func (h *AuthHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	auth := AuthFromContext(r.Context())
	if auth == nil {
		WriteError(w, service.ErrUnauthorized)
		return
	}
	rows, err := h.svc.ListAPIKeys(r.Context(), auth.UserID)
	if err != nil {
		WriteError(w, err)
		return
	}
	out := make([]apiKeyResp, 0, len(rows))
	for _, row := range rows {
		name := ""
		if row.Name.Valid {
			name = row.Name.String
		}
		out = append(out, apiKeyResp{ID: row.ID, Name: name, Prefix: row.KeyPrefix, Scopes: row.Scopes})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"api_keys": out})
}

func (h *AuthHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	auth := AuthFromContext(r.Context())
	if auth == nil {
		WriteError(w, service.ErrUnauthorized)
		return
	}
	id := chi.URLParam(r, "keyID")
	if err := h.svc.RevokeAPIKey(r.Context(), auth.UserID, id); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
