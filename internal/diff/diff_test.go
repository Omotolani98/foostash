package diff

import (
	"testing"
)

func TestCompareAllCategories(t *testing.T) {
	left := map[string]string{
		"DB_HOST":    "localhost",
		"DB_PORT":    "5432",
		"DEBUG_MODE": "true",
		"APP_NAME":   "myapp",
	}
	right := map[string]string{
		"DB_HOST":    "staging-db",
		"DB_PORT":    "5432",
		"SENTRY_DSN": "dsn://...",
		"APP_NAME":   "myapp",
	}

	result := Compare(left, right)

	// only in left
	if len(result.OnlyLeft) != 1 || result.OnlyLeft[0] != "DEBUG_MODE" {
		t.Errorf("OnlyLeft: expected [DEBUG_MODE], got %v", result.OnlyLeft)
	}

	// only in right
	if len(result.OnlyRight) != 1 || result.OnlyRight[0] != "SENTRY_DSN" {
		t.Errorf("OnlyRight: expected [SENTRY_DSN], got %v", result.OnlyRight)
	}

	// different
	if len(result.Different) != 1 {
		t.Errorf("Different: expected 1 entry, got %d", len(result.Different))
	}
	if vals, ok := result.Different["DB_HOST"]; !ok || vals[0] != "localhost" || vals[1] != "staging-db" {
		t.Errorf("Different[DB_HOST]: expected [localhost, staging-db], got %v", vals)
	}

	// identical
	if len(result.Identical) != 2 {
		t.Errorf("Identical: expected 2, got %d", len(result.Identical))
	}
}

func TestCompareEmpty(t *testing.T) {
	result := Compare(map[string]string{}, map[string]string{})

	if len(result.OnlyLeft) != 0 || len(result.OnlyRight) != 0 ||
		len(result.Different) != 0 || len(result.Identical) != 0 {
		t.Error("expected all empty for empty inputs")
	}
}

func TestCompareAllIdentical(t *testing.T) {
	m := map[string]string{"A": "1", "B": "2"}
	result := Compare(m, m)

	if len(result.OnlyLeft) != 0 || len(result.OnlyRight) != 0 || len(result.Different) != 0 {
		t.Error("expected no differences for identical maps")
	}
	if len(result.Identical) != 2 {
		t.Errorf("expected 2 identical, got %d", len(result.Identical))
	}
}
