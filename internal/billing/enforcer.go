package billing

import (
	"context"

	"github.com/google/uuid"
)

// Enforcer sits between services and Billing. Service layer holds a
// *Enforcer; when it's nil (selfhost) every Check is a no-op. The indirection
// keeps the hot path free of interface calls on selfhost.
type Enforcer struct {
	billing Billing
}

// NewEnforcer returns nil if b is nil or Enabled() is false — so selfhost
// wiring gets a nil *Enforcer and every guard is `if e != nil`.
func NewEnforcer(b Billing) *Enforcer {
	if b == nil || !b.Enabled() {
		return nil
	}
	return &Enforcer{billing: b}
}

// Check returns a *LimitExceededError if current >= the plan's limit for kind.
// Callers pass the current count cheaply (a SELECT COUNT(*) usually).
//
// Safe to call on a nil receiver: returns nil. This lets callers write
//
//	if err := e.Check(ctx, orgID, billing.ResourceProjects, count); err != nil { ... }
//
// without a separate nil check.
func (e *Enforcer) Check(ctx context.Context, orgID uuid.UUID, kind ResourceKind, current int) error {
	if e == nil {
		return nil
	}
	plan, err := e.billing.PlanFor(ctx, orgID)
	if err != nil {
		return err
	}
	limit := LimitFor(plan.Name, kind)
	if limit < 0 {
		return nil // unlimited
	}
	if current >= limit {
		return &LimitExceededError{Plan: plan.Name, Kind: kind, Current: current, Limit: limit}
	}
	return nil
}
