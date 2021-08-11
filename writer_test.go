package zlogsentry

import (
	"fmt"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var logEventJSON = []byte(`{"level":"error","requestId":"bee07485-2485-4f64-99e1-d10165884ca7","error":"dial timeout","time":"2020-06-25T17:19:00+03:00","message":"test message"}`)

func newSentryClient() *sentry.Client {
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn: "",
	})
	if err != nil {
		panic(fmt.Sprintf("cannot get sentry client: %v", err))
	}

	return client
}

func TestWithZerolog(t *testing.T) {
	w, err := New(newSentryClient())
	require.Nil(t, err)

	zl := zerolog.New(w).With().Timestamp().Logger()

	zl.Debug().Msg("debug test")
	zl.Error().Msg("error test")
}

func TestParseLogEvent(t *testing.T) {
	ts := time.Now()

	now = func() time.Time { return ts }

	w, err := New(newSentryClient())
	require.Nil(t, err)

	ev, ok := w.parseLogEvent(logEventJSON)
	require.True(t, ok)

	assert.Equal(t, ts, ev.Timestamp)
	assert.Equal(t, sentry.LevelError, ev.Level)
	assert.Equal(t, "zerolog", ev.Logger)
	assert.Equal(t, "test message", ev.Message)

	require.Len(t, ev.Exception, 1)
	assert.Equal(t, "dial timeout", ev.Exception[0].Value)
}

func BenchmarkParseLogEvent(b *testing.B) {
	w, err := New(newSentryClient())
	if err != nil {
		b.Errorf("failed to create writer: %v", err)
	}

	for i := 0; i < b.N; i++ {
		w.parseLogEvent(logEventJSON)
	}
}

func BenchmarkParseLogEvent_DisabledLevel(b *testing.B) {
	w, err := New(newSentryClient(), WithLevels(zerolog.FatalLevel))
	if err != nil {
		b.Errorf("failed to create writer: %v", err)
	}

	for i := 0; i < b.N; i++ {
		w.parseLogEvent(logEventJSON)
	}
}
