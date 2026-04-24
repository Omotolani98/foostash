// Package billing isolates paid-tier logic from the rest of the server so
// self-hosted deployments can run without any billing code on the hot path.
//
// The server constructs exactly one Billing at boot via FromEnv. Self-hosters
// get a noop implementation (Enabled() == false); the foosta.sh SaaS wires the
// Lemon Squeezy implementation by setting FOOSTASH_BILLING=lemonsqueezy.
//
// Routes mounted under /v1/billing are only registered when Enabled() is true,
// and service-layer limit checks short-circuit on a nil *Enforcer. The goal is
// that toggling FOOSTASH_BILLING off removes every user-visible billing signal.
package billing

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/google/uuid"
)

// Plan names. Selfhost is sentinel for "no billing configured".
const (
	PlanSelfHosted = "self_hosted"
	PlanFree       = "free"
	PlanPro        = "pro"
	PlanTeam       = "team"
	PlanLifetime   = "lifetime"
)

// Limits express per-org caps. -1 means unlimited.
type Limits struct {
	Projects          int
	EnvsPerProject    int
	SecretsPerEnv     int
	Users             int
	VaultSecrets      int
	AuditRetentionDays int
}

// Unlimited returns Limits with every field set to -1.
func Unlimited() Limits {
	return Limits{-1, -1, -1, -1, -1, -1}
}

// Plan pairs a plan name with its enforced limits.
type Plan struct {
	Name   string
	Limits Limits
}

// Billing is the contract the server uses to resolve an org's plan, produce a
// checkout URL, and accept webhooks. Implementations must be safe for
// concurrent use.
type Billing interface {
	Enabled() bool
	PlanFor(ctx context.Context, orgID uuid.UUID) (Plan, error)
	CheckoutURL(ctx context.Context, orgID uuid.UUID, variant string) (string, error)
	HandleWebhook(ctx context.Context, headers http.Header, body []byte) error
}

// ErrBillingDisabled is returned by the noop implementation whenever a caller
// asks for something that only makes sense on the SaaS (checkout URL etc.).
// Handlers translate this to a 404 so self-hosters see no billing surface.
var ErrBillingDisabled = errors.New("billing disabled")

// ErrUnknownVariant is returned when CheckoutURL is called with a variant the
// current deployment does not know about.
var ErrUnknownVariant = errors.New("unknown billing variant")

// ErrInvalidSignature is returned by webhook handlers on HMAC mismatch.
var ErrInvalidSignature = errors.New("invalid webhook signature")

// FromEnv constructs a Billing implementation based on FOOSTASH_BILLING.
//
//	"" | "none" | "noop"     -> NoopBilling (self-host default)
//	"lemonsqueezy"           -> LemonSqueezyBilling, requires FOOSTASH_LS_* vars
//
// Any unrecognised value returns an error so misconfigured SaaS deployments
// fail loudly on boot instead of silently falling back to noop.
func FromEnv() (Billing, error) {
	switch os.Getenv("FOOSTASH_BILLING") {
	case "", "none", "noop":
		return NewNoop(), nil
	case "lemonsqueezy":
		return NewLemonSqueezy(LemonSqueezyConfig{
			APIKey:          os.Getenv("FOOSTASH_LS_API_KEY"),
			WebhookSecret:   os.Getenv("FOOSTASH_LS_WEBHOOK_SECRET"),
			StoreID:         os.Getenv("FOOSTASH_LS_STORE_ID"),
			VariantPro:      os.Getenv("FOOSTASH_LS_VARIANT_PRO"),
			VariantTeam:     os.Getenv("FOOSTASH_LS_VARIANT_TEAM"),
			VariantLifetime: os.Getenv("FOOSTASH_LS_VARIANT_LIFETIME"),
		})
	default:
		return nil, errors.New("FOOSTASH_BILLING: unknown value (expected '', 'noop', or 'lemonsqueezy')")
	}
}
