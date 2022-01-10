# zerolog-sentry
[![Build Status](https://travis-ci.org/archdx/zerolog-sentry.svg?branch=master)](https://travis-ci.org/archdx/zerolog-sentry)

### Example
```go
package main

import (
	"errors"
	"io"
	stdlog "log"
	"os"

	"github.com/sveatlo/zerolog-sentry"
	"github.com/rs/zerolog"
	"github.com/getsentry/sentry-go"
)

func main() {
	scope := sentry.NewScope()
	client, _ := sentry.NewClient(sentry.ClientOptions{
		// Either set your DSN here or set the SENTRY_DSN environment variable.
		Dsn: "...",
		// Enable printing of SDK debug messages.
		// Useful when getting started or trying to figure something out.
		Environment: "local",
		Debug:   true,
	})
	_ = sentry.NewHub(client, scope)

	w, err := zlogsentry.New(client1)
	if err != nil {
		stdlog.Fatal(err)
	}

	defer w.Close()

	logger := zerolog.New(io.MultiWriter(w, os.Stdout)).With().Timestamp().Logger()
	logger.Error().Err(errors.New("dial timeout")).Msg("test message")
}

```

