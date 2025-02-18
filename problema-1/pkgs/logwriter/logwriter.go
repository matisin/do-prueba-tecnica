package logwriter

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

type logWriter struct {
	stderr   io.Writer
	messages chan []byte
	errors   chan error
	options  options
}

const Sig = "logwritter.go"

func NewLogWriter(
	stderr io.Writer,
	opts ...Option,
) (*logWriter, chan error) {
	defautBufferSize := uint(1024)
	textFormat := true

	if stderr == nil {
		panic(fmt.Sprintf("%s new: stderr cannot be nil", Sig))
	}

	lf := &logWriter{
		stderr: stderr,
		options: options{
			bufferSize: &defautBufferSize,
			textFormat: &textFormat,
		},
	}

	for _, opt := range opts {
		opt(&lf.options)
	}

	lf.errors = make(chan error, *lf.options.bufferSize)
	lf.messages = make(chan []byte, *lf.options.bufferSize)

	if lf.options.logger == nil {
		if *lf.options.textFormat {
			lf.options.logger = slog.New(slog.NewTextHandler(stderr, nil))
		} else {
			lf.options.logger = slog.New(slog.NewJSONHandler(stderr, nil))
		}
		lf.options.logger.Info(fmt.Sprintf("%s new: using default slog logger", Sig))
	}

	return lf, lf.errors
}

// Write ahora simplemente envía los bytes al canal y retorna inmediatamente
func (lf *logWriter) Write(p []byte) (n int, err error) {
	msg := make([]byte, len(p))
	copy(msg, p)

	// Enviamos al canal de manera no bloqueante
	select {
	case lf.messages <- msg:
		return len(p), nil
	default:
		// Si el canal está lleno, escribimos directamente al stderr
		// para evitar pérdida de logs
		return lf.stderr.Write(p)
	}
}

func (lf *logWriter) Start(ctx context.Context) error {
	for {
		select {
		case err := <-lf.errors:
			if err == nil {
				continue
			}
			lf.options.logger.Error(err.Error())
		case msg := <-lf.messages:
			if strMsg := processMessage(msg); strMsg != "" {
				lf.options.logger.Info(strMsg)
			}
		case <-ctx.Done():
			return nil
		}

	}
}

func processMessage(msg []byte) string {
	msgLen := len(msg)
	for msgLen > 0 && msg[msgLen-1] == '\n' {
		msgLen--
	}
	if msgLen > 0 {
		// aca se puede agregar mas tags pero hay que pensar bien de que y como antes.
		return string(msg[:msgLen])
	}
	return ""
}

func (lf *logWriter) Shutdown(ctx context.Context) error {
	lf.options.logger.Info(fmt.Sprintf("%s shutdown: closing stderr channel, processing last messages", Sig))

	done := make(chan struct{})
	go func() {
		close(lf.messages)
		for msg := range lf.messages {
			if strMsg := processMessage(msg); strMsg != "" {
				lf.options.logger.Info(strMsg)
			}
		}

		close(lf.errors)
		for err := range lf.errors {
			if err != nil {
				lf.options.logger.Error(err.Error())
			}
		}

		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
