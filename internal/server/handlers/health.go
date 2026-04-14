package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/Omotolani98/foostash/internal/server/respond"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	Pool    *pgxpool.Pool
	Version string
}

type healthResponse struct {
	Status   string `json:"status"`
	Version  string `json:"version,omitempty"`
	Database string `json:"database"`
}

// Get reports service status. Pings the DB with a short timeout so
// orchestrators can distinguish "process up" from "usable".
func (h *HealthHandler) Get(w http.ResponseWriter, r *http.Request) {
	pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	db := "ok"
	status := http.StatusOK
	if err := h.Pool.Ping(pingCtx); err != nil {
		db = "unreachable"
		status = http.StatusServiceUnavailable
	}

	respond.WriteJSON(w, status, healthResponse{
		Status:   statusText(status),
		Version:  h.Version,
		Database: db,
	})
}

func statusText(code int) string {
	if code == http.StatusOK {
		return "ok"
	}
	return "degraded"
}
