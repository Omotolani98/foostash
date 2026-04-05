package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type GlobalConfig struct {
	Server string `yaml:"server"`
	Token  string `yaml:"token"`
}

type ProjectConfig struct {
	Project    string `yaml:"project"`
	DefaultEnv string `yaml:"default_env"`
}

func globalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".foostash", "config.yaml"), nil
}

func LoadGlobalConfig() (*GlobalConfig, error) {
	path, err := globalConfigPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &GlobalConfig{Server: "http://localhost:8400"}, nil
		}
		return nil, err
	}
	var cfg GlobalConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse global config: %w", err)
	}
	if cfg.Server == "" {
		cfg.Server = "http://localhost:8400"
	}
	return &cfg, nil
}

func SaveGlobalConfig(cfg *GlobalConfig) error {
	path, err := globalConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

func LoadProjectConfig() (*ProjectConfig, error) {
	b, err := os.ReadFile(".foostash.yaml")
	if err != nil {
		return nil, err
	}
	var cfg ProjectConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse .foostash.yaml: %w", err)
	}
	return &cfg, nil
}

func SaveProjectConfig(cfg *ProjectConfig) error {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(".foostash.yaml", b, 0644)
}
