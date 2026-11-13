# dev-modules

Reusable Go building blocks for application logging, GORM database connections,
Redis, JWT authentication, HTTP middleware, and common utility functions.

## Package guide

| Package | Purpose |
| --- | --- |
| [`logger`](#logger) | Zap console and JSON file logging. |
| [`db`](#gorm-database-connection) | Configured GORM connections and SQL logging. |
| [`redis`](#redis-connection) | Configured standalone Redis connections. |
| [`auth`](#jwt-authentication) | JWT creation, validation, and context claims. |
| [`middleware`](#http-middleware) | Authentication, rate limiting, and HTTP logging. |
| [`utils`](#utility-functions) | Environment, password, ID, date, file, and pointer helpers. |
| [Complete setup](#complete-http-setup) | A combined `net/http` application example. |

## Requirements and installation

- Go version: see the `go` directive in [`go.mod`](go.mod).
- SQLite uses `github.com/mattn/go-sqlite3`, so SQLite builds require CGO and a C
  compiler. PostgreSQL, MySQL, SQL Server, Redis, logging, and middleware do not
  require SQLite at runtime.

Install the module:

```sh
go get github.com/naushadnizarali/dev-modules@latest
```

Import only the packages the application needs:

```go
import (
    "github.com/naushadnizarali/dev-modules/auth"
    "github.com/naushadnizarali/dev-modules/db"
    modulelogger "github.com/naushadnizarali/dev-modules/logger"
    "github.com/naushadnizarali/dev-modules/middleware"
    moduleredis "github.com/naushadnizarali/dev-modules/redis"
    "github.com/naushadnizarali/dev-modules/utils"
)
```

## Logger

The `logger` package creates a Zap logger that writes human-readable entries to
the console, newline-delimited JSON to a file, or both.

### `logger.DefaultConfig`

Returns an `info`-level console configuration with caller information and error
stack traces enabled:

```go
cfg := modulelogger.DefaultConfig()
```

Configuration fields:

| Field | Purpose |
| --- | --- |
| `Level` | Minimum Zap level, such as `debug`, `info`, `warn`, or `error`. |
| `Console` | Enables human-readable console output. |
| `File` | Enables JSON file output. |
| `FilePath` | Required when `File` is true. Parent directories are created. |
| `Development` | Uses Zap's development encoder and behavior. |
| `Caller` | Adds the source file and line to entries. |
| `Stacktrace` | Adds stack traces at error level, or warn level in development. |
| `ConsoleWriter` | Optional `io.Writer`; defaults to `os.Stdout`. Useful in tests. |

### `logger.New`

Returns `(*zap.Logger, error)` and should be used when configuration errors need
to be handled:

```go
cfg := modulelogger.DefaultConfig()
cfg.Level = "debug"
cfg.Console = true
cfg.File = true
cfg.FilePath = "logs/app.log"

log, err := modulelogger.New(cfg)
if err != nil {
    panic(err)
}
defer func() { _ = log.Sync() }()

log.Info("server starting", zap.Int("port", 8080))
```

Use the logger throughout an application through Zap's global logger:

```go
undo := zap.ReplaceGlobals(log)
defer undo()

zap.L().Info("using the configured logger")
```

### `logger.MustNew`

Returns a logger or panics when the configuration is invalid. It is convenient
for configuration that is known at compile time:

```go
log := modulelogger.MustNew(modulelogger.Config{
    Level:   "info",
    Console: true,
})
```

### Global `logger.Log`

For applications that prefer `logger.Log.Info(...)` in every file, initialize
the package-wide logger once in `main`. These methods use `fmt.Printf`-style
format strings:

```go
func main() {
    cfg := modulelogger.DefaultConfig()
    cfg.Level = "info"
    cfg.Console = true
    cfg.File = true
    cfg.FilePath = filepath.Join(
        "logs/app",
        time.Now().Format("app-2006-01-02.log"),
    )

    if err := modulelogger.Init(cfg); err != nil {
        panic(err)
    }
    defer func() { _ = modulelogger.Sync() }()

    modulelogger.Log.Info("server started on port %d", 8080)
    run()
}
```

Then use the same imported package from any application file:

```go
import log "github.com/naushadnizarali/dev-modules/logger"

func createOrder(orderID string) {
    log.Debug("creating order %s", orderID)
    log.Info("created order %s", orderID)
}
```

The package-level `Debug`, `Info`, `Warn`, `Error`, and `Fatal` functions forward
to `logger.Log`. Use `logger.Log.Info(...)` instead when an interface value is
preferred for dependency injection or testing.

`logger.Init` creates a configured logger and installs it for both
`logger.Log` and Zap's `zap.L()`. Do not defer `Sync` inside an initialization
function: that defer would run as soon as initialization returns. Defer
`logger.Sync()` from `main`, so logs remain active for the application lifetime.

`FilePath` must be a complete file path. To create one file per application
start date, format the filename as shown above. The logger creates missing parent
directories automatically.

### `logger.SetGlobal`

Installs an existing `*zap.Logger` for both APIs. Use this when custom Zap
options are added outside this module:

```go
log, err := modulelogger.New(cfg)
if err != nil {
    return err
}
log = log.WithOptions(zap.AddCallerSkip(1))
if err := modulelogger.SetGlobal(log); err != nil {
    return err
}
```

### `logger.Sync`

Flushes the package-wide logger. Call it during shutdown:

```go
defer func() { _ = modulelogger.Sync() }()
```

`logger.Log.Sync()` performs the same operation. `logger.Log` safely discards
entries before initialization, so package startup code does not panic. `Fatal`
logs the message and terminates the process, following Zap's fatal behavior.

## GORM database connection

The `db` package supports PostgreSQL, MySQL/MariaDB, SQLite, and SQL Server. It
opens the selected GORM driver, configures the underlying `database/sql` pool,
and verifies the connection with `PingContext`.

Driver constants are `db.DriverPostgres`, `db.DriverMySQL`, `db.DriverSQLite`,
and `db.DriverSQLServer`. Aliases such as `postgresql`, `pgx`, `mariadb`,
`sqlite3`, and `mssql` are also accepted.

### `db.DefaultConfig`

Returns these pool defaults:

- 10 idle connections
- 100 open connections
- 10-minute maximum idle time
- 1-hour maximum connection lifetime
- GORM log level `warn`
- 200 ms slow-query threshold
- ignored record-not-found errors

The driver and DSN remain required:

```go
cfg := db.DefaultConfig()
cfg.Driver = db.DriverPostgres
cfg.DSN = os.Getenv("DATABASE_DSN")
```

### `db.Open`

Opens and verifies a database connection:

```go
cfg := db.DefaultConfig()
cfg.Driver = db.DriverPostgres
cfg.DSN = "host=localhost user=app password=secret dbname=app port=5432 sslmode=disable"
cfg.PrepareStmt = true
cfg.TranslateError = true
cfg.Logger = zap.L()
cfg.LogLevel = "info" // silent, error, warn, or info
cfg.SlowThreshold = 300 * time.Millisecond
cfg.IgnoreNotFound = true

database, err := db.Open(context.Background(), cfg)
if err != nil {
    zap.L().Fatal("database connection failed", zap.Error(err))
}
defer func() { _ = db.Close(database) }()
```

The DSN format depends on the selected driver:

```go
// MySQL or MariaDB
cfg.Driver = db.DriverMySQL
cfg.DSN = "user:password@tcp(localhost:3306)/app?charset=utf8mb4&parseTime=True&loc=Local"

// SQLite
cfg.Driver = db.DriverSQLite
cfg.DSN = "data/app.db"

// SQL Server
cfg.Driver = db.DriverSQLServer
cfg.DSN = "sqlserver://user:password@localhost:1433?database=app"
```

Set `cfg.Logger` to a logger created by this module to send GORM SQL logs to its
configured console, file, or both. GORM logging is silent when `Logger` is nil.
Each SQL entry includes the statement, elapsed time, row count, and error or
slow-query threshold when applicable.

Pool fields with a value greater than zero override the driver's defaults:
`MaxIdleConns`, `MaxOpenConns`, `ConnMaxIdleTime`, and `ConnMaxLifetime`.

### `db.Close`

Closes the underlying `database/sql` pool. Calling it with nil is safe:

```go
if err := db.Close(database); err != nil {
    zap.L().Error("database shutdown failed", zap.Error(err))
}
```

## Redis connection

The `redis` package creates a standalone `go-redis` client and verifies it with
`PING`.

### `redis.DefaultConfig`

Returns a configuration for `localhost:6379`, RESP 3, five-second dial/read/write
timeouts, and context deadline support:

```go
cfg := moduleredis.DefaultConfig()
```

### `redis.Open`

Connect by URL:

```go
cfg := moduleredis.DefaultConfig()
cfg.URL = os.Getenv("REDIS_URL") // redis://user:password@host:6379/0

redisClient, err := moduleredis.Open(context.Background(), cfg)
if err != nil {
    zap.L().Fatal("Redis connection failed", zap.Error(err))
}
defer func() { _ = redisClient.Close() }()
```

Or configure individual fields:

```go
cfg := moduleredis.DefaultConfig()
cfg.Addr = "localhost:6379"
cfg.Username = "default"
cfg.Password = os.Getenv("REDIS_PASSWORD")
cfg.DB = 1
cfg.ClientName = "orders-api"
cfg.PoolSize = 20
cfg.MinIdleConns = 2
cfg.MaxIdleConns = 10
cfg.MaxActiveConns = 50
cfg.ConnMaxIdleTime = 10 * time.Minute
cfg.ConnMaxLifetime = time.Hour
cfg.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
```

When `URL` is set, its address and credentials take precedence. Explicit pool,
timeout, protocol, and client-name fields are then applied to the parsed options.

## JWT authentication

The `auth` package signs HMAC JWTs, validates their standard claims, supports
application-defined claims, and stores validated claims in a context. Supported
algorithms are HS256, HS384, and HS512. Secrets must contain at least 32 bytes.

### `auth.New`

Creates a reusable JWT manager:

```go
jwtManager, err := auth.New(auth.Config{
    Secret:    []byte(os.Getenv("JWT_SECRET")),
    Algorithm: "HS256",
    Issuer:    "orders-service",
    Audience:  []string{"orders-api"},
    TTL:       15 * time.Minute,
    Leeway:    30 * time.Second,
})
if err != nil {
    panic(err)
}
```

`Now` can supply a custom clock in deterministic tests. Leave it nil in normal
applications.

### `Manager.Generate`

Generates a signed token with the configured TTL:

```go
token, err := jwtManager.Generate("user-42", auth.Claims{
    "role":       "admin",
    "roles":      []string{"admin", "editor"},
    "account_id": 72,
    "verified":   true,
})
```

The subject becomes `sub`. The manager creates `iat`, `nbf`, `exp`, and a random
`jti`, plus configured `iss` and `aud` values. Custom claims cannot replace these
registered claims.

### `Manager.GenerateWithTTL`

Overrides the lifetime for one token, for example a refresh token:

```go
refreshToken, err := jwtManager.GenerateWithTTL(
    "user-42",
    auth.Claims{"token_type": "refresh"},
    30*24*time.Hour,
)
```

### `Manager.Parse`

Validates signature, algorithm, expiry, not-before time, issued-at time, issuer,
and audience, then returns every claim:

```go
claims, err := jwtManager.Parse(token)
if err != nil {
    // Token is missing, malformed, expired, signed incorrectly, or has invalid claims.
}
```

### Context claim functions

Authentication middleware calls `auth.ContextWithClaims` automatically. It can
also be used directly when integrating with another transport:

```go
ctx = auth.ContextWithClaims(ctx, claims)
```

Read the complete map with `ClaimsFromContext` or one untyped value with `Claim`:

```go
claims, found := auth.ClaimsFromContext(r.Context())
value, found := auth.Claim(r.Context(), "custom_claim")
```

Typed helpers return `(value, true)` only when the claim exists and has the
expected representation:

```go
subject, ok := auth.StringClaim(r.Context(), "sub")
verified, ok := auth.BoolClaim(r.Context(), "verified")
roles, ok := auth.StringSliceClaim(r.Context(), "roles")
accountID, ok := auth.Int64Claim(r.Context(), "account_id")
```

`Int64Claim` accepts integral JSON numbers, and `StringSliceClaim` handles arrays
decoded from JWT JSON.

## HTTP middleware

All middleware uses the standard `net/http` signature, so it works with
`http.ServeMux` and routers that accept `http.Handler`.

### `middleware.Authenticate`

Validates a JWT and stores its claims in the request context:

```go
authenticate, err := middleware.Authenticate(middleware.AuthConfig{
    Manager: jwtManager,
    Skip: func(r *http.Request) bool {
        return r.URL.Path == "/health"
    },
})
if err != nil {
    panic(err)
}

protected := authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    subject, ok := auth.StringClaim(r.Context(), "sub")
    if !ok {
        http.Error(w, "missing subject", http.StatusUnauthorized)
        return
    }
    _, _ = w.Write([]byte(subject))
}))
```

The default response is `401 Unauthorized` with `WWW-Authenticate: Bearer`.
Customize it without exposing token-validation details to clients:

```go
Unauthorized: func(w http.ResponseWriter, r *http.Request, err error) {
    zap.L().Warn("authentication failed", zap.Error(err))
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusUnauthorized)
    _, _ = w.Write([]byte(`{"error":"unauthorized"}`))
},
```

### `middleware.BearerToken`

The default token extractor accepts `Authorization: Bearer <token>`:

```go
token, err := middleware.BearerToken(r)
```

It returns `middleware.ErrMissingToken` or
`middleware.ErrInvalidAuthorization` for invalid headers.

### `middleware.CookieToken`

Returns an extractor for a named cookie:

```go
authenticate, err := middleware.Authenticate(middleware.AuthConfig{
    Manager: jwtManager,
    Token:   middleware.CookieToken("access_token"),
})
```

Applications using cookie authentication should also apply their normal CSRF
protections and secure cookie settings.

Any custom extractor can be supplied through `AuthConfig.Token`:

```go
Token: func(r *http.Request) (string, error) {
    return r.URL.Query().Get("token"), nil
},
```

### `middleware.RateLimit`

Creates a Redis-backed atomic fixed-window limiter. Counters are shared across
application instances:

```go
rateLimit, err := middleware.RateLimit(middleware.RateLimitConfig{
    Redis:    redisClient,
    Limit:    100,
    Window:   time.Minute,
    Prefix:   "orders-api",
    FailOpen: false,
    Skip: func(r *http.Request) bool {
        return r.URL.Path == "/health"
    },
})
if err != nil {
    panic(err)
}
```

The default key is the direct peer IP from `RemoteAddr`. Use an authenticated
user or API key when limits should follow identity:

```go
Key: func(r *http.Request) string {
    subject, _ := auth.StringClaim(r.Context(), "sub")
    return subject
},
```

If a reverse proxy supplies client addresses, validate that proxy and provide a
custom `Key`; the default intentionally does not trust forwarded headers.

Allowed and rejected responses contain `X-RateLimit-Limit`,
`X-RateLimit-Remaining`, and `X-RateLimit-Reset`. A rejected response also has
`Retry-After` and status `429`. A Redis error returns `503` unless `FailOpen` is
true, in which case the request continues without rate limiting.

### `middleware.RequestLog`

Logs request and response metadata through Zap:

```go
requestLog := middleware.RequestLog(middleware.RequestLogConfig{
    Logger:          zap.L(),
    RequestIDHeader: "X-Request-ID",
    Skip: func(r *http.Request) bool {
        return r.URL.Path == "/health"
    },
})
```

Each completed request logs the method, path, raw query, remote address, user
agent, request ID, response status, response bytes, and duration. Panics are
logged and rethrown so an outer recovery handler can produce the response.
Request and response bodies and sensitive headers are not logged.

### Middleware order

Middleware is applied from the handler outward. With the default IP-based rate
limit key, this order limits before authentication and logs both auth and
rate-limit responses:

```go
handler := http.Handler(http.HandlerFunc(apiHandler))
handler = authenticate(handler)
handler = rateLimit(handler)
handler = requestLog(handler)

server := &http.Server{
    Addr:              ":8080",
    Handler:           handler,
    ReadHeaderTimeout: 5 * time.Second,
}
```

If the rate-limit `Key` reads JWT claims, place `authenticate` outside
`rateLimit`, so authentication runs first:

```go
handler = rateLimit(http.HandlerFunc(apiHandler))
handler = authenticate(handler)
handler = requestLog(handler)
```

## Utility functions

The `utils` package contains small compatibility helpers. Functions that cannot
return errors use a documented zero-value fallback; use that behavior only when
it fits the calling application.

### Environment and log-level parsing

#### `utils.GetEnv`

Returns a trimmed environment value, or the fallback when unset or blank:

```go
environment := utils.GetEnv("APP_ENV", "development")
```

#### `utils.GetInt`

Parses a base-10 integer environment value. Invalid or missing values return the
fallback:

```go
port := utils.GetInt("PORT", 8080)
```

#### `utils.GetBool`

Uses `strconv.ParseBool`; invalid or missing values return the fallback:

```go
debug := utils.GetBool("DEBUG", false)
```

#### `utils.SplitCSV`

Splits and trims comma-separated values. An empty string or `*` returns
`[]string{"*"}`:

```go
origins := utils.SplitCSV("https://a.example, https://b.example")
```

#### `utils.ParseLevel`

Maps `DEBUG`, `INFO`, `WARN`, and `ERROR` to `slog.Level`. Empty or invalid
input returns `slog.LevelInfo`; invalid input also emits an `slog` warning:

```go
level := utils.ParseLevel(os.Getenv("LOG_LEVEL"))
```

### Passwords, IDs, paths, JSON, and collections

#### `utils.HashPassword`

Hashes a password using bcrypt's default cost. It logs an internal error and
returns an empty string if hashing fails:

```go
hash := utils.HashPassword(password)
if hash == "" {
    return errors.New("could not hash password")
}
```

#### `utils.ComparePassword`

Compares a bcrypt hash with a plain password:

```go
if !utils.ComparePassword(storedHash, suppliedPassword) {
    // Invalid credentials.
}
```

#### `utils.NewUUID`

Returns a UUID version 7 string:

```go
id := utils.NewUUID()
```

#### `utils.GetInitials`

Returns uppercase initials from space- or dot-separated text:

```go
initials := utils.GetInitials("Ada Lovelace") // "AL"
```

#### `utils.GetFilePath`

Prefixes a path with the `ENV` value captured when the `utils` package is
initialized. `ENV` defaults to `development`:

```go
path := utils.GetFilePath("config.json") // development/config.json by default
```

#### `utils.ToJsonString`

Marshals a value to JSON. It returns an empty string if marshaling fails:

```go
payload := utils.ToJsonString(map[string]any{"ready": true})
```

#### `utils.ExtractIDs`

Maps a slice to a slice of comparable identifiers using a generic callback:

```go
type User struct {
    ID uint
}

ids := utils.ExtractIDs(users, func(user User) uint { return user.ID })
```

#### `utils.DownloadFile`

Downloads a URL into a temporary file and returns its local path and the final
URL path's base name. The caller owns and should delete the temporary file:

```go
path, filename, err := utils.DownloadFile("https://example.com/report.pdf")
if err != nil {
    return err
}
defer os.Remove(path)
_ = filename
```

### Date, time, and timezone helpers

Go layouts use the reference time `Mon Jan 2 15:04:05 MST 2006`.

#### `utils.GetCurrentTime`

Returns `time.Now()`:

```go
now := utils.GetCurrentTime()
```

#### `utils.GetFirstDayOfCurrentMonth`

Returns local midnight on the first day of the current month:

```go
first := utils.GetFirstDayOfCurrentMonth()
```

#### `utils.GetCurrentYear`

Returns the current local year:

```go
year := utils.GetCurrentYear()
```

#### `utils.FormatTime`

Formats a time as `YYYY-MM-DD HH:MM:SS`:

```go
formatted := utils.FormatTime(time.Now())
```

#### `utils.ExcelDateToTime`

Converts an Excel serial number or one of the supported string formats to
`YYYY-MM-DD`:

```go
date, err := utils.ExcelDateToTime(45663.0)
date, err = utils.ExcelDateToTime("6/Jan/25")
```

Supported strings include `2/Jan/06`, `DD/MM/YYYY`, `M/D/YYYY`, `YYYY-MM-DD`,
`YYYY/MM/DD`, and numeric Excel serial strings.

#### `utils.FormatDate`

Formats the current time with the supplied Go layout. An empty layout uses
`DD/Mon/YYYY`:

```go
today := utils.FormatDate("")
isoDate := utils.FormatDate("2006-01-02")
```

#### `utils.FormatDate2`

Formats a time pointer. Nil returns an empty string and an empty layout uses
`DD/Mon/YYYY`:

```go
formatted := utils.FormatDate2(&date, "2006-01-02")
```

#### `utils.ParseDate`

Parses `YYYY-MM-DD HH:MM:SS`. A nil input returns `(nil, nil)`:

```go
value := "2026-11-13 14:30:00"
parsed, err := utils.ParseDate(&value)
```

#### `utils.ParseToSQLDate`

Parses an RFC3339-nanosecond timestamp and formats `YYYY-MM-DD`. Empty input
returns an empty string. This helper does not return parse errors, so validate
untrusted timestamps before calling it:

```go
date := utils.ParseToSQLDate("2026-11-13T14:30:00Z")
```

#### `utils.FormatDateStrToDate`

Parses RFC3339 and returns `YYYY-MM-DD` with an error on invalid input:

```go
date, err := utils.FormatDateStrToDate("2026-11-13T14:30:00Z")
```

#### `utils.AppLocation`

Loads the timezone in `APP_TIMEZONE`, defaulting to UTC. An invalid timezone logs
a warning through `zap.L()` and returns UTC:

```go
location := utils.AppLocation()
```

#### `utils.TimeNow`

Returns the current time in `AppLocation`:

```go
localNow := utils.TimeNow()
```

### Numeric and pointer helpers

#### `utils.StringToUint`

Parses an unsigned integer and returns zero on invalid input:

```go
id := utils.StringToUint("42")
```

#### `utils.StringToUintPtr`

Parses an unsigned integer pointer and returns nil on invalid input:

```go
id := utils.StringToUintPtr("42")
```

#### `utils.StringToUint32`

Parses a 32-bit unsigned integer and returns zero on invalid or overflowing input:

```go
value := utils.StringToUint32("42")
```

#### `utils.StringToUint32Ptr`

Parses a 32-bit unsigned integer pointer and returns nil on invalid input:

```go
value := utils.StringToUint32Ptr("42")
```

#### `utils.StringToUint64`

Parses a 64-bit unsigned integer and returns zero on invalid input:

```go
value := utils.StringToUint64("42")
```

#### `utils.StringPtr`

Dereferences a string pointer and returns an empty string for nil:

```go
value := utils.StringPtr(optionalString)
```

#### `utils.UIntPtr`

Dereferences a uint pointer. The pointer must be non-nil:

```go
if optionalID != nil {
    id := utils.UIntPtr(optionalID)
    _ = id
}
```

## Complete HTTP setup

This example combines logging, PostgreSQL, Redis, JWT authentication, rate
limiting, request logging, and graceful resource cleanup:

```go
package main

import (
    "context"
    "net/http"
    "os"
    "time"

    "github.com/naushadnizarali/dev-modules/auth"
    "github.com/naushadnizarali/dev-modules/db"
    modulelogger "github.com/naushadnizarali/dev-modules/logger"
    "github.com/naushadnizarali/dev-modules/middleware"
    moduleredis "github.com/naushadnizarali/dev-modules/redis"
    "go.uber.org/zap"
)

func main() {
    logConfig := modulelogger.DefaultConfig()
    logConfig.File = true
    logConfig.FilePath = "logs/app.log"
    log, err := modulelogger.New(logConfig)
    if err != nil {
        panic(err)
    }
    defer func() { _ = log.Sync() }()
    undo := zap.ReplaceGlobals(log)
    defer undo()

    databaseConfig := db.DefaultConfig()
    databaseConfig.Driver = db.DriverPostgres
    databaseConfig.DSN = os.Getenv("DATABASE_DSN")
    databaseConfig.Logger = log
    database, err := db.Open(context.Background(), databaseConfig)
    if err != nil {
        log.Fatal("database connection failed", zap.Error(err))
    }
    defer func() { _ = db.Close(database) }()

    redisConfig := moduleredis.DefaultConfig()
    redisConfig.URL = os.Getenv("REDIS_URL")
    redisClient, err := moduleredis.Open(context.Background(), redisConfig)
    if err != nil {
        log.Fatal("Redis connection failed", zap.Error(err))
    }
    defer func() { _ = redisClient.Close() }()

    jwtManager, err := auth.New(auth.Config{
        Secret:   []byte(os.Getenv("JWT_SECRET")),
        Issuer:   "example-service",
        Audience: []string{"example-api"},
        TTL:      15 * time.Minute,
    })
    if err != nil {
        log.Fatal("JWT configuration failed", zap.Error(err))
    }

    authenticate, err := middleware.Authenticate(middleware.AuthConfig{
        Manager: jwtManager,
        Skip: func(r *http.Request) bool { return r.URL.Path == "/health" },
    })
    if err != nil {
        log.Fatal("auth middleware failed", zap.Error(err))
    }
    rateLimit, err := middleware.RateLimit(middleware.RateLimitConfig{
        Redis:  redisClient,
        Limit:  100,
        Window: time.Minute,
        Skip: func(r *http.Request) bool { return r.URL.Path == "/health" },
        Key: func(r *http.Request) string {
            subject, _ := auth.StringClaim(r.Context(), "sub")
            return subject
        },
    })
    if err != nil {
        log.Fatal("rate-limit middleware failed", zap.Error(err))
    }

    mux := http.NewServeMux()
    mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(http.StatusNoContent)
    })
    mux.HandleFunc("GET /me", func(w http.ResponseWriter, r *http.Request) {
        subject, _ := auth.StringClaim(r.Context(), "sub")
        _, _ = w.Write([]byte(subject))
    })

    var handler http.Handler = mux
    handler = rateLimit(handler)
    handler = authenticate(handler)
    handler = middleware.RequestLog(middleware.RequestLogConfig{Logger: log})(handler)

    server := &http.Server{
        Addr:              ":8080",
        Handler:           handler,
        ReadHeaderTimeout: 5 * time.Second,
    }
    log.Fatal("HTTP server stopped", zap.Error(server.ListenAndServe()))
}
```

## Validation

Run the repository checks with:

```sh
go test ./...
go vet ./...
```

Redis middleware tests start a temporary loopback Redis server. Environments
that prohibit local sockets must grant that test process loopback access.

## License

See [`LICENSE`](LICENSE).
