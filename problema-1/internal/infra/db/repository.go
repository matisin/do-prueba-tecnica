package db

import (
	"io"
	"log/slog"

	"github.com/go-on-bike/bike/adapters/secondary/libsql"
)

type Operator struct {
    libsql.Operator
	logger  *slog.Logger
}

func NewOperator(stdout io.Writer, outformat string, opts ...libsql.Option) *Operator {
	var logger *slog.Logger

	if outformat == "json" {
		logger = slog.New(slog.NewJSONHandler(stdout, nil))
	} else {
		logger = slog.New(slog.NewTextHandler(stdout, nil))
	}
	libsqlOp := libsql.NewOperator(opts...)

    dbOp := Operator{
        Operator: *libsqlOp,
        logger: logger,
    }

	return &dbOp
}
