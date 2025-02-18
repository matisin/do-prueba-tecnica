package migrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/matisin/pizzapp/pkgs/sql/tools"
)

// const Sig is a constant that represents the name of the file where this code resides.
const Sig = "migrator.go"

type migrator struct {
	db      *sql.DB
	options options
	stderr  io.Writer
}

type migration struct {
	ID   int
	Name string
	SQL  string
}

type options struct {
	dir *string
}

type Option func(options *options)

// ErrNegSteps is an error returned by [migrator.RunMigrations] that indicates that the number of steps provided is negative.
var ErrNegSteps = errors.New("migrator.go: steps cannot be negative")

// WithMigrationsDir sets the directory for migrations. It panics if the provided directory string is empty.
func WithMigrationsDir(dir string) Option {
	return func(options *options) {
		if dir == "" {
			panic(fmt.Sprintf("%s: migration dir cannot be empty", Sig))
		}
		options.dir = &dir
	}
}

// New creates a new migrator instance with the provided database connection and error output stream,
// optionally applying any given options to customize its behavior. It panics if stderr is nil.
func New(db *sql.DB, stderr io.Writer, opts ...Option) *migrator {
	if stderr == nil {
		panic(fmt.Sprintf("%s new: stderr cannot be nil", Sig))
	}

	m := &migrator{stderr: stderr, db: db}
	for _, opt := range opts {
		opt(&m.options)
	}

	return m
}

func (m *migrator) initMigrations() error {
	fmt.Fprintf(m.stderr, "%s init migrations: executing a query in init", Sig)
	_, err := m.db.Exec(`
        CREATE TABLE IF NOT EXISTS migrations (
            id INTEGER PRIMARY KEY,
            name TEXT NOT NULL,
            executed_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    `)
	if err != nil {
		return fmt.Errorf("%s init migrations: failed to initialize migrations table: %w", Sig, err)
	}
	return nil
}

func (m *migrator) findMigrationLastID() (int, error) {
	var lastID int
	fmt.Fprintf(m.stderr, "%s find last migration id: executing query row in find last id", Sig)
	err := m.db.QueryRow(`
        SELECT id 
        FROM migrations 
        ORDER BY id DESC 
        LIMIT 1
    `).Scan(&lastID)

	if err == sql.ErrNoRows {
		return 0, err
	}
	if err != nil {
		return 0, fmt.Errorf("%s find last migration id: failed to find last migration ID: %w", Sig, err)
	}
	return lastID, nil
}

func (m *migrator) load(path string, inverse bool, steps int) ([]migration, error) {
	direction := map[bool]string{true: "down", false: "up"}

	if steps < 0 {
		return nil, ErrNegSteps
	}

	lastID, err := m.findMigrationLastID()
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("%s load: failed to find last migration ID: %w", Sig, err)
	}

	filenames, err := filepath.Glob(filepath.Join(path, fmt.Sprintf("*.%s.sql", direction[inverse])))
	if err != nil {
		return nil, fmt.Errorf("%s load: failed to get migration files: %w", Sig, err)
	}

	maxIDs := len(filenames)

	// Calcular steps si es 0
	if steps == 0 {
		if !inverse {
			steps = maxIDs - lastID
		} else {
			steps = lastID
		}
	}

	if steps == 0 {
		return []migration{}, nil
	}

	if steps < 0 {
		return nil, fmt.Errorf("%s load: step calculation failed, got %d", Sig, steps)
	}

	var fromID, toID int
	if !inverse {
		fromID = lastID + 1
		toID = lastID + steps
		if toID > maxIDs {
			toID = maxIDs
		}
	} else {
		toID = lastID
		fromID = lastID - steps
		if fromID < 1 {
			fromID = 1
		}
	}
	if fromID > toID {
		return []migration{}, nil
	}

	migrations := make([]migration, toID-fromID+1)
	for _, filename := range filenames {
		_, name := filepath.Split(filename)
		noSuffix := strings.TrimSuffix(name, fmt.Sprintf(".%s.sql", direction[inverse]))
		nameParts := strings.Split(noSuffix, "_")

		if len(nameParts) < 2 {
			return nil, fmt.Errorf("%s load: invalid migration filename format: %s", Sig, filename)
		}

		id, err := strconv.Atoi(nameParts[0])
		if err != nil {
			return nil, fmt.Errorf("%s load: invalid migration ID in filename %s: %w", Sig, filename, err)
		}
		if id == 0 {
			return nil, fmt.Errorf("%s load: migration ID cannot be 0 in file: %s", Sig, filename)
		}

		if id < fromID || id > toID {
			continue
		}

		content, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("%s load: failed to read migration file %s: %w", Sig, filename, err)
		}
		if len(content) == 0 {
			return nil, fmt.Errorf("%s load: migration file is empty: %s", Sig, filename)
		}

		counterpartPath := filepath.Join(path, fmt.Sprintf("%s.%s.sql", noSuffix, direction[!inverse]))
		counterpartContent, err := os.ReadFile(counterpartPath)
		if err != nil {
			return nil, fmt.Errorf("%s load: failed to read counterpart file for %s: %w", Sig, filename, err)
		}
		if len(counterpartContent) == 0 {
			return nil, fmt.Errorf("%s load: counterpart migration file is empty: %s", Sig, counterpartPath)
		}

		index := id - fromID
		migrations[index] = migration{
			ID:   id,
			Name: strings.Join(nameParts[1:], "_"),
			SQL:  string(content),
		}
	}

	sort.Slice(migrations, func(i, j int) bool {
		if !inverse {
			return migrations[i].ID < migrations[j].ID
		}
		return migrations[i].ID > migrations[j].ID
	})

	return migrations, nil
}

func (m *migrator) up(ctx context.Context, migrations []migration, inverse bool) error {
	for _, mig := range migrations {
		// check if the context is closed first
		select {
		default:
		case <-ctx.Done():
			return ctx.Err()
		}

		tx, err := m.db.BeginTx(ctx, nil)
		if err != nil {
			if !inverse {
				return fmt.Errorf("%s up: failed to start transaction: %w", Sig, err)
			}
			return fmt.Errorf("%s down: failed to start transaction: %w", Sig, err)
		}

		stmts := strings.Split(mig.SQL, ";")
		for _, s := range stmts[:len(stmts)-1] {
			if s = strings.TrimSpace(s); s == "" {
				continue
			}

			if _, err := tx.Exec(fmt.Sprintf("%s;", s)); err != nil {
				rollErr := tx.Rollback()
				if rollErr != nil {
					return fmt.Errorf("%s up: migration %d failed: %v, additionally rollback failed: %v", Sig, mig.ID, err, rollErr)
				}
				return fmt.Errorf("%s up: migration %d failed: %w", Sig, mig.ID, err)
			}
		}

		if _, err := tx.Exec(`INSERT INTO migrations (id, name) VALUES (?, ?)`, mig.ID, mig.Name); err != nil {
			rollErr := tx.Rollback()
			if rollErr != nil {
				return fmt.Errorf("%s up: failed to register migration %d: %v,Sig, additionally rollback failed: %v", Sig, mig.ID, err, rollErr)
			}
			return fmt.Errorf("%s up: failed to register migration %d: %w", Sig, mig.ID, err)
		}

		if err = tx.Commit(); err != nil {
			return fmt.Errorf("%s up: failed to commit migration %d: %w", Sig, mig.ID, err)
		}
	}

	return nil
}

func (m *migrator) down(ctx context.Context, migrations []migration) error {
	for _, mig := range migrations {
		tx, err := m.db.Begin()
		if err != nil {
			return fmt.Errorf("%s down: failed to start transaction: %w", Sig, err)
		}

		stmts := strings.Split(mig.SQL, ";")
		for _, s := range stmts[:len(stmts)-1] {
			if s = strings.TrimSpace(s); s == "" {
				continue
			}

			if _, err := tx.Exec(fmt.Sprintf("%s;", s)); err != nil {
				rollErr := tx.Rollback()
				if rollErr != nil {
					return fmt.Errorf("%s down: migration %d rollback failed: %v,Sig, additionally transaction rollback failed: %v", Sig, mig.ID, err, rollErr)
				}
				return fmt.Errorf("%s down: migration %d rollback failed: %w", Sig, mig.ID, err)
			}
		}

		if _, err := tx.Exec(`DELETE from migrations WHERE id = ?`, mig.ID); err != nil {
			rollErr := tx.Rollback()
			if rollErr != nil {
				return fmt.Errorf("%s down: failed to remove migration %d record: %v,Sig, additionally rollback failed: %v", Sig, mig.ID, err, rollErr)
			}
			return fmt.Errorf("%s down: failed to remove migration %d record: %w", Sig, mig.ID, err)
		}

		if err = tx.Commit(); err != nil {
			return fmt.Errorf("%s down: failed to commit migration %d rollback: %w", Sig, mig.ID, err)
		}
	}

	return nil
}

// Version returns the current version of the database schema as recorded by the last migration applied.
func (m *migrator) Version() (int, error) {
	if !tools.IsConnected(m.db) {
		return 0, fmt.Errorf("%s version: db in migrations is desconnected", Sig)
	}

	version, err := m.findMigrationLastID()
	return version, err
}

// RunMigrations runs the database migrations with the specified number of steps and inverse option.
// It initializes the migrations if necessary, loads them based on the inverse flag and step count,
// and then executes either up or down migrations accordingly. If there are no migrations to run, it returns nil.
func (m *migrator) RunMigrations(ctx context.Context, steps int, inverse bool) (int, error) {
	migrCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if !tools.IsConnected(m.db) {
		return 0, fmt.Errorf("%s run migrations: cannot run migrations, db is disconnected", Sig)
	}

	if err := m.initMigrations(); err != nil {
		return 0, fmt.Errorf("%s run migrations: failed to initialize migrations: %w", Sig, err)
	}

	migrations, err := m.load(*m.options.dir, inverse, steps)

	if err != nil {
		return 0, err
	}

	if len(migrations) == 0 {
		return 0, nil
	}

	if !inverse {
		if err := m.up(migrCtx, migrations); err != nil {
			return 0, fmt.Errorf("%s run migrations: failed to run up migrations: %w", Sig, err)
		}
		return len(migrations), nil
	}

	if err := m.down(migrCtx, migrations); err != nil {
		return 0, fmt.Errorf("%s run migrations: failed to run down migrations: %w", Sig, err)
	}
	return len(migrations), nil
}
