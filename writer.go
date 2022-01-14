// Package zlogsentry provides a zerolog writer which sends selected events to sentry.
package zlogsentry

import (
	"errors"
	"io"
	"time"
	"unsafe"

	"github.com/buger/jsonparser"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

var levelsMapping = map[zerolog.Level]sentry.Level{
	zerolog.DebugLevel: sentry.LevelDebug,
	zerolog.InfoLevel:  sentry.LevelInfo,
	zerolog.WarnLevel:  sentry.LevelWarning,
	zerolog.ErrorLevel: sentry.LevelError,
	zerolog.FatalLevel: sentry.LevelFatal,
	zerolog.PanicLevel: sentry.LevelFatal,
}

var _ io.WriteCloser = new(Writer)

var now = time.Now

type Writer struct {
	client *sentry.Client

	levels       map[zerolog.Level]struct{}
	flushTimeout time.Duration
}

func New(client *sentry.Client, opts ...WriterOption) (*Writer, error) {
	cfg := newDefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	levels := make(map[zerolog.Level]struct{}, len(cfg.levels))
	for _, lvl := range cfg.levels {
		levels[lvl] = struct{}{}
	}

	return &Writer{
		client:       client,
		levels:       levels,
		flushTimeout: cfg.flushTimeout,
	}, nil
}

func (w *Writer) Write(data []byte) (int, error) {
	event, ok := w.parseLogEvent(data)
	if ok {
		w.client.CaptureEvent(event, nil, nil)
		// should flush before os.Exit
		if event.Level == sentry.LevelFatal {
			w.client.Flush(w.flushTimeout)
		}
	}

	return len(data), nil
}

func (w *Writer) Close() error {
	w.client.Flush(w.flushTimeout)
	return nil
}

func (w *Writer) parseLogEvent(data []byte) (*sentry.Event, bool) {
	const logger = "zerolog"

	sentryLvl, err := w.extractSentryLvl(data)
	if err != nil {
		return nil, false
	}

	event := sentry.Event{
		Timestamp: now(),
		Level:     sentryLvl,
		Logger:    logger,
		Extra:     map[string]interface{}{},
	}

	err = jsonparser.ObjectEach(data, func(key, value []byte, vt jsonparser.ValueType, offset int) error {
		switch string(key) {
		case zerolog.LevelFieldName, zerolog.TimestampFieldName:
		case zerolog.MessageFieldName:
			event.Message = bytesToStrUnsafe(value)
		case zerolog.ErrorFieldName:
			event.Exception = append(event.Exception, sentry.Exception{
				Value:      bytesToStrUnsafe(value),
				Stacktrace: newStacktrace(),
			})
		default:
			event.Extra[string(key)] = string(value)
		}

		return nil
	})

	if err != nil {
		return nil, false
	}

	return &event, true
}

func (w *Writer) extractSentryLvl(data []byte) (sentryLvl sentry.Level, err error) {
	lvlStr, err := jsonparser.GetUnsafeString(data, zerolog.LevelFieldName)
	if err != nil {
		return
	}

	lvl, err := zerolog.ParseLevel(lvlStr)
	if err != nil {
		return
	}

	_, enabled := w.levels[lvl]
	if !enabled {
		err = errors.New("level disabled")
		return
	}

	sentryLvl, ok := levelsMapping[lvl]
	if !ok {
		err = errors.New("no such sentry level")
		return
	}

	return
}

func newStacktrace() *sentry.Stacktrace {
	const (
		currentModule = "github.com/sveatlo/zerolog-sentry"
		zerologModule = "github.com/rs/zerolog"
	)

	st := sentry.NewStacktrace()

	threshold := len(st.Frames) - 1
	// drop current module frames
	for ; threshold > 0 && st.Frames[threshold].Module == currentModule; threshold-- {
	}

outer:
	// try to drop zerolog module frames after logger call point
	for i := threshold; i > 0; i-- {
		if st.Frames[i].Module == zerologModule {
			for j := i - 1; j >= 0; j-- {
				if st.Frames[j].Module != zerologModule {
					threshold = j
					break outer
				}
			}

			break
		}
	}

	st.Frames = st.Frames[:threshold+1]

	return st
}

func bytesToStrUnsafe(data []byte) string {
	return *(*string)(unsafe.Pointer(&data))
}
