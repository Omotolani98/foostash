package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ProjectConfig struct {
	Project      string   `yaml:"project"`
	DefaultEnv   string   `yaml:"default_env"`
	Environments []string `yaml:"environments"`
}

func LoadProject(dir string) (*ProjectConfig, error) {
	path := filepath.Join(dir, ".foostash.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read .foostash.yaml: %w", err)
	}
	var cfg ProjectConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse .foostash.yaml: %w", err)
	}
	if cfg.DefaultEnv == "" {
		cfg.DefaultEnv = "dev"
	}
	return &cfg, nil
}

func SaveProject(dir string, cfg *ProjectConfig) error {
	path := filepath.Join(dir, ".foostash.yaml")
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal .foostash.yaml: %w", err)
	}
	return os.WriteFile(path, b, 0644)
}
