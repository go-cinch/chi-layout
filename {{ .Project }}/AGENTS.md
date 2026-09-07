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

{{ if eq .Computed.http_router_final "gin" -}}
- Modules implement `HTTP(*gin.RouterGroup)` and register relative paths on the group provided by the application.
- Keep Gin routing and `*gin.Context` inside the HTTP boundary; pass `c.Request.Context()` to business code.
- Use the shared JSON helpers with `c.Writer` and call `c.Abort()` when middleware stops a request.
{{ else -}}
- Modules expose `http.Handler` and are mounted in `internal/app`.
- Keep chi-specific routing inside the HTTP boundary.
{{ end -}}
- Use singular nouns for business resource path segments, such as `/user`, `/order`, and `/payment`, including collection endpoints and nested resources. Match the prefix returned by `Name()` to the singular module name.
- Propagate `context.Context` through database, Redis, and downstream calls.
- Response bodies use JSON, errors use `{"msg":"..."}`, and bodyless HTTP 200 responses use `WriteOK(w)`.
- A missing business/database record returns HTTP 404 with `{"msg":"..."}` (for example `{"msg":"user not found"}`). Unmatched routes also return HTTP 404.
- Apply this mapping at the module's HTTP boundary. Keep invalid input as HTTP 400 and unexpected database/server failures as HTTP 500; do not globally rewrite error status codes.
- Generated OpenAPI response statuses and body schemas must match the actual HTTP behavior.
- Do not retain request-scoped values after the handler returns unless copied.

## Logging

- Apply these principles to all logs: business operations, infrastructure, startup, configuration, and HTTP requests.
- Put the information the reader needs first in `msg`: what happened, the outcome, and the key values for understanding it. Choose those values for the event rather than following a fixed field checklist.
- Keep messages concise. Use values directly when their meaning and order are clear; include field names only when they resolve ambiguity.
- Add fields only for useful secondary context. Omit irrelevant details, avoid duplicating values already in `msg`, and do not turn every variable into a separate field or add unnecessary wrapper objects.
- Let the shared slog setup supply standard metadata (`time`, `level`, `v`, `caller`) and trace correlation fields.
- Keep fixed prose lowercase, preserve the original case of values, and redact sensitive data before placing it in either `msg` or additional fields.

HTTP example (automatic metadata omitted):

Good: the message immediately shows the method, result, elapsed time, route, and path; client details remain available as secondary fields.

```json
{"msg":"GET 404 25ms /user/{id} /user/1","remote_addr":"127.0.0.1:53441","user_agent":"Mozilla/5.0"}
```

Bad: the generic message says little, and the reader must scan separate fields to understand the request.

```json
{"msg":"http request","method":"GET","status":404,"duration_ms":25,"route":"/user/{id}","path":"/user/1","remote_addr":"127.0.0.1:53441","user_agent":"Mozilla/5.0"}
```

## Configuration

- `conf/*.yml` is the source of truth for the configuration model.
- Environment variables override existing YAML scalar leaves using uppercase paths joined by underscores and the `SERVICE_` prefix. Lists use zero-based indices, for example `SERVICE_HTTP_DOCS_SERVERS_2_URL` maps to `http.docs.servers[2].url`; nested lists include each index.
- Preserve untouched list items and sibling fields. Unknown fields and out-of-range indices are ignored; do not grow lists implicitly. Explicitly empty values are valid overrides.
- Share YAML/environment loading between the application and generators through `config.LoadValues`. Redact sensitive values in override logs, including indexed fields.
- Serve OpenAPI with the final runtime `http.docs.servers` configuration so environment overrides work with prebuilt binaries. Keep generated interface definitions intact and perform runtime rendering in memory.
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
