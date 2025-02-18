package tools

import (
	"database/sql"
	"testing"

	"github.com/matisin/pizzapp/pkgs/tester"
)

// TestIsConnected tests the functionality of the IsConnected function, which checks if a database connection is established.
func TestIsConnected(t *testing.T) {
	t.Run("connected db", func(t *testing.T) {
		_, _, db := tester.GenLibSqlDB(t)
		t.Cleanup(func() {
			if err := db.Close(); err != nil {
				t.Fatalf("error closing conection after test")
			}
		})
		if conn := IsConnected(db); !conn {
			t.Fatal("expected conn to be true")
		}
	})

	t.Run("nil db", func(t *testing.T) {
		var db *sql.DB
		if conn := IsConnected(db); conn {
			t.Fatal("expected conn to be false")
		}
	})

	t.Run("connected and disconected db", func(t *testing.T) {
		_, _, db := tester.GenLibSqlDB(t)
		if conn := IsConnected(db); !conn {
			t.Fatal("expected conn to be true")
		}

		if err := db.Close(); err != nil {
			t.Fatalf("error closing conection after test")
		}

		if conn := IsConnected(db); conn {
			t.Fatal("expected conn to be false")
		}
	})

}
