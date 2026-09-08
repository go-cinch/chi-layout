# Business Module Guidelines

## Organization

- Organize business code by capability under `internal/modules/<capability>`.
- Use a short, singular package name such as `user`, `order`, or `payment`.
- Keep one capability cohesive. Its business types, rules, database operations, and transport adapters may share the same Go package.
- Do not create MVC-style `model`, `controller`, `service`, or `repository` directories by default.
- Keep shared technical infrastructure in `internal/common` or `internal/infra`; do not move capability-specific behavior into those directories.

## Module Contract

- Keep the root `HTTPModule`, `GRPCModule`, and `Module` contracts together in `internal/modules/module.go`.

{{ if eq .Computed.http_router_final "gin" -}}
- HTTP capabilities implement `modules.HTTPModule` with `HTTP(r *gin.RouterGroup)` and register routes on the provided group.
{{ else -}}
- HTTP capabilities implement the root `modules.HTTPModule` interface and expose their subrouter through `HTTP()`.
{{ end -}}
- gRPC capabilities implement `modules.GRPCModule` with `GRPC(grpc.ServiceRegistrar)`; put the assertion in `grpc.go`. HTTP assertions and `Name()` belong in `http.go`, so either transport can be removed independently.
- Every module provides a package-level `New` constructor. HTTP modules return their mount path from `Name()`.
- Business route resource names must be singular: the `game` module returns `/game` from `Name()`. Use `/game` for the collection and `/game/{id}` (chi) or `/game/:id` (Gin) for one record; nested resource nouns are also singular.
- Constructor dependency types must match fields on `app.Application`; `make api` wires matching fields automatically.
- Use `PATCH /<resource>/{id}` for partial updates. Preserve omitted fields; use pointer fields in HTTP inputs and Proto `optional` fields to distinguish omission from an explicit empty value.
- Do not impose an `/api` or version prefix inside the module.

## Naming and Pagination

- Follow the [root CRUD naming rules and AIP references](../../AGENTS.md#grpc). RPC requests use `<Method>Request`; list responses use `<Method>Response`. Get/Create/Update return the resource; Delete returns `google.protobuf.Empty`.
- Use short business method names within a resource-scoped Go package, such as `Get` and `List`.
- Inject `pagination.Limits` from `conf/pagination.yml` into business modules and documentation. HTTP parses parameter presence and format; business methods decide whether the requested window can return data.
- In chi, inspect keys in `r.URL.Query()`; in Gin, use `c.GetQuery`. Reject malformed query encoding, duplicate numeric parameters and values that cannot be parsed as int32.
- Use Proto `optional` fields and Go pointers to preserve presence. Omitted `p`/`s` select page 1 and size `min(1, maxS)`; explicit values outside the data window return an empty list without querying the database. Format errors return HTTP 400 or gRPC `InvalidArgument`.
- Order paginated results consistently. Pages beyond the data return an empty list with the matching total count; configured out-of-range requests return `total: 0`. Consider [cursor pagination](https://google.aip.dev/158) when deep offsets become expensive.

## Recommended Layout

Start an HTTP capability with this layout:

```text
internal/modules/game/
  game.go
  http.go
```

- `game.go` contains business types, business rules, and small capability-specific database operations.
- `http.go` contains the public HTTP boundary: {{ .Computed.http_router_final }} route registration, handlers, request decoding, response mapping, and transport-only validation.

If another transport is required, add a transport file in the same package:

```text
internal/modules/game/
  game.go
  http.go
  grpc.go
```

- `grpc.go` contains registration and the adapter. Embed the generated unimplemented server on the adapter and call shared business methods.

## Minimal Layout

Use a single file when the capability has no HTTP router or database queries, or when its behavior is genuinely trivial:

```text
internal/modules/game/
  game.go
```

- Do not create empty placeholder files or directories for possible future layers.
- Add `http.go`, `grpc.go`, or other files only when the capability actually gains that boundary.

## Dependencies

- Accept required dependencies explicitly through constructors or handler-maker functions; avoid package globals.
- Pass `context.Context` through database, Redis, gRPC, and downstream HTTP operations.
- Keep interfaces close to the code that consumes them and introduce them only when substitution or isolation is needed.
- Cross-module calls use small consumer-owned interfaces and are wired in `internal/app`.
- Module-specific SQL belongs in the owning capability. `internal/infra/db` contains only generic connection, transaction, creation, and migration support.

## HTTP

{{ if eq .Computed.http_router_final "gin" -}}
- Register routes on the provided `*gin.RouterGroup` inside `http.go`; the application owns the single Gin engine and the module prefix.
- Apply capability-wide middleware with `Use`, nested resource middleware with `Group`, and route-specific middleware before the final handler.
- Use `c.Param` for path parameters and `c.Request.Context()` for downstream operations. Keep `*gin.Context` out of business methods.
- Call `c.Abort()` when middleware rejects a request; writing an error alone does not stop subsequent handlers.
{{ else -}}
- Construct module routes with `chi.NewRouter` inside `http.go`.
- Apply capability-wide middleware with `Use`, route-specific middleware with `With`, and nested resource middleware with `Route`.
{{ end -}}
- Use literal route paths and inline groups inside `HTTP` so `make api` can discover complete paths and final handlers.
- Parse transport input and map transport output at the HTTP boundary; keep reusable business decisions in non-transport functions.
- Follow the root response-format rules and use the shared JSON helpers.
- Translate an expected missing record (`ErrNotFound`, including mapped `sql.ErrNoRows`) into HTTP 404 using `server.WriteError(w, http.StatusNotFound, message)` (or `c.Writer` for Gin). The body remains `{"msg":"..."}`.
- HTTP 404 covers missing records and unmatched routes. Do not convert unexpected SQL/connection errors into a missing-record result; retain HTTP 500 for those failures.
- Cover an existing record, a missing record (404 with message), an unmatched route (404), invalid input (400), and a database failure (500) in HTTP tests.
- Propagate the request context and stop work when it is canceled.

## Growth and File Size

- When a non-generated file approaches 1,000 lines, review whether it contains multiple business behaviors and split it only when doing so improves cohesion and navigation.
- Do not impose a total line limit on a module or package.
- Prefer names such as `registration.go`, `profile.go`, `password.go`, or `permission.go` when splitting by behavior.
- Keep files in the same package until the capability has a real internal boundary that justifies another package.
- Generated files do not participate in line-count guidance.

## Testing

- Prioritize business rules, permission boundaries, state transitions, and failure behavior over tests written only to raise coverage.
- Test exported behavior and HTTP routes with `net/http/httptest`.
- Add database integration coverage when module behavior depends on SQL semantics or constraints.
