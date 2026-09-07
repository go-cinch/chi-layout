{{ if eq .Computed.http_router_final "gin" -}}
package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"{{ .Computed.module_name_final }}/internal/common/config"
	"{{ .Computed.module_name_final }}/internal/common/redact"
	"{{ .Computed.module_name_final }}/internal/modules"
)

func NewRouter(cfg *config.Config, mounted ...modules.Module) (http.Handler, error) {
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.HandleMethodNotAllowed = true
	// Access logs use RemoteAddr, matching the chi implementation.
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, err
	}
	policy := redact.New(cfg.Redact.Keys...)
{{- if .Computed.enable_trace_final }}
	r.Use(TraceID(cfg.Server.Name, policy))
{{- end }}
	r.Use(AccessLog(policy), Recoverer(), RequestTimeout(cfg.HTTP.Timeout))
	r.NoRoute(func(c *gin.Context) {
		WriteError(c.Writer, http.StatusNotFound, http.StatusText(http.StatusNotFound))
	})
	r.NoMethod(func(c *gin.Context) {
		WriteError(c.Writer, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
	})
	seen := make(map[string]struct{}, len(mounted))
	for _, module := range mounted {
		if module == nil {
			return nil, errors.New("http module is required")
		}
		pattern := strings.TrimSpace(module.Name())
		if pattern == "" || !strings.HasPrefix(pattern, "/") {
			return nil, errors.New("http module pattern must start with /")
		}
		if _, exists := seen[pattern]; exists {
			return nil, errors.New("http module pattern is duplicated")
		}
		seen[pattern] = struct{}{}
		module.HTTP(r.Group(pattern))
	}
	return r, nil
}
{{ else -}}
package server

import (
	"errors"
	"net/http"
	"strings"

	"{{ .Computed.module_name_final }}/internal/common/config"
	"{{ .Computed.module_name_final }}/internal/common/redact"
	"{{ .Computed.module_name_final }}/internal/modules"

	"github.com/go-chi/chi/v5"
)

func NewRouter(cfg *config.Config, mounted ...modules.Module) (http.Handler, error) {
	r := chi.NewRouter()
	policy := redact.New(cfg.Redact.Keys...)
{{- if .Computed.enable_trace_final }}
	r.Use(TraceID(cfg.Server.Name, policy))
{{- end }}
	r.Use(AccessLog(policy), Recoverer())

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		WriteError(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		WriteError(w, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
	})
	routes := r.With(RequestTimeout(cfg.HTTP.Timeout))
	seen := make(map[string]struct{}, len(mounted))
	for _, module := range mounted {
		if module == nil {
			return nil, errors.New("http module is required")
		}
		pattern := strings.TrimSpace(module.Name())
		if pattern == "" || !strings.HasPrefix(pattern, "/") {
			return nil, errors.New("http module pattern must start with /")
		}
		if _, exists := seen[pattern]; exists {
			return nil, errors.New("http module pattern is duplicated")
		}
		seen[pattern] = struct{}{}
		handler := module.HTTP()
		if handler == nil {
			return nil, errors.New("http module handler is required")
		}
		routes.Mount(pattern, handler)
	}

	return r, nil
}
{{ end -}}
