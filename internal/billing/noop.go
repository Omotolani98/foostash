package billing

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// NoopBilling is the selfhost implementation. Every org is on PlanSelfHosted
// with Unlimited() limits; the checkout + webhook surfaces return
// ErrBillingDisabled so handlers can 404 them out.
//
// The goal: a selfhost binary built without --FOOSTASH_BILLING set has no
// billing-visible routes and no limit checks in the service layer hot path.
type NoopBilling struct{}

// NewNoop returns a NoopBilling. Stateless; safe to share.
func NewNoop() *NoopBilling { return &NoopBilling{} }

func (*NoopBilling) Enabled() bool { return false }

func (*NoopBilling) PlanFor(_ context.Context, _ uuid.UUID) (Plan, error) {
	return Plan{Name: PlanSelfHosted, Limits: Unlimited()}, nil
}

func (*NoopBilling) CheckoutURL(_ context.Context, _ uuid.UUID, _ string) (string, error) {
	return "", ErrBillingDisabled
}

func (*NoopBilling) HandleWebhook(_ context.Context, _ http.Header, _ []byte) error {
	return ErrBillingDisabled
}
