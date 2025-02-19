package main

import (
	"context"
	"fmt"
	"os"

	http_adapter "github.com/do-prueba-tecnica/problema-1/internal/infra/http"
)

func main() {
	ctx := context.Background()
	if err := http_adapter.Run(ctx, os.Getenv, os.Stdin, os.Stdout, os.Stderr, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
