#!/bin/sh

set -eu

template_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/chi-layout-test.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

for preset in default full; do
  project="chi-$preset"
  output_dir="$tmp_dir/$preset"
  mkdir -p "$output_dir"

  printf 'Generating preset %s...\n' "$preset"
  scaffold new "$template_dir" \
    --output-dir="$output_dir" \
    --run-hooks=never \
    --no-prompt \
    --preset="$preset" \
    "Project=$project" \
    "module_name=example.com/$project"

  generated="$output_dir/$project"
  (
    cd "$generated"
    test -f .dockerignore
    ! grep -Eq '^/?\.git/?$' .dockerignore
    grep -q '^RUN make build$' Dockerfile
    ! grep -q 'ARG VERSION' Dockerfile
    ! grep -q 'go build.*ldflags' Dockerfile
    make gen
    gofmt -w .
    go mod tidy
    make test
    go build ./...
  )
done

# Exercise the template hook itself. The preset checks above deliberately skip
# hooks so each build step can be asserted independently; this smoke test makes
# sure the user-facing --run-hooks=always path remains executable as well.
project="chi-hook"
output_dir="$tmp_dir/hook"
mkdir -p "$output_dir"
printf 'Generating project with post-scaffold hook enabled...\n'
scaffold new "$template_dir" \
  --output-dir="$output_dir" \
  --run-hooks=always \
  --no-prompt \
  --preset=default \
  "Project=$project" \
  "module_name=example.com/$project"
test -f "$output_dir/$project/internal/common/config/config.gen.go"
test -f "$output_dir/$project/internal/app/modules.gen.go"
test -x "$output_dir/$project/bin/$project"

project="chi-mysql"
output_dir="$tmp_dir/mysql"
mkdir -p "$output_dir"
printf 'Generating MySQL module combination...\n'
scaffold new "$template_dir" \
  --output-dir="$output_dir" \
  --run-hooks=never \
  --no-prompt \
  --preset=full \
  "Project=$project" \
  "module_name=example.com/$project" \
  "database_driver=mysql" \
  "enable_trace=false" \
  "enable_redis=false"
(
  cd "$output_dir/$project"
  make gen
  gofmt -w .
  go mod tidy
  make test
  go build ./...
)

project="chi-redis"
output_dir="$tmp_dir/redis"
mkdir -p "$output_dir"
printf 'Generating Redis-only module combination...\n'
scaffold new "$template_dir" \
  --output-dir="$output_dir" \
  --run-hooks=never \
  --no-prompt \
  --preset=default \
  "Project=$project" \
  "module_name=example.com/$project" \
  "enable_redis=true" \
  "enable_trace=false"
(
  cd "$output_dir/$project"
  make gen
  gofmt -w .
  go mod tidy
  make test
  go build ./...
)

printf 'All chi-layout presets and module combinations passed.\n'
