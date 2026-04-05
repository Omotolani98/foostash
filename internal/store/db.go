package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}

type Migration struct {
	Version int
	Name    string
	UpSQL   string
	DownSQL string
}

func LoadMigrations(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	byVersion := map[int]*Migration{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		isUp := strings.HasSuffix(name, ".up.sql")
		isDown := strings.HasSuffix(name, ".down.sql")
		if !isUp && !isDown {
			continue
		}

		base := strings.TrimSuffix(name, ".up.sql")
		base = strings.TrimSuffix(base, ".down.sql")

		parts := strings.SplitN(base, "_", 2)
		if len(parts) < 2 {
			continue
		}
		var version int
		if _, err := fmt.Sscanf(parts[0], "%d", &version); err != nil {
			continue
		}

		contents, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", name, err)
		}

		m, ok := byVersion[version]
		if !ok {
			m = &Migration{Version: version, Name: parts[1]}
			byVersion[version] = m
		}
		if isUp {
			m.UpSQL = string(contents)
		} else {
			m.DownSQL = string(contents)
		}
	}

	migrations := make([]Migration, 0, len(byVersion))
	for _, m := range byVersion {
		migrations = append(migrations, *m)
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	return migrations, nil
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	return err
}

func appliedVersions(ctx context.Context, db *sql.DB) (map[int]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int]bool{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

func MigrateUp(ctx context.Context, db *sql.DB, dir string) (int, error) {
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return 0, err
	}
	migrations, err := LoadMigrations(dir)
	if err != nil {
		return 0, err
	}
	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}
		if m.UpSQL == "" {
			return count, fmt.Errorf("migration %d has no up sql", m.Version)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return count, err
		}
		if _, err := tx.ExecContext(ctx, m.UpSQL); err != nil {
			tx.Rollback()
			return count, fmt.Errorf("apply migration %d: %w", m.Version, err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, m.Version, m.Name); err != nil {
			tx.Rollback()
			return count, err
		}
		if err := tx.Commit(); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func MigrateDown(ctx context.Context, db *sql.DB, dir string, steps int) (int, error) {
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return 0, err
	}
	migrations, err := LoadMigrations(dir)
	if err != nil {
		return 0, err
	}
	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return 0, err
	}

	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version > migrations[j].Version })

	count := 0
	for _, m := range migrations {
		if count >= steps {
			break
		}
		if !applied[m.Version] {
			continue
		}
		if m.DownSQL == "" {
			return count, fmt.Errorf("migration %d has no down sql", m.Version)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return count, err
		}
		if _, err := tx.ExecContext(ctx, m.DownSQL); err != nil {
			tx.Rollback()
			return count, fmt.Errorf("revert migration %d: %w", m.Version, err)
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM schema_migrations WHERE version = $1`, m.Version); err != nil {
			tx.Rollback()
			return count, err
		}
		if err := tx.Commit(); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
