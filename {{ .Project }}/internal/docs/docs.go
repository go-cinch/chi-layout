package docs

import (
	"embed"
	"net/http"

	"{{ .Computed.module_name_final }}/internal/common/server"
	"{{ .Computed.module_name_final }}/internal/modules"
)

//go:embed index.html openapi.yaml
var files embed.FS

type Module struct {
	handler http.Handler
}

var _ modules.Module = (*Module)(nil)

func New() *Module {
	fileServer := http.FileServer(http.FS(files))
	return &Module{handler: http.StripPrefix("/docs/", fileServer)}
}

func (*Module) Name() string {
	return "/docs"
}

func (m *Module) HTTP() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			server.WriteError(w, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
			return
		}
		if r.URL.Path == "/docs" {
			http.Redirect(w, r, "/docs/", http.StatusMovedPermanently)
			return
		}
		m.handler.ServeHTTP(w, r)
	})
}
