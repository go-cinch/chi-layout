# Go-Cinch Chi Layout

A minimal, configurable Go HTTP service scaffold built on `net/http` and
[`go-chi/chi`](https://github.com/go-chi/chi).

## Presets

| Preset | Included |
| --- | --- |
| `default` | HTTP server, Chi router, health check, JSON logging, tracing and pprof |
| `full` | Everything in `default`, plus PostgreSQL/MySQL, SQL migrations, Redis and a User HTTP/database example |

Neither preset includes jobs or schedulers.

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
the User example.
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

Configuration is defined in `conf/*.yml`. After changing YAML fields, run
`make config` to regenerate `internal/common/config/config.gen.go`.

Business modules implement `Name()` and `HTTP()`, expose a `New` constructor,
and add a compile-time `modules.Module` assertion. Run `make api` to regenerate
the module registry and OpenAPI document. Run `make gen` to execute both
`make config` and `make api`. The build, test, lint and run targets execute
`make gen` automatically.

## Validate the Layout

```bash
make test
```

This validates both presets and supported feature combinations.
