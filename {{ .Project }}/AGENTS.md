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

- `internal/common` must not import concrete business modules; server{{ if .Computed.enable_grpc_final }}/RPC{{ end }} infrastructure may use the root module contracts.
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
- Use singular nouns for business resource path segments, such as `/game`, `/order`, and `/payment`, including collection endpoints and nested resources. Match the prefix returned by `Name()` to the singular module name.
- Propagate `context.Context` through database, Redis, and downstream calls.
- Response bodies use JSON, errors use `{"msg":"..."}`, and bodyless HTTP 200 responses use `WriteOK(w)`.
- A missing business/database record returns HTTP 404 with `{"msg":"..."}` (for example `{"msg":"game not found"}`). Unmatched routes also return HTTP 404.
- Apply this mapping at the module's HTTP boundary. Return HTTP 400 for malformed input or invalid business fields, and HTTP 500 for unexpected database/server failures; do not globally rewrite error status codes.
- Generated OpenAPI response statuses and body schemas must match the actual HTTP behavior.
- Date/time response fields use `int64` Unix milliseconds since `1970-01-01T00:00:00Z` in Go{{ if .Computed.enable_grpc_final }} and Proto{{ end }}. Convert database time values with `UnixMilli()`.
- HTTP JSON uses numbers.{{ if .Computed.enable_grpc_final }} ProtoJSON uses decimal strings for `int64`.{{ end }} Convert to JavaScript `Number` only within its safe integer range (±(2^53−1)); otherwise use `BigInt` or lossless JSON parsing.
- Do not retain request-scoped values after the handler returns unless copied.

## CRUD and Pagination

Google CRUD naming references:

| Operation | Method pattern | Specification |
| --- | --- | --- |
| Get one | `Get<Resource>` | [AIP-131](https://google.aip.dev/131) |
| List | `List<Resources>` | [AIP-132](https://google.aip.dev/132) |
| Create | `Create<Resource>` | [AIP-133](https://google.aip.dev/133) |
| Update | `Update<Resource>` | [AIP-134](https://google.aip.dev/134) |
| Delete one | `Delete<Resource>` | [AIP-135](https://google.aip.dev/135) |

{{ if .Computed.enable_grpc_final -}}
Use `BatchDelete<Resources>` for bulk RPC deletion ([AIP-235](https://google.aip.dev/235)).
{{ end -}}
HTTP uses singular resource paths, `p`/`s`, and comma-separated IDs for bulk deletion.

Use `p` and `s` for pagination fields in HTTP{{ if .Computed.enable_grpc_final }} and Proto{{ end }} requests/responses. Request fields are optional: omission selects defaults; explicit zero, negative or configured out-of-range values return an empty list. Configure their maxima in `conf/pagination.yml` (`maxP`/`maxS`, both default to 10000). Use the same data-window limits in business methods and Swagger; environment overrides are `SERVICE_PAGINATION_MAXP` and `SERVICE_PAGINATION_MAXS`.

{{ if .Computed.enable_grpc_final -}}
## gRPC

- Store Proto contracts in `api/<service>-proto/`, with `<service>.proto` as the service entry file; generate Go bindings into `api/<service>/`. For published contracts, keep field numbers stable and reserve removed fields.
- Format Proto files with two-space indentation and a blank line between top-level definitions; see the [Proto style guide](https://protobuf.dev/programming-guides/style/).
- Use service-specific Proto packages, such as `catalog.v1`. Preserve imported contracts and paths; `make gen` maps their Go imports to the local module. Shared Google contracts belong in `third_party`.
- Put gRPC adapters in each capability's `grpc.go`. Implement `modules.GRPCModule` and register via `GRPC(grpc.ServiceRegistrar)`; HTTP and gRPC share the business module instance.
- Start gRPC only when business modules register it. gRPC-only modules require neither `Name()` nor `HTTP()`.
- Configure address, unary timeout, shutdown deadline and reflection in `conf/grpc.yml`. Streams use caller deadlines; graceful shutdown has a forced-stop fallback.
- Initialize downstream clients with `rpc.NewClient` in `internal/app/client.go`, using `conf/client.yml`. Store typed clients on `Application`, inject through constructors, and register cleanup for shutdown and startup failures.
- Client unary calls default to 5 seconds and preserve shorter deadlines. Optional health checks require `SERVING`; otherwise initialization is lazy. Clients use TLS by default; set `Insecure: true` for plaintext.

Map business errors at the gRPC boundary:

| Error case | Go status code | Numeric code |
| --- | --- | --- |
| Invalid input, such as a non-positive game ID | `codes.InvalidArgument` | 3 |
| Missing business/database record | `codes.NotFound` | 5 |
| Request canceled | `codes.Canceled` | 1 |
| Request deadline exceeded | `codes.DeadlineExceeded` | 4 |
| Unexpected database/server failure | `codes.Internal` | 13 |

Use `status.FromContextError(err).Err()` for cancellation and deadline errors; keep internal failure details in logs and return a generic message to clients. See the [official complete gRPC status code list](https://grpc.io/docs/guides/status-codes/#the-full-list-of-status-codes) for other cases.

### User context between services

| Information | Location | Example |
| --- | --- | --- |
| Business input that defines the operation or must be persisted | Proto request fields | `owner_id`, `resource_id`, `quantity` |
| Current caller identity and authentication context | gRPC metadata | Verified identity claims or an appropriate credential |

- Distinguish the operator from the business subject. Identify users by stable user IDs; usernames are display/context information.
- Pass the request `ctx` to preserve deadlines and tracing. Add selected identity fields to outgoing metadata explicitly.
- Forward identity from verified context. Before authorizing a request, validate its credential or trust an authenticated calling service; `x-username` alone is not authentication. Never forward all inbound headers or credentials blindly.

Example using `google.golang.org/grpc/metadata` (`SomeRPC` stands for the generated client method):

```go
// Caller: use the verified username; Copy/Set preserves other metadata without duplicates.
username := "abc"
md, _ := metadata.FromOutgoingContext(ctx)
md = md.Copy()
md.Set("x-username", username)
ctx = metadata.NewOutgoingContext(ctx, md)
reply, err := client.SomeRPC(ctx, request)
```

```go
// Receiver: read at the gRPC boundary, after establishing caller trust.
md, _ := metadata.FromIncomingContext(ctx)
username := ""
if values := md.Get("x-username"); len(values) == 1 {
    username = values[0]
}
```

See the [official gRPC metadata guide](https://grpc.io/docs/guides/metadata/).

{{ end -}}
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
{"msg":"GET 404 25ms /game/{id} /game/1","remote_addr":"127.0.0.1:53441","user_agent":"Mozilla/5.0"}
```

Bad: the generic message says little, and the reader must scan separate fields to understand the request.

```json
{"msg":"http request","method":"GET","status":404,"duration_ms":25,"route":"/game/{id}","path":"/game/1","remote_addr":"127.0.0.1:53441","user_agent":"Mozilla/5.0"}
```

## Configuration

- `conf/*.yml` is the source of truth for the configuration model.
- Environment variables override existing YAML scalar leaves using uppercase paths joined by underscores and the `SERVICE_` prefix. Lists use zero-based indices, for example `SERVICE_HTTP_DOCS_SERVERS_2_URL` maps to `http.docs.servers[2].url`; nested lists include each index.
- Preserve untouched list items and sibling fields. Unknown fields and out-of-range indices are ignored; do not grow lists implicitly. Explicitly empty values are valid overrides.
- Share YAML/environment loading between the application and generators through `config.LoadValues`. Redact sensitive values in override logs, including indexed fields.
- Render OpenAPI server URLs and marked pagination range descriptions from runtime configuration, including environment overrides.
- Add project-specific sensitive names to `redact.keys`; never log raw secrets.

## Code Generation

- Run `make config` when configuration fields or types change. Value changes take effect on restart.
- Run `make api` after changing module constructors, HTTP routes, or shared response helper calls.
- Run `make gen` to regenerate configuration,{{ if .Computed.enable_grpc_final }} protobuf bindings,{{ end }} module registrations and HTTP documentation.{{ if .Computed.enable_grpc_final }} Install `protoc` when adding Proto definitions; Go plugins are pinned and installed automatically into `.tools`.{{ end }}
- Generated config, module registries,{{ if .Computed.enable_grpc_final }} `.pb.go` files,{{ end }} and OpenAPI are tool-owned; regenerate them instead of editing by hand.
- Automatic generation tools must print a unified diff when an existing generated file changes; print `create` for a new file and `unchanged` when no change is needed.

## Testing

- Put package-level unit tests beside their implementation files.
- Name each unit test file after the implementation file it covers, such as `game.go` and `game_test.go`; do not add generic package-wide test filenames without a corresponding source file.
- Put cross-package and external-service integration tests in `internal/tests`.
- Use `net/http/httptest` for handlers and routers.
- Prefer external test packages for public behavior; use the implementation package when testing internal helpers is justified.
- Require at least 80% statement coverage for every package under `internal/modules` and at least 60% for every other production package.{{ if .Computed.enable_grpc_final }} Exclude compiler-generated `.pb.go` files from coverage thresholds; test adapters and RPC behavior with generated clients.{{ end }}
- Treat coverage as a guardrail: test meaningful behavior and failure paths, and do not add low-value tests or production indirection only to increase the percentage.
- Run `make lint`, `make test`, and `make build` before committing.
