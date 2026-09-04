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
    make gen
    gofmt -w .
    go mod tidy
    make test
    go build ./...
  )
done

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
