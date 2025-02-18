// Package tools provides a set of tools for some common situations when dealing with a sql.DB
package tools

import (
	"database/sql"
)

// IsConnected checks if a database connection is established by attempting to ping it.
// It returns true if the connection is alive, otherwise false.
func IsConnected(db *sql.DB) bool {
	return db != nil && db.Ping() == nil
}
