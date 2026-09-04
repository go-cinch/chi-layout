package app

import (
	"bytes"
	"context"
	"log/slog"
	"net"
	"net/http"
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

func TestRunAndClose(t *testing.T) {
	application := &Application{
		server:         &http.Server{Addr: "127.0.0.1:0", Handler: http.NotFoundHandler(), ReadHeaderTimeout: time.Second},
		profilerServer: &http.Server{Addr: "127.0.0.1:0", Handler: http.NotFoundHandler(), ReadHeaderTimeout: time.Second},
	}
	closed := false
	application.cleanups = []func(){func() { closed = true }}
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(20*time.Millisecond, cancel)
	if err := application.Run(ctx); err != nil {
		t.Fatal(err)
	}
	application.Close()
	if !closed {
		t.Fatal("cleanup was not called")
	}
	application = &Application{server: &http.Server{Addr: "127.0.0.1:-1", Handler: http.NotFoundHandler()}}
	if err := application.Run(t.Context()); err == nil || !strings.Contains(err.Error(), "http server") {
		t.Fatalf("listener error = %v", err)
	}
}

func TestRunDoesNotLogBeforeListenSucceeds(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	application := &Application{server: &http.Server{Addr: listener.Addr().String(), Handler: http.NotFoundHandler()}}
	if err := application.Run(t.Context()); err == nil || !strings.Contains(err.Error(), "address already in use") {
		t.Fatalf("Run() error = %v", err)
	}
	if strings.Contains(output.String(), "http server running") {
		t.Fatalf("startup log emitted before listen succeeded: %q", output.String())
	}

	application = &Application{
		server:         &http.Server{Addr: "127.0.0.1:0", Handler: http.NotFoundHandler()},
		profilerServer: &http.Server{Addr: listener.Addr().String(), Handler: http.NotFoundHandler()},
	}
	if err := application.Run(t.Context()); err == nil || !strings.Contains(err.Error(), "listen profiler server") {
		t.Fatalf("profiler Run() error = %v", err)
	}
	if strings.Contains(output.String(), "server running") {
		t.Fatalf("startup log emitted before all listeners succeeded: %q", output.String())
	}

	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if err := (&Application{}).Run(canceled); err != nil {
		t.Fatalf("canceled Run() error = %v", err)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}
