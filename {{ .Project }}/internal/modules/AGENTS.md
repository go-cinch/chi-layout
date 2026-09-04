# Business Module Guidelines

## Organization

- Organize business code by capability under `internal/modules/<capability>`.
- Use a short, singular package name such as `user`, `order`, or `payment`.
- Keep one capability cohesive. Its business types, rules, database operations, and transport adapters may share the same Go package.
- Do not create MVC-style `model`, `controller`, `service`, or `repository` directories by default.
- Keep shared technical infrastructure in `internal/common` or `internal/infra`; do not move capability-specific behavior into those directories.

## Module Contract

- HTTP capabilities implement the root `modules.Module` interface and expose their subrouter through `HTTP()`.
- Add a compile-time assertion such as `var _ modules.Module = (*Module)(nil)` beside every module implementation.
- Return the final mount path from `Name()` and provide a package-level `New` constructor.
- Constructor dependency types must match fields on `app.Application`; `make api` wires matching fields automatically.
- Do not impose an `/api` or version prefix inside the module.

## Recommended Layout

Start an HTTP capability with this layout:

```text
internal/modules/user/
  user.go
  http.go
```

- `user.go` contains business types, business rules, and small capability-specific database operations.
- `http.go` contains the public HTTP boundary: Chi router construction, handlers, request decoding, response mapping, and transport-only validation.

If another transport is required, add a transport file in the same package:

```text
internal/modules/user/
  user.go
  http.go
  grpc.go
```

- `grpc.go` contains only the gRPC boundary and delegates to the same business behavior used by HTTP.
- Do not duplicate business rules between `http.go` and `grpc.go`.

## Minimal Layout

Use a single file when the capability has no HTTP router or database queries, or when its behavior is genuinely trivial:

```text
internal/modules/user/
  user.go
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

- Construct module routes with `chi.NewRouter` inside `http.go`.
- Apply capability-wide middleware with `Use`, route-specific middleware with `With`, and nested resource middleware with `Route`.
- Parse transport input and map transport output at the HTTP boundary; keep reusable business decisions in non-transport functions.
- Use the shared JSON response helpers and error shape defined by the root guidelines.
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
