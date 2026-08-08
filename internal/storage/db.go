package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type DB struct{ sql *sql.DB }

func Open(path string) (*DB, error) {
	database, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	if _, err = database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		database.Close()
		return nil, err
	}
	db := &DB{sql: database}
	if err = db.migrate(); err != nil {
		database.Close()
		return nil, err
	}
	return db, nil
}

func (db *DB) Close() error { return db.sql.Close() }
func (db *DB) SQL() *sql.DB { return db.sql }

func (db *DB) migrate() error {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	version := 0
	err = db.sql.QueryRow("SELECT version FROM schema_version LIMIT 1").Scan(&version)
	if err != nil && err != sql.ErrNoRows && !strings.Contains(err.Error(), "no such table") {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		part := strings.SplitN(entry.Name(), "_", 2)[0]
		number, parseErr := strconv.Atoi(part)
		if parseErr != nil {
			return fmt.Errorf("invalid migration name %q", entry.Name())
		}
		if number <= version {
			continue
		}
		contents, readErr := migrationFiles.ReadFile("migrations/" + entry.Name())
		if readErr != nil {
			return readErr
		}
		if _, err = db.sql.Exec(string(contents)); err != nil {
			return fmt.Errorf("apply %s: %w", entry.Name(), err)
		}
		if err = db.sql.QueryRow("SELECT version FROM schema_version LIMIT 1").Scan(&version); err != nil || version != number {
			return fmt.Errorf("migration %s did not set schema version %d", entry.Name(), number)
		}
	}
	return nil
}
