package user

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"{{ .Computed.module_name_final }}/internal/common/server"

	"github.com/go-chi/chi/v5"
)

func (m *Module) HTTP() http.Handler {
	r := chi.NewRouter()
	r.Get("/{id}", m.get)
	return r
}

func (m *Module) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		server.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	value, err := m.Find(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		server.WriteError(w, http.StatusNotFound, ErrNotFound.Error())
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "find user failed: "+err.Error())
		server.WriteError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}
	server.WriteOK(w, value)
}
