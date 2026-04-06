package foostash

import (
	"errors"
	"os"
	"time"
)

type Config struct {
	Server    string
	Token     string
	ProjectID string
	Env       string
	CacheTTL  time.Duration
}

func configFromEnv() Config {
	return Config{
		Server:    os.Getenv("FOOSTASH_SERVER"),
		Token:     os.Getenv("FOOSTASH_TOKEN"),
		ProjectID: os.Getenv("FOOSTASH_PROJECT"),
		Env:       os.Getenv("FOOSTASH_ENV"),
	}
}

func (c *Config) applyDefaults() error {
	env := configFromEnv()
	if c.Server == "" {
		c.Server = env.Server
	}
	if c.Token == "" {
		c.Token = env.Token
	}
	if c.ProjectID == "" {
		c.ProjectID = env.ProjectID
	}
	if c.Env == "" {
		c.Env = env.Env
	}
	if c.Server == "" {
		c.Server = "http://localhost:8400"
	}
	if c.CacheTTL == 0 {
		c.CacheTTL = 30 * time.Second
	}

	var missing []string
	if c.Token == "" {
		missing = append(missing, "token (FOOSTASH_TOKEN)")
	}
	if c.ProjectID == "" {
		missing = append(missing, "project (FOOSTASH_PROJECT)")
	}
	if c.Env == "" {
		missing = append(missing, "env (FOOSTASH_ENV)")
	}
	if len(missing) > 0 {
		return errors.New("foostash: missing required config: " + joinStrings(missing, ", "))
	}
	return nil
}

func joinStrings(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}
