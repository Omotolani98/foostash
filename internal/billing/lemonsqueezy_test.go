package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeResolver records the last SetPlan call so we can assert on webhook
// dispatch without touching a database.
type fakeResolver struct {
	plan       string
	setOrg     uuid.UUID
	setPlan    string
	setSubID   string
	setCustID  string
	setExpires *time.Time
}

func (f *fakeResolver) PlanByOrg(_ context.Context, _ uuid.UUID) (string, error) {
	return f.plan, nil
}

func (f *fakeResolver) SetPlan(_ context.Context, orgID uuid.UUID, plan, subID, custID string, expiresAt *time.Time) error {
	f.setOrg = orgID
	f.setPlan = plan
	f.setSubID = subID
	f.setCustID = custID
	f.setExpires = expiresAt
	return nil
}

func newTestLS(t *testing.T, r OrgResolver) *LemonSqueezyBilling {
	t.Helper()
	ls, err := NewLemonSqueezy(LemonSqueezyConfig{
		APIKey:          "test-key",
		WebhookSecret:   "test-secret",
		StoreID:         "42",
		VariantPro:      "var-pro",
		VariantTeam:     "var-team",
		VariantLifetime: "var-lifetime",
		Resolver:        r,
	})
	if err != nil {
		t.Fatalf("NewLemonSqueezy: %v", err)
	}
	return ls
}

func sign(secret, body string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(body))
	return hex.EncodeToString(m.Sum(nil))
}

func TestLS_RejectsMissingConfig(t *testing.T) {
	cases := []LemonSqueezyConfig{
		{WebhookSecret: "s", StoreID: "1"},
		{APIKey: "k", StoreID: "1"},
		{APIKey: "k", WebhookSecret: "s"},
	}
	for i, c := range cases {
		if _, err := NewLemonSqueezy(c); err == nil {
			t.Fatalf("case %d: expected error for missing config", i)
		}
	}
}

func TestLS_Webhook_InvalidSignature(t *testing.T) {
	ls := newTestLS(t, &fakeResolver{})
	body := []byte(`{"meta":{"event_name":"order_created"}}`)
	h := http.Header{}
	h.Set("X-Signature", "deadbeef")
	if err := ls.HandleWebhook(context.Background(), h, body); err != ErrInvalidSignature {
		t.Fatalf("err = %v, want ErrInvalidSignature", err)
	}
}

func TestLS_Webhook_SubscriptionCreated(t *testing.T) {
	r := &fakeResolver{}
	ls := newTestLS(t, r)

	orgID := uuid.New()
	body := `{
		"meta": {"event_name": "subscription_created", "custom_data": {"org_id": "` + orgID.String() + `"}},
		"data": {"attributes": {"variant_id": "var-pro", "subscription_id": "sub_1", "customer_id": "cus_1"}}
	}`
	h := http.Header{}
	h.Set("X-Signature", sign("test-secret", body))

	if err := ls.HandleWebhook(context.Background(), h, []byte(body)); err != nil {
		t.Fatalf("HandleWebhook: %v", err)
	}
	if r.setPlan != PlanPro {
		t.Fatalf("plan = %q, want %q", r.setPlan, PlanPro)
	}
	if r.setOrg != orgID {
		t.Fatalf("orgID mismatch")
	}
	if r.setSubID != "sub_1" || r.setCustID != "cus_1" {
		t.Fatalf("ids not recorded: sub=%q cust=%q", r.setSubID, r.setCustID)
	}
}

func TestLS_Webhook_Lifetime(t *testing.T) {
	r := &fakeResolver{}
	ls := newTestLS(t, r)

	orgID := uuid.New()
	body := `{
		"meta": {"event_name": "order_created", "custom_data": {"org_id": "` + orgID.String() + `"}},
		"data": {"attributes": {"variant_id": "var-lifetime"}}
	}`
	h := http.Header{}
	h.Set("X-Signature", sign("test-secret", body))

	if err := ls.HandleWebhook(context.Background(), h, []byte(body)); err != nil {
		t.Fatalf("HandleWebhook: %v", err)
	}
	if r.setPlan != PlanLifetime {
		t.Fatalf("plan = %q, want %q", r.setPlan, PlanLifetime)
	}
}

func TestLS_Webhook_Cancelled(t *testing.T) {
	r := &fakeResolver{}
	ls := newTestLS(t, r)

	orgID := uuid.New()
	body := `{
		"meta": {"event_name": "subscription_cancelled", "custom_data": {"org_id": "` + orgID.String() + `"}},
		"data": {"attributes": {"variant_id": "var-pro"}}
	}`
	h := http.Header{}
	h.Set("X-Signature", sign("test-secret", body))

	if err := ls.HandleWebhook(context.Background(), h, []byte(body)); err != nil {
		t.Fatalf("HandleWebhook: %v", err)
	}
	if r.setPlan != PlanFree {
		t.Fatalf("plan = %q, want %q", r.setPlan, PlanFree)
	}
}

func TestLS_Webhook_UnknownEventIgnored(t *testing.T) {
	r := &fakeResolver{}
	ls := newTestLS(t, r)

	body := `{"meta":{"event_name":"license_key_created","custom_data":{"org_id":"` + uuid.New().String() + `"}},"data":{"attributes":{}}}`
	h := http.Header{}
	h.Set("X-Signature", sign("test-secret", body))

	if err := ls.HandleWebhook(context.Background(), h, []byte(body)); err != nil {
		t.Fatalf("HandleWebhook: %v", err)
	}
	if r.setPlan != "" {
		t.Fatalf("unexpected SetPlan on unknown event: %q", r.setPlan)
	}
}

func TestLS_CheckoutURL_UnknownVariant(t *testing.T) {
	ls := newTestLS(t, nil)
	if _, err := ls.CheckoutURL(context.Background(), uuid.New(), "bogus"); err != ErrUnknownVariant {
		t.Fatalf("err = %v, want ErrUnknownVariant", err)
	}
}

func TestLS_PlanFor_NoResolverDefaultsFree(t *testing.T) {
	ls := newTestLS(t, nil)
	p, err := ls.PlanFor(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("PlanFor: %v", err)
	}
	if p.Name != PlanFree {
		t.Fatalf("plan = %q, want %q", p.Name, PlanFree)
	}
}
