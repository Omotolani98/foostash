package billing

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestNoop_Disabled(t *testing.T) {
	n := NewNoop()
	if n.Enabled() {
		t.Fatal("noop must not be enabled")
	}
}

func TestNoop_UnlimitedPlan(t *testing.T) {
	n := NewNoop()
	p, err := n.PlanFor(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("PlanFor: %v", err)
	}
	if p.Name != PlanSelfHosted {
		t.Fatalf("plan = %q, want %q", p.Name, PlanSelfHosted)
	}
	if p.Limits != Unlimited() {
		t.Fatalf("limits = %+v, want unlimited", p.Limits)
	}
}

func TestNoop_CheckoutDisabled(t *testing.T) {
	n := NewNoop()
	if _, err := n.CheckoutURL(context.Background(), uuid.New(), "pro"); !errors.Is(err, ErrBillingDisabled) {
		t.Fatalf("CheckoutURL err = %v, want ErrBillingDisabled", err)
	}
	if err := n.HandleWebhook(context.Background(), http.Header{}, nil); !errors.Is(err, ErrBillingDisabled) {
		t.Fatalf("HandleWebhook err = %v, want ErrBillingDisabled", err)
	}
}

func TestEnforcer_NilIsNoop(t *testing.T) {
	var e *Enforcer // selfhost passes nil
	if err := e.Check(context.Background(), uuid.New(), ResourceProjects, 1_000_000); err != nil {
		t.Fatalf("nil enforcer must allow: %v", err)
	}
}

func TestNewEnforcer_NilOnNoopBilling(t *testing.T) {
	if NewEnforcer(NewNoop()) != nil {
		t.Fatal("NewEnforcer on disabled billing must return nil")
	}
}
