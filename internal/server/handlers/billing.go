package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Omotolani98/foostash/internal/billing"
	"github.com/Omotolani98/foostash/internal/server/middleware"
	"github.com/Omotolani98/foostash/internal/server/respond"
	"github.com/Omotolani98/foostash/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BillingHandler struct {
	Billing billing.Billing
	Auth    *service.Auth
	Pool    *pgxpool.Pool
}

type billingResponse struct {
	Plan     string `json:"plan"`
	Limits  limitsDTO `json:"limits"`
	Enabled bool   `json:"enabled"`
}

type limitsDTO struct {
	Projects          int `json:"projects"`
	EnvsPerProject    int `json:"envs_per_project"`
	SecretsPerEnv     int `json:"secrets_per_env"`
	Users             int `json:"users"`
	VaultSecrets      int `json:"vault_secrets"`
	AuditRetentionDays int `json:"audit_retention_days"`
}

func (h *BillingHandler) Get(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	plan, err := h.Billing.PlanFor(r.Context(), actx.OrgID)
	if err != nil {
		respond.WriteServiceError(w, err)
		return
	}
	respond.WriteJSON(w, http.StatusOK, billingResponse{
		Plan: plan.Name,
		Limits: limitsDTO{
			Projects:          plan.Limits.Projects,
			EnvsPerProject:    plan.Limits.EnvsPerProject,
			SecretsPerEnv:     plan.Limits.SecretsPerEnv,
			Users:             plan.Limits.Users,
			VaultSecrets:      plan.Limits.VaultSecrets,
			AuditRetentionDays: plan.Limits.AuditRetentionDays,
		},
		Enabled: h.Billing.Enabled(),
	})
}

type checkoutRequest struct {
	Variant string `json:"variant"`
}

func (h *BillingHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	actx := middleware.AuthFromContext(r.Context())
	body := middleware.BodyFromContext(r.Context())
	var req checkoutRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respond.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	url, err := h.Billing.CheckoutURL(r.Context(), actx.OrgID, req.Variant)
	if err != nil {
		if err == billing.ErrBillingDisabled {
			respond.WriteError(w, http.StatusNotFound, "billing_disabled", "Billing is not enabled for this deployment")
			return
		}
		if err == billing.ErrUnknownVariant {
			respond.WriteError(w, http.StatusBadRequest, "unknown_variant", err.Error())
			return
		}
		respond.WriteServiceError(w, err)
		return
	}
	type checkoutResponse struct {
		URL string `json:"url"`
	}
	respond.WriteJSON(w, http.StatusOK, checkoutResponse{URL: url})
}

func (h *BillingHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	body := middleware.BodyFromContext(r.Context())
	err := h.Billing.HandleWebhook(r.Context(), r.Header, body)
	if err != nil {
		if err == billing.ErrInvalidSignature {
			respond.WriteError(w, http.StatusUnauthorized, "invalid_signature", err.Error())
			return
		}
		respond.WriteServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}