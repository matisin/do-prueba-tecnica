package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// esta simulacion ejecuta cli/main con comando migration con 120 operaciones con un set de migraciones especifico que se encuentra
// en repo/migrations usando un seed para mantener la reproducibilidad.

// Se mueve entre diferentes direcciones con diferentes steps no mas de 10. Esto puede ayudar a testear
// los down  que si o si deben ir con el up para que este proceso no tenga problemas

// las 120 operacioes son secuenciales pero si se implementa se podría intentar usar operaciones concurrentes.
func TestMigrationSimulations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	const masterSeed int64 = 3151546024680003035 
    const operations = 120

    t.Logf("Master seed: %d", masterSeed)

	masterRand := rand.New(rand.NewSource(masterSeed))

	seeds := make([]int64, operations)
	for i := 0; i < operations; i++ {
		seeds[i] = masterRand.Int63()
	}
	// Configuración inicial
	dbPath := filepath.Join(t.TempDir(), "local.db")
	dbUrl := fmt.Sprintf("file:%s", dbPath)

	pwd := os.Getenv("PWD")
	if pwd == "" {
		t.Fatal("no PWD env var")
	}
	migrationsPath := filepath.Join(filepath.Dir(filepath.Dir(pwd)), "migrations")

	getenv := func(key string) string {
		switch key {
		case "DB_URL":
			return dbUrl
		case "MIGRATIONS_PATH":
			return migrationsPath
        case "PWD":
            return pwd
		default:
			return ""
		}
	}

	t.Logf("Iniciando simulación con %d operaciones", operations)
	for i, seed := range seeds {
		rnd := rand.New(rand.NewSource(seed))

		changeDirection := rnd.Intn(2) == 0

		direction := "up"
		if changeDirection {
			direction = "down"
		}

		steps := rnd.Intn(8)

        stdin := strings.NewReader("")
		stdout := &strings.Builder{}
		stderr := &strings.Builder{}

		args := []string{
			"agendas_app",
			"migrations",
			fmt.Sprintf("--direction=%s", direction),
			fmt.Sprintf("--steps=%d", steps),
		}
		// t.Logf("Operación %d: direction=%s, steps=%d", i, direction, steps)

		errChan := make(chan error, 1)
		go func() {
			errChan <- run(ctx, getenv, args, stdin, stdout, stderr)
		}()
		select {
		case <-ctx.Done():
			t.Logf("Test cancelado después de %d operaciones", i)
			return
		case err := <-errChan:
			if err != nil {
				t.Fatalf("Error en operación %d: %v\nStdout: %s\nStderr: %s",
					i+1, err, stdout.String(), stderr.String())
			}
			assertSetupDB(t, dbPath, strconv.Itoa(i))
		}
	}
}
