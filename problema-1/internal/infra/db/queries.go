package db

import (
	"database/sql"

	"github.com/go-on-bike/bike/assert"
)

func (op *Operator) GetFailedSystem() string {
	stmt := `SELECT name FROM systems WHERE needs_repair = true ORDER BY created_at;`

	var system string

	err := op.DB.QueryRow(stmt).Scan(&system)

	if err == sql.ErrNoRows {
		return ""
	}

	assert.ErrNil(err, "Select System need repair failed")

	return system
}

func (op *Operator) GetFailedSystemCode() string {
	stmt := `
       UPDATE systems 
       SET needs_repair = false 
       WHERE name = (
           SELECT name FROM systems 
           WHERE needs_repair = true 
           ORDER BY created_at LIMIT 1
       )
       RETURNING code;`

	var code string
	err := op.DB.QueryRow(stmt).Scan(&code)

	if err == sql.ErrNoRows {
		return ""
	}
	assert.ErrNil(err, "Select and update system failed")

	return code
}
