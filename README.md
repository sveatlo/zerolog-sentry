# zerolog-sentry
[![Build Status](https://travis-ci.org/archdx/zerolog-sentry.svg?branch=master)](https://travis-ci.org/archdx/zerolog-sentry)

### Example
```go
import (
	"errors"
	"io"
	stdlog "log"
	"os"

	"github.com/archdx/zerolog-sentry"
	"github.com/rs/zerolog"
)

func main() {
    _ = sentry.Init(...)

	w, err := zlogsentry.New(sentry.CurrentHub().Client())
	if err != nil {
		stdlog.Fatal(err)
	}

	defer w.Close()

	logger := zerolog.New(io.MultiWriter(w, os.Stdout)).With().Timestamp().Logger()

	logger.Error().Err(errors.New("dial timeout")).Msg("test message")
}

```

