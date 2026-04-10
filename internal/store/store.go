package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Omotolani98/foostash/internal/crypto"
)

type SecretEntry struct {
	Value     string    `json:"value"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

type HistoryEntry struct {
	Version int       `json:"version"`
	Value   string    `json:"value"`
	SetAt   time.Time `json:"set_at"`
}

type SecretFile struct {
	Version   int                       `json:"version"`
	UpdatedAt time.Time                 `json:"updated_at"`
	Secrets   map[string]SecretEntry    `json:"secrets"`
	History   map[string][]HistoryEntry `json:"history"`
}

func NewSecretFile() *SecretFile {
	return &SecretFile{
		Version: 0,
		Secrets: make(map[string]SecretEntry),
		History: make(map[string][]HistoryEntry),
	}
}

type Store struct {
	engine *crypto.Engine
}

func New(engine *crypto.Engine) *Store {
	return &Store{engine: engine}
}

// Load reads and decrypts a .enc file. Returns an empty SecretFile if the file does not exist.
func (s *Store) Load(path string) (*SecretFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewSecretFile(), nil
		}
		return nil, fmt.Errorf("read secret file: %w", err)
	}

	if len(data) < 12 {
		return nil, fmt.Errorf("secret file too short")
	}

	// nonce is prepended to ciphertext
	nonce := data[:12]
	ciphertext := data[12:]

	plaintext, err := s.engine.Decrypt(ciphertext, nonce)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret file: %w", err)
	}

	var sf SecretFile
	if err := json.Unmarshal(plaintext, &sf); err != nil {
		return nil, fmt.Errorf("unmarshal secret file: %w", err)
	}
	if sf.Secrets == nil {
		sf.Secrets = make(map[string]SecretEntry)
	}
	if sf.History == nil {
		sf.History = make(map[string][]HistoryEntry)
	}
	return &sf, nil
}

// Save encrypts and writes a SecretFile atomically (write to tmp, then rename).
func (s *Store) Save(path string, sf *SecretFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	plaintext, err := json.Marshal(sf)
	if err != nil {
		return fmt.Errorf("marshal secret file: %w", err)
	}

	ciphertext, nonce, err := s.engine.Encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("encrypt secret file: %w", err)
	}

	// prepend nonce to ciphertext
	combined := make([]byte, 0, len(nonce)+len(ciphertext))
	combined = append(combined, nonce...)
	combined = append(combined, ciphertext...)

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, combined, 0600); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

// EnvPath returns the path to a project's environment .enc file.
func EnvPath(project, env string) (string, error) {
	dir, err := crypto.ProjectsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, project, env+".enc"), nil
}

// GlobalsPath returns the path to the globals.enc file.
func GlobalsPath() (string, error) {
	dir, err := crypto.FoostashDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "globals.enc"), nil
}

// ProjectDir returns the path to a project's directory under ~/.foostash/projects/.
func ProjectDir(project string) (string, error) {
	dir, err := crypto.ProjectsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, project), nil
}
