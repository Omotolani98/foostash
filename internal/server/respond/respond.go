// Package respond holds shared HTTP response + error mapping helpers.
package respond

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Omotolani98/foostash/internal/service"
	"github.com/Omotolani98/foostash/internal/sshauth"
)

// ErrorEnvelope matches the shape documented in ARCHITECTURE.md §6.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("write json", "err", err)
	}
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorEnvelope{Error: ErrorBody{Code: code, Message: message, Status: status}})
}

// WriteServiceError translates a domain error to the appropriate HTTP response.
// Unknown errors surface as 500.
func WriteServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidArgument):
		WriteError(w, http.StatusBadRequest, "invalid_argument", err.Error())
	case errors.Is(err, service.ErrOrgExists):
		WriteError(w, http.StatusConflict, "org_exists", err.Error())
	case errors.Is(err, service.ErrUnknownKey):
		WriteError(w, http.StatusUnauthorized, "unknown_key", err.Error())
	case errors.Is(err, service.ErrInviteNotFound):
		WriteError(w, http.StatusNotFound, "invite_not_found", err.Error())
	case errors.Is(err, service.ErrInviteExpired):
		WriteError(w, http.StatusGone, "expired_invite", err.Error())
	case errors.Is(err, service.ErrInviteAlreadyUsed):
		WriteError(w, http.StatusConflict, "invite_used", err.Error())
	case errors.Is(err, service.ErrForbidden):
		WriteError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, service.ErrProjectNotFound):
		WriteError(w, http.StatusNotFound, "project_not_found", err.Error())
	case errors.Is(err, service.ErrProjectExists):
		WriteError(w, http.StatusConflict, "project_exists", err.Error())
	case errors.Is(err, service.ErrEnvNotFound):
		WriteError(w, http.StatusNotFound, "env_not_found", err.Error())
	case errors.Is(err, service.ErrEnvExists):
		WriteError(w, http.StatusConflict, "env_exists", err.Error())
	case errors.Is(err, sshauth.ErrExpiredRequest):
		WriteError(w, http.StatusUnauthorized, "expired_request", err.Error())
	case errors.Is(err, sshauth.ErrBadSignature), errors.Is(err, sshauth.ErrBadTimestamp):
		WriteError(w, http.StatusUnauthorized, "bad_signature", err.Error())
	default:
		slog.Error("unhandled error", "err", err)
		WriteError(w, http.StatusInternalServerError, "internal", "internal error")
	}
}
