package user

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"{{ .Computed.module_name_final }}/internal/common/config"
	"{{ .Computed.module_name_final }}/internal/common/server"
)

func TestHTTP(t *testing.T) {
	previous := slog.Default()
	var logs bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelError})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	router, err := server.NewRouter(&config.Config{}, newTestModule(t))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/user/1", nil))
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
		body   string
	}{
		{method: http.MethodGet, path: "/user/bad", status: http.StatusBadRequest},
		{method: http.MethodGet, path: "/user/0", status: http.StatusBadRequest},
		{method: http.MethodGet, path: "/user/2", status: http.StatusNotFound, body: `{"msg":"user not found"}`},
		{method: http.MethodGet, path: "/user/3", status: http.StatusInternalServerError},
		{method: http.MethodGet, path: "/user/1/missing", status: http.StatusNotFound},
		{method: http.MethodPost, path: "/user/1", status: http.StatusMethodNotAllowed},
	}
	for _, test := range tests {
		recorder = httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
		if recorder.Code != test.status {
			t.Errorf("%s %s status = %d, want %d", test.method, test.path, recorder.Code, test.status)
		}
		if test.body != "" && recorder.Body.String() != test.body {
			t.Errorf("%s %s body = %q, want %q", test.method, test.path, recorder.Body.String(), test.body)
		}
		if recorder.Header().Get("Content-Type") != "application/json" {
			t.Errorf("%s %s content type = %q", test.method, test.path, recorder.Header().Get("Content-Type"))
		}
	}
	var entry struct{ Msg string }
	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatalf("error log: %s, %v", logs.String(), err)
	}
	if !strings.HasPrefix(entry.Msg, "find user 3 failed: ") {
		t.Fatalf("error summary: %q", entry.Msg)
	}

}
