package connector

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/matisin/pizzapp/pkgs/sql/tools"
)

type connector struct {
	db      *sql.DB
	options options
	stderr  io.Writer
}

type options struct {
	url          *string
	driver       *string
	maxOpenConns *int
	maxIdleConns *int
}

type Option func(options *options)

// Sig is the constant that represents the source file name "connector.go".
const Sig = "connector.go"

// WithURL returns an Option that sets the database URL in the options. It panics if the provided URL is empty.
func WithURL(url string) Option {
	return func(options *options) {
		if url == "" {
			panic(fmt.Sprintf("%s: database URL cannot be empty", Sig))
		}
		options.url = &url
	}
}

var drivers = []string{"libsql"}

// Drivers returns a list of available drivers for migrations.
func Drivers() []string {
	return drivers
}

// WithDriver returns an Option that sets the migration driver to the specified driver if it is supported by the application.
// It panics if the provided driver is empty or not supported
func WithDriver(driver string) Option {
	return func(options *options) {
		if driver == "" {
			panic(fmt.Sprintf("%s: migration driver cannot be empty", Sig))
		}
		if !slices.Contains(drivers, driver) {
			panic(fmt.Sprintf("%s with driver: migration driver %s not implemented", Sig, driver))
		}
		options.driver = &driver
	}
}

// WithMaxOpenConns returns an Option that sets the maximum number of open connections for the database.
// It panics if the provided maxOpenConns is less than 0
func WithMaxOpenConns(maxOpenConns int) Option {
	return func(options *options) {
		if maxOpenConns <= 0 {
			panic(fmt.Sprintf("%s with max open conns: max open connections have to be grater than 0", Sig))
		}
		options.maxOpenConns = &maxOpenConns
	}
}

// WithMaxIdleConns sets the maximum number of idle connections allowed in the pool.
// It panis if the provided maxIdleConns is less than 0
func WithMaxIdleConns(maxIdleConns int) Option {
	return func(options *options) {
		if maxIdleConns <= 0 {
			panic(fmt.Sprintf("%s with max idle conns: max iddle connections have to be grater than 0", Sig))
		}
		options.maxIdleConns = &maxIdleConns
	}
}

// New creates and returns a new connector instance with the provided database connection, error stream, and optional configuration options.
func New(db *sql.DB, stderr io.Writer, opts ...Option) *connector {
	driver := "libsql"
	idleConns := 25
	openConns := 25

	if stderr == nil {
		panic(fmt.Sprintf("%s new: stderr cannot be nil", Sig))
	}

	if tools.IsConnected(db) {
		panic(fmt.Sprintf("%s new: db is already connected", Sig))
	}

	o := &connector{db: db, stderr: stderr}
	for _, opt := range opts {
		opt(&o.options)
	}
	if o.options.driver == nil {
		o.options.driver = &driver
	}

	if o.options.maxIdleConns == nil {
		o.options.maxIdleConns = &idleConns
	}

	if o.options.maxOpenConns == nil {
		o.options.maxOpenConns = &openConns
	}

	if o.options.url == nil {
		panic(fmt.Sprintf("%s new: cannot create Connector without a db url", Sig))
	}

	return o
}

// Connect attempts to establish a connection to the database using the provided driver and URL.
// It first checks if a connection already exists, then opens a new connection, pings it, and finally sets c.db to the newly opened connection.
func (c *connector) Connect(ctx context.Context) error {
	if tools.IsConnected(c.db) {
		return fmt.Errorf("%s start: cannot create new connection: database connection already exists", Sig)
	}

	fmt.Fprintf(c.stderr, "%s start: connecting to url", Sig)
	db, err := sql.Open(*c.options.driver, *c.options.url)
	if err != nil {
		return fmt.Errorf("%s start: failed to open connection with driver %s: %v", Sig, *c.options.driver, err)
	}

	fmt.Fprintf(c.stderr, "%s start: ping to db connection", Sig)
	if err = db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("%s start: failed to ping database with driver %s: %v", Sig, *c.options.driver, err)
	}

	fmt.Fprintf(c.stderr, "%s start: connected succesfully", Sig)

	c.db = db
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = c.Shutdown(shutdownCtx)
	return err
}

func (c *connector) close() error {
	if c.db == nil {
		return fmt.Errorf("%s shutdown: database connection is nil ", Sig)
	}
	err := c.db.Close()
	c.db = nil
	if err != nil {
		return fmt.Errorf("%s shutdown: close on connection failed %v", Sig, err)
	}
	return nil
}

// Shutdown gracefully shuts down the database connection by closing it after waiting for connections to finish using them.
func (c *connector) Shutdown(ctx context.Context) error {
	fmt.Fprintf(c.stderr, "%s shutdown: closing connection to current connection", Sig)
	if c.db == nil {
		return fmt.Errorf("%s shutdown: database connection is nil ", Sig)
	}

	c.db.SetMaxOpenConns(0)

	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()

	for {
		stats := c.db.Stats()
		if stats.InUse == 0 {
			return c.close()
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			timer.Reset(100 * time.Millisecond)
		}
	}
}
