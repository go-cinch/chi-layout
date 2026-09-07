package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"{{ .Computed.module_name_final }}/internal/common/config"
{{- if .Computed.enable_database_final }}
	"{{ .Computed.module_name_final }}/internal/infra/db"
{{- end }}
)

func TestNewRouter(t *testing.T) {
	application := &Application{
{{- if .Computed.enable_database_final }}
		db: &db.Store{},
{{- end }}
{{- if .Computed.enable_redis_final }}
		rds: nil,
{{- end }}
	}
	var cfg config.Config
	cfg.HTTP.Docs.Enabled = true
	handler, err := application.NewRouter(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	path := "/missing"
	want := http.StatusNotFound
{{- if .Computed.enable_user_example_final }}
	path = "/user/bad"
	want = http.StatusBadRequest
{{- end }}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	if recorder.Code != want {
		t.Fatalf("status = %d, want %d", recorder.Code, want)
	}
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/docs/", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("docs status = %d", recorder.Code)
	}
}
