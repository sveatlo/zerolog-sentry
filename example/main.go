package main

import (
	"errors"
	"io"
	stdlog "log"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
	zlogsentry "github.com/sveatlo/zerolog-sentry"
)

var Version = "example@0.0.0"

func main() {

	var sentryClient *sentry.Client
	{
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "cannot get hostname"
		}

		sentryClient, err = sentry.NewClient(sentry.ClientOptions{
			Dsn:        "https://db7560b821e24219a791ed2b88e95063@o303183.ingest.sentry.io/5905197",
			ServerName: hostname,
			Release:    Version,
		})
		if err != nil {
			stdlog.Printf("cannot create sentry client (sentry will be disabled): %v", err)
		}

		defer sentryClient.Flush(5 * time.Second)
	}

	// logging
	var log zerolog.Logger
	{
		var writers = []io.Writer{
			zerolog.ConsoleWriter{
				Out:        os.Stderr,
				TimeFormat: time.RFC3339,
			},
		}

		if sentryClient != nil {
			w, err := zlogsentry.New(sentryClient)
			if err != nil {
				stdlog.Printf("[ERROR] sentry initialization failed: %v", err)
			}

			defer w.Close()

			writers = append(writers, w)
		}

		log = zerolog.New(zerolog.MultiLevelWriter(writers...)).With().Timestamp().Logger()
	}

	log.Error().Err(errors.New("cannot do that one thing")).Str("component", "example").Msg("doing that one thing failed")
}
