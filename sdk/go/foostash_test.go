package foostash

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetAllAndCache(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(pullResponse{
			Project: "myapp", Environment: "prod",
			Secrets: map[string]string{"DB_HOST": "db.internal", "DB_PORT": "5432"},
			Version: 3,
		})
	}))
	defer srv.Close()

	c, err := New(Config{
		Server: srv.URL, Token: "test-token", ProjectID: "myapp", Env: "prod",
		CacheTTL: time.Minute,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	ctx := context.Background()
	got, err := c.GetAll(ctx)
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	if got["DB_HOST"] != "db.internal" {
		t.Fatalf("unexpected db host: %q", got["DB_HOST"])
	}

	if _, err := c.GetAll(ctx); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("expected cache hit on second call, server hits=%d", hits)
	}

	c.Refresh()
	if _, err := c.GetAll(ctx); err != nil {
		t.Fatalf("refresh call: %v", err)
	}
	if atomic.LoadInt32(&hits) != 2 {
		t.Fatalf("expected refetch after refresh, server hits=%d", hits)
	}
}

func TestGetMissingKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(pullResponse{Secrets: map[string]string{"A": "1"}})
	}))
	defer srv.Close()

	c, _ := New(Config{
		Server: srv.URL, Token: "t", ProjectID: "p", Env: "e", CacheTTL: time.Minute,
	})
	_, err := c.Get(context.Background(), "MISSING")
	if err != ErrKeyNotFound {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}
}

func TestMissingConfig(t *testing.T) {
	_, err := New(Config{})
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}
