package tester

import (
	"os"
	"path/filepath"
	"testing"
)

// GenMigrationsPATH returns the file path for migrations based on the environment variables PWD and MIGRATION_TEST_PATH. It checks if the PWD environment variable is set, otherwise it fails the test with a fatal error. If the MIGRATION_TEST_PATH is not set, it defaults to a predefined path relative to the current working directory.
func GenMigrationsPATH(t *testing.T) string {
	pwd := os.Getenv("PWD")
	if pwd == "" {
		t.Fatalf("failed to get env var PWD")
	}

	path := os.Getenv("MIGRATION_TEST_PATH")

	if path == "" {
		path = filepath.Join(filepath.Dir(filepath.Dir(pwd)), "migrations")
	}

	return path
}
