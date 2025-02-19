package http_adapter

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/do-prueba-tecnica/problema-1/internal/app"
	"github.com/do-prueba-tecnica/problema-1/pkgs/assertor"
	"github.com/docopt/docopt-go"
)

type HTTP struct {
	app    *app.App
	stderr io.Writer
	stdout io.Writer
	logger *slog.Logger
	mux    *http.ServeMux
}

func Run(
	ctx context.Context,
	getenv func(string) string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	args []string,
) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	// Canal para manejar errores del servidor
	errChan := make(chan error, 1)

	usage := `sos beacon app http.

Usage:
    sos_beacon [--format=<j>] [--host=<h>]
    sos_beacon -h | --help
    sos_beacon --version
    
Options:
    -h --help         Show this screen.
    --version         Show version.
    --steps=<n>       Steps to move the migration [default: 0].
    --direction=<d>   Direction to move the migrations [default: up].
    --path=<p>        Path with the migrations [default: migrations/].
    --format=<j>      Format output as json [default: text]
    --host=<h>        Host to bind [default: 0.0.0.0]`

	const version = "0.0.1"

	opts, err := docopt.ParseArgs(usage, args[1:], version)
	assertor.ErrNil(err, "Failed to pars cli args")

	format, err := opts.String("--format")
	assertor.ErrNil(err, "Failed to get format option")

	host, err := opts.String("--host")
	assertor.ErrNil(err, "Failed to get host option")

	var logger *slog.Logger
	if format == "json" {
		logger = slog.New(slog.NewJSONHandler(stdout, nil))
	} else {
		logger = slog.New(slog.NewTextHandler(stdout, nil))
	}

	portStr := getenv("HTTP_PORT")
	port := getAvailablePort(portStr)
	fmt.Fprintf(stdout, "PORT=%d\n", port)

	assertor.IntBetween(port, 1024, 65535, "port in http adapter out of range")
	assertor.IntNot(port, 22, "port cannot be 22")
	assertor.PortClosed(port, "the port is closed")

	app := app.NewApp(stderr, stdout, format)

	mux := http.NewServeMux()

	h := HTTP{
		app:    app,
		logger: logger,
		mux:    mux,
		stdout: stdout,
		stderr: stderr,
	}

	h.SetRoutes()

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: mux,
	}

	// Iniciar el servidor en una goroutine
	go func() {
		errChan <- server.ListenAndServe()
	}()

	// Esperar por cancelación del contexto o error del servidor
	select {
	case <-ctx.Done():
		// Dar un timeout para el shutdown graceful
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}
