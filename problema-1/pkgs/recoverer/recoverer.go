package recoverer

import "fmt"

// TODO: terminar este paquete
// TheresNothing returns an error with a predefined message indicating that this package is useless.
func TheresNothing() error {
	message := "This package is useless"

	return fmt.Errorf("%s", message)
}
