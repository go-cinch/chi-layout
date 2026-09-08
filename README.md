# Go-Cinch Chi Layout

A minimal, configurable Go HTTP service scaffold built on `net/http`, with
[`go-chi/chi`](https://github.com/go-chi/chi) (default) or
[`gin-gonic/gin`](https://github.com/gin-gonic/gin).

## Presets

| Preset | Included |
| --- | --- |
| `default` | HTTP server, selected router, health check, JSON logging, tracing and pprof |
| `full` | Everything in `default`, plus PostgreSQL/MySQL, SQL migrations, Redis and a Game HTTP/gRPC CRUD example |

## Generate with Make

Clone the layout, then generate a project next to the cloned repository:

```bash
git clone https://github.com/go-cinch/chi-layout.git
cd chi-layout

make default PROJECT=my-service
```

Generate the full version with:

```bash
make full PROJECT=my-service
```

Both presets default to chi. Select Gin independently of the preset:

```bash
make full PROJECT=my-service HTTP_ROUTER=gin
make default PROJECT=my-service HTTP_ROUTER=gin
```

`HTTP_ROUTER` accepts only `chi` or `gin`. Router selection happens at generation
time; generated projects contain only the chosen router implementation and dependency.
Omitting `HTTP_ROUTER` is equivalent to `HTTP_ROUTER=chi`, including for `full`.

Use `OUTPUT_DIR` to select another destination:

```bash
make default PROJECT=my-service OUTPUT_DIR=/path/to/projects
```

## Generate with Scaffold

```bash
scaffold new https://github.com/go-cinch/chi-layout \
  --output-dir=. \
  --run-hooks=always \
  --no-prompt \
  --preset=default \
  Project=my-service
```

Replace `default` with `full` to include the database, migrations, Redis and
the Game example.
Pass `http_router=gin` to `scaffold new` to select Gin, or choose it interactively.
Interactive generation can also adjust the HTTP port, timeout, database driver
and individual features.

## Generated Project

```bash
cd my-service
make gen
make tidy
make test
make run
```

Configuration is defined in `conf/*.yml`. Run `make config` when fields or types
change; configuration values are loaded at startup.

HTTP modules implement `modules.HTTPModule`; gRPC modules implement
`modules.GRPCModule`. Each provides a `New` constructor and a compile-time
interface assertion. `make gen` generates configuration, Proto bindings, module
registrations and OpenAPI. Build, test, lint and run targets invoke it automatically.

## Module Routing

Chi modules expose `HTTP() http.Handler` and construct a chi subrouter. Gin
modules implement `HTTP(r *gin.RouterGroup)` and register relative routes on the
provided group. The application creates one engine and adds the `Name()` prefix.
Business resource paths use singular nouns: the `game` module returns `/game`
from `Name()`, including for collection routes.
Business operations continue to accept `context.Context`; Gin handlers pass
`c.Request.Context()` and use the shared JSON helpers with `c.Writer`.

Gin uses explicit logging, tracing, recovery and timeout middleware, JSON 404/405
responses, and disables automatic trailing-slash redirects. `/docs` explicitly
redirects to `/docs/`. Request timeouts cancel the context; handlers must observe
cancellation. Set `GIN_MODE=release` when running a Gin service in production. Gin startup diagnostics use the existing slog JSON logger: route registrations are INFO, warnings are WARN, other debug messages are DEBUG, and internal errors are ERROR. `log.level` filters these records; release mode retains Gin's suppression of debug startup output.

OpenAPI generation supports literal chi routes, inline `Route`/`Group` callbacks,
Gin `Group` prefixes and native handlers. Gin `:id` and `*filepath` become OpenAPI
path parameters. Use literal route paths and register routes directly inside the
module's `HTTP` method (including inline groups); dynamic paths and registration
in separate helper functions are not inferred. Gin route middleware is skipped
when finding the final handler. Shared JSON helpers and native Gin JSON/status
responses are recognized. Regenerate with `make api`; do not edit generated files.

## Validate the Layout

```bash
make test
```

This validates both routers with both presets, MySQL and Redis-only combinations,
post-generation hooks, invalid router values and dependency isolation.

## Environment Overrides

Environment variables use the `SERVICE_` prefix and uppercase YAML paths joined
with underscores. List indices start at **0**. For example,
`SERVICE_HTTP_DOCS_SERVERS_2_URL` overrides `http.docs.servers[2].url`:

```bash
SERVICE_HTTP_ADDR=:8083 \
SERVICE_HTTP_DOCS_ENABLED=true \
SERVICE_HTTP_DOCS_SERVERS_2_URL=http://127.0.0.1:8083 \
make run
```

Only existing scalar fields and list elements are overridden. Other fields and
list items remain intact. Nested lists use additional numeric segments; unknown
fields, negative/out-of-range indices and attempts to extend empty lists are
ignored. An explicitly empty variable replaces the value with an empty string.
Environment names preserve the YAML field spelling before uppercasing, so
`readHeaderTimeout` maps to `READHEADERTIMEOUT`.

`make api` uses these overrides when generating OpenAPI. At runtime the docs
endpoint uses the final configured server list, so the same variables also work
with a prebuilt binary. Restart the service after changing environment variables.
The Swagger target URL and the HTTP listening address are configured separately.
