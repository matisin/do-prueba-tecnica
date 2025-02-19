package app

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

var patentRX = regexp.MustCompile(`^[A-Za-z]{4}[0-9]{3}$`)

func (app *App) IDtoPatent(id uint) (error, string) {
	if id < 1 || id > 456976000 { // 456976000 = ZZZZ999
		return fmt.Errorf("id to patent: invalid ID range"), ""
	}

	// restamos 1 del id para que comienze en 0
	id--

	// parte de id que representa el texto
	idText := id / 1000

	// parte del id que representa los numeros
	patentNumbers := id % 1000

	patentText := make([]byte, 4)
	for i := 3; i >= 0; i-- {
		// sacamos la representacion en base 26 y se la sumamos al ascii A para obtener la letra
		patentText[i] = byte('A' + idText%26)
		// dividimos por 26 para avanzar a los siguientes numeros en base 26
		idText /= 26
	}

	// Formateamos la patent
	patent := fmt.Sprintf("%s%03d", string(patentText), patentNumbers)
	return nil, patent
}

func (app *App) PatentToID(patent string) (error, uint) {
	if patent == "" {
		return fmt.Errorf("patent to id: patent cannot be empty string"), 0
	}
	if !patentRX.MatchString(patent) {
		return fmt.Errorf("patent to id: patent string does not match correct format"), 0
	}

	// pre calculamos las potencias en base a 26 para poder subir el nivel de cada letra de acuerdo
	// a la posicion en el string
	abcPowers := []uint{26 * 26 * 26, 26 * 26, 26, 1} // 26^3, 26^2, 26^1, 26^0
	// aplicamos to upper para cubrir mas casos
	patentText := strings.ToUpper(patent[:4])
	var idText uint = 0
	for i, char := range patentText {
		// char es el valor ASCII de la letra de la patente actual y si le restamos el valor de A,
		// nos daria su posicion en el abcdario, el base 26 del digito, esto, lo multiplicamos acorde
		// a la potencia i del la letra y asi podemos sumarlo en idText
		idText += uint(char-'A') * abcPowers[i]
	}

	// para los numeros de la patente simplemente convertimos el string a int con la funcion Atoi
	patentNumbers := patent[4:]
	idNumbers, err := strconv.Atoi(patentNumbers)
	if err != nil {
		return fmt.Errorf(
			"patent to id: chars in numbers position in patent string failed int conversion: %w",
			err,
		), 0
	}

	// Combinamos los valores, como la primera patente es 1, debemos sumar 1
	id := idText*1000 + uint(idNumbers+1)
	return nil, id
}
