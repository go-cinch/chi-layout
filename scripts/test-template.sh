#!/bin/sh

set -eu

template_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/chi-layout-test.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

validate_project() (
  generated=$1
  router=$2
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
  make -C "$template_dir" full "PROJECT=$project" "OUTPUT_DIR=$output_dir" "HTTP_ROUTER=$router"
  test -f "$output_dir/$project/internal/common/config/config.gen.go"
  test -f "$output_dir/$project/internal/app/modules.gen.go"
  test -x "$output_dir/$project/bin/$project"
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

printf 'All router, preset, feature, hook and validation checks passed.\n'
