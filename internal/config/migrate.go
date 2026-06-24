package config

import (
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func migrationDatabaseURL(cfg *Config) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DB.User, cfg.DB.Password),
		Host:   net.JoinHostPort(cfg.DB.Host, strconv.Itoa(cfg.DB.Port)),
		Path:   "/" + cfg.DB.Name,
	}
	q := u.Query()
	q.Set("sslmode", cfg.DB.SSLMode)
	// track migrations in a dedicated table so olympics versioning stays
	// independent of any other service sharing the same database
	q.Set("x-migrations-table", "olympics_schema_migrations")
	u.RawQuery = q.Encode()
	return u.String()
}

func runMigrations(cfg *Config) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to load embedded migrations: %w", err)
	}

	// own connection (not gorm's pool), so the deferred m.Close is safe
	m, err := migrate.NewWithSourceInstance("iofs", src, migrationDatabaseURL(cfg))
	if err != nil {
		_ = src.Close()
		return fmt.Errorf("failed to initialize migrator: %w", err)
	}
	defer func() {
		if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
			slog.Warn("failed to close migrator", "source_error", srcErr, "db_error", dbErr)
		}
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("database schema already up to date")
			return nil
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	slog.Info("database schema migrations applied")
	return nil
}
