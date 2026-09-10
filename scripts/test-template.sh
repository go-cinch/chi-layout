#!/bin/sh

set -eu

template_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/chi-layout-test.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

validate_project() (
  generated=$1
  router=$2
  grpc=${3:-true}
  cd "$generated"
  test -f .dockerignore
  ! rg -q '^/?\.git/?$' .dockerignore
  rg -q '^RUN make build$' Dockerfile
  ! rg -q 'ARG VERSION|go build.*ldflags' Dockerfile
  cp internal/docs/openapi.yaml "$tmp_dir/expected-openapi.yaml"
  make gen
  cmp internal/docs/openapi.yaml "$tmp_dir/expected-openapi.yaml"
  gofmt -w .
  go mod tidy
  make lint test build
  go list -m all > "$tmp_dir/dependencies.txt"
  if [ "$grpc" = false ]; then
    test ! -e conf/grpc.yml
    test ! -e internal/common/rpc
    test ! -e internal/cmd/protogen
    test ! -e api
    test ! -e third_party
    test ! -e .tools
    test ! -e internal/modules/game/grpc.go
    test ! -e internal/modules/game/grpc_test.go
    if rg -ni 'grpc|protobuf|protogen' internal/app internal/modules internal/cmd AGENTS.md Makefile Dockerfile README.md; then
      echo "gRPC references remain in an HTTP-only project" >&2
      exit 1
    fi
    if [ ! -e conf/tracer.yml ]; then
      if rg -q '^google.golang.org/grpc ' "$tmp_dir/dependencies.txt"; then
        echo "gRPC dependency remains with gRPC and tracing disabled" >&2
        exit 1
      fi
      # Gin's protobuf HTTP binding can retain protobuf independently of gRPC.
      if [ "$router" = chi ] && rg -q '^google.golang.org/protobuf ' "$tmp_dir/dependencies.txt"; then
        echo "protobuf dependency remains in the minimal chi project" >&2
        exit 1
      fi
    fi
  else
    test -f conf/grpc.yml
    test -f internal/common/rpc/server.go
    test -f internal/cmd/protogen/main.go
  fi
  if [ "$router" = gin ]; then
    rg -q '^github.com/gin-gonic/gin ' "$tmp_dir/dependencies.txt"
    ! rg -q '^github.com/go-chi/chi' "$tmp_dir/dependencies.txt"
    test ! -f internal/common/server/writer.go
  else
    rg -q '^github.com/go-chi/chi/v5 ' "$tmp_dir/dependencies.txt"
    ! rg -q '^github.com/gin-gonic/gin ' "$tmp_dir/dependencies.txt"
  fi
)

for router in chi gin; do
  for preset in default full; do
    project="$router-$preset"
    output_dir="$tmp_dir/$project"
    mkdir -p "$output_dir"
    printf 'Generating %s with preset %s...\n' "$router" "$preset"
    # Omitting the selector exercises both presets' chi default.
    set --
    if [ "$router" = gin ]; then set -- "http_router=gin"; fi
    scaffold new "$template_dir" \
      --output-dir="$output_dir" --run-hooks=never --no-prompt \
      --preset="$preset" "Project=$project" "module_name=example.com/$project" "$@"
    validate_project "$output_dir/$project" "$router"
    if [ "$preset" = default ]; then
      test ! -e "$output_dir/$project/.tools/protoc-gen-go"
    else
      test -f "$output_dir/$project/api/$project-proto/$project.proto"
      test ! -f "$output_dir/$project/api/$project-proto/$project.pb.go"
      test -f "$output_dir/$project/api/$project/$project.pb.go"
      test -f "$output_dir/$project/api/$project/${project}_grpc.pb.go"
    fi
  done

  for preset in default full; do
    project="$router-$preset-http-only"
    output_dir="$tmp_dir/$project"
    mkdir -p "$output_dir"
    printf 'Generating %s with preset %s and gRPC disabled...\n' "$router" "$preset"
    set --
    if [ "$preset" = default ]; then set -- "enable_trace=false"; fi
    if [ "$preset" = full ] && [ "$router" = gin ]; then set -- "database_driver=mysql"; fi
    scaffold new "$template_dir" \
      --output-dir="$output_dir" --run-hooks=never --no-prompt --preset="$preset" \
      "Project=$project" "http_router=$router" "enable_grpc=false" "$@"
    validate_project "$output_dir/$project" "$router" false
    if [ "$preset" = full ]; then
      test -f "$output_dir/$project/internal/modules/game/http.go"
      test -f "$output_dir/$project/internal/infra/db/migrations/YYYYMMDDHH-game.sql"
      rg -q '/game/\{id\}' "$output_dir/$project/internal/docs/openapi.yaml"
    fi
  done

  project="$router-mysql"
  output_dir="$tmp_dir/$project"
  mkdir -p "$output_dir"
  printf 'Generating %s with MySQL and tracing disabled...\n' "$router"
  scaffold new "$template_dir" \
    --output-dir="$output_dir" --run-hooks=never --no-prompt --preset=full \
    "Project=$project" "module_name=example.com/$project" "http_router=$router" \
    "database_driver=mysql" "enable_trace=false" "enable_redis=false"
  validate_project "$output_dir/$project" "$router"

  project="$router-redis"
  output_dir="$tmp_dir/$project"
  mkdir -p "$output_dir"
  printf 'Generating %s with Redis only and health/tracing disabled...\n' "$router"
  scaffold new "$template_dir" \
    --output-dir="$output_dir" --run-hooks=never --no-prompt --preset=default \
    "Project=$project" "module_name=example.com/$project" "http_router=$router" \
    "enable_redis=true" "enable_trace=false" "enable_health_check=false"
  validate_project "$output_dir/$project" "$router"

  # Exercise the public Make entry point and post-scaffold hook for each router.
  project="$router-hook"
  output_dir="$tmp_dir/$project"
  mkdir -p "$output_dir"
  printf 'Generating %s through Make with hooks...\n' "$router"
  grpc=true
  if [ "$router" = gin ]; then grpc=false; fi
  make -C "$template_dir" full "PROJECT=$project" "OUTPUT_DIR=$output_dir" "HTTP_ROUTER=$router" "ENABLE_GRPC=$grpc"
  test -f "$output_dir/$project/internal/common/config/config.gen.go"
  test -f "$output_dir/$project/internal/app/modules.gen.go"
  test -x "$output_dir/$project/bin/$project"
done

# Exercise removal of either transport from a full generated module. Keep only
# generated files under management; developers' adapter files are never deleted by gen.
for transport in http grpc; do
  project="only-$transport"
  output_dir="$tmp_dir/$project"
  mkdir -p "$output_dir"
  scaffold new "$template_dir" --output-dir="$output_dir" --run-hooks=never --no-prompt --preset=full "Project=$project"
  generated="$output_dir/$project"
  (
    cd "$generated"
    make gen
    if [ "$transport" = grpc ]; then
      rm internal/modules/game/http.go internal/modules/game/http_test.go
    else
      rm internal/modules/game/grpc.go internal/modules/game/grpc_test.go api/$project-proto/$project.proto
    fi
    make gen
    gofmt -w .
    go mod tidy
    if [ "$transport" = grpc ]; then
      ! rg -q 'httpModules = append' internal/app/modules.gen.go
      rg -q 'grpcModules = append' internal/app/modules.gen.go
      ! rg -q '/game/\{id\}' internal/docs/openapi.yaml
    else
      ! rg -q 'grpcModules = append' internal/app/modules.gen.go
      test ! -f api/$project/$project.pb.go
      test ! -f api/$project/${project}_grpc.pb.go
    fi
    go test -race ./internal/modules/... ./internal/common/rpc ./internal/cmd/modulegen
    go build ./...
  )
done

printf 'Checking invalid router values...\n'
if make -C "$template_dir" full HTTP_ROUTER=invalid "OUTPUT_DIR=$tmp_dir/invalid-make" > "$tmp_dir/invalid-make.log" 2>&1; then
  cat "$tmp_dir/invalid-make.log"
  exit 1
fi
rg -q 'HTTP_ROUTER must be chi or gin' "$tmp_dir/invalid-make.log"
if scaffold new "$template_dir" --output-dir="$tmp_dir/invalid-scaffold" \
  --run-hooks=never --no-prompt --preset=full Project=invalid http_router=invalid > "$tmp_dir/invalid-scaffold.log" 2>&1; then
  cat "$tmp_dir/invalid-scaffold.log"
  exit 1
fi
rg -q 'http_router must be chi or gin' "$tmp_dir/invalid-scaffold.log"
test ! -e "$tmp_dir/invalid-scaffold/invalid/go.mod"

printf 'Checking invalid gRPC selector values...\n'
if make -C "$template_dir" default ENABLE_GRPC=invalid "OUTPUT_DIR=$tmp_dir/invalid-grpc-make" > "$tmp_dir/invalid-grpc-make.log" 2>&1; then
  cat "$tmp_dir/invalid-grpc-make.log"
  exit 1
fi
rg -q 'ENABLE_GRPC must be true or false' "$tmp_dir/invalid-grpc-make.log"
if scaffold new "$template_dir" --output-dir="$tmp_dir/invalid-grpc-scaffold" \
  --run-hooks=never --no-prompt --preset=default Project=invalid enable_grpc=invalid > "$tmp_dir/invalid-grpc-scaffold.log" 2>&1; then
  cat "$tmp_dir/invalid-grpc-scaffold.log"
  exit 1
fi
rg -q 'enable_grpc must be true or false' "$tmp_dir/invalid-grpc-scaffold.log"
test ! -e "$tmp_dir/invalid-grpc-scaffold/invalid/go.mod"

printf 'All router, preset, feature, hook and validation checks passed.\n'
