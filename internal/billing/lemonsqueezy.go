package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// LemonSqueezyConfig holds the credentials and variant IDs the SaaS needs to
// talk to Lemon Squeezy. All fields except APIKey and WebhookSecret may be
// empty: a variant that is not configured simply returns ErrUnknownVariant
// from CheckoutURL, which is useful during a partial rollout.
type LemonSqueezyConfig struct {
	APIKey          string
	WebhookSecret   string
	StoreID         string
	VariantPro      string
	VariantTeam     string
	VariantLifetime string
	// Resolver maps an org UUID to its plan/LS ids; production injects a DB-
	// backed implementation. Used by PlanFor and the webhook handler so this
	// package doesn't import `repo`.
	Resolver OrgResolver
	// HTTPClient is optional; defaults to a 10s-timeout client.
	HTTPClient *http.Client
	// Now is injectable for deterministic tests.
	Now func() time.Time
}

// OrgResolver is the tiny contract the LS implementation needs against
// persistent storage. Keeps `billing` from depending on `repo` directly.
type OrgResolver interface {
	// PlanByOrg returns the stored plan name for the given org, or PlanFree
	// if the org has never been linked to a subscription.
	PlanByOrg(ctx context.Context, orgID uuid.UUID) (string, error)
	// SetPlan writes a new plan (and optionally the LS subscription id) for
	// an org. Called from the webhook handler.
	SetPlan(ctx context.Context, orgID uuid.UUID, plan, lsSubscriptionID, lsCustomerID string, expiresAt *time.Time) error
}

// LemonSqueezyBilling is the SaaS-only implementation. Safe for concurrent use.
type LemonSqueezyBilling struct {
	cfg LemonSqueezyConfig
}

// NewLemonSqueezy validates the config and returns a ready implementation.
// APIKey + WebhookSecret + StoreID are required; without them webhook
// verification or checkout creation would silently no-op.
func NewLemonSqueezy(cfg LemonSqueezyConfig) (*LemonSqueezyBilling, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("lemonsqueezy: FOOSTASH_LS_API_KEY is required")
	}
	if cfg.WebhookSecret == "" {
		return nil, errors.New("lemonsqueezy: FOOSTASH_LS_WEBHOOK_SECRET is required")
	}
	if cfg.StoreID == "" {
		return nil, errors.New("lemonsqueezy: FOOSTASH_LS_STORE_ID is required")
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &LemonSqueezyBilling{cfg: cfg}, nil
}

func (*LemonSqueezyBilling) Enabled() bool { return true }

func (b *LemonSqueezyBilling) PlanFor(ctx context.Context, orgID uuid.UUID) (Plan, error) {
	if b.cfg.Resolver == nil {
		// No resolver wired: fall back to free. Keeps tests simple and
		// prevents nil-deref if the server wires billing before storage.
		return Plan{Name: PlanFree, Limits: PlanLimits[PlanFree]}, nil
	}
	name, err := b.cfg.Resolver.PlanByOrg(ctx, orgID)
	if err != nil {
		return Plan{}, err
	}
	if name == "" {
		name = PlanFree
	}
	limits, ok := PlanLimits[name]
	if !ok {
		limits = PlanLimits[PlanFree]
	}
	return Plan{Name: name, Limits: limits}, nil
}

// variantFor translates a public plan slug ("pro"|"team"|"lifetime") to the
// configured Lemon Squeezy variant id. Returns ErrUnknownVariant if unset.
func (b *LemonSqueezyBilling) variantFor(slug string) (string, error) {
	var v string
	switch slug {
	case "pro":
		v = b.cfg.VariantPro
	case "team":
		v = b.cfg.VariantTeam
	case "lifetime":
		v = b.cfg.VariantLifetime
	default:
		return "", ErrUnknownVariant
	}
	if v == "" {
		return "", ErrUnknownVariant
	}
	return v, nil
}

// CheckoutURL creates a one-off Lemon Squeezy checkout. The org id is embedded
// in checkout.custom so the webhook can reconcile without needing a local
// pre-record.
//
// API reference: https://docs.lemonsqueezy.com/api/checkouts/create-checkout
func (b *LemonSqueezyBilling) CheckoutURL(ctx context.Context, orgID uuid.UUID, slug string) (string, error) {
	variantID, err := b.variantFor(slug)
	if err != nil {
		return "", err
	}

	payload := map[string]any{
		"data": map[string]any{
			"type": "checkouts",
			"attributes": map[string]any{
				"checkout_data": map[string]any{
					"custom": map[string]string{"org_id": orgID.String()},
				},
			},
			"relationships": map[string]any{
				"store":   map[string]any{"data": map[string]string{"type": "stores", "id": b.cfg.StoreID}},
				"variant": map[string]any{"data": map[string]string{"type": "variants", "id": variantID}},
			},
		},
	}

	buf, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal checkout payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.lemonsqueezy.com/v1/checkouts", bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+b.cfg.APIKey)
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := b.cfg.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create checkout: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("lemonsqueezy %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Data struct {
			Attributes struct {
				URL string `json:"url"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse checkout response: %w", err)
	}
	if parsed.Data.Attributes.URL == "" {
		return "", errors.New("lemonsqueezy: checkout response missing url")
	}
	return parsed.Data.Attributes.URL, nil
}

// HandleWebhook verifies the signature and dispatches recognised events to the
// resolver. Unknown events return nil so LS doesn't retry indefinitely.
//
// Signature header: X-Signature is hex(hmac_sha256(body, secret)).
func (b *LemonSqueezyBilling) HandleWebhook(ctx context.Context, h http.Header, body []byte) error {
	if !b.verifySignature(h.Get("X-Signature"), body) {
		return ErrInvalidSignature
	}
	if b.cfg.Resolver == nil {
		// Accept the webhook (valid signature) but nothing to update. This
		// happens in tests; never in production.
		return nil
	}

	var evt webhookEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		return fmt.Errorf("parse webhook body: %w", err)
	}

	orgIDStr, ok := evt.Meta.CustomData["org_id"]
	if !ok || orgIDStr == "" {
		return errors.New("webhook missing custom_data.org_id")
	}
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		return fmt.Errorf("webhook org_id: %w", err)
	}

	name := evt.Meta.EventName
	attrs := evt.Data.Attributes
	plan, expires := b.planFromEvent(name, attrs)
	if plan == "" {
		return nil // event we don't care about
	}

	return b.cfg.Resolver.SetPlan(ctx, orgID, plan, attrs.SubscriptionID, attrs.CustomerID, expires)
}

// verifySignature uses hmac.Equal to avoid timing leaks.
func (b *LemonSqueezyBilling) verifySignature(sigHeader string, body []byte) bool {
	if sigHeader == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(b.cfg.WebhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sigHeader))
}

// planFromEvent maps an LS event to (plan, expiresAt). Returns ("", nil) for
// events that don't affect plan state.
func (b *LemonSqueezyBilling) planFromEvent(name string, a webhookAttributes) (string, *time.Time) {
	switch name {
	case "subscription_created", "subscription_updated", "subscription_resumed":
		plan := variantToPlan(a.VariantID, b.cfg)
		if plan == "" {
			return "", nil
		}
		return plan, a.EndsAt
	case "subscription_cancelled", "subscription_expired", "subscription_paused":
		return PlanFree, nil
	case "order_created":
		// Lifetime comes through as a one-time order.
		if a.VariantID == b.cfg.VariantLifetime && b.cfg.VariantLifetime != "" {
			return PlanLifetime, nil
		}
		return "", nil
	default:
		return "", nil
	}
}

func variantToPlan(variantID string, cfg LemonSqueezyConfig) string {
	switch variantID {
	case cfg.VariantPro:
		if cfg.VariantPro == "" {
			return ""
		}
		return PlanPro
	case cfg.VariantTeam:
		if cfg.VariantTeam == "" {
			return ""
		}
		return PlanTeam
	case cfg.VariantLifetime:
		if cfg.VariantLifetime == "" {
			return ""
		}
		return PlanLifetime
	}
	return ""
}

// --- webhook wire types --------------------------------------------------

type webhookEvent struct {
	Meta struct {
		EventName  string            `json:"event_name"`
		CustomData map[string]string `json:"custom_data"`
	} `json:"meta"`
	Data struct {
		Attributes webhookAttributes `json:"attributes"`
	} `json:"data"`
}

type webhookAttributes struct {
	VariantID      string     `json:"variant_id"`
	SubscriptionID string     `json:"subscription_id"`
	CustomerID     string     `json:"customer_id"`
	EndsAt         *time.Time `json:"ends_at"`
}
