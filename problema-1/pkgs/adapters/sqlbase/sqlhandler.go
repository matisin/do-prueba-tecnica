package sqlbase

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"time"
)

type sqlBase struct {
	db      *sql.DB
	options options
	stderr  io.Writer
}

const Sig = "sqlbase.go"

func NewDataHandler(stderr io.Writer, opts ...Option) *sqlBase {
	defaultDriver := "libsql"
	defaultIdleConns := 25
	defaultOpenConns := 25

	if stderr == nil {
		panic(fmt.Sprintf("%s new: stderr cannot be nil", Sig))
	}

	h := &sqlBase{stderr: stderr}
	for _, opt := range opts {
		opt(&h.options)
	}
	if h.options.driver == nil {
		h.options.driver = &defaultDriver
	}

	if h.options.maxIdleConns == nil {
		h.options.maxIdleConns = &defaultIdleConns
	}

	if h.options.maxOpenConns == nil {
		h.options.maxOpenConns = &defaultOpenConns
	}

	return h
}

func (h *sqlBase) Connect() error {
	if h.db != nil {
		return fmt.Errorf("%s start: cannot create new connection: database connection already exists", Sig)
	}

	if h.options.driver == nil {
		return fmt.Errorf("%s start: cannot create new conection without a driver", Sig)
	}

	if h.options.url == nil {
		return fmt.Errorf("%s start: cannot create new connection without a url", Sig)
	}

	fmt.Fprintf(h.stderr, "%s start: connecting to url", Sig)
	db, err := sql.Open(*h.options.driver, *h.options.url)
	if err != nil {
		return fmt.Errorf("%s start: failed to open connection with driver %s: %v", Sig, *h.options.driver, err)
	}

	fmt.Fprintf(h.stderr, "%s start: ping to db connection", Sig)
	if err = db.Ping(); err != nil {
		db.Close() // Cerramos la conexión si el ping falla
		return fmt.Errorf("%s start: failed to ping database with driver %s: %v", Sig, *h.options.driver, err)
	}

	fmt.Fprintf(h.stderr, "%s start: connected succesfully", Sig)

	h.db = db
	return nil
}

func (h *sqlBase) close() error {
	if h.db == nil {
		return fmt.Errorf("%s shutdown: database connection is nil ", Sig)
	}
	err := h.db.Close()
	h.db = nil
	if err != nil {
		return fmt.Errorf("%s shutdown: close on connection failed %v", Sig, err)
	}
	return nil
}

func (h *sqlBase) Shutdown(ctx context.Context) error {
	fmt.Fprintf(h.stderr, "%s shutdown: closing connection to current connection", Sig)
	if h.db == nil {
		return fmt.Errorf("%s shutdown: database connection is nil ", Sig)
	}

	h.db.SetMaxOpenConns(0)

	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()

	for {
		stats := h.db.Stats()
		if stats.InUse == 0 {
			return h.close()
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			timer.Reset(100 * time.Millisecond)
		}
	}
}

func (h *sqlBase) QueryDB() QueryDB {
	fmt.Fprintf(h.stderr, "%s db: returning QueryDB", Sig)

	if !isConnected(h.db) {
		panic(fmt.Sprintf("%s db: database connection is nil", Sig))
	}
	return h.db
}

func (h *sqlBase) IsConnected() bool {
	fmt.Fprintf(h.stderr, "%s is connected: checking if db is still connected", Sig)
	return isConnected(h.db)
}
