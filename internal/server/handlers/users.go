package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Omotolani98/foostash/internal/server/middleware"
	"github.com/Omotolani98/foostash/internal/server/respond"
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type UsersHandler struct {
	Users *service.Users
}

func (h *UsersHandler) List(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	out, err := h.Users.List(r.Context(), actx)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	respond.WriteJSON(w, http.StatusOK, map[string]any{"users": out})
}

type changeRoleRequest struct {
	Role string `json:"role"`
}

func (h *UsersHandler) ChangeRole(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.WriteError(w, http.StatusBadRequest, "invalid_argument", "bad user id")
		return
	}
	var req changeRoleRequest
	if err := json.Unmarshal(middleware.BodyFromContext(r.Context()), &req); err != nil {
		respond.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	out, err := h.Users.ChangeRole(r.Context(), actx, id, req.Role)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	respond.WriteJSON(w, http.StatusOK, out)
}

func (h *UsersHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.WriteError(w, http.StatusBadRequest, "invalid_argument", "bad user id")
		return
	}
	if err := h.Users.Revoke(r.Context(), actx, id); err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
