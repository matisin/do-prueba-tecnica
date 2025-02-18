package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
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

	dbPath := filepath.Join(f.TempDir(), fmt.Sprintf("%s.db", f.Name()))
	dbUrl := fmt.Sprintf("file:%s", dbPath)

	err := copyFile(filepath.Join(pwd, "test.db"), dbPath)
	if err != nil {
		f.Fatalf("error: %v", err)
		f.Fatalf("Could not create test db")
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
			case "DB_URL":
				return dbUrl
			}
			return ""
		}
		stdin := strings.NewReader("")
		stdout := &strings.Builder{}
		stderr := &strings.Builder{}
		args := []string{
			"sos_beacon",
		}
		// Ejecutar el servidor en una goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- run(ctx, getenv, stdin, stdout, stderr, args)
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

func TestPhaseChangeDiagramHTTP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	baseURL := setupServer(t, ctx)

	// Creamos una estructura para manejar la respuesta JSON
	type PhaseResponse struct {
		SpecificVolumeLiquid float64 `json:"specific_volume_liquid"`
		SpecificVolumeVapor  float64 `json:"specific_volume_vapor"`
	}

	// Esta función nos ayudará a hacer las peticiones HTTP y procesar las respuestas
	makeRequest := func(pressure float64) (*PhaseResponse, error) {
		url := fmt.Sprintf("%s/phase-change-diagram?pressure=%.4f", baseURL, pressure)

		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("error haciendo petición HTTP: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("código de estado inesperado: %d", resp.StatusCode)
		}

		var response PhaseResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("error decodificando respuesta: %v", err)
		}

		return &response, nil
	}

	// Definimos nuestros casos de prueba, incluyendo las verificaciones físicas
	tests := []struct {
		name           string
		pressure       float64
		expectedVapor  float64
		expectedLiquid float64
		checkExact     bool
	}{
		{
			name:           "Punto crítico",
			pressure:       10.0,
			expectedVapor:  0.0035,
			expectedLiquid: 0.0035,
		},
		{
			name:           "Punto de baja presión",
			pressure:       0.05,
			expectedVapor:  30.00,
			expectedLiquid: 0.00105,
		},
		{
			name:           "SpecificVolume 5.0000bar",
			pressure:       5,
			expectedVapor:  15.0771,
			expectedLiquid: 0.0023,
		},
		{
			name:           "SpecificVolume 2.5000bar",
			pressure:       2.5,
			expectedVapor:  22.6139,
			expectedLiquid: 0.0017,
		},
		{
			name:           "SpecificVolume 7.5000bar",
			pressure:       7.5,
			expectedVapor:  7.5403,
			expectedLiquid: 0.0029,
		},
		{
			name:           "SpecificVolume 1.0000bar",
			pressure:       1,
			expectedVapor:  27.1360,
			expectedLiquid: 0.0013,
		},
		{
			name:           "SpecificVolume 8.5000bar",
			pressure:       8.5,
			expectedVapor:  4.5256,
			expectedLiquid: 0.0031,
		},
		{
			name:           "SpecificVolume 3.5000bar",
			pressure:       3.5,
			expectedVapor:  19.5992,
			expectedLiquid: 0.0019,
		},
		{
			name:           "SpecificVolume 9.5000bar",
			pressure:       9.5,
			expectedVapor:  1.5109,
			expectedLiquid: 0.0034,
		},
		{
			name:           "SpecificVolume 0.1000bar",
			pressure:       0.1,
			expectedVapor:  29.8493,
			expectedLiquid: 0.0011,
		},
	}

	// Ejecutamos las pruebas para cada caso
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Obtenemos la respuesta del servidor
			response, err := makeRequest(tc.pressure)
			if err != nil {
				t.Fatalf("Error en la petición: %v", err)
			}

			// Si este es un punto donde conocemos los valores exactos
			if tc.checkExact {
				if response.SpecificVolumeVapor != tc.expectedVapor {
					t.Errorf("Volumen de vapor incorrecto: esperado %.5f, obtenido %.5f",
						tc.expectedVapor, response.SpecificVolumeVapor)
				}

				if response.SpecificVolumeLiquid != tc.expectedLiquid {
					t.Errorf("Volumen de líquido incorrecto: esperado %.5f, obtenido %.5f",
						tc.expectedLiquid, response.SpecificVolumeLiquid)
				}
			}

			// Verificaciones físicas que deben cumplirse en todos los casos
			if response.SpecificVolumeLiquid <= 0 {
				t.Error("El volumen del líquido no puede ser cero o negativo")
			}

			if response.SpecificVolumeVapor <= 0 {
				t.Error("El volumen del vapor no puede ser cero o negativo")
			}

			// El vapor debe ocupar más volumen que el líquido (excepto en el punto crítico)
			if tc.pressure != 10.0 && response.SpecificVolumeVapor <= response.SpecificVolumeLiquid {
				t.Error("El volumen del vapor debe ser mayor que el del líquido excepto en el punto crítico")
			}
		})
	}

	// Verificación adicional de continuidad
	t.Run("Verificación de continuidad", func(t *testing.T) {
		// Tomamos dos puntos muy cercanos para verificar que no hay saltos bruscos
		p1 := 5.0
		p2 := 5.0001

		r1, err := makeRequest(p1)
		if err != nil {
			t.Fatalf("Error en primera petición: %v", err)
		}

		r2, err := makeRequest(p2)
		if err != nil {
			t.Fatalf("Error en segunda petición: %v", err)
		}

		// Verificamos que el cambio sea suave
		deltaVapor := math.Abs(r2.SpecificVolumeVapor - r1.SpecificVolumeVapor)
		deltaLiquid := math.Abs(r2.SpecificVolumeLiquid - r1.SpecificVolumeLiquid)

		if deltaVapor > 0.01 || deltaLiquid > 0.01 {
			t.Error("Detectado cambio brusco en los valores entre puntos cercanos")
		}
	})
	t.Run("Valores fuera de rango", func(t *testing.T) {
		// Definimos los casos de prueba para valores inválidos
		invalidCases := []struct {
			name     string
			pressure float64
			message  string
		}{
			{
				name:     "Presión mayor al límite superior",
				pressure: 11.0,
				message:  "pressure must be between 0.05 and 10.0 bar",
			},
			{
				name:     "Presión menor al límite inferior",
				pressure: 0.04,
				message:  "pressure must be between 0.05 and 10.0 bar",
			},
		}

		for _, tc := range invalidCases {
			t.Run(tc.name, func(t *testing.T) {
				// Construimos la URL con el valor de presión inválido
				url := fmt.Sprintf("%s/phase-change-diagram?pressure=%.4f", baseURL, tc.pressure)

				// Realizamos la petición HTTP
				resp, err := http.Get(url)
				if err != nil {
					t.Fatalf("Error haciendo petición HTTP: %v", err)
				}
				defer resp.Body.Close()

				// Verificamos que el código de estado sea 400 Bad Request
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("Código de estado incorrecto: esperado %d (Bad Request), obtenido %d",
						http.StatusBadRequest, resp.StatusCode)
				}
			})
		}
	})
}

func TestPhaseChangeDiagramKnownPoints(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	baseURL := setupServer(t, ctx)

	testCases := []struct {
		pressure             float64
		specificVolumeLiquid float64
		specificVolumVapor   float64
		description          string
	}{
		{
			pressure:             10.0,
			specificVolumeLiquid: 0.0035,
			specificVolumVapor:   0.0035,
			description:          "Punto crítico (explícitamente dado en el diagrama)",
		},
		{
			pressure:             0.05,
			specificVolumeLiquid: 0.00105,
			specificVolumVapor:   0.0, // Usaremos un valor especial para indicar que no lo verificaremos
			description:          "Punto de baja presión (visible en la intersección)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			url := fmt.Sprintf("%s/phase-change-diagram?pressure=%.4f", baseURL, tc.pressure)

			res, err := http.Get(url)
			if err != nil {
				t.Fatalf("Error haciendo petición: %v", err)
			}
			defer res.Body.Close()

			var response struct {
				SpecificVolumeLiquid float64 `json:"specific_volume_liquid"`
				SpecificVolumeVapor  float64 `json:"specific_volume_vapor"`
			}

			if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
				t.Fatalf("Error decodificando respuesta: %v", err)
			}

			if !almostEqual(response.SpecificVolumeLiquid, tc.specificVolumeLiquid, 0.00001) {
				t.Errorf("Volumen líquido incorrecto: esperado %.5f, se obtuvo %.5f",
					tc.specificVolumeLiquid, response.SpecificVolumeLiquid)
			}

			if tc.specificVolumVapor > 0 {
				if !almostEqual(response.SpecificVolumeVapor, tc.specificVolumVapor, 0.00001) {
					t.Errorf("Volumen vapor incorrecto: esperado %.5f, se obtuvo %.5f",
						tc.specificVolumVapor, response.SpecificVolumeVapor)
				}
			}

			if tc.pressure != 10.0 && response.SpecificVolumeVapor <= response.SpecificVolumeLiquid {
				t.Error("El volumen del vapor debe ser mayor que el del líquido excepto en el punto crítico")
			}
		})
	}
}

// Helper function para comparar flotantes con tolerancia
func almostEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

func setupServer(t *testing.T, ctx context.Context) string {
	pwd := filepath.Dir(filepath.Dir(os.Getenv("PWD")))
	if pwd == "" {
		t.Fatal("no PWD env var")
	}

	dbPath := filepath.Join(t.TempDir(), fmt.Sprintf("%s.db", t.Name()))
	dbUrl := fmt.Sprintf("file:%s", dbPath)

	err := copyFile(filepath.Join(pwd, "test.db"), dbPath)
	if err != nil {
		t.Fatalf("error: %v", err)
		t.Fatalf("Could not create test db")
	}

	getenv := func(key string) string {
		switch key {
		case "HTTP_PORT":
			return "0"
		case "PWD":
			return pwd
		case "DB_URL":
			return dbUrl
		}
		return ""
	}

	stdin := strings.NewReader("")
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	args := []string{
		"sos_beacon",
		"--host=127.0.0.1",
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- run(ctx, getenv, stdin, stdout, stderr, args)
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

func TestSpaceshipRescueSim(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	baseURL := setupServer(t, ctx)

	t.Run("Rescue Sequence", func(t *testing.T) {
		// 1. Status check
		res, err := http.DefaultClient.Get(fmt.Sprintf("%s/status", baseURL))
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}

		defer res.Body.Close()

		if res.StatusCode == 404 {
			t.Fatal("Status request responded 404")
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		var statusData struct {
			DamagedSystem string `json:"damaged_system"`
		}

		if err := json.Unmarshal(body, &statusData); err != nil {
			t.Fatalf("Failed to decode status: %v", err)
		}

		validSystems := map[string]string{
			"navigation":       "NAV-01",
			"communications":   "COM-02",
			"life_support":     "LIFE-03",
			"engines":          "ENG-04",
			"deflector_shield": "SHLD-05",
		}
		expectedCode, valid := validSystems[statusData.DamagedSystem]
		if !valid {
			t.Fatalf("Invalid damaged system: %s", statusData.DamagedSystem)
		}

		// 2. Repair bay check
		repairRes, err := http.Get(baseURL + "/repair-bay")
		if err != nil {
			t.Fatalf("Failed to get repair bay: %v", err)
		}
		defer repairRes.Body.Close()

		if repairRes.StatusCode == 404 {
			t.Fatal("Status request responded 404")
		}

		body, err = io.ReadAll(repairRes.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		expectedDiv := fmt.Sprintf(`<div class="anchor-point">%s</div>`, expectedCode)
		if !strings.Contains(string(body), expectedDiv) {
			t.Errorf("Response does not contain expected div: %s", expectedDiv)
		}
		// 1. Status check
		res, err = http.DefaultClient.Get(fmt.Sprintf("%s/status", baseURL))
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}

		defer res.Body.Close()

		if res.StatusCode == 404 {
			t.Fatal("Status request responded 404")
		}

		body, err = io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		if err := json.Unmarshal(body, &statusData); err != nil {
			t.Fatalf("Failed to decode status: %v", err)
		}

		expectedCode, valid = validSystems[statusData.DamagedSystem]
		if !valid {
			t.Fatalf("Invalid damaged system: %s", statusData.DamagedSystem)
		}

		// 2. Repair bay check
		repairRes, err = http.Get(baseURL + "/repair-bay")
		if err != nil {
			t.Fatalf("Failed to get repair bay: %v", err)
		}
		defer repairRes.Body.Close()

		if repairRes.StatusCode == 404 {
			t.Fatal("Status request responded 404")
		}

		body, err = io.ReadAll(repairRes.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		expectedDiv = fmt.Sprintf(`<div class="anchor-point">%s</div>`, expectedCode)
		if !strings.Contains(string(body), expectedDiv) {
			t.Errorf("Response does not contain expected div: %s", expectedDiv)
		}

		// 3. Teapot check
		teapotRes, err := http.DefaultClient.Post(fmt.Sprintf("%s/teapot", baseURL), "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to post to teapot: %v", err)
		}
		defer teapotRes.Body.Close()
		if teapotRes.StatusCode == 404 {
			t.Fatal("Status request responded 404")
		}

		if teapotRes.StatusCode != 418 {
			t.Errorf("Wrong status code: got %d, want %d", teapotRes.StatusCode, http.StatusTeapot)
		}
	})
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
