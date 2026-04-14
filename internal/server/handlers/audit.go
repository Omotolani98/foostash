package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Omotolani98/foostash/internal/server/middleware"
	"github.com/Omotolani98/foostash/internal/server/respond"
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/google/uuid"
)

type AuditHandler struct {
	Audit *service.Audit
}

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	q := service.AuditQuery{}
	if v := r.URL.Query().Get("user"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			respond.WriteError(w, http.StatusBadRequest, "invalid_argument", "bad user id")
			return
		}
		q.UserID = &id
	}
	q.Action = r.URL.Query().Get("action")
	if v := r.URL.Query().Get("since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			respond.WriteError(w, http.StatusBadRequest, "invalid_argument", "since must be RFC3339")
			return
		}
		q.Since = &t
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			respond.WriteError(w, http.StatusBadRequest, "invalid_argument", "limit must be positive integer")
			return
		}
		q.Limit = n
	}
	out, err := h.Audit.Query(r.Context(), actx, q)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	respond.WriteJSON(w, http.StatusOK, map[string]any{"entries": out})
}
