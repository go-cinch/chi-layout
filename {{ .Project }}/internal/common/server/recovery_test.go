package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoverer(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	handler := Recoverer()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if recorder.Code != http.StatusInternalServerError || !strings.Contains(output.String(), "panic recovered") {
		t.Fatalf("response = %d, log = %q", recorder.Code, output.String())
	}
}

func TestRecovererLogsAbort(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	handler := Recoverer()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/abort", nil))
	if !strings.Contains(output.String(), "http request aborted") || strings.Contains(output.String(), "stack") {
		t.Fatalf("abort log = %q", output.String())
	}
}
