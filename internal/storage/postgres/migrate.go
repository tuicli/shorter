package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var migrationFilePattern = regexp.MustCompile(`^([0-9]+)_.+\.(up|down)\.sql$`)

type migrationFile struct {
	Version   string
	Dir       string
	Path      string
	SortValue string
}

func Migrate(ctx context.Context, db *sql.DB, dir string, direction string) error {
	if direction == "" {
		direction = "up"
	}
	if direction != "up" && direction != "down" {
		return errors.New("migration direction must be up or down")
	}
	if err := ensureMigrationTable(ctx, db); err != nil {
		return err
	}

	files, err := collectMigrationFiles(dir)
	if err != nil {
		return err
	}

	switch direction {
	case "up":
		return migrateUp(ctx, db, files)
	case "down":
		return migrateDown(ctx, db, files)
	default:
		return nil
	}
}

func ensureMigrationTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	return nil
}

func collectMigrationFiles(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	files := []migrationFile{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		matches := migrationFilePattern.FindStringSubmatch(name)
		if matches == nil {
			continue
		}
		files = append(files, migrationFile{
			Version:   matches[1],
			Dir:       matches[2],
			Path:      filepath.Join(dir, name),
			SortValue: name,
		})
	}

	slices.SortFunc(files, func(a migrationFile, b migrationFile) int {
		return strings.Compare(a.SortValue, b.SortValue)
	})
	return files, nil
}

func migrateUp(ctx context.Context, db *sql.DB, files []migrationFile) error {
	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.Dir != "up" {
			continue
		}
		if _, ok := applied[file.Version]; ok {
			continue
		}
		if err := applyMigration(ctx, db, file, true); err != nil {
			return err
		}
	}

	return nil
}

func migrateDown(ctx context.Context, db *sql.DB, files []migrationFile) error {
	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return err
	}

	downByVersion := map[string]migrationFile{}
	for _, file := range files {
		if file.Dir == "down" {
			downByVersion[file.Version] = file
		}
	}

	versions := make([]string, 0, len(applied))
	for version := range applied {
		versions = append(versions, version)
	}
	slices.Sort(versions)
	slices.Reverse(versions)

	for _, version := range versions {
		file, ok := downByVersion[version]
		if !ok {
			return fmt.Errorf("missing down migration for version %s", version)
		}
		if err := applyMigration(ctx, db, file, false); err != nil {
			return err
		}
	}

	return nil
}

func appliedVersions(ctx context.Context, db *sql.DB) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()

	out := map[string]struct{}{}
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan migration version: %w", err)
		}
		out[version] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migration versions: %w", err)
	}

	return out, nil
}

func applyMigration(ctx context.Context, db *sql.DB, file migrationFile, up bool) error {
	content, err := os.ReadFile(file.Path)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", file.Path, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", file.Path, err)
	}
	defer rollback(tx)

	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("apply migration %s: %w", file.Path, err)
	}

	if up {
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, file.Version); err != nil {
			return fmt.Errorf("record migration %s: %w", file.Version, err)
		}
	} else {
		if _, err := tx.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version = $1`, file.Version); err != nil {
			return fmt.Errorf("remove migration %s: %w", file.Version, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", file.Path, err)
	}

	return nil
}
