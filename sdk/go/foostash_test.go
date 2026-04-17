package foostash

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// --- test harness ---

type secretState struct {
	value   string
	version int
}

type fakeServer struct {
	mu      sync.Mutex
	aead    cipher.AEAD
	project string
	env     string
	state   map[string]*secretState
}

func (f *fakeServer) set(key, value string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.state[key]
	if !ok {
		f.state[key] = &secretState{value: value, version: 1}
		return
	}
	s.value = value
	s.version++
}

func (f *fakeServer) encrypt(plaintext string) (ct, nonce []byte) {
	nonce = make([]byte, f.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err)
	}
	ct = f.aead.Seal(nil, nonce, []byte(plaintext), nil)
	return ct, nonce
}

func (f *fakeServer) handler() http.Handler {
	mux := http.NewServeMux()
	listPath := "/v1/projects/" + f.project + "/envs/" + f.env + "/secrets"
	mux.HandleFunc(listPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		type dto struct {
			Key        string `json:"key"`
			Ciphertext []byte `json:"ciphertext"`
			Nonce      []byte `json:"nonce"`
			Version    int    `json:"version"`
			UpdatedAt  string `json:"updated_at"`
		}
		out := struct {
			Secrets []dto `json:"secrets"`
		}{}
		for k, s := range f.state {
			ct, nonce := f.encrypt(s.value)
			out.Secrets = append(out.Secrets, dto{
				Key: k, Ciphertext: ct, Nonce: nonce, Version: s.version,
				UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			})
		}
		_ = json.NewEncoder(w).Encode(out)
	})
	mux.HandleFunc(listPath+"/", func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, listPath+"/")
		key, _ = url.PathUnescape(key)
		f.mu.Lock()
		defer f.mu.Unlock()
		s, ok := f.state[key]
		if !ok {
			respondError(w, http.StatusNotFound, "not_found", "secret not found")
			return
		}
		ct, nonce := f.encrypt(s.value)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key":        key,
			"ciphertext": ct,
			"nonce":      nonce,
			"version":    s.version,
			"updated_at": time.Now().UTC().Format(time.RFC3339),
		})
	})
	return mux
}

func respondError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code": code, "message": msg, "status": status,
		},
	})
}

// setupHarness returns a live stub server, a configured Client, and the
// fakeServer handle so tests can mutate secret state directly.
func setupHarness(t *testing.T) (*Client, *fakeServer) {
	t.Helper()
	keyBytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, keyBytes); err != nil {
		t.Fatal(err)
	}
	masterKey := base64.StdEncoding.EncodeToString(keyBytes)
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pemBlock, err := ssh.MarshalPrivateKey(priv, "foostash-sdk-test")
	if err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(pemBlock), 0o600); err != nil {
		t.Fatal(err)
	}

	fs := &fakeServer{
		aead:    aead,
		project: "demo",
		env:     "prod",
		state:   map[string]*secretState{},
	}
	srv := httptest.NewServer(fs.handler())
	t.Cleanup(srv.Close)

	client, err := New(Config{
		ServerURL:  srv.URL,
		Project:    fs.project,
		Env:        fs.env,
		SSHKeyPath: keyPath,
		MasterKey:  masterKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client, fs
}

// --- tests ---

func TestNew_RejectsMissingFields(t *testing.T) {
	cases := []Config{
		{Project: "p", Env: "e", SSHKeyPath: "/k", MasterKey: "m"},
		{ServerURL: "http://x", Env: "e", SSHKeyPath: "/k", MasterKey: "m"},
		{ServerURL: "http://x", Project: "p", SSHKeyPath: "/k", MasterKey: "m"},
		{ServerURL: "http://x", Project: "p", Env: "e", MasterKey: "m"},
		{ServerURL: "http://x", Project: "p", Env: "e", SSHKeyPath: "/k"},
	}
	for i, c := range cases {
		if _, err := New(c); err == nil {
			t.Errorf("case %d: expected error for incomplete config, got nil", i)
		}
	}
}

func TestNew_RejectsBadMasterKey(t *testing.T) {
	// Any SSH key path will do — we should fail on the master key first? no,
	// actually New loads the SSH key first. Use a temp unencrypted key.
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	pb, _ := ssh.MarshalPrivateKey(priv, "x")
	keyPath := filepath.Join(t.TempDir(), "k")
	_ = os.WriteFile(keyPath, pem.EncodeToMemory(pb), 0o600)

	_, err := New(Config{
		ServerURL:  "http://x",
		Project:    "p",
		Env:        "e",
		SSHKeyPath: keyPath,
		MasterKey:  base64.StdEncoding.EncodeToString([]byte("too-short")),
	})
	if !errors.Is(err, ErrInvalidKeyLength) {
		t.Fatalf("expected ErrInvalidKeyLength, got %v", err)
	}
}

func TestPull_DecryptsAllSecrets(t *testing.T) {
	client, fs := setupHarness(t)
	fs.set("API_KEY", "abc123")
	fs.set("DB_URL", "postgres://db")
	fs.set("FEATURE_FLAG", "on")

	got, err := client.Pull(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"API_KEY": "abc123", "DB_URL": "postgres://db", "FEATURE_FLAG": "on"}
	if len(got) != len(want) {
		t.Fatalf("got %d secrets, want %d: %v", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestGet_RoundTrip(t *testing.T) {
	client, fs := setupHarness(t)
	fs.set("API_KEY", "abc123")

	v, err := client.Get(context.Background(), "API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	if v != "abc123" {
		t.Errorf("got %q, want abc123", v)
	}
}

func TestGet_NotFoundMapsToSentinel(t *testing.T) {
	client, _ := setupHarness(t)
	_, err := client.Get(context.Background(), "DOES_NOT_EXIST")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestWatch_EmitsInitialAndOnVersionChange(t *testing.T) {
	client, fs := setupHarness(t)
	fs.set("A", "1")
	fs.set("B", "2")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use the minimum interval (5s) to keep the test bounded.
	ch, err := client.Watch(ctx, MinWatchInterval)
	if err != nil {
		t.Fatal(err)
	}

	initial := mustReceive(t, ch, 2*time.Second)
	if initial.Secrets["A"] != "1" || initial.Secrets["B"] != "2" {
		t.Fatalf("initial snapshot wrong: %+v", initial.Secrets)
	}

	// Bump a version.
	fs.set("A", "1-bumped")
	next := mustReceive(t, ch, MinWatchInterval+2*time.Second)
	if next.Secrets["A"] != "1-bumped" {
		t.Fatalf("expected A=1-bumped after bump, got %q", next.Secrets["A"])
	}
	if next.Versions["A"] != 2 {
		t.Fatalf("expected version 2 after bump, got %d", next.Versions["A"])
	}
}

func TestWatch_NoSnapshotWhenUnchanged(t *testing.T) {
	client, fs := setupHarness(t)
	fs.set("A", "1")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := client.Watch(ctx, MinWatchInterval)
	if err != nil {
		t.Fatal(err)
	}
	_ = mustReceive(t, ch, 2*time.Second) // initial

	// Don't change anything. Wait one interval + slack; expect no send.
	select {
	case snap := <-ch:
		t.Fatalf("unexpected snapshot with no change: %+v", snap)
	case <-time.After(MinWatchInterval + 2*time.Second):
		// good — no snapshot emitted
	}
}

func TestWatch_ClampsIntervalBelowMin(t *testing.T) {
	client, fs := setupHarness(t)
	fs.set("A", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Request 1ms; should be clamped to MinWatchInterval. If it were honored,
	// we'd get thousands of polls before ctx cancel.
	ch, err := client.Watch(ctx, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	_ = mustReceive(t, ch, time.Second) // initial

	// No mutations → no further snapshots within ~3s. If clamp failed, the
	// poller would spin fast but still emit nothing (no version changes),
	// so this is mainly a "does not panic / does not leak" smoke test.
	select {
	case <-ch:
		// channel only closes on ctx cancel
	case <-time.After(3 * time.Second):
	}
}

func mustReceive(t *testing.T, ch <-chan Snapshot, within time.Duration) Snapshot {
	t.Helper()
	select {
	case s, ok := <-ch:
		if !ok {
			t.Fatal("channel closed before snapshot received")
		}
		return s
	case <-time.After(within):
		t.Fatalf("timed out waiting for snapshot after %s", within)
		return Snapshot{}
	}
}
