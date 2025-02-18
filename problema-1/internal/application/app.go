package application

import (
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
)

type App struct {
	stderr io.Writer
	stdout io.Writer
	logger *slog.Logger
}

func NewApp(
	stderr io.Writer,
	stdout io.Writer,
	format string,
) *App {
	var logger *slog.Logger
	if format == "json" {
		logger = slog.New(slog.NewJSONHandler(stdout, nil))
	} else {
		logger = slog.New(slog.NewTextHandler(stdout, nil))

	}

	app := App{
		stderr: stderr,
		stdout: stdout,
		logger: logger,
	}
	return &app
}

var patenteRX = regexp.MustCompile(`^[A-Za-z]{4}[0-9]{3}$`)

func (app *App) GetPatente(id uint) (error, string) {
	if id < 1 || id > 456976000 { // 456976000 = ZZZZ999
		return fmt.Errorf("ID inválido"), ""
	}

	// restamos 1 del id para que comienze en 0
	id--

	// obtenemos los numeros al dividir por mil
	letters := id / 1000

	letterByte := make([]byte, 4)
	for i := 3; i >= 0; i-- {
		letterByte[i] = byte('A' + letters%26)
		letters /= 26
	}

	// primero obtenemos los numeros, que estan en base 10 y son los ultimos digitos
	numbers := id % 1000

	// Formateamos la patente
	patente := fmt.Sprintf("%s%03d", string(letterByte), numbers)
	return nil, patente
}

func (app *App) GetID(patente string) (error, uint) {
	if patente == "" {
		return fmt.Errorf("patente no puede ser string vacio"), 0
	}
	if !patenteRX.MatchString(patente) {
		return fmt.Errorf("patente tiene formato invalido"), 0
	}
	// calculamos el valor de cada letra dentro de la patente en base a 26 y multiplicamos por 26 elevado a su posicion
	letterWeights := []uint{26 * 26 * 26, 26 * 26, 26, 1} // 26^3, 26^2, 26^1, 26^0
	letters := strings.ToUpper(patente[:4])
	var letterValue uint = 0
	for i, char := range letters {
		// char es acii y si le restamos el valor de A nos daria su posicion en el abcdario
		letterValue += uint(char-'A') * letterWeights[i]
	}

	// para los numeros simplemente convertimos el string a int con la funcion Atoi
	numbers := patente[4:]
	numberValue, err := strconv.Atoi(numbers)
	if err != nil {
		return err, 0
	}

	// Combinamos los valores
	return nil, letterValue*1000 + uint(numberValue+1)
}

// borrar desde aca
// func (app *App) RunMigrations(path, direction string, steps int) error {
// app.logger.Info("running migrations", "path", path, "direction", direction, "steps", steps)

// return app.db.RunMigrations(path, direction, steps)
// }

// func (app *App) Status() string {
// app.logger.Info("getting status of systems")
// return app.db.GetFailedSystem()
// }

// func (app *App) StatusCode() string {
// app.logger.Info("getting status of systems with code")
// return app.db.GetFailedSystemCode()
// }

// const (
// CriticalPressure = 10.0    // MPa
// CriticalTemp     = 500.0   // °C
// CriticalVolume   = 0.00350 // m³/kg
// MinPressure      = 0.05    // MPa
// MaxVaporVolume   = 30.00
// )

// func liquidoPresion(x float64) float64 {
// // Usando los puntos (0.05, 0.00105) y (10, 0.0035)
// pendiente := (0.0035 - 0.00105) / (10 - 0.05)
// liquido := pendiente*(x-0.05) + 0.00105
// return math.Round(liquido*1000000) / 1000000
// }

// // Calcula y para la segunda línea usando la ecuación punto-pendiente
// func vaporPresion(x float64) float64 {
// // Usando los puntos (0.05, 30) y (10, 0.0035)
// pendiente := (0.0035 - 30) / (10 - 0.05)
// vapor := pendiente*(x-0.05) + 30
// return math.Round(vapor*1000000) / 1000000
// }

// func (app *App) PhaseChangeDiagram(pressure float64) (vapor float64, liquid float64) {
// app.logger.Info("starting volume phase calculations", "preassure", pressure)

// assert.FloatGeq(pressure, MinPressure, "pressure must be between 0.05 MPa and 10 Mpa")
// assert.FloatLeq(pressure, CriticalPressure, "pressure must be between 0.05 MPa and 10 Mpa")

// liquidVolume := liquidoPresion(pressure)
// vaporVolume := vaporPresion(pressure)

// return vaporVolume, liquidVolume
// }

// // Borrar hasta aca
