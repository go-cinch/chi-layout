# {{ .Computed.service_name_final }}

A Go service using **{{ .Computed.http_router_final }}** {{ if .Computed.enable_grpc_final }}with optional gRPC{{ else }}for HTTP{{ end }}.

## Run

```bash
make gen
make tidy
make test
make run
```

## API conventions

HTTP JSON keys use snake case, including `created_at` and `updated_at`. Timestamps
are numeric Unix milliseconds. Paginated responses use `items`, `p`, `s`, and `t`
(total count). Missing pagination parameters use defaults; malformed, empty,
duplicate or overflowing int32 values return 400. Valid int32 values outside the
configured page/size limits return an empty list.

Errors contain `error_code` and `msg`. Clients should branch on stable error codes,
not translated text. `Accept-Language` selects `en-US` or `zh-CN`, with English as
the fallback. Responses expose `Content-Language` and `Vary: Accept-Language`.
Expected errors belong to their capability and use `apperror.New`; unexpected
errors return a generic public message. Add translations to both locale catalogs.

## Duplicate POST requests

Modules opt in with `Idempotent() bool` returning true; the Game example opts in.
A non-empty `X-Idempotent` header atomically claims a key for `idempotency.ttl`
(default `1h`). A duplicate returns 409. Missing/empty keys bypass this check;
invalid keys return 400. Keys allow 1–128 ASCII letters, digits, `-`, `_`, `.`, `:`.
Keys are shared across opted-in endpoints in one service deployment. Use a unique
key per logical creation and reuse it for retries of that creation.

Claims happen before the handler, remain consumed on validation/server failures,
and expire by TTL. This feature does not replay the original response. Redis
shares claims across replicas; without Redis, claims are limited to one process.

## Redis namespaces

When Redis is enabled, `redis.prefix` defaults to `dev:<Project>:` using the exact
Project name, independently of service/module overrides. For example,
`Project=order-service` produces `dev:order-service:`. Override it with
`SERVICE_REDIS_PREFIX`; all replicas share it, and different environments/services
use different prefixes. Prefix components allow letters, digits, hyphens,
underscores and dots.

Business code passes relative keys. The Redis wrapper prefixes every exposed
key-bearing command, including each key in `Del` and `Eval`. Lua scripts use
`KEYS`, not hardcoded key names or keys hidden in `ARGV`. Add new commands through
the wrapper with corresponding key-isolation tests.

## Module wiring and migrations

Shared modules and constructors returning errors are initialized in `internal/app`
and stored as concrete pointers on `Application` before `generatedModules()`.
The generator reuses these instances; simple modules can still be constructed
automatically from matching application dependencies.

New migrations use `YYYYMMDDHH-NN-description.sql`, starting at `01` each hour.
The scaffold hook replaces the timestamp placeholder. Never rename or renumber
migrations already committed in a generated service. Transactions use the callback
context with `Store.SQL(txCtx)` throughout; nested `Store.Tx` calls do not reuse a
transaction. Panic paths roll back and preserve the panic.
