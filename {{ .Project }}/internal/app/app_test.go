package app

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSignalContextCanBeCanceled(t *testing.T) {
	ctx, cancel := SignalContext()
	cancel()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("signal context was not canceled")
	}
}

func TestNewErrors(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
	if _, err := New(t.Context(), filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing config directory was accepted")
	}
	if _, err := New(t.Context(), writeConfig(t, "log:\n  level: invalid\n")); err == nil || !strings.Contains(err.Error(), "parse log level") {
		t.Fatalf("invalid log error = %v", err)
	}
{{- if .Computed.enable_database_final }}
	if _, err := New(t.Context(), writeConfig(t, "log:\n  level: info\ndatabase:\n  driver: sqlite\n")); err == nil || !strings.Contains(err.Error(), "unsupported database driver") {
		t.Fatalf("invalid database error = %v", err)
	}
{{- end }}
{{- if and .Computed.enable_redis_final (not .Computed.enable_database_final) }}
	if _, err := New(t.Context(), writeConfig(t, "log:\n  level: info\nredis:\n  dsn: ://invalid\n")); err == nil || !strings.Contains(err.Error(), "parse redis dsn") {
		t.Fatalf("invalid redis error = %v", err)
	}
{{- end }}
{{- if and .Computed.enable_trace_final .Computed.enable_database_final }}
	traced := "server:\n  name: test\nlog:\n  level: info\ntracer:\n  enabled: true\n  ratio: 0\n  stdout: true\ndatabase:\n  driver: sqlite\n"
	if _, err := New(t.Context(), writeConfig(t, traced)); err == nil || !strings.Contains(err.Error(), "unsupported database driver") {
		t.Fatalf("traced invalid database error = %v", err)
	}
{{- end }}
}
{{ if and .Computed.enable_trace_final (not .Computed.enable_database_final) (not .Computed.enable_redis_final) -}}
func TestNewWithTracing(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
	dir := writeConfig(t, "server:\n  name: test\nhttp:\n  addr: 127.0.0.1:0\nlog:\n  level: error\ntracer:\n  enabled: true\n  ratio: 0\n  stdout: true\n")
	application, err := New(t.Context(), dir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	application.Close()
}
{{ end }}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestInvalidPaginationConfig(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
	if _, err := New(t.Context(), writeConfig(t, "log:\n  level: info\npagination:\n  maxP: -1\n  maxS: 10000\n")); err == nil || !strings.Contains(err.Error(), "pagination.maxP") {
		t.Fatalf("pagination: %v", err)
	}
}
