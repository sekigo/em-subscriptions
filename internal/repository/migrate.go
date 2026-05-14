package repository

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies all pending migrations from sourceURL to the database
// at dbURL. Safe to call repeatedly: ErrNoChange is treated as success.
func RunMigrations(sourceURL, dbURL string) error {
	m, err := migrate.New(sourceURL, dbURL)
	if err != nil {
		return fmt.Errorf("migrate new: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
