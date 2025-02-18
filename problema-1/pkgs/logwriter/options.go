package logwriter

import (
	"fmt"

	"github.com/go-on-bike/bike/interfaces"
)

type options struct {
	logger     interfaces.Logger
	textFormat *bool
	bufferSize *uint
}

type Option func(options *options)

func WithLogger(logger interfaces.Logger) Option {
	return func(options *options) {
		if logger == nil {
			panic(fmt.Sprintf("%s with logger: logger cannot be nil", Sig))
		}
		options.logger = logger
	}
}

func WithTextFormat(format bool) Option {
	return func(options *options) {
		options.textFormat = &format
	}
}

func WithBufferSize(bufferSize uint) Option {
	return func(options *options) {
		if bufferSize < 1 {
			panic(fmt.Sprintf("%s with buffer size: buffer size cannot be less than one", Sig))
		}
		options.bufferSize = &bufferSize
	}
}
