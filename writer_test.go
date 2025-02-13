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

var logEventJSON = []byte(`{"level":"error","requestId":"bee07485-2485-4f64-99e1-d10165884ca7","error":"dial timeout","component":"foobar","time":"2020-06-25T17:19:00+03:00","user_id": 12345, "message":"test message"}`)

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
	zl.Error().
		Err(fmt.Errorf("some custom error")).
		Str("component", "test").
		Int("n", 69).
		Dur("lat", 42*time.Second).
		Msg("error test with extra data")
}

func TestParseLogEvent(t *testing.T) {
	ts := time.Now()
	now = func() time.Time { return ts }

	w, err := New(newSentryClient())
	require.Nil(t, err)

	ev, _, err := w.parseLogEvent(logEventJSON)
	require.Nil(t, err)

	assert.Equal(t, ts, ev.Timestamp)
	assert.Equal(t, sentry.LevelError, ev.Level)
	// logger
	assert.Equal(t, "zerolog", ev.Logger)
	// message from message field
	assert.Equal(t, "test message", ev.Message)

	// extra data from other zerolog fields
	assert.Equal(t, "foobar", ev.Extra["component"])

	// exception from error field
	require.Len(t, ev.Exception, 1)
	assert.Equal(t, "dial timeout", ev.Exception[0].Value)
}

func TestParseLogEventSpecialTag(t *testing.T) {
	ts := time.Now()
	now = func() time.Time { return ts }

	w, err := New(newSentryClient(), WithSpecialFieldType("component", SpecialFieldTag))
	require.Nil(t, err)

	ev, _, err := w.parseLogEvent(logEventJSON)
	require.Nil(t, err)

	assert.Equal(t, map[string]string{"component": "foobar"}, ev.Tags)
}

func TestParseLogEventSpecialUserID(t *testing.T) {
	ts := time.Now()
	now = func() time.Time { return ts }

	w, err := New(newSentryClient(), WithSpecialFieldType("user_id", SpecialFieldUserID))
	require.Nil(t, err)

	ev, _, err := w.parseLogEvent(logEventJSON)
	require.Nil(t, err)

	assert.Equal(t, "12345", ev.User.ID)
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
