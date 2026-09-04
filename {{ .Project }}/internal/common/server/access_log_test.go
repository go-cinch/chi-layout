package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"{{ .Computed.module_name_final }}/internal/common/redact"

	"github.com/go-chi/chi/v5"
)

func TestAccessLogRedactsPath(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	router := chi.NewRouter()
	router.Use(AccessLog(redact.New()))
	router.Get("/reset/{token}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/reset/abcdef", nil))
	if strings.Contains(output.String(), "/reset/abcdef") || !strings.Contains(output.String(), `"path":"/reset/abc***"`) {
		t.Fatalf("log output = %q", output.String())
	}
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/reset/not-sensitive", nil))
	plainRouter := chi.NewRouter()
	plainRouter.Use(AccessLog(redact.New()))
	plainRouter.Get("/users/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	plainRouter.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/users/42", nil))
	request := httptest.NewRequest(http.MethodGet, "/plain", nil)
	if path := redactedPath(request, redact.New()); path != "/plain" {
		t.Fatalf("plain path = %q", path)
	}
}
