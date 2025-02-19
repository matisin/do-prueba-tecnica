package app

import (
	"testing"
)

func TestGetID(t *testing.T) {
	app := &App{}

	tests := []struct {
		name     string
		patente  string
		expected uint
		hasError bool
	}{
		{"AAAA000 should return 1", "AAAA000", 1, false},
		{"AAAA001 should return 2", "AAAA001", 2, false},
		{"aAAA001 should return 2", "aAAA001", 2, false},
		{"AAAB000 should return 1001", "AAAB000", 1001, false},
		{"ZZZZ999 should return 456976000", "ZZZZ999", 456976000, false},
		{"Invalid patente should return error", "AAAA", 0, true},
		{"Invalid more letters patente should return error", "AAAAA99", 0, true},
		{"Invalid more numbers patente should return error", "AAAA9999", 0, true},
		{"Empty patente should return error", "", 0, true},
		{"Patente con ñ should return error", "AAAÑ889", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err, id := app.PatentToID(tt.patente)
			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if id != tt.expected {
					t.Errorf("Expected ID %d, but got %d", tt.expected, id)
				}
			}
		})
	}
}

func TestIDtoPatent(t *testing.T) {
	app := &App{}

	tests := []struct {
		name     string
		id       uint
		expected string
		hasError bool
	}{
		{"ID 1 should return AAAA000", 1, "AAAA000", false},
		{"ID 2 should return AAAA001", 2, "AAAA001", false},
		{"ID 1001 should return AAAB000", 1001, "AAAB000", false},
		{"ID 456976000 should return ZZZZ999", 456976000, "ZZZZ999", false},
		{"ID 0 should return error", 0, "", true},                 // ID inválido
		{"ID 456976001 should return error", 456976001, "", true}, // Fuera del rango
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err, patente := app.IDtoPatent(tt.id)
			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if patente != tt.expected {
					t.Errorf("Expected patente %s, but got %s", tt.expected, patente)
				}
			}
		})
	}
}
