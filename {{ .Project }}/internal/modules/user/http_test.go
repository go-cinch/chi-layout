package user

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"{{ .Computed.module_name_final }}/internal/common/config"
	"{{ .Computed.module_name_final }}/internal/common/server"
)

func TestHTTP(t *testing.T) {
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	router, err := server.NewRouter(&config.Config{}, newTestModule(t))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/users/1", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var value User
	if err := json.Unmarshal(recorder.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if value.ID != 1 || value.Name != "Alice" {
		t.Fatalf("user = %#v", value)
	}

	tests := []struct {
		method string
		path   string
		status int
	}{
		{method: http.MethodGet, path: "/users/bad", status: http.StatusBadRequest},
		{method: http.MethodGet, path: "/users/0", status: http.StatusBadRequest},
		{method: http.MethodGet, path: "/users/2", status: http.StatusNotFound},
		{method: http.MethodGet, path: "/users/3", status: http.StatusInternalServerError},
		{method: http.MethodGet, path: "/users/1/missing", status: http.StatusNotFound},
		{method: http.MethodPost, path: "/users/1", status: http.StatusMethodNotAllowed},
	}
	for _, test := range tests {
		recorder = httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
		if recorder.Code != test.status {
			t.Errorf("%s %s status = %d, want %d", test.method, test.path, recorder.Code, test.status)
		}
		if recorder.Header().Get("Content-Type") != "application/json" {
			t.Errorf("%s %s content type = %q", test.method, test.path, recorder.Header().Get("Content-Type"))
		}
	}
}
