package connector

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	bike "github.com/matisin/pizzapp/pkgs"
	"github.com/matisin/pizzapp/pkgs/sql/tools"
	"github.com/matisin/pizzapp/pkgs/tester"
	_ "github.com/tursodatabase/go-libsql"
)

// TestSQLIsConnector tests the functionality of the connector to ensure it properly connects and disconnects from a libsql database.
func TestSQLIsConnector(t *testing.T) {
	dbURL, dbPath := tester.GenTestLibsqlURL(t)

	stderr := &strings.Builder{}
	ctx, cancel := context.WithCancel(context.Background())

	var db *sql.DB

	var connector bike.Connector = New(
		db,
		stderr,
		WithURL(dbURL),
		WithDriver("libsql"),
	)

	errChan := make(chan error, 1)
	go func() {
		err := connector.Connect(ctx)
		if err != nil {
			t.Fatalf("failed to connect to database: %v", err)
		}
		errChan <- err
	}()

	if err := connector.Connect(); err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	if err := connector.Shutdown(ctx); err != nil {
		t.Fatalf("error calling shutdown: %v", err)
	}
	cancel()

	time.Sleep(100 * time.Millisecond)

	if tools.IsConnected(db) {
		t.Error("connector still connected after context cancellation")
	}

	tester.AssertLibsqlDBCreated(t, dbPath)
}

// New creates a new Connector instance. It accepts a database connection and an error stream to handle errors,
// along with optional options for configuring the connector such as URL or driver name. The function panics if
// the provided URL is empty or if the provided driver name is not accepted by the connector.
func TestNew(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		dbURL, _ := tester.GenTestLibsqlURL(t)
		stderr := &strings.Builder{}
		var db *sql.DB

		c := New(db, stderr, WithURL(dbURL))
		if c == nil {
			t.Fatal("expected non-nil Connector")
		}
		if c.options.url == nil {
			t.Fatal("expected non-nil URL option")
		}
	})

	t.Run("empty url panics", func(t *testing.T) {
		stderr := &strings.Builder{}
		var db *sql.DB
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic with empty URL")
			}
		}()
		New(db, stderr, WithURL(""))
	})

	t.Run("nil stderr panics", func(t *testing.T) {
		var db *sql.DB
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic with empty URL")
			}
		}()
		New(db, nil, WithURL(""))
	})

	t.Run("nil stderr panics", func(t *testing.T) {
		stderr := &strings.Builder{}
		var db *sql.DB
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic with not accepted driver")
			}
		}()
		New(db, stderr, WithDriver("driver"))
	})
}

func TestConnect(t *testing.T) {
	t.Run("successful connection", func(t *testing.T) {
		dbURL, _ := tester.GenTestLibsqlURL(t)
		stderr := &strings.Builder{}
		var db *sql.DB

		c := New(db, stderr, WithURL(dbURL), WithDriver("libsql"))

		t.Cleanup(func() {
			if err := c.close(); err != nil {
				t.Fatalf("error closing conection after test")
			}
		})

		err := c.Connect()
		if err != nil {
			t.Fatalf("unexpected error connecting: %v", err)
		}
		if c.db == nil {
			t.Fatal("expected non-nil db after connection")
		}
	})

	t.Run("default driver", func(t *testing.T) {
		dbURL, _ := tester.GenTestLibsqlURL(t)
		stderr := &strings.Builder{}
		var db *sql.DB

		c := New(db, stderr, WithURL(dbURL))

		t.Cleanup(func() {
			if err := c.close(); err != nil {
				t.Fatalf("error closing conection after test")
			}
		})

		err := c.Connect()
		if err != nil {
			t.Fatal("expected no error connecting")
		}
	})

	t.Run("double connection attempt", func(t *testing.T) {
		dbURL, _ := tester.GenTestLibsqlURL(t)
		stderr := &strings.Builder{}
		var db *sql.DB

		c := New(db, stderr, WithURL(dbURL), WithDriver("libsql"))

		t.Cleanup(func() {
			if err := c.close(); err != nil {
				t.Fatalf("error closing conection after test")
			}
		})

		err := c.Connect()
		if err != nil {
			t.Fatalf("unexpected error on first connect: %v", err)
		}

		// Intentar conectar de nuevo debería fallar
		err = c.Connect()
		if err == nil {
			t.Fatal("expected error on second connect")
		}
	})

	t.Run("malformed url", func(t *testing.T) {
		stderr := &strings.Builder{}
		var db *sql.DB

		c := New(db, stderr, WithURL("invalid://url"))

		err := c.Connect()
		if err == nil {
			t.Fatal("expected error with invalid URL")
		}

	})
}

// TestConnectorClose verifica el comportamiento del cierre de conexiones
func TestConnectorShutdown(t *testing.T) {
	t.Run("successful close", func(t *testing.T) {
		dbURL, _ := tester.GenTestLibsqlURL(t)
		stderr := &strings.Builder{}
		var db *sql.DB

		ctx, cancel := context.WithCancel(context.Background())

		connector := New(
			db,
			stderr,
			WithURL(dbURL),
			WithDriver("libsql"),
		)

		if err := connector.Connect(); err != nil {
			t.Fatalf("failed to connect to database: %v", err)
		}

		go func() {
			if err := connector.Shutdown(ctx); err != nil {
				t.Fatalf("error calling shutdown: %v", err)
			}
		}()
		cancel()
		if tools.IsConnected(db) {
			t.Error("connector still connected after context cancellation")
		}
	})

	// t.Run("close without connect panics", func(t *testing.T) {
	// dbURL, _ := GenTestLibsqlDBPath(t)
	// stderr := &strings.Builder{}

	// h := NewDataHandler(stderr, WithURL(dbURL), WithDriver("libsql"))

	// defer func() {
	// if r := recover(); r == nil {
	// t.Fatal("expected panic when closing without connection")
	// }
	// }()
	// h.Shutdown()
	// })

	// t.Run("DB() without connect panics", func(t *testing.T) {
	// dbURL, _ := GenTestLibsqlDBPath(t)
	// stderr := &strings.Builder{}

	// h := NewDataHandler(stderr, WithURL(dbURL), WithDriver("libsql"))

	// defer func() {
	// if r := recover(); r == nil {
	// t.Fatal("expected panic when getting DB without connection")
	// }
	// }()
	// h.DB()
	// })

	// t.Run("checking functionality of IsConnected", func(t *testing.T) {
	// dbURL, _ := GenTestLibsqlDBPath(t)
	// stderr := &strings.Builder{}

	// h := NewDataHandler(stderr, WithURL(dbURL), WithDriver("libsql"))

	// connected := h.IsConnected()
	// if connected {
	// t.Fatal("connected should false")
	// }

	// err := h.Start()
	// if err != nil {
	// t.Fatalf("unexpected error connecting: %v", err)
	// }

	// connected = h.IsConnected()
	// if !connected {
	// t.Fatal("connected should true")
	// }

	// h.Shutdown()
	// connected = h.IsConnected()
	// if connected {
	// t.Fatal("connected should false")
	// }
	// })

	// t.Run("double close", func(t *testing.T) {
	// dbURL, _ := GenTestLibsqlDBPath(t)
	// stderr := &strings.Builder{}

	// h := NewDataHandler(stderr, WithURL(dbURL), WithDriver("libsql"))

	// err := h.Start()
	// if err != nil {
	// t.Fatalf("unexpected error connecting: %v", err)
	// }

	// err = h.Shutdown()
	// if err != nil {
	// t.Fatalf("unexpected error on first close: %v", err)
	// }

	// defer func() {
	// if r := recover(); r == nil {
	// t.Fatal("expected panic on second close")
	// }
	// }()
	// h.Shutdown()
	// })
}

// TestConnectorIntegration tests the integration of a database connector by setting up a test table and inserting data, then querying it to verify correctness.
func TestConnectorIntegration(t *testing.T) {
	dbURL, _ := tester.GenTestLibsqlURL(t)
	stderr := &strings.Builder{}
	var db *sql.DB

	c := New(db, stderr, WithURL(dbURL), WithDriver("libsql"))

	t.Cleanup(func() {
		if err := c.close(); err != nil {
			t.Fatalf("error closing conection after test")
		}
	})

	err := c.Connect()
	if err != nil {
		t.Fatalf("unexpected error connecting: %v", err)
	}

	var journalMode string
	err = c.db.QueryRow("PRAGMA journal_mode=WAL").Scan(&journalMode)
	if err != nil {
		t.Fatalf("failed to set WAL mode: %v", err)
	}
	// El valor retornado debería ser "wal"
	if journalMode != "wal" {
		t.Fatalf("expected journal_mode to be 'wal', got '%s'", journalMode)
	}

	_, err = c.db.Exec(`
        CREATE TABLE test (
            id INTEGER PRIMARY KEY,
            name TEXT NOT NULL
        )
    `)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	testData := []string{"Alice", "Bob", "Charlie"}
	for _, name := range testData {
		_, err = c.db.Exec("INSERT INTO test (name) VALUES (?)", name)
		if err != nil {
			t.Fatalf("failed to insert test data '%s': %v", name, err)
		}
	}

	rows, err := c.db.Query("SELECT name FROM test ORDER BY id")
	if err != nil {
		t.Fatalf("failed to query test data: %v", err)
	}
	defer rows.Close()

	var retrievedNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("failed to scan row: %v", err)
		}
		retrievedNames = append(retrievedNames, name)
	}

	if err = rows.Err(); err != nil {
		t.Fatalf("error during row iteration: %v", err)
	}

	if len(retrievedNames) != len(testData) {
		t.Fatalf("expected %d names, got %d", len(testData), len(retrievedNames))
	}
	for i, expected := range testData {
		if retrievedNames[i] != expected {
			t.Fatalf("at position %d: expected '%s', got '%s'", i, expected, retrievedNames[i])
		}
	}
}
