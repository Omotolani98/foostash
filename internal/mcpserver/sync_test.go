package mcpserver

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	fscrypto "github.com/Omotolani98/foostash/internal/crypto"
	"golang.org/x/crypto/ssh"
)

func TestParseDotenvFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("# comment\nA=one\nexport B=\"two\"\nC='three'\nignored\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	pairs, err := parseDotenvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if pairs["A"] != "one" || pairs["B"] != "two" || pairs["C"] != "three" || len(pairs) != 3 {
		t.Fatalf("pairs = %#v", pairs)
	}
}

func TestEnvironmentSyncEncryptsAndBulkUploadsChangedSecrets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	masterKey, err := fscrypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("FOOSTASH_MASTER_KEY", masterKey)
	engine, err := fscrypto.NewEngine(masterKey)
	if err != nil {
		t.Fatal(err)
	}
	keyPath := writeTestSSHKey(t)
	dotenv := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(dotenv, []byte("API_KEY=secret-value\nDB_URL=postgres://db\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var uploaded struct {
		Secrets []struct {
			Key        string `json:"key"`
			Ciphertext []byte `json:"ciphertext"`
			Nonce      []byte `json:"nonce"`
		} `json:"secrets"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/demo/envs/dev/secrets" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"secrets": []any{}})
		case http.MethodPut:
			if err := json.NewDecoder(r.Body).Decode(&uploaded); err != nil {
				t.Fatalf("decode upload: %v", err)
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	defer srv.Close()

	_, out, err := environmentSyncTool(context.Background(), nil, environmentSyncInput{
		ServerURL:  srv.URL,
		Project:    "demo",
		Env:        "dev",
		DotenvPath: dotenv,
		SSHKeyPath: keyPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Changed != 2 || out.TotalKeys != 2 {
		t.Fatalf("out = %+v", out)
	}
	if len(uploaded.Secrets) != 2 {
		t.Fatalf("uploaded %d secrets, want 2", len(uploaded.Secrets))
	}
	for _, s := range uploaded.Secrets {
		pt, err := engine.Decrypt(s.Ciphertext, s.Nonce)
		if err != nil {
			t.Fatalf("decrypt uploaded %s: %v", s.Key, err)
		}
		if s.Key == "API_KEY" && string(pt) != "secret-value" {
			t.Fatalf("decrypted API_KEY = %q", pt)
		}
		if string(s.Ciphertext) == string(pt) {
			t.Fatalf("%s was uploaded in plaintext", s.Key)
		}
	}
}

func writeTestSSHKey(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "foostash-mcp-test")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
