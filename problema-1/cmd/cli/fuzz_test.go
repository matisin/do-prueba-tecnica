package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func FuzzCliRunMigrations(f *testing.F) {
	f.Add("down", "3", 0)
	f.Add("down", "", 2)
	f.Add("down", "1", 0)
	f.Add("up", "2", 3)
	f.Add("up", "0", 0)
	f.Add("up", "/", 3)
	f.Add("other", "/", 2)

	pwd := os.Getenv("PWD")
	if pwd == "" {
		f.Skip("no PWD env var")
		return
	}
	migrationsPath := filepath.Join(filepath.Dir(filepath.Dir(pwd)), "migrations")

	f.Fuzz(func(t *testing.T, direction string, steps string, firstSteps int) {
		if firstSteps < 0 {
			t.Skip()
		}
		// Crear un directorio y DB temporal para cada caso
		dbPath := filepath.Join(t.TempDir(), fmt.Sprintf("%s.db", t.Name()))
		dbUrl := fmt.Sprintf("file:%s", dbPath)

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

		stdout := &strings.Builder{}
		stderr := &strings.Builder{}

		// Primera llamada: ejecutar algunas migraciones iniciales (por ejemplo, 2 steps up)
		err := run(context.Background(), getenv, []string{
			"agendas_app",
			"migrations",
			"--direction=up",
			fmt.Sprintf("--steps=%d", firstSteps),
		}, strings.NewReader(""), stdout, stderr)

		// esto me podria servir si quiero comparar archivos
		// firstStats, err := os.Stat(dbPath)
		// if err != nil {
		// t.Fatalf("Error verificando BD: %v", err)
		// }
		// timeBefore := firstStats.ModTime()

		if err != nil {
			t.Fatalf("Error en migraciones iniciales: %v", err)
		}

		// Segunda llamada: ejecutar el caso de fuzzing
		err = run(context.Background(), getenv, []string{
			"agendas_app",
			"migrations",
			fmt.Sprintf("--direction=%s", direction),
			fmt.Sprintf("--steps=%s", steps),
		}, strings.NewReader(""), stdout, stderr)

		t.Log(stdout)

		// Manejar errores conocidos
		if err != nil {
			if err.Error() == "direction must be up or down" && direction != "up" && direction != "down" {
				return
			}
			t.Fatalf("Error en fuzzing: %v\nStderr: %s", err, stderr.String())
		}
		assertSetupDB(t, dbPath, t.Name())
	})
}
