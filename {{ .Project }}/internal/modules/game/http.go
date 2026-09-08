{{ if eq .Computed.http_router_final "gin" -}}
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

	"github.com/gin-gonic/gin"
	"{{ .Computed.module_name_final }}/internal/common/server"
	"{{ .Computed.module_name_final }}/internal/modules"
)

var _ modules.HTTPModule = (*Module)(nil)

func (*Module) Name() string { return "/game" }
func (m *Module) HTTP(r *gin.RouterGroup) {
	r.GET("", m.list)
	r.GET("/:id", m.get)
	r.POST("", m.create)
	r.PATCH("/:id", m.update)
	r.DELETE("/:id", m.delete)
}

func (m *Module) get(c *gin.Context) {
	w, r := c.Writer, c.Request
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		server.WriteError(w, http.StatusBadRequest, "invalid game id")
		return
	}
	value, err := m.Get(r.Context(), id)
	if errors.Is(err, ErrInvalid) {
		server.WriteError(w, http.StatusBadRequest, ErrInvalid.Error())
		return
	}
	if errors.Is(err, ErrNotFound) {
		server.WriteError(w, http.StatusNotFound, ErrNotFound.Error())
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), r.Method+" "+r.URL.Path+" failed: "+err.Error())
		server.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	server.WriteOK(w, value)
}
func (m *Module) create(c *gin.Context) {
	w, r := c.Writer, c.Request
	var input Input
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		server.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		server.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	value, err := m.Create(r.Context(), input)
	if errors.Is(err, ErrInvalid) {
		server.WriteError(w, http.StatusBadRequest, ErrInvalid.Error())
		return
	}
	if errors.Is(err, ErrNotFound) {
		server.WriteError(w, http.StatusNotFound, ErrNotFound.Error())
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), r.Method+" "+r.URL.Path+" failed: "+err.Error())
		server.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	server.WriteOK(w, value)
}
func (m *Module) update(c *gin.Context) {
	w, r := c.Writer, c.Request
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		server.WriteError(w, http.StatusBadRequest, "invalid game id")
		return
	}
	var input UpdateInput
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		server.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		server.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	value, err := m.Update(r.Context(), id, input)
	if errors.Is(err, ErrInvalid) {
		server.WriteError(w, http.StatusBadRequest, ErrInvalid.Error())
		return
	}
	if errors.Is(err, ErrNotFound) {
		server.WriteError(w, http.StatusNotFound, ErrNotFound.Error())
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), r.Method+" "+r.URL.Path+" failed: "+err.Error())
		server.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	server.WriteOK(w, value)
}
func (m *Module) delete(c *gin.Context) {
	w, r := c.Writer, c.Request
	parts := strings.Split(c.Param("id"), ",")
	if len(parts) > 100 {
		server.WriteError(w, http.StatusBadRequest, ErrIDs.Error())
		return
	}
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id <= 0 {
			server.WriteError(w, http.StatusBadRequest, ErrIDs.Error())
			return
		}
		ids = append(ids, id)
	}
	err := m.Delete(r.Context(), ids...)
	if errors.Is(err, ErrIDs) {
		server.WriteError(w, http.StatusBadRequest, ErrIDs.Error())
		return
	}
	if errors.Is(err, ErrNotFound) {
		server.WriteError(w, http.StatusNotFound, ErrNotFound.Error())
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), r.Method+" "+r.URL.Path+" failed: "+err.Error())
		server.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	server.WriteOK(w)
}

func (m *Module) list(c *gin.Context) {
	w, r := c.Writer, c.Request
	// URL.Query/GetQuery discard malformed entries; reject those before reading.
	if _, err := url.ParseQuery(r.URL.RawQuery); err != nil {
		server.WriteError(w, http.StatusBadRequest, "invalid query parameters")
		return
	}
	input := ListInput{Name: c.Query("name")}
	if value, provided := c.GetQuery("p"); provided {
		values, _ := c.GetQueryArray("p")
		if len(values) != 1 {
			server.WriteError(w, http.StatusBadRequest, "p must be one int32 value")
			return
		}
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			server.WriteError(w, http.StatusBadRequest, "p must be one int32 value")
			return
		}
		value32 := int32(parsed)
		input.Page = &value32
	}
	if value, provided := c.GetQuery("s"); provided {
		values, _ := c.GetQueryArray("s")
		if len(values) != 1 {
			server.WriteError(w, http.StatusBadRequest, "s must be one int32 value")
			return
		}
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			server.WriteError(w, http.StatusBadRequest, "s must be one int32 value")
			return
		}
		value32 := int32(parsed)
		input.PageSize = &value32
	}
	value, err := m.List(r.Context(), input)
	if err != nil {
		slog.ErrorContext(r.Context(), "list games failed: "+err.Error())
		server.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	server.WriteOK(w, value)
}
{{ else -}}
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

	"github.com/go-chi/chi/v5"
	"{{ .Computed.module_name_final }}/internal/common/server"
	"{{ .Computed.module_name_final }}/internal/modules"
)

var _ modules.HTTPModule = (*Module)(nil)

func (*Module) Name() string { return "/game" }
func (m *Module) HTTP() http.Handler {
	r := chi.NewRouter()
	r.Get("/", m.list)
	r.Get("/{id}", m.get)
	r.Post("/", m.create)
	r.Patch("/{id}", m.update)
	r.Delete("/{id}", m.delete)
	return r
}

func (m *Module) get(w http.ResponseWriter, r *http.Request) {

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		server.WriteError(w, http.StatusBadRequest, "invalid game id")
		return
	}
	value, err := m.Get(r.Context(), id)
	if errors.Is(err, ErrInvalid) {
		server.WriteError(w, http.StatusBadRequest, ErrInvalid.Error())
		return
	}
	if errors.Is(err, ErrNotFound) {
		server.WriteError(w, http.StatusNotFound, ErrNotFound.Error())
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), r.Method+" "+r.URL.Path+" failed: "+err.Error())
		server.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	server.WriteOK(w, value)
}
func (m *Module) create(w http.ResponseWriter, r *http.Request) {

	var input Input
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		server.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		server.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	value, err := m.Create(r.Context(), input)
	if errors.Is(err, ErrInvalid) {
		server.WriteError(w, http.StatusBadRequest, ErrInvalid.Error())
		return
	}
	if errors.Is(err, ErrNotFound) {
		server.WriteError(w, http.StatusNotFound, ErrNotFound.Error())
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), r.Method+" "+r.URL.Path+" failed: "+err.Error())
		server.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	server.WriteOK(w, value)
}
func (m *Module) update(w http.ResponseWriter, r *http.Request) {

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		server.WriteError(w, http.StatusBadRequest, "invalid game id")
		return
	}
	var input UpdateInput
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		server.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		server.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	value, err := m.Update(r.Context(), id, input)
	if errors.Is(err, ErrInvalid) {
		server.WriteError(w, http.StatusBadRequest, ErrInvalid.Error())
		return
	}
	if errors.Is(err, ErrNotFound) {
		server.WriteError(w, http.StatusNotFound, ErrNotFound.Error())
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), r.Method+" "+r.URL.Path+" failed: "+err.Error())
		server.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	server.WriteOK(w, value)
}
func (m *Module) delete(w http.ResponseWriter, r *http.Request) {

	parts := strings.Split(chi.URLParam(r, "id"), ",")
	if len(parts) > 100 {
		server.WriteError(w, http.StatusBadRequest, ErrIDs.Error())
		return
	}
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id <= 0 {
			server.WriteError(w, http.StatusBadRequest, ErrIDs.Error())
			return
		}
		ids = append(ids, id)
	}
	err := m.Delete(r.Context(), ids...)
	if errors.Is(err, ErrIDs) {
		server.WriteError(w, http.StatusBadRequest, ErrIDs.Error())
		return
	}
	if errors.Is(err, ErrNotFound) {
		server.WriteError(w, http.StatusNotFound, ErrNotFound.Error())
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), r.Method+" "+r.URL.Path+" failed: "+err.Error())
		server.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	server.WriteOK(w)
}

func (m *Module) list(w http.ResponseWriter, r *http.Request) {
	// URL.Query/GetQuery discard malformed entries; reject those before reading.
	if _, err := url.ParseQuery(r.URL.RawQuery); err != nil {
		server.WriteError(w, http.StatusBadRequest, "invalid query parameters")
		return
	}
	query := r.URL.Query()
	input := ListInput{Name: query.Get("name")}
	if values, provided := query["p"]; provided {
		if len(values) != 1 {
			server.WriteError(w, http.StatusBadRequest, "p must be one int32 value")
			return
		}
		parsed, err := strconv.ParseInt(values[0], 10, 32)
		if err != nil {
			server.WriteError(w, http.StatusBadRequest, "p must be one int32 value")
			return
		}
		value32 := int32(parsed)
		input.Page = &value32
	}
	if values, provided := query["s"]; provided {
		if len(values) != 1 {
			server.WriteError(w, http.StatusBadRequest, "s must be one int32 value")
			return
		}
		parsed, err := strconv.ParseInt(values[0], 10, 32)
		if err != nil {
			server.WriteError(w, http.StatusBadRequest, "s must be one int32 value")
			return
		}
		value32 := int32(parsed)
		input.PageSize = &value32
	}
	value, err := m.List(r.Context(), input)
	if err != nil {
		slog.ErrorContext(r.Context(), "list games failed: "+err.Error())
		server.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	server.WriteOK(w, value)
}
{{ end -}}
