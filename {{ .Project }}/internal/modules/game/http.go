package game

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	{{ if eq .Computed.http_router_final "gin" -}}
	"github.com/gin-gonic/gin"
{{ else -}}
	"github.com/go-chi/chi/v5"
{{ end -}}
	"{{ .Computed.module_name_final }}/internal/common/apperror"
	"{{ .Computed.module_name_final }}/internal/common/server"
	"{{ .Computed.module_name_final }}/internal/modules"
)

var errInvalidID = apperror.New("GAME_INVALID_ID", "invalid game id")

var _ modules.HTTPModule = (*Module)(nil)

func (*Module) Name() string { return "/game" }
func (*Module) Idempotent() bool { return true }

{{ if eq .Computed.http_router_final "gin" -}}
func (m *Module) HTTP(r *gin.RouterGroup) {
	r.GET("", m.list)
	r.GET("/:id", m.get)
	r.POST("", m.create)
	r.PATCH("/:id", m.update)
	r.DELETE("/:id", m.delete)
}
{{ else -}}
func (m *Module) HTTP() http.Handler {
	r := chi.NewRouter()
	r.Get("/", m.list)
	r.Get("/{id}", m.get)
	r.Post("/", m.create)
	r.Patch("/{id}", m.update)
	r.Delete("/{id}", m.delete)
	return r
}
{{ end -}}

{{ if eq .Computed.http_router_final "gin" -}}
func (m *Module) get(c *gin.Context) {
	w, r := c.Writer, c.Request
{{ else -}}
func (m *Module) get(w http.ResponseWriter, r *http.Request) {
{{ end }}
	id, err := strconv.ParseInt({{ if eq .Computed.http_router_final "gin" }}c.Param("id"){{ else }}chi.URLParam(r, "id"){{ end }}, 10, 64)
	if err != nil {
		server.WriteError(w, r, http.StatusBadRequest, errInvalidID)
		return
	}
	value, err := m.Get(r.Context(), id)
	if errors.Is(err, ErrInvalid) {
		server.WriteError(w, r, http.StatusBadRequest, ErrInvalid)
		return
	}
	if errors.Is(err, ErrNotFound) {
		server.WriteError(w, r, http.StatusNotFound, ErrNotFound)
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), r.Method+" "+r.URL.Path+" failed: "+err.Error())
		server.WriteError(w, r, http.StatusInternalServerError, apperror.Internal)
		return
	}
	server.WriteOK(w, value)
}

{{ if eq .Computed.http_router_final "gin" -}}
func (m *Module) create(c *gin.Context) {
	w, r := c.Writer, c.Request
{{ else -}}
func (m *Module) create(w http.ResponseWriter, r *http.Request) {
{{ end }}
	var input Input
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		server.WriteError(w, r, http.StatusBadRequest, apperror.InvalidBody)
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		server.WriteError(w, r, http.StatusBadRequest, apperror.InvalidBody)
		return
	}
	value, err := m.Create(r.Context(), input)
	if errors.Is(err, ErrInvalid) {
		server.WriteError(w, r, http.StatusBadRequest, ErrInvalid)
		return
	}
	if errors.Is(err, ErrNotFound) {
		server.WriteError(w, r, http.StatusNotFound, ErrNotFound)
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), r.Method+" "+r.URL.Path+" failed: "+err.Error())
		server.WriteError(w, r, http.StatusInternalServerError, apperror.Internal)
		return
	}
	server.WriteOK(w, value)
}

{{ if eq .Computed.http_router_final "gin" -}}
func (m *Module) update(c *gin.Context) {
	w, r := c.Writer, c.Request
{{ else -}}
func (m *Module) update(w http.ResponseWriter, r *http.Request) {
{{ end }}
	id, err := strconv.ParseInt({{ if eq .Computed.http_router_final "gin" }}c.Param("id"){{ else }}chi.URLParam(r, "id"){{ end }}, 10, 64)
	if err != nil {
		server.WriteError(w, r, http.StatusBadRequest, errInvalidID)
		return
	}
	var input UpdateInput
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		server.WriteError(w, r, http.StatusBadRequest, apperror.InvalidBody)
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		server.WriteError(w, r, http.StatusBadRequest, apperror.InvalidBody)
		return
	}
	value, err := m.Update(r.Context(), id, input)
	if errors.Is(err, ErrInvalid) {
		server.WriteError(w, r, http.StatusBadRequest, ErrInvalid)
		return
	}
	if errors.Is(err, ErrNotFound) {
		server.WriteError(w, r, http.StatusNotFound, ErrNotFound)
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), r.Method+" "+r.URL.Path+" failed: "+err.Error())
		server.WriteError(w, r, http.StatusInternalServerError, apperror.Internal)
		return
	}
	server.WriteOK(w, value)
}

{{ if eq .Computed.http_router_final "gin" -}}
func (m *Module) delete(c *gin.Context) {
	w, r := c.Writer, c.Request
{{ else -}}
func (m *Module) delete(w http.ResponseWriter, r *http.Request) {
{{ end }}
	parts := strings.Split({{ if eq .Computed.http_router_final "gin" }}c.Param("id"){{ else }}chi.URLParam(r, "id"){{ end }}, ",")
	if len(parts) > 100 {
		server.WriteError(w, r, http.StatusBadRequest, ErrIDs)
		return
	}
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id <= 0 {
			server.WriteError(w, r, http.StatusBadRequest, ErrIDs)
			return
		}
		ids = append(ids, id)
	}
	err := m.Delete(r.Context(), ids...)
	if errors.Is(err, ErrIDs) {
		server.WriteError(w, r, http.StatusBadRequest, ErrIDs)
		return
	}
	if errors.Is(err, ErrNotFound) {
		server.WriteError(w, r, http.StatusNotFound, ErrNotFound)
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), r.Method+" "+r.URL.Path+" failed: "+err.Error())
		server.WriteError(w, r, http.StatusInternalServerError, apperror.Internal)
		return
	}
	server.WriteOK(w)
}

{{ if eq .Computed.http_router_final "gin" -}}
func (m *Module) list(c *gin.Context) {
	w, r := c.Writer, c.Request
{{ else -}}
func (m *Module) list(w http.ResponseWriter, r *http.Request) {
{{ end }}
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		server.WriteError(w, r, http.StatusBadRequest, apperror.InvalidQuery)
		return
	}
	input := ListInput{Name: query.Get("name")}
	if values, provided := query["p"]; provided {
		if len(values) != 1 {
			server.WriteError(w, r, http.StatusBadRequest, apperror.Page)
			return
		}
		parsed, err := strconv.ParseInt(values[0], 10, 32)
		if err != nil {
			server.WriteError(w, r, http.StatusBadRequest, apperror.Page)
			return
		}
		value32 := int32(parsed)
		input.Page = &value32
	}
	if values, provided := query["s"]; provided {
		if len(values) != 1 {
			server.WriteError(w, r, http.StatusBadRequest, apperror.PageSize)
			return
		}
		parsed, err := strconv.ParseInt(values[0], 10, 32)
		if err != nil {
			server.WriteError(w, r, http.StatusBadRequest, apperror.PageSize)
			return
		}
		value32 := int32(parsed)
		input.PageSize = &value32
	}
	value, err := m.List(r.Context(), input)
	if err != nil {
		slog.ErrorContext(r.Context(), "list games failed: "+err.Error())
		server.WriteError(w, r, http.StatusInternalServerError, apperror.Internal)
		return
	}
	server.WriteOK(w, value)
}
