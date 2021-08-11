package zlogsentry

import (
	"time"

	"github.com/rs/zerolog"
)

type config struct {
	levels       []zerolog.Level
	flushTimeout time.Duration
}

func newDefaultConfig() config {
	return config{
		levels: []zerolog.Level{
			zerolog.ErrorLevel,
			zerolog.FatalLevel,
			zerolog.PanicLevel,
		},
		flushTimeout: 3 * time.Second,
	}
}

type WriterOption func(*config)

// WithLevels configures zerolog levels that have to be sent to Sentry. Default levels are error, fatal, panic
func WithLevels(levels ...zerolog.Level) WriterOption {
	return func(cfg *config) {
		cfg.levels = levels
	}
}
