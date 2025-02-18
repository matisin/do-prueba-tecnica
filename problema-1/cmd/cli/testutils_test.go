package main

import (
	"os"
	"testing"
)

func assertSetupDB(t *testing.T, dbPath string, id string) {
	// Verificar estado después de cada operación
	// t.Logf("Verificando DB %d", id)
	stats, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("Error verificando BD %s: %v", id, err)
	}
	if stats.Size() == 0 {
		t.Fatalf("BD %s está vacía", id)
	}
	if stats.Size() < 8192 {
		t.Fatalf("BD %s No se ejecutaron migraciones", id)
	}
}
