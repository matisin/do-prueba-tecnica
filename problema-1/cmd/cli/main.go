package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/altscore_challenges/sos-beacon/internal/application"
	"github.com/altscore_challenges/sos-beacon/internal/infra/db"
	"github.com/docopt/docopt-go"
	"github.com/go-on-bike/bike/adapters/secondary/libsql"
	"github.com/go-on-bike/bike/assert"
)

func run(
	ctx context.Context,
	getenv func(string) string,
	args []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	errChan := make(chan error, 1)

	usage := `sos beacon app cli.

Usage:
    agendas_app migrations [--direction=<d>] [--steps=<n>] [--path=<p>] [--format=<j>]
    agendas_app populate
    agendas_app get_balanced_planet
    agendas_app -h | --help
    agendas_app --version
    
Options:
    -h --help         Show this screen.
    --version         Show version.
    --steps=<n>       Steps to move the migration [default: 0].
    --direction=<d>   Direction to move the migrations [default: up].
    --path=<p>        Path with the migrations [default: migrations/].
    --format=<j>        Format output as json [default: text]`

	const version = "0.0.1"

	opts, err := docopt.ParseArgs(usage, args[1:], version)
	assert.ErrNil(err, "Failed to parse cli args")

	format, err := opts.String("--format")
	assert.ErrNil(err, "Failed to get format option")

	dburl := getenv("DB_URL")
	assert.NotEmptyString(dburl, "no DB_URL env var")

	var logger *slog.Logger
if format == "json" {
		logger = slog.New(slog.NewJSONHandler(stdout, nil))
	} else {
		logger = slog.New(slog.NewTextHandler(stdout, nil))
	}

	db := db.NewOperator(stdout, format, libsql.WithURL(dburl))
	var app *application.App

	commands := map[string]func() error{
		"migrations": func() error {
			pwd := getenv("PWD")
			assert.NotEmptyString(pwd, "no PWD env var")

			migrations_path := getenv("MIGRATIONS_PATH")
			if len(migrations_path) == 0 {
				migrations_path = filepath.Join(pwd, "migrations")
			}
			assert.NotEmptyString(migrations_path, "no MIGRATIONS_PATH env var")

			steps, err := opts.Int("--steps")
			if err != nil {
				steps = 0
			}

			direction, err := opts.String("--direction")
			assert.ErrNil(err, "Failed to get dir option")

			if direction != "up" && direction != "down" {
				return fmt.Errorf("direction must be up or down")
			}

			app = application.NewApp(db, stderr, stdout, format)
			return app.RunMigrations(migrations_path, direction, steps)
		},
	}

	type Runner func() error
	var command Runner

	for cmd, handler := range commands {
		if ok, _ := opts.Bool(cmd); ok {
			command = handler
			continue
		}
	}

	assert.NotNil(command, "command does not exists")

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Error in app", "error", fmt.Errorf("%v", r))
				errChan <- fmt.Errorf("%v", r)
			}
		}()
		err := command()
		if app != nil {
			defer app.Shutdown()
		}
		errChan <- err
	}()

	// Esperar por cancelación del contexto o error del servidor
	select {
	case <-ctx.Done():
		// Dar un timeout para el shutdown graceful
		_, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		return app.Shutdown()
	case err := <-errChan:
		return err
	}
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Getenv, os.Args, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
