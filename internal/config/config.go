package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port          int
	DatabaseURL   string
	RedisURL      string
	MasterKey     string
	MasterKeyFile string
	JWTSecret     string
	MigrationsDir string
}

func Load() (*Config, error) {
	c := &Config{
		Port:          8400,
		DatabaseURL:   os.Getenv("FOOSTASH_DB_URL"),
		RedisURL:      os.Getenv("FOOSTASH_REDIS_URL"),
		MasterKey:     os.Getenv("FOOSTASH_MASTER_KEY"),
		MasterKeyFile: os.Getenv("FOOSTASH_MASTER_KEY_FILE"),
		JWTSecret:     os.Getenv("FOOSTASH_JWT_SECRET"),
		MigrationsDir: os.Getenv("FOOSTASH_MIGRATIONS_DIR"),
	}

	if p := os.Getenv("FOOSTASH_PORT"); p != "" {
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid FOOSTASH_PORT: %w", err)
		}
		c.Port = v
	}

	if c.MigrationsDir == "" {
		c.MigrationsDir = "./migrations"
	}

	var missing []string
	if c.DatabaseURL == "" {
		missing = append(missing, "FOOSTASH_DB_URL")
	}
	if c.JWTSecret == "" {
		missing = append(missing, "FOOSTASH_JWT_SECRET")
	}
	if c.MasterKey == "" && c.MasterKeyFile == "" {
		missing = append(missing, "FOOSTASH_MASTER_KEY or FOOSTASH_MASTER_KEY_FILE")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	return c, nil
}

func (c *Config) ResolveMasterKey() (string, error) {
	if c.MasterKey != "" {
		return c.MasterKey, nil
	}
	b, err := os.ReadFile(c.MasterKeyFile)
	if err != nil {
		return "", fmt.Errorf("read master key file: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}
