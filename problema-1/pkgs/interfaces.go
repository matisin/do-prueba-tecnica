package bike

import "context"

type Starter interface {
	Start() error
}

type Connector interface {
	Connect(ctx context.Context) error
}

type Migrator interface {
	Version() (int, error)
	RunMigrations(ctx context.Context, steps int, inverse bool) error
}

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}
