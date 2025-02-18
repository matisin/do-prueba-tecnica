package logwriter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-on-bike/bike"
	"github.com/go-on-bike/bike/adapters/sql"
	"github.com/go-on-bike/bike/tester"
	_ "github.com/tursodatabase/go-libsql"
)

func TestLogFormatter_Integration(t *testing.T) {
	stderr := &bytes.Buffer{}

	logwriter, formatErrChan := NewLogWriter(stderr, WithTextFormat(false))
	var errChan chan error = make(chan error)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	dbURL, _ := tester.GenTestLibsqlDBPath(t)
	handler := sql.NewDataHandler(logwriter, sql.WithURL(dbURL), sql.WithDriver("libsql"))

	components := []bike.GracefulShutdowner{handler, logwriter}

	// Nos aseguramos de que la conexión se cierre al finalizar
	t.Cleanup(func() {
		// Cancelamos el contexto para iniciar el shutdown
		cancel()
		// Esperamos un poco para que el shutdown se complete
		time.Sleep(100 * time.Millisecond)
	})

	go func() {
		err := logwriter.Start(ctx)
		if err != nil {
			errChan <- err
		}
	}()
	time.Sleep(100 * time.Millisecond)

	go func() {
		err := handler.Connect()
		formatErrChan <- err
		if err != nil {
			errChan <- err
			return
		}

		handler.QueryDB()

		if connected := handler.IsConnected(); !connected {
			errChan <- fmt.Errorf("handler should be connected")
		}
		cancel()
	}()

	select {
	case err := <-errChan:
		t.Fatalf("%v", err)
	case <-ctx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		for _, comp := range components {
			err := comp.Shutdown(shutdownCtx)
			if err != nil {
				t.Fatal("Error when shutting down")
			}
		}
	}

	logs := stderr.String()
	t.Log(logs)
	if len(logs) == 0 {
		t.Errorf("final stream is empty")
	}
	for i, line := range strings.Split(logs, "\n") {
		// Ignorar líneas vacías
		if line == "" {
			continue
		}

		// Intentar parsear como JSON
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			t.Errorf("línea %d no es JSON válido: %v\ncontenido: %s", i+1, err, line)
			continue
		}

		// Verificsterrar campos esperados en el JSON
		if _, ok := logEntry["time"]; !ok {
			t.Errorf("línea %d no tiene campo 'time'", i+1)
		}
		if _, ok := logEntry["level"]; !ok {
			t.Errorf("línea %d no tiene campo 'level'", i+1)
		}
		if _, ok := logEntry["msg"]; !ok {
			t.Errorf("línea %d no tiene campo 'msg'", i+1)
		}
	}
}
