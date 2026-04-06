package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/Omotolani98/foostash/internal/service"
)

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
	if body != nil {
		if err := json.NewEncoder(w).Encode(body); err != nil {
			log.Printf("write json: %v", err)
		}
	}
}

func WriteError(w http.ResponseWriter, err error) {
	code, status := mapError(err)
	WriteJSON(w, status, ErrorEnvelope{Error: ErrorBody{
		Code:    code,
		Message: err.Error(),
		Status:  status,
	}})
}

func mapError(err error) (string, int) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		return "not_found", http.StatusNotFound
	case errors.Is(err, service.ErrConflict):
		return "conflict", http.StatusConflict
	case errors.Is(err, service.ErrUnauthorized):
		return "unauthorized", http.StatusUnauthorized
	case errors.Is(err, service.ErrForbidden):
		return "forbidden", http.StatusForbidden
	case errors.Is(err, service.ErrInvalidCredentials):
		return "unauthorized", http.StatusUnauthorized
	case errors.Is(err, service.ErrValidation):
		return "bad_request", http.StatusBadRequest
	case errors.Is(err, service.ErrPlanLimitReached):
		return "forbidden", http.StatusForbidden
	case errors.Is(err, service.ErrRateLimited):
		return "rate_limited", http.StatusTooManyRequests
	default:
		log.Printf("internal error: %v", err)
		return "internal", http.StatusInternalServerError
	}
}

func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("%w: invalid request body: %s", service.ErrValidation, err.Error())
	}
	if dec.More() {
		return fmt.Errorf("%w: request body must contain a single JSON object", service.ErrValidation)
	}
	return nil
}
