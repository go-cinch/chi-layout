{{ if eq .Computed.http_router_final "gin" -}}
package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"{{ .Computed.module_name_final }}/internal/common/config"
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
	handler, err := NewRouter(&cfg, module)
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
	if _, err := NewRouter(&cfg, nil); err == nil {
		t.Fatal("nil module accepted")
	}
	if _, err := NewRouter(&cfg, testModule{name: "invalid"}); err == nil {
		t.Fatal("invalid name accepted")
	}
	if _, err := NewRouter(&cfg, module, module); err == nil {
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
			WriteError(c.Writer, 403, "forbidden")
			return
		}
		c.Next()
	})
	group.GET("/user/:id", func(c *gin.Context) {
		WriteOK(c.Writer, map[string]string{"team": c.Param("team"), "id": c.Param("id")})
	})
}
func TestNewRouterGroupedRoutes(t *testing.T) {
	handler, err := NewRouter(&config.Config{}, groupedModule{})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		path   string
		status int
		body   string
	}{
		{"/team/public/user/42", 200, `{"id":"42","team":"public"}`},
		{"/team/private/user/42", 403, `{"msg":"forbidden"}`},
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
{{ end -}}
