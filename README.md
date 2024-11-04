# slog utilities

Utilities and handlers for the Go `log/slog` package. 

```sh
go get github.com/networkteam/slogutils
```

## Features

### CLI handler

<img src="https://user-images.githubusercontent.com/33351/259967209-8c9583f6-e7c5-4c16-854a-01f45f006be8.png">
<br>

* Unobstrusive minimal logging output for humans (no timestamps, no log levels)
* Supports `ReplaceAttr` as in `slog.HandlerOptions`
* Grouping and quoting of attributes
* Prefixes, colors and paddings can be fully customized
* Supports an additional `slogutils.LevelTrace` level that is below `slog.LevelDebug` and can be used for tracing

<details>
<summary><strong>Example</strong></summary>

```go
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/networkteam/slogutils"
)

func main() {
	slog.SetDefault(slog.New(
		slogutils.NewCLIHandler(os.Stderr, &slogutils.CLIHandlerOptions{
			Level: slogutils.LevelTrace,
		}),
	))

	slog.Info("Starting server", "addr", ":8080", "env", "production")
	slog.Debug("Connected to DB", "db", "myapp", "host", "localhost:5432")
	slog.Warn("Slow request", "method", "GET", "path", "/users", "duration", 497*time.Millisecond)
	slog.Error("DB connection lost", slogutils.Err(errors.New("connection reset")), "db", "myapp")
	// This is a non-standard log level that is below debug
	slog.Log(context.Background(), slogutils.LevelTrace, "Trace message", "foo", "bar")
}

```
</details>

### Context helper

Setting a logger instance with groups / attributes on a context is very useful e.g. in request processing or distributed tracing.

* Use `slogutils.FromContext` to get a logger instance from a context (or `slog.Default()` as a fallback)
* Use `slogutils.WithLogger` to set a logger instance on a context

<details>
<summary><strong>Example</strong></summary>

```go
package main

import (
	"log/slog"
	"net/http"
	"time"
)

func main() {
	// An HTTP middleware with a simple request ID
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger := slogutils.FromContext(r.Context())
		logger.Info("Handling main route")
	})

	http.ListenAndServe(":8080", withRequestID(mux))
}

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := time.Now().UnixNano() // Note: use a better request ID based on UUIDs in production

		// Get the logger from the request context and add the request ID as an attribute,
		// then set the logger back to a new context.
		logger := slogutils.FromContext(r.Context()).With(slog.Group("request", "id", requestID))
		ctx := slogutils.WithLogger(r.Context(), logger)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```
</details>

### PGX tracelog adapter for `slog`

See `adapter/pgx/v5/tracelog`. 

## Acknowledgements

* The output of the CLI handler is based on the CLI handler of the [github.com/apex/log](https://github.com/apex/log/tree/master/handlers/cli) package.
* Some code and tests of the CLI handler implementation were adapted from [github.com/lmittmann/tint](https://github.com/lmittmann/tint).
