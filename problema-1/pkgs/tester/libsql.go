// Package tester provides utilities for testing database interactions and file paths.package tester
package tester

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/tursodatabase/go-libsql"
)

// GenTestLibsqlURL generates a test database URL and the file path for a temporary SQLite database.
// It takes a testing.T object as input and returns the URL and file path.
func GenTestLibsqlURL(t *testing.T) (url string, dbPath string) {
	path := filepath.Join(t.TempDir(), "test.db")
	return "file:" + path, path
}

// AssertLibsqlDBCreated checks if the database file at the given path exists.
// If it does not exist, it fails the test with a fatal error.
func AssertLibsqlDBCreated(t *testing.T, dbPath string) {
	_, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("failed to stat database: %v", err)
	}
}

// AssertLibsqlDBNotEmtpy checks if the given database path exists and is not empty.
// It fails the test with a fatal error if the database does not exist or is empty.
func AssertLibsqlDBNotEmtpy(t *testing.T, dbPath string) {
	stats, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("failed to stat database: %v", err)
	}
	if stats.Size() == 0 {
		t.Fatal("database is empty")
	}
}

// GenLibSqlDB generates a test LibSQL database URL and opens a connection to it for testing purposes.
func GenLibSqlDB(t *testing.T) (url, dbPath string, db *sql.DB) {
	url, path := GenTestLibsqlURL(t)
	db, err := sql.Open("libsql", url)
	if err != nil {
		t.Fatalf("failed to open test libsql connection: %v", err)
	}

	return url, path, db
}
