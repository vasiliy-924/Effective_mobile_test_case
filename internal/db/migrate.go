package db

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// normalizeMigrateDatabaseURL maps postgres:// to pgx5:// so golang-migrate uses the
// registered pgx/v5 driver. jackc/pgxpool still accepts ordinary postgres:// URLs.
func normalizeMigrateDatabaseURL(databaseURL string) string {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return databaseURL
	}
	switch strings.ToLower(u.Scheme) {
	case "postgres", "postgresql":
		u.Scheme = "pgx5"
		return u.String()
	default:
		return databaseURL
	}
}

// RunMigrations applies SQL migrations from migrationsPath (e.g. file://./migrations).
func RunMigrations(migrationsPath, databaseURL string) error {
	m, err := migrate.New(migrationsPath, normalizeMigrateDatabaseURL(databaseURL))
	if err != nil {
		return fmt.Errorf("migrate new: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
