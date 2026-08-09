// Package db — миграции схемы, встроенные в бинарник через go:embed.
// Файлы версионированы (000001_*, 000002_*...) по конвенции golang-migrate:
// https://github.com/golang-migrate/migrate — <version>_<name>.up.sql / .down.sql
package db

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // регистрирует драйвер для схемы "postgres://"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations применяет все ещё не применённые миграции. Безопасно вызывать при каждом старте
// сервиса — golang-migrate ведёт таблицу schema_migrations и просто ничего не сделает,
// если всё уже накачено (в этом случае возвращает migrate.ErrNoChange, что не ошибка).
func RunMigrations(databaseURL string) error {
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
