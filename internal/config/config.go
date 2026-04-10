package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Omotolani98/foostash/internal/crypto"
	"gopkg.in/yaml.v3"
)

type Defaults struct {
	Env    string `yaml:"env"`
	Format string `yaml:"format"`
}

type GlobalConfig struct {
	Version  int      `yaml:"version"`
	Defaults Defaults `yaml:"defaults"`
}

func globalConfigPath() (string, error) {
	dir, err := crypto.FoostashDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func LoadGlobal() (*GlobalConfig, error) {
	path, err := globalConfigPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultGlobalConfig(), nil
		}
		return nil, fmt.Errorf("read global config: %w", err)
	}
	var cfg GlobalConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse global config: %w", err)
	}
	applyGlobalDefaults(&cfg)
	return &cfg, nil
}

func SaveGlobal(cfg *GlobalConfig) error {
	path, err := globalConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal global config: %w", err)
	}
	return os.WriteFile(path, b, 0600)
}

func defaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		Version: 1,
		Defaults: Defaults{
			Env:    "dev",
			Format: "dotenv",
		},
	}
}

func applyGlobalDefaults(cfg *GlobalConfig) {
	if cfg.Defaults.Env == "" {
		cfg.Defaults.Env = "dev"
	}
	if cfg.Defaults.Format == "" {
		cfg.Defaults.Format = "dotenv"
	}
}
