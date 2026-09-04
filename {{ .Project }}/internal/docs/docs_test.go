package docs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestModule(t *testing.T) {
	module := New()
	if module.Name() != "/docs" {
		t.Fatalf("Name() = %q", module.Name())
	}
	recorder := httptest.NewRecorder()
	module.HTTP().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if recorder.Code != http.StatusMovedPermanently || recorder.Header().Get("Location") != "/docs/" {
		t.Fatalf("redirect = %d %q", recorder.Code, recorder.Header().Get("Location"))
	}
	recorder = httptest.NewRecorder()
	module.HTTP().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/docs/", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "SwaggerUIBundle") {
		t.Fatalf("index response = %d", recorder.Code)
	}
	recorder = httptest.NewRecorder()
	module.HTTP().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/docs/openapi.yaml", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "openapi: 3.0.3") ||
		!strings.Contains(recorder.Body.String(), `url: "https://example.com/api/{{ .Computed.service_name_kebab }}"`) {
		t.Fatalf("spec response = %d", recorder.Code)
	}
	recorder = httptest.NewRecorder()
	module.HTTP().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/docs/", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method response = %d", recorder.Code)
	}
}
