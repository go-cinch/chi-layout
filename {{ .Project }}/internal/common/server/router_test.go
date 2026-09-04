package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"{{ .Computed.module_name_final }}/internal/common/config"
)

type testModule struct {
	name    string
	handler http.Handler
}

func (m testModule) Name() string {
	return m.name
}

func (m testModule) HTTP() http.Handler {
	return m.handler
}

func TestNewRouter(t *testing.T) {
	var cfg config.Config
	cfg.Server.Name = "test"
	cfg.HTTP.Timeout = time.Second
	module := testModule{name: "/module", handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		WriteOK(w, map[string]bool{"ok": true})
	})}
	handler, err := NewRouter(&cfg, module)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/module", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if _, err := NewRouter(&config.Config{}, testModule{name: "invalid", handler: http.NotFoundHandler()}); err == nil {
		t.Fatal("invalid module was accepted")
	}
	if _, err := NewRouter(&config.Config{}, nil); err == nil {
		t.Fatal("nil module was accepted")
	}
	if _, err := NewRouter(&config.Config{}, testModule{name: "/nil-handler"}); err == nil {
		t.Fatal("nil module handler was accepted")
	}
	if _, err := NewRouter(&config.Config{}, module, module); err == nil {
		t.Fatal("duplicate module was accepted")
	}
}
