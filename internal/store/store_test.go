package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Omotolani98/foostash/internal/crypto"
)

func testEngine(t *testing.T) *crypto.Engine {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	engine, err := crypto.NewEngine(key)
	if err != nil {
		t.Fatal(err)
	}
	return engine
}

func TestStoreRoundTrip(t *testing.T) {
	engine := testEngine(t)
	s := New(engine)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.enc")

	// save
	sf := NewSecretFile()
	sf.Secrets["DB_HOST"] = SecretEntry{Value: "localhost", Version: 1}
	sf.Secrets["DB_PORT"] = SecretEntry{Value: "5432", Version: 1}
	sf.History["DB_HOST"] = []HistoryEntry{{Version: 1, Value: "localhost"}}

	if err := s.Save(path, sf); err != nil {
		t.Fatal("save:", err)
	}

	// load
	loaded, err := s.Load(path)
	if err != nil {
		t.Fatal("load:", err)
	}

	if loaded.Secrets["DB_HOST"].Value != "localhost" {
		t.Errorf("expected localhost, got %s", loaded.Secrets["DB_HOST"].Value)
	}
	if loaded.Secrets["DB_PORT"].Value != "5432" {
		t.Errorf("expected 5432, got %s", loaded.Secrets["DB_PORT"].Value)
	}
	if len(loaded.History["DB_HOST"]) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(loaded.History["DB_HOST"]))
	}
}

func TestStoreLoadMissing(t *testing.T) {
	engine := testEngine(t)
	s := New(engine)

	sf, err := s.Load(filepath.Join(t.TempDir(), "nonexistent.enc"))
	if err != nil {
		t.Fatal("expected no error for missing file:", err)
	}
	if len(sf.Secrets) != 0 {
		t.Errorf("expected empty secrets, got %d", len(sf.Secrets))
	}
}

func TestStoreAtomicWrite(t *testing.T) {
	engine := testEngine(t)
	s := New(engine)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.enc")

	sf := NewSecretFile()
	sf.Secrets["KEY"] = SecretEntry{Value: "value", Version: 1}

	if err := s.Save(path, sf); err != nil {
		t.Fatal("save:", err)
	}

	// verify no .tmp file left behind
	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file should not exist after save")
	}
}
