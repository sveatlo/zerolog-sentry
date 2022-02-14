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

	levels        map[zerolog.Level]struct{}
	specialFields map[string]SpecialFieldType
	flushTimeout  time.Duration

	name string
}

func New(client *sentry.Client, opts ...WriterOption) (*Writer, error) {
	return NewWithName(client, "zerolog", opts...)
}

func NewWithName(client *sentry.Client, name string, opts ...WriterOption) (*Writer, error) {
	cfg := newDefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	levels := make(map[zerolog.Level]struct{}, len(cfg.levels))
	for _, lvl := range cfg.levels {
		levels[lvl] = struct{}{}
	}

	specialFields := make(map[string]SpecialFieldType, len(cfg.specialFields))
	for k, t := range cfg.specialFields {
		specialFields[k] = t
	}

	return &Writer{
		client:        client,
		levels:        levels,
		specialFields: specialFields,
		flushTimeout:  cfg.flushTimeout,
		name:          name,
	}, nil
}

func (w *Writer) Write(data []byte) (int, error) {
	event, hint, err := w.parseLogEvent(data)
	if err == nil {
		w.client.CaptureEvent(event, hint, nil)
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

func (w *Writer) parseLogEvent(data []byte) (event *sentry.Event, eventHint *sentry.EventHint, err error) {
	sentryLvl, err := w.extractSentryLvl(data)
	if err != nil {
		return
	}

	event = &sentry.Event{
		Timestamp: now(),
		Level:     sentryLvl,
		Logger:    w.name,
		Extra:     map[string]interface{}{},
		Tags:      map[string]string{},
	}
	eventHint = &sentry.EventHint{}

	err = jsonparser.ObjectEach(data, func(keyRaw, valueRaw []byte, vt jsonparser.ValueType, offset int) error {
		key := string(keyRaw)
		// value := string(valueRaw)
		value := bytesToStrUnsafe(valueRaw)

		switch key {
		case zerolog.LevelFieldName, zerolog.TimestampFieldName:
		case zerolog.MessageFieldName:
			event.Message = value
		case zerolog.ErrorFieldName:
			event.Exception = append(event.Exception, sentry.Exception{
				Value:      value,
				Stacktrace: newStacktrace(),
			})
		default:
			// first check if field is of special type
			specialType, ok := w.specialFields[key]
			if !ok {
				// not special => add to additional data
				event.Extra[key] = value
				return nil
			}

			switch specialType {
			case SpecialFieldTag:
				event.Tags[key] = value
			case SpecialFieldUserID:
				event.User = sentry.User{
					ID: value,
				}
			}
		}

		return nil
	})

	return
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
		module       = "github.com/sveatlo/zerolog-sentry"
		loggerModule = "github.com/rs/zerolog"
	)

	st := sentry.NewStacktrace()

	threshold := len(st.Frames) - 1
	// drop current module frames
	for ; threshold > 0 && st.Frames[threshold].Module == module; threshold-- {
	}

outer:
	// try to drop zerolog module frames after logger call point
	for i := threshold; i > 0; i-- {
		if st.Frames[i].Module == loggerModule {
			for j := i - 1; j >= 0; j-- {
				if st.Frames[j].Module != loggerModule {
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
