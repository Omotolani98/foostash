package service

import (
	"context"
	"errors"
	"testing"
)

// These tests construct services with nil repos: the admin gate runs before
// any repository call, so a non-admin caller must get ErrForbidden without
// touching the database. A missing gate would panic on the nil repos and
// fail the test.

func TestProjectsCreateRequiresAdmin(t *testing.T) {
	p := NewProjects(nil)
	for _, actx := range []*AuthContext{nil, {Role: "developer"}} {
		if _, err := p.Create(context.Background(), actx, "demo"); !errors.Is(err, ErrForbidden) {
			t.Fatalf("actx=%v: err = %v, want ErrForbidden", actx, err)
		}
	}
}

func TestEnvironmentsCreateRequiresAdmin(t *testing.T) {
	envs := NewProjects(nil).Envs()
	for _, actx := range []*AuthContext{nil, {Role: "developer"}} {
		if _, err := envs.Create(context.Background(), actx, "demo", "staging"); !errors.Is(err, ErrForbidden) {
			t.Fatalf("actx=%v: err = %v, want ErrForbidden", actx, err)
		}
	}
}

func TestEnvironmentsCloneRequiresAdmin(t *testing.T) {
	envs := NewProjects(nil).Envs()
	for _, actx := range []*AuthContext{nil, {Role: "developer"}} {
		if _, err := envs.Clone(context.Background(), actx, "demo", "dev", "staging"); !errors.Is(err, ErrForbidden) {
			t.Fatalf("actx=%v: err = %v, want ErrForbidden", actx, err)
		}
	}
}
