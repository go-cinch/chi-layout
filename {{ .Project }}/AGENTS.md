# Development Guidelines

## Design

- Keep `cmd/*` thin: parse flags, initialize the application, and run it.
- Put process wiring and lifecycle code in `internal/app`.
- Put reusable technical infrastructure in `internal/common` and `internal/infra`.
- Organize business code by capability under `internal/modules/<capability>`.
- Do not introduce global `handler`, `service`, `repository`, or `model` folders.
- Add internal layers inside a capability only when that capability needs them.

## Scoped Guidelines

- Business module changes must follow [`internal/modules/AGENTS.md`](internal/modules/AGENTS.md).
{{ if .Computed.enable_database_final }}
- Database changes must follow [`internal/infra/db/AGENTS.md`](internal/infra/db/AGENTS.md).
{{ end }}

## Dependencies

- `internal/common` must not import concrete business module packages; `internal/common/server` may depend on the root `internal/modules` contract.
- Business modules may depend on common infrastructure and small interfaces.
- Wire dependencies explicitly in `internal/app`; avoid package globals.

## HTTP

- Modules expose `http.Handler` and are mounted in `internal/app`.
- Keep chi-specific routing inside the HTTP boundary.
- Propagate `context.Context` through database, Redis, and downstream calls.
- Response bodies use JSON, errors use `{"msg":"..."}`, and bodyless HTTP 200 responses use `WriteOK(w)`.
- Do not retain request-scoped values after the handler returns unless copied.

## Logging

- Write all log message text in lowercase.
- Prefer `component action: value` messages for one-off explanatory values; use attributes only for reusable query dimensions.

## Configuration

- `conf/*.yml` is the source of truth for the configuration model.
- Add project-specific sensitive names to `redact.keys`; never log raw secrets.
- Configure published OpenAPI servers under `http.docs.servers`.

## Code Generation

- Run `make config` after changing `conf/*.yml`.
- Run `make api` after changing module constructors, HTTP routes, or shared response helper calls.
- Run `make gen` to regenerate both configuration and API artifacts.
- Never edit `internal/common/config/config.gen.go`, `internal/app/modules.gen.go`, or `internal/docs/openapi.yaml` manually.
- Automatic generation tools must print a unified diff when an existing generated file changes; print `create` for a new file and `unchanged` when no change is needed.

## Testing

- Put package-level unit tests beside their implementation files.
- Name each unit test file after the implementation file it covers, such as `user.go` and `user_test.go`; do not add generic package-wide test filenames without a corresponding source file.
- Put cross-package and external-service integration tests in `internal/tests`.
- Use `net/http/httptest` for handlers and routers.
- Prefer external test packages for public behavior; use the implementation package when testing internal helpers is justified.
- Require at least 80% statement coverage for every package under `internal/modules` and at least 60% for every other production package.
- Treat coverage as a guardrail: test meaningful behavior and failure paths, and do not add low-value tests or production indirection only to increase the percentage.
- Run `make lint`, `make test`, and `make build` before committing.
