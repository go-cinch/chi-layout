{{ if eq .Computed.http_router_final "gin" -}}
package server

import (
	"encoding/json"
	"net/http"
	"{{ .Computed.module_name_final }}/internal/common/apperror"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"{{ .Computed.module_name_final }}/internal/common/config"
	"{{ .Computed.module_name_final }}/internal/common/idempotency"
)

type testModule struct {
	name    string
	handler http.Handler
}

func (m testModule) Name() string            { return m.name }
func (m testModule) HTTP(r *gin.RouterGroup) { r.GET("", gin.WrapH(m.handler)) }

func TestNewRouter(t *testing.T) {
	var cfg config.Config
	cfg.Server.Name = "test"
	cfg.HTTP.Timeout = time.Second
	module := testModule{name: "/module", handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { WriteOK(w, map[string]bool{"ok": true}) })}
	handler, err := NewRouter(&cfg, nil, module)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/module", 200}, {"POST", "/module", 405},
		{"GET", "/missing", 404}, {"GET", "/module/", 404},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(test.method, test.path, nil))
		if w.Code != test.status {
			t.Fatalf("%s %s: %d", test.method, test.path, w.Code)
		}
		if w.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("content type: %s", w.Header().Get("Content-Type"))
		}
		if test.status >= 400 {
			var body ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Msg != http.StatusText(test.status) {
				t.Fatalf("error body: %s", w.Body.String())
			}
		}
	}
	if _, err := NewRouter(&cfg, nil, nil); err == nil {
		t.Fatal("nil module accepted")
	}
	if _, err := NewRouter(&cfg, nil, testModule{name: "invalid"}); err == nil {
		t.Fatal("invalid name accepted")
	}
	if _, err := NewRouter(&cfg, nil, module, module); err == nil {
		t.Fatal("duplicate module accepted")
	}
}

type groupedModule struct{}

func (groupedModule) Name() string { return "/team" }
func (groupedModule) HTTP(r *gin.RouterGroup) {
	group := r.Group("/:team")
	group.Use(func(c *gin.Context) {
		if c.Param("team") == "private" {
			c.Abort()
			WriteError(c.Writer, c.Request, 403, apperror.Forbidden)
			return
		}
		c.Next()
	})
	group.GET("/user/:id", func(c *gin.Context) {
		WriteOK(c.Writer, map[string]string{"team": c.Param("team"), "id": c.Param("id")})
	})
}
func TestNewRouterGroupedRoutes(t *testing.T) {
	handler, err := NewRouter(&config.Config{}, nil, groupedModule{})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		path   string
		status int
		body   string
	}{
		{"/team/public/user/42", 200, `{"id":"42","team":"public"}`},
		{"/team/private/user/42", 403, `{"error_code":"HTTP_FORBIDDEN","msg":"Forbidden"}`},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", test.path, nil))
		if w.Code != test.status || w.Body.String() != test.body {
			t.Fatalf("response: %d %s", w.Code, w.Body.String())
		}
	}
}
{{ else -}}
package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"{{ .Computed.module_name_final }}/internal/common/config"
	"{{ .Computed.module_name_final }}/internal/common/idempotency"
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
	handler, err := NewRouter(&cfg, nil, module)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/module", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if _, err := NewRouter(&config.Config{}, nil, testModule{name: "invalid", handler: http.NotFoundHandler()}); err == nil {
		t.Fatal("invalid module was accepted")
	}
	if _, err := NewRouter(&config.Config{}, nil, nil); err == nil {
		t.Fatal("nil module was accepted")
	}
	if _, err := NewRouter(&config.Config{}, nil, testModule{name: "/nil-handler"}); err == nil {
		t.Fatal("nil module handler was accepted")
	}
	if _, err := NewRouter(&config.Config{}, nil, module, module); err == nil {
		t.Fatal("duplicate module was accepted")
	}
}
{{ end -}}

// These routes exercise the router-level opt-in for both generated routers.
type claimModule struct {
    enabled bool
    handled *int
}

func (claimModule) Name() string { return "/claim" }

func (m claimModule) Idempotent() bool { return m.enabled }

{{ if eq .Computed.http_router_final "gin" -}}
func (m claimModule) HTTP(r *gin.RouterGroup) {
    r.POST("", func(c *gin.Context) { m.serve(c.Writer, c.Request) })
}
{{ else -}}
func (m claimModule) HTTP() http.Handler { return http.HandlerFunc(m.serve) }
{{ end -}}

func (m claimModule) serve(w http.ResponseWriter, _ *http.Request) {
    *m.handled++
    w.WriteHeader(http.StatusBadRequest) // Even unsuccessful requests consume a claim.
}

func TestRouterIdempotency(t *testing.T) {
    cfg := &config.Config{}
    cfg.Idempotency.TTL = time.Hour
    count := 0
    module := claimModule{enabled: true, handled: &count}
    store := idempotency.NewMemoryStore()
    if _, err := NewRouter(cfg, nil, module); err == nil { t.Fatal("missing store accepted") }
    for _, ttl := range []time.Duration{0, -time.Second} {
        invalid := *cfg
        invalid.Idempotency.TTL = ttl
        if _, err := NewRouter(&invalid, store, module); err == nil { t.Fatal("non-positive ttl accepted") }
    }
    // Two routers must use the same application-owned store.
    first, err := NewRouter(cfg, store, module)
    if err != nil { t.Fatal(err) }
    second, err := NewRouter(cfg, store, module)
    if err != nil { t.Fatal(err) }
    for index, handler := range []http.Handler{first, second} {
        request := httptest.NewRequest("POST", "/claim", nil)
        request.Header.Set("X-Idempotent", "same-request")
        request.Header.Set("Accept-Language", "zh-CN")
        response := httptest.NewRecorder()
        handler.ServeHTTP(response, request)
        want := 400
        if index == 1 { want = 409 }
        if response.Code != want { t.Fatalf("request %d: %d %s", index, response.Code, response.Body.String()) }
        if index == 1 {
            var body ErrorResponse
            if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.ErrorCode != "HTTP_DUPLICATE_REQUEST" || response.Header().Get("Content-Language") != "zh-CN" {
                t.Fatalf("duplicate error: %s", response.Body.String())
            }
        }
    }
    if count != 1 { t.Fatalf("handler called %d times", count) }
    module.enabled = false
    handler, err := NewRouter(&config.Config{}, nil, module)
    if err != nil { t.Fatal(err) }
    for range 2 {
        request := httptest.NewRequest("POST", "/claim", nil)
        request.Header.Set("X-Idempotent", "same-request")
        response := httptest.NewRecorder()
        handler.ServeHTTP(response, request)
        if response.Code != 400 { t.Fatalf("unclaimed request: %d", response.Code) }
    }
    if count != 3 { t.Fatalf("opt-out handler count: %d", count) }
}
