package http_adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func FuzzPort(f *testing.F) {
	pwd := filepath.Dir(filepath.Dir(os.Getenv("PWD")))
	if pwd == "" {
		f.Skip("no PWD env var")
		return
	}

	// Añadimos casos base y casos límite al corpus
	f.Add("8080")  // Caso válido
	f.Add("0")     // Valores inválidos
	f.Add("99999") // Más valores inválidos
	f.Add("22")    // Puerto privilegiado

	expectedOutput := "The server is responding ok"

	f.Fuzz(func(t *testing.T, port string) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		getenv := func(key string) string {
			switch key {
			case "HTTP_PORT":
				return port
			case "PWD":
				return pwd
			}
			return ""
		}
		stdin := strings.NewReader("")
		stdout := &strings.Builder{}
		stderr := &strings.Builder{}
		args := []string{
			"http",
		}
		// Ejecutar el servidor en una goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- Run(ctx, getenv, stdin, stdout, stderr, args)
		}()

		// Esperar hasta que veamos el puerto en la salida o haya un error
		var assignedPort string
		for {
			if stdout.Len() > 0 {
				first := strings.Split(stdout.String(), "\n")[0]
				assignedPort = strings.TrimPrefix(first, "PORT=")
				break
			}
			select {
			case err := <-errChan:
				t.Fatalf("Server failed to start: %v", err)
			case <-ctx.Done():
				t.Fatal("Timeout waiting for server to start")
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}

		// Intentar conectar al servidor con retries
		var res *http.Response
		var err error
		for retries := 3; retries > 0; retries-- {
			res, err = http.DefaultClient.Get(fmt.Sprintf("http://localhost:%s/healthcheck", assignedPort))
			if err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if err != nil {
			t.Fatalf("Failed to connect to server after retries: %v", err)
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		if got := string(body); got != expectedOutput {
			t.Errorf("Wrong body content:\nexpected: %q\ngot: %q", expectedOutput, got)
		} else {
			t.Log("✅ Response content matched expected output")
		}
	})
}

func TestGetPatenteByID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	baseURL := setupTestServer(t, ctx)

	// Casos de prueba
	tests := []struct {
		name         string
		id           string
		expectedCode int
		expectedBody map[string]string
	}{
		{"ID válido", "1", http.StatusOK, map[string]string{"patente": "AAAA000"}},
		{"ID válido", "1001", http.StatusOK, map[string]string{"patente": "AAAB000"}},
		{"ID inválido", "0", http.StatusBadRequest, nil},
		{"ID fuera de rango", "456976001", http.StatusBadRequest, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/patente/%s", baseURL, tt.id)
			// Realizar la solicitud GET al servidor de prueba
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Error al realizar la solicitud: %v", err)
			}
			defer resp.Body.Close()

			// Verificar el código de estado
			if resp.StatusCode != tt.expectedCode {
				t.Errorf("Código de estado esperado %d, pero obtuvo %d", tt.expectedCode, resp.StatusCode)
			}

			if resp.StatusCode != http.StatusBadRequest {
				// Verificar el cuerpo de la respuesta
				var body map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
					t.Fatalf("Error al decodificar la respuesta JSON: %v", err)
				}

				if body["patente"] != tt.expectedBody["patente"] {
					t.Errorf("Respuesta esperada %v, pero obtuvo %v", tt.expectedBody, body)
				}
			}

		})
	}
}

func TestGetIDByPatent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	baseURL := setupTestServer(t, ctx)

	// Casos de prueba
	tests := []struct {
		name         string
		patent       string
		expectedCode int
		expectedBody map[string]int
	}{
		{"patente válido", "AAAA000", http.StatusOK, map[string]int{"id": 1}},
		{"patente válida 2", "AAAB000", http.StatusOK, map[string]int{"id": 1001}},
		{"patente inválida", "AAAA0000", http.StatusBadRequest, nil},
		{"patente vacia", "", http.StatusNotFound, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/id/%s", baseURL, tt.patent)
			// Realizar la solicitud GET al servidor de prueba
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Error al realizar la solicitud: %v", err)
			}
			defer resp.Body.Close()

			// Verificar el código de estado
			if resp.StatusCode != tt.expectedCode {
				t.Errorf("Código de estado esperado %d, pero obtuvo %d", tt.expectedCode, resp.StatusCode)
			}

			if (resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound ){
				// Verificar el cuerpo de la respuesta
				var body map[string]int
				if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
					t.Fatalf("Error al decodificar la respuesta JSON: %v", err)
				}

				if body["id"] != tt.expectedBody["id"] {
					t.Errorf("Respuesta esperada %v, pero obtuvo %v", tt.expectedBody["id"], body["id"])
				}
			}
		})
	}
}

func setupTestServer(t *testing.T, ctx context.Context) string {
	pwd := filepath.Dir(filepath.Dir(os.Getenv("PWD")))
	if pwd == "" {
		t.Fatal("no PWD env var")
	}

	getenv := func(key string) string {
		switch key {
		case "HTTP_PORT":
			return "0"
		case "PWD":
			return pwd
		}
		return ""
	}

	stdin := strings.NewReader("")
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	args := []string{
		"http",
		"--host=127.0.0.1",
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- Run(ctx, getenv, stdin, stdout, stderr, args)
	}()

	var assignedPort string
	for {
		if stdout.Len() > 0 {
			first := strings.Split(stdout.String(), "\n")[0]
			assignedPort = strings.TrimPrefix(first, "PORT=")
			break
		}
		select {
		case err := <-errChan:
			t.Fatalf("Server failed to start: %v", err)
		case <-ctx.Done():
			t.Fatal("Timeout waiting for server to start")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	return fmt.Sprintf("http://localhost:%s", assignedPort)
}
